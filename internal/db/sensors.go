// Package db
package db

import (
	"context"
	"strconv"
	"time"

	"github.com/gocql/gocql"
	"github.com/ntentasd/nostradamus-api/pkg/types"
	"gopkg.in/inf.v0"
)

type DB struct {
	sess *gocql.Session
}

func New(sess *gocql.Session) *DB {
	return &DB{sess: sess}
}

func (db *DB) Close() {
	db.sess.Close()
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
	query := db.sess.Query(`
SELECT timestamp, value
FROM temperatures
WHERE sensor_id=?
AND bucket_date=?
ORDER BY timestamp DESC LIMIT 5
`, sensor, date).WithContext(ctx)

	var results []types.Entry
	iter := query.Iter()

	var ts time.Time
	var dec *inf.Dec

	for iter.Scan(&ts, &dec) {
		val, _ := strconv.ParseFloat(dec.String(), 64)
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
