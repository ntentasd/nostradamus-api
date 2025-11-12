package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/ntentasd/nostradamus-api/pkg/types"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

var _ Cache = (*Memcached)(nil)

type Memcached struct {
	client  *memcache.Client
	metrics *CacheMetrics
}

func NewMemcached(addr string) *Memcached {
	client := memcache.New(addr)
	cm := NewCacheMetrics("memcached")
	return &Memcached{client, cm}
}

func (m *Memcached) store(key string, val []byte, ttl time.Duration) error {
	done := make(chan error, 1)
	go func() {
		done <- m.client.Set(&memcache.Item{Key: key, Value: val, Expiration: int32(ttl.Seconds())})
	}()
	select {
	case err := <-done:
		return err
	case <-time.After(100 * time.Millisecond):
		return context.DeadlineExceeded
	}
}

func (m *Memcached) Store(prefix string, entry types.Entry) error {
	// unimplemented
	return nil
}

func (m *Memcached) FetchLast(prefix string, n int) ([]types.Entry, error) {
	// unimplemented
	return nil, nil
}

func (m *Memcached) StoreAggregate(ctx context.Context, key string, data any, ttl time.Duration) error {
	ctx, span := otel.Tracer("nostradamus-cache").Start(ctx, "cache.StoreAggregate")
	defer span.End()

	span.SetAttributes(
		attribute.String("cache.driver", "memcached"),
		attribute.String("cache.key", key),
		attribute.Int64("cache.ttl", int64(ttl.Seconds())),
	)

	b, err := json.Marshal(data)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("failed to marshal aggregate: %w", err)
	}

	start := time.Now()
	if err := m.store(key, b, ttl); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("failed to store aggregate: %w", err)
	}
	m.metrics.RecordWrite(start)
	span.SetStatus(codes.Ok, "")

	return nil
}

func (m *Memcached) FetchAggregate(ctx context.Context, key string) ([]byte, error) {
	ctx, span := otel.Tracer("nostradamus-cache").Start(ctx, "cache.FetchAggregate")
	defer span.End()

	span.SetAttributes(
		attribute.String("cache.driver", "memcached"),
		attribute.String("cache.key", key),
	)

	start := time.Now()
	val, err := m.client.Get(key)
	switch {
	case err == memcache.ErrCacheMiss:
		m.metrics.RecordMiss()
		span.SetAttributes(attribute.String("cache.result", "miss"))
		span.SetStatus(codes.Ok, "")
		return nil, fmt.Errorf("cache miss")
	case err != nil:
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("cache fetch: %w", err)
	default:
		m.metrics.RecordHit(start)
		span.SetAttributes(attribute.String("cache.result", "hit"))
		span.SetStatus(codes.Ok, "")
		return val.Value, nil
	}
}

func (m *Memcached) Ping(ctx context.Context) error {
	return m.client.Ping()
}

func (m *Memcached) Close() {
	m.client.Close()
}
