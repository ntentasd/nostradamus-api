package db

import (
	"context"
	"time"

	"github.com/gocql/gocql"
	"github.com/google/uuid"
	"github.com/ntentasd/nostradamus-api/pkg/types"
)

func (db *DB) GetFieldsByUserID(userID uuid.UUID) ([]types.Field, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*500)
	defer cancel()

	query := db.sess.Query(`
SELECT field_id, field_name
FROM sensors_meta.fields
WHERE user_id = ?
`, gocql.UUID(userID)).WithContext(ctx)

	var results []types.Field
	iter := query.Iter()

	var fieldID uuid.UUID
	var fieldName string

	for iter.Scan(&fieldID, &fieldName) {
		results = append(results, types.Field{
			UserID:    &userID,
			FieldID:   fieldID,
			FieldName: fieldName,
		})
	}

	if err := iter.Close(); err != nil {
		return nil, err
	}

	return results, nil
}

func (db *DB) GetFieldByID(fieldID uuid.UUID) ([]types.Field, error) {
	return nil, nil
}

func (db *DB) GetSensorsByFieldID(fieldID uuid.UUID) ([]types.Sensor, *types.Field, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*500)
	defer cancel()

	query := db.sess.Query(`
SELECT sensor_id, sensor_name, sensor_type, field_name
FROM sensors_meta.sensors_by_field
WHERE field_id = ?
`, gocql.UUID(fieldID)).WithContext(ctx)

	var results []types.Sensor
	iter := query.Iter()

	var (
		sensorID   uuid.UUID
		sensorName string
		sensorType string
		fieldName  string
	)

	for iter.Scan(&sensorID, &sensorName, &sensorType, &fieldName) {
		sType, err := types.ToSensorType(sensorType)
		if err != nil {
			return nil, &types.Field{}, err
		}

		results = append(results, types.Sensor{
			SensorID:   sensorID,
			SensorName: sensorName,
			SensorType: sType,
		})
	}

	field := types.Field{
		FieldID:   fieldID,
		FieldName: fieldName,
	}

	if err := iter.Close(); err != nil {
		return nil, &types.Field{}, err
	}

	return results, &field, nil
}
