package nifcloud_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/nifcloud/nifcloud-cloud-controller-manager/pkg/cloudprovider/providers/nifcloud"
	"github.com/nifcloud/nifcloud-cloud-controller-manager/test/helper"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"go.uber.org/mock/gomock"
	cloudprovider "k8s.io/cloud-provider"
)

var _ = Describe("nifcloudAPIClient", func() {
	var (
		region                string
		ctrl                  *gomock.Controller
		ts                    *httptest.Server
		handler               http.HandlerFunc
		testNifcloudAPIClient *nifcloud.ExportNifcloudAPIClient
	)

	BeforeEach(func() {
		region = "jp-east-1"
		ctrl = gomock.NewController(GinkgoT())
	})

	JustBeforeEach(func() {
		if handler == nil {
			GinkgoT().Fatal("handler for *httptest.Server is nil")
		}
		ts = helper.NewTestServer(handler)
		testNifcloudAPIClient = nifcloud.NewNIFCLOUDAPIClientWithEndpoint("testkey", "testsecretkey", region, ts.URL)
	})

	AfterEach(func() {
		ts.Close()
		ctrl.Finish()
		handler = nil
	})

	var _ = Describe("DescribeInstancesByInstanceID", func() {
		Describe("given instance is existed", func() {
			testInstanceIDs := []string{"testinstance"}

			BeforeEach(func() {
				handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					lo.Must0(r.ParseForm())
					Expect(r.Form.Get("InstanceId.1")).Should(Equal(testInstanceIDs[0]))
					_, _ = w.Write(lo.Must(os.ReadFile("./testdata/describe_instances_instance_id.xml")))
				})
			})

			It("return the instance", func() {
				ctx := context.Background()
				expectedInstances := []nifcloud.Instance{*helper.NewTestInstance()}
				gotInstances, gotErr := testNifcloudAPIClient.DescribeInstancesByInstanceID(ctx, testInstanceIDs)
				Expect(gotErr).ShouldNot(HaveOccurred())
				Expect(gotInstances).Should(Equal(expectedInstances))
			})
		})

		Describe("given instance is not existed", func() {
			testInstanceIDs := []string{"noinstance"}

			BeforeEach(func() {
				handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					lo.Must0(r.ParseForm())
					Expect(r.Form.Get("InstanceId.1")).Should(Equal(testInstanceIDs[0]))
					_, _ = w.Write(lo.Must(os.ReadFile("./testdata/describe_instances_not_found_instance_error.xml")))
				})
			})

			It("return InstanceNotFound error", func() {
				ctx := context.Background()
				gotInstances, gotErr := testNifcloudAPIClient.DescribeInstancesByInstanceID(ctx, testInstanceIDs)
				Expect(gotErr).Should(Equal(cloudprovider.InstanceNotFound))
				Expect(gotInstances).Should(BeNil())
			})
		})
	})

	var _ = Describe("DescribeInstancesByInstanceUniqueID", func() {
		Describe("given instance is existed", func() {
			testInstanceUniqueIDs := []string{"i-abcd1234"}

			BeforeEach(func() {
				handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					lo.Must0(r.ParseForm())
					_, _ = w.Write(lo.Must(os.ReadFile("./testdata/describe_instances.xml")))
				})
			})

			It("return the instance", func() {
				ctx := context.Background()
				expectedInstances := []nifcloud.Instance{*helper.NewTestInstance()}
				gotInstances, gotErr := testNifcloudAPIClient.DescribeInstancesByInstanceUniqueID(ctx, testInstanceUniqueIDs)
				Expect(gotErr).ShouldNot(HaveOccurred())
				Expect(gotInstances).Should(Equal(expectedInstances))
			})
		})

		Describe("given instance is not existed", func() {
			testInstanceUniqueIDs := []string{"1-xxxx0000"}

			BeforeEach(func() {
				handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					lo.Must0(r.ParseForm())
					_, _ = w.Write(lo.Must(os.ReadFile("./testdata/describe_instances.xml")))
				})
			})

			It("return InstanceNotFound error", func() {
				ctx := context.Background()
				gotInstances, gotErr := testNifcloudAPIClient.DescribeInstancesByInstanceUniqueID(ctx, testInstanceUniqueIDs)
				Expect(gotErr).Should(Equal(cloudprovider.InstanceNotFound))
				Expect(gotInstances).Should(BeNil())
			})
		})
	})

	var _ = Describe("DescribeLoadBalancers", func() {
		Describe("given l4 load balancer is existed", func() {
			testLoadBalancerName := "testl4lb"

			BeforeEach(func() {
				handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					lo.Must0(r.ParseForm())
					Expect(r.Form.Get("LoadBalancerNames.member.1")).Should(Equal(testLoadBalancerName))
					_, _ = w.Write(lo.Must(os.ReadFile("./testdata/describe_load_balancers_load_balancer_name.xml")))
				})
			})

			It("return the l4 load balancer", func() {
				ctx := context.Background()
				expectedL4LoadBalancers := helper.NewTestL4LoadBalancer(testLoadBalancerName)
				expectedL4LoadBalancers[0].VIP = "203.0.113.5"
				expectedL4LoadBalancers[0].BalancingTargets[0].InstanceType = ""
				expectedL4LoadBalancers[0].BalancingTargets[0].PublicIPAddress = ""
				expectedL4LoadBalancers[0].BalancingTargets[0].PrivateIPAddress = ""
				expectedL4LoadBalancers[0].BalancingTargets[0].Zone = ""
				expectedL4LoadBalancers[0].BalancingTargets[0].State = ""
				gotL4LoadBalancer, gotErr := testNifcloudAPIClient.DescribeLoadBalancers(ctx, testLoadBalancerName)
				Expect(gotErr).ShouldNot(HaveOccurred())
				Expect(gotL4LoadBalancer).Should(Equal(expectedL4LoadBalancers))
			})
		})

		Describe("given l4 load balancer is existed and it has two ports and filters", func() {
			testLoadBalancerName := "testl4lb"

			BeforeEach(func() {
				handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					lo.Must0(r.ParseForm())
					Expect(r.Form.Get("LoadBalancerNames.member.1")).Should(Equal(testLoadBalancerName))
					_, _ = w.Write(lo.Must(os.ReadFile("./testdata/describe_load_balancers_two_ports_and_filters.xml")))
				})
			})

			It("return the l4 load balancer", func() {
				ctx := context.Background()
				expectedL4LoadBalancers := helper.NewTestL4LoadBalancerWithTwoPort(testLoadBalancerName)
				for i := range expectedL4LoadBalancers {
					expectedL4LoadBalancers[i].VIP = "203.0.113.5"
					expectedL4LoadBalancers[i].BalancingTargets[0].InstanceType = ""
					expectedL4LoadBalancers[i].BalancingTargets[0].PublicIPAddress = ""
					expectedL4LoadBalancers[i].BalancingTargets[0].PrivateIPAddress = ""
					expectedL4LoadBalancers[i].BalancingTargets[0].Zone = ""
					expectedL4LoadBalancers[i].BalancingTargets[0].State = ""
				}
				expectedL4LoadBalancers[0].Filters = []string{"203.0.113.6", "203.0.113.7"}
				gotL4LoadBalancer, gotErr := testNifcloudAPIClient.DescribeLoadBalancers(ctx, testLoadBalancerName)
				Expect(gotErr).ShouldNot(HaveOccurred())
				Expect(gotL4LoadBalancer).Should(Equal(expectedL4LoadBalancers))
			})
		})

		Describe("given l4 load balancer is not existed", func() {
			testLoadBalancerName := "testl4lb"

			BeforeEach(func() {
				handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					lo.Must0(r.ParseForm())
					Expect(r.Form.Get("LoadBalancerNames.member.1")).Should(Equal(testLoadBalancerName))
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = w.Write(lo.Must(os.ReadFile("./testdata/describe_load_balancers_not_found_load_balancer.xml")))
				})
			})

			It("return the l4 load balancer", func() {
				ctx := context.Background()
				gotL4LoadBalancer, gotErr := testNifcloudAPIClient.DescribeLoadBalancers(ctx, testLoadBalancerName)
				Expect(gotErr).Should(HaveOccurred())
				Expect(nifcloud.IsAPIError(gotErr, nifcloud.ExportErrorCodeLoadBalancerNotFound)).Should(BeTrue())
				Expect(gotL4LoadBalancer).Should(BeNil())
			})
		})
	})

	var _ = Describe("createLoadBalancer", func() {
		Describe("creating l4 load balancer is success", func() {
			testLoadBalancerName := "testl4lb"

			BeforeEach(func() {
				handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					lo.Must0(r.ParseForm())
					Expect(r.Form.Get("LoadBalancerName")).Should(Equal(testLoadBalancerName))
					_, _ = w.Write(lo.Must(os.ReadFile("./testdata/create_load_balancer.xml")))
				})
			})

			It("return the DNS name and nil", func() {
				ctx := context.Background()
				expectDNSName := "203.0.113.5"
				testLoadBalancers := helper.NewTestL4LoadBalancer(testLoadBalancerName)
				gotDNSName, gotErr := nifcloud.ExportCreateLoadBalancer(testNifcloudAPIClient, ctx, &testLoadBalancers[0])
				Expect(gotErr).ShouldNot(HaveOccurred())
				Expect(gotDNSName).Should(Equal(expectDNSName))
			})
		})
	})

	Describe("the specified l4 load balancer is already existed", func() {
		testLoadBalancerName := "testl4lb"

		BeforeEach(func() {
			handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				lo.Must0(r.ParseForm())
				Expect(r.Form.Get("LoadBalancerName")).Should(Equal(testLoadBalancerName))
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write(lo.Must(os.ReadFile("./testdata/create_load_balancer_duplicate_load_balancer.xml")))
			})
		})

		It("return error", func() {
			ctx := context.Background()
			testLoadBalancers := helper.NewTestL4LoadBalancer(testLoadBalancerName)
			gotDNSName, gotErr := nifcloud.ExportCreateLoadBalancer(testNifcloudAPIClient, ctx, &testLoadBalancers[0])
			Expect(gotErr).Should(HaveOccurred())
			Expect(gotDNSName).Should(BeEmpty())
		})
	})

	var _ = Describe("registerPortWithLoadBalancer", func() {
		Describe("creating port with l4 load balancer is success", func() {
			testLoadBalancerName := "testl4lb"

			BeforeEach(func() {
				handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					lo.Must0(r.ParseForm())
					Expect(r.Form.Get("LoadBalancerName")).Should(Equal(testLoadBalancerName))
					Expect(r.Form.Get("Listeners.member.1.LoadBalancerPort")).Should(Equal("443"))
					Expect(r.Form.Get("Listeners.member.1.InstancePort")).Should(Equal("30001"))
					Expect(r.Form.Get("Listeners.member.1.BalancingType")).Should(Equal("1"))
					_, _ = w.Write(lo.Must(os.ReadFile("./testdata/register_port_with_load_balancer.xml")))
				})
			})

			It("return the l4 load balancer", func() {
				ctx := context.Background()
				expectedL4LoadBalancers := helper.NewTestL4LoadBalancerWithTwoPort(testLoadBalancerName)
				gotErr := nifcloud.ExportRegisterPortWithLoadBalancer(testNifcloudAPIClient, ctx, &expectedL4LoadBalancers[1])
				Expect(gotErr).ShouldNot(HaveOccurred())
			})
		})

		Describe("the specified port is already registered", func() {
			testLoadBalancerName := "testl4lb"

			BeforeEach(func() {
				handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					lo.Must0(r.ParseForm())
					Expect(r.Form.Get("LoadBalancerName")).Should(Equal(testLoadBalancerName))
					Expect(r.Form.Get("Listeners.member.1.LoadBalancerPort")).Should(Equal("443"))
					Expect(r.Form.Get("Listeners.member.1.InstancePort")).Should(Equal("30001"))
					Expect(r.Form.Get("Listeners.member.1.BalancingType")).Should(Equal("1"))
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = w.Write(lo.Must(os.ReadFile("./testdata/register_prot_with_load_balancer_duplicate_port.xml")))
				})
			})

			It("return error", func() {
				ctx := context.Background()
				expectedL4LoadBalancers := helper.NewTestL4LoadBalancerWithTwoPort(testLoadBalancerName)
				gotErr := nifcloud.ExportRegisterPortWithLoadBalancer(testNifcloudAPIClient, ctx, &expectedL4LoadBalancers[1])
				Expect(gotErr).Should(HaveOccurred())
			})
		})

		Describe("the specified l4 load balancer is not existed", func() {
			testLoadBalancerName := "testl4lb"

			BeforeEach(func() {
				handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					lo.Must0(r.ParseForm())
					Expect(r.Form.Get("LoadBalancerName")).Should(Equal(testLoadBalancerName))
					Expect(r.Form.Get("Listeners.member.1.LoadBalancerPort")).Should(Equal("443"))
					Expect(r.Form.Get("Listeners.member.1.InstancePort")).Should(Equal("30001"))
					Expect(r.Form.Get("Listeners.member.1.BalancingType")).Should(Equal("1"))
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = w.Write(lo.Must(os.ReadFile("./testdata/register_port_with_load_balancer_not_found_load_balancer.xml")))
				})
			})

			It("return error", func() {
				ctx := context.Background()
				expectedL4LoadBalancers := helper.NewTestL4LoadBalancerWithTwoPort(testLoadBalancerName)
				gotErr := nifcloud.ExportRegisterPortWithLoadBalancer(testNifcloudAPIClient, ctx, &expectedL4LoadBalancers[1])
				Expect(gotErr).Should(HaveOccurred())
			})
		})
	})
})
