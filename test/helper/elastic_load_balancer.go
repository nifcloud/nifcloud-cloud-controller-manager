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

func NewTestInstance() *nifcloud.Instance {
	return &nifcloud.Instance{
		InstanceID:       "testinstance",
		InstanceUniqueID: "i-abcd1234",
		InstanceType:     "h2-large16",
		PublicIPAddress:  "203.0.113.1",
		PrivateIPAddress: "192.168.0.100",
		Zone:             "east-11",
		State:            "running",
	}
}

func NewTestEmptySecurityGroups() []nifcloud.SecurityGroup {
	return []nifcloud.SecurityGroup{
		{
			GroupName: "testsecuritygroup",
			Rules:     []nifcloud.SecurityGroupRule{},
		},
	}
}

func NewTestNetworkInterfaceCommonGlobal() nifcloud.NetworkInterface {
	return nifcloud.NetworkInterface{
		NetworkId:    "net-COMMON_GLOBAL",
		IsVipNetwork: true,
	}
}
