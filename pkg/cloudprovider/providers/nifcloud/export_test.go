package nifcloud

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/nifcloud/nifcloud-sdk-go/nifcloud"
	"github.com/nifcloud/nifcloud-sdk-go/service/computing"
)

// nifcloud.go

func (c *Cloud) SetClient(client CloudAPIClient) {
	c.client = client
}

func (c *Cloud) SetRegion(region string) {
	c.region = region
}

// nifcloud_client.go

type ExportNifcloudAPIClient = nifcloudAPIClient

func NewNIFCLOUDAPIClientWithEndpoint(accessKeyID, secretAccessKey, region, endpoint string) *ExportNifcloudAPIClient {
	cfg := nifcloud.NewConfig(accessKeyID, secretAccessKey, region)
	cfg.EndpointResolverWithOptions = aws.EndpointResolverWithOptionsFunc(
		func(_, region string, _ ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{
				URL:           endpoint,
				SigningRegion: region,
			}, nil
		},
	)
	return &ExportNifcloudAPIClient{
		client: computing.NewFromConfig(cfg),
	}
}

var ExportCreateLoadBalancer = (*ExportNifcloudAPIClient).createLoadBalancer
var ExportRegisterPortWithLoadBalancer = (*ExportNifcloudAPIClient).registerPortWithLoadBalancer
var ExportCreateElasticLoadBalancer = (*ExportNifcloudAPIClient).createElasticLoadBalancer
var ExportRegisterPortWithElasticLoadBalancer = (*ExportNifcloudAPIClient).registerPortWithElasticLoadBalancer

// nifcloud_instances.go

var ExportGetInstance = (*Cloud).getInstance
var ExportGetNodeAddress = getNodeAddress

// nifcloud_load_balancer.go

var ExportMaxLoadBalancerNameLength = maxLoadBalancerNameLength
var ExportValidateLoadBalancerAnnotations = validateLoadBalancerAnnotations

// nifcloud_l4_load_balancer.go

var ExportIsL4LoadBalancer = isL4LoadBalancer
var ExportGetL4LoadBalancer = (*Cloud).getL4LoadBalancer
var ExportEnsureL4LoadBalancer = (*Cloud).ensureL4LoadBalancer
var ExportUpdateL4LoadBalancer = (*Cloud).updateL4LoadBalancer
var ExportEnsureL4LoadBalancerDeleted = (*Cloud).ensureL4LoadBalancerDeleted
var ExportFindL4LoadBalancer = findL4LoadBalancer
var ExportL4LoadBalancerDifferences = l4LoadBalancerDifferences
var ExportL4LoadBalancingTargetsDifferences = l4LoadBalancingTargetsDifferences
var ExportFilterDifferences = filterDifferences

// nifcloud_elastic_load_balancer.go

var ExportGetElasticLoadBalancer = (*Cloud).getElasticLoadBalancer
var ExportEnsureElasticLoadBalancer = (*Cloud).ensureElasticLoadBalancer
var ExportUpdateElasticLoadBalancer = (*Cloud).updateElasticLoadBalancer
var ExportEnsureElasticLoadBalancerDeleted = (*Cloud).ensureElasticLoadBalancerDeleted
var ExportSecurityGroupRulesOfElasticLoadBalancer = securityGroupRulesOfElasticLoadBalancer
var ExportSeparateHealthCheckTarget = separateHealthCheckTarget
var ExportFindElasticLoadBalancer = findElasticLoadBalancer
var ExportElasticLoadBalancerDifferences = elasticLoadBalancerDifferences
var ExportElasticLoadBalancingTargetsDifferences = elasticLoadBalancingTargetsDifferences

// nifcloud_error_code.go

const (
	ExportErrorCodeInstanceNotFound             = errorCodeInstanceNotFound
	ExportErrorCodeLoadBalancerNotFound         = errorCodeLoadBalancerNotFound
	ExportErrorCodeElasticLoadBalancerNotFound  = errorCodeElasticLoadBalancerNotFound
	ExportErrorCodeSecurityGroupIngressNotFound = errorCodeSecurityGroupIngressNotFound
	ExportErrorCodeSecurityGroupDuplicate       = errorCodeSecurityGroupDuplicate
)
