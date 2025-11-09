package db

import (
	"github.com/gocql/gocql"
	"github.com/rs/zerolog"
)

type DB struct {
	Meta   *gocql.Session // sensors_data
	Data   *gocql.Session // sensors_data
	logger zerolog.Logger
}

func New(metaSess, dataSess *gocql.Session, logger zerolog.Logger) *DB {
	return &DB{
		Meta:   metaSess,
		Data:   dataSess,
		logger: logger,
	}
}

func (db *DB) Close() {
	if db.Meta != nil {
		db.Meta.Close()
	}
	if db.Data != nil {
		db.Data.Close()
	}
}
