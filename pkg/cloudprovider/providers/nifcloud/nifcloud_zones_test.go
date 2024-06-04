package nifcloud_test

import (
	"context"
	"os"

	"github.com/nifcloud/nifcloud-cloud-controller-manager/pkg/cloudprovider/providers/nifcloud"
	"github.com/nifcloud/nifcloud-cloud-controller-manager/test/helper"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	cloudprovider "k8s.io/cloud-provider"
)

var _ = Describe("GetZone", func() {
	var ctrl *gomock.Controller
	var region string = "east1"

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("instance id is set in NODE_NAME environment variable", func() {
		BeforeEach(func() {
			err := os.Unsetenv("NODE_NAME")
			Expect(err).NotTo(HaveOccurred())
		})

		It("return the error", func() {
			ctx := context.Background()
			expectedZone := cloudprovider.Zone{}

			cloud := &nifcloud.Cloud{}

			gotZone, err := cloud.GetZone(ctx)
			Expect(err).Should(HaveOccurred())
			Expect(gotZone).Should(Equal(expectedZone))
		})
	})

	Context("instance id is set in NODE_NAME environment variable", func() {
		BeforeEach(func() {
			err := os.Setenv("NODE_NAME", "testinstance")
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			err := os.Unsetenv("NODE_NAME")
			Expect(err).NotTo(HaveOccurred())
		})

		Context("single instance is existed", func() {
			It("return the Zone", func() {
				ctx := context.Background()
				testInstances := []nifcloud.Instance{*helper.NewTestInstance()}
				nodeName := testInstances[0].InstanceID
				expectedZone := cloudprovider.Zone{
					FailureDomain: testInstances[0].Zone,
					Region:        region,
				}

				c := nifcloud.NewMockCloudAPIClient(ctrl)
				c.EXPECT().
					DescribeInstancesByInstanceID(gomock.Any(), []string{nodeName}).
					Return(testInstances, nil).
					Times(1)

				cloud := &nifcloud.Cloud{}
				cloud.SetClient(c)
				cloud.SetRegion(region)

				gotZone, err := cloud.GetZone(ctx)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(gotZone).Should(Equal(expectedZone))
			})
		})

		Context("the instance is not existed", func() {
			It("return error", func() {
				ctx := context.Background()
				testInstances := []nifcloud.Instance{}
				nodeName := "testinstance"
				expectedZone := cloudprovider.Zone{}

				notFoundErr := helper.NewMockAPIError(nifcloud.ExportErrorCodeInstanceNotFound)
				c := nifcloud.NewMockCloudAPIClient(ctrl)
				c.EXPECT().
					DescribeInstancesByInstanceID(gomock.Any(), []string{nodeName}).
					Return(testInstances, notFoundErr).
					Times(1)

				cloud := &nifcloud.Cloud{}
				cloud.SetClient(c)
				cloud.SetRegion(region)

				gotZone, err := cloud.GetZone(ctx)
				Expect(err).Should(HaveOccurred())
				Expect(gotZone).Should(Equal(expectedZone))
			})
		})

		Context("some instances have same InstanceID are existed", func() {
			It("return error", func() {
				ctx := context.Background()
				testInstances := []nifcloud.Instance{*helper.NewTestInstance(), *helper.NewTestInstance()}
				nodeName := testInstances[0].InstanceID
				expectedZone := cloudprovider.Zone{}

				c := nifcloud.NewMockCloudAPIClient(ctrl)
				c.EXPECT().
					DescribeInstancesByInstanceID(gomock.Any(), []string{nodeName}).
					Return(testInstances, nil).
					Times(1)

				cloud := &nifcloud.Cloud{}
				cloud.SetClient(c)
				cloud.SetRegion(region)

				gotZone, err := cloud.GetZone(ctx)
				Expect(err).Should(HaveOccurred())
				Expect(gotZone).Should(Equal(expectedZone))
			})
		})
	})
})
