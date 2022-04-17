package lru

import (
	"testing"
	"time"
)

type String struct {
	s      string
	expire time.Time
}

func (s *String) Len() int {
	return len(s.s)
}

func (s *String) Expire() time.Time {
	return s.expire
}

func TestCache_Get(t *testing.T) {
	lru := New(0, nil)
	testKey, testValue := "key1", &String{
		s:      "value1",
		expire: time.Time{},
	}
	lru.Add(testKey, testValue)
	if value, ok := lru.Get(testKey); !ok || value.(*String) != testValue {
		t.Fatalf("cache hit %v:%v failed\n", testKey, testValue)
	}
	notCacheKey := "key2"
	if _, ok := lru.Get(notCacheKey); ok {
		t.Fatalf("hit not cache key=%s\n", notCacheKey)
	}
}

func TestCache_RemoveOldest(t *testing.T) {
	k1, k2, k3 := "key1", "key2", "key3"
	v1, v2, v3 := &String{s: "value1"}, &String{s: "value2"}, &String{s: "value3"}
	capacity := len(k1) + len(k2) + v1.Len() + v2.Len()
	lru := New(capacity, nil)
	lru.Add(k1, v1)
	lru.Add(k2, v2)
	lru.Add(k3, v3)

	if _, ok := lru.Get(k1); ok || lru.Len() != 2 {
		t.Fatalf("remove oldest %v:%v failed, len=%d\n", k1, v1, lru.Len())
	}
}

func TestCache_Remove(t *testing.T) {
	k1, k2, k3 := "key1", "key2", "key3"
	v1, v2, v3 := &String{s: "value1"}, &String{s: "value2"}, &String{s: "value3"}
	capacity := len(k1) + len(k2) + v1.Len() + v2.Len()
	lru := New(capacity, nil)
	lru.Add(k1, v1)
	lru.Add(k2, v2)
	lru.Add(k3, v3)

	if _, ok := lru.Get(k1); ok || lru.Len() != 2 {
		t.Fatalf("remove oldest %v:%v failed, len=%d\n", k1, v1, lru.Len())
	}
	lru.Remove(k2)
	if _, ok := lru.Get(k2); ok || lru.Len() != 1 {
		t.Fatalf("remove key %v:%v failed, len=%d\n", k2, v2, lru.Len())
	}
	if val, ok := lru.Get(k3); !ok || val != v3 {
		t.Fatalf("get key %v:%v failed, len=%d\n", k3, v3, lru.Len())
	}
}

func TestCache_OnEvicted(t *testing.T) {
	k1, k2, k3 := "key1", "key2", "key3"
	v1, v2, v3 := &String{s: "value1"}, &String{s: "value2"}, &String{s: "value3"}
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
	if value, ok := evictedValue.(*String); !ok || evictedKey != k1 || value != v1 {
		t.Fatalf("evicted failed; evicted key = %v, value = %v\n", evictedKey, value)
	}
}

func TestCache_Expire(t *testing.T) {
	k1, k2, k3 := "key1", "key2", "key3"
	v1, v2, v3 := &String{s: "value1"}, &String{s: "value2", expire: time.Now().Add(time.Second)}, &String{s: "value3"}
	capacity := len(k1) + len(k2) + v1.Len() + v2.Len()
	lru := New(capacity, nil)
	lru.Add(k1, v1)
	lru.Add(k2, v2)
	lru.Add(k3, v3)

	if _, ok := lru.Get(k1); ok || lru.Len() != 2 {
		t.Fatalf("remove oldest %v:%v failed, len=%d\n", k1, v1, lru.Len())
	}
	if _, ok := lru.Get(k2); !ok || lru.Len() != 2 {
		t.Fatalf("get %v:%v failed, len=%d\n", k2, v2, lru.Len())
	}
	time.Sleep(time.Second)
	if _, ok := lru.Get(k2); ok || lru.Len() != 1 {
		t.Fatalf("expire %v:%v failed, len=%d\n", k2, v2, lru.Len())
	}
}

func TestCache_RemoveExpire(t *testing.T) {
	k1, k2, k3 := "key1", "key2", "key3"
	v1, v2, v3 := &String{s: "value1", expire: time.Now().Add(time.Second * 3)},
		&String{s: "value2", expire: time.Now().Add(time.Second * 2)},
		&String{s: "value3", expire: time.Now().Add(time.Second * 1)}
	lru := New(1000, nil)
	lru.Add(k1, v1)
	lru.Add(k2, v2)
	lru.Add(k3, v3)

	if _, ok := lru.Get(k1); !ok || lru.Len() != 3 {
		t.Fatalf("get %v:%v failed, len=%d\n", k1, v1, lru.Len())
	}
	if _, ok := lru.Get(k2); !ok || lru.Len() != 3 {
		t.Fatalf("get %v:%v failed, len=%d\n", k2, v2, lru.Len())
	}
	time.Sleep(time.Second * 2)
	lru.removeExpire(10)
	if _, ok := lru.Get(k1); !ok || lru.Len() != 1 {
		t.Fatalf("remove expire keys failed, len=%d\n", lru.Len())
	}
}
