package helper

import (
	"github.com/nifcloud/nifcloud-cloud-controller-manager/pkg/cloudprovider/providers/nifcloud"
)

func NewTestElasticLoadBalancer(name string) []nifcloud.ElasticLoadBalancer {
	instances := []nifcloud.Instance{
		*NewTestInstance(),
	}
	return []nifcloud.ElasticLoadBalancer{
		{
			AvailabilityZone:              "east-11",
			Name:                          name,
			AccountingType:                "1",
			Protocol:                      "TCP",
			NetworkVolume:                 100,
			LoadBalancerPort:              80,
			InstancePort:                  30000,
			HealthCheckInterval:           10,
			HealthCheckUnhealthyThreshold: 1,
			HealthCheckTarget:             "TCP:30000",
			BalancingType:                 1,
			BalancingTargets:              instances,
			NetworkInterfaces: []nifcloud.NetworkInterface{
				NewTestNetworkInterfaceCommonGlobal(),
			},
		},
	}
}

func NewTestElasticLoadBalancerWithTwoPort(name string) []nifcloud.ElasticLoadBalancer {
	instances := []nifcloud.Instance{
		*NewTestInstance(),
	}
	return []nifcloud.ElasticLoadBalancer{
		{
			AvailabilityZone:              "east-11",
			Name:                          name,
			AccountingType:                "1",
			Protocol:                      "TCP",
			NetworkVolume:                 100,
			LoadBalancerPort:              80,
			InstancePort:                  30000,
			HealthCheckInterval:           10,
			HealthCheckUnhealthyThreshold: 1,
			HealthCheckTarget:             "TCP:30000",
			BalancingType:                 1,
			BalancingTargets:              instances,
			NetworkInterfaces: []nifcloud.NetworkInterface{
				NewTestNetworkInterfaceCommonGlobal(),
			},
		},
		{
			AvailabilityZone:              "east-11",
			Name:                          name,
			AccountingType:                "1",
			Protocol:                      "TCP",
			NetworkVolume:                 100,
			LoadBalancerPort:              443,
			InstancePort:                  30001,
			HealthCheckInterval:           10,
			HealthCheckUnhealthyThreshold: 1,
			HealthCheckTarget:             "TCP:30001",
			BalancingType:                 1,
			BalancingTargets:              instances,
			NetworkInterfaces: []nifcloud.NetworkInterface{
				NewTestNetworkInterfaceCommonGlobal(),
			},
		},
	}
}
