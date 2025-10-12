package db

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/gocql/gocql"
	"github.com/google/uuid"
	"github.com/ntentasd/nostradamus-api/pkg/types"
)

type SensorAlreadyExistsError struct {
	SensorName string
}

func (e *SensorAlreadyExistsError) Error() string {
	return fmt.Sprintf("sensor '%s' already exists", e.SensorName)
}

func (db *DB) GetLast5Values(
	sensor string,
	date string,
) ([]types.Entry, error) {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Millisecond*500,
	)
	defer cancel()

	id, err := gocql.ParseUUID(sensor)
	if err != nil {
		return nil, err
	}

	query := db.Data.Query(`
SELECT timestamp, value
FROM temperatures
WHERE sensor_id=?
AND bucket_date=?
ORDER BY timestamp DESC LIMIT 5
`, id, date).WithContext(ctx)

	var results []types.Entry
	iter := query.Iter()

	var ts time.Time
	var val float64

	for iter.Scan(&ts, &val) {
		val, _ = strconv.ParseFloat(fmt.Sprintf("%.4f", val), 64)
		results = append(results, types.Entry{
			Timestamp: ts,
			Value:     val,
		})
	}

	if err := iter.Close(); err != nil {
		return nil, err
	}

	return results, nil
}

func (db *DB) GetSensorsByFieldID(fieldID uuid.UUID) ([]types.Sensor, *types.Field, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	query := db.Meta.Query(`
SELECT sensor_id, sensor_name, sensor_type, field_name
FROM sensors_by_field
WHERE field_id = ?
`, gocql.UUID(fieldID)).WithContext(ctx)

	var results []types.Sensor
	var field *types.Field
	iter := query.Iter()

	var (
		sensorID   uuid.UUID
		sensorName string
		sensorType string
		fieldName  string
	)

	for iter.Scan(&sensorID, &sensorName, &sensorType, &fieldName) {
		var sType types.SensorType
		sensorTypeInt, convErr := strconv.Atoi(sensorType)
		if convErr == nil {
			sType = types.SensorType(sensorTypeInt)
		} else {
			var err error
			sType, err = types.ToSensorType(sensorType)
			if err != nil {
				log.Printf("[WARN] skipping sensor %s: invalid type '%s': %v\n", sensorName, sensorType, err)
				continue
			}
		}

		results = append(results, types.Sensor{
			SensorID:   sensorID,
			SensorName: sensorName,
			SensorType: sType,
		})

		if field == nil {
			field = &types.Field{
				FieldID:   fieldID,
				FieldName: fieldName,
			}
		}
	}

	if err := iter.Close(); err != nil {
		return nil, &types.Field{}, err
	}

	if results == nil {
		results = []types.Sensor{}
	}
	if field == nil {
		field = &types.Field{
			FieldID:   fieldID,
			FieldName: "",
		}
	}

	return results, field, nil
}

func (db *DB) RegisterSensor(fieldID uuid.UUID, sensorName string, sensorType int) (*types.Sensor, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	checkQuery := db.Meta.Query(`
SELECT sensor_id FROM sensors_by_field
WHERE field_id = ?
`, gocql.UUID(fieldID)).WithContext(ctx)

	iter := checkQuery.Iter()
	var sensorID gocql.UUID
	for iter.Scan(&sensorID) {
		var existingName string
		subQuery := db.Meta.Query(`
SELECT sensor_name FROM sensors_by_field
WHERE field_id = ? AND sensor_id = ?
`, gocql.UUID(fieldID), sensorID).WithContext(ctx)
		if err := subQuery.Scan(&existingName); err == nil && existingName == sensorName {
			return nil, &SensorAlreadyExistsError{SensorName: sensorName}
		}
	}
	if err := iter.Close(); err != nil {
		return nil, err
	}

	newID := uuid.New()
	query := db.Meta.Query(`
INSERT INTO sensors_by_field (field_id, sensor_id, sensor_name, sensor_type)
VALUES (?, ?, ?, ?)
`, gocql.UUID(fieldID), gocql.UUID(newID), sensorName, fmt.Sprintf("%d", sensorType)).WithContext(ctx)
	if err := query.Exec(); err != nil {
		return nil, err
	}

	if err := db.Meta.Query(`
INSERT INTO sensors (sensor_id, sensor_name, sensor_type)
VALUES (?, ?, ?)
`, gocql.UUID(newID), sensorName, fmt.Sprintf("%d", sensorType)).WithContext(ctx).Exec(); err != nil {
		return nil, err
	}

	return &types.Sensor{
		SensorID:   newID,
		SensorName: sensorName,
		SensorType: types.SensorType(sensorType),
	}, nil
}

func (db *DB) StoreSensorCredentials(fieldID gocql.UUID, sensorID gocql.UUID, mqttUser, mqttPass string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	query := db.Meta.Query(`
UPDATE sensors_by_field
SET mqtt_username = ?, mqtt_password = ?
WHERE field_id = ? AND sensor_id = ?
`, mqttUser, mqttPass, fieldID, sensorID).WithContext(ctx)

	return query.Exec()
}

var ErrSensorNotFound = errors.New("sensor not found")

type SensorCredentials struct {
	MqttUser string `json:"username"`
	MqttPass string `json:"password"`
}

func (db *DB) GetSensorCredentials(sensorID uuid.UUID) (*SensorCredentials, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	query := db.Meta.Query(`
SELECT mqtt_username, mqtt_password
FROM sensors_by_field
WHERE sensor_id = ?
ALLOW FILTERING
`, gocql.UUID(sensorID)).WithContext(ctx)

	var username, password string
	if err := query.Scan(&username, &password); err != nil {
		if err == gocql.ErrNotFound {
			return nil, ErrSensorNotFound
		}
		return nil, err
	}

	return &SensorCredentials{
		MqttUser: username,
		MqttPass: password,
	}, nil
}
