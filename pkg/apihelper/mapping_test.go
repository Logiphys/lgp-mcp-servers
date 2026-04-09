package apihelper

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestMappingCache_GetAndCache(t *testing.T) {
	var calls atomic.Int32
	mc := NewMappingCache[int, string](30 * time.Minute)
	fetch := func(id int) (string, error) {
		calls.Add(1)
		return "Company A", nil
	}
	v1, err := mc.Get(context.Background(), 1, fetch)
	if err != nil || v1 != "Company A" {
		t.Fatalf("Get = (%q, %v)", v1, err)
	}
	v2, _ := mc.Get(context.Background(), 1, fetch)
	if v2 != "Company A" {
		t.Error("cached value should be returned")
	}
	if calls.Load() != 1 {
		t.Errorf("fetch called %d times, want 1", calls.Load())
	}
}

func TestMappingCache_TTLExpiry(t *testing.T) {
	var calls atomic.Int32
	mc := NewMappingCache[int, string](50 * time.Millisecond)
	fetch := func(id int) (string, error) { calls.Add(1); return "value", nil }
	_, _ = mc.Get(context.Background(), 1, fetch)
	time.Sleep(60 * time.Millisecond)
	_, _ = mc.Get(context.Background(), 1, fetch)
	if calls.Load() != 2 {
		t.Errorf("calls = %d, want 2", calls.Load())
	}
}

func TestMappingCache_Warm(t *testing.T) {
	mc := NewMappingCache[int, string](30 * time.Minute)
	err := mc.Warm(context.Background(), func() (map[int]string, error) {
		return map[int]string{1: "A", 2: "B", 3: "C"}, nil
	})
	if err != nil {
		t.Fatalf("Warm failed: %v", err)
	}
	var fetchCalled bool
	v, _ := mc.Get(context.Background(), 2, func(int) (string, error) { fetchCalled = true; return "", nil })
	if fetchCalled {
		t.Error("fetch should not be called for warmed key")
	}
	if v != "B" {
		t.Errorf("value = %q, want B", v)
	}
}

func TestMappingCache_Clear(t *testing.T) {
	mc := NewMappingCache[int, string](30 * time.Minute)
	_, _ = mc.Get(context.Background(), 1, func(int) (string, error) { return "v", nil })
	mc.Clear()
	size, _ := mc.Stats()
	if size != 0 {
		t.Errorf("size after Clear = %d", size)
	}
}

func TestMappingCache_Stats(t *testing.T) {
	mc := NewMappingCache[int, string](30 * time.Minute)
	fetch := func(int) (string, error) { return "v", nil }
	mc.Get(context.Background(), 1, fetch) // miss
	mc.Get(context.Background(), 1, fetch) // hit
	mc.Get(context.Background(), 1, fetch) // hit
	size, hitRate := mc.Stats()
	if size != 1 {
		t.Errorf("size = %d", size)
	}
	if hitRate < 60 || hitRate > 70 {
		t.Errorf("hitRate = %f, want ~66.6", hitRate)
	}
}

func TestMappingCache_ConcurrentAccess(t *testing.T) {
	mc := NewMappingCache[int, string](30 * time.Minute)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			mc.Get(context.Background(), id%5, func(k int) (string, error) { return "val", nil })
		}(i)
	}
	wg.Wait()
}

func TestMappingCache_SingleFlight(t *testing.T) {
	var calls atomic.Int32
	mc := NewMappingCache[int, string](30 * time.Minute)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mc.Get(context.Background(), 1, func(int) (string, error) {
				calls.Add(1)
				time.Sleep(20 * time.Millisecond)
				return "val", nil
			})
		}()
	}
	wg.Wait()
	if calls.Load() > 2 {
		t.Errorf("fetch called %d times, want <=2 (singleflight)", calls.Load())
	}
}
