// Package onion implements protocols and encryption onion.
//
// Example onion could look like this:
//
// 	onion := New(
// 		NetLayer(),
// 		TorLayer(),
// 		TLSLayer(),
// 		NaClLayer(),
// 		JWTLayer(keyLookupFunc),
// 		NaClLayer(),
// 	)
//
package onion

import (
	"net"
	"net/http"
	"time"

	"github.com/golang/glog"
)

// Onion - Onion interface.
type Onion interface {
	// Dial - Connect for net.Dialer.
	Dial(string, string) (net.Conn, error)

	// Connect - Dials to a target through an onion.
	Connect(string, time.Duration) (net.Conn, error)

	// Listener - Wraps a listener with an onion.
	// Sometimes listener can or even should be empty.
	Listener(net.Listener) (net.Listener, error)

	// Close - Closes all layers of the onion.
	Close() error
}

// New - Creates a new onion from given layers.
func New(layers ...Layer) Onion {
	return &onion{layers: layers}
}

// HTTP - Returns http Client that dials through an onion.
func HTTP(o Onion) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			Dial:                o.Dial,
			TLSHandshakeTimeout: time.Second * 10,
		},
	}
}

type onion struct {
	layers  []Layer
	verbose bool
}

// Dial - Dials to a target through an onion.
func (on *onion) Dial(network, addr string) (conn net.Conn, err error) {
	return on.Connect(addr, time.Second*30)
}

// Connect - Connect for net.Dialer.
func (on *onion) Connect(addr string, timeout time.Duration) (conn net.Conn, err error) {
	for _, layer := range on.layers {
		if conn == nil && layer.IsDialer() {
			if on.verbose {
				glog.Infof("[%s] dial => %s", layer.Name(), addr)
			}
			if c, err := layer.Dial(addr, timeout); err == nil {
				conn = c
			} else {
				return nil, err
			}
		} else {
			if on.verbose {
				glog.Infof("[%s] conn => %s", layer.Name(), addr)
			}
			conn, err = layer.Conn(conn)
			if err != nil {
				return nil, err
			}
		}
	}
	return
}

// Listener - Wraps a listener with an onion.
func (on *onion) Listener(in net.Listener) (l net.Listener, err error) {
	l = in
	for _, layer := range on.layers {
		if on.verbose {
			if l == nil {
				glog.Infof("[%s] listen init", layer.Name())
			} else {
				glog.Infof("[%s] listen => %s", layer.Name(), l.Addr())
			}
		}
		l, err = layer.Listener(l)
		if err != nil {
			return
		}
	}
	return
}

// Close - Closes all layers of the onion.
func (on *onion) Close() (err error) {
	for _, layer := range on.layers {
		if on.verbose {
			glog.Infof("[%s] close", layer.Name())
		}
		err = layer.Close()
		if err != nil {
			return
		}
	}
	return
}

// SetVerbose - When verbose is set to true it logs some info.
func (on *onion) SetVerbose(v bool) {
	on.verbose = v
}
