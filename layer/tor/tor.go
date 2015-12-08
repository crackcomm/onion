package tor

import (
	"crypto"
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/yawning/bulb"

	"golang.org/x/net/context"
	"golang.org/x/net/proxy"

	"github.com/crackcomm/onion/proxyutil"
	"github.com/crackcomm/torctl"
)

// Layer - TOR Layer.
type Layer struct {
	o *options

	control *bulb.Conn

	mutex   *sync.Mutex
	client  *torctl.Client
	created bool // true if created by layer, and if true closed by the layer
}

// NewLayer - Creates a new TOR layer.
func NewLayer(opts ...Option) (layer *Layer) {
	layer = &Layer{mutex: new(sync.Mutex)}
	layer.o = new(options)
	for _, opt := range opts {
		opt(layer)
	}
	return
}

// Name - Returns "tor".
func (layer *Layer) Name() string { return "tor" }

// Listener returns a net.Listener backed by a Onion Service, optionally
// having Tor generate an ephemeral private key.  Regardless of the status of
// the returned Listener, the Onion Service will be torn down when the control
// connection is closed.
//
// WARNING: Only one port can be listened to per PrivateKey if this interface
// is used.  To bind to more ports, use the  AddOnion call directly.
func (layer *Layer) Listener(l net.Listener) (net.Listener, error) {
	addr, ok := l.Addr().(*net.TCPAddr)
	if !ok {
		l.Close()
		return nil, errors.New("failed to extract local port")
	}

	control, err := layer.torControl()
	if err != nil {
		l.Close()
		return nil, err
	}

	port := layer.o.port
	if port == 0 {
		port = uint16(addr.Port)
	}

	ports := []bulb.OnionPortSpec{
		{port, strconv.Itoa(int(addr.Port))},
	}

	info, err := control.AddOnion(ports, nil, true)
	if err != nil {
		l.Close()
		return nil, err
	}

	return &listener{
		Listener: l,
		address:  &Addr{Port: port, OnionInfo: info},
		control:  layer.control,
	}, nil
}

// Dial - Dials through a TOR proxy.
func (layer *Layer) Dial(addr string, timeout time.Duration) (net.Conn, error) {
	proxyaddr, err := layer.getProxy()
	if err != nil {
		layer.mutex.Unlock()
		return nil, err
	}

	socks, err := proxy.SOCKS5("tcp", proxyaddr, nil, &net.Dialer{
		Timeout: timeout,
	})
	if err != nil {
		return nil, err
	}

	glog.Infof("[tor] proxy => %s => %s", proxyaddr, addr)
	conn, err := socks.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func (layer *Layer) getProxy() (string, error) {
	if layer.o.proxy != "" {
		return layer.o.proxy, nil
	}

	layer.mutex.Lock() // LOCK!
	defer layer.mutex.Unlock()

	client, err := layer.torClient()
	if err != nil {
		return "", err
	}
	layer.o.proxy = client.ProxyAddress()
	return layer.o.proxy, nil
}

// Conn - Returns back the same connection. TOR can't wrap an existing connection.
func (layer *Layer) Conn(conn net.Conn) (net.Conn, error) {
	return conn, nil
}

// Close - Closes a TOR
func (layer *Layer) Close() error {
	if !layer.created || layer.client == nil {
		return nil
	}
	layer.mutex.Lock()
	defer func() {
		layer.created = false
		layer.client = nil
		layer.mutex.Unlock()
	}()
	return layer.client.Close()
}

// IsDialer - Returns true, You can Dial through a TOR proxy.
func (layer *Layer) IsDialer() bool {
	return true
}

func (layer *Layer) torControl() (*bulb.Conn, error) {
	layer.mutex.Lock()
	defer layer.mutex.Unlock()

	if layer.control != nil {
		return layer.control, nil
	}

	client, err := layer.torClient()
	if err != nil {
		return nil, err
	}

	layer.control, err = client.Control()
	if err != nil {
		return nil, err
	}

	return layer.control, nil
}

// torClient - Returns tor client set with options or starts tor binary
// specified in options with default control password,
// automatically generated torrc and random ports.
func (layer *Layer) torClient() (*torctl.Client, error) {

	// Launch TOR instance if empty
	if layer.client == nil {
		var err error
		layer.client, err = torctl.Launch(context.Background(), &torctl.LaunchOptions{
			Quiet:       !layer.o.verbose,
			Path:        layer.o.bin,
			ProxyPort:   proxyutil.FreePort(),
			ControlPort: proxyutil.FreePort(),
			Binary:      layer.o.binBody,
		})
		if err != nil {
			return nil, err
		}
		layer.created = true
	}

	return layer.client, nil
}

// Option - TOR layer option.
type Option func(*Layer)

// options - TOR layer options.
type options struct {
	bin     string
	key     crypto.PrivateKey
	port    uint16
	verbose bool
	proxy   string

	binBody []byte

	timeout time.Duration
}

// WithBin - Sets tor binary path.
func WithBin(bin string) Option {
	return func(layer *Layer) {
		layer.o.bin = bin
	}
}

// WithBinaryBody - Sets tor binary body.
func WithBinaryBody(bin []byte) Option {
	return func(layer *Layer) {
		layer.o.binBody = bin
	}
}

// WithVerbose - Sets tor verbose.
func WithVerbose(verbose bool) Option {
	return func(layer *Layer) {
		layer.o.verbose = verbose
	}
}

// WithProxy - Sets tor listener proxy address.
func WithProxy(proxy string) Option {
	return func(layer *Layer) {
		layer.o.proxy = proxy
	}
}

// WithPort - Sets tor listener port.
func WithPort(port uint16) Option {
	return func(layer *Layer) {
		layer.o.port = port
	}
}

// WithClient - Sets tor client.
func WithClient(client *torctl.Client) Option {
	return func(layer *Layer) {
		layer.mutex.Lock()
		layer.client = client
		layer.mutex.Unlock()
	}
}

// WithControl - Sets tor client.
func WithControl(control *bulb.Conn) Option {
	return func(layer *Layer) {
		layer.mutex.Lock()
		layer.control = control
		layer.mutex.Unlock()
	}
}

// WithKey - Sets tor hidden service private key.
func WithKey(key crypto.PrivateKey) Option {
	return func(layer *Layer) {
		layer.o.key = key
	}
}

// Addr - TOR hidden service address.
type Addr struct {
	*bulb.OnionInfo
	Port uint16
}

// String - Returns tor onion domain with a port.
func (addr *Addr) String() string {
	return fmt.Sprintf("%s.onion:%d", addr.OnionInfo.OnionID, addr.Port)
}

// Network - Always returns "tcp".
func (addr *Addr) Network() string {
	return "tcp"
}

type listener struct {
	net.Listener
	control *bulb.Conn
	address *Addr
}

func (listener *listener) Addr() net.Addr {
	return listener.address
}

func (listener *listener) Accept() (net.Conn, error) {
	return listener.Listener.Accept()
}

// Close - Closes listener and removes onion service.
func (listener *listener) Close() error {
	if err := listener.Listener.Close(); err != nil {
		return err
	}
	return listener.control.DeleteOnion(listener.address.OnionInfo.OnionID)
}
