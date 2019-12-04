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
	cloudprovider "k8s.io/cloud-provider"
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
