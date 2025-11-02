package db

import (
	"context"
	"fmt"
	"time"

	"github.com/gocql/gocql"
	"github.com/ntentasd/nostradamus-api/pkg/types"
)

// GetReadings returns all sensor readings between two timestamps, possibly spanning multiple bucket_dates.
func (db *DB) GetReadings(sensorID string, sType int, from, to time.Time) ([]float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	sid, err := gocql.ParseUUID(sensorID)
	if err != nil {
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
		return nil, fmt.Errorf("invalid sensor type")
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

		iter := db.Data.Query(query, sid, bucket, from, to).WithContext(ctx).Iter()

		var v float64
		for iter.Scan(&v) {
			readings = append(readings, v)
		}

		if err := iter.Close(); err != nil {
			return nil, fmt.Errorf("failed to query bucket %s: %w", bucket, err)
		}
	}

	return readings, nil
}
