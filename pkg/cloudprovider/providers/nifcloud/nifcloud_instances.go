package nifcloud

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	cloudprovider "k8s.io/cloud-provider"
)

var nifcloudInstanceRegMatch = regexp.MustCompile("^i-[^/]*$")

// NodeAddresses returns the addresses of the specified instance.
func (c *Cloud) NodeAddresses(ctx context.Context, name types.NodeName) ([]v1.NodeAddress, error) {
	instanceID := string(name)
	instances, err := c.client.DescribeInstancesByInstanceID(ctx, []string{instanceID})
	if err != nil {
		return nil, fmt.Errorf("cloud not fetch instance info for %q: %w", instanceID, err)
	}

	if err := isSingleInstance(instances, instanceID); err != nil {
		return nil, err
	}

	return getNodeAddress(instances[0]), nil
}

// NodeAddressesByProviderID returns the addresses of the specified instance.
func (c *Cloud) NodeAddressesByProviderID(ctx context.Context, providerID string) ([]v1.NodeAddress, error) {
	instanceUniqueID, err := getInstanceUniqueIDFromProviderID(providerID)
	if err != nil {
		return nil, fmt.Errorf("unable to convert provider id %q: %v", providerID, err)
	}

	instances, err := c.client.DescribeInstancesByInstanceUniqueID(ctx, []string{instanceUniqueID})
	if err != nil {
		return nil, fmt.Errorf("cloud not fetch instance info for %q: %w", instanceUniqueID, err)
	}

	if err := isSingleInstance(instances, instanceUniqueID); err != nil {
		return nil, err
	}

	return getNodeAddress(instances[0]), nil
}

// InstanceID returns the cloud provider ID of the node with the specified NodeName.
func (c *Cloud) InstanceID(ctx context.Context, name types.NodeName) (string, error) {
	instanceID := string(name)
	instances, err := c.client.DescribeInstancesByInstanceID(ctx, []string{instanceID})
	if err != nil {
		return "", fmt.Errorf("cloud not fetch instance info for %q: %w", instanceID, err)
	}

	if err := isSingleInstance(instances, instanceID); err != nil {
		return "", err
	}

	return fmt.Sprintf(
		"/%s/%s",
		instances[0].Zone,
		instances[0].InstanceUniqueID,
	), nil
}

// InstanceType returns the type of the specified instance.
func (c *Cloud) InstanceType(ctx context.Context, name types.NodeName) (string, error) {
	instanceID := string(name)
	instances, err := c.client.DescribeInstancesByInstanceID(ctx, []string{instanceID})
	if err != nil {
		return "", fmt.Errorf("cloud not fetch instance info for %q: %w", instanceID, err)
	}

	if err := isSingleInstance(instances, instanceID); err != nil {
		return "", err
	}

	return instances[0].InstanceType, nil
}

// InstanceTypeByProviderID returns the type of the specified instance.
func (c *Cloud) InstanceTypeByProviderID(ctx context.Context, providerID string) (string, error) {
	instanceUniqueID, err := getInstanceUniqueIDFromProviderID(providerID)
	if err != nil {
		return "", fmt.Errorf("unable to convert provider id %q: %v", providerID, err)
	}

	instances, err := c.client.DescribeInstancesByInstanceUniqueID(ctx, []string{instanceUniqueID})
	if err != nil {
		return "", fmt.Errorf("cloud not fetch instance info for %q: %w", instanceUniqueID, err)
	}

	if err := isSingleInstance(instances, instanceUniqueID); err != nil {
		return "", err
	}

	return instances[0].InstanceType, nil
}

// AddSSHKeyToAllInstances adds an SSH public key as a legal identity for all instances
func (c *Cloud) AddSSHKeyToAllInstances(ctx context.Context, user string, keyData []byte) error {
	return cloudprovider.NotImplemented
}

// CurrentNodeName returns the name of the node we are currently running on
// On most clouds (e.g. GCE) this is the hostname, so we provide the hostname
func (c *Cloud) CurrentNodeName(ctx context.Context, hostname string) (types.NodeName, error) {
	return types.NodeName(hostname), nil
}

// InstanceExistsByProviderID returns true if the instance for the given provider exists.
func (c *Cloud) InstanceExistsByProviderID(ctx context.Context, providerID string) (bool, error) {
	instanceUniqueID, err := getInstanceUniqueIDFromProviderID(providerID)
	if err != nil {
		return false, fmt.Errorf("unable to convert provider id %q: %v", providerID, err)
	}

	instances, err := c.client.DescribeInstancesByInstanceUniqueID(ctx, []string{instanceUniqueID})
	if err != nil {
		return false, fmt.Errorf("cloud not fetch instance info for %q: %w", instanceUniqueID, err)
	}

	if err := isSingleInstance(instances, instanceUniqueID); err != nil {
		return false, err
	}

	return true, nil
}

// InstanceShutdownByProviderID returns true if the instance is shutdown in cloudprovider
func (c *Cloud) InstanceShutdownByProviderID(ctx context.Context, providerID string) (bool, error) {
	return false, cloudprovider.NotImplemented
}

// InstanceExists returns true if the instance for the given node exists according to the cloud provider.
func (c *Cloud) InstanceExists(ctx context.Context, node *v1.Node) (bool, error) {
	_, err := c.getInstance(ctx, node)
	if err != nil {
		if errors.Is(err, cloudprovider.InstanceNotFound) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// InstanceShutdown returns true if the instance is shutdown according to the cloud provider.
func (c *Cloud) InstanceShutdown(ctx context.Context, node *v1.Node) (bool, error) {
	instance, err := c.getInstance(ctx, node)
	if err != nil {
		return false, err
	}

	if instance.State == "stopped" {
		return true, nil
	}

	return false, nil
}

// InstanceMetadata returns the instance's metadata.
func (c *Cloud) InstanceMetadata(ctx context.Context, node *v1.Node) (*cloudprovider.InstanceMetadata, error) {
	instance, err := c.getInstance(ctx, node)
	if err != nil {
		return nil, err
	}

	return &cloudprovider.InstanceMetadata{
		ProviderID:    fmt.Sprintf("nifcloud:///%s/%s", instance.Zone, instance.InstanceUniqueID),
		InstanceType:  instance.InstanceType,
		NodeAddresses: getNodeAddress(*instance),
		Zone:          instance.Zone,
		Region:        c.region,
	}, nil
}

func (c *Cloud) getInstance(ctx context.Context, node *v1.Node) (*Instance, error) {
	var (
		instances []Instance
		err       error
	)
	if node.Spec.ProviderID != "" {
		instanceUniqueID, err := getInstanceUniqueIDFromProviderID(node.Spec.ProviderID)
		if err != nil {
			return nil, fmt.Errorf("failed to get instance unique id from provider id: %w", err)
		}

		instances, err = c.client.DescribeInstancesByInstanceUniqueID(ctx, []string{instanceUniqueID})
		if err != nil {
			return nil, fmt.Errorf("could not fetch instance info by instance unique id %s: %w", instanceUniqueID, err)
		}
	} else {
		instances, err = c.client.DescribeInstancesByInstanceID(ctx, []string{node.Name})
		if err != nil {
			return nil, fmt.Errorf("could not fetch instance info by node name %s: %w", node.Name, err)
		}
	}

	if err := isSingleInstance(instances, node.Name); err != nil {
		return nil, err
	}

	return &instances[0], nil
}

func getNodeAddress(instance Instance) []v1.NodeAddress {
	address := []v1.NodeAddress{}
	if instance.PublicIPAddress != "" {
		address = append(address, v1.NodeAddress{
			Type:    v1.NodeExternalIP,
			Address: instance.PublicIPAddress,
		})
	}
	if instance.PrivateIPAddress != "" {
		address = append(address, v1.NodeAddress{
			Type:    v1.NodeInternalIP,
			Address: instance.PrivateIPAddress,
		})
	}

	return address
}
