package metrics

import (
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	ValkeyCache = "valkey"
)

func cacheDriver() string {
	if os.Getenv("VALKEY_SERVICE") != "" {
		return ValkeyCache
	}

	return "none"
}

var (
	CacheMissesTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name:        "cache_misses_total",
		Namespace:   NostradamusNamespace,
		ConstLabels: prometheus.Labels{"cache": cacheDriver()},
		Help:        "The total number of cache misses since the application started.",
	})

	CacheHitsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name:        "cache_hits_total",
		Namespace:   NostradamusNamespace,
		ConstLabels: prometheus.Labels{"cache": cacheDriver()},
		Help:        "The total number of cache hits since the application started.",
	})

	CacheReadLatencySeconds = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:        "cache_read_latency_seconds",
		Namespace:   NostradamusNamespace,
		ConstLabels: prometheus.Labels{"cache": cacheDriver()},
		Buckets:     prometheus.DefBuckets,
		Help:        "The latency of cache read operations in seconds.",
	})

	CacheWriteLatencySeconds = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:        "cache_write_latency_seconds",
		Namespace:   NostradamusNamespace,
		ConstLabels: prometheus.Labels{"cache": cacheDriver()},
		Buckets:     prometheus.DefBuckets,
		Help:        "The latency of cache write operations in seconds.",
	})
)
