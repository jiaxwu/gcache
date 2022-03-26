package gcache

import (
	"github.com/jiaxwu/gcache/lru"
	"sync"
)

const (
	// 每次移除过期键数量
	removeExpireN = 10
)

// 并发安全的缓存操作
type cache struct {
	mu         sync.Mutex
	lru        *lru.Cache
	cacheBytes int
}

func (c *cache) add(key string, value ByteView) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lru == nil {
		c.lru = lru.New(c.cacheBytes, nil)
	}
	c.lru.RemoveExpire(removeExpireN)
	c.lru.Add(key, value)
}

func (c *cache) get(key string) (ByteView, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lru == nil {
		return ByteView{}, false
	}
	c.lru.RemoveExpire(removeExpireN)
	if v, ok := c.lru.Get(key); ok {
		return v.(ByteView), ok
	}
	return ByteView{}, false
}

func (c *cache) remove(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lru == nil {
		return
	}
	c.lru.RemoveExpire(removeExpireN)
	c.lru.Remove(key)
}
