package helper

import (
	"github.com/nifcloud/nifcloud-cloud-controller-manager/pkg/cloudprovider/providers/nifcloud"
)

func NewTestL4LoadBalancer(name string) []nifcloud.LoadBalancer {
	instances := []nifcloud.Instance{
		*NewTestInstance(),
	}
	return []nifcloud.LoadBalancer{
		{
			Name:                          name,
			AccountingType:                "1",
			NetworkVolume:                 100,
			PolicyType:                    "standard",
			BalancingType:                 1,
			BalancingTargets:              instances,
			LoadBalancerPort:              80,
			InstancePort:                  30000,
			HealthCheckTarget:             "TCP:30000",
			HealthCheckInterval:           10,
			HealthCheckUnhealthyThreshold: 1,
			Filters:                       []string{},
		},
	}
}

func NewTestL4LoadBalancerWithTwoPort(name string) []nifcloud.LoadBalancer {
	instances := []nifcloud.Instance{
		*NewTestInstance(),
	}
	return []nifcloud.LoadBalancer{
		{
			Name:                          name,
			AccountingType:                "1",
			NetworkVolume:                 100,
			PolicyType:                    "standard",
			BalancingType:                 1,
			BalancingTargets:              instances,
			LoadBalancerPort:              80,
			InstancePort:                  30000,
			HealthCheckTarget:             "TCP:30000",
			HealthCheckInterval:           10,
			HealthCheckUnhealthyThreshold: 1,
			Filters:                       []string{},
		},
		{
			Name:                          name,
			AccountingType:                "1",
			NetworkVolume:                 100,
			PolicyType:                    "standard",
			BalancingType:                 1,
			BalancingTargets:              instances,
			LoadBalancerPort:              443,
			InstancePort:                  30001,
			HealthCheckTarget:             "TCP:30001",
			HealthCheckInterval:           10,
			HealthCheckUnhealthyThreshold: 1,
			Filters:                       []string{},
		},
	}
}
