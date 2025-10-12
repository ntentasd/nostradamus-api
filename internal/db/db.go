package db

import "github.com/gocql/gocql"

type DB struct {
	Meta *gocql.Session // sensors_data
	Data *gocql.Session // sensors_data
}

func New(metaSess, dataSess *gocql.Session) *DB {
	return &DB{
		Meta: metaSess,
		Data: dataSess,
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
