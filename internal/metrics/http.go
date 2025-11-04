package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	HttpRequestLatencySeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:      "http_request_latency_seconds",
			Namespace: NostradamusNamespace,
			Buckets:   prometheus.DefBuckets,
			Help:      "The latency of http operations in seconds.",
		},
		[]string{"verb"},
	)
)
