package nifcloud

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/exp/slices"
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

	// ServiceAnnotationLoadBalancerHCUnhealthyThreshold is the annotation that specify the number of unsuccessful
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

var allowedElasticLoadBalancerNetworkVolume = []string{"10", "20", "30", "40", "100", "200", "300", "400", "500"}
var allowedL4LoadBalancerNetworkVolume = []string{
	"10", "20", "30", "40", "100", "200", "300", "400", "500", "600", "700", "800", "900", "1000",
	"1100", "1200", "1300", "1400", "1500", "1600", "1700", "1800", "1900", "2000",
}

// GetLoadBalancer returns whether the specified load balancer exists, and if so, what its status is
func (c *Cloud) GetLoadBalancer(ctx context.Context, clusterName string, service *v1.Service) (status *v1.LoadBalancerStatus, exists bool, err error) {
	if isElasticLoadBalancer(service.Annotations) {
		return c.getElasticLoadBalancer(ctx, clusterName, service)
	}
	if isL4LoadBalancer(service.Annotations) {
		return c.getL4LoadBalancer(ctx, clusterName, service)
	}
	return nil, false, nil
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

	err = validateLoadBalancerAnnotations(service.Annotations)
	if err != nil {
		return nil, err
	}

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
	err := validateLoadBalancerAnnotations(service.Annotations)
	if err != nil {
		return err
	}

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

func validateLoadBalancerAnnotations(annotations map[string]string) error {
	// validation of both l4 load balancer and elastic load balancer
	loadBalancerType, ok := annotations[ServiceAnnotationLoadBalancerType]
	if ok {
		if loadBalancerType != "lb" && loadBalancerType != "elb" {
			return fmt.Errorf("annotation %s=%s is invalid", ServiceAnnotationLoadBalancerType, loadBalancerType)
		}
	} else {
		loadBalancerType = "lb"
	}

	if balancingType, ok := annotations[ServiceAnnotationLoadBalancerBalancingType]; ok {
		if balancingType != "1" && balancingType != "2" {
			return fmt.Errorf("annotation %s=%s is invalid", ServiceAnnotationLoadBalancerBalancingType, balancingType)
		}
	}

	if accountingType, ok := annotations[ServiceAnnotationLoadBalancerAccountingType]; ok {
		if accountingType != "1" && accountingType != "2" {
			return fmt.Errorf("annotation %s=%s is invalid", ServiceAnnotationLoadBalancerAccountingType, accountingType)
		}
	}

	if unhealthyThreshold, ok := annotations[ServiceAnnotationLoadBalancerHCUnhealthyThreshold]; ok {
		t, err := strconv.Atoi(unhealthyThreshold)
		if err != nil || t < 1 || 10 < t {
			return fmt.Errorf("annotation %s=%s is invalid", ServiceAnnotationLoadBalancerHCUnhealthyThreshold, unhealthyThreshold)
		}
	}

	if healthCheckInterval, ok := annotations[ServiceAnnotationLoadBalancerHCInterval]; ok {
		interval, err := strconv.Atoi(healthCheckInterval)
		if err != nil || interval < 5 || 300 < interval {
			return fmt.Errorf("annotation %s=%s is invalid", ServiceAnnotationLoadBalancerHCInterval, healthCheckInterval)
		}
	}

	if loadBalancerType == "lb" {
		// validation of l4 load balancer
		if networkVolume, ok := annotations[ServiceAnnotationLoadBalancerNetworkVolume]; ok {
			if !slices.Contains(allowedL4LoadBalancerNetworkVolume, networkVolume) {
				return fmt.Errorf("annotation %s=%s is invalid", ServiceAnnotationLoadBalancerNetworkVolume, networkVolume)
			}
		}

		if policyType, ok := annotations[ServiceAnnotationLoadBalancerPolicyType]; ok {
			if policyType != "standard" && policyType != "ats" {
				return fmt.Errorf("annotation %s=%s is invalid", ServiceAnnotationLoadBalancerPolicyType, policyType)
			}
		}

		if proto, ok := annotations[ServiceAnnotationLoadBalancerHCProtocol]; ok {
			if proto != "TCP" && proto != "ICMP" {
				return fmt.Errorf("annotation %s=%s is invalid", ServiceAnnotationLoadBalancerHCProtocol, proto)
			}
		}

		if networkInterface, ok := annotations[ServiceAnnotationLoadBalancerNetworkInterface1]; ok {
			if networkInterface != "" {
				return fmt.Errorf("annotation %s is only enabled for %s=elb", ServiceAnnotationLoadBalancerNetworkInterface1, ServiceAnnotationLoadBalancerType)
			}
		}

		if networkInterface, ok := annotations[ServiceAnnotationLoadBalancerNetworkInterface2]; ok {
			if networkInterface != "" {
				return fmt.Errorf("annotation %s is only enabled for %s=elb", ServiceAnnotationLoadBalancerNetworkInterface2, ServiceAnnotationLoadBalancerType)
			}
		}

		if ipAddress, ok := annotations[ServiceAnnotationLoadBalancerNetworkInterface1IPAddress]; ok {
			if ipAddress != "" {
				return fmt.Errorf("annotation %s is only enabled for %s=elb", ServiceAnnotationLoadBalancerNetworkInterface1IPAddress, ServiceAnnotationLoadBalancerType)
			}
		}

		if ipAddress, ok := annotations[ServiceAnnotationLoadBalancerNetworkInterface2IPAddress]; ok {
			if ipAddress != "" {
				return fmt.Errorf("annotation %s is only enabled for %s=elb", ServiceAnnotationLoadBalancerNetworkInterface2IPAddress, ServiceAnnotationLoadBalancerType)
			}
		}

		if systemIPAddresses, ok := annotations[ServiceAnnotationLoadBalancerNetworkInterface1SystemIPAddresses]; ok {
			if systemIPAddresses != "" {
				return fmt.Errorf("annotation %s is only enabled for %s=elb", ServiceAnnotationLoadBalancerNetworkInterface1SystemIPAddresses, ServiceAnnotationLoadBalancerType)
			}
		}

		if systemIPAddresses, ok := annotations[ServiceAnnotationLoadBalancerNetworkInterface2SystemIPAddresses]; ok {
			if systemIPAddresses != "" {
				return fmt.Errorf("annotation %s is only enabled for %s=elb", ServiceAnnotationLoadBalancerNetworkInterface2SystemIPAddresses, ServiceAnnotationLoadBalancerType)
			}
		}

		if vipNetwork, ok := annotations[ServiceAnnotationLoadBalancerVipNetwork]; ok {
			if vipNetwork != "" {
				return fmt.Errorf("annotation %s is only enabled for %s=elb", ServiceAnnotationLoadBalancerVipNetwork, ServiceAnnotationLoadBalancerType)
			}
		}
	}
	if loadBalancerType == "elb" {
		// validation of elastic load balancer
		if networkVolume, ok := annotations[ServiceAnnotationLoadBalancerNetworkVolume]; ok {
			if !slices.Contains(allowedElasticLoadBalancerNetworkVolume, networkVolume) {
				return fmt.Errorf("annotation %s=%s is invalid", ServiceAnnotationLoadBalancerNetworkVolume, networkVolume)
			}
		}

		if policyType, ok := annotations[ServiceAnnotationLoadBalancerPolicyType]; ok {
			if policyType != "" {
				return fmt.Errorf("annotation %s is only enabled for %s=lb", ServiceAnnotationLoadBalancerPolicyType, ServiceAnnotationLoadBalancerType)
			}
		}

		if proto, ok := annotations[ServiceAnnotationLoadBalancerHCProtocol]; ok {
			if proto != "TCP" && proto != "ICMP" {
				return fmt.Errorf("annotation %s=%s is invalid", ServiceAnnotationLoadBalancerHCProtocol, proto)
			}
		}

		if networkInterface1, ok := annotations[ServiceAnnotationLoadBalancerNetworkInterface1]; ok {
			if isPrivateLanNetworkID(networkInterface1) {
				if ipAddress, ok := annotations[ServiceAnnotationLoadBalancerNetworkInterface1IPAddress]; ok {
					if !isIPAddress(ipAddress) {
						return fmt.Errorf("annotation %s=%s is invalid", ServiceAnnotationLoadBalancerNetworkInterface1IPAddress, ipAddress)
					}
				} else {
					return fmt.Errorf("annotation %s is required when %s is private ip", ServiceAnnotationLoadBalancerNetworkInterface1IPAddress, ServiceAnnotationLoadBalancerNetworkInterface1)
				}

				if systemIPAddresses, ok := annotations[ServiceAnnotationLoadBalancerNetworkInterface1SystemIPAddresses]; ok {
					separatedSystemIPAdresses := strings.Split(systemIPAddresses, ",")
					if len(separatedSystemIPAdresses) != 2 {
						return fmt.Errorf("annotation %s is required two ip addresses", ServiceAnnotationLoadBalancerNetworkInterface1SystemIPAddresses)
					}
					for i := range separatedSystemIPAdresses {
						if !isIPAddress(separatedSystemIPAdresses[i]) {
							return fmt.Errorf("annotation %s=%s is invalid", ServiceAnnotationLoadBalancerNetworkInterface1SystemIPAddresses, systemIPAddresses)
						}
					}
				} else {
					return fmt.Errorf("annotation %s is required when %s is private ip", ServiceAnnotationLoadBalancerNetworkInterface1SystemIPAddresses, ServiceAnnotationLoadBalancerNetworkInterface1)
				}
			} else {
				if ipAddress, ok := annotations[ServiceAnnotationLoadBalancerNetworkInterface1IPAddress]; ok {
					if ipAddress != "" {
						return fmt.Errorf("can set %s only %s is private ip", ServiceAnnotationLoadBalancerNetworkInterface1IPAddress, ServiceAnnotationLoadBalancerNetworkInterface1)
					}
				}

				if systemIPAddresses, ok := annotations[ServiceAnnotationLoadBalancerNetworkInterface1SystemIPAddresses]; ok {
					if systemIPAddresses != "" {
						return fmt.Errorf("can set %s only %s is private ip", ServiceAnnotationLoadBalancerNetworkInterface1SystemIPAddresses, ServiceAnnotationLoadBalancerNetworkInterface1)
					}
				}
			}
		}

		if networkInterface2, ok := annotations[ServiceAnnotationLoadBalancerNetworkInterface2]; ok {
			if isPrivateLanNetworkID(networkInterface2) {
				if ipAddress, ok := annotations[ServiceAnnotationLoadBalancerNetworkInterface2IPAddress]; ok {
					if !isIPAddress(ipAddress) {
						return fmt.Errorf("annotation %s=%s is invalid", ServiceAnnotationLoadBalancerNetworkInterface2IPAddress, ipAddress)
					}
				} else {
					return fmt.Errorf("%s is required when %s is private ip", ServiceAnnotationLoadBalancerNetworkInterface2IPAddress, ServiceAnnotationLoadBalancerNetworkInterface2)
				}

				if systemIPAddresses, ok := annotations[ServiceAnnotationLoadBalancerNetworkInterface2SystemIPAddresses]; ok {
					separatedSystemIPAdresses := strings.Split(systemIPAddresses, ",")
					if len(separatedSystemIPAdresses) != 2 {
						return fmt.Errorf("%s is required two ip addresses", ServiceAnnotationLoadBalancerNetworkInterface2SystemIPAddresses)
					}
					for i := range separatedSystemIPAdresses {
						if !isIPAddress(separatedSystemIPAdresses[i]) {
							return fmt.Errorf("annotation %s=%s is invalid", ServiceAnnotationLoadBalancerNetworkInterface2SystemIPAddresses, systemIPAddresses)
						}
					}
				} else {
					return fmt.Errorf("annotation %s is required when %s is private ip", ServiceAnnotationLoadBalancerNetworkInterface2SystemIPAddresses, ServiceAnnotationLoadBalancerNetworkInterface1)
				}
			} else {
				if ipAddress, ok := annotations[ServiceAnnotationLoadBalancerNetworkInterface2IPAddress]; ok {
					if ipAddress != "" {
						return fmt.Errorf("can set %s only %s is private ip", ServiceAnnotationLoadBalancerNetworkInterface2IPAddress, ServiceAnnotationLoadBalancerNetworkInterface2)
					}
				}

				if systemIPAddresses, ok := annotations[ServiceAnnotationLoadBalancerNetworkInterface2SystemIPAddresses]; ok {
					if systemIPAddresses != "" {
						return fmt.Errorf("can set %s only %s is private ip", ServiceAnnotationLoadBalancerNetworkInterface2SystemIPAddresses, ServiceAnnotationLoadBalancerNetworkInterface2)
					}
				}
			}
		}

		if vipNetwork, ok := annotations[ServiceAnnotationLoadBalancerVipNetwork]; ok {
			if vipNetwork != "1" && vipNetwork != "2" {
				return fmt.Errorf("annotation %s=%s is invalid", ServiceAnnotationLoadBalancerVipNetwork, vipNetwork)
			}
		}
	}

	return nil
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
