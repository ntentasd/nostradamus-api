// Package types
package types

import (
	"time"
)

type Entry struct {
	Timestamp time.Time
	Value     float64
}
