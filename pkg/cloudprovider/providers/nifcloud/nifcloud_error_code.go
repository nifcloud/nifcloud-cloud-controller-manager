package nifcloud

import (
	goerrors "errors"

	"github.com/aws/smithy-go"
)

const (
	// Instance
	errorCodeInstanceNotFound = "Client.InvalidParameterNotFound.Instance"

	// LoadBalancer
	errorCodeLoadBalancerNotFound = "Client.InvalidParameterNotFound.LoadBalancer"

	// ElasticLoadBalancer
	errorCodeElasticLoadBalancerNotFound = "Client.InvalidParameterNotFound.ElasticLoadBalancer"

	// SecurityGroup
	errorCodeSecurityGroupIngressNotFound = "Client.InvalidParameterNotFound.SecurityGroupIngress"
	errorCodeSecurityGroupDuplicate       = "Client.InvalidParameterDuplicate.SecurityGroup"
)

func isAPIError(err error, code string) bool {
	var awsErr smithy.APIError
	if goerrors.As(err, &awsErr) {
		return awsErr.ErrorCode() == code
	}
	return false
}
