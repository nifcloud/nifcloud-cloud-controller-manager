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
		testNifcloudAPIClient nifcloud.CloudAPIClient
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
})
