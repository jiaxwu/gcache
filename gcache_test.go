package gcache

import (
	"fmt"
	"log"
	"testing"
)

func TestGetter(t *testing.T) {
	var f Getter = GetterFunc(func(key string) ([]byte, error) {
		return []byte(key), nil
	})
	key1 := "key1"
	if value, err := f.Get(key1); err != nil || string(value) != key1 {
		t.Errorf("getter expect %s but %s\n", key1, value)
	}
}

func TestGroup_Get(t *testing.T) {
	var db = map[string]string{
		"Tom":  "630",
		"Jack": "589",
		"Sam":  "567",
	}
	loadCounts := make(map[string]int, len(db))
	g := NewGroup("scores", 2<<10, GetterFunc(func(key string) ([]byte, error) {
		log.Printf("[SlowDB] sear key %s\n", key)
		if v, ok := db[key]; ok {
			loadCounts[key]++
			return []byte(v), nil
		}
		return nil, fmt.Errorf("%s does not exists\n", key)
	}))

	for k, v := range db {
		if view, err := g.Get(k); err != nil || view.String() != v {
			t.Fatalf("failed to get value of key %s\n", k)
		}
		if _, err := g.Get(k); err != nil || loadCounts[k] > 1 {
			t.Fatalf("cache key %s miss key\n", k)
		}
	}

	if view, err := g.Get("unknown"); err == nil {
		t.Fatalf("the value of key unknown should be empty, but %s got\n", view.String())
	}
}
