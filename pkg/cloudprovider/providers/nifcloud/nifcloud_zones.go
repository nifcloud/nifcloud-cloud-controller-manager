package nifcloud

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/types"
	cloudprovider "k8s.io/cloud-provider"
)

// GetZone returns the Zone containing the current failure zone and locality region that the program is running in
func (c *Cloud) GetZone(ctx context.Context) (cloudprovider.Zone, error) {
	instanceID, err := getInstanceIDFromGuestInfo()
	if err != nil {
		return cloudprovider.Zone{}, fmt.Errorf("could not get instance id for this node: %w", err)
	}

	instances, err := c.client.DescribeInstancesByInstanceID(ctx, []string{instanceID})
	if err != nil {
		return cloudprovider.Zone{}, fmt.Errorf("cloud not fetch instance info for %q: %w", instanceID, err)
	}

	if err := isSingleInstance(instances, instanceID); err != nil {
		return cloudprovider.Zone{}, err
	}

	return cloudprovider.Zone{
		FailureDomain: instances[0].Zone,
		Region:        c.region,
	}, nil
}

// GetZoneByProviderID returns the Zone containing the current zone and locality region of the node specified by providerID
func (c *Cloud) GetZoneByProviderID(ctx context.Context, providerID string) (cloudprovider.Zone, error) {
	instanceUniqueID, err := getInstanceUniqueIDFromProviderID(providerID)
	if err != nil {
		return cloudprovider.Zone{}, fmt.Errorf("unable to convert provider id %q: %v", providerID, err)
	}

	instances, err := c.client.DescribeInstancesByInstanceUniqueID(ctx, []string{instanceUniqueID})
	if err != nil {
		return cloudprovider.Zone{}, fmt.Errorf("cloud not fetch instance info for %q: %w", instanceUniqueID, err)
	}

	if err := isSingleInstance(instances, instanceUniqueID); err != nil {
		return cloudprovider.Zone{}, err
	}

	return cloudprovider.Zone{
		FailureDomain: instances[0].Zone,
		Region:        c.region,
	}, nil
}

// GetZoneByNodeName returns the Zone containing the current zone and locality region of the node specified by node name
func (c *Cloud) GetZoneByNodeName(ctx context.Context, name types.NodeName) (cloudprovider.Zone, error) {
	instanceID := string(name)
	instances, err := c.client.DescribeInstancesByInstanceID(ctx, []string{instanceID})
	if err != nil {
		return cloudprovider.Zone{}, fmt.Errorf("cloud not fetch instance info for %q: %w", instanceID, err)
	}

	if err := isSingleInstance(instances, instanceID); err != nil {
		return cloudprovider.Zone{}, err
	}

	return cloudprovider.Zone{
		FailureDomain: instances[0].Zone,
		Region:        c.region,
	}, nil
}
