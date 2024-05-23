package nifcloud_test

import (
	"fmt"

	"github.com/nifcloud/nifcloud-cloud-controller-manager/pkg/cloudprovider/providers/nifcloud"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

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
