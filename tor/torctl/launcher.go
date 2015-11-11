package torctl

import (
	"os"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/golang/glog"
	"github.com/satori/go.uuid"
	"golang.org/x/net/context"

	"github.com/crackcomm/onion/proxyutil"
)

// Launcher - Tor proxy service launcher.
type Launcher struct {
	opts *LauncherOptions

	mutex    *sync.RWMutex
	services map[string]*Client
	count    int
}

// LauncherOptions - Launcher options.
type LauncherOptions struct {
	Limit                int
	MaxExpireSeconds     uint32
	DefaultExpireSeconds uint32
	Defaults             *LaunchOptions
}

// NewLauncher - Creates a new TOR launcher.
func NewLauncher(o *LauncherOptions) *Launcher {
	if o.MaxExpireSeconds == 0 {
		o.MaxExpireSeconds = 120
	}
	if o.DefaultExpireSeconds == 0 {
		o.DefaultExpireSeconds = 0
	}
	if o.Defaults.TempDir == "" {
		o.Defaults.TempDir = os.TempDir()
	}

	// We need to know both
	// Also the tor is listening locally so it should not be a problem
	// on not exposed system
	if o.Defaults.ControlPassword == "" || o.Defaults.ControlPasswordHash == "" {
		o.Defaults.ControlPassword = defaultCtlPassword
		o.Defaults.ControlPasswordHash = defaultCtlPasswordHash
	}

	return &Launcher{
		opts:     o,
		mutex:    new(sync.RWMutex),
		services: make(map[string]*Client),
	}
}

// GetProxy - Gets a proxy.
// First checks if there are running tor services that are not used.
// If yes then use one. If not then check if the limit was not yet reached.
// If limit was exceed return ResourceExhausted error.
// If we can launch a to service then do it.
func (launcher *Launcher) GetProxy(ctx context.Context, req *ProxyRequest) (resp *ProxyResponse, err error) {
	// Sanitize expire seconds
	req.ExpireSeconds = launcher.expireSeconds(req.ExpireSeconds)

	// Get proxy from running
	resp, err = launcher.getProxy(ctx, req)
	if err != nil || resp != nil {
		return
	}

	// Check if we can create more
	if launcher.count >= launcher.opts.Limit {
		return nil, grpc.Errorf(codes.ResourceExhausted, "Instances limit exceeded")
	}

	launcher.mutex.Lock()
	num := launcher.count
	launcher.count = num + 1
	launcher.mutex.Unlock()

	// Launch a tor
	service, err := Launch(ctx, &LaunchOptions{
		Path:                launcher.opts.Defaults.Path,
		Quiet:               launcher.opts.Defaults.Quiet,
		ControlPassword:     launcher.opts.Defaults.ControlPassword,
		ControlPasswordHash: launcher.opts.Defaults.ControlPasswordHash,
		MaxMindDB:           launcher.opts.Defaults.MaxMindDB,
		ProxyPort:           proxyutil.FreePort(),
		ControlPort:         proxyutil.FreePort(),
	})
	if err != nil {
		glog.Warningf("Launch error: %v", err)
		return nil, grpc.Errorf(codes.Internal, "Cannot launch TOR instance")
	}

	// Generate random service ID
	id := uuid.NewV4().String()

	// Add service to map
	launcher.mutex.Lock()

	service.LockInSeconds(req.ExpireSeconds)
	launcher.services[id] = service
	glog.Infof("Launched (tor-%d) - %s", num+1, service.PublicIP())

	launcher.mutex.Unlock()

	return &ProxyResponse{
		Id:            id,
		Endpoint:      service.Endpoint(),
		ExpireSeconds: service.ProxyExpirationSeconds(),
	}, nil
}

// getProxy - Gets but doesnt create on not found.
func (launcher *Launcher) getProxy(ctx context.Context, req *ProxyRequest) (resp *ProxyResponse, err error) {
	launcher.mutex.RLock()
	defer launcher.mutex.RUnlock()

	for id, service := range launcher.services {
		if service.IsBusy() {
			continue
		}

		// Restart service meaning request a new IP address
		// Do it if its possible
		if ok, err := service.RequestNewIdentityAndLock(req.ExpireSeconds); err != nil {
			glog.Warningf("Error requesting new identity: %v", err)
			continue
		} else if !ok && req.NewIp {
			continue
		}

		glog.Infof("Proxy - %s", service.PublicIP())
		// Get service information and return it in response
		return &ProxyResponse{
			Id:            id,
			Endpoint:      service.Endpoint(),
			ExpireSeconds: service.ProxyExpirationSeconds(),
		}, nil
	}

	return nil, nil
}

// Release - Releases a proxy.
func (launcher *Launcher) Release(ctx context.Context, req *ReleaseRequest) (resp *Empty, err error) {
	launcher.mutex.Lock()
	defer launcher.mutex.Unlock()

	// Get tor service by id
	service, ok := launcher.services[req.Id]
	if !ok || service == nil {
		return nil, grpc.Errorf(codes.NotFound, "Service was not found")
	}

	// Release the service
	service.Release()

	return new(Empty), nil
}

// RefreshLock - Refresh proxy lock.
func (launcher *Launcher) RefreshLock(ctx context.Context, req *RefreshProxy) (resp *Empty, err error) {
	launcher.mutex.Lock()
	defer launcher.mutex.Unlock()

	// Get tor service by id
	service, ok := launcher.services[req.Id]
	if !ok || service == nil {
		return nil, grpc.Errorf(codes.NotFound, "Service was not found")
	}

	// Refresh the lock
	service.LockInSeconds(launcher.expireSeconds(req.ExpireSeconds))

	return new(Empty), nil
}

// Close - Kills all TOR instances.
func (launcher *Launcher) Close() (err error) {
	launcher.mutex.Lock()
	defer launcher.mutex.Unlock()

	for _, service := range launcher.services {
		err = service.Kill()
	}

	return
}

func (launcher *Launcher) expireSeconds(s uint32) uint32 {
	if s == 0 {
		return launcher.opts.DefaultExpireSeconds
	} else if s > launcher.opts.MaxExpireSeconds {
		return launcher.opts.MaxExpireSeconds
	}
	return s
}
