package proxyutil

import (
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/proxy"
)

// IPAddressURL - URL which returns requestee IP in response.
var IPAddressURL = "https://api.ipify.org/?format=text"

// DefaultTimeout - Default timeout for everything.
var DefaultTimeout = 30 * time.Second

// Socks5HTTPClient - Socks5 proxy HTTP client.
func Socks5HTTPClient(network, addr string) (*http.Client, error) {
	socks, err := proxy.SOCKS5(network, addr, nil, &net.Dialer{
		Timeout:   DefaultTimeout,
		KeepAlive: DefaultTimeout,
	})
	if err != nil {
		return nil, err
	}
	return &http.Client{
		Transport: &http.Transport{
			TLSHandshakeTimeout: DefaultTimeout,
			Dial:                socks.Dial,
		},
	}, nil
}

// GetIPAddress - Gets client IP address.
func GetIPAddress(client *http.Client) (addr string, err error) {
	resp, err := client.Get(IPAddressURL)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	return strings.TrimSpace(string(body)), nil
}

// FreePort - Gets random free port.
func FreePort() int {
	address, err := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	listener, err := net.ListenTCP("tcp", address)
	if err != nil {
		panic(err)
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}
