package nifcloud

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
)

const (
	commonGlobalNetworkID  = "net-COMMON_GLOBAL"
	commonPrivateNetworkID = "net-COMMON_PRIVATE"
)

func isElasticLoadBalancer(annotations map[string]string) bool {
	return annotations[ServiceAnnotationLoadBalancerType] == "elb"
}

func isPrivateLanNetworkID(networkID string) bool {
	return networkID != commonGlobalNetworkID && networkID != commonPrivateNetworkID
}

func (c *Cloud) getElasticLoadBalancer(ctx context.Context, clusterName string, service *v1.Service) (status *v1.LoadBalancerStatus, exists bool, err error) {
	// get load balancer name
	loadBalancerName := c.GetLoadBalancerName(ctx, clusterName, service)

	// describe load balancer
	loadBalancers, err := c.client.DescribeElasticLoadBalancers(ctx, loadBalancerName)
	if err != nil {
		switch {
		case isAPIError(err, errorCodeElasticLoadBalancerNotFound):
			return nil, false, nil
		}
		return nil, false, err
	}

	if len(loadBalancers) == 0 {
		return nil, false, nil
	}

	// return load balancer status
	return toLoadBalancerStatus(loadBalancers[0].VIP), true, nil
}

func (c *Cloud) ensureElasticLoadBalancer(ctx context.Context, loadBalancerName string, desire []ElasticLoadBalancer) (*v1.LoadBalancerStatus, error) {
	// correct state differences
	if len(desire) == 0 {
		return nil, fmt.Errorf("desire ElasticLoadBalancer length must be larger than 1")
	}

	// get current load balancer status
	current, err := c.client.DescribeElasticLoadBalancers(ctx, loadBalancerName)

	// if not exist, create load balancer
	if err != nil {
		if isAPIError(err, errorCodeElasticLoadBalancerNotFound) {
			// create all load balancers
			var vip string
			var networkInterfaces []NetworkInterface
			for i, lb := range desire {
				klog.Infof("Creating ElasticLoadBalancer %q (%d -> %d)", lb.Name, lb.LoadBalancerPort, lb.InstancePort)
				if i == 0 {
					_, err = c.client.CreateElasticLoadBalancer(ctx, &lb)
					if err != nil {
						return nil, fmt.Errorf("failed to create elastic load balancer: %w", err)
					}
					current, err = c.client.DescribeElasticLoadBalancers(ctx, loadBalancerName)
					if err != nil {
						return nil, fmt.Errorf("failed to describe elastic load balancer %q: %w", loadBalancerName, err)
					}
					vip = current[0].VIP
					networkInterfaces = current[0].NetworkInterfaces
				} else {
					if err := c.client.RegisterPortWithElasticLoadBalancer(ctx, &lb); err != nil {
						return nil, fmt.Errorf("failed to add port to elastic load balancer: %w", err)
					}
				}
				lb.VIP = vip
				lb.NetworkInterfaces = networkInterfaces
				if err := c.allowSecurityGroupRulesFromElasticLoadBalancer(ctx, &lb, lb.BalancingTargets); err != nil {
					return nil, fmt.Errorf("failed to allow security group rules from elastic load balancer: %w", err)
				}
			}
			return toLoadBalancerStatus(vip), nil
		}
		return nil, fmt.Errorf("failed to describe elastic load balancer %q: %w", loadBalancerName, err)
	}

	// if exist, configure load balancers

	for i := range desire {
		desire[i].VIP = current[0].VIP
		desire[i].NetworkInterfaces = current[0].NetworkInterfaces
	}

	loadBalancerResourceChanged := false

	// if need to register port
	toCreate := elasticLoadBalancerDifferences(desire, current)
	for _, lb := range toCreate {
		klog.Infof("Registering ElasticLoadBalancer port %q (%d -> %d)", lb.Name, lb.LoadBalancerPort, lb.InstancePort)
		if err := c.client.RegisterPortWithElasticLoadBalancer(ctx, &lb); err != nil {
			return nil, fmt.Errorf("failed to add port to elastic load balancer: %w", err)
		}
		loadBalancerResourceChanged = true
		if err := c.allowSecurityGroupRulesFromElasticLoadBalancer(ctx, &lb, lb.BalancingTargets); err != nil {
			return nil, fmt.Errorf("failed to allow security group rules from elastic load balancer: %w", err)
		}
	}

	// if need to delete port
	toDelete := elasticLoadBalancerDifferences(current, desire)
	for _, lb := range toDelete {
		klog.Infof("Deleting LoadBalancer %q (%d -> %d)", lb.Name, lb.LoadBalancerPort, lb.InstancePort)
		if err := c.client.DeleteElasticLoadBalancer(ctx, &lb); err != nil {
			return nil, fmt.Errorf("failed to delete elastic load balancer: %w", err)
		}
		loadBalancerResourceChanged = true
		if err := c.denySecurityGroupRulesFromElasticLoadBalancer(ctx, &lb, lb.BalancingTargets); err != nil {
			return nil, fmt.Errorf("failed to deny security group rules from elastic load balancer: %w", err)
		}
	}

	if loadBalancerResourceChanged {
		current, err = c.client.DescribeElasticLoadBalancers(ctx, loadBalancerName)
		if err != nil {
			return nil, fmt.Errorf("failed to describe elastic load balancer %q: %w", loadBalancerName, err)
		}
	}

	for _, currentLB := range current {
		desireLB, err := findElasticLoadBalancer(desire, currentLB)
		if err != nil {
			return nil, err
		}

		// reconcile balancing targets
		toRegister := elasticLoadBalancingTargetsDifferences(desireLB.BalancingTargets, currentLB.BalancingTargets)
		if len(toRegister) > 0 {
			klog.Infof(
				"Register instances with elastic load balancer %q (%d -> %d): %v",
				currentLB.Name, currentLB.LoadBalancerPort, currentLB.InstancePort, toRegister,
			)
			if err := c.client.RegisterInstancesWithElasticLoadBalancer(ctx, &currentLB, toRegister); err != nil {
				return nil, fmt.Errorf("failed to register instances: %w", err)
			}
			if err := c.allowSecurityGroupRulesFromElasticLoadBalancer(ctx, &currentLB, toRegister); err != nil {
				return nil, fmt.Errorf("failed to allow security group rules from elastic load balancer: %w", err)
			}
		}

		toDeregister := elasticLoadBalancingTargetsDifferences(currentLB.BalancingTargets, desireLB.BalancingTargets)
		if len(toDeregister) > 0 {
			klog.Infof(
				"Deregister instances from load balancer %q (%d -> %d): %v",
				currentLB.Name, currentLB.LoadBalancerPort, currentLB.InstancePort, toDeregister,
			)
			if err := c.client.DeregisterInstancesFromElasticLoadBalancer(ctx, &currentLB, toDeregister); err != nil {
				return nil, fmt.Errorf("failed to deregister instances: %w", err)
			}
			if err := c.denySecurityGroupRulesFromElasticLoadBalancer(ctx, &currentLB, toDeregister); err != nil {
				return nil, fmt.Errorf("failed to deny security group rules from elastic load balancer: %w", err)
			}
		}
	}

	return toLoadBalancerStatus(current[0].VIP), nil
}

func NewElasticLoadBalancerFromService(loadBalancerName string, instances []Instance, service *v1.Service) ([]ElasticLoadBalancer, error) {
	portCount := len(service.Spec.Ports)

	// detect state differences
	desire := make([]ElasticLoadBalancer, portCount)
	annotations := service.Annotations

	// load balancer name
	for i := range desire {
		desire[i].Name = loadBalancerName
	}

	// Availability zones
	for i := range desire {
		desire[i].AvailabilityZone = instances[0].Zone
	}

	// balancing type
	if rawBalancingType, ok := annotations[ServiceAnnotationLoadBalancerBalancingType]; ok {
		balancingType, err := strconv.Atoi(rawBalancingType)
		if err != nil {
			return nil, fmt.Errorf(
				"balancing type %q is invalid for service %q: %w",
				rawBalancingType, service.GetName(), err,
			)
		}
		for i := range desire {
			desire[i].BalancingType = int32(balancingType)
		}
	}

	// accounting type
	if accountingType, ok := annotations[ServiceAnnotationLoadBalancerAccountingType]; ok {
		for i := range desire {
			desire[i].AccountingType = accountingType
		}
	}

	// network volume
	if networkVolume, ok := annotations[ServiceAnnotationLoadBalancerNetworkVolume]; ok {
		v, err := strconv.Atoi(networkVolume)
		if err != nil {
			return nil, fmt.Errorf(
				"network volume %q is invalid for service %q: %v",
				networkVolume, service.GetName(), err,
			)
		}
		for i := range desire {
			desire[i].NetworkVolume = int32(v)
		}
	}

	// protocol
	for i, port := range service.Spec.Ports {
		if port.Protocol != v1.ProtocolTCP {
			return nil, fmt.Errorf("only TCP load balancer is supported")
		}
		desire[i].Protocol = "TCP"
		if port.NodePort == 0 {
			klog.Errorf("Ignoring port without NodePort defined: %v", port)
			continue
		}
	}

	// load balancer port
	for i, port := range service.Spec.Ports {
		desire[i].LoadBalancerPort = int32(port.Port)
	}

	// instance port
	for i, port := range service.Spec.Ports {
		desire[i].InstancePort = int32(port.NodePort)
	}

	// health check interval
	if strInterval, ok := annotations[ServiceAnnotationLoadBalancerHCInterval]; ok {
		interval, err := strconv.Atoi(strInterval)
		if err != nil {
			return nil, fmt.Errorf(
				"health check interval %q is invalid for service %q: %w",
				strInterval, service.GetName(), err,
			)
		}
		for i := range desire {
			desire[i].HealthCheckInterval = int32(interval)
		}
	} else {
		for i := range desire {
			desire[i].HealthCheckInterval = defaultHealthCheckInterval
		}
	}

	// unhealthy threshold
	if unhealthyThreshold, ok := annotations[ServiceAnnotationLoadBalancerHCUnhealthyThreshold]; ok {
		t, err := strconv.Atoi(unhealthyThreshold)
		if err != nil {
			return nil, fmt.Errorf(
				"unhealthy threshold %q is invalid for service %q: %w",
				unhealthyThreshold, service.GetName(), err,
			)
		}
		for i := range desire {
			desire[i].HealthCheckUnhealthyThreshold = int32(t)
		}
	} else {
		for i := range desire {
			desire[i].HealthCheckUnhealthyThreshold = defaultHealthCheckUnhealthyThreshold
		}
	}

	// health check target
	if proto, ok := annotations[ServiceAnnotationLoadBalancerHCProtocol]; ok {
		switch strings.ToUpper(proto) {
		case "TCP":
			for i, port := range service.Spec.Ports {
				desire[i].HealthCheckTarget = fmt.Sprintf("TCP:%d", port.NodePort)
			}
		case "ICMP":
			for i := range desire {
				desire[i].HealthCheckTarget = "ICMP"
			}
		default:
			return nil, fmt.Errorf(
				"health check protocol %q is invalid for service %q",
				proto, service.GetName(),
			)
		}
	} else {
		for i, port := range service.Spec.Ports {
			desire[i].HealthCheckTarget = fmt.Sprintf("%s:%d", defaultHealthCheckTarget, port.NodePort)
		}
	}

	// balancing targets
	for i := range desire {
		desire[i].BalancingTargets = instances
	}

	// network interfaces
	networkInterfaces := []NetworkInterface{}
	if networkInterface1, ok := annotations[ServiceAnnotationLoadBalancerNetworkInterface1]; ok {
		networkInterface := NetworkInterface{
			NetworkId: networkInterface1,
		}

		if ipAddress, ok := annotations[ServiceAnnotationLoadBalancerNetworkInterface1IPAddress]; ok {
			networkInterface.IPAddress = ipAddress
		}

		if isPrivateLanNetworkID(networkInterface.NetworkId) {
			separatedSystemIPAdresses := []string{}
			if systemIPAddresses, ok := annotations[ServiceAnnotationLoadBalancerNetworkInterface1SystemIPAddresses]; ok {
				separatedSystemIPAdresses = strings.Split(systemIPAddresses, ",")
			}
			if len(separatedSystemIPAdresses) == 2 {
				networkInterface.SystemIpAddresses = separatedSystemIPAdresses
			} else {
				return nil, fmt.Errorf("system ip address require two value")
			}
		}

		if vipNetwork, ok := annotations[ServiceAnnotationLoadBalancerVipNetwork]; ok {
			if vipNetwork == "1" {
				networkInterface.IsVipNetwork = true
			}
		}

		networkInterfaces = append(networkInterfaces, networkInterface)
	}

	if networkInterface2, ok := annotations[ServiceAnnotationLoadBalancerNetworkInterface2]; ok {
		networkInterface := NetworkInterface{
			NetworkId: networkInterface2,
		}

		if ipAddress, ok := annotations[ServiceAnnotationLoadBalancerNetworkInterface2IPAddress]; ok {
			networkInterface.IPAddress = ipAddress
		}

		if isPrivateLanNetworkID(networkInterface.NetworkId) {
			separatedSystemIPAdresses := []string{}
			if systemIPAddresses, ok := annotations[ServiceAnnotationLoadBalancerNetworkInterface2SystemIPAddresses]; ok {
				separatedSystemIPAdresses = strings.Split(systemIPAddresses, ",")
			}
			if len(separatedSystemIPAdresses) == 2 {
				networkInterface.SystemIpAddresses = separatedSystemIPAdresses
			} else {
				return nil, fmt.Errorf("system ip address require two value")
			}
		}

		if vipNetwork, ok := annotations[ServiceAnnotationLoadBalancerVipNetwork]; ok {
			if vipNetwork == "2" {
				networkInterface.IsVipNetwork = true
			}
		}

		networkInterfaces = append(networkInterfaces, networkInterface)
	}

	if len(networkInterfaces) == 1 || len(networkInterfaces) == 2 {
		for i := range desire {
			desire[i].NetworkInterfaces = networkInterfaces
		}
	} else {
		networkInterface := NetworkInterface{
			NetworkId:    elasticLoadBalancerDefaultNetworkInterface,
			IsVipNetwork: true,
		}
		for i := range desire {
			desire[i].NetworkInterfaces = append(desire[i].NetworkInterfaces, networkInterface)
		}
	}

	return desire, nil
}

func (c *Cloud) allowSecurityGroupRulesFromElasticLoadBalancer(ctx context.Context, elasticLoadBalancer *ElasticLoadBalancer, instances []Instance) error {
	instanceIDs := []string{}
	for _, instance := range instances {
		instanceIDs = append(instanceIDs, instance.InstanceID)
	}

	securityGroups, err := c.client.DescribeSecurityGroupsByInstanceIDs(ctx, instanceIDs)
	if err != nil {
		return err
	}

	securityGroupRules, err := securityGroupRulesOfElasticLoadBalancer(ctx, elasticLoadBalancer)
	if err != nil {
		return err
	}

	for _, securityGroup := range securityGroups {
		for _, securityGroupRule := range securityGroupRules {
			err = c.client.AuthorizeSecurityGroupIngress(ctx, securityGroup.GroupName, &securityGroupRule)
			if err != nil {
				if isAPIError(err, errorCodeSecurityGroupDuplicate) {
					// ignore error
				} else {
					return err
				}
			}
			err = c.client.WaitSecurityGroupApplied(ctx, securityGroup.GroupName)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *Cloud) denySecurityGroupRulesFromElasticLoadBalancer(ctx context.Context, elasticLoadBalancer *ElasticLoadBalancer, instances []Instance) error {
	instanceIDs := []string{}
	for _, instance := range instances {
		instanceIDs = append(instanceIDs, instance.InstanceID)
	}

	securityGroups, err := c.client.DescribeSecurityGroupsByInstanceIDs(ctx, instanceIDs)
	if err != nil {
		return err
	}

	securityGroupRules, err := securityGroupRulesOfElasticLoadBalancer(ctx, elasticLoadBalancer)
	if err != nil {
		return err
	}

	for _, securityGroup := range securityGroups {
		for _, securityGroupRule := range securityGroupRules {
			err = c.client.RevokeSecurityGroupIngress(ctx, securityGroup.GroupName, &securityGroupRule)
			if err != nil {
				if isAPIError(err, errorCodeSecurityGroupIngressNotFound) {
					// ignore error
				} else {
					return err
				}
			}
			err = c.client.WaitSecurityGroupApplied(ctx, securityGroup.GroupName)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func securityGroupRulesOfElasticLoadBalancer(ctx context.Context, elasticLoadBalancer *ElasticLoadBalancer) ([]SecurityGroupRule, error) {
	securityGroupRules := []SecurityGroupRule{}
	VIPRanges := []string{elasticLoadBalancer.VIP}

	healthCheckProtocol, _ := separateHealthCheckTarget(elasticLoadBalancer.HealthCheckTarget)

	if len(elasticLoadBalancer.NetworkInterfaces) == 1 {
		// one arm
		securityGroupRule := SecurityGroupRule{
			IpProtocol: elasticLoadBalancer.Protocol,
			FromPort:   elasticLoadBalancer.InstancePort,
			ToPort:     elasticLoadBalancer.InstancePort,
			InOut:      "IN",
			IpRanges:   VIPRanges,
		}
		securityGroupRules = append(securityGroupRules, securityGroupRule)

		if healthCheckProtocol == "ICMP" {
			IPAddressRule := SecurityGroupRule{
				IpProtocol: healthCheckProtocol,
				InOut:      "IN",
				IpRanges:   VIPRanges,
			}
			securityGroupRules = append(securityGroupRules, IPAddressRule)
			for _, systemIPAddress := range elasticLoadBalancer.NetworkInterfaces[0].SystemIpAddresses {
				systemIPAddressRule := SecurityGroupRule{
					IpProtocol: healthCheckProtocol,
					InOut:      "IN",
					IpRanges:   []string{systemIPAddress},
				}
				securityGroupRules = append(securityGroupRules, systemIPAddressRule)
			}
		} else {
			if elasticLoadBalancer.Protocol != healthCheckProtocol {
				IPAddressRule := SecurityGroupRule{
					IpProtocol: healthCheckProtocol,
					FromPort:   elasticLoadBalancer.InstancePort,
					ToPort:     elasticLoadBalancer.InstancePort,
					InOut:      "IN",
					IpRanges:   VIPRanges,
				}
				securityGroupRules = append(securityGroupRules, IPAddressRule)
			}
			for _, systemIPAddress := range elasticLoadBalancer.NetworkInterfaces[0].SystemIpAddresses {
				systemIPAddressRule := SecurityGroupRule{
					IpProtocol: healthCheckProtocol,
					FromPort:   elasticLoadBalancer.InstancePort,
					ToPort:     elasticLoadBalancer.InstancePort,
					InOut:      "IN",
					IpRanges:   []string{systemIPAddress},
				}
				securityGroupRules = append(securityGroupRules, systemIPAddressRule)
			}
		}

	} else if len(elasticLoadBalancer.NetworkInterfaces) == 2 {
		// two arm
		if healthCheckProtocol == "ICMP" {
			var notVIPNetworkInterface NetworkInterface
			if elasticLoadBalancer.NetworkInterfaces[0].IsVipNetwork {
				notVIPNetworkInterface = elasticLoadBalancer.NetworkInterfaces[1]
			} else {
				notVIPNetworkInterface = elasticLoadBalancer.NetworkInterfaces[0]
			}

			IPAddressRule := SecurityGroupRule{
				IpProtocol: healthCheckProtocol,
				InOut:      "IN",
				IpRanges:   []string{notVIPNetworkInterface.IPAddress},
			}
			securityGroupRules = append(securityGroupRules, IPAddressRule)

			for _, systemIPAddress := range notVIPNetworkInterface.SystemIpAddresses {
				systemIPAddressRule := SecurityGroupRule{
					IpProtocol: healthCheckProtocol,
					InOut:      "IN",
					IpRanges:   []string{systemIPAddress},
				}
				securityGroupRules = append(securityGroupRules, systemIPAddressRule)
			}
		} else {
			var notVIPNetworkInterface NetworkInterface
			if elasticLoadBalancer.NetworkInterfaces[0].IsVipNetwork {
				notVIPNetworkInterface = elasticLoadBalancer.NetworkInterfaces[1]
			} else {
				notVIPNetworkInterface = elasticLoadBalancer.NetworkInterfaces[0]
			}

			IPAddressRule := SecurityGroupRule{
				IpProtocol: healthCheckProtocol,
				FromPort:   elasticLoadBalancer.InstancePort,
				ToPort:     elasticLoadBalancer.InstancePort,
				InOut:      "IN",
				IpRanges:   []string{notVIPNetworkInterface.IPAddress},
			}
			securityGroupRules = append(securityGroupRules, IPAddressRule)

			for _, systemIPAddress := range notVIPNetworkInterface.SystemIpAddresses {
				systemIPAddressRule := SecurityGroupRule{
					IpProtocol: healthCheckProtocol,
					FromPort:   elasticLoadBalancer.InstancePort,
					ToPort:     elasticLoadBalancer.InstancePort,
					InOut:      "IN",
					IpRanges:   []string{systemIPAddress},
				}
				securityGroupRules = append(securityGroupRules, systemIPAddressRule)
			}
		}
	} else {
		return nil, fmt.Errorf("the number of NetworkInterfaces (%d) is invalid", len(elasticLoadBalancer.NetworkInterfaces))
	}

	return securityGroupRules, nil
}

func separateHealthCheckTarget(healthCheckTarget string) (string, string) {
	if healthCheckTarget == "ICMP" {
		return "ICMP", ""
	}
	r := regexp.MustCompile("^(TCP|HTTP|HTTPS):([0-9]+)$")
	match := r.FindStringSubmatch(healthCheckTarget)
	return match[1], match[2]
}

func (c *Cloud) updateElasticLoadBalancer(ctx context.Context, clusterName string, service *v1.Service) error {
	// get elastic load balancer name
	loadBalancerName := c.GetLoadBalancerName(ctx, clusterName, service)

	// describe load balancer
	loadBalancers, err := c.client.DescribeElasticLoadBalancers(ctx, loadBalancerName)
	if err != nil {
		return err
	}
	if len(loadBalancers) == 0 {
		return fmt.Errorf("load balancer %q not found", loadBalancerName)
	}

	return nil
}

func (c *Cloud) ensureElasticLoadBalancerDeleted(ctx context.Context, clusterName string, service *v1.Service) error {
	// get elastic load balancer name
	loadBalancerName := c.GetLoadBalancerName(ctx, clusterName, service)

	// describe load balancer
	loadBalancers, err := c.client.DescribeElasticLoadBalancers(ctx, loadBalancerName)
	if err != nil {
		switch {
		case isAPIError(err, errorCodeElasticLoadBalancerNotFound):
			klog.Infof("Load balancer %q is not found", loadBalancerName)
			return nil
		}
		return err
	}
	if len(loadBalancers) == 0 {
		klog.Infof("Load balancer %q already deleted", loadBalancerName)
		return nil
	}

	// delete load balancer
	for _, lb := range loadBalancers {
		klog.Infof("Deleting LoadBalancer %q (%d -> %d)", lb.Name, lb.LoadBalancerPort, lb.InstancePort)
		if err := c.client.DeleteElasticLoadBalancer(ctx, &lb); err != nil {
			return fmt.Errorf("failed to delete load balancer: %w", err)
		}
		if err := c.denySecurityGroupRulesFromElasticLoadBalancer(ctx, &lb, lb.BalancingTargets); err != nil {
			return err
		}
	}

	return nil
}

func findElasticLoadBalancer(from []ElasticLoadBalancer, target ElasticLoadBalancer) (*ElasticLoadBalancer, error) {
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

func elasticLoadBalancerDifferences(target, other []ElasticLoadBalancer) []ElasticLoadBalancer {
	diff := []ElasticLoadBalancer{}
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

func elasticLoadBalancingTargetsDifferences(target, other []Instance) []Instance {
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
