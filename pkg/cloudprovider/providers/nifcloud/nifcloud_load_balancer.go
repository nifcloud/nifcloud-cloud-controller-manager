package nifcloud

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	v1 "k8s.io/api/core/v1"
	servicehelpers "k8s.io/cloud-provider/service/helpers"
	"k8s.io/klog/v2"
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
	// valid values are 'lb' or 'elb'
	ServiceAnnotationLoadBalancerType = "service.beta.kubernetes.io/nifcloud-load-balancer-type"

	// ServiceAnnotationLoadBalancerNetworkInterface(1-2) is the annotation that specify network interface of elastic load balancer
	// net-COMMON_GLOBAL, net-COMMON_PRIVATE or network ID of private LAN
	ServiceAnnotationLoadBalancerNetworkInterface1 = "service.beta.kubernetes.io/nifcloud-load-balancer-network-interface-1"
	ServiceAnnotationLoadBalancerNetworkInterface2 = "service.beta.kubernetes.io/nifcloud-load-balancer-network-interface-2"

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

	loadBalancerName := c.GetLoadBalancerName(ctx, clusterName, service)
	loadBalancers, err := c.client.DescribeLoadBalancers(ctx, loadBalancerName)
	if err != nil {
		return nil, false, err
	}

	if len(loadBalancers) == 0 {
		return nil, false, fmt.Errorf("not found load balancer: %q", loadBalancerName)
	}

	// service can have many ports, but the load balancer vip is the same
	return toLoadBalancerStatus(loadBalancers[0].VIP), true, nil
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
		return nil, fmt.Errorf("could not fetch instances info for %v: %v", instanceIDs, err)
	}

	loadBalancerName := c.GetLoadBalancerName(ctx, clusterName, service)

	if isElasticLoadBalancer(service.Annotations) {
		elb, err := NewElasticLoadBalancerFromService(loadBalancerName, instances, service)
		if err != nil {
			return nil, err
		}
		return c.ensureElasticLoadBalancer(ctx, loadBalancerName, elb)
	} else {
		desire := make([]LoadBalancer, portCount)
		for i, port := range service.Spec.Ports {
			// basic load balancer options
			desire[i].Name = loadBalancerName
			annotations := service.Annotations
			if rawBalancingType, ok := annotations[ServiceAnnotationLoadBalancerBalancingType]; ok {
				balancingType, err := strconv.Atoi(rawBalancingType)
				if err != nil {
					return nil, fmt.Errorf(
						"balancing type %q is invalid for service %q: %v",
						rawBalancingType, service.GetName(), err,
					)
				}
				desire[i].BalancingType = int32(balancingType)
			}

			if accountingType, ok := annotations[ServiceAnnotationLoadBalancerAccountingType]; ok {
				desire[i].AccountingType = accountingType
			}

			if networkVolume, ok := annotations[ServiceAnnotationLoadBalancerNetworkVolume]; ok {
				v, err := strconv.Atoi(networkVolume)
				if err != nil {
					return nil, fmt.Errorf(
						"network volume %q is invalid for service %q: %v",
						networkVolume, service.GetName(), err,
					)
				}
				desire[i].NetworkVolume = int32(v)
			}
			if policyType, ok := annotations[ServiceAnnotationLoadBalancerPolicyType]; ok {
				desire[i].PolicyType = policyType
			}

			if port.Protocol != v1.ProtocolTCP {
				return nil, fmt.Errorf("only TCP load balancer is supported")
			}
			if port.NodePort == 0 {
				klog.Errorf("Ignoring port without NodePort defined: %v", port)
				continue
			}

			desire[i].LoadBalancerPort = int32(port.Port)
			desire[i].InstancePort = int32(port.NodePort)

			// health check
			if strInterval, ok := annotations[ServiceAnnotationLoadBalancerHCInterval]; ok {
				interval, err := strconv.Atoi(strInterval)
				if err != nil {
					return nil, fmt.Errorf(
						"health check interval %q is invalid for service %q: %v",
						strInterval, service.GetName(), err,
					)
				}
				desire[i].HealthCheckInterval = int32(interval)
			} else {
				desire[i].HealthCheckInterval = defaultHealthCheckInterval
			}

			if unhealthyThreshold, ok := annotations[ServiceAnnotationLoadBalancerHCUnhealthyThreshold]; ok {
				t, err := strconv.Atoi(unhealthyThreshold)
				if err != nil {
					return nil, fmt.Errorf(
						"unhealthy threshold %q is invalid for service %q: %v",
						unhealthyThreshold, service.GetName(), err,
					)
				}
				desire[i].HealthCheckUnhealthyThreshold = int32(t)
			} else {
				desire[i].HealthCheckUnhealthyThreshold = defaultHealthCheckUnhealthyThreshold
			}

			if proto, ok := annotations[ServiceAnnotationLoadBalancerHCProtocol]; ok {
				switch strings.ToUpper(proto) {
				case "TCP":
					desire[i].HealthCheckTarget = fmt.Sprintf("TCP:%d", port.NodePort)
				case "ICMP":
					desire[i].HealthCheckTarget = "ICMP"
				default:
					return nil, fmt.Errorf(
						"health check protocol %q is invalid for service %q",
						proto, service.GetName(),
					)
				}
			} else {
				desire[i].HealthCheckTarget = fmt.Sprintf("%s:%d", defaultHealthCheckTarget, port.NodePort)
			}

			// balancing targets
			desire[i].BalancingTargets = instances

			// filter
			sourceRanges, err := servicehelpers.GetLoadBalancerSourceRanges(service)
			if err != nil {
				return nil, err
			}
			filters := []string{}
			if !servicehelpers.IsAllowAll(sourceRanges) {
				for cidr := range sourceRanges {
					if strings.HasSuffix(cidr, "/32") {
						filters = append(filters, strings.TrimSuffix(cidr, "/32"))
					} else {
						filters = append(filters, strings.Replace(cidr, "/32", "", 1))
					}
				}
			}
			desire[i].Filters = sort.StringSlice(filters)
		}
		return c.ensureLoadBalancer(ctx, desire)
	}
}

// UpdateLoadBalancer updates hosts under the specified load balancer
func (c *Cloud) UpdateLoadBalancer(ctx context.Context, clusterName string, service *v1.Service, nodes []*v1.Node) error {
	if isElasticLoadBalancer(service.Annotations) {
		return c.updateElasticLoadBalancer(ctx, clusterName, service, nodes)
	}

	loadBalancerName := c.GetLoadBalancerName(ctx, clusterName, service)

	loadBalancers, err := c.client.DescribeLoadBalancers(ctx, loadBalancerName)
	if err != nil {
		return err
	}
	if len(loadBalancers) == 0 {
		return fmt.Errorf("load balancer %q not found", loadBalancerName)
	}

	_, err = c.EnsureLoadBalancer(ctx, clusterName, service, nodes)

	return err
}

// EnsureLoadBalancerDeleted deletes the specified load balancer if it exists
func (c *Cloud) EnsureLoadBalancerDeleted(ctx context.Context, clusterName string, service *v1.Service) error {
	if isElasticLoadBalancer(service.Annotations) {
		return c.ensureElasticLoadBalancerDeleted(ctx, clusterName, service)
	}

	loadBalancerName := c.GetLoadBalancerName(ctx, clusterName, service)

	loadBalancers, err := c.client.DescribeLoadBalancers(ctx, loadBalancerName)
	if err != nil {
		return err
	}
	if len(loadBalancers) == 0 {
		return fmt.Errorf("load balancer %q already deleted", loadBalancerName)
	}

	for _, lb := range loadBalancers {
		klog.Infof("Deleting LoadBalancer %q (%d -> %d)", lb.Name, lb.LoadBalancerPort, lb.InstancePort)
		if err := c.client.DeleteLoadBalancer(ctx, &lb); err != nil {
			return fmt.Errorf("failed to delete load balancer: %w", err)
		}
	}

	return nil
}

func (c *Cloud) ensureLoadBalancer(ctx context.Context, desire []LoadBalancer) (*v1.LoadBalancerStatus, error) {
	if len(desire) == 0 {
		return nil, fmt.Errorf("desire LoadBalancer length must be larger than 1")
	}

	loadBalancerName := desire[0].Name
	current, err := c.client.DescribeLoadBalancers(ctx, loadBalancerName)
	if err != nil {
		if strings.Contains(err.Error(), "NotFound") {
			// create all load balancers
			var vip string
			for i, lb := range desire {
				klog.Infof("Creating LoadBalancer %q (%d -> %d)", lb.Name, lb.LoadBalancerPort, lb.InstancePort)
				if i == 0 {
					vip, err = c.client.CreateLoadBalancer(ctx, &lb)
					if err != nil {
						return nil, fmt.Errorf("failed to create load balancer: %w", err)
					}
				} else {
					if err := c.client.RegisterPortWithLoadBalancer(ctx, &lb); err != nil {
						return nil, fmt.Errorf("failed to add port to load balancer: %w", err)
					}
				}
			}

			return toLoadBalancerStatus(vip), nil
		}

		return nil, fmt.Errorf("failed to describe load balanacer %q: %w", loadBalancerName, err)
	}

	klog.Infof("desire: %v, current: %v", desire, current)

	loadBalancerResourceChanged := false
	if len(current) < len(desire) {
		toCreate := loadBalancerDifferences(desire, current)
		for _, lb := range toCreate {
			klog.Infof("Creating LoadBalancer %q (%d -> %d)", lb.Name, lb.LoadBalancerPort, lb.InstancePort)
			if err := c.client.RegisterPortWithLoadBalancer(ctx, &lb); err != nil {
				return nil, fmt.Errorf("failed to add port to load balancer: %w", err)
			}
			loadBalancerResourceChanged = true
		}
	} else if len(current) > len(desire) {
		toDelete := loadBalancerDifferences(current, desire)
		for _, lb := range toDelete {
			klog.Infof("Deleting LoadBalancer %q (%d -> %d)", lb.Name, lb.LoadBalancerPort, lb.InstancePort)
			if err := c.client.DeleteLoadBalancer(ctx, &lb); err != nil {
				return nil, fmt.Errorf("failed to delete load balancer: %w", err)
			}
			loadBalancerResourceChanged = true
		}
	}

	// fetch load balancers again to update latest load balancer info
	if loadBalancerResourceChanged {
		current, err = c.client.DescribeLoadBalancers(ctx, loadBalancerName)
		if err != nil {
			return nil, fmt.Errorf("failed to describe load balanacer %q: %w", loadBalancerName, err)
		}
	}

	klog.Infof("desire: %v, current: %v", desire, current)

	for _, currentLB := range current {
		desireLB, err := findLoadBalancer(desire, currentLB)
		if err != nil {
			return nil, err
		}

		// reconcile balancing targets
		toRegister := loadBalancingTargetsDifferences(desireLB.BalancingTargets, currentLB.BalancingTargets)
		if len(toRegister) > 0 {
			klog.Infof(
				"Register instances with load balancer %q (%d -> %d): %v",
				currentLB.Name, currentLB.LoadBalancerPort, currentLB.InstancePort, toRegister,
			)
			if err := c.client.RegisterInstancesWithLoadBalancer(ctx, &currentLB, toRegister); err != nil {
				return nil, fmt.Errorf("failed to register instances: %w", err)
			}
		}

		toDeregister := loadBalancingTargetsDifferences(currentLB.BalancingTargets, desireLB.BalancingTargets)
		if len(toDeregister) > 0 {
			klog.Infof(
				"Deregister instances from load balancer %q (%d -> %d): %v",
				currentLB.Name, currentLB.LoadBalancerPort, currentLB.InstancePort, toDeregister,
			)
			if err := c.client.DeregisterInstancesFromLoadBalancer(ctx, &currentLB, toDeregister); err != nil {
				return nil, fmt.Errorf("failed to deregister instances: %w", err)
			}
		}

		// reconcile filters
		toAuthorize := filterDifferences(desireLB.Filters, currentLB.Filters)
		toRevoke := filterDifferences(currentLB.Filters, desireLB.Filters)
		toSet := []Filter{}
		for _, addr := range toAuthorize {
			if addr == filterAnyIPAddresses {
				continue
			}
			toSet = append(toSet, Filter{AddOnFilter: true, IPAddress: addr})
		}
		for _, addr := range toRevoke {
			if addr == filterAnyIPAddresses {
				continue
			}
			toSet = append(toSet, Filter{AddOnFilter: false, IPAddress: addr})
		}
		if len(toSet) > 0 {
			klog.Infof("Applying filter: %v", toSet)
			if err := c.client.SetFilterForLoadBalancer(ctx, &currentLB, toSet); err != nil {
				return nil, fmt.Errorf("failed to set filter for load balancer: %w", err)
			}
		}
	}

	return toLoadBalancerStatus(current[0].VIP), nil
}

func findLoadBalancer(from []LoadBalancer, target LoadBalancer) (*LoadBalancer, error) {
	for _, lb := range from {
		if target.Equals(lb) {
			return &lb, nil
		}
	}

	return nil, fmt.Errorf(
		"target load balancer (%q: %d -> %d) not found",
		target.Name, target.LoadBalancerPort, target.InstancePort,
	)
}

func loadBalancerDifferences(target, other []LoadBalancer) []LoadBalancer {
	diff := []LoadBalancer{}
	for _, x := range target {
		found := false
		for _, y := range other {
			if x.Equals(y) {
				found = true
				break
			}
		}

		if !found {
			diff = append(diff, x)
		}
	}

	return diff
}

func loadBalancingTargetsDifferences(target, other []Instance) []Instance {
	diff := []Instance{}
	for _, x := range target {
		found := false
		for _, y := range other {
			if x.Equals(y) {
				found = true
				break
			}
		}

		if !found {
			diff = append(diff, x)
		}
	}

	return diff
}

func filterDifferences(target, other []string) []string {
	diff := []string{}
	for _, x := range target {
		found := false
		for _, y := range other {
			if x == y {
				found = true
				break
			}
		}

		if !found {
			diff = append(diff, x)
		}
	}

	return diff
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
