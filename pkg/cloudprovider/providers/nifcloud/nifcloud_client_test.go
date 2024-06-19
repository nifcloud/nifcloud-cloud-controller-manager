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

})
