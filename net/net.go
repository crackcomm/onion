// Package net implements net interface.
//
// Network layer is just used as listener for proxies like tor.
package net

import (
	"fmt"
	"net"
	"time"
)

// Layer - Net layer.
type Layer struct {
	network  string
	address  string
	port     int
	isDialer bool
}

// NewLayer - Creates a new net layer.
func NewLayer(opts ...Option) (layer *Layer) {
	layer = &Layer{network: "tcp4", address: "127.0.0.1", port: 0}
	for _, opt := range opts {
		opt(layer)
	}
	return
}

// Name - Returns "net".
func (layer *Layer) Name() string { return "net" }

// Listener - Input should be empty.
// It will return a listener on random port (or a specified port)
// on a tpc4 (or specified) network interface.
func (layer *Layer) Listener(listener net.Listener) (net.Listener, error) {
	return net.Listen(layer.network, fmt.Sprintf("%s:%d", layer.address, layer.port))
}

// Option - Net layer option.
type Option func(*Layer)

// WithAddress - Sets net layer address (default: 127.0.0.1).
func WithAddress(address string) Option {
	return func(layer *Layer) {
		layer.address = address
	}
}

// WithPort - Sets net layer port (default: 127.0.0.1).
func WithPort(port int) Option {
	return func(layer *Layer) {
		layer.port = port
	}
}

// WithNetwork - Sets net layer network - tcp4 by default.
func WithNetwork(network string) Option {
	return func(layer *Layer) {
		layer.network = network
	}
}

// WithDial - Enables net Dial.
func WithDial() Option {
	return func(layer *Layer) {
		layer.isDialer = true
	}
}

// IsDialer - Returns false. Net layer is not a Dialer, this is a security package.
func (layer *Layer) IsDialer() bool {
	return layer.isDialer
}

// Dial - Dials
func (layer *Layer) Dial(addr string, timeout time.Duration) (net.Conn, error) {
	return net.Dial("tcp", addr)
}

// Conn - Returns back the same connection.
func (layer *Layer) Conn(conn net.Conn) (net.Conn, error) {
	return conn, nil
}

// Close - Does nothing.
func (layer *Layer) Close() error {
	return nil
}
