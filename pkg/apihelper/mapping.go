package apihelper

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/singleflight"
)

type cacheEntry[V any] struct {
	value   V
	expires time.Time
}

type MappingCache[K comparable, V any] struct {
	mu      sync.RWMutex
	entries map[K]cacheEntry[V]
	ttl     time.Duration
	group   singleflight.Group
	hits    atomic.Int64
	total   atomic.Int64
}

func NewMappingCache[K comparable, V any](ttl time.Duration) *MappingCache[K, V] {
	return &MappingCache[K, V]{
		entries: make(map[K]cacheEntry[V]),
		ttl:     ttl,
	}
}

func (m *MappingCache[K, V]) Get(ctx context.Context, key K, fetch func(K) (V, error)) (V, error) {
	m.total.Add(1)
	m.mu.RLock()
	if e, ok := m.entries[key]; ok && time.Now().Before(e.expires) {
		m.mu.RUnlock()
		m.hits.Add(1)
		return e.value, nil
	}
	m.mu.RUnlock()

	cacheKey := fmt.Sprintf("%v", key)
	result, err, _ := m.group.Do(cacheKey, func() (any, error) {
		v, err := fetch(key)
		if err != nil {
			return nil, err
		}
		m.mu.Lock()
		m.entries[key] = cacheEntry[V]{value: v, expires: time.Now().Add(m.ttl)}
		m.mu.Unlock()
		return v, nil
	})
	if err != nil {
		var zero V
		return zero, err
	}
	return result.(V), nil
}

func (m *MappingCache[K, V]) Warm(ctx context.Context, fetchAll func() (map[K]V, error)) error {
	all, err := fetchAll()
	if err != nil {
		return fmt.Errorf("warming cache: %w", err)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	expires := time.Now().Add(m.ttl)
	for k, v := range all {
		m.entries[k] = cacheEntry[V]{value: v, expires: expires}
	}
	return nil
}

func (m *MappingCache[K, V]) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries = make(map[K]cacheEntry[V])
	m.hits.Store(0)
	m.total.Store(0)
}

func (m *MappingCache[K, V]) Stats() (size int, hitRate float64) {
	m.mu.RLock()
	size = len(m.entries)
	m.mu.RUnlock()
	total := m.total.Load()
	if total == 0 {
		return size, 0
	}
	return size, float64(m.hits.Load()) / float64(total) * 100
}
