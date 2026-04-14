package cache

import (
	"testing"
	"time"
)

func TestCacheGetSet(t *testing.T) {
	cache := New[string](100 * time.Millisecond)

	// Test Get on empty cache
	val, found := cache.Get("key1")
	if found {
		t.Error("expected key to not be found in empty cache")
	}
	if val != "" {
		t.Errorf("expected empty string, got %q", val)
	}

	// Test Set and Get
	cache.Set("key1", "value1")
	val, found = cache.Get("key1")
	if !found {
		t.Error("expected key to be found after Set")
	}
	if val != "value1" {
		t.Errorf("expected 'value1', got %q", val)
	}

	// Test Get on different key
	_, found = cache.Get("key2")
	if found {
		t.Error("expected key2 to not be found")
	}
}

func TestCacheExpiration(t *testing.T) {
	cache := New[string](50 * time.Millisecond)

	// Set a value
	cache.Set("key1", "value1")

	// Value should be available immediately
	val, found := cache.Get("key1")
	if !found {
		t.Error("expected key to be found immediately after Set")
	}
	if val != "value1" {
		t.Errorf("expected 'value1', got %q", val)
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Value should be expired now
	val, found = cache.Get("key1")
	if found {
		t.Error("expected key to be expired after TTL")
	}
	if val != "" {
		t.Errorf("expected empty string for expired key, got %q", val)
	}
}

func TestCacheDelete(t *testing.T) {
	cache := New[string](time.Minute)

	// Set and then delete
	cache.Set("key1", "value1")
	cache.Delete("key1")

	val, found := cache.Get("key1")
	if found {
		t.Error("expected key to not be found after Delete")
	}
	if val != "" {
		t.Errorf("expected empty string after Delete, got %q", val)
	}

	// Delete non-existent key should not panic
	cache.Delete("nonexistent")
}

func TestCacheClear(t *testing.T) {
	cache := New[string](time.Minute)

	// Set multiple values
	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Set("key3", "value3")

	// Clear all
	cache.Clear()

	// All should be gone
	for _, key := range []string{"key1", "key2", "key3"} {
		val, found := cache.Get(key)
		if found {
			t.Errorf("expected %s to not be found after Clear", key)
		}
		if val != "" {
			t.Errorf("expected empty string for %s after Clear, got %q", key, val)
		}
	}
}

func TestCacheSize(t *testing.T) {
	cache := New[string](time.Minute)

	// Empty cache should have size 0
	if cache.Size() != 0 {
		t.Errorf("expected size 0 for empty cache, got %d", cache.Size())
	}

	// Add entries
	cache.Set("key1", "value1")
	cache.Set("key2", "value2")

	if cache.Size() != 2 {
		t.Errorf("expected size 2, got %d", cache.Size())
	}

	// Delete one
	cache.Delete("key1")
	if cache.Size() != 1 {
		t.Errorf("expected size 1 after Delete, got %d", cache.Size())
	}

	// Clear
	cache.Clear()
	if cache.Size() != 0 {
		t.Errorf("expected size 0 after Clear, got %d", cache.Size())
	}
}

func TestCacheSizeIncludesExpired(t *testing.T) {
	cache := New[string](50 * time.Millisecond)

	// Set a value and let it expire
	cache.Set("key1", "value1")
	time.Sleep(100 * time.Millisecond)

	// Size should still include expired entries
	if cache.Size() != 1 {
		t.Errorf("expected size 1 (including expired), got %d", cache.Size())
	}

	// But Get should not return it
	_, found := cache.Get("key1")
	if found {
		t.Error("expected expired key to not be found via Get")
	}
}

func TestCacheOverwrite(t *testing.T) {
	cache := New[string](time.Minute)

	// Set initial value
	cache.Set("key1", "value1")

	// Overwrite with new value
	cache.Set("key1", "value2")

	val, found := cache.Get("key1")
	if !found {
		t.Error("expected key to be found")
	}
	if val != "value2" {
		t.Errorf("expected 'value2', got %q", val)
	}

	if cache.Size() != 1 {
		t.Errorf("expected size 1 after overwrite, got %d", cache.Size())
	}
}

func TestCacheDifferentTypes(t *testing.T) {
	// Test with int type
	intCache := New[int](time.Minute)
	intCache.Set("key1", 42)
	val, found := intCache.Get("key1")
	if !found || val != 42 {
		t.Errorf("expected 42, got %d (found=%v)", val, found)
	}

	// Test with struct type
	type TestStruct struct {
		Name  string
		Value int
	}
	structCache := New[TestStruct](time.Minute)
	structCache.Set("key1", TestStruct{Name: "test", Value: 123})
	sval, found := structCache.Get("key1")
	if !found || sval.Name != "test" || sval.Value != 123 {
		t.Errorf("expected {test 123}, got %+v (found=%v)", sval, found)
	}
}

func TestCacheConcurrentAccess(t *testing.T) {
	cache := New[int](time.Minute)

	// Run concurrent operations
	done := make(chan bool, 10)

	// Multiple goroutines writing
	for i := 0; i < 5; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				// nolint:gosec // id is between 0-4, so 'a'+id is always a valid rune
				cache.Set(string(rune('a'+id)), j)
			}
			done <- true
		}(i)
	}

	// Multiple goroutines reading
	for i := 0; i < 5; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				// nolint:gosec // id is between 0-4, so 'a'+id is always a valid rune
				cache.Get(string(rune('a' + id)))
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Cache should still be in a valid state
	if cache.Size() != 5 {
		t.Errorf("expected size 5, got %d", cache.Size())
	}
}
