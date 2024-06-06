package nifcloud_test

import (
	"context"
	"fmt"
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

var _ = Describe("EnsureLoadBalancer", func() {
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

	Context("given valid service for l4 load balancer", func() {
		It("create the l4 load balancer", func() {
			ctx := context.Background()
			testClusterName := "testcluster"
			testIPAddress := "203.0.113.1"
			testService := corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testlbsvc",
					UID:  loadBalancerUID,
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
			testService.SetUID(loadBalancerUID)
			testNode := &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testinstance",
				},
			}
			testDesire := helper.NewTestL4LoadBalancer(loadBalancerName)
			testInstances := []nifcloud.Instance{*helper.NewTestInstance()}
			testInstanceID := "testinstance"

			expectedStatus := &corev1.LoadBalancerStatus{
				Ingress: []corev1.LoadBalancerIngress{
					{
						IP: testIPAddress,
					},
				},
			}

			c := nifcloud.NewMockCloudAPIClient(ctrl)
			c.EXPECT().
				DescribeInstancesByInstanceID(gomock.Any(), gomock.Eq([]string{testInstanceID})).
				Return(testInstances, nil).
				Times(1)
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

			status, err := cloud.EnsureLoadBalancer(ctx, testClusterName, &testService, []*corev1.Node{testNode})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(*status).Should(Equal(*expectedStatus))
		})
	})

	Context("given invalid service for l4 load balancer", func() {
		Context("the load balancer has no ports", func() {
			It("return error", func() {
				ctx := context.Background()
				testClusterName := "testcluster"
				testService := corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testlbsvc",
						UID:  loadBalancerUID,
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
				}
				testService.SetUID(loadBalancerUID)
				testNode := &corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testinstance",
					},
				}

				cloud := &nifcloud.Cloud{}
				cloud.SetRegion(region)

				status, err := cloud.EnsureLoadBalancer(ctx, testClusterName, &testService, []*corev1.Node{testNode})
				Expect(err).Should(HaveOccurred())
				Expect(status).Should(BeNil())
			})
		})

		Context("the load balancer has 4 ports", func() {
			It("return error", func() {
				ctx := context.Background()
				testClusterName := "testcluster"
				testService := corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testlbsvc",
						UID:  loadBalancerUID,
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
							{
								Port:     443,
								NodePort: 30001,
								Protocol: corev1.ProtocolTCP,
							},
							{
								Port:     8000,
								NodePort: 30002,
								Protocol: corev1.ProtocolTCP,
							},
							{
								Port:     8080,
								NodePort: 30003,
								Protocol: corev1.ProtocolTCP,
							},
						},
					},
				}
				testService.SetUID(loadBalancerUID)
				testNode := &corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testinstance",
					},
				}

				cloud := &nifcloud.Cloud{}
				cloud.SetRegion(region)

				status, err := cloud.EnsureLoadBalancer(ctx, testClusterName, &testService, []*corev1.Node{testNode})
				Expect(err).Should(HaveOccurred())
				Expect(status).Should(BeNil())
			})
		})

		Context("LoadBalancerIP is defined in the service", func() {
			It("return error", func() {
				ctx := context.Background()
				testClusterName := "testcluster"
				testService := corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testlbsvc",
						UID:  loadBalancerUID,
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
						LoadBalancerIP: "203.0.113.1",
						Ports: []corev1.ServicePort{
							{
								Port:     80,
								NodePort: 30000,
								Protocol: corev1.ProtocolTCP,
							},
						},
					},
				}
				testService.SetUID(loadBalancerUID)
				testNode := &corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testinstance",
					},
				}

				cloud := &nifcloud.Cloud{}
				cloud.SetRegion(region)

				status, err := cloud.EnsureLoadBalancer(ctx, testClusterName, &testService, []*corev1.Node{testNode})
				Expect(err).Should(HaveOccurred())
				Expect(status).Should(BeNil())
			})
		})
	})

	Context("given service is not for l4 load balancer and elastic load balancer", func() {
		It("return error", func() {
			ctx := context.Background()
			testClusterName := "testcluster"
			testService := corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testlbsvc",
					UID:  loadBalancerUID,
					Annotations: map[string]string{
						nifcloud.ServiceAnnotationLoadBalancerType: "unknown",
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
			testService.SetUID(loadBalancerUID)
			testNode := &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testinstance",
				},
			}
			testInstances := []nifcloud.Instance{*helper.NewTestInstance()}
			testInstanceID := "testinstance"

			c := nifcloud.NewMockCloudAPIClient(ctrl)
			c.EXPECT().
				DescribeInstancesByInstanceID(gomock.Any(), gomock.Eq([]string{testInstanceID})).
				Return(testInstances, nil).
				Times(1)

			cloud := &nifcloud.Cloud{}
			cloud.SetClient(c)
			cloud.SetRegion(region)

			status, err := cloud.EnsureLoadBalancer(ctx, testClusterName, &testService, []*corev1.Node{testNode})
			Expect(err).Should(HaveOccurred())
			Expect(status).Should(BeNil())
		})
	})
})

var _ = Describe("validateLoadBalancerAnnotations", func() {
	var testAnnotations map[string]string

	Describe("load balancer type is l4 load balancer", func() {
		var testAnnotationsEmptyLBType map[string]string

		BeforeEach(func() {
			testAnnotations = map[string]string{
				nifcloud.ServiceAnnotationLoadBalancerType: "lb",
			}
			testAnnotationsEmptyLBType = map[string]string{}
		})

		Context("not define other annotations", func() {
			It("return nil", func() {
				err := nifcloud.ExportValidateLoadBalancerAnnotations(testAnnotationsEmptyLBType)
				Expect(err).ShouldNot(HaveOccurred())

				err = nifcloud.ExportValidateLoadBalancerAnnotations(testAnnotations)
				Expect(err).ShouldNot(HaveOccurred())
			})
		})

		DescribeTable("given valid annotations",
			func(key string, value string) {
				testAnnotationsEmptyLBType[key] = value
				err := nifcloud.ExportValidateLoadBalancerAnnotations(testAnnotationsEmptyLBType)
				Expect(err).ShouldNot(HaveOccurred())

				testAnnotations[key] = value
				err = nifcloud.ExportValidateLoadBalancerAnnotations(testAnnotations)
				Expect(err).ShouldNot(HaveOccurred())
			},
			func(key string, value string) string {
				return fmt.Sprintf("%s=%s", key, value)
			},
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerBalancingType, "1"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerBalancingType, "2"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerAccountingType, "1"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerAccountingType, "2"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerHCProtocol, "TCP"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerHCProtocol, "ICMP"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerHCUnhealthyThreshold, "1"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerHCUnhealthyThreshold, "10"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerHCInterval, "5"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerHCInterval, "300"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerNetworkVolume, "10"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerNetworkVolume, "2000"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerPolicyType, "standard"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerPolicyType, "ats"),
		)

		DescribeTable("given invalid annotations",
			func(key string, value string) {
				testAnnotationsEmptyLBType[key] = value
				err := nifcloud.ExportValidateLoadBalancerAnnotations(testAnnotationsEmptyLBType)
				Expect(err).Should(HaveOccurred())

				testAnnotations[key] = value
				err = nifcloud.ExportValidateLoadBalancerAnnotations(testAnnotations)
				Expect(err).Should(HaveOccurred())
			},
			func(key string, value string) string {
				return fmt.Sprintf("%s=%s", key, value)
			},
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerBalancingType, "0"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerBalancingType, "3"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerBalancingType, "notNumber"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerAccountingType, "0"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerAccountingType, "3"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerAccountingType, "notNumber"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerHCProtocol, "HTTP"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerHCProtocol, "HTTPS"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerHCUnhealthyThreshold, "0"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerHCUnhealthyThreshold, "11"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerHCUnhealthyThreshold, "notNumber"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerHCInterval, "4"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerHCInterval, "301"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerHCInterval, "notNumber"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerNetworkVolume, "0"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerNetworkVolume, "2100"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerNetworkVolume, "notNumber"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerPolicyType, "undefined"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerNetworkInterface1, "any"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerNetworkInterface2, "any"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerNetworkInterface1IPAddress, "any"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerNetworkInterface2IPAddress, "any"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerNetworkInterface1SystemIPAddresses, "any"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerNetworkInterface2SystemIPAddresses, "any"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerVipNetwork, "any"),
		)
	})

	Describe("load balancer type is elastic load balancer", func() {
		BeforeEach(func() {
			testAnnotations = map[string]string{
				nifcloud.ServiceAnnotationLoadBalancerType: "elb",
			}
		})

		Context("not define other annotations", func() {
			It("return nil", func() {
				err := nifcloud.ExportValidateLoadBalancerAnnotations(testAnnotations)
				Expect(err).ShouldNot(HaveOccurred())
			})
		})

		DescribeTable("given valid annotations",
			func(key string, value string) {
				testAnnotations[key] = value
				err := nifcloud.ExportValidateLoadBalancerAnnotations(testAnnotations)
				Expect(err).ShouldNot(HaveOccurred())
			},
			func(key string, value string) string {
				return fmt.Sprintf("%s=%s", key, value)
			},
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerBalancingType, "1"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerBalancingType, "2"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerAccountingType, "1"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerAccountingType, "2"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerHCProtocol, "TCP"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerHCProtocol, "ICMP"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerHCUnhealthyThreshold, "1"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerHCUnhealthyThreshold, "10"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerHCInterval, "5"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerHCInterval, "300"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerNetworkVolume, "10"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerNetworkVolume, "500"),
		)

		DescribeTable("given invalid annotations",
			func(key string, value string) {
				testAnnotations[key] = value
				err := nifcloud.ExportValidateLoadBalancerAnnotations(testAnnotations)
				Expect(err).Should(HaveOccurred())
			},
			func(key string, value string) string {
				return fmt.Sprintf("%s=%s", key, value)
			},
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerBalancingType, "0"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerBalancingType, "3"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerBalancingType, "notNumber"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerAccountingType, "0"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerAccountingType, "3"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerAccountingType, "notNumber"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerHCProtocol, "HTTP"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerHCProtocol, "HTTPS"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerHCUnhealthyThreshold, "0"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerHCUnhealthyThreshold, "11"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerHCUnhealthyThreshold, "notNumber"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerHCInterval, "4"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerHCInterval, "301"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerHCInterval, "notNumber"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerNetworkVolume, "0"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerNetworkVolume, "600"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerNetworkVolume, "notNumber"),
			Entry(nil, nifcloud.ServiceAnnotationLoadBalancerPolicyType, "any"),
		)

		Context("annotations has common global network or common private network", func() {
			var testAnnotationsNetworkInterface1Global map[string]string
			var testAnnotationsNetworkInterface1Private map[string]string
			var testAnnotationsNetworkInterface2Global map[string]string
			var testAnnotationsNetworkInterface2Private map[string]string

			BeforeEach(func() {
				testAnnotationsNetworkInterface1Global = map[string]string{
					nifcloud.ServiceAnnotationLoadBalancerType:              "elb",
					nifcloud.ServiceAnnotationLoadBalancerNetworkInterface1: "net-COMMON_GLOBAL",
				}
				testAnnotationsNetworkInterface1Private = map[string]string{
					nifcloud.ServiceAnnotationLoadBalancerType:              "elb",
					nifcloud.ServiceAnnotationLoadBalancerNetworkInterface1: "net-COMMON_PRIVATE",
				}
				testAnnotationsNetworkInterface2Global = map[string]string{
					nifcloud.ServiceAnnotationLoadBalancerType:              "elb",
					nifcloud.ServiceAnnotationLoadBalancerNetworkInterface2: "net-COMMON_GLOBAL",
				}
				testAnnotationsNetworkInterface2Private = map[string]string{
					nifcloud.ServiceAnnotationLoadBalancerType:              "elb",
					nifcloud.ServiceAnnotationLoadBalancerNetworkInterface2: "net-COMMON_PRIVATE",
				}
			})

			Context("not define ip address and system ip addresses", func() {
				It("return nil", func() {
					err := nifcloud.ExportValidateLoadBalancerAnnotations(testAnnotationsNetworkInterface1Global)
					Expect(err).ShouldNot(HaveOccurred())

					err = nifcloud.ExportValidateLoadBalancerAnnotations(testAnnotationsNetworkInterface1Private)
					Expect(err).ShouldNot(HaveOccurred())

					err = nifcloud.ExportValidateLoadBalancerAnnotations(testAnnotationsNetworkInterface2Global)
					Expect(err).ShouldNot(HaveOccurred())

					err = nifcloud.ExportValidateLoadBalancerAnnotations(testAnnotationsNetworkInterface2Private)
					Expect(err).ShouldNot(HaveOccurred())
				})
			})

			Context("ip address and system ip addresses are empty", func() {
				It("return nil", func() {
					testAnnotationsNetworkInterface1Global[nifcloud.ServiceAnnotationLoadBalancerNetworkInterface1IPAddress] = ""
					testAnnotationsNetworkInterface1Global[nifcloud.ServiceAnnotationLoadBalancerNetworkInterface1SystemIPAddresses] = ""
					err := nifcloud.ExportValidateLoadBalancerAnnotations(testAnnotationsNetworkInterface1Global)
					Expect(err).ShouldNot(HaveOccurred())

					testAnnotationsNetworkInterface1Private[nifcloud.ServiceAnnotationLoadBalancerNetworkInterface1IPAddress] = ""
					testAnnotationsNetworkInterface1Private[nifcloud.ServiceAnnotationLoadBalancerNetworkInterface1SystemIPAddresses] = ""
					err = nifcloud.ExportValidateLoadBalancerAnnotations(testAnnotationsNetworkInterface1Private)
					Expect(err).ShouldNot(HaveOccurred())

					testAnnotationsNetworkInterface2Global[nifcloud.ServiceAnnotationLoadBalancerNetworkInterface2IPAddress] = ""
					testAnnotationsNetworkInterface2Global[nifcloud.ServiceAnnotationLoadBalancerNetworkInterface2SystemIPAddresses] = ""
					err = nifcloud.ExportValidateLoadBalancerAnnotations(testAnnotationsNetworkInterface2Global)
					Expect(err).ShouldNot(HaveOccurred())

					testAnnotationsNetworkInterface2Private[nifcloud.ServiceAnnotationLoadBalancerNetworkInterface2IPAddress] = ""
					testAnnotationsNetworkInterface2Private[nifcloud.ServiceAnnotationLoadBalancerNetworkInterface2SystemIPAddresses] = ""
					err = nifcloud.ExportValidateLoadBalancerAnnotations(testAnnotationsNetworkInterface2Private)
					Expect(err).ShouldNot(HaveOccurred())
				})
			})

			Context("set ip address", func() {
				It("return error", func() {
					testAnnotationsNetworkInterface1Global[nifcloud.ServiceAnnotationLoadBalancerNetworkInterface1IPAddress] = "203.0.113.1"
					err := nifcloud.ExportValidateLoadBalancerAnnotations(testAnnotationsNetworkInterface1Global)
					Expect(err).Should(HaveOccurred())

					testAnnotationsNetworkInterface1Private[nifcloud.ServiceAnnotationLoadBalancerNetworkInterface1IPAddress] = "203.0.113.1"
					err = nifcloud.ExportValidateLoadBalancerAnnotations(testAnnotationsNetworkInterface1Private)
					Expect(err).Should(HaveOccurred())

					testAnnotationsNetworkInterface2Global[nifcloud.ServiceAnnotationLoadBalancerNetworkInterface2IPAddress] = "203.0.113.1"
					err = nifcloud.ExportValidateLoadBalancerAnnotations(testAnnotationsNetworkInterface2Global)
					Expect(err).Should(HaveOccurred())

					testAnnotationsNetworkInterface2Private[nifcloud.ServiceAnnotationLoadBalancerNetworkInterface2IPAddress] = "203.0.113.1"
					err = nifcloud.ExportValidateLoadBalancerAnnotations(testAnnotationsNetworkInterface2Private)
					Expect(err).Should(HaveOccurred())
				})
			})

			Context("set system ip addresses", func() {
				It("return error", func() {
					testAnnotationsNetworkInterface1Global[nifcloud.ServiceAnnotationLoadBalancerNetworkInterface1SystemIPAddresses] = "203.0.113.10,203.0.113.11"
					err := nifcloud.ExportValidateLoadBalancerAnnotations(testAnnotationsNetworkInterface1Global)
					Expect(err).Should(HaveOccurred())

					testAnnotationsNetworkInterface1Private[nifcloud.ServiceAnnotationLoadBalancerNetworkInterface1SystemIPAddresses] = "203.0.113.10,203.0.113.11"
					err = nifcloud.ExportValidateLoadBalancerAnnotations(testAnnotationsNetworkInterface1Private)
					Expect(err).Should(HaveOccurred())

					testAnnotationsNetworkInterface2Global[nifcloud.ServiceAnnotationLoadBalancerNetworkInterface2SystemIPAddresses] = "203.0.113.10,203.0.113.11"
					err = nifcloud.ExportValidateLoadBalancerAnnotations(testAnnotationsNetworkInterface2Global)
					Expect(err).Should(HaveOccurred())

					testAnnotationsNetworkInterface2Private[nifcloud.ServiceAnnotationLoadBalancerNetworkInterface2SystemIPAddresses] = "203.0.113.10,203.0.113.11"
					err = nifcloud.ExportValidateLoadBalancerAnnotations(testAnnotationsNetworkInterface2Private)
					Expect(err).Should(HaveOccurred())
				})
			})
		})

		Context("annotations has private network as network interface 1", func() {
			var testAnnotationsNetworkInterface1 map[string]string
			var testAnnotationsNetworkInterface2 map[string]string

			BeforeEach(func() {
				testAnnotationsNetworkInterface1 = map[string]string{
					nifcloud.ServiceAnnotationLoadBalancerType:              "elb",
					nifcloud.ServiceAnnotationLoadBalancerNetworkInterface1: "net-abcd2134",
				}
				testAnnotationsNetworkInterface2 = map[string]string{
					nifcloud.ServiceAnnotationLoadBalancerType:              "elb",
					nifcloud.ServiceAnnotationLoadBalancerNetworkInterface2: "net-abcd2134",
				}
			})

			Context("not define ip address", func() {
				It("return error", func() {
					testAnnotationsNetworkInterface1[nifcloud.ServiceAnnotationLoadBalancerNetworkInterface1SystemIPAddresses] = "203.0.113.10,203.0.113.11"
					err := nifcloud.ExportValidateLoadBalancerAnnotations(testAnnotationsNetworkInterface1)
					Expect(err).Should(HaveOccurred())

					testAnnotationsNetworkInterface2[nifcloud.ServiceAnnotationLoadBalancerNetworkInterface2SystemIPAddresses] = "203.0.113.10,203.0.113.11"
					err = nifcloud.ExportValidateLoadBalancerAnnotations(testAnnotationsNetworkInterface2)
					Expect(err).Should(HaveOccurred())
				})
			})

			Context("ip address is empty", func() {
				It("return error", func() {
					testAnnotationsNetworkInterface1[nifcloud.ServiceAnnotationLoadBalancerNetworkInterface1IPAddress] = ""
					testAnnotationsNetworkInterface1[nifcloud.ServiceAnnotationLoadBalancerNetworkInterface1SystemIPAddresses] = "203.0.113.10,203.0.113.11"
					err := nifcloud.ExportValidateLoadBalancerAnnotations(testAnnotationsNetworkInterface1)
					Expect(err).Should(HaveOccurred())

					testAnnotationsNetworkInterface2[nifcloud.ServiceAnnotationLoadBalancerNetworkInterface2IPAddress] = ""
					testAnnotationsNetworkInterface2[nifcloud.ServiceAnnotationLoadBalancerNetworkInterface2SystemIPAddresses] = "203.0.113.10,203.0.113.11"
					err = nifcloud.ExportValidateLoadBalancerAnnotations(testAnnotationsNetworkInterface2)
					Expect(err).Should(HaveOccurred())
				})
			})

			Context("ip address is invalid format", func() {
				It("return error", func() {
					testAnnotationsNetworkInterface1[nifcloud.ServiceAnnotationLoadBalancerNetworkInterface1IPAddress] = "notIPAddress"
					testAnnotationsNetworkInterface1[nifcloud.ServiceAnnotationLoadBalancerNetworkInterface1SystemIPAddresses] = "203.0.113.10,203.0.113.11"
					err := nifcloud.ExportValidateLoadBalancerAnnotations(testAnnotationsNetworkInterface1)
					Expect(err).Should(HaveOccurred())

					testAnnotationsNetworkInterface2[nifcloud.ServiceAnnotationLoadBalancerNetworkInterface2IPAddress] = "notIPAddress"
					testAnnotationsNetworkInterface2[nifcloud.ServiceAnnotationLoadBalancerNetworkInterface2SystemIPAddresses] = "203.0.113.10,203.0.113.11"
					err = nifcloud.ExportValidateLoadBalancerAnnotations(testAnnotationsNetworkInterface2)
					Expect(err).Should(HaveOccurred())
				})
			})

			Context("not define system ip address", func() {
				It("return error", func() {
					testAnnotationsNetworkInterface1[nifcloud.ServiceAnnotationLoadBalancerNetworkInterface1IPAddress] = "203.0.113.1"
					err := nifcloud.ExportValidateLoadBalancerAnnotations(testAnnotationsNetworkInterface1)
					Expect(err).Should(HaveOccurred())

					testAnnotationsNetworkInterface2[nifcloud.ServiceAnnotationLoadBalancerNetworkInterface2IPAddress] = "203.0.113.1"
					err = nifcloud.ExportValidateLoadBalancerAnnotations(testAnnotationsNetworkInterface2)
					Expect(err).Should(HaveOccurred())
				})
			})

			Context("system ip address are empty", func() {
				It("return error", func() {
					testAnnotationsNetworkInterface1[nifcloud.ServiceAnnotationLoadBalancerNetworkInterface1IPAddress] = "203.0.113.1"
					testAnnotationsNetworkInterface1[nifcloud.ServiceAnnotationLoadBalancerNetworkInterface1SystemIPAddresses] = ""
					err := nifcloud.ExportValidateLoadBalancerAnnotations(testAnnotationsNetworkInterface1)
					Expect(err).Should(HaveOccurred())

					testAnnotationsNetworkInterface2[nifcloud.ServiceAnnotationLoadBalancerNetworkInterface2IPAddress] = "203.0.113.1"
					testAnnotationsNetworkInterface2[nifcloud.ServiceAnnotationLoadBalancerNetworkInterface2SystemIPAddresses] = ""
					err = nifcloud.ExportValidateLoadBalancerAnnotations(testAnnotationsNetworkInterface2)
					Expect(err).Should(HaveOccurred())
				})
			})

			Context("system ip address are invalid format", func() {
				It("return error", func() {
					testAnnotationsNetworkInterface1[nifcloud.ServiceAnnotationLoadBalancerNetworkInterface1IPAddress] = "203.0.113.1"
					testAnnotationsNetworkInterface1[nifcloud.ServiceAnnotationLoadBalancerNetworkInterface1SystemIPAddresses] = "notIPAddresses"
					err := nifcloud.ExportValidateLoadBalancerAnnotations(testAnnotationsNetworkInterface1)
					Expect(err).Should(HaveOccurred())

					testAnnotationsNetworkInterface2[nifcloud.ServiceAnnotationLoadBalancerNetworkInterface2IPAddress] = "203.0.113.1"
					testAnnotationsNetworkInterface2[nifcloud.ServiceAnnotationLoadBalancerNetworkInterface2SystemIPAddresses] = "notIPAddresses"
					err = nifcloud.ExportValidateLoadBalancerAnnotations(testAnnotationsNetworkInterface2)
					Expect(err).Should(HaveOccurred())
				})
			})

			Context("fewer system ip address than two", func() {
				It("return error", func() {
					testAnnotationsNetworkInterface1[nifcloud.ServiceAnnotationLoadBalancerNetworkInterface1IPAddress] = "203.0.113.1"
					testAnnotationsNetworkInterface1[nifcloud.ServiceAnnotationLoadBalancerNetworkInterface1SystemIPAddresses] = "203.0.113.10"
					err := nifcloud.ExportValidateLoadBalancerAnnotations(testAnnotationsNetworkInterface1)
					Expect(err).Should(HaveOccurred())

					testAnnotationsNetworkInterface2[nifcloud.ServiceAnnotationLoadBalancerNetworkInterface2IPAddress] = "203.0.113.1"
					testAnnotationsNetworkInterface2[nifcloud.ServiceAnnotationLoadBalancerNetworkInterface2SystemIPAddresses] = "203.0.113.10"
					err = nifcloud.ExportValidateLoadBalancerAnnotations(testAnnotationsNetworkInterface2)
					Expect(err).Should(HaveOccurred())
				})
			})

			Context("more system ip address than two", func() {
				It("return error", func() {
					testAnnotationsNetworkInterface1[nifcloud.ServiceAnnotationLoadBalancerNetworkInterface1IPAddress] = "203.0.113.1"
					testAnnotationsNetworkInterface1[nifcloud.ServiceAnnotationLoadBalancerNetworkInterface1SystemIPAddresses] = "203.0.113.10,203.0.113.11,203.0.113.12"
					err := nifcloud.ExportValidateLoadBalancerAnnotations(testAnnotationsNetworkInterface1)
					Expect(err).Should(HaveOccurred())

					testAnnotationsNetworkInterface2[nifcloud.ServiceAnnotationLoadBalancerNetworkInterface2IPAddress] = "203.0.113.1"
					testAnnotationsNetworkInterface2[nifcloud.ServiceAnnotationLoadBalancerNetworkInterface2SystemIPAddresses] = "203.0.113.10,203.0.113.11,203.0.113.12"
					err = nifcloud.ExportValidateLoadBalancerAnnotations(testAnnotationsNetworkInterface2)
					Expect(err).Should(HaveOccurred())
				})
			})

			Context("set valid ip address and system ip addresses", func() {
				It("return nil", func() {
					testAnnotationsNetworkInterface1[nifcloud.ServiceAnnotationLoadBalancerNetworkInterface1IPAddress] = "203.0.113.1"
					testAnnotationsNetworkInterface1[nifcloud.ServiceAnnotationLoadBalancerNetworkInterface1SystemIPAddresses] = "203.0.113.10,203.0.113.11"
					err := nifcloud.ExportValidateLoadBalancerAnnotations(testAnnotationsNetworkInterface1)
					Expect(err).ShouldNot(HaveOccurred())

					testAnnotationsNetworkInterface2[nifcloud.ServiceAnnotationLoadBalancerNetworkInterface2IPAddress] = "203.0.113.1"
					testAnnotationsNetworkInterface2[nifcloud.ServiceAnnotationLoadBalancerNetworkInterface2SystemIPAddresses] = "203.0.113.10,203.0.113.11"
					err = nifcloud.ExportValidateLoadBalancerAnnotations(testAnnotationsNetworkInterface2)
					Expect(err).ShouldNot(HaveOccurred())
				})
			})
		})
	})
})
