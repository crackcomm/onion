package torctl

import (
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"sync"
	"time"

	"github.com/yawning/bulb"

	"github.com/crackcomm/onion/proxyutil"
)

// Client - TOR Client.
type Client struct {
	opts *LaunchOptions

	mutex          *sync.RWMutex
	lockExpiration time.Time
	nextRestart    time.Time

	cmd *exec.Cmd

	publicIP, countryCode string
}

// NewNymInterval - Default Interval for new identity.
var NewNymInterval = 100 * time.Millisecond

// PublicIP - Returns proxy public IP.
func (client *Client) PublicIP() string {
	return client.publicIP
}

// ProxyAddress - Returns proxy address.
func (client *Client) ProxyAddress() string {
	return fmt.Sprintf("%s:%d", client.opts.ProxyAddress, client.opts.ProxyPort)
}

// HTTPClient - Returns http Client using tor proxy.
func (client *Client) HTTPClient() (*http.Client, error) {
	return proxyutil.Socks5HTTPClient("tcp", client.ProxyAddress())
}

// Endpoint - Returns information about the proxy endpoint.
func (client *Client) Endpoint() *Endpoint {
	return &Endpoint{
		Address:     client.ProxyAddress(),
		Type:        Endpoint_SOCKS5,
		PublicIp:    client.publicIP,
		CountryCode: client.countryCode,
	}
}

// Control - Dials to tor control.
func (client *Client) Control() (*bulb.Conn, error) {
	addr := fmt.Sprintf("%s:%d", client.opts.ControlAddress, client.opts.ControlPort)
	conn, err := bulb.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}

	// Authenticate TOR Control
	if client.opts.ControlPassword != "" {
		err := conn.Authenticate(client.opts.ControlPassword)
		if err != nil {
			conn.Close()
			return nil, err
		}
	}

	return conn, nil
}

// Release - Locks the service for n seconds.
func (client *Client) Release() {
	client.mutex.Lock()
	client.lockExpiration = time.Now()
	client.mutex.Unlock()
}

// LockInSeconds - Locks the service for n seconds.
func (client *Client) LockInSeconds(seconds uint32) {
	client.mutex.Lock()
	client.lockExpiration = time.Now().Add(time.Duration(seconds) * time.Second)
	client.mutex.Unlock()
}

// RequestNewIdentity - Requests new identity (IP address).
func (client *Client) RequestNewIdentity() (ok bool, err error) {
	return client.RequestNewIdentityAndLock(0)
}

// RequestNewIdentityAndLock - Requests new identity (IP address).
func (client *Client) RequestNewIdentityAndLock(seconds uint32) (ok bool, err error) {
	client.mutex.Lock()
	defer client.mutex.Unlock()

	if client.isBusy() || !client.canRestart() {
		return false, nil
	}

	// Dial to TOR Control
	conn, err := client.Control()
	if err != nil {
		return
	}
	defer conn.Close()

	// Request new identity
	if _, err = conn.Request("SIGNAL NEWNYM"); err != nil {
		return
	}

	// Set lock expiration time if any
	if seconds > 0 {
		client.lockExpiration = time.Now().Add(time.Duration(seconds) * time.Second)
	}

	// Set next possible restart time
	client.nextRestart = time.Now().Add(NewNymInterval)

	// Check our IP address and save it in memory
	client.publicIP, client.countryCode, err = client.getIPAddress()
	if err != nil {
		return
	}

	return true, nil
}

// Close - Closess a tor service.
func (client *Client) Close() error {
	return client.Kill()
}

// Kill - Kills a tor service.
func (client *Client) Kill() error {
	return client.cmd.Process.Kill()
}

// CanRestart - Checks if can restart ip address.
func (client *Client) CanRestart() bool {
	client.mutex.RLock()
	defer client.mutex.RUnlock()
	return client.canRestart()
}

// IsBusy - Checks if the service is busy (locked).
func (client *Client) IsBusy() bool {
	client.mutex.RLock()
	defer client.mutex.RUnlock()
	return client.isBusy()
}

// ProxyExpiration - Proxy expiration time.
func (client *Client) ProxyExpiration() time.Time {
	return client.lockExpiration
}

// ProxyExpirationSeconds - Proxy expiration time in seconds.
func (client *Client) ProxyExpirationSeconds() uint32 {
	return uint32(client.ProxyExpiration().Sub(time.Now()))
}

// UpdateIPAddress - Updates TOR public ip address.
func (client *Client) UpdateIPAddress() (addr string, err error) {
	addr, country, err := client.getIPAddress()
	if err != nil {
		return
	}

	client.mutex.Lock()
	client.publicIP = addr
	client.countryCode = country
	client.mutex.Unlock()

	return
}

func (client *Client) getIPAddress() (addr, country string, err error) {
	c, err := client.HTTPClient()
	if err != nil {
		return
	}

	addr, err = proxyutil.GetIPAddress(c)
	if err != nil {
		return
	}

	if db := client.opts.MaxMindDB; db != nil {
		record, err := db.City(net.ParseIP(addr))
		if err != nil {
			return "", "", err
		}
		country = record.Country.IsoCode
	}

	return
}

func (client *Client) canRestart() bool {
	return time.Now().After(client.nextRestart)
}

func (client *Client) isBusy() bool {
	return !time.Now().After(client.lockExpiration)
}
