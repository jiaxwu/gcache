package lru

import (
	"testing"
)

type String string

func (s String) Len() int {
	return len(s)
}

func TestCache_Get(t *testing.T) {
	lru := New(0, nil)
	testKey, testValue := "key1", String("value1")
	lru.Add(testKey, testValue)
	if value, ok := lru.Get(testKey); !ok || value.(String) != testValue {
		t.Fatalf("cache hit %s:%s failed\n", testKey, testValue)
	}
	notCacheKey := "key2"
	if _, ok := lru.Get(notCacheKey); ok {
		t.Fatalf("hit not cache key=%s\n", notCacheKey)
	}
}

func TestCache_RemoveOldest(t *testing.T) {
	k1, k2, k3 := "key1", "key2", "key3"
	v1, v2, v3 := String("value1"), String("value2"), String("value3")
	capacity := len(k1) + len(k2) + v1.Len() + v2.Len()
	lru := New(capacity, nil)
	lru.Add(k1, v1)
	lru.Add(k2, v2)
	lru.Add(k3, v3)

	if _, ok := lru.Get(k1); ok || lru.Len() != 2 {
		t.Fatalf("remove oldest %s:%s failed, len=%d\n", k1, v1, lru.Len())
	}
}

func TestCache_Remove(t *testing.T) {
	k1, k2, k3 := "key1", "key2", "key3"
	v1, v2, v3 := String("value1"), String("value2"), String("value3")
	capacity := len(k1) + len(k2) + v1.Len() + v2.Len()
	lru := New(capacity, nil)
	lru.Add(k1, v1)
	lru.Add(k2, v2)
	lru.Add(k3, v3)

	if _, ok := lru.Get(k1); ok || lru.Len() != 2 {
		t.Fatalf("remove oldest %s:%s failed, len=%d\n", k1, v1, lru.Len())
	}
	lru.Remove(k2)
	if _, ok := lru.Get(k2); ok || lru.Len() != 1 {
		t.Fatalf("remove key %s:%s failed, len=%d\n", k2, v2, lru.Len())
	}
	if val, ok := lru.Get(k3); !ok || val != v3 {
		t.Fatalf("get key %s:%s failed, len=%d\n", k3, v3, lru.Len())
	}
}

func TestCache_OnEvicted(t *testing.T) {
	k1, k2, k3 := "key1", "key2", "key3"
	v1, v2, v3 := String("value1"), String("value2"), String("value3")
	capacity := len(k1) + len(k2) + v1.Len() + v2.Len()
	var evictedKey string
	var evictedValue Value
	lru := New(capacity, func(key string, value Value) {
		evictedKey = key
		evictedValue = value
	})
	lru.Add(k1, v1)
	lru.Add(k2, v2)
	lru.Add(k3, v3)
	if value, ok := evictedValue.(String); !ok || evictedKey != k1 || value != v1 {
		t.Fatalf("evicted failed; evicted key = %s, value = %s\n", evictedKey, value)
	}
}
