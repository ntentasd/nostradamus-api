package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	dbBuckets = []float64{0.0005, 0.001, 0.0025, 0.005, 0.01, 0.025}

	DbReadLatencySeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:      "db_read_latency_seconds",
			Namespace: NostradamusNamespace,
			Buckets:   dbBuckets,
			Help:      "The latency of db read operations in seconds.",
		},
		[]string{"query"},
	)
)
