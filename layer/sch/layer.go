package sch

import (
	"errors"
	"net"
	"time"

	"github.com/kisom/go-schannel/schannel"
)

// Layer - Schannel Layer.
type Layer struct {
	pub  *[32]byte
	priv *[64]byte
}

// NewLayer - Creates a new Schannel layer.
func NewLayer(opts ...Option) (layer *Layer) {
	layer = &Layer{}
	for _, opt := range opts {
		opt(layer)
	}
	return
}

// Name - Returns "sch".
func (layer *Layer) Name() string { return "sch" }

// Listener - Wraps listener with a Schannel layer.
func (layer *Layer) Listener(l net.Listener) (net.Listener, error) {
	return &listener{
		Listener: l,
		layer:    layer,
	}, nil
}

// Conn - Wraps the connection with a secure channel that uses Schannel.
func (layer *Layer) Conn(conn net.Conn) (net.Conn, error) {
	sch, ok := schannel.Dial(conn, layer.priv, layer.pub)
	if !ok {
		return nil, errors.New("sch dial error")
	}
	return &connection{
		Conn: conn,
		sch:  sch,
	}, nil
}

// Option - Schannel layer option.
type Option func(*Layer)

// WithPubKey - Sets Schannel public key.
func WithPubKey(key *[32]byte) Option {
	return func(layer *Layer) {
		layer.pub = key
	}
}

// WithPrivKey - Sets Schannel private key.
func WithPrivKey(key *[64]byte) Option {
	return func(layer *Layer) {
		layer.priv = key
	}
}

// listener - Schannel layer listener.
type listener struct {
	net.Listener
	layer *Layer
}

// Accept - Accepts connection. Look at schannel.Listen.
func (listener *listener) Accept() (conn net.Conn, err error) {
	conn, err = listener.Listener.Accept()
	if err != nil {
		return
	}
	sch, ok := schannel.Listen(conn, listener.layer.priv, listener.layer.pub)
	if !ok {
		return nil, errors.New("Schannel error")
	}
	return &connection{
		Conn: conn,
		sch:  sch,
	}, nil
}

// connection - Connection wrapped with a schannel.
type connection struct {
	net.Conn
	sch *schannel.SChannel
}

func (conn *connection) Read(b []byte) (n int, err error) {
	msg, ok := conn.sch.Receive()
	if !ok {
		return 0, errors.New("read error")
	}
	return copy(b, msg.Contents), nil
}

func (conn *connection) Write(b []byte) (n int, err error) {
	if ok := conn.sch.Send(b); !ok {
		return 0, errors.New("write error")
	}
	return len(b), nil
}

// IsDialer - Returns false. Schannel layer is not a Dialer.
func (layer *Layer) IsDialer() bool {
	return false
}

// Dial - Returns nil, nil. Not a dialer.
func (layer *Layer) Dial(addr string, timeout time.Duration) (net.Conn, error) {
	return nil, nil
}

// Close - Does nothing.
func (layer *Layer) Close() error {
	return nil
}
