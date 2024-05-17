package nifcloud

// nifcloud.go

func (c *Cloud) SetClint(client CloudAPIClient) {
	c.client = client
}

func (c *Cloud) SetRegion(region string) {
	c.region = region
}

// nifcloud_load_balancer.go

var ExportMaxLoadBalancerNameLength = maxLoadBalancerNameLength

// nifcloud_elastic_load_balancer.go

var ExportGetElasticLoadBalancer = (*Cloud).getElasticLoadBalancer
var ExportEnsureElasticLoadBalancer = (*Cloud).ensureElasticLoadBalancer
var ExportUpdateElasticLoadBalancer = (*Cloud).updateElasticLoadBalancer
var ExportEnsureElasticLoadBalancerDeleted = (*Cloud).ensureElasticLoadBalancerDeleted
var ExportSecurityGroupRulesOfElasticLoadBalancer = securityGroupRulesOfElasticLoadBalancer
var ExportSeparateHealthCheckTarget = separateHealthCheckTarget
var ExportFindElasticLoadBalancer = findElasticLoadBalancer

// nifcloud_error_code.go

const (
	ExportErrorCodeInstanceNotFound             = errorCodeInstanceNotFound
	ExportErrorCodeLoadBalancerNotFound         = errorCodeLoadBalancerNotFound
	ExportErrorCodeElasticLoadBalancerNotFound  = errorCodeElasticLoadBalancerNotFound
	ExportErrorCodeSecurityGroupIngressNotFound = errorCodeSecurityGroupIngressNotFound
	ExportErrorCodeSecurityGroupDuplicate       = errorCodeSecurityGroupDuplicate
)
