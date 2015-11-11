package torctl

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/golang/glog"
	"github.com/oschwald/geoip2-golang"
	"github.com/satori/go.uuid"
	"golang.org/x/net/context"
)

// LaunchOptions - Tor launch options.
type LaunchOptions struct {
	Path                string // tor binary (required)
	ProxyAddress        string // by default 127.0.0.1
	ProxyPort           int    // use FreePort to get random one
	ControlAddress      string // by default 127.0.0.1
	ControlPort         int    // use FreePort to get random one
	ControlPassword     string // default is used when empty
	ControlPasswordHash string // default is used when empty
	DataDir             string // random dir in TempDir by default
	TempDir             string // by default: os.TempDir()
	Torrc               string // generated from this configuration by default
	Quiet               bool   // set to true if you dont want tor logs in console

	// MaxMindDB is used to get country of public proxy ip
	MaxMindDB *geoip2.Reader

	Binary []byte
}

// Launch - Launches a tor service.
// TOR proxy public IP can be unknown after this process.
func Launch(ctx context.Context, o *LaunchOptions) (t *Client, err error) {
	// Set Launch options defaults and create
	// default data directory and torrc if required.
	o, err = setLaunchDefaults(o)
	if err != nil {
		return
	}

	if o.Path == "" && len(o.Binary) > 0 {
		o.Path = filepath.Join(os.TempDir(), uuid.NewV4().String())
		err = ioutil.WriteFile(o.Path, o.Binary, os.ModePerm)
		if err != nil {
			return
		}
	}

	// Create a TOR launch command
	torlogs := newLogReader(o.Quiet)
	cmd := &exec.Cmd{
		Path:   o.Path,
		Args:   []string{"tor", "-f", o.Torrc},
		Dir:    o.DataDir,
		Stdout: torlogs,
		Stderr: torlogs,
	}

	// Start TOR command
	err = cmd.Start()
	if err != nil {
		return
	}

	// Wait for bootstrap done
	// Or for context deadline exceed
	select {
	case <-ctx.Done():
		if err := cmd.Process.Kill(); err != nil {
			glog.Warningf("Error killing cmd: %v", err)
		}
		return nil, ctx.Err()
	case <-torlogs.Done():
	}

	// TOR Client
	t = &Client{
		opts:  o,
		cmd:   cmd,
		mutex: new(sync.RWMutex),
	}

	// Update TOR public IP
	// IGNORING errors
	// In result the public IP can be unknown
	if _, err := t.UpdateIPAddress(); err != nil {
		glog.Warningf("Update IP error: %v", err)
	}
	return t, nil
}

var (
	defaultCtlPassword     = "0iermg5m-eo5dfre43"
	defaultCtlPasswordHash = "16:BEA511ABE2DE1E92609493874821BC66776DF252068D2CDE71F09A0CED"
)

// Set Launch options defaults and create
// default data directory and torrc if required.
func setLaunchDefaults(o *LaunchOptions) (*LaunchOptions, error) {
	if o.ProxyAddress == "" {
		o.ProxyAddress = "127.0.0.1"
	}

	if o.ControlAddress == "" {
		o.ControlAddress = "127.0.0.1"
	}

	// We need to know both
	// Also the tor is listening locally so it should not be a problem
	// on not exposed system
	if o.ControlPassword == "" || o.ControlPasswordHash == "" {
		o.ControlPassword = defaultCtlPassword
		o.ControlPasswordHash = defaultCtlPasswordHash
	}

	if o.TempDir == "" {
		o.TempDir = os.TempDir()
	}

	if o.DataDir == "" {
		o.DataDir = filepath.Join(o.TempDir, uuid.NewV4().String())
		err := os.MkdirAll(o.DataDir, os.ModePerm)
		if err != nil {
			return nil, err
		}
	}

	if o.Torrc == "" {
		body, err := TorrcBody(o)
		if err != nil {
			return nil, err
		}

		o.Torrc = filepath.Join(o.DataDir, "torrc")
		err = ioutil.WriteFile(o.Torrc, body, os.ModePerm)
		if err != nil {
			return nil, err
		}
	}

	var err error
	o.Torrc, err = filepath.Abs(o.Torrc)
	if err != nil {
		return nil, err
	}

	return o, nil
}
