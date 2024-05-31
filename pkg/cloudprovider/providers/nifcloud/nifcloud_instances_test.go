package nifcloud_test

import (
	"context"

	"github.com/nifcloud/nifcloud-cloud-controller-manager/pkg/cloudprovider/providers/nifcloud"
	"github.com/nifcloud/nifcloud-cloud-controller-manager/test/helper"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("NodeAddresses", func() {
	var ctrl *gomock.Controller
	var region string = "east1"

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("single instance is existed", func() {
		It("return the NodeAddress", func() {
			ctx := context.Background()
			testInstances := []nifcloud.Instance{*helper.NewTestInstance()}
			nodeName := testInstances[0].InstanceID
			expectedNodeAddress := []v1.NodeAddress{
				{
					Type:    v1.NodeExternalIP,
					Address: testInstances[0].PublicIPAddress,
				},
				{
					Type:    v1.NodeInternalIP,
					Address: testInstances[0].PrivateIPAddress,
				},
			}

			c := nifcloud.NewMockCloudAPIClient(ctrl)
			c.EXPECT().
				DescribeInstancesByInstanceID(gomock.Any(), []string{nodeName}).
				Return(testInstances, nil).
				Times(1)

			cloud := &nifcloud.Cloud{}
			cloud.SetClient(c)
			cloud.SetRegion(region)

			nodeAddress, err := cloud.NodeAddresses(ctx, types.NodeName(nodeName))
			Expect(err).ShouldNot(HaveOccurred())
			Expect(nodeAddress).Should(Equal(expectedNodeAddress))
		})
	})

	Context("the instance is not existed", func() {
		It("return error", func() {
			ctx := context.Background()
			testInstances := []nifcloud.Instance{}
			nodeName := "testinstance"

			notFoundErr := helper.NewMockAPIError(nifcloud.ExportErrorCodeInstanceNotFound)
			c := nifcloud.NewMockCloudAPIClient(ctrl)
			c.EXPECT().
				DescribeInstancesByInstanceID(gomock.Any(), []string{nodeName}).
				Return(testInstances, notFoundErr).
				Times(1)

			cloud := &nifcloud.Cloud{}
			cloud.SetClient(c)
			cloud.SetRegion(region)

			nodeAddress, err := cloud.NodeAddresses(ctx, types.NodeName(nodeName))
			Expect(err).Should(HaveOccurred())
			Expect(nodeAddress).Should(BeNil())
		})
	})

	Context("some instances have same InstanceID are existed", func() {
		It("return error", func() {
			ctx := context.Background()
			testInstances := []nifcloud.Instance{*helper.NewTestInstance(), *helper.NewTestInstance()}
			nodeName := testInstances[0].InstanceID

			c := nifcloud.NewMockCloudAPIClient(ctrl)
			c.EXPECT().
				DescribeInstancesByInstanceID(gomock.Any(), []string{nodeName}).
				Return(testInstances, nil).
				Times(1)

			cloud := &nifcloud.Cloud{}
			cloud.SetClient(c)
			cloud.SetRegion(region)

			nodeAddress, err := cloud.NodeAddresses(ctx, types.NodeName(nodeName))
			Expect(err).Should(HaveOccurred())
			Expect(nodeAddress).Should(BeNil())
		})
	})
})

var _ = Describe("NodeAddressesByProviderID", func() {
	var ctrl *gomock.Controller
	var region string = "east1"

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("single instance is existed", func() {
		It("return the NodeAddress", func() {
			ctx := context.Background()
			testInstances := []nifcloud.Instance{*helper.NewTestInstance()}
			testProviderID := "nifcloud:///east-11/i-abcd1234"
			testInstanceUniqueID := "i-abcd1234"
			expectedNodeAddress := []v1.NodeAddress{
				{
					Type:    v1.NodeExternalIP,
					Address: testInstances[0].PublicIPAddress,
				},
				{
					Type:    v1.NodeInternalIP,
					Address: testInstances[0].PrivateIPAddress,
				},
			}

			c := nifcloud.NewMockCloudAPIClient(ctrl)
			c.EXPECT().
				DescribeInstancesByInstanceUniqueID(gomock.Any(), []string{testInstanceUniqueID}).
				Return(testInstances, nil).
				Times(1)

			cloud := &nifcloud.Cloud{}
			cloud.SetClient(c)
			cloud.SetRegion(region)

			nodeAddress, err := cloud.NodeAddressesByProviderID(ctx, testProviderID)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(nodeAddress).Should(Equal(expectedNodeAddress))
		})
	})

	Context("the instance is not existed", func() {
		It("return error", func() {
			ctx := context.Background()
			testInstances := []nifcloud.Instance{}
			testProviderID := "nifcloud:///east-11/i-abcd1234"
			testInstanceUniqueID := "i-abcd1234"

			notFoundErr := helper.NewMockAPIError(nifcloud.ExportErrorCodeInstanceNotFound)
			c := nifcloud.NewMockCloudAPIClient(ctrl)
			c.EXPECT().
				DescribeInstancesByInstanceUniqueID(gomock.Any(), []string{testInstanceUniqueID}).
				Return(testInstances, notFoundErr).
				Times(1)

			cloud := &nifcloud.Cloud{}
			cloud.SetClient(c)
			cloud.SetRegion(region)

			nodeAddress, err := cloud.NodeAddressesByProviderID(ctx, testProviderID)
			Expect(err).Should(HaveOccurred())
			Expect(nodeAddress).Should(BeNil())
		})
	})

	Context("some instances have same InstanceID are existed", func() {
		It("return error", func() {
			ctx := context.Background()
			testInstances := []nifcloud.Instance{*helper.NewTestInstance(), *helper.NewTestInstance()}
			testProviderID := "nifcloud:///east-11/i-abcd1234"
			testInstanceUniqueID := "i-abcd1234"

			c := nifcloud.NewMockCloudAPIClient(ctrl)
			c.EXPECT().
				DescribeInstancesByInstanceUniqueID(gomock.Any(), []string{testInstanceUniqueID}).
				Return(testInstances, nil).
				Times(1)

			cloud := &nifcloud.Cloud{}
			cloud.SetClient(c)
			cloud.SetRegion(region)

			nodeAddress, err := cloud.NodeAddressesByProviderID(ctx, testProviderID)
			Expect(err).Should(HaveOccurred())
			Expect(nodeAddress).Should(BeNil())
		})
	})
})
