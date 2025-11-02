// Package types
package types

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Entry struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

type Field struct {
	FieldID   uuid.UUID  `json:"field_id"`
	FieldName string     `json:"field_name"`
	UserID    *uuid.UUID `json:"user_id,omitempty"`
}

type SensorType int

const (
	SensorTypeTemperature SensorType = iota
	SensorTypeHumidity
	SensorTypePHLevel
)

var ErrInvalidSensorType = fmt.Errorf("invalid sensor type")

type Sensor struct {
	SensorID   uuid.UUID  `json:"sensor_id"`
	SensorName string     `json:"sensor_name"`
	SensorType SensorType `json:"sensor_type"`
	FieldID    *uuid.UUID `json:"field_id,omitempty"`
	FieldName  string     `json:"field_name,omitempty"`
}

func ToSensorType(sensorType string) (SensorType, error) {
	switch sensorType {
	case "temperature":
		return SensorTypeTemperature, nil
	case "humidity":
		return SensorTypeHumidity, nil
	case "ph_level":
		return SensorTypePHLevel, nil
	default:
		return -1, ErrInvalidSensorType
	}
}

type StateType string

const (
	StateTypeRunning    StateType = "Running"
	StateTypeStopped    StateType = "Stopped"
	StateTypeScheduling StateType = "Scheduling"
	StateTypeFailed     StateType = "Failed"
)

type Job struct {
	ID        string    `json:"id"`
	State     StateType `json:"state"`
	StartedAt int64     `json:"started_at"`
}

type Pipeline struct {
	Name        string `json:"name"`
	Query       string `json:"query,omitempty"`
	Parallelism *int   `json:"parallelism,omitempty"`
}

type Aggregate struct {
	Avg       float64   `json:"avg"`
	Min       float64   `json:"min"`
	Max       float64   `json:"max"`
	Count     int       `json:"count"`
	Timestamp time.Time `json:"timestamp"`
}
