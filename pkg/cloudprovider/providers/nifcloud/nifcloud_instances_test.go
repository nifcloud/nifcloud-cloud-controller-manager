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
	cloudprovider "k8s.io/cloud-provider"
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

			notFoundErr := cloudprovider.InstanceNotFound
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

			notFoundErr := cloudprovider.InstanceNotFound
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

var _ = Describe("InstanceID", func() {
	var ctrl *gomock.Controller
	var region string = "east1"

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("single instance is existed", func() {
		It("return the InstanceID", func() {
			ctx := context.Background()
			testInstances := []nifcloud.Instance{*helper.NewTestInstance()}
			nodeName := testInstances[0].InstanceID
			expectedInstanceID := "/east-11/i-abcd1234"

			c := nifcloud.NewMockCloudAPIClient(ctrl)
			c.EXPECT().
				DescribeInstancesByInstanceID(gomock.Any(), []string{nodeName}).
				Return(testInstances, nil).
				Times(1)

			cloud := &nifcloud.Cloud{}
			cloud.SetClient(c)
			cloud.SetRegion(region)

			gotInstanceID, err := cloud.InstanceID(ctx, types.NodeName(nodeName))
			Expect(err).ShouldNot(HaveOccurred())
			Expect(gotInstanceID).Should(Equal(expectedInstanceID))
		})
	})

	Context("the instance is not existed", func() {
		It("return error", func() {
			ctx := context.Background()
			testInstances := []nifcloud.Instance{}
			nodeName := "testinstance"

			notFoundErr := cloudprovider.InstanceNotFound
			c := nifcloud.NewMockCloudAPIClient(ctrl)
			c.EXPECT().
				DescribeInstancesByInstanceID(gomock.Any(), []string{nodeName}).
				Return(testInstances, notFoundErr).
				Times(1)

			cloud := &nifcloud.Cloud{}
			cloud.SetClient(c)
			cloud.SetRegion(region)

			gotInstanceID, err := cloud.InstanceID(ctx, types.NodeName(nodeName))
			Expect(err).Should(HaveOccurred())
			Expect(gotInstanceID).Should(BeEmpty())
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

			gotInstanceID, err := cloud.InstanceID(ctx, types.NodeName(nodeName))
			Expect(err).Should(HaveOccurred())
			Expect(gotInstanceID).Should(BeEmpty())
		})
	})
})

var _ = Describe("InstanceType", func() {
	var ctrl *gomock.Controller
	var region string = "east1"

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("single instance is existed", func() {
		It("return the InstanceType", func() {
			ctx := context.Background()
			testInstances := []nifcloud.Instance{*helper.NewTestInstance()}
			nodeName := testInstances[0].InstanceID

			c := nifcloud.NewMockCloudAPIClient(ctrl)
			c.EXPECT().
				DescribeInstancesByInstanceID(gomock.Any(), []string{nodeName}).
				Return(testInstances, nil).
				Times(1)

			cloud := &nifcloud.Cloud{}
			cloud.SetClient(c)
			cloud.SetRegion(region)

			gotInstanceType, err := cloud.InstanceType(ctx, types.NodeName(nodeName))
			Expect(err).ShouldNot(HaveOccurred())
			Expect(gotInstanceType).Should(Equal(testInstances[0].InstanceType))
		})
	})

	Context("the instance is not existed", func() {
		It("return error", func() {
			ctx := context.Background()
			testInstances := []nifcloud.Instance{}
			nodeName := "testinstance"

			notFoundErr := cloudprovider.InstanceNotFound
			c := nifcloud.NewMockCloudAPIClient(ctrl)
			c.EXPECT().
				DescribeInstancesByInstanceID(gomock.Any(), []string{nodeName}).
				Return(testInstances, notFoundErr).
				Times(1)

			cloud := &nifcloud.Cloud{}
			cloud.SetClient(c)
			cloud.SetRegion(region)

			gotInstanceType, err := cloud.InstanceType(ctx, types.NodeName(nodeName))
			Expect(err).Should(HaveOccurred())
			Expect(gotInstanceType).Should(BeEmpty())
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

			gotInstanceType, err := cloud.InstanceType(ctx, types.NodeName(nodeName))
			Expect(err).Should(HaveOccurred())
			Expect(gotInstanceType).Should(BeEmpty())
		})
	})
})

var _ = Describe("InstanceTypeByProviderID", func() {
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

			c := nifcloud.NewMockCloudAPIClient(ctrl)
			c.EXPECT().
				DescribeInstancesByInstanceUniqueID(gomock.Any(), []string{testInstanceUniqueID}).
				Return(testInstances, nil).
				Times(1)

			cloud := &nifcloud.Cloud{}
			cloud.SetClient(c)
			cloud.SetRegion(region)

			gotInstanceType, err := cloud.InstanceTypeByProviderID(ctx, testProviderID)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(gotInstanceType).Should(Equal(testInstances[0].InstanceType))
		})
	})

	Context("the instance is not existed", func() {
		It("return error", func() {
			ctx := context.Background()
			testInstances := []nifcloud.Instance{}
			testProviderID := "nifcloud:///east-11/i-abcd1234"
			testInstanceUniqueID := "i-abcd1234"

			notFoundErr := cloudprovider.InstanceNotFound
			c := nifcloud.NewMockCloudAPIClient(ctrl)
			c.EXPECT().
				DescribeInstancesByInstanceUniqueID(gomock.Any(), []string{testInstanceUniqueID}).
				Return(testInstances, notFoundErr).
				Times(1)

			cloud := &nifcloud.Cloud{}
			cloud.SetClient(c)
			cloud.SetRegion(region)

			nodeAddress, err := cloud.InstanceTypeByProviderID(ctx, testProviderID)
			Expect(err).Should(HaveOccurred())
			Expect(nodeAddress).Should(BeEmpty())
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

			nodeAddress, err := cloud.InstanceTypeByProviderID(ctx, testProviderID)
			Expect(err).Should(HaveOccurred())
			Expect(nodeAddress).Should(BeEmpty())
		})
	})
})

var _ = Describe("InstanceExistsByProviderID", func() {
	var ctrl *gomock.Controller
	var region string = "east1"

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("single instance is existed", func() {
		It("return true", func() {
			ctx := context.Background()
			testInstances := []nifcloud.Instance{*helper.NewTestInstance()}
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

			exist, err := cloud.InstanceExistsByProviderID(ctx, testProviderID)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(exist).Should(BeTrue())
		})
	})

	Context("the instance is not existed", func() {
		It("return false and error", func() {
			ctx := context.Background()
			testInstances := []nifcloud.Instance{}
			testProviderID := "nifcloud:///east-11/i-abcd1234"
			testInstanceUniqueID := "i-abcd1234"

			notFoundErr := cloudprovider.InstanceNotFound
			c := nifcloud.NewMockCloudAPIClient(ctrl)
			c.EXPECT().
				DescribeInstancesByInstanceUniqueID(gomock.Any(), []string{testInstanceUniqueID}).
				Return(testInstances, notFoundErr).
				Times(1)

			cloud := &nifcloud.Cloud{}
			cloud.SetClient(c)
			cloud.SetRegion(region)

			exist, err := cloud.InstanceExistsByProviderID(ctx, testProviderID)
			Expect(err).Should(HaveOccurred())
			Expect(exist).Should(BeFalse())
		})
	})

	Context("some instances have same InstanceID are existed", func() {
		It("return false and error", func() {
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

			exist, err := cloud.InstanceExistsByProviderID(ctx, testProviderID)
			Expect(err).Should(HaveOccurred())
			Expect(exist).Should(BeFalse())
		})
	})
})

var _ = Describe("InstanceExists", func() {
	var ctrl *gomock.Controller
	var region string = "east1"

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("single instance is existed", func() {
		It("return true", func() {
			ctx := context.Background()
			testInstances := []nifcloud.Instance{*helper.NewTestInstance()}
			testProviderID := "nifcloud:///east-11/i-abcd1234"
			testInstanceUniqueID := "i-abcd1234"
			testNode := &v1.Node{
				Spec: v1.NodeSpec{
					ProviderID: testProviderID,
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

			exist, err := cloud.InstanceExists(ctx, testNode)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(exist).Should(BeTrue())
		})
	})

	Context("the instance is not existed", func() {
		It("return false and error", func() {
			ctx := context.Background()
			testInstances := []nifcloud.Instance{}
			testProviderID := "nifcloud:///east-11/i-abcd1234"
			testInstanceUniqueID := "i-abcd1234"
			testNode := &v1.Node{
				Spec: v1.NodeSpec{
					ProviderID: testProviderID,
				},
			}

			notFoundErr := cloudprovider.InstanceNotFound
			c := nifcloud.NewMockCloudAPIClient(ctrl)
			c.EXPECT().
				DescribeInstancesByInstanceUniqueID(gomock.Any(), []string{testInstanceUniqueID}).
				Return(testInstances, notFoundErr).
				Times(1)

			cloud := &nifcloud.Cloud{}
			cloud.SetClient(c)
			cloud.SetRegion(region)

			exist, err := cloud.InstanceExists(ctx, testNode)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(exist).Should(BeFalse())
		})
	})

	Context("some instances have same InstanceID are existed", func() {
		It("return false and error", func() {
			ctx := context.Background()
			testInstances := []nifcloud.Instance{*helper.NewTestInstance(), *helper.NewTestInstance()}
			testProviderID := "nifcloud:///east-11/i-abcd1234"
			testInstanceUniqueID := "i-abcd1234"
			testNode := &v1.Node{
				Spec: v1.NodeSpec{
					ProviderID: testProviderID,
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

			exist, err := cloud.InstanceExists(ctx, testNode)
			Expect(err).Should(HaveOccurred())
			Expect(exist).Should(BeFalse())
		})
	})

	Context("DescribeInstances return unknown error", func() {
		It("return false and the error", func() {
			ctx := context.Background()
			testInstances := []nifcloud.Instance{}
			testProviderID := "nifcloud:///east-11/i-abcd1234"
			testInstanceUniqueID := "i-abcd1234"
			testNode := &v1.Node{
				Spec: v1.NodeSpec{
					ProviderID: testProviderID,
				},
			}

			unknownErr := helper.NewMockAPIError("client.unknown")
			c := nifcloud.NewMockCloudAPIClient(ctrl)
			c.EXPECT().
				DescribeInstancesByInstanceUniqueID(gomock.Any(), []string{testInstanceUniqueID}).
				Return(testInstances, unknownErr).
				Times(1)

			cloud := &nifcloud.Cloud{}
			cloud.SetClient(c)
			cloud.SetRegion(region)

			exist, err := cloud.InstanceExists(ctx, testNode)
			Expect(err).Should(HaveOccurred())
			Expect(exist).Should(BeFalse())
		})
	})
})

var _ = Describe("InstanceShutdown", func() {
	var ctrl *gomock.Controller
	var region string = "east1"

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("single instance is existed", func() {
		Context("the instance is stopped", func() {
			It("return true", func() {
				ctx := context.Background()
				testInstances := []nifcloud.Instance{*helper.NewTestInstance()}
				testInstances[0].State = "stopped"
				testProviderID := "nifcloud:///east-11/i-abcd1234"
				testInstanceUniqueID := "i-abcd1234"
				testNode := &v1.Node{
					Spec: v1.NodeSpec{
						ProviderID: testProviderID,
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

				stopped, err := cloud.InstanceShutdown(ctx, testNode)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(stopped).Should(BeTrue())
			})
		})

		Context("the instance is running", func() {
			It("return false", func() {
				ctx := context.Background()
				testInstances := []nifcloud.Instance{*helper.NewTestInstance()}
				testInstances[0].State = "running"
				testProviderID := "nifcloud:///east-11/i-abcd1234"
				testInstanceUniqueID := "i-abcd1234"
				testNode := &v1.Node{
					Spec: v1.NodeSpec{
						ProviderID: testProviderID,
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

				stopped, err := cloud.InstanceShutdown(ctx, testNode)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(stopped).Should(BeFalse())
			})
		})
	})

	Context("the instance is not existed", func() {
		It("return false and error", func() {
			ctx := context.Background()
			testInstances := []nifcloud.Instance{}
			testProviderID := "nifcloud:///east-11/i-abcd1234"
			testInstanceUniqueID := "i-abcd1234"
			testNode := &v1.Node{
				Spec: v1.NodeSpec{
					ProviderID: testProviderID,
				},
			}

			notFoundErr := cloudprovider.InstanceNotFound
			c := nifcloud.NewMockCloudAPIClient(ctrl)
			c.EXPECT().
				DescribeInstancesByInstanceUniqueID(gomock.Any(), []string{testInstanceUniqueID}).
				Return(testInstances, notFoundErr).
				Times(1)

			cloud := &nifcloud.Cloud{}
			cloud.SetClient(c)
			cloud.SetRegion(region)

			stopped, err := cloud.InstanceShutdown(ctx, testNode)
			Expect(err).Should(HaveOccurred())
			Expect(stopped).Should(BeFalse())
		})
	})

	Context("some instances have same InstanceID are existed", func() {
		It("return false and error", func() {
			ctx := context.Background()
			testInstances := []nifcloud.Instance{*helper.NewTestInstance(), *helper.NewTestInstance()}
			testProviderID := "nifcloud:///east-11/i-abcd1234"
			testInstanceUniqueID := "i-abcd1234"
			testNode := &v1.Node{
				Spec: v1.NodeSpec{
					ProviderID: testProviderID,
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

			stopped, err := cloud.InstanceShutdown(ctx, testNode)
			Expect(err).Should(HaveOccurred())
			Expect(stopped).Should(BeFalse())
		})
	})
})

var _ = Describe("InstanceMetadata", func() {
	var ctrl *gomock.Controller
	var region string = "east1"

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("single instance is existed", func() {
		It("return the metadata", func() {
			ctx := context.Background()
			testInstances := []nifcloud.Instance{*helper.NewTestInstance()}
			testProviderID := "nifcloud:///east-11/i-abcd1234"
			testInstanceUniqueID := "i-abcd1234"
			testNode := &v1.Node{
				Spec: v1.NodeSpec{
					ProviderID: testProviderID,
				},
			}

			expectedMetadata := &cloudprovider.InstanceMetadata{
				ProviderID:   testProviderID,
				InstanceType: testInstances[0].InstanceType,
				NodeAddresses: []v1.NodeAddress{
					{
						Type:    v1.NodeExternalIP,
						Address: testInstances[0].PublicIPAddress,
					},
					{
						Type:    v1.NodeInternalIP,
						Address: testInstances[0].PrivateIPAddress,
					},
				},
				Zone:   testInstances[0].Zone,
				Region: region,
			}

			c := nifcloud.NewMockCloudAPIClient(ctrl)
			c.EXPECT().
				DescribeInstancesByInstanceUniqueID(gomock.Any(), []string{testInstanceUniqueID}).
				Return(testInstances, nil).
				Times(1)

			cloud := &nifcloud.Cloud{}
			cloud.SetClient(c)
			cloud.SetRegion(region)

			metadata, err := cloud.InstanceMetadata(ctx, testNode)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(metadata).Should(Equal(expectedMetadata))
		})
	})

	Context("the instance is not existed", func() {
		It("return error", func() {
			ctx := context.Background()
			testInstances := []nifcloud.Instance{}
			testProviderID := "nifcloud:///east-11/i-abcd1234"
			testInstanceUniqueID := "i-abcd1234"
			testNode := &v1.Node{
				Spec: v1.NodeSpec{
					ProviderID: testProviderID,
				},
			}

			notFoundErr := cloudprovider.InstanceNotFound
			c := nifcloud.NewMockCloudAPIClient(ctrl)
			c.EXPECT().
				DescribeInstancesByInstanceUniqueID(gomock.Any(), []string{testInstanceUniqueID}).
				Return(testInstances, notFoundErr).
				Times(1)

			cloud := &nifcloud.Cloud{}
			cloud.SetClient(c)
			cloud.SetRegion(region)

			metadata, err := cloud.InstanceMetadata(ctx, testNode)
			Expect(err).Should(HaveOccurred())
			Expect(metadata).Should(BeNil())
		})
	})

	Context("some instances have same InstanceID are existed", func() {
		It("return error", func() {
			ctx := context.Background()
			testInstances := []nifcloud.Instance{*helper.NewTestInstance(), *helper.NewTestInstance()}
			testProviderID := "nifcloud:///east-11/i-abcd1234"
			testInstanceUniqueID := "i-abcd1234"
			testNode := &v1.Node{
				Spec: v1.NodeSpec{
					ProviderID: testProviderID,
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

			metadata, err := cloud.InstanceMetadata(ctx, testNode)
			Expect(err).Should(HaveOccurred())
			Expect(metadata).Should(BeNil())
		})
	})
})

var _ = Describe("getInstance", func() {
	var ctrl *gomock.Controller
	var region string = "east1"

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("single instance is existed", func() {
		Context("given node has a provider id", func() {
			It("return true", func() {
				ctx := context.Background()
				testInstances := []nifcloud.Instance{*helper.NewTestInstance()}
				testProviderID := "nifcloud:///east-11/i-abcd1234"
				testInstanceUniqueID := "i-abcd1234"
				testNode := &v1.Node{
					Spec: v1.NodeSpec{
						ProviderID: testProviderID,
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

				gotInstance, err := nifcloud.ExportGetInstance(cloud, ctx, testNode)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(gotInstance).Should(Equal(&testInstances[0]))
			})
		})
	})

	Context("the instance is not existed", func() {
		It("return false and error", func() {
			ctx := context.Background()
			testInstances := []nifcloud.Instance{}
			testProviderID := "nifcloud:///east-11/i-abcd1234"
			testInstanceUniqueID := "i-abcd1234"
			testNode := &v1.Node{
				Spec: v1.NodeSpec{
					ProviderID: testProviderID,
				},
			}

			notFoundErr := cloudprovider.InstanceNotFound
			c := nifcloud.NewMockCloudAPIClient(ctrl)
			c.EXPECT().
				DescribeInstancesByInstanceUniqueID(gomock.Any(), []string{testInstanceUniqueID}).
				Return(testInstances, notFoundErr).
				Times(1)

			cloud := &nifcloud.Cloud{}
			cloud.SetClient(c)
			cloud.SetRegion(region)

			gotInstance, err := nifcloud.ExportGetInstance(cloud, ctx, testNode)
			Expect(err).Should(HaveOccurred())
			Expect(gotInstance).Should(BeNil())
		})
	})

	Context("some instances have same InstanceID are existed", func() {
		It("return false and error", func() {
			ctx := context.Background()
			testInstances := []nifcloud.Instance{*helper.NewTestInstance(), *helper.NewTestInstance()}
			testProviderID := "nifcloud:///east-11/i-abcd1234"
			testInstanceUniqueID := "i-abcd1234"
			testNode := &v1.Node{
				Spec: v1.NodeSpec{
					ProviderID: testProviderID,
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

			gotInstance, err := nifcloud.ExportGetInstance(cloud, ctx, testNode)
			Expect(err).Should(HaveOccurred())
			Expect(gotInstance).Should(BeNil())
		})
	})
})

var _ = Describe("getNodeAddress", func() {
	Context("given an instance has only public network", func() {
		It("return the NodeAddress", func() {
			testInstance := *helper.NewTestInstance()
			testInstance.PrivateIPAddress = ""
			expectedNodeAddress := []v1.NodeAddress{
				{
					Type:    v1.NodeExternalIP,
					Address: testInstance.PublicIPAddress,
				},
			}

			nodeAddress := nifcloud.ExportGetNodeAddress(testInstance)
			Expect(nodeAddress).Should(Equal(expectedNodeAddress))
		})
	})

	Context("given an instance has only private network", func() {
		It("return the NodeAddress", func() {
			testInstance := *helper.NewTestInstance()
			testInstance.PublicIPAddress = ""
			expectedNodeAddress := []v1.NodeAddress{
				{
					Type:    v1.NodeInternalIP,
					Address: testInstance.PrivateIPAddress,
				},
			}

			nodeAddress := nifcloud.ExportGetNodeAddress(testInstance)
			Expect(nodeAddress).Should(Equal(expectedNodeAddress))
		})
	})

	Context("given an instance has both of public and private network", func() {
		It("return the NodeAddress", func() {
			testInstance := *helper.NewTestInstance()
			expectedNodeAddress := []v1.NodeAddress{
				{
					Type:    v1.NodeExternalIP,
					Address: testInstance.PublicIPAddress,
				},
				{
					Type:    v1.NodeInternalIP,
					Address: testInstance.PrivateIPAddress,
				},
			}

			nodeAddress := nifcloud.ExportGetNodeAddress(testInstance)
			Expect(nodeAddress).Should(Equal(expectedNodeAddress))
		})
	})
})
