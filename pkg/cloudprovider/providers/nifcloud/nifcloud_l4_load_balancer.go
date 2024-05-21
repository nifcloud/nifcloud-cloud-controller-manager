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

func isL4LoadBalancer(annotations map[string]string) bool {
	if t := annotations[ServiceAnnotationLoadBalancerType]; t == "lb" || t == "" {
		return true
	}
	return false
}

func (c *Cloud) getL4LoadBalancer(ctx context.Context, clusterName string, service *v1.Service) (status *v1.LoadBalancerStatus, exists bool, err error) {
	loadBalancerName := c.GetLoadBalancerName(ctx, clusterName, service)
	loadBalancers, err := c.client.DescribeLoadBalancers(ctx, loadBalancerName)
	if err != nil {
		switch {
		case isAPIError(err, errorCodeLoadBalancerNotFound):
			return nil, false, nil
		}
		return nil, false, err
	}

	if len(loadBalancers) == 0 {
		return nil, false, fmt.Errorf("not found load balancer: %q", loadBalancerName)
	}

	// service can have many ports, but the load balancer vip is the same
	return toLoadBalancerStatus(loadBalancers[0].VIP), true, nil
}

func (c *Cloud) ensureL4LoadBalancer(ctx context.Context, loadBalancerName string, desire []LoadBalancer) (*v1.LoadBalancerStatus, error) {
	if len(desire) == 0 {
		return nil, fmt.Errorf("desire LoadBalancer length must be larger than 1")
	}

	current, err := c.client.DescribeLoadBalancers(ctx, loadBalancerName)
	if err != nil {
		if isAPIError(err, errorCodeLoadBalancerNotFound) {
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

		return nil, fmt.Errorf("failed to describe load balancer %q: %w", loadBalancerName, err)
	}

	klog.Infof("desire: %v, current: %v", desire, current)

	loadBalancerResourceChanged := false
	if len(current) < len(desire) {
		toCreate := l4LoadBalancerDifferences(desire, current)
		for _, lb := range toCreate {
			klog.Infof("Creating LoadBalancer %q (%d -> %d)", lb.Name, lb.LoadBalancerPort, lb.InstancePort)
			if err := c.client.RegisterPortWithLoadBalancer(ctx, &lb); err != nil {
				return nil, fmt.Errorf("failed to add port to load balancer: %w", err)
			}
			loadBalancerResourceChanged = true
		}
	} else if len(current) > len(desire) {
		toDelete := l4LoadBalancerDifferences(current, desire)
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
			return nil, fmt.Errorf("failed to describe load balancer %q: %w", loadBalancerName, err)
		}
	}

	klog.Infof("desire: %v, current: %v", desire, current)

	for _, currentLB := range current {
		desireLB, err := findL4LoadBalancer(desire, currentLB)
		if err != nil {
			return nil, err
		}

		// reconcile balancing targets
		toRegister := l4LoadBalancingTargetsDifferences(desireLB.BalancingTargets, currentLB.BalancingTargets)
		if len(toRegister) > 0 {
			klog.Infof(
				"Register instances with load balancer %q (%d -> %d): %v",
				currentLB.Name, currentLB.LoadBalancerPort, currentLB.InstancePort, toRegister,
			)
			if err := c.client.RegisterInstancesWithLoadBalancer(ctx, &currentLB, toRegister); err != nil {
				return nil, fmt.Errorf("failed to register instances: %w", err)
			}
		}

		toDeregister := l4LoadBalancingTargetsDifferences(currentLB.BalancingTargets, desireLB.BalancingTargets)
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

func NewL4LoadBalancerFromService(loadBalancerName string, instances []Instance, service *v1.Service) ([]LoadBalancer, error) {
	portCount := len(service.Spec.Ports)

	desire := make([]LoadBalancer, portCount)
	for i, port := range service.Spec.Ports {
		// basic load balancer options
		desire[i].Name = loadBalancerName
		annotations := service.Annotations
		if rawBalancingType, ok := annotations[ServiceAnnotationLoadBalancerBalancingType]; ok {
			balancingType, err := strconv.Atoi(rawBalancingType)
			if err != nil {
				return nil, fmt.Errorf(
					"balancing type %q is invalid for service %q: %w",
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
					"network volume %q is invalid for service %q: %w",
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
					"health check interval %q is invalid for service %q: %w",
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
					"unhealthy threshold %q is invalid for service %q: %w",
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

	return desire, nil
}

func (c *Cloud) updateL4LoadBalancer(ctx context.Context, clusterName string, service *v1.Service) error {
	loadBalancerName := c.GetLoadBalancerName(ctx, clusterName, service)

	loadBalancers, err := c.client.DescribeLoadBalancers(ctx, loadBalancerName)
	if err != nil {
		return err
	}
	if len(loadBalancers) == 0 {
		return fmt.Errorf("load balancer %q not found", loadBalancerName)
	}
	return nil
}

func (c *Cloud) ensureL4LoadBalancerDeleted(ctx context.Context, clusterName string, service *v1.Service) error {
	loadBalancerName := c.GetLoadBalancerName(ctx, clusterName, service)

	loadBalancers, err := c.client.DescribeLoadBalancers(ctx, loadBalancerName)
	if err != nil {
		switch {
		case isAPIError(err, errorCodeLoadBalancerNotFound):
			klog.Infof("load balancer %q is not found", loadBalancerName)
			return nil
		}
		return err
	}
	if len(loadBalancers) == 0 {
		klog.Infof("load balancer %q already deleted", loadBalancerName)
		return nil
	}

	for _, lb := range loadBalancers {
		klog.Infof("Deleting LoadBalancer %q (%d -> %d)", lb.Name, lb.LoadBalancerPort, lb.InstancePort)
		if err := c.client.DeleteLoadBalancer(ctx, &lb); err != nil {
			return fmt.Errorf("failed to delete load balancer: %w", err)
		}
	}

	return nil
}

func findL4LoadBalancer(from []LoadBalancer, target LoadBalancer) (*LoadBalancer, error) {
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

func l4LoadBalancerDifferences(target, other []LoadBalancer) []LoadBalancer {
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

func l4LoadBalancingTargetsDifferences(target, other []Instance) []Instance {
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
