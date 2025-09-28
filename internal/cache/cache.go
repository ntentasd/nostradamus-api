// Package cache
package cache

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/ntentasd/nostradamus-api/pkg/types"
	"github.com/redis/go-redis/v9"
)

type DB struct {
	client *redis.ClusterClient
}

func New(addrs ...string) *DB {
	opts := &redis.ClusterOptions{Addrs: addrs}
	client := redis.NewClusterClient(opts)
	return &DB{client}
}

func (db *DB) Close() {
	db.client.Close()
}

func (db *DB) Store(prefix string, entry types.Entry) error {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Millisecond*100,
	)
	defer cancel()

	_, err := db.client.ZAdd(ctx, prefix, redis.Z{
		Score:  float64(entry.Timestamp.UnixMilli()),
		Member: entry.Value,
	}).Result()
	if err != nil {
		return err
	}

	_, err = db.client.Expire(ctx, prefix, time.Hour).Result()
	if err != nil {
		return err
	}

	return nil
}

func (db *DB) FetchLast(prefix string, last int) ([]types.Entry, error) {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Millisecond*100,
	)
	defer cancel()

	entries, err := db.client.ZRevRangeWithScores(ctx, prefix, 0, int64(last-1)).
		Result()
	if err != nil {
		return nil, err
	}

	ret := make([]types.Entry, 0, len(entries))

	for _, e := range entries {
		ts := time.Unix(0, int64(e.Score)*int64(time.Millisecond))

		s, ok := e.Member.(string)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", e.Member)
		}

		val, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse value: %w", err)
		}

		ret = append(ret, types.Entry{
			Timestamp: ts,
			Value:     val,
		})
	}

	return ret, nil
}

func (db *DB) FetchLast5(prefix string) ([]types.Entry, error) {
	entries, err := db.FetchLast(prefix, 5)
	if err != nil {
		return nil, err
	}

	return entries, nil
}
