package helper

import (
	"github.com/nifcloud/nifcloud-cloud-controller-manager/pkg/cloudprovider/providers/nifcloud"
)

func NewTestEmptySecurityGroups() []nifcloud.SecurityGroup {
	return []nifcloud.SecurityGroup{
		{
			GroupName: "testsecuritygroup",
			Rules:     []nifcloud.SecurityGroupRule{},
		},
	}
}
