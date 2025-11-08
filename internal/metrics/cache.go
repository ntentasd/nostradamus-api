package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	CacheMissesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name:      "cache_misses_total",
			Namespace: NostradamusNamespace,
			Help:      "The total number of cache misses since the application started.",
		},
		[]string{"driver"},
	)

	CacheHitsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name:      "cache_hits_total",
			Namespace: NostradamusNamespace,
			Help:      "The total number of cache hits since the application started.",
		},
		[]string{"driver"},
	)

	cacheBuckets = []float64{0.0001, 0.00025, 0.0005, 0.001, 0.0025, 0.005, 0.01, 0.025, 0.05}

	CacheReadLatencySeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:      "cache_read_latency_seconds",
			Namespace: NostradamusNamespace,
			Buckets:   cacheBuckets,
			Help:      "The latency of cache read operations in seconds.",
		},
		[]string{"driver"},
	)

	CacheWriteLatencySeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:      "cache_write_latency_seconds",
			Namespace: NostradamusNamespace,
			Buckets:   cacheBuckets,
			Help:      "The latency of cache write operations in seconds.",
		},
		[]string{"driver"},
	)
)
