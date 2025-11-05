package db

import (
	"context"
	"fmt"
	"time"

	"github.com/gocql/gocql"
	"github.com/ntentasd/nostradamus-api/internal/metrics"
	"github.com/ntentasd/nostradamus-api/pkg/types"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// GetReadings returns all sensor readings between two timestamps, possibly spanning multiple bucket_dates.
func (db *DB) GetReadings(ctx context.Context, sensorID string, sType int, from, to time.Time) ([]float64, error) {
	ctx, span := otel.Tracer("nostradamus-db").Start(ctx, "db.GetReadings")
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	sid, err := gocql.ParseUUID(sensorID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("invalid sensor_id: %w", err)
	}

	var sensorType string
	switch sType {
	case int(types.SensorTypeTemperature):
		sensorType = "temperatures"
	case int(types.SensorTypeHumidity):
		sensorType = "humidities"
	case int(types.SensorTypePHLevel):
		sensorType = "ph_levels"
	default:
		err := fmt.Errorf("invalid sensor type")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	readings := make([]float64, 0, 256)

	start := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	end := time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, time.UTC)

	// Calculate all days (buckets) between from and to
	for date := start; !date.After(end); date = date.Add(24 * time.Hour) {
		bucket := date

		query := fmt.Sprintf(`
SELECT value
FROM sensors_data.%s
WHERE sensor_id = ? AND bucket_date = ? AND timestamp >= ? AND timestamp <= ?
ORDER BY timestamp DESC
`, sensorType)

		qctx, qspan := otel.Tracer("nostradamus-db").Start(ctx, "db.query")
		qspan.SetAttributes(
			attribute.String("table", sensorType),
			attribute.String("bucket.date", bucket.String()),
			attribute.String("query", query),
		)

		start := time.Now()
		iter := db.Data.Query(query, sid, bucket, from, to).WithContext(qctx).Iter()

		var v float64
		for iter.Scan(&v) {
			readings = append(readings, v)
		}

		if err := iter.Close(); err != nil {
			qspan.RecordError(err)
			qspan.SetStatus(codes.Error, err.Error())
			qspan.End()
			return nil, fmt.Errorf("failed to query bucket %s: %w", bucket, err)
		}

		metrics.DbReadLatencySeconds.WithLabelValues("readings").Observe(time.Since(start).Seconds())
		qspan.End()
	}

	return readings, nil
}
