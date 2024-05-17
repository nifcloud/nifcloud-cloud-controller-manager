package nifcloud_test

import (
	"context"
	"strings"

	"github.com/nifcloud/nifcloud-cloud-controller-manager/pkg/cloudprovider/providers/nifcloud"
	"github.com/nifcloud/nifcloud-cloud-controller-manager/test/helper"
	"github.com/samber/lo"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("getElasticLoadBalancer", func() {
	var ctrl *gomock.Controller
	var region string = "east1"
	var loadBalancerUID types.UID
	var loadBalancerName string

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		loadBalancerUID = types.UID(uuid.NewString())
		loadBalancerName = strings.Replace(string(loadBalancerUID), "-", "", -1)[:nifcloud.ExportMaxLoadBalancerNameLength]
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("the specified elastic load balancer is existed", func() {
		It("return the status", func() {
			ctx := context.Background()
			clusterName := "testCluster"
			service := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name: clusterName,
					UID:  loadBalancerUID,
				},
			}
			testIPAddress := "203.0.113.1"

			expectedStatus := &corev1.LoadBalancerStatus{
				Ingress: []corev1.LoadBalancerIngress{
					{
						IP: testIPAddress,
					},
				},
			}

			c := nifcloud.NewMockCloudAPIClient(ctrl)
			c.EXPECT().
				DescribeElasticLoadBalancers(gomock.Any(), gomock.Eq(loadBalancerName)).
				Return([]nifcloud.ElasticLoadBalancer{
					{
						Name: loadBalancerName,
						VIP:  testIPAddress,
					},
				}, nil).
				Times(1)

			cloud := &nifcloud.Cloud{}
			cloud.SetClint(c)
			cloud.SetRegion(region)

			status, exists, err := nifcloud.ExportGetElasticLoadBalancer(cloud, ctx, clusterName, service)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(exists).Should(BeTrue())
			Expect(*status).Should(Equal(*expectedStatus))
		})
	})

	Context("the specified elastic load balancer is not existed", func() {
		It("return that exists is false", func() {
			ctx := context.Background()
			clusterName := "testCluster"
			service := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name: clusterName,
					UID:  loadBalancerUID,
				},
			}

			apiErr := helper.NewMockAPIError(nifcloud.ExportErrorCodeElasticLoadBalancerNotFound)

			c := nifcloud.NewMockCloudAPIClient(ctrl)
			c.EXPECT().
				DescribeElasticLoadBalancers(gomock.Any(), gomock.Eq(loadBalancerName)).
				Return([]nifcloud.ElasticLoadBalancer{}, apiErr).
				Times(1)

			cloud := &nifcloud.Cloud{}
			cloud.SetClint(c)
			cloud.SetRegion(region)

			status, exists, err := nifcloud.ExportGetElasticLoadBalancer(cloud, ctx, clusterName, service)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(exists).Should(BeFalse())
			Expect(status).Should(BeNil())
		})
	})

	Context("DescribeElasticLoadBalancers return unknown error code", func() {
		It("return the error", func() {
			ctx := context.Background()
			clusterName := "testCluster"
			service := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name: clusterName,
					UID:  loadBalancerUID,
				},
			}

			errorCodeUnknown := "Client.Unknown"

			apiErr := helper.NewMockAPIError(errorCodeUnknown)

			c := nifcloud.NewMockCloudAPIClient(ctrl)
			c.EXPECT().
				DescribeElasticLoadBalancers(gomock.Any(), gomock.Eq(loadBalancerName)).
				Return([]nifcloud.ElasticLoadBalancer{}, apiErr).
				Times(1)

			cloud := &nifcloud.Cloud{}
			cloud.SetClint(c)
			cloud.SetRegion(region)

			status, exists, err := nifcloud.ExportGetElasticLoadBalancer(cloud, ctx, clusterName, service)
			Expect(err).Should(HaveOccurred())
			Expect(exists).Should(BeFalse())
			Expect(status).Should(BeNil())
		})
	})
})

var _ = Describe("ensureElasticLoadBalancer", func() {
	var ctrl *gomock.Controller
	var region string = "east1"
	var loadBalancerUID types.UID
	var loadBalancerName string

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		loadBalancerUID = types.UID(uuid.NewString())
		loadBalancerName = strings.Replace(string(loadBalancerUID), "-", "", -1)[:nifcloud.ExportMaxLoadBalancerNameLength]
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("the specified elastic load balancer is not existed", func() {
		Context("the elastic load balancer has one port", func() {
			It("create the elastic load balancer", func() {
				ctx := context.Background()
				testIPAddress := "203.0.113.1"
				testDesire := helper.NewTestElasticLoadBalancer(loadBalancerName)
				createdELB := helper.NewTestElasticLoadBalancer(loadBalancerName)
				createdELB[0].VIP = testIPAddress
				createdELB[0].NetworkInterfaces[0].SystemIpAddresses = []string{"203.0.113.10", "203.0.113.11"}

				testSecurityGroups := helper.NewTestEmptySecurityGroups()

				expectedStatus := &corev1.LoadBalancerStatus{
					Ingress: []corev1.LoadBalancerIngress{
						{
							IP: testIPAddress,
						},
					},
				}

				c := nifcloud.NewMockCloudAPIClient(ctrl)
				notFoundErr := helper.NewMockAPIError(nifcloud.ExportErrorCodeElasticLoadBalancerNotFound)
				gomock.InOrder(
					c.EXPECT().
						DescribeElasticLoadBalancers(gomock.Any(), gomock.Eq(loadBalancerName)).
						Return([]nifcloud.ElasticLoadBalancer{}, notFoundErr).
						Times(1),
					c.EXPECT().
						DescribeElasticLoadBalancers(gomock.Any(), gomock.Eq(loadBalancerName)).
						Return(createdELB, nil).
						Times(1),
				)
				c.EXPECT().
					CreateElasticLoadBalancer(gomock.Any(), gomock.Eq(&testDesire[0])).
					Return(testIPAddress, nil).
					Times(1)
				expectedInstanceIDs := lo.Map(createdELB[0].BalancingTargets, func(instance nifcloud.Instance, _ int) string {
					return instance.InstanceID
				})
				c.EXPECT().
					DescribeSecurityGroupsByInstanceIDs(gomock.Any(), gomock.Eq(expectedInstanceIDs)).
					Return(testSecurityGroups, nil).
					Times(1)
				createdSecurityGroupRules := []nifcloud.SecurityGroupRule{
					{
						IpProtocol: createdELB[0].Protocol,
						FromPort:   createdELB[0].InstancePort,
						ToPort:     createdELB[0].InstancePort,
						InOut:      "IN",
						IpRanges:   []string{testIPAddress},
					},
					{
						IpProtocol: createdELB[0].Protocol,
						FromPort:   createdELB[0].InstancePort,
						ToPort:     createdELB[0].InstancePort,
						InOut:      "IN",
						IpRanges:   []string{createdELB[0].NetworkInterfaces[0].SystemIpAddresses[0]},
					},
					{
						IpProtocol: createdELB[0].Protocol,
						FromPort:   createdELB[0].InstancePort,
						ToPort:     createdELB[0].InstancePort,
						InOut:      "IN",
						IpRanges:   []string{createdELB[0].NetworkInterfaces[0].SystemIpAddresses[1]},
					},
				}
				callAuthorizeSecurityGroupIngressTime := 0
				c.EXPECT().
					AuthorizeSecurityGroupIngress(gomock.Any(), gomock.Eq(testSecurityGroups[0].GroupName), gomock.Any()).
					Do(func(_ context.Context, _ string, securityGroupRule *nifcloud.SecurityGroupRule) {
						Expect(*securityGroupRule).Should(Equal(createdSecurityGroupRules[callAuthorizeSecurityGroupIngressTime]))
						callAuthorizeSecurityGroupIngressTime += 1
					}).
					Return(nil).
					Times(3)
				c.EXPECT().
					WaitSecurityGroupApplied(gomock.Any(), gomock.Eq(testSecurityGroups[0].GroupName)).
					Return(nil).
					Times(3)

				cloud := &nifcloud.Cloud{}
				cloud.SetClint(c)
				cloud.SetRegion(region)

				status, err := nifcloud.ExportEnsureElasticLoadBalancer(cloud, ctx, loadBalancerName, testDesire)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(*status).Should(Equal(*expectedStatus))
			})
		})

		Context("the elastic load balancer has two ports", func() {
			It("create the elastic load balancer", func() {
				ctx := context.Background()
				testIPAddress := "203.0.113.1"
				testDesire := helper.NewTestElasticLoadBalancerWithTwoPort(loadBalancerName)
				createdELB := helper.NewTestElasticLoadBalancerWithTwoPort(loadBalancerName)
				for i := range createdELB {
					createdELB[i].VIP = testIPAddress
					createdELB[i].NetworkInterfaces[0].SystemIpAddresses = []string{"203.0.113.10", "203.0.113.11"}
				}

				testSecurityGroups := helper.NewTestEmptySecurityGroups()

				expectedStatus := &corev1.LoadBalancerStatus{
					Ingress: []corev1.LoadBalancerIngress{
						{
							IP: testIPAddress,
						},
					},
				}

				c := nifcloud.NewMockCloudAPIClient(ctrl)
				notFoundErr := helper.NewMockAPIError(nifcloud.ExportErrorCodeElasticLoadBalancerNotFound)
				gomock.InOrder(
					c.EXPECT().
						DescribeElasticLoadBalancers(gomock.Any(), gomock.Eq(loadBalancerName)).
						Return([]nifcloud.ElasticLoadBalancer{}, notFoundErr).
						Times(1),
					c.EXPECT().
						DescribeElasticLoadBalancers(gomock.Any(), gomock.Eq(loadBalancerName)).
						Return(createdELB, nil).
						Times(1),
				)
				c.EXPECT().
					CreateElasticLoadBalancer(gomock.Any(), gomock.Eq(&testDesire[0])).
					Return(testIPAddress, nil).
					Times(1)
				expectedInstanceIDs := lo.Map(createdELB[0].BalancingTargets, func(instance nifcloud.Instance, _ int) string {
					return instance.InstanceID
				})
				c.EXPECT().
					DescribeSecurityGroupsByInstanceIDs(gomock.Any(), gomock.Eq(expectedInstanceIDs)).
					Return(testSecurityGroups, nil).
					Times(2)
				creaetdSecurityGroupRules := []nifcloud.SecurityGroupRule{
					{
						IpProtocol: createdELB[0].Protocol,
						FromPort:   createdELB[0].InstancePort,
						ToPort:     createdELB[0].InstancePort,
						InOut:      "IN",
						IpRanges:   []string{testIPAddress},
					},
					{
						IpProtocol: createdELB[0].Protocol,
						FromPort:   createdELB[0].InstancePort,
						ToPort:     createdELB[0].InstancePort,
						InOut:      "IN",
						IpRanges:   []string{createdELB[0].NetworkInterfaces[0].SystemIpAddresses[0]},
					},
					{
						IpProtocol: createdELB[0].Protocol,
						FromPort:   createdELB[0].InstancePort,
						ToPort:     createdELB[0].InstancePort,
						InOut:      "IN",
						IpRanges:   []string{createdELB[0].NetworkInterfaces[0].SystemIpAddresses[1]},
					},
					{
						IpProtocol: createdELB[1].Protocol,
						FromPort:   createdELB[1].InstancePort,
						ToPort:     createdELB[1].InstancePort,
						InOut:      "IN",
						IpRanges:   []string{testIPAddress},
					},
					{
						IpProtocol: createdELB[1].Protocol,
						FromPort:   createdELB[1].InstancePort,
						ToPort:     createdELB[1].InstancePort,
						InOut:      "IN",
						IpRanges:   []string{createdELB[1].NetworkInterfaces[0].SystemIpAddresses[0]},
					},
					{
						IpProtocol: createdELB[1].Protocol,
						FromPort:   createdELB[1].InstancePort,
						ToPort:     createdELB[1].InstancePort,
						InOut:      "IN",
						IpRanges:   []string{createdELB[1].NetworkInterfaces[0].SystemIpAddresses[1]},
					},
				}
				callAuthorizeSecurityGroupIngressTime := 0
				c.EXPECT().
					AuthorizeSecurityGroupIngress(gomock.Any(), gomock.Eq(testSecurityGroups[0].GroupName), gomock.Any()).
					Do(func(_ context.Context, _ string, securityGroupRule *nifcloud.SecurityGroupRule) {
						Expect(*securityGroupRule).Should(Equal(creaetdSecurityGroupRules[callAuthorizeSecurityGroupIngressTime]))
						callAuthorizeSecurityGroupIngressTime += 1
					}).
					Return(nil).
					Times(6)
				c.EXPECT().
					WaitSecurityGroupApplied(gomock.Any(), gomock.Eq(testSecurityGroups[0].GroupName)).
					Return(nil).
					Times(6)
				c.EXPECT().
					RegisterPortWithElasticLoadBalancer(gomock.Any(), gomock.Eq(&testDesire[1])).
					Return(nil).
					Times(1)

				cloud := &nifcloud.Cloud{}
				cloud.SetClint(c)
				cloud.SetRegion(region)

				status, err := nifcloud.ExportEnsureElasticLoadBalancer(cloud, ctx, loadBalancerName, testDesire)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(*status).Should(Equal(*expectedStatus))
			})
		})
	})

	Context("the specified elastic load balancer is existed", func() {
		Context("add a port to the elastic load balancer", func() {
			It("create the port", func() {
				ctx := context.Background()
				testIPAddress := "203.0.113.1"
				existedELB := helper.NewTestElasticLoadBalancer(loadBalancerName)
				existedELB[0].VIP = testIPAddress
				existedELB[0].NetworkInterfaces[0].SystemIpAddresses = []string{"203.0.113.10", "203.0.113.11"}
				testDesire := helper.NewTestElasticLoadBalancerWithTwoPort(loadBalancerName)
				updatedELB := helper.NewTestElasticLoadBalancerWithTwoPort(loadBalancerName)
				for i := range updatedELB {
					updatedELB[i].VIP = testIPAddress
					updatedELB[i].NetworkInterfaces[0].SystemIpAddresses = []string{"203.0.113.10", "203.0.113.11"}
				}
				testSecurityGroups := helper.NewTestEmptySecurityGroups()

				expectedStatus := &corev1.LoadBalancerStatus{
					Ingress: []corev1.LoadBalancerIngress{
						{
							IP: testIPAddress,
						},
					},
				}

				c := nifcloud.NewMockCloudAPIClient(ctrl)
				gomock.InOrder(
					c.EXPECT().
						DescribeElasticLoadBalancers(gomock.Any(), gomock.Eq(loadBalancerName)).
						Return(existedELB, nil).
						Times(1),
					c.EXPECT().
						DescribeElasticLoadBalancers(gomock.Any(), gomock.Eq(loadBalancerName)).
						Return(updatedELB, nil).
						Times(1),
				)
				c.EXPECT().
					RegisterPortWithElasticLoadBalancer(gomock.Any(), gomock.Eq(&testDesire[1])).
					Return(nil).
					Times(1)
				expectedInstanceIDs := lo.Map(updatedELB[0].BalancingTargets, func(instance nifcloud.Instance, _ int) string {
					return instance.InstanceID
				})
				c.EXPECT().
					DescribeSecurityGroupsByInstanceIDs(gomock.Any(), gomock.Eq(expectedInstanceIDs)).
					Return(testSecurityGroups, nil).
					Times(1)
				createdSecurityGroupRules := []nifcloud.SecurityGroupRule{
					{
						IpProtocol: updatedELB[1].Protocol,
						FromPort:   updatedELB[1].InstancePort,
						ToPort:     updatedELB[1].InstancePort,
						InOut:      "IN",
						IpRanges:   []string{testIPAddress},
					},
					{
						IpProtocol: updatedELB[1].Protocol,
						FromPort:   updatedELB[1].InstancePort,
						ToPort:     updatedELB[1].InstancePort,
						InOut:      "IN",
						IpRanges:   []string{updatedELB[1].NetworkInterfaces[0].SystemIpAddresses[0]},
					},
					{
						IpProtocol: updatedELB[1].Protocol,
						FromPort:   updatedELB[1].InstancePort,
						ToPort:     updatedELB[1].InstancePort,
						InOut:      "IN",
						IpRanges:   []string{updatedELB[1].NetworkInterfaces[0].SystemIpAddresses[1]},
					},
				}
				callAuthorizeSecurityGroupIngressTime := 0
				c.EXPECT().
					AuthorizeSecurityGroupIngress(gomock.Any(), gomock.Eq(testSecurityGroups[0].GroupName), gomock.Any()).
					Do(func(_ context.Context, _ string, securityGroupRule *nifcloud.SecurityGroupRule) {
						Expect(*securityGroupRule).Should(Equal(createdSecurityGroupRules[callAuthorizeSecurityGroupIngressTime]))
						callAuthorizeSecurityGroupIngressTime += 1
					}).
					Return(nil).
					Times(3)
				c.EXPECT().
					WaitSecurityGroupApplied(gomock.Any(), gomock.Eq(testSecurityGroups[0].GroupName)).
					Return(nil).
					Times(3)

				cloud := &nifcloud.Cloud{}
				cloud.SetClint(c)
				cloud.SetRegion(region)

				status, err := nifcloud.ExportEnsureElasticLoadBalancer(cloud, ctx, loadBalancerName, testDesire)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(*status).Should(Equal(*expectedStatus))
			})
		})

		Context("delete one port from the elastic load balancer", func() {
			It("delete the port", func() {
				ctx := context.Background()
				testIPAddress := "203.0.113.1"
				existedELB := helper.NewTestElasticLoadBalancerWithTwoPort(loadBalancerName)
				for i := range existedELB {
					existedELB[i].VIP = testIPAddress
					existedELB[i].NetworkInterfaces[0].SystemIpAddresses = []string{"203.0.113.10", "203.0.113.11"}
				}
				testDesire := helper.NewTestElasticLoadBalancer(loadBalancerName)
				updatedELB := helper.NewTestElasticLoadBalancer(loadBalancerName)
				for i := range updatedELB {
					updatedELB[i].VIP = testIPAddress
					updatedELB[i].NetworkInterfaces[0].SystemIpAddresses = []string{"203.0.113.10", "203.0.113.11"}
				}
				testSecurityGroups := helper.NewTestEmptySecurityGroups()

				expectedStatus := &corev1.LoadBalancerStatus{
					Ingress: []corev1.LoadBalancerIngress{
						{
							IP: testIPAddress,
						},
					},
				}

				c := nifcloud.NewMockCloudAPIClient(ctrl)
				gomock.InOrder(
					c.EXPECT().
						DescribeElasticLoadBalancers(gomock.Any(), gomock.Eq(loadBalancerName)).
						Return(existedELB, nil).
						Times(1),
					c.EXPECT().
						DescribeElasticLoadBalancers(gomock.Any(), gomock.Eq(loadBalancerName)).
						Return(updatedELB, nil).
						Times(1),
				)
				c.EXPECT().
					DeleteElasticLoadBalancer(gomock.Any(), gomock.Eq(&existedELB[1])).
					Return(nil).
					Times(1)
				expectedInstanceIDs := lo.Map(updatedELB[0].BalancingTargets, func(instance nifcloud.Instance, _ int) string {
					return instance.InstanceID
				})
				c.EXPECT().
					DescribeSecurityGroupsByInstanceIDs(gomock.Any(), gomock.Eq(expectedInstanceIDs)).
					Return(testSecurityGroups, nil).
					Times(1)
				createdSecurityGroupRules := []nifcloud.SecurityGroupRule{
					{
						IpProtocol: existedELB[1].Protocol,
						FromPort:   existedELB[1].InstancePort,
						ToPort:     existedELB[1].InstancePort,
						InOut:      "IN",
						IpRanges:   []string{testIPAddress},
					},
					{
						IpProtocol: existedELB[1].Protocol,
						FromPort:   existedELB[1].InstancePort,
						ToPort:     existedELB[1].InstancePort,
						InOut:      "IN",
						IpRanges:   []string{existedELB[1].NetworkInterfaces[0].SystemIpAddresses[0]},
					},
					{
						IpProtocol: existedELB[1].Protocol,
						FromPort:   existedELB[1].InstancePort,
						ToPort:     existedELB[1].InstancePort,
						InOut:      "IN",
						IpRanges:   []string{existedELB[1].NetworkInterfaces[0].SystemIpAddresses[1]},
					},
				}
				callRevokeSecurityGroupIngressTime := 0
				c.EXPECT().
					RevokeSecurityGroupIngress(gomock.Any(), gomock.Eq(testSecurityGroups[0].GroupName), gomock.Any()).
					Do(func(_ context.Context, _ string, securityGroupRule *nifcloud.SecurityGroupRule) {
						Expect(*securityGroupRule).Should(Equal(createdSecurityGroupRules[callRevokeSecurityGroupIngressTime]))
						callRevokeSecurityGroupIngressTime += 1
					}).
					Return(nil).
					Times(3)
				c.EXPECT().
					WaitSecurityGroupApplied(gomock.Any(), gomock.Eq(testSecurityGroups[0].GroupName)).
					Return(nil).
					Times(3)

				cloud := &nifcloud.Cloud{}
				cloud.SetClint(c)
				cloud.SetRegion(region)

				status, err := nifcloud.ExportEnsureElasticLoadBalancer(cloud, ctx, loadBalancerName, testDesire)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(*status).Should(Equal(*expectedStatus))
			})
		})

		Context("update one port of the elastic load balancer", func() {
			It("update the port", func() {
				ctx := context.Background()
				testIPAddress := "203.0.113.1"
				existedELB := helper.NewTestElasticLoadBalancer(loadBalancerName)
				existedELB[0].VIP = testIPAddress
				existedELB[0].NetworkInterfaces[0].SystemIpAddresses = []string{"203.0.113.10", "203.0.113.11"}
				testDesire := helper.NewTestElasticLoadBalancer(loadBalancerName)
				testDesire[0].LoadBalancerPort = 8080
				updatedELB := helper.NewTestElasticLoadBalancer(loadBalancerName)
				updatedELB[0].LoadBalancerPort = 8080
				for i := range updatedELB {
					updatedELB[i].VIP = testIPAddress
					updatedELB[i].NetworkInterfaces[0].SystemIpAddresses = []string{"203.0.113.10", "203.0.113.11"}
				}
				testSecurityGroups := helper.NewTestEmptySecurityGroups()

				expectedStatus := &corev1.LoadBalancerStatus{
					Ingress: []corev1.LoadBalancerIngress{
						{
							IP: testIPAddress,
						},
					},
				}

				c := nifcloud.NewMockCloudAPIClient(ctrl)
				gomock.InOrder(
					c.EXPECT().
						DescribeElasticLoadBalancers(gomock.Any(), gomock.Eq(loadBalancerName)).
						Return(existedELB, nil).
						Times(1),
					c.EXPECT().
						DescribeElasticLoadBalancers(gomock.Any(), gomock.Eq(loadBalancerName)).
						Return(updatedELB, nil).
						Times(1),
				)
				c.EXPECT().
					RegisterPortWithElasticLoadBalancer(gomock.Any(), gomock.Eq(&testDesire[0])).
					Return(nil).
					Times(1)
				c.EXPECT().
					DeleteElasticLoadBalancer(gomock.Any(), gomock.Eq(&existedELB[0])).
					Return(nil).
					Times(1)
				expectedInstanceIDs := lo.Map(updatedELB[0].BalancingTargets, func(instance nifcloud.Instance, _ int) string {
					return instance.InstanceID
				})
				c.EXPECT().
					DescribeSecurityGroupsByInstanceIDs(gomock.Any(), gomock.Eq(expectedInstanceIDs)).
					Return(testSecurityGroups, nil).
					Times(2)
				createdSecurityGroupRules := []nifcloud.SecurityGroupRule{
					{
						IpProtocol: updatedELB[0].Protocol,
						FromPort:   updatedELB[0].InstancePort,
						ToPort:     updatedELB[0].InstancePort,
						InOut:      "IN",
						IpRanges:   []string{testIPAddress},
					},
					{
						IpProtocol: updatedELB[0].Protocol,
						FromPort:   updatedELB[0].InstancePort,
						ToPort:     updatedELB[0].InstancePort,
						InOut:      "IN",
						IpRanges:   []string{updatedELB[0].NetworkInterfaces[0].SystemIpAddresses[0]},
					},
					{
						IpProtocol: updatedELB[0].Protocol,
						FromPort:   updatedELB[0].InstancePort,
						ToPort:     updatedELB[0].InstancePort,
						InOut:      "IN",
						IpRanges:   []string{updatedELB[0].NetworkInterfaces[0].SystemIpAddresses[1]},
					},
				}
				callAuthorizeSecurityGroupIngressTime := 0
				c.EXPECT().
					AuthorizeSecurityGroupIngress(gomock.Any(), gomock.Eq(testSecurityGroups[0].GroupName), gomock.Any()).
					Do(func(_ context.Context, _ string, securityGroupRule *nifcloud.SecurityGroupRule) {
						Expect(*securityGroupRule).Should(Equal(createdSecurityGroupRules[callAuthorizeSecurityGroupIngressTime]))
						callAuthorizeSecurityGroupIngressTime += 1
					}).
					Return(nil).
					Times(3)
				deletedSecurityGroupRules := []nifcloud.SecurityGroupRule{
					{
						IpProtocol: existedELB[0].Protocol,
						FromPort:   existedELB[0].InstancePort,
						ToPort:     existedELB[0].InstancePort,
						InOut:      "IN",
						IpRanges:   []string{testIPAddress},
					},
					{
						IpProtocol: existedELB[0].Protocol,
						FromPort:   existedELB[0].InstancePort,
						ToPort:     existedELB[0].InstancePort,
						InOut:      "IN",
						IpRanges:   []string{existedELB[0].NetworkInterfaces[0].SystemIpAddresses[0]},
					},
					{
						IpProtocol: existedELB[0].Protocol,
						FromPort:   existedELB[0].InstancePort,
						ToPort:     existedELB[0].InstancePort,
						InOut:      "IN",
						IpRanges:   []string{existedELB[0].NetworkInterfaces[0].SystemIpAddresses[1]},
					},
				}
				callRevokeSecurityGroupIngressTime := 0
				c.EXPECT().
					RevokeSecurityGroupIngress(gomock.Any(), gomock.Eq(testSecurityGroups[0].GroupName), gomock.Any()).
					Do(func(_ context.Context, _ string, securityGroupRule *nifcloud.SecurityGroupRule) {
						Expect(*securityGroupRule).Should(Equal(deletedSecurityGroupRules[callRevokeSecurityGroupIngressTime]))
						callRevokeSecurityGroupIngressTime += 1
					}).
					Return(nil).
					Times(3)
				c.EXPECT().
					WaitSecurityGroupApplied(gomock.Any(), gomock.Eq(testSecurityGroups[0].GroupName)).
					Return(nil).
					Times(6)

				cloud := &nifcloud.Cloud{}
				cloud.SetClint(c)
				cloud.SetRegion(region)

				status, err := nifcloud.ExportEnsureElasticLoadBalancer(cloud, ctx, loadBalancerName, testDesire)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(*status).Should(Equal(*expectedStatus))
			})
		})

		Context("register an instance to the elastic load balancer", func() {
			It("register the instance", func() {
				ctx := context.Background()
				testIPAddress := "192.168.0.1"
				registeredInstance := helper.NewTestInstance()
				registeredInstance.InstanceID = "testinstance2"
				registeredInstance.InstanceUniqueID = "i-xyzw5678"
				registeredInstance.PublicIPAddress = "203.0.113.1"
				registeredInstance.PrivateIPAddress = "192.168.0.101"
				existedELB := helper.NewTestElasticLoadBalancer(loadBalancerName)
				existedELB[0].VIP = testIPAddress
				existedELB[0].NetworkInterfaces[0].SystemIpAddresses = []string{"192.168.0.10", "192.168.0.11"}
				testDesire := helper.NewTestElasticLoadBalancer(loadBalancerName)
				testDesire[0].BalancingTargets = append(testDesire[0].BalancingTargets, *registeredInstance)
				updatedELB := helper.NewTestElasticLoadBalancer(loadBalancerName)
				for i := range updatedELB {
					updatedELB[i].VIP = testIPAddress
					updatedELB[i].NetworkInterfaces[0].SystemIpAddresses = []string{"192.168.0.10", "192.168.0.11"}
					updatedELB[i].BalancingTargets = append(updatedELB[i].BalancingTargets, *registeredInstance)
				}
				testSecurityGroups := helper.NewTestEmptySecurityGroups()

				expectedStatus := &corev1.LoadBalancerStatus{
					Ingress: []corev1.LoadBalancerIngress{
						{
							IP: testIPAddress,
						},
					},
				}

				c := nifcloud.NewMockCloudAPIClient(ctrl)
				c.EXPECT().
					DescribeElasticLoadBalancers(gomock.Any(), gomock.Eq(loadBalancerName)).
					Return(existedELB, nil).
					Times(1)
				c.EXPECT().
					RegisterInstancesWithElasticLoadBalancer(gomock.Any(), gomock.Eq(&existedELB[0]), gomock.Eq([]nifcloud.Instance{*registeredInstance})).
					Return(nil).
					Times(1)
				expectedInstanceIDs := []string{registeredInstance.InstanceID}
				c.EXPECT().
					DescribeSecurityGroupsByInstanceIDs(gomock.Any(), gomock.Eq(expectedInstanceIDs)).
					Return(testSecurityGroups, nil).
					Times(1)
				createdSecurityGroupRules := []nifcloud.SecurityGroupRule{
					{
						IpProtocol: updatedELB[0].Protocol,
						FromPort:   updatedELB[0].InstancePort,
						ToPort:     updatedELB[0].InstancePort,
						InOut:      "IN",
						IpRanges:   []string{testIPAddress},
					},
					{
						IpProtocol: updatedELB[0].Protocol,
						FromPort:   updatedELB[0].InstancePort,
						ToPort:     updatedELB[0].InstancePort,
						InOut:      "IN",
						IpRanges:   []string{updatedELB[0].NetworkInterfaces[0].SystemIpAddresses[0]},
					},
					{
						IpProtocol: updatedELB[0].Protocol,
						FromPort:   updatedELB[0].InstancePort,
						ToPort:     updatedELB[0].InstancePort,
						InOut:      "IN",
						IpRanges:   []string{updatedELB[0].NetworkInterfaces[0].SystemIpAddresses[1]},
					},
				}
				callAuthorizeSecurityGroupIngressTime := 0
				c.EXPECT().
					AuthorizeSecurityGroupIngress(gomock.Any(), gomock.Eq(testSecurityGroups[0].GroupName), gomock.Any()).
					Do(func(_ context.Context, _ string, securityGroupRule *nifcloud.SecurityGroupRule) {
						Expect(*securityGroupRule).Should(Equal(createdSecurityGroupRules[callAuthorizeSecurityGroupIngressTime]))
						callAuthorizeSecurityGroupIngressTime += 1
					}).
					Return(nil).
					Times(3)
				c.EXPECT().
					WaitSecurityGroupApplied(gomock.Any(), gomock.Eq(testSecurityGroups[0].GroupName)).
					Return(nil).
					Times(3)

				cloud := &nifcloud.Cloud{}
				cloud.SetClint(c)
				cloud.SetRegion(region)

				status, err := nifcloud.ExportEnsureElasticLoadBalancer(cloud, ctx, loadBalancerName, testDesire)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(*status).Should(Equal(*expectedStatus))
			})
		})

		Context("deregister an instance to the elastic load balancer", func() {
			It("deregister the instance", func() {
				ctx := context.Background()
				testIPAddress := "192.168.0.1"
				deregisteredInstance := helper.NewTestInstance()
				deregisteredInstance.InstanceID = "testinstance2"
				deregisteredInstance.InstanceUniqueID = "i-xyzw5678"
				deregisteredInstance.PublicIPAddress = "203.0.113.1"
				deregisteredInstance.PrivateIPAddress = "192.168.0.101"
				existedELB := helper.NewTestElasticLoadBalancer(loadBalancerName)
				existedELB[0].VIP = testIPAddress
				existedELB[0].NetworkInterfaces[0].SystemIpAddresses = []string{"192.168.0.10", "192.168.0.11"}
				existedELB[0].BalancingTargets = append(existedELB[0].BalancingTargets, *deregisteredInstance)
				testDesire := helper.NewTestElasticLoadBalancer(loadBalancerName)
				updateELB := helper.NewTestElasticLoadBalancer(loadBalancerName)
				for i := range updateELB {
					updateELB[i].VIP = testIPAddress
					updateELB[i].NetworkInterfaces[0].SystemIpAddresses = []string{"192.168.0.10", "192.168.0.11"}
				}
				testSecurityGroups := helper.NewTestEmptySecurityGroups()

				expectedStatus := &corev1.LoadBalancerStatus{
					Ingress: []corev1.LoadBalancerIngress{
						{
							IP: testIPAddress,
						},
					},
				}

				c := nifcloud.NewMockCloudAPIClient(ctrl)
				c.EXPECT().
					DescribeElasticLoadBalancers(gomock.Any(), gomock.Eq(loadBalancerName)).
					Return(existedELB, nil).
					Times(1)
				c.EXPECT().
					DeregisterInstancesFromElasticLoadBalancer(gomock.Any(), gomock.Eq(&existedELB[0]), gomock.Eq([]nifcloud.Instance{*deregisteredInstance})).
					Return(nil).
					Times(1)
				expectedInstanceIDs := []string{deregisteredInstance.InstanceID}
				c.EXPECT().
					DescribeSecurityGroupsByInstanceIDs(gomock.Any(), gomock.Eq(expectedInstanceIDs)).
					Return(testSecurityGroups, nil).
					Times(1)
				deletedSecurityGroupRules := []nifcloud.SecurityGroupRule{
					{
						IpProtocol: updateELB[0].Protocol,
						FromPort:   updateELB[0].InstancePort,
						ToPort:     updateELB[0].InstancePort,
						InOut:      "IN",
						IpRanges:   []string{testIPAddress},
					},
					{
						IpProtocol: updateELB[0].Protocol,
						FromPort:   updateELB[0].InstancePort,
						ToPort:     updateELB[0].InstancePort,
						InOut:      "IN",
						IpRanges:   []string{updateELB[0].NetworkInterfaces[0].SystemIpAddresses[0]},
					},
					{
						IpProtocol: updateELB[0].Protocol,
						FromPort:   updateELB[0].InstancePort,
						ToPort:     updateELB[0].InstancePort,
						InOut:      "IN",
						IpRanges:   []string{updateELB[0].NetworkInterfaces[0].SystemIpAddresses[1]},
					},
				}
				callRevokeSecurityGroupIngressTime := 0
				c.EXPECT().
					RevokeSecurityGroupIngress(gomock.Any(), gomock.Eq(testSecurityGroups[0].GroupName), gomock.Any()).
					Do(func(_ context.Context, _ string, securityGroupRule *nifcloud.SecurityGroupRule) {
						Expect(*securityGroupRule).Should(Equal(deletedSecurityGroupRules[callRevokeSecurityGroupIngressTime]))
						callRevokeSecurityGroupIngressTime += 1
					}).
					Return(nil).
					Times(3)
				c.EXPECT().
					WaitSecurityGroupApplied(gomock.Any(), gomock.Eq(testSecurityGroups[0].GroupName)).
					Return(nil).
					Times(3)

				cloud := &nifcloud.Cloud{}
				cloud.SetClint(c)
				cloud.SetRegion(region)

				status, err := nifcloud.ExportEnsureElasticLoadBalancer(cloud, ctx, loadBalancerName, testDesire)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(*status).Should(Equal(*expectedStatus))
			})
		})
	})
})

var _ = Describe("NewElasticLoadBalancerFromService", func() {
	var loadBalancerName string
	var testService corev1.Service

	BeforeEach(func() {
		loadBalancerName = "testloadbalancer"
		testService = corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name: "testlbsvc",
				Annotations: map[string]string{
					nifcloud.ServiceAnnotationLoadBalancerBalancingType:        "1",
					nifcloud.ServiceAnnotationLoadBalancerAccountingType:       "1",
					nifcloud.ServiceAnnotationLoadBalancerNetworkVolume:        "100",
					nifcloud.ServiceAnnotationLoadBalancerHCInterval:           "10",
					nifcloud.ServiceAnnotationLoadBalancerHCUnhealthyThreshold: "1",
					nifcloud.ServiceAnnotationLoadBalancerHCProtocol:           "TCP",
					nifcloud.ServiceAnnotationLoadBalancerNetworkInterface1:    "net-COMMON_GLOBAL",
					nifcloud.ServiceAnnotationLoadBalancerVipNetwork:           "1",
				},
			},
			Spec: corev1.ServiceSpec{
				Ports: []corev1.ServicePort{
					{
						Port:     80,
						NodePort: 30000,
						Protocol: corev1.ProtocolTCP,
					},
				},
			},
		}
	})

	Context("given valid elastic load balancer", func() {
		It("return the elastic load balancer", func() {
			testInstances := []nifcloud.Instance{*helper.NewTestInstance()}
			expectELB := helper.NewTestElasticLoadBalancer(loadBalancerName)
			gotELB, err := nifcloud.NewElasticLoadBalancerFromService(loadBalancerName, testInstances, &testService)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(gotELB).Should(Equal(expectELB))
		})
	})

	Context("given elastic load balancer that has two ports", func() {
		It("return the elastic load balancer", func() {
			testService.Spec.Ports = append(testService.Spec.Ports, corev1.ServicePort{
				Port:     443,
				NodePort: 30001,
				Protocol: corev1.ProtocolTCP,
			},
			)
			testInstances := []nifcloud.Instance{*helper.NewTestInstance()}
			expectELB := helper.NewTestElasticLoadBalancerWithTwoPort(loadBalancerName)
			gotELB, err := nifcloud.NewElasticLoadBalancerFromService(loadBalancerName, testInstances, &testService)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(gotELB).Should(Equal(expectELB))
		})
	})

	Context("given elastic load balancer that health check protocol is ICMP", func() {
		It("return the elastic load balancer", func() {
			testService.Annotations[nifcloud.ServiceAnnotationLoadBalancerHCProtocol] = "ICMP"
			testInstances := []nifcloud.Instance{*helper.NewTestInstance()}
			expectELB := helper.NewTestElasticLoadBalancer(loadBalancerName)
			expectELB[0].HealthCheckTarget = "ICMP"
			gotELB, err := nifcloud.NewElasticLoadBalancerFromService(loadBalancerName, testInstances, &testService)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(gotELB).Should(Equal(expectELB))
		})
	})

	Context("given elastic load balancer that has two network interfaces", func() {
		It("return the elastic load balancer", func() {
			testService.Annotations[nifcloud.ServiceAnnotationLoadBalancerNetworkInterface2] = "net-COMMON_PRIVATE"
			testInstances := []nifcloud.Instance{*helper.NewTestInstance()}
			expectELB := helper.NewTestElasticLoadBalancer(loadBalancerName)
			expectELB[0].NetworkInterfaces = append(expectELB[0].NetworkInterfaces, nifcloud.NetworkInterface{
				NetworkId: "net-COMMON_PRIVATE",
			})
			gotELB, err := nifcloud.NewElasticLoadBalancerFromService(loadBalancerName, testInstances, &testService)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(gotELB).Should(Equal(expectELB))
		})
	})

	Context("given elastic load balancer that connects private network", func() {
		It("return the elastic load balancer", func() {
			testService.Annotations[nifcloud.ServiceAnnotationLoadBalancerNetworkInterface1] = "net-abcd1234"
			testService.Annotations[nifcloud.ServiceAnnotationLoadBalancerNetworkInterface1IPAddress] = "192.168.0.10"
			testService.Annotations[nifcloud.ServiceAnnotationLoadBalancerNetworkInterface1SystemIPAddresses] = "192.168.0.11,192.168.0.12"
			testService.Annotations[nifcloud.ServiceAnnotationLoadBalancerNetworkInterface2] = "net-xyzw5678"
			testService.Annotations[nifcloud.ServiceAnnotationLoadBalancerNetworkInterface2IPAddress] = "192.168.1.10"
			testService.Annotations[nifcloud.ServiceAnnotationLoadBalancerNetworkInterface2SystemIPAddresses] = "192.168.1.11,192.168.1.12"

			testInstances := []nifcloud.Instance{*helper.NewTestInstance()}
			expectELB := helper.NewTestElasticLoadBalancer(loadBalancerName)
			expectELB[0].NetworkInterfaces = []nifcloud.NetworkInterface{
				{
					NetworkId:         "net-abcd1234",
					IPAddress:         "192.168.0.10",
					SystemIpAddresses: []string{"192.168.0.11", "192.168.0.12"},
					IsVipNetwork:      true,
				},
				{
					NetworkId:         "net-xyzw5678",
					IPAddress:         "192.168.1.10",
					SystemIpAddresses: []string{"192.168.1.11", "192.168.1.12"},
				},
			}

			gotELB, err := nifcloud.NewElasticLoadBalancerFromService(loadBalancerName, testInstances, &testService)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(gotELB).Should(Equal(expectELB))
		})
	})
})

var _ = Describe("securityGroupRulesOfElasticLoadBalancer", func() {
	var loadBalancerName string

	BeforeEach(func() {
		loadBalancerName = "testloadbalancer"
	})

	Context("given elastic load balancer has one network interface", func() {
		Context("the health check protocol is ICMP", func() {
			It("returns security group rules", func() {
				ctx := context.Background()

				testELB := &helper.NewTestElasticLoadBalancer(loadBalancerName)[0]
				testELB.VIP = "203.0.113.1"
				testELB.HealthCheckTarget = "ICMP"
				testELB.NetworkInterfaces[0].SystemIpAddresses = []string{"203.0.113.10", "203.0.113.11"}

				wantSecurityGroupRules := []nifcloud.SecurityGroupRule{
					{
						IpProtocol: testELB.Protocol,
						FromPort:   testELB.InstancePort,
						ToPort:     testELB.InstancePort,
						InOut:      "IN",
						IpRanges:   []string{testELB.VIP},
					},
					{
						IpProtocol: "ICMP",
						InOut:      "IN",
						IpRanges:   []string{testELB.VIP},
					},
					{
						IpProtocol: "ICMP",
						InOut:      "IN",
						IpRanges:   []string{testELB.NetworkInterfaces[0].SystemIpAddresses[0]},
					},
					{
						IpProtocol: "ICMP",
						InOut:      "IN",
						IpRanges:   []string{testELB.NetworkInterfaces[0].SystemIpAddresses[1]},
					},
				}

				gotSecurityGroupRules, err := nifcloud.ExportSecurityGroupRulesOfElasticLoadBalancer(ctx, testELB)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(gotSecurityGroupRules).Should(Equal(wantSecurityGroupRules))
			})
		})
		Context("the health check protocol is not ICMP", func() {
			It("returns security group rules", func() {
				ctx := context.Background()

				testELB := &helper.NewTestElasticLoadBalancer(loadBalancerName)[0]
				testELB.VIP = "203.0.113.1"
				testELB.NetworkInterfaces[0].SystemIpAddresses = []string{"203.0.113.10", "203.0.113.11"}

				wantSecurityGroupRules := []nifcloud.SecurityGroupRule{
					{
						IpProtocol: testELB.Protocol,
						FromPort:   testELB.InstancePort,
						ToPort:     testELB.InstancePort,
						InOut:      "IN",
						IpRanges:   []string{testELB.VIP},
					},
					{
						IpProtocol: testELB.Protocol,
						FromPort:   testELB.InstancePort,
						ToPort:     testELB.InstancePort,
						InOut:      "IN",
						IpRanges:   []string{testELB.NetworkInterfaces[0].SystemIpAddresses[0]},
					},
					{
						IpProtocol: testELB.Protocol,
						FromPort:   testELB.InstancePort,
						ToPort:     testELB.InstancePort,
						InOut:      "IN",
						IpRanges:   []string{testELB.NetworkInterfaces[0].SystemIpAddresses[1]},
					},
				}

				gotSecurityGroupRules, err := nifcloud.ExportSecurityGroupRulesOfElasticLoadBalancer(ctx, testELB)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(gotSecurityGroupRules).Should(Equal(wantSecurityGroupRules))
			})
		})
	})

	Context("given elastic load balancer has two network interface", func() {
		Context("the health check protocol is ICMP", func() {
			It("returns security group rules", func() {
				ctx := context.Background()

				testELB := &helper.NewTestElasticLoadBalancer(loadBalancerName)[0]
				testELB.VIP = "198.168.0.1"
				testELB.HealthCheckTarget = "ICMP"
				testELB.NetworkInterfaces = []nifcloud.NetworkInterface{
					{
						NetworkId:         "net-abcd1234",
						IPAddress:         "192.168.0.10",
						SystemIpAddresses: []string{"192.168.0.11", "192.168.0.12"},
						IsVipNetwork:      true,
					},
					{
						NetworkId:         "net-xyzw5678",
						IPAddress:         "192.168.1.10",
						SystemIpAddresses: []string{"192.168.1.11", "192.168.1.12"},
					},
				}
				wantSecurityGroupRules := []nifcloud.SecurityGroupRule{
					{
						IpProtocol: "ICMP",
						InOut:      "IN",
						IpRanges:   []string{testELB.NetworkInterfaces[1].IPAddress},
					},
					{
						IpProtocol: "ICMP",
						InOut:      "IN",
						IpRanges:   []string{testELB.NetworkInterfaces[1].SystemIpAddresses[0]},
					},
					{
						IpProtocol: "ICMP",
						InOut:      "IN",
						IpRanges:   []string{testELB.NetworkInterfaces[1].SystemIpAddresses[1]},
					},
				}

				gotSecurityGroupRules, err := nifcloud.ExportSecurityGroupRulesOfElasticLoadBalancer(ctx, testELB)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(gotSecurityGroupRules).Should(Equal(wantSecurityGroupRules))
			})
		})
		Context("the health check protocol is not ICMP", func() {
			It("returns security group rules", func() {
				ctx := context.Background()

				testELB := &helper.NewTestElasticLoadBalancer(loadBalancerName)[0]
				testELB.VIP = "198.168.0.1"
				testELB.NetworkInterfaces = []nifcloud.NetworkInterface{
					{
						NetworkId:         "net-abcd1234",
						IPAddress:         "192.168.0.10",
						SystemIpAddresses: []string{"192.168.0.11", "192.168.0.12"},
						IsVipNetwork:      true,
					},
					{
						NetworkId:         "net-xyzw5678",
						IPAddress:         "192.168.1.10",
						SystemIpAddresses: []string{"192.168.1.11", "192.168.1.12"},
					},
				}
				wantSecurityGroupRules := []nifcloud.SecurityGroupRule{
					{
						IpProtocol: testELB.Protocol,
						FromPort:   testELB.InstancePort,
						ToPort:     testELB.InstancePort,
						InOut:      "IN",
						IpRanges:   []string{testELB.NetworkInterfaces[1].IPAddress},
					},
					{
						IpProtocol: testELB.Protocol,
						FromPort:   testELB.InstancePort,
						ToPort:     testELB.InstancePort,
						InOut:      "IN",
						IpRanges:   []string{testELB.NetworkInterfaces[1].SystemIpAddresses[0]},
					},
					{
						IpProtocol: testELB.Protocol,
						FromPort:   testELB.InstancePort,
						ToPort:     testELB.InstancePort,
						InOut:      "IN",
						IpRanges:   []string{testELB.NetworkInterfaces[1].SystemIpAddresses[1]},
					},
				}

				gotSecurityGroupRules, err := nifcloud.ExportSecurityGroupRulesOfElasticLoadBalancer(ctx, testELB)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(gotSecurityGroupRules).Should(Equal(wantSecurityGroupRules))
			})
		})
	})
})

var _ = Describe("updateElasticLoadBalancer", func() {
	var ctrl *gomock.Controller
	var region string = "east1"
	var clusterName string = "testCluster"
	var loadBalancerUID types.UID
	var loadBalancerName string
	var testService *corev1.Service

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		loadBalancerUID = types.UID(uuid.NewString())
		loadBalancerName = strings.Replace(string(loadBalancerUID), "-", "", -1)[:nifcloud.ExportMaxLoadBalancerNameLength]
		testService = &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name: "testlbsvc",
				UID:  loadBalancerUID,
			},
		}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("the elastic load balancer is existed", func() {
		It("return nil", func() {
			ctx := context.Background()

			testELB := helper.NewTestElasticLoadBalancer(loadBalancerName)

			c := nifcloud.NewMockCloudAPIClient(ctrl)
			c.EXPECT().
				DescribeElasticLoadBalancers(gomock.Any(), gomock.Eq(loadBalancerName)).
				Return(testELB, nil).
				Times(1)

			cloud := &nifcloud.Cloud{}
			cloud.SetClint(c)
			cloud.SetRegion(region)

			err := nifcloud.ExportUpdateElasticLoadBalancer(cloud, ctx, clusterName, testService)
			Expect(err).ShouldNot(HaveOccurred())
		})
	})

	Context("the elastic load balancer is not existed", func() {
		It("return nil", func() {
			ctx := context.Background()

			testELB := []nifcloud.ElasticLoadBalancer{}
			notFoundErr := helper.NewMockAPIError(nifcloud.ExportErrorCodeElasticLoadBalancerNotFound)

			c := nifcloud.NewMockCloudAPIClient(ctrl)
			c.EXPECT().
				DescribeElasticLoadBalancers(gomock.Any(), gomock.Eq(loadBalancerName)).
				Return(testELB, notFoundErr).
				Times(1)

			cloud := &nifcloud.Cloud{}
			cloud.SetClint(c)
			cloud.SetRegion(region)

			err := nifcloud.ExportUpdateElasticLoadBalancer(cloud, ctx, clusterName, testService)
			Expect(err).Should(HaveOccurred())
			Expect(err).Should(Equal(notFoundErr))
		})
	})
})

var _ = Describe("separateHealthCheckTarget", func() {
	DescribeTable("valid HealthCheckTarget",
		func(healthCheckTarget, expectProtocol, expectPort string) {
			gotProtocol, gotPort := nifcloud.ExportSeparateHealthCheckTarget(healthCheckTarget)
			Expect(gotProtocol).Should(Equal(expectProtocol))
			Expect(gotPort).Should(Equal(expectPort))
		},
		Entry("the protocol is ICMP", "ICMP", "ICMP", ""),
		Entry("the protocol is TCP", "TCP:8080", "TCP", "8080"),
		Entry("the protocol is HTTP", "HTTP:80", "HTTP", "80"),
		Entry("the protocol is HTTPS", "HTTPS:443", "HTTPS", "443"),
	)
})

var _ = Describe("ensureElasticLoadBalancerDeleted", func() {
	var ctrl *gomock.Controller
	var region string = "east1"
	var clusterName string = "testCluster"
	var loadBalancerUID types.UID
	var loadBalancerName string
	var testService *corev1.Service

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		loadBalancerUID = types.UID(uuid.NewString())
		loadBalancerName = strings.Replace(string(loadBalancerUID), "-", "", -1)[:nifcloud.ExportMaxLoadBalancerNameLength]
		testService = &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name: "testlbsvc",
				UID:  loadBalancerUID,
			},
		}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("the elastic load balancer is existed", func() {
		It("delete the elastic load balancer", func() {
			ctx := context.Background()

			testELB := helper.NewTestElasticLoadBalancer(loadBalancerName)
			testELB[0].VIP = "203.0.113.1"
			testELB[0].NetworkInterfaces[0].SystemIpAddresses = []string{"203.0.113.10", "203.0.113.11"}
			testSecurityGroups := helper.NewTestEmptySecurityGroups()

			c := nifcloud.NewMockCloudAPIClient(ctrl)
			c.EXPECT().
				DescribeElasticLoadBalancers(gomock.Any(), gomock.Eq(loadBalancerName)).
				Return(testELB, nil).
				Times(1)
			c.EXPECT().
				DeleteElasticLoadBalancer(gomock.Any(), gomock.Eq(&testELB[0])).
				Return(nil).
				Times(1)
			expectedInstanceIDs := lo.Map(testELB[0].BalancingTargets, func(instance nifcloud.Instance, _ int) string {
				return instance.InstanceID
			})
			c.EXPECT().
				DescribeSecurityGroupsByInstanceIDs(gomock.Any(), gomock.Eq(expectedInstanceIDs)).
				Return(testSecurityGroups, nil).
				Times(1)

			deletedSecurityGroupRules := []nifcloud.SecurityGroupRule{
				{
					IpProtocol: testELB[0].Protocol,
					FromPort:   testELB[0].InstancePort,
					ToPort:     testELB[0].InstancePort,
					InOut:      "IN",
					IpRanges:   []string{testELB[0].VIP},
				},
				{
					IpProtocol: testELB[0].Protocol,
					FromPort:   testELB[0].InstancePort,
					ToPort:     testELB[0].InstancePort,
					InOut:      "IN",
					IpRanges:   []string{testELB[0].NetworkInterfaces[0].SystemIpAddresses[0]},
				},
				{
					IpProtocol: testELB[0].Protocol,
					FromPort:   testELB[0].InstancePort,
					ToPort:     testELB[0].InstancePort,
					InOut:      "IN",
					IpRanges:   []string{testELB[0].NetworkInterfaces[0].SystemIpAddresses[1]},
				},
			}
			callRevokeSecurityGroupIngressTime := 0
			c.EXPECT().
				RevokeSecurityGroupIngress(gomock.Any(), gomock.Eq(testSecurityGroups[0].GroupName), gomock.Any()).
				Do(func(_ context.Context, _ string, securityGroupRule *nifcloud.SecurityGroupRule) {
					Expect(*securityGroupRule).Should(Equal(deletedSecurityGroupRules[callRevokeSecurityGroupIngressTime]))
					callRevokeSecurityGroupIngressTime += 1
				}).
				Return(nil).
				Times(3)
			c.EXPECT().
				WaitSecurityGroupApplied(gomock.Any(), gomock.Eq(testSecurityGroups[0].GroupName)).
				Return(nil).
				Times(3)

			cloud := &nifcloud.Cloud{}
			cloud.SetClint(c)
			cloud.SetRegion(region)

			err := nifcloud.ExportEnsureElasticLoadBalancerDeleted(cloud, ctx, clusterName, testService)
			Expect(err).ShouldNot(HaveOccurred())
		})
	})

	Context("the elastic load balancer is not existed", func() {
		It("return nil", func() {
			ctx := context.Background()

			testELB := []nifcloud.ElasticLoadBalancer{}
			notFoundErr := helper.NewMockAPIError(nifcloud.ExportErrorCodeElasticLoadBalancerNotFound)

			c := nifcloud.NewMockCloudAPIClient(ctrl)
			c.EXPECT().
				DescribeElasticLoadBalancers(gomock.Any(), gomock.Eq(loadBalancerName)).
				Return(testELB, notFoundErr).
				Times(1)

			cloud := &nifcloud.Cloud{}
			cloud.SetClint(c)
			cloud.SetRegion(region)

			err := nifcloud.ExportEnsureElasticLoadBalancerDeleted(cloud, ctx, clusterName, testService)
			Expect(err).ShouldNot(HaveOccurred())
		})
	})
})
