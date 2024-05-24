package helper

import (
	"github.com/nifcloud/nifcloud-cloud-controller-manager/pkg/cloudprovider/providers/nifcloud"
)

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
