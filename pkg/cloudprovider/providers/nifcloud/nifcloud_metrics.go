package nifcloud

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/component-base/metrics"
	"k8s.io/component-base/metrics/legacyregistry"
)

var (
	nifcloudAPIMetric = metrics.NewHistogramVec(
		&metrics.HistogramOpts{
			Name:           "cloudprovider_nifcloud_api_request_duration_seconds",
			Help:           "Latency of NIFCLOUD API calls",
			StabilityLevel: metrics.ALPHA,
		},
		[]string{"request"})

	nifcloudAPIErrorMetric = metrics.NewCounterVec(
		&metrics.CounterOpts{
			Name:           "cloudprovider_nifcloud_api_request_errors",
			Help:           "NIFCLOUD API errors",
			StabilityLevel: metrics.ALPHA,
		},
		[]string{"request"})
)

func recordNIFCLOUDMetric(actionName string, timeTaken float64, err error) {
	if err != nil {
		nifcloudAPIErrorMetric.With(prometheus.Labels{"request": actionName}).Inc()
	} else {
		nifcloudAPIMetric.With(prometheus.Labels{"request": actionName}).Observe(timeTaken)
	}
}

var registerOnce sync.Once

func registerMetrics() {
	registerOnce.Do(func() {
		legacyregistry.MustRegister(nifcloudAPIMetric)
		legacyregistry.MustRegister(nifcloudAPIErrorMetric)
	})
}
