package tls

import (
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net"
	"time"
)

// Layer - TLS Layer.
type Layer struct {
	config *tls.Config
}

// NewLayer - Creates a new TLS layer.
func NewLayer(opts ...Option) (layer *Layer) {
	layer = &Layer{}
	layer.config = &tls.Config{
		ClientAuth:   tls.RequireAndVerifyClientCert,
		Certificates: []tls.Certificate{},
		ClientCAs:    x509.NewCertPool(),
		Rand:         rand.Reader,
	}
	for _, opt := range opts {
		opt(layer)
	}
	return
}

// Name - Returns "tls".
func (layer *Layer) Name() string { return "tls" }

// Listener - Wraps listener with a TLS layer.
func (layer *Layer) Listener(l net.Listener) (net.Listener, error) {
	return tls.NewListener(l, layer.config), nil
}

// Conn - Wraps the connection with a secure channel that uses TLS.
func (layer *Layer) Conn(conn net.Conn) (net.Conn, error) {
	return tls.Client(conn, layer.config), nil
}

// Option - TLS layer option.
type Option func(*Layer)

// WithConfig - Sets TLS public key.
func WithConfig(config *tls.Config) Option {
	return func(layer *Layer) {
		layer.config = config
	}
}

// WithInsecure -
func WithInsecure() Option {
	return func(layer *Layer) {
		layer.config.InsecureSkipVerify = true
	}
}

// WithCertAndKey - Sets certificate and private key.
func WithCertAndKey(certbody, privbody []byte) Option {
	certx509, err := x509.ParseCertificate(certbody)
	if err != nil {
		panic(err)
	}
	privkey, err := x509.ParsePKCS1PrivateKey(privbody)
	if err != nil {
		panic(err)
	}
	cert := tls.Certificate{
		Certificate: [][]byte{certbody},
		PrivateKey:  privkey,
	}
	return func(layer *Layer) {
		layer.config.Certificates = append(layer.config.Certificates, cert)
		layer.config.ClientCAs.AddCert(certx509)
	}
}

// WithCertAndKeyFile - Reads certificate and key file and adds to layer tls config.
// Panics if read error occurs.
func WithCertAndKeyFile(certfilename, privfilename string) Option {
	certbody, err := ioutil.ReadFile(certfilename)
	if err != nil {
		panic(err)
	}
	privbody, err := ioutil.ReadFile(privfilename)
	if err != nil {
		panic(err)
	}
	return WithCertAndKey(certbody, privbody)
}

// IsDialer - Returns false. TLS layer can't dial.
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
