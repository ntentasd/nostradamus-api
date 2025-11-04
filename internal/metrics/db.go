package metrics

import (
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	ScyllaDb = "scylladb"
)

func dbDriver() string {
	if os.Getenv("SCYLLA_NODES") != "" {
		return ScyllaDb
	}

	return "none"
}

var (
	DbReadLatencySeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:      "db_read_latency_seconds",
			Namespace: NostradamusNamespace,
			ConstLabels: prometheus.Labels{
				"db": dbDriver(),
			},
			Buckets: prometheus.DefBuckets,
			Help:    "The latency of db read operations in seconds.",
		},
		[]string{"query"},
	)
)
