package nifcloud

import (
	"fmt"
	"io"
	"os"

	cloudprovider "k8s.io/cloud-provider"
)

// ProviderName is the name of this cloud provider
const ProviderName = "nifcloud"

// Cloud is an implementation of Interface, LoadBalancer and Instances for NIFCLOUD
type Cloud struct {
	client CloudAPIClient
	region string
}

func init() {
	registerMetrics()
	cloudprovider.RegisterCloudProvider(ProviderName, func(_ io.Reader) (cloudprovider.Interface, error) {
		return newNIFCLOUD()
	})
}

func newNIFCLOUD() (cloudprovider.Interface, error) {
	accessKeyID := os.Getenv("NIFCLOUD_ACCESS_KEY_ID")
	secretAccessKey := os.Getenv("NIFCLOUD_SECRET_ACCESS_KEY")
	region := os.Getenv("NIFCLOUD_REGION")
	if accessKeyID == "" {
		return nil, fmt.Errorf(`environment variable "NIFCLOUD_ACCESS_KEY_ID" is required`)
	}
	if secretAccessKey == "" {
		return nil, fmt.Errorf(`environment variable "NIFCLOUD_SECRET_ACCESS_KEY" is required`)
	}
	if region == "" {
		return nil, fmt.Errorf(`environment variable "NIFCLOUD_REGION" is required`)
	}

	return &Cloud{
		client: newNIFCLOUDAPIClient(accessKeyID, secretAccessKey, region),
		region: region,
	}, nil
}

// Initialize provides the cloud with a kubernetes client builder and may spawn goroutines
// to perform housekeeping or run custom controllers specific to the cloud provider.
// Any tasks started here should be cleaned up when the stop channel closes.
func (c *Cloud) Initialize(clientBuilder cloudprovider.ControllerClientBuilder, stop <-chan struct{}) {
}

// LoadBalancer returns an implementation of LoadBalancer for NIFCLOUD
func (c *Cloud) LoadBalancer() (cloudprovider.LoadBalancer, bool) {
	return c, true
}

// Instances returns an implementation of Instances for NIFCLOUD
func (c *Cloud) Instances() (cloudprovider.Instances, bool) {
	return c, true
}

func (c *Cloud) InstancesV2() (cloudprovider.InstancesV2, bool) {
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
