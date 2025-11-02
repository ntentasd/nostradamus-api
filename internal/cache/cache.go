package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/ntentasd/nostradamus-api/pkg/types"
	"github.com/redis/go-redis/v9"
)

type DB struct {
	client *redis.ClusterClient
}

func New() *DB {
	addrs := resolveValkeyAddrs()
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
		time.Millisecond*200,
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

func (db *DB) FetchLast(prefix string, n int) ([]types.Entry, error) {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Millisecond*100,
	)
	defer cancel()

	entries, err := db.client.ZRevRangeWithScores(ctx, prefix, 0, int64(n-1)).
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

func (db *DB) StoreAggregate(key string, data any, ttl time.Duration) error {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Millisecond*200,
	)
	defer cancel()

	b, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal aggregate: %w", err)
	}

	if err := db.client.Set(ctx, key, b, ttl).Err(); err != nil {
		return fmt.Errorf("failed to store aggregate: %w", err)
	}

	return nil
}

func (db *DB) FetchAggregate(key string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Millisecond*200,
	)
	defer cancel()

	val, err := db.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, fmt.Errorf("cache miss")
	}
	if err != nil {
		return nil, err
	}
	return val, nil
}
