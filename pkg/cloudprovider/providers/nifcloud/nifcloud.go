package nifcloud

import (
	"io"

	cloudprovider "k8s.io/cloud-provider"
)

// ProviderName is the name of this cloud provider
const ProviderName = "nifcloud"

// Cloud is an implementation of Interface, LoadBalancer and Instances for NIFCLOUD
type Cloud struct {
}

func init() {
	registerMetrics()
	cloudprovider.RegisterCloudProvider(ProviderName, func(config io.Reader) (cloudprovider.Interface, error) {
		return newNIFCLOUD()
	})
}

func newNIFCLOUD() (cloudprovider.Interface, error) {
	return &Cloud{}, nil
}

// Initialize provides the cloud with a kubernetes client builder and may spawn goroutines
// to perform housekeeping or run custom controllers specific to the cloud provider.
// Any tasks started here should be cleaned up when the stop channel closes.
func (c *Cloud) Initialize(clientBuilder cloudprovider.ControllerClientBuilder, stop <-chan struct{}) {
}

// LoadBalancer returns an implementation of LoadBalancer for NIFCLOUD
func (c *Cloud) LoadBalancer() (cloudprovider.LoadBalancer, bool) {
	return nil, false
}

// Instances returns an implementation of Instances for NIFCLOUD
func (c *Cloud) Instances() (cloudprovider.Instances, bool) {
	return c, true
}

// Zones returns an implementation of Zones for NIFCLOUD
func (c *Cloud) Zones() (cloudprovider.Zones, bool) {
	return c, true
}

// Clusters returns a clusters interface.  Also returns true if the interface is supported, false otherwise.
func (c *Cloud) Clusters() (cloudprovider.Clusters, bool) {
	return nil, false
}

// Routes returns a routes interface along with whether the interface is supported.
func (c *Cloud) Routes() (cloudprovider.Routes, bool) {
	return nil, false
}

// ProviderName returns the cloud provider ID.
func (c *Cloud) ProviderName() string {
	return ProviderName
}

// HasClusterID returns true if a ClusterID is required and set
func (c *Cloud) HasClusterID() bool {
	return true
}
