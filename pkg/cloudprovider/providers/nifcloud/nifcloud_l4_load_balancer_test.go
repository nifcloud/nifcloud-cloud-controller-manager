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
