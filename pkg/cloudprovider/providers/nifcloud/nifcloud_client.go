package nifcloud

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/aws/smithy-go"
	"github.com/nifcloud/nifcloud-sdk-go/nifcloud"
	"github.com/nifcloud/nifcloud-sdk-go/service/computing"
	"github.com/nifcloud/nifcloud-sdk-go/service/computing/types"
	"github.com/samber/lo"
	cloudprovider "k8s.io/cloud-provider"
)

const (
	// only support filter type: 1 (allow CIDRs)
	loadBalancerFilterType = "1"

	filterAnyIPAddresses = "*.*.*.*"
)

// Instance is instance detail
type Instance struct {
	InstanceID       string
	InstanceUniqueID string
	InstanceType     string
	PublicIPAddress  string
	PrivateIPAddress string
	Zone             string
	State            string
}

// LoadBalancer is load balancer detail
type LoadBalancer struct {
	Name                          string
	VIP                           string
	AccountingType                string
	NetworkVolume                 int32
	PolicyType                    string
	BalancingType                 int32
	BalancingTargets              []Instance
	LoadBalancerPort              int32
	InstancePort                  int32
	HealthCheckTarget             string
	HealthCheckInterval           int32
	HealthCheckUnhealthyThreshold int32
	Filters                       []string
}

// Filter is load balancer filter detail
type Filter struct {
	AddOnFilter bool
	IPAddress   string
}

// Equals method checks whether specified instance is the same
func (i *Instance) Equals(other Instance) bool {
	if i.InstanceUniqueID != "" && other.InstanceUniqueID != "" {
		return i.InstanceUniqueID == other.InstanceUniqueID
	}
	return i.InstanceID == other.InstanceID
}

// Equals method checks whether specified load balancer is the same
func (lb *LoadBalancer) Equals(other LoadBalancer) bool {
	return lb.Name == other.Name &&
		lb.LoadBalancerPort == other.LoadBalancerPort &&
		lb.InstancePort == other.InstancePort
}

func (lb *LoadBalancer) String() string {
	return fmt.Sprintf("%s (%d -> %d)", lb.Name, lb.LoadBalancerPort, lb.InstancePort)
}

// CloudAPIClient is interface
type CloudAPIClient interface {
	// Instance
	DescribeInstancesByInstanceID(ctx context.Context, instanceIDs []string) ([]Instance, error)
	DescribeInstancesByInstanceUniqueID(ctx context.Context, instanceUniqueIDs []string) ([]Instance, error)

	// LoadBalancer
	DescribeLoadBalancers(ctx context.Context, name string) ([]LoadBalancer, error)
	CreateLoadBalancer(ctx context.Context, loadBalancer *LoadBalancer) (string, error)
	RegisterPortWithLoadBalancer(ctx context.Context, loadBalancer *LoadBalancer) error
	DeleteLoadBalancer(ctx context.Context, loadBalancer *LoadBalancer) error
	RegisterInstancesWithLoadBalancer(ctx context.Context, loadBalancer *LoadBalancer, instances []Instance) error
	DeregisterInstancesFromLoadBalancer(ctx context.Context, loadBalancer *LoadBalancer, instances []Instance) error
	SetFilterForLoadBalancer(ctx context.Context, loadBalancer *LoadBalancer, filters []Filter) error
}

type nifcloudAPIClient struct {
	client *computing.Client
}

func newNIFCLOUDAPIClient(accessKeyID, secretAccessKey, region string) CloudAPIClient {
	cfg := nifcloud.NewConfig(accessKeyID, secretAccessKey, region)
	return &nifcloudAPIClient{
		client: computing.NewFromConfig(cfg),
	}
}

func (c *nifcloudAPIClient) DescribeInstancesByInstanceID(ctx context.Context, instanceIDs []string) ([]Instance, error) {
	res, err := c.client.DescribeInstances(ctx, &computing.DescribeInstancesInput{InstanceId: instanceIDs})
	if err != nil {
		var awsErr smithy.APIError
		if errors.As(err, &awsErr) && awsErr.ErrorCode() == "Client.InvalidParameterNotFound.Instance" {
			return nil, cloudprovider.InstanceNotFound
		}
		return nil, fmt.Errorf("failed to call DescribeInstances with instance ids %v: %w", instanceIDs, err)
	}

	instances := []Instance{}
	for _, rs := range res.ReservationSet {
		if len(rs.InstancesSet) == 0 {
			return nil, fmt.Errorf("instances set is empty")
		}
		instance := rs.InstancesSet[0]
		instances = append(instances, Instance{
			InstanceID:       nifcloud.ToString(instance.InstanceId),
			InstanceUniqueID: nifcloud.ToString(instance.InstanceUniqueId),
			InstanceType:     nifcloud.ToString(instance.InstanceType),
			PublicIPAddress:  nifcloud.ToString(instance.IpAddress),
			PrivateIPAddress: nifcloud.ToString(instance.PrivateIpAddress),
			Zone:             nifcloud.ToString(instance.Placement.AvailabilityZone),
			State:            nifcloud.ToString(instance.InstanceState.Name),
		})
	}

	if len(instances) == 0 {
		return nil, cloudprovider.InstanceNotFound
	}

	return instances, nil
}

func (c *nifcloudAPIClient) DescribeInstancesByInstanceUniqueID(ctx context.Context, instanceUniqueIDs []string) ([]Instance, error) {
	res, err := c.client.DescribeInstances(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to call DescribeInstances API: %w", err)
	}

	instances := []Instance{}
	for _, rs := range res.ReservationSet {
		if len(rs.InstancesSet) == 0 {
			return nil, fmt.Errorf("instances set is empty")
		}
		instance := rs.InstancesSet[0]
		if !lo.Contains(instanceUniqueIDs, nifcloud.ToString(instance.InstanceUniqueId)) {
			continue
		}
		instances = append(instances, Instance{
			InstanceID:       nifcloud.ToString(instance.InstanceId),
			InstanceUniqueID: nifcloud.ToString(instance.InstanceUniqueId),
			InstanceType:     nifcloud.ToString(instance.InstanceType),
			PublicIPAddress:  nifcloud.ToString(instance.IpAddress),
			PrivateIPAddress: nifcloud.ToString(instance.PrivateIpAddress),
			Zone:             nifcloud.ToString(instance.Placement.AvailabilityZone),
			State:            nifcloud.ToString(instance.InstanceState.Name),
		})
	}

	if len(instances) == 0 {
		return nil, cloudprovider.InstanceNotFound
	}

	return instances, nil
}

func (c *nifcloudAPIClient) DescribeLoadBalancers(ctx context.Context, name string) ([]LoadBalancer, error) {
	input := &computing.DescribeLoadBalancersInput{
		LoadBalancerNames: &types.ListOfRequestLoadBalancerNames{
			Member: []types.RequestLoadBalancerNames{
				{
					LoadBalancerName: nifcloud.String(name),
				},
			},
		},
	}
	res, err := c.client.DescribeLoadBalancers(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("could not fetch load balancers info for %q: %v", name, err)
	}

	result := []LoadBalancer{}
	for _, lbDesc := range res.DescribeLoadBalancersResult.LoadBalancerDescriptions {
		lb := LoadBalancer{
			Name:                          nifcloud.ToString(lbDesc.LoadBalancerName),
			VIP:                           nifcloud.ToString(lbDesc.DNSName),
			AccountingType:                nifcloud.ToString(lbDesc.AccountingType),
			NetworkVolume:                 nifcloud.ToInt32(lbDesc.NetworkVolume),
			PolicyType:                    nifcloud.ToString(lbDesc.PolicyType),
			BalancingType:                 nifcloud.ToInt32(lbDesc.ListenerDescriptions[0].Listener.BalancingType),
			LoadBalancerPort:              nifcloud.ToInt32(lbDesc.ListenerDescriptions[0].Listener.LoadBalancerPort),
			InstancePort:                  nifcloud.ToInt32(lbDesc.ListenerDescriptions[0].Listener.InstancePort),
			HealthCheckTarget:             nifcloud.ToString(lbDesc.HealthCheck.Target),
			HealthCheckInterval:           nifcloud.ToInt32(lbDesc.HealthCheck.Interval),
			HealthCheckUnhealthyThreshold: nifcloud.ToInt32(lbDesc.HealthCheck.UnhealthyThreshold),
		}

		balancingTargets := []Instance{}
		for _, instance := range lbDesc.Instances {
			balancingTargets = append(balancingTargets,
				Instance{
					InstanceID:       nifcloud.ToString(instance.InstanceId),
					InstanceUniqueID: nifcloud.ToString(instance.InstanceUniqueId),
				},
			)
		}
		lb.BalancingTargets = balancingTargets

		filters := []string{}
		for _, filter := range lbDesc.Filter.IPAddresses {
			if nifcloud.ToString(filter.IPAddress) == filterAnyIPAddresses {
				continue
			}
			filters = append(filters, nifcloud.ToString(filter.IPAddress))
		}
		lb.Filters = sort.StringSlice(filters)

		result = append(result, lb)
	}

	return result, nil
}

func (c *nifcloudAPIClient) CreateLoadBalancer(ctx context.Context, loadBalancer *LoadBalancer) (string, error) {
	if loadBalancer == nil {
		return "", fmt.Errorf("loadBalancer is nil")
	}

	vip, err := c.createLoadBalancer(ctx, loadBalancer)
	if err != nil {
		return "", fmt.Errorf("failed to create load balancer %s: %w", loadBalancer, err)
	}

	if err := c.ConfigureHealthCheck(ctx, loadBalancer); err != nil {
		return "", fmt.Errorf("failed to configure health check of load balancer %s: %w", loadBalancer, err)
	}

	if err := c.RegisterInstancesWithLoadBalancer(ctx, loadBalancer, nil); err != nil {
		return "", fmt.Errorf("failed to register instances with load balancer %s: %w", loadBalancer, err)
	}

	if err := c.SetFilterForLoadBalancer(ctx, loadBalancer, nil); err != nil {
		return "", fmt.Errorf("failed to set filter for load balancer %s: %w", loadBalancer, err)
	}

	return vip, nil
}

func (c *nifcloudAPIClient) createLoadBalancer(ctx context.Context, loadBalancer *LoadBalancer) (string, error) {
	input := &computing.CreateLoadBalancerInput{
		LoadBalancerName: nifcloud.String(loadBalancer.Name),
		Listeners: &types.ListOfRequestListeners{
			Member: []types.RequestListeners{
				{
					LoadBalancerPort: nifcloud.Int32(loadBalancer.LoadBalancerPort),
					InstancePort:     nifcloud.Int32(loadBalancer.InstancePort),
				},
			},
		},
	}
	if loadBalancer.BalancingType != 0 {
		input.Listeners.Member[0].BalancingType = nifcloud.Int32(loadBalancer.BalancingType)
	}
	if loadBalancer.AccountingType != "" {
		input.AccountingType = types.AccountingTypeOfCreateLoadBalancerRequest(loadBalancer.AccountingType)
	}
	if loadBalancer.NetworkVolume != 0 {
		input.NetworkVolume = nifcloud.Int32(loadBalancer.NetworkVolume)
	}
	if loadBalancer.PolicyType != "" {
		input.PolicyType = types.PolicyTypeOfCreateLoadBalancerRequest(loadBalancer.PolicyType)
	}

	res, err := c.client.CreateLoadBalancer(ctx, input)
	if err != nil {
		return "", fmt.Errorf("could not create new load balancer %s: %w", loadBalancer, err)
	}

	return nifcloud.ToString(res.CreateLoadBalancerResult.DNSName), nil
}

func (c *nifcloudAPIClient) RegisterPortWithLoadBalancer(ctx context.Context, loadBalancer *LoadBalancer) error {
	if loadBalancer == nil {
		return fmt.Errorf("loadBalancer is nil")
	}

	if err := c.registerPortWithLoadBalancer(ctx, loadBalancer); err != nil {
		return fmt.Errorf("failed to register port with load balancer %s: %w", loadBalancer.String(), err)
	}

	if err := c.ConfigureHealthCheck(ctx, loadBalancer); err != nil {
		return fmt.Errorf("failed to configure health check of load balancer %s: %w", loadBalancer.String(), err)
	}

	if err := c.RegisterInstancesWithLoadBalancer(ctx, loadBalancer, nil); err != nil {
		return fmt.Errorf("failed to register instances with load balancer %s: %w", loadBalancer, err)
	}

	if err := c.SetFilterForLoadBalancer(ctx, loadBalancer, nil); err != nil {
		return fmt.Errorf("failed to set filter for load balancer %s: %w", loadBalancer, err)
	}

	return nil
}

func (c *nifcloudAPIClient) registerPortWithLoadBalancer(ctx context.Context, loadBalancer *LoadBalancer) error {
	input := &computing.RegisterPortWithLoadBalancerInput{
		LoadBalancerName: nifcloud.String(loadBalancer.Name),
		Listeners: &types.ListOfRequestListenersOfRegisterPortWithLoadBalancer{
			Member: []types.RequestListenersOfRegisterPortWithLoadBalancer{
				{
					LoadBalancerPort: nifcloud.Int32(loadBalancer.LoadBalancerPort),
					InstancePort:     nifcloud.Int32(loadBalancer.InstancePort),
				},
			},
		},
	}
	if loadBalancer.BalancingType != 0 {
		input.Listeners.Member[0].BalancingType = nifcloud.Int32(loadBalancer.BalancingType)
	}

	if _, err := c.client.RegisterPortWithLoadBalancer(ctx, input); err != nil {
		return fmt.Errorf("could not register port with load balancer %s: %w", loadBalancer.String(), err)
	}

	return nil
}

func (c *nifcloudAPIClient) ConfigureHealthCheck(ctx context.Context, loadBalancer *LoadBalancer) error {
	if loadBalancer == nil {
		return fmt.Errorf("loadBalancer is nil")
	}

	input := &computing.ConfigureHealthCheckInput{
		LoadBalancerName: nifcloud.String(loadBalancer.Name),
		LoadBalancerPort: nifcloud.Int32(loadBalancer.LoadBalancerPort),
		InstancePort:     nifcloud.Int32(loadBalancer.InstancePort),
		HealthCheck: &types.RequestHealthCheck{
			Interval:           nifcloud.Int32(loadBalancer.HealthCheckInterval),
			Target:             nifcloud.String(loadBalancer.HealthCheckTarget),
			UnhealthyThreshold: nifcloud.Int32(loadBalancer.HealthCheckUnhealthyThreshold),
		},
	}
	if _, err := c.client.ConfigureHealthCheck(ctx, input); err != nil {
		return fmt.Errorf("failed to configure health check for load balancer %s: %w", loadBalancer, err)
	}

	return nil
}

func (c *nifcloudAPIClient) SetFilterForLoadBalancer(ctx context.Context, loadBalancer *LoadBalancer, filters []Filter) error {
	if loadBalancer == nil {
		return fmt.Errorf("loadBalancer is nil")
	}

	if filters == nil {
		// if filters is nil, authorize all of LoadBalancer.Filters
		filters = []Filter{}
		for _, f := range loadBalancer.Filters {
			filters = append(filters, Filter{AddOnFilter: true, IPAddress: f})
		}
	}

	ipAddresses := []types.RequestIPAddresses{}
	for _, filter := range filters {
		// Skip wildcard
		if filter.IPAddress == filterAnyIPAddresses {
			continue
		}
		ipAddresses = append(ipAddresses, types.RequestIPAddresses{
			AddOnFilter: nifcloud.Bool(filter.AddOnFilter),
			IPAddress:   nifcloud.String(filter.IPAddress),
		})
	}

	input := &computing.SetFilterForLoadBalancerInput{
		LoadBalancerName: nifcloud.String(loadBalancer.Name),
		LoadBalancerPort: nifcloud.Int32(loadBalancer.LoadBalancerPort),
		InstancePort:     nifcloud.Int32(loadBalancer.InstancePort),
		FilterType:       types.FilterTypeOfSetFilterForLoadBalancerRequest(loadBalancerFilterType),
		IPAddresses: &types.ListOfRequestIPAddresses{
			Member: ipAddresses,
		},
	}
	if _, err := c.client.SetFilterForLoadBalancer(ctx, input); err != nil {
		return fmt.Errorf("failed to set filter for load balancer %s: %w", loadBalancer.String(), err)
	}

	return nil
}

func (c *nifcloudAPIClient) RegisterInstancesWithLoadBalancer(ctx context.Context, loadBalancer *LoadBalancer, instances []Instance) error {
	if loadBalancer == nil {
		return fmt.Errorf("loadBalancer is nil")
	}

	if instances == nil {
		instances = loadBalancer.BalancingTargets
	}

	registerInstances := []types.RequestInstances{}
	for _, instance := range instances {
		registerInstances = append(registerInstances,
			types.RequestInstances{
				InstanceId: nifcloud.String(instance.InstanceID),
			})
	}

	input := &computing.RegisterInstancesWithLoadBalancerInput{
		LoadBalancerName: nifcloud.String(loadBalancer.Name),
		LoadBalancerPort: nifcloud.Int32(loadBalancer.LoadBalancerPort),
		InstancePort:     nifcloud.Int32(loadBalancer.InstancePort),
		Instances: &types.ListOfRequestInstances{
			Member: registerInstances,
		},
	}
	if _, err := c.client.RegisterInstancesWithLoadBalancer(ctx, input); err != nil {
		return fmt.Errorf("failed to register instances to load balancer %s: %w", loadBalancer, err)
	}

	return nil
}

func (c *nifcloudAPIClient) DeregisterInstancesFromLoadBalancer(ctx context.Context, loadBalancer *LoadBalancer, instances []Instance) error {
	if loadBalancer == nil {
		return fmt.Errorf("loadBalancer is nil")
	}

	deregisterInstances := []types.RequestInstances{}
	for _, instance := range instances {
		deregisterInstances = append(deregisterInstances,
			types.RequestInstances{
				InstanceId: nifcloud.String(instance.InstanceID),
			})
	}

	input := &computing.DeregisterInstancesFromLoadBalancerInput{
		LoadBalancerName: nifcloud.String(loadBalancer.Name),
		LoadBalancerPort: nifcloud.Int32(loadBalancer.LoadBalancerPort),
		InstancePort:     nifcloud.Int32(loadBalancer.InstancePort),
		Instances: &types.ListOfRequestInstances{
			Member: deregisterInstances,
		},
	}
	if _, err := c.client.DeregisterInstancesFromLoadBalancer(ctx, input); err != nil {
		return fmt.Errorf("failed to deregister instances from load balancer %s: %w", loadBalancer, err)
	}

	return nil
}

func (c *nifcloudAPIClient) DeleteLoadBalancer(ctx context.Context, loadBalancer *LoadBalancer) error {
	if loadBalancer == nil {
		return fmt.Errorf("loadBalancer is nil")
	}

	input := &computing.DeleteLoadBalancerInput{
		LoadBalancerName: nifcloud.String(loadBalancer.Name),
		LoadBalancerPort: nifcloud.Int32(loadBalancer.LoadBalancerPort),
		InstancePort:     nifcloud.Int32(loadBalancer.InstancePort),
	}
	if _, err := c.client.DeleteLoadBalancer(ctx, input); err != nil {
		return fmt.Errorf("failed to delete load balancer %s: %w", loadBalancer.String(), err)
	}

	return nil
}
