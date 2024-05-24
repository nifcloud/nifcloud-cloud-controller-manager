package helper

import (
	"github.com/nifcloud/nifcloud-cloud-controller-manager/pkg/cloudprovider/providers/nifcloud"
)

func NewTestNetworkInterfaceCommonGlobal() nifcloud.NetworkInterface {
	return nifcloud.NetworkInterface{
		NetworkId:    "net-COMMON_GLOBAL",
		IsVipNetwork: true,
	}
}
