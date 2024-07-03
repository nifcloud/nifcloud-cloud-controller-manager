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

	var _ = Describe("ConfigureHealthCheck", func() {
		Describe("configuring health check is success", func() {
			testLoadBalancerName := "testl4lb"

			BeforeEach(func() {
				handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					lo.Must0(r.ParseForm())
					Expect(r.Form.Get("LoadBalancerName")).Should(Equal(testLoadBalancerName))
					Expect(r.Form.Get("HealthCheck.Target")).Should(Equal("TCP:30000"))
					Expect(r.Form.Get("HealthCheck.UnhealthyThreshold")).Should(Equal("1"))
					Expect(r.Form.Get("HealthCheck.Interval")).Should(Equal("10"))
					_, _ = w.Write(lo.Must(os.ReadFile("./testdata/configure_health_check.xml")))
				})
			})

			It("return the l4 load balancer", func() {
				ctx := context.Background()
				expectedL4LoadBalancers := helper.NewTestL4LoadBalancer(testLoadBalancerName)
				gotErr := testNifcloudAPIClient.ConfigureHealthCheck(ctx, &expectedL4LoadBalancers[0])
				Expect(gotErr).ShouldNot(HaveOccurred())
			})
		})

		Describe("the specified l4 load balancer is not existed", func() {
			testLoadBalancerName := "testl4lb"

			BeforeEach(func() {
				handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					lo.Must0(r.ParseForm())
					Expect(r.Form.Get("LoadBalancerName")).Should(Equal(testLoadBalancerName))
					Expect(r.Form.Get("HealthCheck.Target")).Should(Equal("TCP:30000"))
					Expect(r.Form.Get("HealthCheck.UnhealthyThreshold")).Should(Equal("1"))
					Expect(r.Form.Get("HealthCheck.Interval")).Should(Equal("10"))
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = w.Write(lo.Must(os.ReadFile("./testdata/configure_health_check_not_found_load_balancer.xml")))
				})
			})

			It("return error", func() {
				ctx := context.Background()
				expectedL4LoadBalancers := helper.NewTestL4LoadBalancer(testLoadBalancerName)
				gotErr := testNifcloudAPIClient.ConfigureHealthCheck(ctx, &expectedL4LoadBalancers[0])
				Expect(gotErr).Should(HaveOccurred())
			})
		})
	})

	var _ = Describe("SetFilterForLoadBalancer", func() {
		Describe("setting filter is success", func() {
			testLoadBalancerName := "testl4lb"

			BeforeEach(func() {
				handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					lo.Must0(r.ParseForm())
					Expect(r.Form.Get("LoadBalancerName")).Should(Equal(testLoadBalancerName))
					Expect(r.Form.Get("IPAddresses.member.1.IPAddress")).Should(Equal("203.0.113.5"))
					_, _ = w.Write(lo.Must(os.ReadFile("./testdata/set_filter_for_load_balancer.xml")))
				})
			})

			It("return nil", func() {
				ctx := context.Background()
				expectedL4LoadBalancers := helper.NewTestL4LoadBalancer(testLoadBalancerName)
				testFilters := []nifcloud.Filter{
					{
						AddOnFilter: true,
						IPAddress:   "203.0.113.5",
					},
				}
				gotErr := testNifcloudAPIClient.SetFilterForLoadBalancer(ctx, &expectedL4LoadBalancers[0], testFilters)
				Expect(gotErr).ShouldNot(HaveOccurred())
			})
		})

		Describe("the specified l4 load balancer is not existed", func() {
			testLoadBalancerName := "testl4lb"

			BeforeEach(func() {
				handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					lo.Must0(r.ParseForm())
					Expect(r.Form.Get("LoadBalancerName")).Should(Equal(testLoadBalancerName))
					Expect(r.Form.Get("IPAddresses.member.1.IPAddress")).Should(Equal("203.0.113.5"))
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = w.Write(lo.Must(os.ReadFile("./testdata/set_filter_for_load_balancer_not_found_load_balancer.xml")))
				})
			})

			It("return error", func() {
				ctx := context.Background()
				expectedL4LoadBalancers := helper.NewTestL4LoadBalancer(testLoadBalancerName)
				testFilters := []nifcloud.Filter{
					{
						AddOnFilter: true,
						IPAddress:   "203.0.113.5",
					},
				}
				gotErr := testNifcloudAPIClient.SetFilterForLoadBalancer(ctx, &expectedL4LoadBalancers[0], testFilters)
				Expect(gotErr).Should(HaveOccurred())
			})
		})

		Describe("the specified l4 load balancer does not have the port", func() {
			testLoadBalancerName := "testl4lb"

			BeforeEach(func() {
				handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					lo.Must0(r.ParseForm())
					Expect(r.Form.Get("LoadBalancerName")).Should(Equal(testLoadBalancerName))
					Expect(r.Form.Get("IPAddresses.member.1.IPAddress")).Should(Equal("203.0.113.5"))
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = w.Write(lo.Must(os.ReadFile("./testdata/set_filter_for_load_balancer_not_found_port.xml")))
				})
			})

			It("return error", func() {
				ctx := context.Background()
				expectedL4LoadBalancers := helper.NewTestL4LoadBalancer(testLoadBalancerName)
				testFilters := []nifcloud.Filter{
					{
						AddOnFilter: true,
						IPAddress:   "203.0.113.5",
					},
				}
				gotErr := testNifcloudAPIClient.SetFilterForLoadBalancer(ctx, &expectedL4LoadBalancers[0], testFilters)
				Expect(gotErr).Should(HaveOccurred())
			})
		})

		Describe("the specified l4 load balancer already has the filter", func() {
			testLoadBalancerName := "testl4lb"

			BeforeEach(func() {
				handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					lo.Must0(r.ParseForm())
					Expect(r.Form.Get("LoadBalancerName")).Should(Equal(testLoadBalancerName))
					Expect(r.Form.Get("IPAddresses.member.1.IPAddress")).Should(Equal("203.0.113.5"))
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = w.Write(lo.Must(os.ReadFile("./testdata/set_filter_for_load_balancer_duplicate_ip_address.xml")))
				})
			})

			It("return error", func() {
				ctx := context.Background()
				expectedL4LoadBalancers := helper.NewTestL4LoadBalancer(testLoadBalancerName)
				testFilters := []nifcloud.Filter{
					{
						AddOnFilter: true,
						IPAddress:   "203.0.113.5",
					},
				}
				gotErr := testNifcloudAPIClient.SetFilterForLoadBalancer(ctx, &expectedL4LoadBalancers[0], testFilters)
				Expect(gotErr).Should(HaveOccurred())
			})
		})
	})

	var _ = Describe("RegisterInstancesWithLoadBalancer", func() {
		Describe("registering instances is success", func() {
			testLoadBalancerName := "testl4lb"

			BeforeEach(func() {
				handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					lo.Must0(r.ParseForm())
					Expect(r.Form.Get("LoadBalancerName")).Should(Equal(testLoadBalancerName))
					Expect(r.Form.Get("Instances.member.1.InstanceId")).Should(Equal("testinstance2"))
					_, _ = w.Write(lo.Must(os.ReadFile("./testdata/register_instances_with_load_balancer.xml")))
				})
			})

			It("return nil", func() {
				ctx := context.Background()
				expectedL4LoadBalancers := helper.NewTestL4LoadBalancer(testLoadBalancerName)
				testInstances := []nifcloud.Instance{*helper.NewTestInstance()}
				testInstances[0].InstanceID = "testinstance2"
				testInstances[0].InstanceUniqueID = "i-efgh5678"
				gotErr := testNifcloudAPIClient.RegisterInstancesWithLoadBalancer(ctx, &expectedL4LoadBalancers[0], testInstances)
				Expect(gotErr).ShouldNot(HaveOccurred())
			})
		})

		Describe("the specified l4 load balancer is not existed", func() {
			testLoadBalancerName := "testl4lb"

			BeforeEach(func() {
				handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					lo.Must0(r.ParseForm())
					Expect(r.Form.Get("LoadBalancerName")).Should(Equal(testLoadBalancerName))
					Expect(r.Form.Get("Instances.member.1.InstanceId")).Should(Equal("testinstance2"))
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = w.Write(lo.Must(os.ReadFile("./testdata/register_instances_with_load_balancer_not_found_load_balancer.xml")))
				})
			})

			It("return error", func() {
				ctx := context.Background()
				expectedL4LoadBalancers := helper.NewTestL4LoadBalancer(testLoadBalancerName)
				testInstances := []nifcloud.Instance{*helper.NewTestInstance()}
				testInstances[0].InstanceID = "testinstance2"
				testInstances[0].InstanceUniqueID = "i-efgh5678"
				gotErr := testNifcloudAPIClient.RegisterInstancesWithLoadBalancer(ctx, &expectedL4LoadBalancers[0], testInstances)
				Expect(gotErr).Should(HaveOccurred())
			})
		})

		Describe("the specified l4 load balancer does not have the port", func() {
			testLoadBalancerName := "testl4lb"

			BeforeEach(func() {
				handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					lo.Must0(r.ParseForm())
					Expect(r.Form.Get("LoadBalancerName")).Should(Equal(testLoadBalancerName))
					Expect(r.Form.Get("Instances.member.1.InstanceId")).Should(Equal("testinstance2"))
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = w.Write(lo.Must(os.ReadFile("./testdata/register_instances_with_load_balancer_not_found_port.xml")))
				})
			})

			It("return error", func() {
				ctx := context.Background()
				expectedL4LoadBalancers := helper.NewTestL4LoadBalancer(testLoadBalancerName)
				testInstances := []nifcloud.Instance{*helper.NewTestInstance()}
				testInstances[0].InstanceID = "testinstance2"
				testInstances[0].InstanceUniqueID = "i-efgh5678"
				gotErr := testNifcloudAPIClient.RegisterInstancesWithLoadBalancer(ctx, &expectedL4LoadBalancers[0], testInstances)
				Expect(gotErr).Should(HaveOccurred())
			})
		})

		Describe("the specified instance is not existed", func() {
			testLoadBalancerName := "testl4lb"

			BeforeEach(func() {
				handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					lo.Must0(r.ParseForm())
					Expect(r.Form.Get("LoadBalancerName")).Should(Equal(testLoadBalancerName))
					Expect(r.Form.Get("Instances.member.1.InstanceId")).Should(Equal("testinstance2"))
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = w.Write(lo.Must(os.ReadFile("./testdata/register_instances_with_load_balancer_not_found_instances.xml")))
				})
			})

			It("return error", func() {
				ctx := context.Background()
				expectedL4LoadBalancers := helper.NewTestL4LoadBalancer(testLoadBalancerName)
				testInstances := []nifcloud.Instance{*helper.NewTestInstance()}
				testInstances[0].InstanceID = "testinstance2"
				testInstances[0].InstanceUniqueID = "i-efgh5678"
				gotErr := testNifcloudAPIClient.RegisterInstancesWithLoadBalancer(ctx, &expectedL4LoadBalancers[0], testInstances)
				Expect(gotErr).Should(HaveOccurred())
			})
		})

		Describe("the specified instance is already registered", func() {
			testLoadBalancerName := "testl4lb"

			BeforeEach(func() {
				handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					lo.Must0(r.ParseForm())
					Expect(r.Form.Get("LoadBalancerName")).Should(Equal(testLoadBalancerName))
					Expect(r.Form.Get("Instances.member.1.InstanceId")).Should(Equal("testinstance2"))
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = w.Write(lo.Must(os.ReadFile("./testdata/register_instances_with_load_balancer_duplicate.xml")))
				})
			})

			It("return error", func() {
				ctx := context.Background()
				expectedL4LoadBalancers := helper.NewTestL4LoadBalancer(testLoadBalancerName)
				testInstances := []nifcloud.Instance{*helper.NewTestInstance()}
				testInstances[0].InstanceID = "testinstance2"
				testInstances[0].InstanceUniqueID = "i-efgh5678"
				gotErr := testNifcloudAPIClient.RegisterInstancesWithLoadBalancer(ctx, &expectedL4LoadBalancers[0], testInstances)
				Expect(gotErr).Should(HaveOccurred())
			})
		})
	})

	var _ = Describe("DeregisterInstancesFromLoadBalancer", func() {
		Describe("deregistering instances is success", func() {
			testLoadBalancerName := "testl4lb"

			BeforeEach(func() {
				handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					lo.Must0(r.ParseForm())
					Expect(r.Form.Get("LoadBalancerName")).Should(Equal(testLoadBalancerName))
					Expect(r.Form.Get("Instances.member.1.InstanceId")).Should(Equal("testinstance2"))
					_, _ = w.Write(lo.Must(os.ReadFile("./testdata/deregister_instances_from_load_balancer.xml")))
				})
			})

			It("return nil", func() {
				ctx := context.Background()
				expectedL4LoadBalancers := helper.NewTestL4LoadBalancer(testLoadBalancerName)
				testInstances := []nifcloud.Instance{*helper.NewTestInstance()}
				testInstances[0].InstanceID = "testinstance2"
				testInstances[0].InstanceUniqueID = "i-efgh5678"
				gotErr := testNifcloudAPIClient.DeregisterInstancesFromLoadBalancer(ctx, &expectedL4LoadBalancers[0], testInstances)
				Expect(gotErr).ShouldNot(HaveOccurred())
			})
		})

		Describe("the specified l4 load balancer is not existed", func() {
			testLoadBalancerName := "testl4lb"

			BeforeEach(func() {
				handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					lo.Must0(r.ParseForm())
					Expect(r.Form.Get("LoadBalancerName")).Should(Equal(testLoadBalancerName))
					Expect(r.Form.Get("Instances.member.1.InstanceId")).Should(Equal("testinstance2"))
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = w.Write(lo.Must(os.ReadFile("./testdata/register_instances_with_load_balancer_not_found_load_balancer.xml")))
				})
			})

			It("return error", func() {
				ctx := context.Background()
				expectedL4LoadBalancers := helper.NewTestL4LoadBalancer(testLoadBalancerName)
				testInstances := []nifcloud.Instance{*helper.NewTestInstance()}
				testInstances[0].InstanceID = "testinstance2"
				testInstances[0].InstanceUniqueID = "i-efgh5678"
				gotErr := testNifcloudAPIClient.DeregisterInstancesFromLoadBalancer(ctx, &expectedL4LoadBalancers[0], testInstances)
				Expect(gotErr).Should(HaveOccurred())
			})
		})

		Describe("the specified l4 load balancer does not have the port", func() {
			testLoadBalancerName := "testl4lb"

			BeforeEach(func() {
				handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					lo.Must0(r.ParseForm())
					Expect(r.Form.Get("LoadBalancerName")).Should(Equal(testLoadBalancerName))
					Expect(r.Form.Get("Instances.member.1.InstanceId")).Should(Equal("testinstance2"))
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = w.Write(lo.Must(os.ReadFile("./testdata/deregister_instances_from_load_balancer_not_found_port.xml")))
				})
			})

			It("return error", func() {
				ctx := context.Background()
				expectedL4LoadBalancers := helper.NewTestL4LoadBalancer(testLoadBalancerName)
				testInstances := []nifcloud.Instance{*helper.NewTestInstance()}
				testInstances[0].InstanceID = "testinstance2"
				testInstances[0].InstanceUniqueID = "i-efgh5678"
				gotErr := testNifcloudAPIClient.DeregisterInstancesFromLoadBalancer(ctx, &expectedL4LoadBalancers[0], testInstances)
				Expect(gotErr).Should(HaveOccurred())
			})
		})

		Describe("the specified instance is not existed", func() {
			testLoadBalancerName := "testl4lb"

			BeforeEach(func() {
				handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					lo.Must0(r.ParseForm())
					Expect(r.Form.Get("LoadBalancerName")).Should(Equal(testLoadBalancerName))
					Expect(r.Form.Get("Instances.member.1.InstanceId")).Should(Equal("testinstance2"))
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = w.Write(lo.Must(os.ReadFile("./testdata/deregister_instances_from_load_balancer_not_found_instances.xml")))
				})
			})

			It("return error", func() {
				ctx := context.Background()
				expectedL4LoadBalancers := helper.NewTestL4LoadBalancer(testLoadBalancerName)
				testInstances := []nifcloud.Instance{*helper.NewTestInstance()}
				testInstances[0].InstanceID = "testinstance2"
				testInstances[0].InstanceUniqueID = "i-efgh5678"
				gotErr := testNifcloudAPIClient.DeregisterInstancesFromLoadBalancer(ctx, &expectedL4LoadBalancers[0], testInstances)
				Expect(gotErr).Should(HaveOccurred())
			})
		})
	})

	var _ = Describe("DeleteLoadBalancer", func() {
		Describe("deleting l4 load balancer is success", func() {
			testLoadBalancerName := "testl4lb"

			BeforeEach(func() {
				handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					lo.Must0(r.ParseForm())
					Expect(r.Form.Get("LoadBalancerName")).Should(Equal(testLoadBalancerName))
					Expect(r.Form.Get("LoadBalancerPort")).Should(Equal("80"))
					Expect(r.Form.Get("InstancePort")).Should(Equal("30000"))
					_, _ = w.Write(lo.Must(os.ReadFile("./testdata/delete_load_balancer.xml")))
				})
			})

			It("return nil", func() {
				ctx := context.Background()
				expectedL4LoadBalancers := helper.NewTestL4LoadBalancer(testLoadBalancerName)
				gotErr := testNifcloudAPIClient.DeleteLoadBalancer(ctx, &expectedL4LoadBalancers[0])
				Expect(gotErr).ShouldNot(HaveOccurred())
			})
		})

		Describe("the specified l4 load balancer is not existed", func() {
			testLoadBalancerName := "testl4lb"

			BeforeEach(func() {
				handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					lo.Must0(r.ParseForm())
					Expect(r.Form.Get("LoadBalancerName")).Should(Equal(testLoadBalancerName))
					Expect(r.Form.Get("LoadBalancerPort")).Should(Equal("80"))
					Expect(r.Form.Get("InstancePort")).Should(Equal("30000"))
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = w.Write(lo.Must(os.ReadFile("./testdata/delete_load_balancer_not_found_load_balancer.xml")))
				})
			})

			It("return error", func() {
				ctx := context.Background()
				expectedL4LoadBalancers := helper.NewTestL4LoadBalancer(testLoadBalancerName)
				gotErr := testNifcloudAPIClient.DeleteLoadBalancer(ctx, &expectedL4LoadBalancers[0])
				Expect(gotErr).Should(HaveOccurred())
			})
		})

		Describe("the specified l4 load balancer does not have the port", func() {
			testLoadBalancerName := "testl4lb"

			BeforeEach(func() {
				handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					lo.Must0(r.ParseForm())
					Expect(r.Form.Get("LoadBalancerName")).Should(Equal(testLoadBalancerName))
					Expect(r.Form.Get("LoadBalancerPort")).Should(Equal("80"))
					Expect(r.Form.Get("InstancePort")).Should(Equal("30000"))
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = w.Write(lo.Must(os.ReadFile("./testdata/delete_load_balancer_not_found_port.xml")))
				})
			})

			It("return error", func() {
				ctx := context.Background()
				expectedL4LoadBalancers := helper.NewTestL4LoadBalancer(testLoadBalancerName)
				gotErr := testNifcloudAPIClient.DeleteLoadBalancer(ctx, &expectedL4LoadBalancers[0])
				Expect(gotErr).Should(HaveOccurred())
			})
		})
	})

	var _ = Describe("DescribeElasticLoadBalancers", func() {
		Describe("given elastic load balancer is existed", func() {
			testLoadBalancerName := "testelb"

			BeforeEach(func() {
				handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					lo.Must0(r.ParseForm())
					Expect(r.Form.Get("ElasticLoadBalancers.ElasticLoadBalancerName.1")).Should(Equal(testLoadBalancerName))
					_, _ = w.Write(lo.Must(os.ReadFile("./testdata/describe_elastic_load_balancers.xml")))
				})
			})

			It("return the elastic load balancer", func() {
				ctx := context.Background()
				expectedElasticLoadBalancers := helper.NewTestElasticLoadBalancer(testLoadBalancerName)
				expectedElasticLoadBalancers[0].VIP = "203.0.113.5"
				expectedElasticLoadBalancers[0].BalancingTargets[0].InstanceType = ""
				expectedElasticLoadBalancers[0].BalancingTargets[0].PublicIPAddress = ""
				expectedElasticLoadBalancers[0].BalancingTargets[0].PrivateIPAddress = ""
				expectedElasticLoadBalancers[0].BalancingTargets[0].Zone = ""
				expectedElasticLoadBalancers[0].BalancingTargets[0].State = ""
				expectedElasticLoadBalancers[0].NetworkInterfaces[0].IPAddress = "203.0.113.5"
				expectedElasticLoadBalancers[0].NetworkInterfaces[0].SystemIpAddresses = []string{"203.0.113.6", "203.0.113.7"}
				gotElasticLoadBalancer, gotErr := testNifcloudAPIClient.DescribeElasticLoadBalancers(ctx, testLoadBalancerName)
				Expect(gotErr).ShouldNot(HaveOccurred())
				Expect(gotElasticLoadBalancer).Should(Equal(expectedElasticLoadBalancers))
			})
		})

		Describe("given elastic load balancer is existed and it has two ports and network interfaces", func() {
			testLoadBalancerName := "testelb"

			BeforeEach(func() {
				handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					lo.Must0(r.ParseForm())
					Expect(r.Form.Get("ElasticLoadBalancers.ElasticLoadBalancerName.1")).Should(Equal(testLoadBalancerName))
					_, _ = w.Write(lo.Must(os.ReadFile("./testdata/describe_elastic_load_balancers_two_port_and_network_interfaces.xml")))
				})
			})

			It("return the elastic load balancer", func() {
				ctx := context.Background()
				networkInterfaces := []nifcloud.NetworkInterface{
					{
						NetworkId:         "net-xxxx1111",
						NetworkName:       "testlan",
						IPAddress:         "172.16.0.1",
						SystemIpAddresses: []string{"172.16.0.2", "172.16.0.3"},
						IsVipNetwork:      false,
					},
					{
						NetworkId:         "net-COMMON_GLOBAL",
						NetworkName:       "",
						IPAddress:         "203.0.113.5",
						SystemIpAddresses: []string{"203.0.113.6", "203.0.113.7"},
						IsVipNetwork:      true,
					},
				}
				expectedElasticLoadBalancers := helper.NewTestElasticLoadBalancerWithTwoPort(testLoadBalancerName)
				for i := range expectedElasticLoadBalancers {
					expectedElasticLoadBalancers[i].VIP = "203.0.113.5"
					expectedElasticLoadBalancers[i].BalancingTargets[0].InstanceType = ""
					expectedElasticLoadBalancers[i].BalancingTargets[0].PublicIPAddress = ""
					expectedElasticLoadBalancers[i].BalancingTargets[0].PrivateIPAddress = ""
					expectedElasticLoadBalancers[i].BalancingTargets[0].Zone = ""
					expectedElasticLoadBalancers[i].BalancingTargets[0].State = ""
					expectedElasticLoadBalancers[i].NetworkInterfaces = networkInterfaces
				}
				gotElasticLoadBalancer, gotErr := testNifcloudAPIClient.DescribeElasticLoadBalancers(ctx, testLoadBalancerName)
				Expect(gotErr).ShouldNot(HaveOccurred())
				Expect(gotElasticLoadBalancer).Should(Equal(expectedElasticLoadBalancers))
			})
		})

		Describe("given elastic load balancer is not existed", func() {
			testLoadBalancerName := "testelb"

			BeforeEach(func() {
				handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					lo.Must0(r.ParseForm())
					Expect(r.Form.Get("ElasticLoadBalancers.ElasticLoadBalancerName.1")).Should(Equal(testLoadBalancerName))
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = w.Write(lo.Must(os.ReadFile("./testdata/describe_elastic_load_balancers_not_found_load_balancer.xml")))
				})
			})

			It("return error", func() {
				ctx := context.Background()
				gotElasticLoadBalancer, gotErr := testNifcloudAPIClient.DescribeElasticLoadBalancers(ctx, testLoadBalancerName)
				Expect(gotErr).Should(HaveOccurred())
				Expect(nifcloud.IsAPIError(gotErr, nifcloud.ExportErrorCodeElasticLoadBalancerNotFound)).Should(BeTrue())
				Expect(gotElasticLoadBalancer).Should(BeNil())
			})
		})
	})

	var _ = Describe("createElasticLoadBalancer", func() {
		Describe("creating elastic load balancer is success", func() {
			testLoadBalancerName := "testelb"

			BeforeEach(func() {
				handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					lo.Must0(r.ParseForm())
					Expect(r.Form.Get("ElasticLoadBalancerName")).Should(Equal(testLoadBalancerName))
					Expect(r.Form.Get("AccountingType")).Should(Equal("1"))
					Expect(r.Form.Get("NetworkVolume")).Should(Equal("100"))
					Expect(r.Form.Get("AvailabilityZones.member.1")).Should(Equal("east-11"))
					Expect(r.Form.Get("Listeners.member.1.ElasticLoadBalancerPort")).Should(Equal("80"))
					Expect(r.Form.Get("Listeners.member.1.InstancePort")).Should(Equal("30000"))
					Expect(r.Form.Get("Listeners.member.1.Protocol")).Should(Equal("TCP"))
					Expect(r.Form.Get("Listeners.member.1.BalancingType")).Should(Equal("1"))
					Expect(r.Form.Get("NetworkInterface.1.NetworkId")).Should(Equal("net-COMMON_GLOBAL"))
					Expect(r.Form.Get("NetworkInterface.1.IsVipNetwork")).Should(Equal("true"))
					_, _ = w.Write(lo.Must(os.ReadFile("./testdata/create_elastic_load_balancer.xml")))
				})
			})

			It("return nil", func() {
				ctx := context.Background()
				testElasticLoadBalancers := helper.NewTestElasticLoadBalancer(testLoadBalancerName)
				gotDNSName, gotErr := nifcloud.ExportCreateElasticLoadBalancer(testNifcloudAPIClient, ctx, &testElasticLoadBalancers[0])
				Expect(gotErr).ShouldNot(HaveOccurred())
				Expect(gotDNSName).Should(BeEmpty())
			})
		})

		Describe("the specified elastic load balancer is already existed", func() {
			testLoadBalancerName := "testelb"

			BeforeEach(func() {
				handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					lo.Must0(r.ParseForm())
					Expect(r.Form.Get("ElasticLoadBalancerName")).Should(Equal(testLoadBalancerName))
					Expect(r.Form.Get("AccountingType")).Should(Equal("1"))
					Expect(r.Form.Get("NetworkVolume")).Should(Equal("100"))
					Expect(r.Form.Get("AvailabilityZones.member.1")).Should(Equal("east-11"))
					Expect(r.Form.Get("Listeners.member.1.ElasticLoadBalancerPort")).Should(Equal("80"))
					Expect(r.Form.Get("Listeners.member.1.InstancePort")).Should(Equal("30000"))
					Expect(r.Form.Get("Listeners.member.1.Protocol")).Should(Equal("TCP"))
					Expect(r.Form.Get("Listeners.member.1.BalancingType")).Should(Equal("1"))
					Expect(r.Form.Get("NetworkInterface.1.NetworkId")).Should(Equal("net-COMMON_GLOBAL"))
					Expect(r.Form.Get("NetworkInterface.1.IsVipNetwork")).Should(Equal("true"))
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = w.Write(lo.Must(os.ReadFile("./testdata/create_elastic_load_balancer_duplicate.xml")))
				})
			})

			It("return error", func() {
				ctx := context.Background()
				testElasticLoadBalancers := helper.NewTestElasticLoadBalancer(testLoadBalancerName)
				gotDNSName, gotErr := nifcloud.ExportCreateElasticLoadBalancer(testNifcloudAPIClient, ctx, &testElasticLoadBalancers[0])
				Expect(gotErr).Should(HaveOccurred())
				Expect(gotDNSName).Should(BeEmpty())
			})
		})
	})
})
