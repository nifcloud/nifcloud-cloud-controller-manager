package nifcloud

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/aokumasan/nifcloud-sdk-go-v2/nifcloud"
	"github.com/aokumasan/nifcloud-sdk-go-v2/service/computing"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/private/protocol/query/queryutil"
	"golang.org/x/sync/errgroup"
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
}

// LoadBalancer is load balancer detail
type LoadBalancer struct {
	Name                          string
	VIP                           string
	AccountingType                string
	NetworkVolume                 int64
	PolicyType                    string
	BalancingType                 int64
	BalancingTargets              []Instance
	LoadBalancerPort              int64
	InstancePort                  int64
	HealthCheckTarget             string
	HealthCheckInterval           int64
	HealthCheckUnhealthyThreshold int64
	Filters                       []Filter
}

// Filter is load balancer filter detail
type Filter struct {
	AddOnFilter bool
	IPAddress   string
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
	SetFilterForLoadBalancer(ctx context.Context, loadBalancer *LoadBalancer) error
}

type nifcloudAPIClient struct {
	client *computing.Client
}

func newNIFCLOUDAPIClient(accessKeyID, secretAccessKey, region string) CloudAPIClient {
	cfg := nifcloud.NewConfig(accessKeyID, secretAccessKey, region)
	return &nifcloudAPIClient{
		client: computing.New(cfg),
	}
}

func (c *nifcloudAPIClient) DescribeInstancesByInstanceID(ctx context.Context, instanceIDs []string) ([]Instance, error) {
	req := c.client.DescribeInstancesRequest(
		&computing.DescribeInstancesInput{
			InstanceId: instanceIDs,
		},
	)
	res, err := req.Send(ctx)
	if err != nil {
		return nil, handleNotFoundError(err)
	}

	if err := checkReservationSet(res.ReservationSet); err != nil {
		return nil, err
	}

	instances := []Instance{}
	for _, instance := range res.ReservationSet[0].InstancesSet {
		instances = append(instances, Instance{
			InstanceID:       nifcloud.StringValue(instance.InstanceId),
			InstanceUniqueID: nifcloud.StringValue(instance.InstanceUniqueId),
			InstanceType:     nifcloud.StringValue(instance.InstanceType),
			PublicIPAddress:  nifcloud.StringValue(instance.IpAddress),
			PrivateIPAddress: nifcloud.StringValue(instance.PrivateIpAddress),
			Zone:             nifcloud.StringValue(instance.Placement.AvailabilityZone),
		})
	}

	return instances, nil
}

func (c *nifcloudAPIClient) DescribeInstancesByInstanceUniqueID(ctx context.Context, instanceUniqueIDs []string) ([]Instance, error) {
	req := c.client.DescribeInstancesRequest(nil)
	if err := req.Request.Build(); err != nil {
		return nil, fmt.Errorf("failed building request: %v", err)
	}
	body := url.Values{
		"Action":  {req.Operation.Name},
		"Version": {req.Metadata.APIVersion},
	}
	if err := queryutil.Parse(body, req.Params, false); err != nil {
		return nil, fmt.Errorf("failed encoding request: %v", err)
	}
	for i, uniqueID := range instanceUniqueIDs {
		body.Set(fmt.Sprintf("InstanceUniqueId.%d", i), uniqueID)
	}
	req.SetBufferBody([]byte(body.Encode()))

	res, err := req.Send(ctx)
	if err != nil {
		return nil, handleNotFoundError(err)
	}

	if err := checkReservationSet(res.ReservationSet); err != nil {
		return nil, err
	}

	instances := []Instance{}
	for _, instance := range res.ReservationSet[0].InstancesSet {
		instances = append(instances, Instance{
			InstanceID:       nifcloud.StringValue(instance.InstanceId),
			InstanceUniqueID: nifcloud.StringValue(instance.InstanceUniqueId),
			InstanceType:     nifcloud.StringValue(instance.InstanceType),
			PublicIPAddress:  nifcloud.StringValue(instance.IpAddress),
			PrivateIPAddress: nifcloud.StringValue(instance.PrivateIpAddress),
			Zone:             nifcloud.StringValue(instance.Placement.AvailabilityZone),
		})
	}

	return instances, nil
}

func (c *nifcloudAPIClient) DescribeLoadBalancers(ctx context.Context, name string) ([]LoadBalancer, error) {
	req := c.client.DescribeLoadBalancersRequest(
		&computing.DescribeLoadBalancersInput{
			LoadBalancerNames: []computing.RequestLoadBalancerNamesStruct{
				{
					LoadBalancerName: nifcloud.String(name),
				},
			},
		},
	)
	res, err := req.Send(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not fetch load balancers info for %q: %v", name, err)
	}

	if len(res.LoadBalancerDescriptions) == 0 {
		return nil, fmt.Errorf("cloud not find load balancer %q: %v", name, err)
	}

	result := []LoadBalancer{}
	for _, lbDesc := range res.LoadBalancerDescriptions {
		lb := LoadBalancer{
			Name:                          nifcloud.StringValue(lbDesc.LoadBalancerName),
			VIP:                           nifcloud.StringValue(lbDesc.DNSName),
			AccountingType:                nifcloud.StringValue(lbDesc.AccountingType),
			NetworkVolume:                 nifcloud.Int64Value(lbDesc.NetworkVolume),
			PolicyType:                    nifcloud.StringValue(lbDesc.PolicyType),
			BalancingType:                 nifcloud.Int64Value(lbDesc.ListenerDescriptions[0].Listener.BalancingType),
			LoadBalancerPort:              nifcloud.Int64Value(lbDesc.ListenerDescriptions[0].Listener.LoadBalancerPort),
			InstancePort:                  nifcloud.Int64Value(lbDesc.ListenerDescriptions[0].Listener.InstancePort),
			HealthCheckTarget:             nifcloud.StringValue(lbDesc.HealthCheck.Target),
			HealthCheckInterval:           nifcloud.Int64Value(lbDesc.HealthCheck.Interval),
			HealthCheckUnhealthyThreshold: nifcloud.Int64Value(lbDesc.HealthCheck.UnhealthyThreshold),
		}

		balancingTargets := []Instance{}
		for _, instance := range lbDesc.Instances {
			balancingTargets = append(balancingTargets,
				Instance{
					InstanceID:       nifcloud.StringValue(instance.InstanceId),
					InstanceUniqueID: nifcloud.StringValue(instance.InstanceUniqueId),
				},
			)
		}
		lb.BalancingTargets = balancingTargets

		filters := []Filter{}
		for _, filter := range lbDesc.Filter.IPAddresses {
			filters = append(filters, Filter{IPAddress: nifcloud.StringValue(filter.IPAddress)})
		}
		lb.Filters = filters

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
		return "", err
	}

	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return c.ConfigureHealthCheck(ctx, loadBalancer)
	})
	eg.Go(func() error {
		return c.RegisterInstancesWithLoadBalancer(ctx, loadBalancer, nil)
	})
	eg.Go(func() error {
		return c.SetFilterForLoadBalancer(ctx, loadBalancer)
	})

	if err := eg.Wait(); err != nil {
		return "", fmt.Errorf(
			"failed to configure load balancer %q (%d -> %d): %w",
			loadBalancer.Name, loadBalancer.LoadBalancerPort,
			loadBalancer.InstancePort, err,
		)
	}

	return vip, nil
}

func (c *nifcloudAPIClient) createLoadBalancer(ctx context.Context, loadBalancer *LoadBalancer) (string, error) {
	if loadBalancer == nil {
		return "", fmt.Errorf("loadBalancer is nil")
	}

	input := &computing.CreateLoadBalancerInput{
		LoadBalancerName: nifcloud.String(loadBalancer.Name),
		Listeners: []computing.RequestListenersStruct{
			{
				LoadBalancerPort: nifcloud.Int64(loadBalancer.LoadBalancerPort),
				InstancePort:     nifcloud.Int64(loadBalancer.InstancePort),
				BalancingType:    nifcloud.Int64(loadBalancer.BalancingType),
			},
		},
	}
	if loadBalancer.AccountingType != "" {
		input.AccountingType = nifcloud.String(loadBalancer.AccountingType)
	}
	if loadBalancer.NetworkVolume != 0 {
		input.NetworkVolume = nifcloud.Int64(loadBalancer.NetworkVolume)
	}
	if loadBalancer.PolicyType != "" {
		input.PolicyType = nifcloud.String(loadBalancer.PolicyType)
	}

	req := c.client.CreateLoadBalancerRequest(input)
	res, err := req.Send(ctx)
	if err != nil {
		return "", fmt.Errorf(
			"could not create new load balancer %q (%d -> %d): %w",
			loadBalancer.Name, loadBalancer.LoadBalancerPort,
			loadBalancer.InstancePort, err,
		)
	}

	return nifcloud.StringValue(res.DNSName), nil
}

func (c *nifcloudAPIClient) RegisterPortWithLoadBalancer(ctx context.Context, loadBalancer *LoadBalancer) error {
	if loadBalancer == nil {
		return fmt.Errorf("loadBalancer is nil")
	}

	if err := c.registerPortWithLoadBalancer(ctx, loadBalancer); err != nil {
		return err
	}

	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return c.ConfigureHealthCheck(ctx, loadBalancer)
	})
	eg.Go(func() error {
		return c.RegisterInstancesWithLoadBalancer(ctx, loadBalancer, nil)
	})
	eg.Go(func() error {
		return c.SetFilterForLoadBalancer(ctx, loadBalancer)
	})

	if err := eg.Wait(); err != nil {
		return fmt.Errorf(
			"failed to configure load balancer %q (%d -> %d): %w",
			loadBalancer.Name, loadBalancer.LoadBalancerPort,
			loadBalancer.InstancePort, err,
		)
	}

	return nil
}

func (c *nifcloudAPIClient) registerPortWithLoadBalancer(ctx context.Context, loadBalancer *LoadBalancer) error {
	if loadBalancer == nil {
		return fmt.Errorf("loadBalancer is nil")
	}

	req := c.client.RegisterPortWithLoadBalancerRequest(
		&computing.RegisterPortWithLoadBalancerInput{
			LoadBalancerName: nifcloud.String(loadBalancer.Name),
			Listeners: []computing.RequestListenersStruct{
				{
					LoadBalancerPort: nifcloud.Int64(loadBalancer.LoadBalancerPort),
					InstancePort:     nifcloud.Int64(loadBalancer.InstancePort),
					BalancingType:    nifcloud.Int64(loadBalancer.BalancingType),
				},
			},
		},
	)
	_, err := req.Send(ctx)
	if err != nil {
		return fmt.Errorf(
			"could not register port with load balancer %q (%d -> %d): %w",
			loadBalancer.Name, loadBalancer.LoadBalancerPort,
			loadBalancer.InstancePort, err,
		)
	}

	return nil
}

func (c *nifcloudAPIClient) ConfigureHealthCheck(ctx context.Context, loadBalancer *LoadBalancer) error {
	if loadBalancer == nil {
		return fmt.Errorf("loadBalancer is nil")
	}

	req := c.client.ConfigureHealthCheckRequest(
		&computing.ConfigureHealthCheckInput{
			LoadBalancerName: nifcloud.String(loadBalancer.Name),
			LoadBalancerPort: nifcloud.Int64(loadBalancer.LoadBalancerPort),
			InstancePort:     nifcloud.Int64(loadBalancer.InstancePort),
			HealthCheck: &computing.RequestHealthCheckStruct{
				Interval:           nifcloud.Int64(loadBalancer.HealthCheckInterval),
				Target:             nifcloud.String(loadBalancer.HealthCheckTarget),
				UnhealthyThreshold: nifcloud.Int64(loadBalancer.HealthCheckUnhealthyThreshold),
			},
		},
	)
	_, err := req.Send(ctx)
	if err != nil {
		return fmt.Errorf(
			"failed to configure health check for %q (%d -> %d): %w",
			loadBalancer.Name, loadBalancer.LoadBalancerPort,
			loadBalancer.InstancePort, err,
		)
	}

	return nil
}

func (c *nifcloudAPIClient) SetFilterForLoadBalancer(ctx context.Context, loadBalancer *LoadBalancer) error {
	if loadBalancer == nil {
		return fmt.Errorf("loadBalancer is nil")
	}

	ipAddresses := []computing.RequestIPAddressesStruct{}
	for _, filter := range loadBalancer.Filters {
		// Skip wildcard
		if filter.IPAddress == filterAnyIPAddresses {
			continue
		}
		ipAddresses = append(ipAddresses, computing.RequestIPAddressesStruct{
			AddOnFilter: nifcloud.Bool(filter.AddOnFilter),
			IPAddress:   nifcloud.String(filter.IPAddress),
		})
	}

	req := c.client.SetFilterForLoadBalancerRequest(
		&computing.SetFilterForLoadBalancerInput{
			LoadBalancerName: nifcloud.String(loadBalancer.Name),
			LoadBalancerPort: nifcloud.Int64(loadBalancer.LoadBalancerPort),
			InstancePort:     nifcloud.Int64(loadBalancer.InstancePort),
			FilterType:       nifcloud.String(loadBalancerFilterType),
			IPAddresses:      ipAddresses,
		},
	)
	_, err := req.Send(ctx)
	if err != nil {
		return fmt.Errorf(
			"failed to set filter for %q (%d -> %d): %w",
			loadBalancer.Name, loadBalancer.LoadBalancerPort,
			loadBalancer.InstancePort, err,
		)
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

	registerInstances := []computing.RequestInstancesStruct{}
	for _, instance := range instances {
		registerInstances = append(registerInstances,
			computing.RequestInstancesStruct{
				InstanceId: nifcloud.String(instance.InstanceID),
			})
	}
	req := c.client.RegisterInstancesWithLoadBalancerRequest(
		&computing.RegisterInstancesWithLoadBalancerInput{
			LoadBalancerName: nifcloud.String(loadBalancer.Name),
			LoadBalancerPort: nifcloud.Int64(loadBalancer.LoadBalancerPort),
			InstancePort:     nifcloud.Int64(loadBalancer.InstancePort),
			Instances:        registerInstances,
		},
	)
	_, err := req.Send(ctx)
	if err != nil {
		return fmt.Errorf(
			"failed to register instances to load balancer %q (%d -> %d): %w",
			loadBalancer.Name, loadBalancer.LoadBalancerPort,
			loadBalancer.InstancePort, err,
		)
	}

	return nil
}

func (c *nifcloudAPIClient) DeleteLoadBalancer(ctx context.Context, loadBalancer *LoadBalancer) error {
	if loadBalancer == nil {
		return fmt.Errorf("loadBalancer is nil")
	}

	req := c.client.DeleteLoadBalancerRequest(
		&computing.DeleteLoadBalancerInput{
			LoadBalancerName: nifcloud.String(loadBalancer.Name),
			LoadBalancerPort: nifcloud.Int64(loadBalancer.LoadBalancerPort),
			InstancePort:     nifcloud.Int64(loadBalancer.InstancePort),
		},
	)
	_, err := req.Send(ctx)
	if err != nil {
		return fmt.Errorf(
			"failed to delete load balancer %q (%d -> %d): %w",
			loadBalancer.Name, loadBalancer.LoadBalancerPort,
			loadBalancer.InstancePort, err,
		)
	}

	return nil
}

func handleNotFoundError(err error) error {
	switch err.(type) {
	case awserr.Error:
		if strings.Contains(err.Error(), "NotFound") {
			return cloudprovider.InstanceNotFound
		}
		return err
	default:
		return err
	}
}

func checkReservationSet(rs []computing.ReservationSetItem) error {
	if len(rs) == 0 {
		return cloudprovider.InstanceNotFound
	}

	if len(rs[0].InstancesSet) == 0 {
		return cloudprovider.InstanceNotFound
	}

	return nil
}
