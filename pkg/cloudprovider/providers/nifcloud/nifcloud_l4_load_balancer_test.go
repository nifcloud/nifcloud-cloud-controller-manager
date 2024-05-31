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

var _ = Describe("updateL4LoadBalancer", func() {
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

	Context("the l4 load balancer is existed", func() {
		It("return nil", func() {
			ctx := context.Background()

			testLB := helper.NewTestL4LoadBalancer(loadBalancerName)

			c := nifcloud.NewMockCloudAPIClient(ctrl)
			c.EXPECT().
				DescribeLoadBalancers(gomock.Any(), gomock.Eq(loadBalancerName)).
				Return(testLB, nil).
				Times(1)

			cloud := &nifcloud.Cloud{}
			cloud.SetClient(c)
			cloud.SetRegion(region)

			err := nifcloud.ExportUpdateL4LoadBalancer(cloud, ctx, clusterName, testService)
			Expect(err).ShouldNot(HaveOccurred())
		})
	})

	Context("the l4 load balancer is not existed", func() {
		It("return nil", func() {
			ctx := context.Background()

			testLB := []nifcloud.LoadBalancer{}
			notFoundErr := helper.NewMockAPIError(nifcloud.ExportErrorCodeLoadBalancerNotFound)

			c := nifcloud.NewMockCloudAPIClient(ctrl)
			c.EXPECT().
				DescribeLoadBalancers(gomock.Any(), gomock.Eq(loadBalancerName)).
				Return(testLB, notFoundErr).
				Times(1)

			cloud := &nifcloud.Cloud{}
			cloud.SetClient(c)
			cloud.SetRegion(region)

			err := nifcloud.ExportUpdateL4LoadBalancer(cloud, ctx, clusterName, testService)
			Expect(err).Should(HaveOccurred())
			Expect(err).Should(Equal(notFoundErr))
		})
	})
})

var _ = Describe("ensureL4LoadBalancerDeleted", func() {
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

	Context("the l4 load balancer is existed", func() {
		It("delete the l4 load balancer", func() {
			ctx := context.Background()

			testLB := helper.NewTestL4LoadBalancer(loadBalancerName)

			c := nifcloud.NewMockCloudAPIClient(ctrl)
			c.EXPECT().
				DescribeLoadBalancers(gomock.Any(), gomock.Eq(loadBalancerName)).
				Return(testLB, nil).
				Times(1)
			c.EXPECT().
				DeleteLoadBalancer(gomock.Any(), &testLB[0]).
				Return(nil).
				Times(1)

			cloud := &nifcloud.Cloud{}
			cloud.SetClient(c)
			cloud.SetRegion(region)

			err := nifcloud.ExportEnsureL4LoadBalancerDeleted(cloud, ctx, clusterName, testService)
			Expect(err).ShouldNot(HaveOccurred())
		})
	})

	Context("the l4 load balancer is not existed", func() {
		It("return nil", func() {
			ctx := context.Background()

			testLB := []nifcloud.LoadBalancer{}
			notFoundErr := helper.NewMockAPIError(nifcloud.ExportErrorCodeLoadBalancerNotFound)

			c := nifcloud.NewMockCloudAPIClient(ctrl)
			c.EXPECT().
				DescribeLoadBalancers(gomock.Any(), gomock.Eq(loadBalancerName)).
				Return(testLB, notFoundErr).
				Times(1)

			cloud := &nifcloud.Cloud{}
			cloud.SetClient(c)
			cloud.SetRegion(region)

			err := nifcloud.ExportEnsureL4LoadBalancerDeleted(cloud, ctx, clusterName, testService)
			Expect(err).ShouldNot(HaveOccurred())
		})
	})
})

var _ = Describe("findL4LoadBalancer", func() {
	var loadBalancerName = "testloadbalancer"

	Context("target is existed in the array", func() {
		It("return target elastic load balancer", func() {
			testLB := helper.NewTestL4LoadBalancerWithTwoPort(loadBalancerName)
			target := testLB[0]

			gotLB, err := nifcloud.ExportFindL4LoadBalancer(testLB, target)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(gotLB).Should(Equal(&testLB[0]))
		})
	})

	Context("target is not existed in the array", func() {
		It("return error", func() {
			testLB := helper.NewTestL4LoadBalancerWithTwoPort(loadBalancerName)
			target := helper.NewTestL4LoadBalancer("notexistedloadbalancer")[0]

			_, err := nifcloud.ExportFindL4LoadBalancer(testLB, target)
			Expect(err).Should(HaveOccurred())
		})
	})
})

var _ = Describe("l4LoadBalancerDifferences", func() {
	var loadBalancerName = "testloadbalancer"

	Context("target has a l4 load balancer is not existed in other", func() {
		It("return the l4 load balancer", func() {
			targetLB := helper.NewTestL4LoadBalancerWithTwoPort(loadBalancerName)
			otherLB := helper.NewTestL4LoadBalancer(loadBalancerName)

			expectLB := []nifcloud.LoadBalancer{targetLB[1]}

			gotLB := nifcloud.ExportL4LoadBalancerDifferences(targetLB, otherLB)
			Expect(gotLB).Should(Equal(expectLB))
		})
	})

	Context("target and other have same elastic load balancers", func() {
		It("return empty array", func() {
			targetLB := helper.NewTestL4LoadBalancerWithTwoPort(loadBalancerName)
			otherLB := helper.NewTestL4LoadBalancerWithTwoPort(loadBalancerName)

			expectLB := []nifcloud.LoadBalancer{}

			gotLB := nifcloud.ExportL4LoadBalancerDifferences(targetLB, otherLB)
			Expect(gotLB).Should(Equal(expectLB))
		})
	})

	Context("other has an elastic load balancer is not existed in target", func() {
		It("return empty array", func() {
			targetLB := helper.NewTestL4LoadBalancer(loadBalancerName)
			otherLB := helper.NewTestL4LoadBalancerWithTwoPort(loadBalancerName)

			expectLB := []nifcloud.LoadBalancer{}

			gotLB := nifcloud.ExportL4LoadBalancerDifferences(targetLB, otherLB)
			Expect(gotLB).Should(Equal(expectLB))
		})
	})
})

var _ = Describe("l4LoadBalancingTargetsDifferences", func() {
	Context("target has an instance is not existed in other", func() {
		It("return the instance", func() {
			targetInstance := []nifcloud.Instance{*helper.NewTestInstance(), *helper.NewTestInstance()}
			targetInstance[1].InstanceUniqueID = "i-xyzw5678"
			targetInstance[1].InstanceID = "testinstance2"
			otherInstance := []nifcloud.Instance{*helper.NewTestInstance()}

			expectInstance := []nifcloud.Instance{targetInstance[1]}

			gotInstance := nifcloud.ExportL4LoadBalancingTargetsDifferences(targetInstance, otherInstance)
			Expect(gotInstance).Should(Equal(expectInstance))
		})
	})

	Context("target and other have same instances", func() {
		It("return empty array", func() {
			targetInstance := []nifcloud.Instance{*helper.NewTestInstance(), *helper.NewTestInstance()}
			targetInstance[1].InstanceUniqueID = "i-xyzw5678"
			targetInstance[1].InstanceID = "testinstance2"
			otherInstance := []nifcloud.Instance{*helper.NewTestInstance(), *helper.NewTestInstance()}
			otherInstance[1].InstanceUniqueID = "i-xyzw5678"
			otherInstance[1].InstanceID = "testinstance2"

			expectInstance := []nifcloud.Instance{}

			gotInstance := nifcloud.ExportL4LoadBalancingTargetsDifferences(targetInstance, otherInstance)
			Expect(gotInstance).Should(Equal(expectInstance))
		})
	})

	Context("other has an instance is not existed in target", func() {
		It("return empty array", func() {
			targetInstance := []nifcloud.Instance{*helper.NewTestInstance()}
			otherInstance := []nifcloud.Instance{*helper.NewTestInstance(), *helper.NewTestInstance()}
			otherInstance[1].InstanceUniqueID = "i-xyzw5678"
			otherInstance[1].InstanceID = "testinstance2"

			expectInstance := []nifcloud.Instance{}

			gotInstance := nifcloud.ExportL4LoadBalancingTargetsDifferences(targetInstance, otherInstance)
			Expect(gotInstance).Should(Equal(expectInstance))
		})
	})
})

var _ = Describe("filterDifferences", func() {
	Context("target has an filter is not existed in other", func() {
		It("return the filter", func() {
			targetFilter := []string{"198.51.100.0/24", "192.0.2.0/24"}
			otherFilter := []string{"198.51.100.0/24"}

			expectFilter := []string{"192.0.2.0/24"}

			gotInstance := nifcloud.ExportFilterDifferences(targetFilter, otherFilter)
			Expect(gotInstance).Should(Equal(expectFilter))
		})
	})

	Context("target and other have same instances", func() {
		It("return empty array", func() {
			targetFilter := []string{"198.51.100.0/24", "192.0.2.0/24"}
			otherFilter := []string{"198.51.100.0/24", "192.0.2.0/24"}

			expectFilter := []string{}

			gotInstance := nifcloud.ExportFilterDifferences(targetFilter, otherFilter)
			Expect(gotInstance).Should(Equal(expectFilter))
		})
	})

	Context("other has an instance is not existed in target", func() {
		It("return empty array", func() {
			targetFilter := []string{"198.51.100.0/24"}
			otherFilter := []string{"198.51.100.0/24", "192.0.2.0/24"}

			expectFilter := []string{}

			gotInstance := nifcloud.ExportFilterDifferences(targetFilter, otherFilter)
			Expect(gotInstance).Should(Equal(expectFilter))
		})
	})
})
