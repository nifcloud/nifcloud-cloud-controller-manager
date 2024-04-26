package nifcloud

import (
	"context"
	"strings"

	"github.com/nifcloud/nifcloud-cloud-controller-manager/test/helper"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("getElasticLoadBalancer", func ()  {
	var ctrl *gomock.Controller
	var region string = "east1"
	var loadBalancerUID types.UID
	var loadBalancerName string

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		loadBalancerUID = types.UID(uuid.NewString())
		loadBalancerName = strings.Replace(string(loadBalancerUID), "-", "", -1)[:maxLoadBalancerNameLength]
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("the specified elastic load balancer is existed", func ()  {
		It("return the status", func ()  {
			ctx := context.Background()
			clusterName := "testCluster"
			service := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name: clusterName,
					UID: loadBalancerUID,
				},
			}
			testIPAddress := "192.168.0.1"

			expectedStatus := &corev1.LoadBalancerStatus{
				Ingress: []corev1.LoadBalancerIngress{
					{
						IP: testIPAddress,
					},
				},
			}

			c := NewMockCloudAPIClient(ctrl)
			c.EXPECT().
				DescribeElasticLoadBalancers(gomock.Any(), gomock.Eq(loadBalancerName)).
				Return([]ElasticLoadBalancer{
					{
						Name:             clusterName,
						VIP:              testIPAddress,
					},
				}, nil).
				Times(1)

			cloud := &Cloud{
				client: c,
				region: region,
			}

			status, exists, err := cloud.getElasticLoadBalancer(ctx, clusterName, service)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(exists).To(BeTrue())
			Expect(*status).To(Equal(*expectedStatus))
		})
	})

	Context("the specified elastic load balancer is not existed", func ()  {
		It("return that exists is false", func () {
			ctx := context.Background()
			clusterName := "testCluster"
			service := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name: clusterName,
					UID: loadBalancerUID,
				},
			}

			apiErr := helper.NewMockAPIError(errorCodeElasticLoadBalancerNotFound)

			c := NewMockCloudAPIClient(ctrl)
			c.EXPECT().
				DescribeElasticLoadBalancers(gomock.Any(), gomock.Eq(loadBalancerName)).
				Return([]ElasticLoadBalancer{}, apiErr).
				Times(1)

			cloud := &Cloud{
				client: c,
				region: region,
			}

			status, exists, err := cloud.getElasticLoadBalancer(ctx, clusterName, service)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(exists).To(BeFalse())
			Expect(status).To(BeNil())
		})
	})
})
