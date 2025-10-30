// Package cache
package cache

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
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

func resolveValkeyAddrs() []string {
	if nodes := os.Getenv("VALKEY_NODES"); nodes != "" {
		return strings.Split(nodes, ",")
	}

	if svc := os.Getenv("VALKEY_SERVICE"); svc != "" {
		addrs, err := net.LookupHost(svc)
		if err != nil {
			log.Fatalf("failed to resolve %s: %v", svc, err)
		}
		var out []string
		for _, ip := range addrs {
			out = append(out, fmt.Sprintf("%s:6379", ip))
		}
		return out
	}

	log.Fatal(
		"no Valkey discovery env provided (VALKEY_NODES or VALKEY_SERVICE)",
	)
	return nil
}
