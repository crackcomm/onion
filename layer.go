package onion

import (
	"net"
	"time"
)

// Layer - Onion layer.
type Layer interface {
	// Name - Layer name.
	Name() string

	// Conn - Wraps connection with a layer.
	// Some layers do nothing - for example TOR cant wrap existing connection.
	// It's usable for encryption layers.
	Conn(net.Conn) (net.Conn, error)

	// Listener - Wraps listener with a layer.
	Listener(net.Listener) (net.Listener, error)

	// Dial - Dials to address with a timeout.
	Dial(addr string, timeout time.Duration) (net.Conn, error) // Bool is false when it should be ignored

	// IsDialer - Returns false if the layer is just an encryption (and/or listening) layer.
	// Returns true if layer can dial. For example net, tor etc.
	IsDialer() bool

	// Close - Closes the layer, removes the keys, closes tor instance etc.
	Close() error
}
