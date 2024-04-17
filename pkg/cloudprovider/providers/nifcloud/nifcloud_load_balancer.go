package nifcloud

import (
	"context"
	"fmt"
	"strings"

	v1 "k8s.io/api/core/v1"
)

const (
	// limits for NIFCLOUD load balancer
	maxLoadBalancerNameLength   = 15
	maxPortCountPerLoadBalancer = 3

	// default health check parameter values
	defaultHealthCheckInterval           = 10
	defaultHealthCheckUnhealthyThreshold = 1
	defaultHealthCheckTarget             = "TCP"

	// default network interface
	elasticLoadBalancerDefaultNetworkInterface = commonGlobalNetworkID

	// ServiceAnnotationLoadBalancerNetworkVolume is the annotation that specify network volume for load balancer
	// valid volume is 10, 20, ..., 2000
	// See https://pfs.nifcloud.com/api/rest/CreateLoadBalancer.htm
	ServiceAnnotationLoadBalancerNetworkVolume = "service.beta.kubernetes.io/nifcloud-load-balancer-network-volume"

	// ServiceAnnotationLoadBalancerAccountingType is the annotation that specify accounting type for load balancer
	// 1: monthly, 2: pay-per-use
	// See https://pfs.nifcloud.com/api/rest/CreateLoadBalancer.htm
	ServiceAnnotationLoadBalancerAccountingType = "service.beta.kubernetes.io/nifcloud-load-balancer-accounting-type"

	// ServiceAnnotationLoadBalancerPolicyType is the annotation that specify policy type for load balancer
	// valid values are 'standard' or 'ats'
	// See https://pfs.nifcloud.com/api/rest/CreateLoadBalancer.htm
	ServiceAnnotationLoadBalancerPolicyType = "service.beta.kubernetes.io/nifcloud-load-balancer-policy-type"

	// ServiceAnnotationLoadBalancerBalancingType is the annotation that specify balancing type for load balancer
	// 1: Round-Robin, 2: Least-Connection
	// See https://pfs.nifcloud.com/api/rest/CreateLoadBalancer.htm
	ServiceAnnotationLoadBalancerBalancingType = "service.beta.kubernetes.io/nifcloud-load-balancer-balancing-type"

	// ServiceAnnotationLoadBalancerHCProtocol is the annotation that specify health check protocol for load balancer
	// valid values are 'TCP' or 'ICMP'
	// See https://pfs.nifcloud.com/api/rest/ConfigureHealthCheck.htm
	ServiceAnnotationLoadBalancerHCProtocol = "service.beta.kubernetes.io/nifcloud-load-balancer-healthcheck-protocol"

	// ServiceAnnotationLoadBalancerHCUnhealthyThreshold is the annotation that specify the number of unsuccessfull
	// health checks count required for a backend to be considered unhealthy for traffic
	// See https://pfs.nifcloud.com/api/rest/ConfigureHealthCheck.htm
	ServiceAnnotationLoadBalancerHCUnhealthyThreshold = "service.beta.kubernetes.io/nifcloud-load-balancer-healthcheck-unhealthy-threshold"

	// ServiceAnnotationLoadBalancerHCInterval is the annotation that specify interval seconds for health check
	// See https://pfs.nifcloud.com/api/rest/ConfigureHealthCheck.htm
	ServiceAnnotationLoadBalancerHCInterval = "service.beta.kubernetes.io/nifcloud-load-balancer-healthcheck-interval"

	// ServiceAnnotationLoadBalancerType is the annotation that specify using load balancer type
	// valid values are 'lb'(default) or 'elb'
	ServiceAnnotationLoadBalancerType = "service.beta.kubernetes.io/nifcloud-load-balancer-type"

	// ServiceAnnotationLoadBalancerNetworkInterface(1-2) is the annotation that specify network interface of elastic load balancer
	// net-COMMON_GLOBAL, net-COMMON_PRIVATE or network ID of private LAN
	ServiceAnnotationLoadBalancerNetworkInterface1 = "service.beta.kubernetes.io/nifcloud-load-balancer-network-interface-1-network-id"
	ServiceAnnotationLoadBalancerNetworkInterface2 = "service.beta.kubernetes.io/nifcloud-load-balancer-network-interface-2-network-id"

	// ServiceAnnotationLoadBalancerNetworkInterface(1-2)IPAddress is the annotation that specify IPAdress of elastic load balancer
	// Set IP address only when corresponding network interface is private
	ServiceAnnotationLoadBalancerNetworkInterface1IPAddress = "service.beta.kubernetes.io/nifcloud-load-balancer-network-interface-1-ip-address"
	ServiceAnnotationLoadBalancerNetworkInterface2IPAddress = "service.beta.kubernetes.io/nifcloud-load-balancer-network-interface-2-ip-address"

	// ServiceAnnotationLoadBalancerNetworkInterface(1-2)SystemIPAddresses is the annotation that specify SystemIPAdresses of elastic load balancer
	// Set system IP address only when corresponding network interface is private
	ServiceAnnotationLoadBalancerNetworkInterface1SystemIPAddresses = "service.beta.kubernetes.io/nifcloud-load-balancer-network-interface-1-system-ip-addresses"
	ServiceAnnotationLoadBalancerNetworkInterface2SystemIPAddresses = "service.beta.kubernetes.io/nifcloud-load-balancer-network-interface-2-system-ip-addresses"

	// ServiceAnnotationLoadBalancerVipNetwork is the annotation that specify VIP network
	// valid values are '1' or '2'
	ServiceAnnotationLoadBalancerVipNetwork = "service.beta.kubernetes.io/nifcloud-load-balancer-vip-network"
)

// GetLoadBalancer returns whether the specified load balancer exists, and if so, what its status is
func (c *Cloud) GetLoadBalancer(ctx context.Context, clusterName string, service *v1.Service) (status *v1.LoadBalancerStatus, exists bool, err error) {
	if isElasticLoadBalancer(service.Annotations) {
		return c.getElasticLoadBalancer(ctx, clusterName, service)
	}
	if isL4LoadBalancer(service.Annotations) {
		return c.getL4LoadBalancer(ctx, clusterName, service)
	}
	return nil, false, fmt.Errorf("the load balancer type is not supported")
}

// GetLoadBalancerName returns the name of the load balancer
func (c *Cloud) GetLoadBalancerName(ctx context.Context, clusterName string, service *v1.Service) string {
	return strings.Replace(string(service.UID), "-", "", -1)[:maxLoadBalancerNameLength]
}

// EnsureLoadBalancer creates a new load balancer 'name', or updates the existing one. Returns the status of the balancer
func (c *Cloud) EnsureLoadBalancer(ctx context.Context, clusterName string, service *v1.Service, nodes []*v1.Node) (*v1.LoadBalancerStatus, error) {
	portCount := len(service.Spec.Ports)
	if portCount == 0 {
		return nil, fmt.Errorf("requested load balancer with no ports")
	}
	if portCount > maxPortCountPerLoadBalancer {
		return nil, fmt.Errorf("cannot create load balancer with %d ports. max port count is %d", portCount, maxPortCountPerLoadBalancer)
	}
	if service.Spec.LoadBalancerIP != "" {
		return nil, fmt.Errorf("LoadBalancerIP cannot be specified for NIFCLOUD load balancer")
	}

	// check nodes exist
	instanceIDs := make([]string, len(nodes))
	for i, node := range nodes {
		instanceIDs[i] = node.GetName()
	}
	instances, err := c.client.DescribeInstancesByInstanceID(ctx, instanceIDs)
	if err != nil {
		return nil, fmt.Errorf("could not fetch instances info for %v: %w", instanceIDs, err)
	}

	loadBalancerName := c.GetLoadBalancerName(ctx, clusterName, service)

	if isElasticLoadBalancer(service.Annotations) {
		elb, err := NewElasticLoadBalancerFromService(loadBalancerName, instances, service)
		if err != nil {
			return nil, err
		}
		return c.ensureElasticLoadBalancer(ctx, loadBalancerName, elb)
	}
	if isL4LoadBalancer(service.Annotations) {
		l4lb, err := NewL4LoadBalancerFromService(loadBalancerName, instances, service)
		if err != nil {
			return nil, err
		}
		return c.ensureL4LoadBalancer(ctx, loadBalancerName, l4lb)
	}
	return nil, fmt.Errorf("the load balancer type is not supported")
}

// UpdateLoadBalancer updates hosts under the specified load balancer
func (c *Cloud) UpdateLoadBalancer(ctx context.Context, clusterName string, service *v1.Service, nodes []*v1.Node) error {
	if isElasticLoadBalancer(service.Annotations) {
		return c.updateElasticLoadBalancer(ctx, clusterName, service, nodes)
	}
	if isL4LoadBalancer(service.Annotations) {
		return c.updateL4LoadBalancer(ctx, clusterName, service, nodes)
	}
	return fmt.Errorf("the load balancer type is not supported")
}

// EnsureLoadBalancerDeleted deletes the specified load balancer if it exists
func (c *Cloud) EnsureLoadBalancerDeleted(ctx context.Context, clusterName string, service *v1.Service) error {
	if isElasticLoadBalancer(service.Annotations) {
		return c.ensureElasticLoadBalancerDeleted(ctx, clusterName, service)
	}
	if isL4LoadBalancer(service.Annotations) {
		return c.ensureL4LoadBalancerDeleted(ctx, clusterName, service)
	}
	return fmt.Errorf("the load balancer type is not supported")
}

func toLoadBalancerStatus(vip string) *v1.LoadBalancerStatus {
	return &v1.LoadBalancerStatus{
		Ingress: []v1.LoadBalancerIngress{
			{
				IP: vip,
			},
		},
	}
}
