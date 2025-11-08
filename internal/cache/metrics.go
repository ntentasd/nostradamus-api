package cache

import (
	"time"

	"github.com/ntentasd/nostradamus-api/internal/metrics"
)

type CacheMetrics struct {
	driver string
}

func NewCacheMetrics(driver string) *CacheMetrics {
	return &CacheMetrics{
		driver,
	}
}

// RecordHit marks a cache hit and logs latency since start
func (cm *CacheMetrics) RecordHit(start time.Time) {
	metrics.CacheHitsTotal.WithLabelValues(cm.driver).Inc()
	metrics.CacheReadLatencySeconds.WithLabelValues(cm.driver).Observe(time.Since(start).Seconds())
}

// RecordMiss marks a cache miss
func (cm *CacheMetrics) RecordMiss() {
	metrics.CacheMissesTotal.WithLabelValues(cm.driver).Inc()
}

// RecordWrite logs cache write latency since start
func (cm *CacheMetrics) RecordWrite(start time.Time) {
	metrics.CacheWriteLatencySeconds.WithLabelValues(cm.driver).Observe(float64(time.Since(start).Seconds()))
}
