package nifcloud_test

import (
	"context"
	"strings"

	"github.com/nifcloud/nifcloud-cloud-controller-manager/pkg/cloudprovider/providers/nifcloud"
	"github.com/nifcloud/nifcloud-cloud-controller-manager/test/helper"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("isL4LoadBalancer", func() {
	Context("load balancer type is not defined", func() {
		It("returns true", func() {
			testAnnotations := map[string]string{}
			isL4LB := nifcloud.ExportIsL4LoadBalancer(testAnnotations)
			Expect(isL4LB).Should(BeTrue())
		})
	})

	Context("load balancer type is empty", func() {
		It("returns true", func() {
			testAnnotations := map[string]string{
				nifcloud.ServiceAnnotationLoadBalancerType: "",
			}
			isL4LB := nifcloud.ExportIsL4LoadBalancer(testAnnotations)
			Expect(isL4LB).Should(BeTrue())
		})
	})

	Context("load balancer type is lb", func() {
		It("returns true", func() {
			testAnnotations := map[string]string{
				nifcloud.ServiceAnnotationLoadBalancerType: "lb",
			}
			isL4LB := nifcloud.ExportIsL4LoadBalancer(testAnnotations)
			Expect(isL4LB).Should(BeTrue())
		})
	})

	Context("load balancer type is not lb", func() {
		It("returns false", func() {
			testAnnotations := map[string]string{
				nifcloud.ServiceAnnotationLoadBalancerType: "any",
			}
			isL4LB := nifcloud.ExportIsL4LoadBalancer(testAnnotations)
			Expect(isL4LB).Should(BeFalse())
		})
	})
})

var _ = Describe("getL4LoadBalancer", func() {
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

	Context("the specified l4 load balancer is existed", func() {
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
				DescribeLoadBalancers(gomock.Any(), gomock.Eq(loadBalancerName)).
				Return([]nifcloud.LoadBalancer{
					{
						Name: loadBalancerName,
						VIP:  testIPAddress,
					},
				}, nil).
				Times(1)

			cloud := &nifcloud.Cloud{}
			cloud.SetClient(c)
			cloud.SetRegion(region)

			status, exists, err := nifcloud.ExportGetL4LoadBalancer(cloud, ctx, clusterName, service)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(exists).Should(BeTrue())
			Expect(*status).Should(Equal(*expectedStatus))
		})
	})

	Context("the specified l4 load balancer is not existed", func() {
		It("return that exists is false", func() {
			ctx := context.Background()
			clusterName := "testCluster"
			service := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name: clusterName,
					UID:  loadBalancerUID,
				},
			}

			apiErr := helper.NewMockAPIError(nifcloud.ExportErrorCodeLoadBalancerNotFound)

			c := nifcloud.NewMockCloudAPIClient(ctrl)
			c.EXPECT().
				DescribeLoadBalancers(gomock.Any(), gomock.Eq(loadBalancerName)).
				Return([]nifcloud.LoadBalancer{}, apiErr).
				Times(1)

			cloud := &nifcloud.Cloud{}
			cloud.SetClient(c)
			cloud.SetRegion(region)

			status, exists, err := nifcloud.ExportGetL4LoadBalancer(cloud, ctx, clusterName, service)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(exists).Should(BeFalse())
			Expect(status).Should(BeNil())
		})
	})

	Context("DescribeLoadBalancers return unknown error code", func() {
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
				DescribeLoadBalancers(gomock.Any(), gomock.Eq(loadBalancerName)).
				Return([]nifcloud.LoadBalancer{}, apiErr).
				Times(1)

			cloud := &nifcloud.Cloud{}
			cloud.SetClient(c)
			cloud.SetRegion(region)

			status, exists, err := nifcloud.ExportGetL4LoadBalancer(cloud, ctx, clusterName, service)
			Expect(err).Should(HaveOccurred())
			Expect(exists).Should(BeFalse())
			Expect(status).Should(BeNil())
		})
	})
})

var _ = Describe("ensureL4LoadBalancer", func() {
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

	Context("the specified l4 load balancer is not existed", func() {
		Context("the l4 load balancer has one port", func() {
			It("create the l4 load balancer", func() {
				ctx := context.Background()
				testIPAddress := "203.0.113.1"
				testDesire := helper.NewTestL4LoadBalancer(loadBalancerName)

				expectedStatus := &corev1.LoadBalancerStatus{
					Ingress: []corev1.LoadBalancerIngress{
						{
							IP: testIPAddress,
						},
					},
				}

				c := nifcloud.NewMockCloudAPIClient(ctrl)
				notFoundErr := helper.NewMockAPIError(nifcloud.ExportErrorCodeLoadBalancerNotFound)
				c.EXPECT().
					DescribeLoadBalancers(gomock.Any(), gomock.Eq(loadBalancerName)).
					Return([]nifcloud.LoadBalancer{}, notFoundErr).
					Times(1)
				c.EXPECT().
					CreateLoadBalancer(gomock.Any(), gomock.Eq(&testDesire[0])).
					Return(testIPAddress, nil).
					Times(1)

				cloud := &nifcloud.Cloud{}
				cloud.SetClient(c)
				cloud.SetRegion(region)

				status, err := nifcloud.ExportEnsureL4LoadBalancer(cloud, ctx, loadBalancerName, testDesire)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(*status).Should(Equal(*expectedStatus))
			})
		})

		Context("the l4 load balancer has two ports", func() {
			It("create the l4 load balancer", func() {
				ctx := context.Background()
				testIPAddress := "203.0.113.1"
				testDesire := helper.NewTestL4LoadBalancerWithTwoPort(loadBalancerName)

				expectedStatus := &corev1.LoadBalancerStatus{
					Ingress: []corev1.LoadBalancerIngress{
						{
							IP: testIPAddress,
						},
					},
				}

				c := nifcloud.NewMockCloudAPIClient(ctrl)
				notFoundErr := helper.NewMockAPIError(nifcloud.ExportErrorCodeLoadBalancerNotFound)
				c.EXPECT().
					DescribeLoadBalancers(gomock.Any(), gomock.Eq(loadBalancerName)).
					Return([]nifcloud.LoadBalancer{}, notFoundErr).
					Times(1)
				c.EXPECT().
					CreateLoadBalancer(gomock.Any(), gomock.Eq(&testDesire[0])).
					Return(testIPAddress, nil).
					Times(1)
				c.EXPECT().
					RegisterPortWithLoadBalancer(gomock.Any(), gomock.Eq(&testDesire[1])).
					Return(nil).
					Times(1)

				cloud := &nifcloud.Cloud{}
				cloud.SetClient(c)
				cloud.SetRegion(region)

				status, err := nifcloud.ExportEnsureL4LoadBalancer(cloud, ctx, loadBalancerName, testDesire)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(*status).Should(Equal(*expectedStatus))
			})
		})
	})

	Context("the specified l4 load balancer is existed", func() {
		Context("add a port to the l4 load balancer", func() {
			It("create the l4 load balancer", func() {
				ctx := context.Background()
				testIPAddress := "203.0.113.1"
				existedLB := helper.NewTestL4LoadBalancer(loadBalancerName)
				existedLB[0].VIP = testIPAddress
				testDesire := helper.NewTestL4LoadBalancerWithTwoPort(loadBalancerName)
				updatedLB := helper.NewTestL4LoadBalancerWithTwoPort(loadBalancerName)
				for i := range updatedLB {
					updatedLB[i].VIP = testIPAddress
				}

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
						DescribeLoadBalancers(gomock.Any(), gomock.Eq(loadBalancerName)).
						Return(existedLB, nil).
						Times(1),
					c.EXPECT().
						DescribeLoadBalancers(gomock.Any(), gomock.Eq(loadBalancerName)).
						Return(updatedLB, nil).
						Times(1),
				)

				c.EXPECT().
					RegisterPortWithLoadBalancer(gomock.Any(), gomock.Eq(&testDesire[1])).
					Return(nil).
					Times(1)

				cloud := &nifcloud.Cloud{}
				cloud.SetClient(c)
				cloud.SetRegion(region)

				status, err := nifcloud.ExportEnsureL4LoadBalancer(cloud, ctx, loadBalancerName, testDesire)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(*status).Should(Equal(*expectedStatus))
			})
		})

		Context("delete one port from the l4 load balancer", func() {
			It("delete the port", func() {
				ctx := context.Background()
				testIPAddress := "203.0.113.1"
				existedLB := helper.NewTestL4LoadBalancerWithTwoPort(loadBalancerName)
				for i := range existedLB {
					existedLB[i].VIP = testIPAddress
				}
				testDesire := helper.NewTestL4LoadBalancer(loadBalancerName)
				updatedLB := helper.NewTestL4LoadBalancer(loadBalancerName)
				updatedLB[0].VIP = testIPAddress

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
						DescribeLoadBalancers(gomock.Any(), gomock.Eq(loadBalancerName)).
						Return(existedLB, nil).
						Times(1),
					c.EXPECT().
						DescribeLoadBalancers(gomock.Any(), gomock.Eq(loadBalancerName)).
						Return(updatedLB, nil).
						Times(1),
				)

				c.EXPECT().
					DeleteLoadBalancer(gomock.Any(), gomock.Eq(&existedLB[1])).
					Return(nil).
					Times(1)

				cloud := &nifcloud.Cloud{}
				cloud.SetClient(c)
				cloud.SetRegion(region)

				status, err := nifcloud.ExportEnsureL4LoadBalancer(cloud, ctx, loadBalancerName, testDesire)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(*status).Should(Equal(*expectedStatus))
			})
		})

		Context("update one port of the l4 load balancer", func() {
			It("update the port", func() {
				ctx := context.Background()
				testIPAddress := "203.0.113.1"
				existedLB := helper.NewTestL4LoadBalancer(loadBalancerName)
				existedLB[0].VIP = testIPAddress
				testDesire := helper.NewTestL4LoadBalancer(loadBalancerName)
				testDesire[0].LoadBalancerPort = 8080
				updatedLB := helper.NewTestL4LoadBalancer(loadBalancerName)
				updatedLB[0].VIP = testIPAddress
				updatedLB[0].LoadBalancerPort = 8080

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
						DescribeLoadBalancers(gomock.Any(), gomock.Eq(loadBalancerName)).
						Return(existedLB, nil).
						Times(1),
					c.EXPECT().
						DescribeLoadBalancers(gomock.Any(), gomock.Eq(loadBalancerName)).
						Return(updatedLB, nil).
						Times(1),
				)

				c.EXPECT().
					RegisterPortWithLoadBalancer(gomock.Any(), gomock.Eq(&testDesire[0])).
					Return(nil).
					Times(1)
				c.EXPECT().
					DeleteLoadBalancer(gomock.Any(), gomock.Eq(&existedLB[0])).
					Return(nil).
					Times(1)

				cloud := &nifcloud.Cloud{}
				cloud.SetClient(c)
				cloud.SetRegion(region)

				status, err := nifcloud.ExportEnsureL4LoadBalancer(cloud, ctx, loadBalancerName, testDesire)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(*status).Should(Equal(*expectedStatus))
			})
		})

		Context("register an instance to the l4 load balancer", func() {
			It("register the instance", func() {
				ctx := context.Background()
				testIPAddress := "203.0.113.1"
				registeredInstance := helper.NewTestInstance()
				registeredInstance.InstanceID = "testinstance2"
				registeredInstance.InstanceUniqueID = "i-xyzw5678"
				registeredInstance.PublicIPAddress = "203.0.113.1"
				registeredInstance.PrivateIPAddress = "192.168.0.101"
				existedLB := helper.NewTestL4LoadBalancer(loadBalancerName)
				existedLB[0].VIP = testIPAddress
				testDesire := helper.NewTestL4LoadBalancer(loadBalancerName)
				testDesire[0].BalancingTargets = append(testDesire[0].BalancingTargets, *registeredInstance)

				expectedStatus := &corev1.LoadBalancerStatus{
					Ingress: []corev1.LoadBalancerIngress{
						{
							IP: testIPAddress,
						},
					},
				}

				c := nifcloud.NewMockCloudAPIClient(ctrl)
				c.EXPECT().
					DescribeLoadBalancers(gomock.Any(), gomock.Eq(loadBalancerName)).
					Return(existedLB, nil).
					Times(1)

				c.EXPECT().
					RegisterInstancesWithLoadBalancer(gomock.Any(), gomock.Eq(&existedLB[0]), []nifcloud.Instance{*registeredInstance}).
					Return(nil).
					Times(1)

				cloud := &nifcloud.Cloud{}
				cloud.SetClient(c)
				cloud.SetRegion(region)

				status, err := nifcloud.ExportEnsureL4LoadBalancer(cloud, ctx, loadBalancerName, testDesire)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(*status).Should(Equal(*expectedStatus))
			})
		})

		Context("deregister an instance from the l4 load balancer", func() {
			It("deregister the instance", func() {
				ctx := context.Background()
				testIPAddress := "203.0.113.1"
				deregisteredInstance := helper.NewTestInstance()
				deregisteredInstance.InstanceID = "testinstance2"
				deregisteredInstance.InstanceUniqueID = "i-xyzw5678"
				deregisteredInstance.PublicIPAddress = "203.0.113.1"
				deregisteredInstance.PrivateIPAddress = "192.168.0.101"
				existedLB := helper.NewTestL4LoadBalancer(loadBalancerName)
				existedLB[0].VIP = testIPAddress
				existedLB[0].BalancingTargets = append(existedLB[0].BalancingTargets, *deregisteredInstance)
				testDesire := helper.NewTestL4LoadBalancer(loadBalancerName)

				expectedStatus := &corev1.LoadBalancerStatus{
					Ingress: []corev1.LoadBalancerIngress{
						{
							IP: testIPAddress,
						},
					},
				}

				c := nifcloud.NewMockCloudAPIClient(ctrl)
				c.EXPECT().
					DescribeLoadBalancers(gomock.Any(), gomock.Eq(loadBalancerName)).
					Return(existedLB, nil).
					Times(1)

				c.EXPECT().
					DeregisterInstancesFromLoadBalancer(gomock.Any(), gomock.Eq(&existedLB[0]), []nifcloud.Instance{*deregisteredInstance}).
					Return(nil).
					Times(1)

				cloud := &nifcloud.Cloud{}
				cloud.SetClient(c)
				cloud.SetRegion(region)

				status, err := nifcloud.ExportEnsureL4LoadBalancer(cloud, ctx, loadBalancerName, testDesire)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(*status).Should(Equal(*expectedStatus))
			})
		})

		Context("add a filter to the l4 load balancer", func() {
			It("add the filter", func() {
				ctx := context.Background()
				testIPAddress := "203.0.113.1"
				existedLB := helper.NewTestL4LoadBalancer(loadBalancerName)
				existedLB[0].VIP = testIPAddress
				testDesire := helper.NewTestL4LoadBalancer(loadBalancerName)
				testDesire[0].Filters = append(testDesire[0].Filters, "198.51.100.0/24")
				testFilters := []nifcloud.Filter{
					{
						AddOnFilter: true,
						IPAddress:   "198.51.100.0/24",
					},
				}

				expectedStatus := &corev1.LoadBalancerStatus{
					Ingress: []corev1.LoadBalancerIngress{
						{
							IP: testIPAddress,
						},
					},
				}

				c := nifcloud.NewMockCloudAPIClient(ctrl)
				c.EXPECT().
					DescribeLoadBalancers(gomock.Any(), gomock.Eq(loadBalancerName)).
					Return(existedLB, nil).
					Times(1)

				c.EXPECT().
					SetFilterForLoadBalancer(gomock.Any(), gomock.Eq(&existedLB[0]), testFilters).
					Return(nil).
					Times(1)

				cloud := &nifcloud.Cloud{}
				cloud.SetClient(c)
				cloud.SetRegion(region)

				status, err := nifcloud.ExportEnsureL4LoadBalancer(cloud, ctx, loadBalancerName, testDesire)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(*status).Should(Equal(*expectedStatus))
			})
		})

		Context("remove a filter from the l4 load balancer", func() {
			It("remove the filter", func() {
				ctx := context.Background()
				testIPAddress := "203.0.113.1"
				existedLB := helper.NewTestL4LoadBalancer(loadBalancerName)
				existedLB[0].VIP = testIPAddress
				existedLB[0].Filters = append(existedLB[0].Filters, "198.51.100.0/24")
				testDesire := helper.NewTestL4LoadBalancer(loadBalancerName)
				testFilters := []nifcloud.Filter{
					{
						AddOnFilter: false,
						IPAddress:   "198.51.100.0/24",
					},
				}

				expectedStatus := &corev1.LoadBalancerStatus{
					Ingress: []corev1.LoadBalancerIngress{
						{
							IP: testIPAddress,
						},
					},
				}

				c := nifcloud.NewMockCloudAPIClient(ctrl)
				c.EXPECT().
					DescribeLoadBalancers(gomock.Any(), gomock.Eq(loadBalancerName)).
					Return(existedLB, nil).
					Times(1)

				c.EXPECT().
					SetFilterForLoadBalancer(gomock.Any(), gomock.Eq(&existedLB[0]), testFilters).
					Return(nil).
					Times(1)

				cloud := &nifcloud.Cloud{}
				cloud.SetClient(c)
				cloud.SetRegion(region)

				status, err := nifcloud.ExportEnsureL4LoadBalancer(cloud, ctx, loadBalancerName, testDesire)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(*status).Should(Equal(*expectedStatus))
			})
		})

		Context("update a filter of the l4 load balancer", func() {
			It("update the filter", func() {
				ctx := context.Background()
				testIPAddress := "203.0.113.1"
				existedLB := helper.NewTestL4LoadBalancer(loadBalancerName)
				existedLB[0].VIP = testIPAddress
				existedLB[0].Filters = append(existedLB[0].Filters, "198.51.100.0/24")
				testDesire := helper.NewTestL4LoadBalancer(loadBalancerName)
				testDesire[0].Filters = append(testDesire[0].Filters, "192.0.2.0/24")

				testFilters := []nifcloud.Filter{
					{
						AddOnFilter: true,
						IPAddress:   "192.0.2.0/24",
					},
					{
						AddOnFilter: false,
						IPAddress:   "198.51.100.0/24",
					},
				}

				expectedStatus := &corev1.LoadBalancerStatus{
					Ingress: []corev1.LoadBalancerIngress{
						{
							IP: testIPAddress,
						},
					},
				}

				c := nifcloud.NewMockCloudAPIClient(ctrl)
				c.EXPECT().
					DescribeLoadBalancers(gomock.Any(), gomock.Eq(loadBalancerName)).
					Return(existedLB, nil).
					Times(1)

				c.EXPECT().
					SetFilterForLoadBalancer(gomock.Any(), gomock.Eq(&existedLB[0]), testFilters).
					Return(nil).
					Times(1)

				cloud := &nifcloud.Cloud{}
				cloud.SetClient(c)
				cloud.SetRegion(region)

				status, err := nifcloud.ExportEnsureL4LoadBalancer(cloud, ctx, loadBalancerName, testDesire)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(*status).Should(Equal(*expectedStatus))
			})
		})
	})
})

var _ = Describe("NewL4LoadBalancerFromService", func() {
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
					nifcloud.ServiceAnnotationLoadBalancerPolicyType:           "standard",
					nifcloud.ServiceAnnotationLoadBalancerHCInterval:           "10",
					nifcloud.ServiceAnnotationLoadBalancerHCUnhealthyThreshold: "1",
					nifcloud.ServiceAnnotationLoadBalancerHCProtocol:           "TCP",
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

	Context("given valid l4 load balancer", func() {
		It("return the l4 load balancer", func() {
			testInstances := []nifcloud.Instance{*helper.NewTestInstance()}
			expectLB := helper.NewTestL4LoadBalancer(loadBalancerName)
			gotLB, err := nifcloud.NewL4LoadBalancerFromService(loadBalancerName, testInstances, &testService)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(gotLB).Should(Equal(expectLB))
		})
	})

	Context("given l4 load balancer that has two ports", func() {
		It("return the l4 load balancer", func() {
			testService.Spec.Ports = append(testService.Spec.Ports, corev1.ServicePort{
				Port:     443,
				NodePort: 30001,
				Protocol: corev1.ProtocolTCP,
			},
			)
			testInstances := []nifcloud.Instance{*helper.NewTestInstance()}
			expectLB := helper.NewTestL4LoadBalancerWithTwoPort(loadBalancerName)
			gotLB, err := nifcloud.NewL4LoadBalancerFromService(loadBalancerName, testInstances, &testService)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(gotLB).Should(Equal(expectLB))
		})
	})

	Context("given l4 load balancer that health check protocol is ICMP", func() {
		It("return the l4 load balancer", func() {
			testService.Annotations[nifcloud.ServiceAnnotationLoadBalancerHCProtocol] = "ICMP"
			testInstances := []nifcloud.Instance{*helper.NewTestInstance()}
			expectLB := helper.NewTestL4LoadBalancer(loadBalancerName)
			expectLB[0].HealthCheckTarget = "ICMP"
			gotLB, err := nifcloud.NewL4LoadBalancerFromService(loadBalancerName, testInstances, &testService)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(gotLB).Should(Equal(expectLB))
		})
	})

	Context("given l4 load balancer that has a filter", func() {
		It("return the l4 load balancer", func() {
			testService.Spec.LoadBalancerSourceRanges = append(testService.Spec.LoadBalancerSourceRanges, "192.0.2.0/24")
			testInstances := []nifcloud.Instance{*helper.NewTestInstance()}
			expectLB := helper.NewTestL4LoadBalancer(loadBalancerName)
			expectLB[0].Filters = append(expectLB[0].Filters, "192.0.2.0/24")
			gotLB, err := nifcloud.NewL4LoadBalancerFromService(loadBalancerName, testInstances, &testService)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(gotLB).Should(Equal(expectLB))
		})
	})

	Context("given l4 load balancer that has a filter (/32)", func() {
		It("return the l4 load balancer", func() {
			testService.Spec.LoadBalancerSourceRanges = append(testService.Spec.LoadBalancerSourceRanges, "192.0.2.0/32")
			testInstances := []nifcloud.Instance{*helper.NewTestInstance()}
			expectLB := helper.NewTestL4LoadBalancer(loadBalancerName)
			expectLB[0].Filters = append(expectLB[0].Filters, "192.0.2.0")
			gotLB, err := nifcloud.NewL4LoadBalancerFromService(loadBalancerName, testInstances, &testService)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(gotLB).Should(Equal(expectLB))
		})
	})
})
