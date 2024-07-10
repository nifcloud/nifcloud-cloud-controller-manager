//go:generate mockgen -source=$GOFILE -destination=zz_generated.mock_$GOFILE -package=$GOPACKAGE
package nifcloud

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"time"

	"golang.org/x/exp/slices"

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

	securityGroupAppliedWaiterTimeout       = 3 * time.Minute
	elasticLoadBalancerAppliedWaiterTimeout = 10 * time.Minute
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

// ElasticLoadBalancer is elastic load balancer detail
type ElasticLoadBalancer struct {
	AvailabilityZone              string
	Name                          string
	VIP                           string
	AccountingType                string
	Protocol                      string
	NetworkVolume                 int32
	BalancingType                 int32
	BalancingTargets              []Instance
	LoadBalancerPort              int32
	InstancePort                  int32
	HealthCheckTarget             string
	HealthCheckInterval           int32
	HealthCheckUnhealthyThreshold int32
	NetworkInterfaces             []NetworkInterface
}

// NetworkInterface is network interface detail
type NetworkInterface struct {
	NetworkId         string
	NetworkName       string
	IPAddress         string
	SystemIpAddresses []string
	IsVipNetwork      bool
}

// SecurityGroup is security group detail
type SecurityGroup struct {
	GroupName string
	Rules     []SecurityGroupRule
}

// SecurityGroupRule is security group rule detail
type SecurityGroupRule struct {
	IpProtocol string
	FromPort   int32
	ToPort     int32
	InOut      string
	Groups     []string
	IpRanges   []string
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

func (lb *ElasticLoadBalancer) Equals(other ElasticLoadBalancer) bool {
	return lb.Name == other.Name &&
		lb.LoadBalancerPort == other.LoadBalancerPort &&
		lb.InstancePort == other.InstancePort
}

func (lb *ElasticLoadBalancer) String() string {
	return fmt.Sprintf("%s (%d -> %d)", lb.Name, lb.LoadBalancerPort, lb.InstancePort)
}

func (r *SecurityGroupRule) String() string {
	if len(r.Groups) > 0 {
		return fmt.Sprintf("%s %s [%d-%d] : %s", r.InOut, r.IpProtocol, r.FromPort, r.ToPort, r.Groups)
	}
	return fmt.Sprintf("%s %s [%d-%d] : %s", r.InOut, r.IpProtocol, r.FromPort, r.ToPort, r.IpRanges)
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

	// ElasticLoadBalancer
	DescribeElasticLoadBalancers(ctx context.Context, name string) ([]ElasticLoadBalancer, error)
	CreateElasticLoadBalancer(ctx context.Context, loadBalancer *ElasticLoadBalancer) (string, error)
	RegisterPortWithElasticLoadBalancer(ctx context.Context, loadBalancer *ElasticLoadBalancer) error
	ConfigureElasticLoadBalancerHealthCheck(ctx context.Context, elasticLoadBalancer *ElasticLoadBalancer) error
	DeleteElasticLoadBalancer(ctx context.Context, loadBalancer *ElasticLoadBalancer) error
	RegisterInstancesWithElasticLoadBalancer(ctx context.Context, loadBalancer *ElasticLoadBalancer, instances []Instance) error
	DeregisterInstancesFromElasticLoadBalancer(ctx context.Context, loadBalancer *ElasticLoadBalancer, instances []Instance) error

	// SecurityGroup
	DescribeSecurityGroupsByInstanceIDs(ctx context.Context, instanceIDs []string) ([]SecurityGroup, error)
	AuthorizeSecurityGroupIngress(ctx context.Context, securityGroupName string, securityGroupRule *SecurityGroupRule) error
	RevokeSecurityGroupIngress(ctx context.Context, securityGroupName string, securityGroupRule *SecurityGroupRule) error
	WaitSecurityGroupApplied(ctx context.Context, securityGroupName string) error
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
		if IsAPIError(err, errorCodeInstanceNotFound) {
			return nil, cloudprovider.InstanceNotFound
		}
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
		return nil, fmt.Errorf("could not fetch load balancers info for %q: %w", name, err)
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

func (c *nifcloudAPIClient) DescribeElasticLoadBalancers(ctx context.Context, name string) ([]ElasticLoadBalancer, error) {
	input := &computing.NiftyDescribeElasticLoadBalancersInput{
		ElasticLoadBalancers: &types.RequestElasticLoadBalancers{
			ListOfRequestElasticLoadBalancerName: []string{name},
		},
	}
	res, err := c.client.NiftyDescribeElasticLoadBalancers(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("could not fetch load balancers info for %q: %w", name, err)
	}

	result := []ElasticLoadBalancer{}
	for _, elbDesc := range res.NiftyDescribeElasticLoadBalancersResult.ElasticLoadBalancerDescriptions {
		for _, listener := range elbDesc.ElasticLoadBalancerListenerDescriptions {
			elb := ElasticLoadBalancer{
				Name:                          nifcloud.ToString(elbDesc.ElasticLoadBalancerName),
				VIP:                           nifcloud.ToString(elbDesc.DNSName),
				AvailabilityZone:              elbDesc.AvailabilityZones[0],
				AccountingType:                nifcloud.ToString(elbDesc.AccountingType),
				Protocol:                      nifcloud.ToString(listener.Listener.Protocol),
				BalancingType:                 nifcloud.ToInt32(listener.Listener.BalancingType),
				LoadBalancerPort:              nifcloud.ToInt32(listener.Listener.ElasticLoadBalancerPort),
				InstancePort:                  nifcloud.ToInt32(listener.Listener.InstancePort),
				HealthCheckTarget:             nifcloud.ToString(listener.Listener.HealthCheck.Target),
				HealthCheckInterval:           nifcloud.ToInt32(listener.Listener.HealthCheck.Interval),
				HealthCheckUnhealthyThreshold: nifcloud.ToInt32(listener.Listener.HealthCheck.UnhealthyThreshold),
			}

			networkVolume, err := strconv.Atoi(*elbDesc.NetworkVolume)
			if err != nil {
				return nil, err
			}
			elb.NetworkVolume = int32(networkVolume)

			balancingTargets := []Instance{}
			for _, instance := range elbDesc.ElasticLoadBalancerListenerDescriptions[0].Listener.Instances {
				balancingTargets = append(balancingTargets,
					Instance{
						InstanceID:       nifcloud.ToString(instance.InstanceId),
						InstanceUniqueID: nifcloud.ToString(instance.InstanceUniqueId),
					},
				)
			}
			elb.BalancingTargets = balancingTargets

			for _, networkInterfaceDesc := range elbDesc.NetworkInterfaces {
				systemIPAddresses := []string{}
				for _, systemIPAddress := range networkInterfaceDesc.SystemIpAddresses {
					systemIPAddresses = append(systemIPAddresses, nifcloud.ToString(systemIPAddress.SystemIpAddress))
				}
				networkInterface := NetworkInterface{
					NetworkId:         nifcloud.ToString(networkInterfaceDesc.NetworkId),
					NetworkName:       nifcloud.ToString(networkInterfaceDesc.NetworkName),
					IPAddress:         nifcloud.ToString(networkInterfaceDesc.IpAddress),
					SystemIpAddresses: systemIPAddresses,
					IsVipNetwork:      nifcloud.ToBool(networkInterfaceDesc.IsVipNetwork),
				}
				elb.NetworkInterfaces = append(elb.NetworkInterfaces, networkInterface)
			}

			result = append(result, elb)
		}
	}

	return result, nil
}

func (c *nifcloudAPIClient) CreateElasticLoadBalancer(ctx context.Context, elasticLoadBalancer *ElasticLoadBalancer) (string, error) {
	if elasticLoadBalancer == nil {
		return "", fmt.Errorf("loadBalancer is nil")
	}

	vip, err := c.createElasticLoadBalancer(ctx, elasticLoadBalancer)
	if err != nil {
		return "", fmt.Errorf("failed to create load balancer %s: %w", elasticLoadBalancer, err)
	}
	if err := c.WaitElasticLoadBalancerApplied(ctx, elasticLoadBalancer.Name); err != nil {
		return "", err
	}

	if err := c.ConfigureElasticLoadBalancerHealthCheck(ctx, elasticLoadBalancer); err != nil {
		return "", fmt.Errorf("failed to configure health check of load balancer %s: %w", elasticLoadBalancer, err)
	}
	if err := c.WaitElasticLoadBalancerApplied(ctx, elasticLoadBalancer.Name); err != nil {
		return "", err
	}

	if err := c.RegisterInstancesWithElasticLoadBalancer(ctx, elasticLoadBalancer, nil); err != nil {
		return "", fmt.Errorf("failed to register instances with load balancer %s: %w", elasticLoadBalancer, err)
	}
	if err := c.WaitElasticLoadBalancerApplied(ctx, elasticLoadBalancer.Name); err != nil {
		return "", err
	}

	return vip, nil
}

func (c *nifcloudAPIClient) createElasticLoadBalancer(ctx context.Context, elasticLoadBalancer *ElasticLoadBalancer) (string, error) {
	input := &computing.NiftyCreateElasticLoadBalancerInput{
		AvailabilityZones: &types.ListOfRequestAvailabilityZones{
			Member: []string{elasticLoadBalancer.AvailabilityZone},
		},
		ElasticLoadBalancerName: nifcloud.String(elasticLoadBalancer.Name),
		Listeners: &types.ListOfRequestListenersOfNiftyCreateElasticLoadBalancer{
			Member: []types.RequestListenersOfNiftyCreateElasticLoadBalancer{
				{
					ElasticLoadBalancerPort: nifcloud.Int32(elasticLoadBalancer.LoadBalancerPort),
					Protocol:                types.ProtocolOfListenersForNiftyCreateElasticLoadBalancer(elasticLoadBalancer.Protocol),
					InstancePort:            nifcloud.Int32(elasticLoadBalancer.InstancePort),
				},
			},
		},
	}
	if elasticLoadBalancer.BalancingType != 0 {
		input.Listeners.Member[0].BalancingType = nifcloud.Int32(elasticLoadBalancer.BalancingType)
	}
	if elasticLoadBalancer.AccountingType != "" {
		input.AccountingType = types.AccountingTypeOfNiftyCreateElasticLoadBalancerRequest(elasticLoadBalancer.AccountingType)
	}
	if elasticLoadBalancer.NetworkVolume != 0 {
		input.NetworkVolume = nifcloud.Int32(elasticLoadBalancer.NetworkVolume)
	}

	input.NetworkInterface = []types.RequestNetworkInterfaceOfNiftyCreateElasticLoadBalancer{}
	for _, networkInterface := range elasticLoadBalancer.NetworkInterfaces {
		systemIPAdresses := []types.RequestSystemIpAddresses{}
		for _, systemIPAdress := range networkInterface.SystemIpAddresses {
			systemIPAdresses = append(systemIPAdresses, types.RequestSystemIpAddresses{
				SystemIpAddress: nifcloud.String(systemIPAdress),
			})
		}
		input.NetworkInterface = append(input.NetworkInterface, types.RequestNetworkInterfaceOfNiftyCreateElasticLoadBalancer{
			NetworkId:                      nifcloud.String(networkInterface.NetworkId),
			IpAddress:                      nifcloud.String(networkInterface.IPAddress),
			ListOfRequestSystemIpAddresses: systemIPAdresses,
			IsVipNetwork:                   nifcloud.Bool(networkInterface.IsVipNetwork),
		})
	}

	res, err := c.client.NiftyCreateElasticLoadBalancer(ctx, input)
	if err != nil {
		return "", fmt.Errorf("could not create new load balancer %s: %w", elasticLoadBalancer, err)
	}

	return nifcloud.ToString(res.NiftyCreateElasticLoadBalancerResult.DNSName), nil
}

func (c *nifcloudAPIClient) RegisterPortWithElasticLoadBalancer(ctx context.Context, elasticLoadBalancer *ElasticLoadBalancer) error {
	if elasticLoadBalancer == nil {
		return fmt.Errorf("loadBalancer is nil")
	}

	if err := c.registerPortWithElasticLoadBalancer(ctx, elasticLoadBalancer); err != nil {
		return fmt.Errorf("failed to register port with load balancer %s: %w", elasticLoadBalancer.String(), err)
	}
	if err := c.WaitElasticLoadBalancerApplied(ctx, elasticLoadBalancer.Name); err != nil {
		return err
	}

	if err := c.ConfigureElasticLoadBalancerHealthCheck(ctx, elasticLoadBalancer); err != nil {
		return fmt.Errorf("failed to configure health check of load balancer %s: %w", elasticLoadBalancer.String(), err)
	}
	if err := c.WaitElasticLoadBalancerApplied(ctx, elasticLoadBalancer.Name); err != nil {
		return err
	}

	if err := c.RegisterInstancesWithElasticLoadBalancer(ctx, elasticLoadBalancer, nil); err != nil {
		return fmt.Errorf("failed to register instances with load balancer %s: %w", elasticLoadBalancer, err)
	}
	if err := c.WaitElasticLoadBalancerApplied(ctx, elasticLoadBalancer.Name); err != nil {
		return err
	}

	return nil
}

func (c *nifcloudAPIClient) registerPortWithElasticLoadBalancer(ctx context.Context, elasticLoadBalancer *ElasticLoadBalancer) error {
	input := &computing.NiftyRegisterPortWithElasticLoadBalancerInput{
		ElasticLoadBalancerName: nifcloud.String(elasticLoadBalancer.Name),
		Listeners: &types.ListOfRequestListenersOfNiftyRegisterPortWithElasticLoadBalancer{
			Member: []types.RequestListenersOfNiftyRegisterPortWithElasticLoadBalancer{
				{
					ElasticLoadBalancerPort: nifcloud.Int32(elasticLoadBalancer.LoadBalancerPort),
					InstancePort:            nifcloud.Int32(elasticLoadBalancer.InstancePort),
					Protocol:                types.ProtocolOfListenersForNiftyRegisterPortWithElasticLoadBalancer(elasticLoadBalancer.Protocol),
				},
			},
		},
	}
	if elasticLoadBalancer.BalancingType != 0 {
		input.Listeners.Member[0].BalancingType = nifcloud.Int32(elasticLoadBalancer.BalancingType)
	}

	if _, err := c.client.NiftyRegisterPortWithElasticLoadBalancer(ctx, input); err != nil {
		return fmt.Errorf("could not register port with load balancer %s: %w", elasticLoadBalancer.String(), err)
	}

	return nil
}

func (c *nifcloudAPIClient) ConfigureElasticLoadBalancerHealthCheck(ctx context.Context, elasticLoadBalancer *ElasticLoadBalancer) error {
	if elasticLoadBalancer == nil {
		return fmt.Errorf("loadBalancer is nil")
	}

	input := &computing.NiftyConfigureElasticLoadBalancerHealthCheckInput{
		ElasticLoadBalancerName: nifcloud.String(elasticLoadBalancer.Name),
		ElasticLoadBalancerPort: nifcloud.Int32(elasticLoadBalancer.LoadBalancerPort),
		InstancePort:            nifcloud.Int32(elasticLoadBalancer.InstancePort),
		HealthCheck: &types.RequestHealthCheckOfNiftyConfigureElasticLoadBalancerHealthCheck{
			Interval:           nifcloud.Int32(elasticLoadBalancer.HealthCheckInterval),
			Target:             nifcloud.String(elasticLoadBalancer.HealthCheckTarget),
			UnhealthyThreshold: nifcloud.Int32(elasticLoadBalancer.HealthCheckUnhealthyThreshold),
		},
		Protocol: types.ProtocolOfNiftyConfigureElasticLoadBalancerHealthCheckRequest(elasticLoadBalancer.Protocol),
	}
	if _, err := c.client.NiftyConfigureElasticLoadBalancerHealthCheck(ctx, input); err != nil {
		return fmt.Errorf("failed to configure health check for load balancer %s: %w", elasticLoadBalancer, err)
	}

	return nil
}

func (c *nifcloudAPIClient) DeleteElasticLoadBalancer(ctx context.Context, elasticLoadBalancer *ElasticLoadBalancer) error {
	if elasticLoadBalancer == nil {
		return fmt.Errorf("loadBalancer is nil")
	}

	input := &computing.NiftyDeleteElasticLoadBalancerInput{
		ElasticLoadBalancerName: nifcloud.String(elasticLoadBalancer.Name),
		ElasticLoadBalancerPort: nifcloud.Int32(elasticLoadBalancer.LoadBalancerPort),
		InstancePort:            nifcloud.Int32(elasticLoadBalancer.InstancePort),
		Protocol:                types.ProtocolOfNiftyDeleteElasticLoadBalancerRequest(elasticLoadBalancer.Protocol),
	}
	if _, err := c.client.NiftyDeleteElasticLoadBalancer(ctx, input); err != nil {
		return fmt.Errorf("failed to delete load balancer %s: %w", elasticLoadBalancer.String(), err)
	}

	return nil
}

func (c *nifcloudAPIClient) RegisterInstancesWithElasticLoadBalancer(ctx context.Context, elasticLoadBalancer *ElasticLoadBalancer, instances []Instance) error {
	if elasticLoadBalancer == nil {
		return fmt.Errorf("loadBalancer is nil")
	}

	if instances == nil {
		instances = elasticLoadBalancer.BalancingTargets
	}

	registerInstances := []types.RequestInstancesOfNiftyRegisterInstancesWithElasticLoadBalancer{}
	for _, instance := range instances {
		registerInstances = append(registerInstances,
			types.RequestInstancesOfNiftyRegisterInstancesWithElasticLoadBalancer{
				InstanceId: nifcloud.String(instance.InstanceID),
			})
	}

	input := &computing.NiftyRegisterInstancesWithElasticLoadBalancerInput{
		ElasticLoadBalancerName: nifcloud.String(elasticLoadBalancer.Name),
		ElasticLoadBalancerPort: nifcloud.Int32(elasticLoadBalancer.LoadBalancerPort),
		InstancePort:            nifcloud.Int32(elasticLoadBalancer.InstancePort),
		Protocol:                types.ProtocolOfNiftyRegisterInstancesWithElasticLoadBalancerRequest(elasticLoadBalancer.Protocol),
		Instances: &types.ListOfRequestInstancesOfNiftyRegisterInstancesWithElasticLoadBalancer{
			Member: registerInstances,
		},
	}
	if _, err := c.client.NiftyRegisterInstancesWithElasticLoadBalancer(ctx, input); err != nil {
		return fmt.Errorf("failed to register instances to load balancer %s: %w", elasticLoadBalancer, err)
	}

	return nil
}

func (c *nifcloudAPIClient) DeregisterInstancesFromElasticLoadBalancer(ctx context.Context, elasticLoadBalancer *ElasticLoadBalancer, instances []Instance) error {
	if elasticLoadBalancer == nil {
		return fmt.Errorf("loadBalancer is nil")
	}

	deregisterInstances := []types.RequestInstancesOfNiftyDeregisterInstancesFromElasticLoadBalancer{}
	for _, instance := range instances {
		deregisterInstances = append(deregisterInstances,
			types.RequestInstancesOfNiftyDeregisterInstancesFromElasticLoadBalancer{
				InstanceId: nifcloud.String(instance.InstanceID),
			})
	}

	input := &computing.NiftyDeregisterInstancesFromElasticLoadBalancerInput{
		ElasticLoadBalancerName: nifcloud.String(elasticLoadBalancer.Name),
		ElasticLoadBalancerPort: nifcloud.Int32(elasticLoadBalancer.LoadBalancerPort),
		InstancePort:            nifcloud.Int32(elasticLoadBalancer.InstancePort),
		Protocol:                types.ProtocolOfNiftyDeregisterInstancesFromElasticLoadBalancerRequest(elasticLoadBalancer.Protocol),
		Instances: &types.ListOfRequestInstancesOfNiftyDeregisterInstancesFromElasticLoadBalancer{
			Member: deregisterInstances,
		},
	}
	if _, err := c.client.NiftyDeregisterInstancesFromElasticLoadBalancer(ctx, input); err != nil {
		return fmt.Errorf("failed to deregister instances from load balancer %s: %w", elasticLoadBalancer, err)
	}

	return nil
}

func (c *nifcloudAPIClient) WaitElasticLoadBalancerApplied(ctx context.Context, elasticLoadBalancerName string) error {
	waiter := computing.NewElasticLoadBalancerAvailableWaiter(c.client)
	params := &computing.NiftyDescribeElasticLoadBalancersInput{
		ElasticLoadBalancers: &types.RequestElasticLoadBalancers{
			ListOfRequestElasticLoadBalancerName: []string{elasticLoadBalancerName},
		},
	}
	if err := waiter.Wait(ctx, params, elasticLoadBalancerAppliedWaiterTimeout); err != nil {
		return fmt.Errorf("failed waiting elastic load balancer: %w", err)
	}
	return nil
}

func (c *nifcloudAPIClient) DescribeSecurityGroups(ctx context.Context) ([]SecurityGroup, error) {
	res, err := c.client.DescribeSecurityGroups(ctx, &computing.DescribeSecurityGroupsInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to request DescribeSecurityGroups: %w", err)
	}

	securityGroup := []SecurityGroup{}
	for _, rs := range res.SecurityGroupInfo {
		securityGroupRules := []SecurityGroupRule{}
		for _, rule := range rs.IpPermissions {
			groups := []string{}
			ipRanges := []string{}
			for _, group := range rule.Groups {
				groups = append(groups, nifcloud.ToString(group.GroupName))
			}
			for _, ipRange := range rule.IpRanges {
				ipRanges = append(ipRanges, nifcloud.ToString(ipRange.CidrIp))
			}
			securityGroupRules = append(securityGroupRules, SecurityGroupRule{
				IpProtocol: nifcloud.ToString(rule.IpProtocol),
				FromPort:   nifcloud.ToInt32(rule.FromPort),
				ToPort:     nifcloud.ToInt32(rule.ToPort),
				InOut:      nifcloud.ToString(rule.InOut),
				Groups:     groups,
				IpRanges:   ipRanges,
			})
		}
		securityGroup = append(securityGroup, SecurityGroup{
			GroupName: nifcloud.ToString(rs.GroupName),
			Rules:     securityGroupRules,
		})
	}

	return securityGroup, nil
}

func (c *nifcloudAPIClient) DescribeSecurityGroupsByInstanceIDs(ctx context.Context, instanceIDs []string) ([]SecurityGroup, error) {
	res, err := c.client.DescribeSecurityGroups(ctx, &computing.DescribeSecurityGroupsInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to request DescribeSecurityGroups: %w", err)
	}

	securityGroup := []SecurityGroup{}
	for _, rs := range res.SecurityGroupInfo {
		containTargetInstance := false
		for _, instance := range rs.InstancesSet {
			if slices.Contains(instanceIDs, nifcloud.ToString(instance.InstanceId)) {
				containTargetInstance = true
			}
		}
		if !containTargetInstance {
			continue
		}

		securityGroupRules := []SecurityGroupRule{}
		for _, rule := range rs.IpPermissions {
			groups := []string{}
			ipRanges := []string{}
			for _, group := range rule.Groups {
				groups = append(groups, nifcloud.ToString(group.GroupName))
			}
			for _, ipRange := range rule.IpRanges {
				ipRanges = append(ipRanges, nifcloud.ToString(ipRange.CidrIp))
			}
			securityGroupRules = append(securityGroupRules, SecurityGroupRule{
				IpProtocol: nifcloud.ToString(rule.IpProtocol),
				FromPort:   nifcloud.ToInt32(rule.FromPort),
				ToPort:     nifcloud.ToInt32(rule.ToPort),
				InOut:      nifcloud.ToString(rule.InOut),
				Groups:     groups,
				IpRanges:   ipRanges,
			})
		}
		securityGroup = append(securityGroup, SecurityGroup{
			GroupName: nifcloud.ToString(rs.GroupName),
			Rules:     securityGroupRules,
		})
	}

	return securityGroup, nil
}

func (c *nifcloudAPIClient) AuthorizeSecurityGroupIngress(ctx context.Context, securityGroupName string, securityGroupRule *SecurityGroupRule) error {
	// TODO: Support for multiple securityGroupRules
	ipRanges := []types.RequestIpRanges{}
	for _, ipRange := range securityGroupRule.IpRanges {
		ipRanges = append(ipRanges, types.RequestIpRanges{
			CidrIp: &ipRange,
		})
	}

	ipPermissions := []types.RequestIpPermissions{
		{
			IpProtocol:            types.IpProtocolOfIpPermissionsForAuthorizeSecurityGroupIngress(securityGroupRule.IpProtocol),
			FromPort:              nifcloud.Int32(securityGroupRule.FromPort),
			ToPort:                nifcloud.Int32(securityGroupRule.ToPort),
			InOut:                 types.InOutOfIpPermissionsForAuthorizeSecurityGroupIngress(securityGroupRule.InOut),
			ListOfRequestIpRanges: ipRanges,
		},
	}
	if securityGroupRule.FromPort == securityGroupRule.ToPort {
		ipPermissions[0].ToPort = nil
	}

	input := &computing.AuthorizeSecurityGroupIngressInput{
		GroupName:     nifcloud.String(securityGroupName),
		IpPermissions: ipPermissions,
	}
	res, err := c.client.AuthorizeSecurityGroupIngress(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to request AuthorizeSecurityGroupIngress %s: %w", securityGroupRule, err)
	}

	if !nifcloud.ToBool(res.Return) {
		return fmt.Errorf("failed to authorize security group rules %s", securityGroupRule)
	}

	return nil
}

func (c *nifcloudAPIClient) RevokeSecurityGroupIngress(ctx context.Context, securityGroupName string, securityGroupRule *SecurityGroupRule) error {
	// TODO: Support for multiple securityGroupRules
	ipRanges := []types.RequestIpRanges{}
	for _, ipRange := range securityGroupRule.IpRanges {
		ipRanges = append(ipRanges, types.RequestIpRanges{
			CidrIp: &ipRange,
		})
	}
	ipPermissions := []types.RequestIpPermissionsOfRevokeSecurityGroupIngress{
		{
			IpProtocol:            types.IpProtocolOfIpPermissionsForRevokeSecurityGroupIngress(securityGroupRule.IpProtocol),
			FromPort:              nifcloud.Int32(securityGroupRule.FromPort),
			ToPort:                nifcloud.Int32(securityGroupRule.ToPort),
			InOut:                 types.InOutOfIpPermissionsForRevokeSecurityGroupIngress(securityGroupRule.InOut),
			ListOfRequestIpRanges: ipRanges,
		},
	}
	if securityGroupRule.FromPort == securityGroupRule.ToPort {
		ipPermissions[0].ToPort = nil
	}

	input := &computing.RevokeSecurityGroupIngressInput{
		GroupName:     nifcloud.String(securityGroupName),
		IpPermissions: ipPermissions,
	}
	res, err := c.client.RevokeSecurityGroupIngress(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to request RevokeSecurityGroupIngress %s: %w", securityGroupRule, err)
	}

	if !nifcloud.ToBool(res.Return) {
		return fmt.Errorf("failed to revoke security group rules %s", securityGroupRule)
	}

	return nil
}

func (c *nifcloudAPIClient) WaitSecurityGroupApplied(ctx context.Context, securityGroupName string) error {
	waiter := computing.NewSecurityGroupAppliedWaiter(c.client)
	params := &computing.DescribeSecurityGroupsInput{GroupName: []string{securityGroupName}}
	if err := waiter.Wait(ctx, params, securityGroupAppliedWaiterTimeout); err != nil {
		return fmt.Errorf("failed waiting security group: %w", err)
	}
	return nil
}
