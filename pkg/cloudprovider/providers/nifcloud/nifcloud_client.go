package nifcloud

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/aokumasan/nifcloud-sdk-go-v2/nifcloud"
	"github.com/aokumasan/nifcloud-sdk-go-v2/service/computing"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/private/protocol/query/queryutil"
	cloudprovider "k8s.io/cloud-provider"
)

// Instance is instance detail
type Instance struct {
	InstanceID       string
	InstanceUniqueID string
	InstanceType     string
	PublicIPAddress  string
	PrivateIPAddress string
	Zone             string
}

// CloudAPIClient is interface
type CloudAPIClient interface {
	// Instance
	DescribeInstancesByInstanceID(ctx context.Context, instanceIDs []string) ([]Instance, error)
	DescribeInstancesByInstanceUniqueID(ctx context.Context, instanceUniqueIDs []string) ([]Instance, error)
}

type nifcloudAPIClient struct {
	client *computing.Client
}

func newNIFCLOUDAPIClient(accessKeyID, secretAccessKey, region string) CloudAPIClient {
	cfg := nifcloud.NewConfig(accessKeyID, secretAccessKey, region)
	return &nifcloudAPIClient{
		client: computing.New(cfg),
	}
}

func (c *nifcloudAPIClient) DescribeInstancesByInstanceID(ctx context.Context, instanceIDs []string) ([]Instance, error) {
	req := c.client.DescribeInstancesRequest(
		&computing.DescribeInstancesInput{
			InstanceId: instanceIDs,
		},
	)
	res, err := req.Send(ctx)
	if err != nil {
		return nil, handleNotFoundError(err)
	}

	if err := checkReservationSet(res.ReservationSet); err != nil {
		return nil, err
	}

	instances := []Instance{}
	for _, instance := range res.ReservationSet[0].InstancesSet {
		instances = append(instances, Instance{
			InstanceID:       nifcloud.StringValue(instance.InstanceId),
			InstanceUniqueID: nifcloud.StringValue(instance.InstanceUniqueId),
			InstanceType:     nifcloud.StringValue(instance.InstanceType),
			PublicIPAddress:  nifcloud.StringValue(instance.IpAddress),
			PrivateIPAddress: nifcloud.StringValue(instance.PrivateIpAddress),
			Zone:             nifcloud.StringValue(instance.Placement.AvailabilityZone),
		})
	}

	return instances, nil
}

func (c *nifcloudAPIClient) DescribeInstancesByInstanceUniqueID(ctx context.Context, instanceUniqueIDs []string) ([]Instance, error) {
	req := c.client.DescribeInstancesRequest(nil)
	if err := req.Request.Build(); err != nil {
		return nil, fmt.Errorf("failed building request: %v", err)
	}
	body := url.Values{
		"Action":  {req.Operation.Name},
		"Version": {req.Metadata.APIVersion},
	}
	if err := queryutil.Parse(body, req.Params, false); err != nil {
		return nil, fmt.Errorf("failed encoding request: %v", err)
	}
	for i, uniqueID := range instanceUniqueIDs {
		body.Set(fmt.Sprintf("InstanceUniqueId.%d", i), uniqueID)
	}
	req.SetBufferBody([]byte(body.Encode()))

	res, err := req.Send(ctx)
	if err != nil {
		return nil, handleNotFoundError(err)
	}

	if err := checkReservationSet(res.ReservationSet); err != nil {
		return nil, err
	}

	instances := []Instance{}
	for _, instance := range res.ReservationSet[0].InstancesSet {
		instances = append(instances, Instance{
			InstanceID:       nifcloud.StringValue(instance.InstanceId),
			InstanceUniqueID: nifcloud.StringValue(instance.InstanceUniqueId),
			InstanceType:     nifcloud.StringValue(instance.InstanceType),
			PublicIPAddress:  nifcloud.StringValue(instance.IpAddress),
			PrivateIPAddress: nifcloud.StringValue(instance.PrivateIpAddress),
			Zone:             nifcloud.StringValue(instance.Placement.AvailabilityZone),
		})
	}

	return instances, nil
}

func handleNotFoundError(err error) error {
	switch err.(type) {
	case awserr.Error:
		if strings.Contains(err.Error(), "NotFound") {
			return cloudprovider.InstanceNotFound
		}
		return err
	default:
		return err
	}
}

func checkReservationSet(rs []computing.ReservationSetItem) error {
	if len(rs) == 0 {
		return cloudprovider.InstanceNotFound
	}

	if len(rs[0].InstancesSet) == 0 {
		return cloudprovider.InstanceNotFound
	}

	return nil
}
