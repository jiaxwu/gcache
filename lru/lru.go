package lru

import (
	"container/list"
	"github.com/jiaxwu/gcache/zset"
	"time"
)

const (
	expiresZSetKey = ""
)

// Cache LRU缓存
type Cache struct {
	// 最大缓存字节数
	maxBytes int
	// 已经缓存字节数
	nBytes int
	ll     *list.List
	cache  map[string]*list.Element
	// 可选，在entry被移除的时候执行
	onEvicted func(key string, value Value)
	// 过期键集合
	expires *zset.SortedSet
}

type entry struct {
	key   string
	value Value
}

// Value 用于计算值占用了多少字节和过期时间
type Value interface {
	Len() int
	Expire() time.Time
}

func New(maxBytes int, onEvicted func(key string, value Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		onEvicted: onEvicted,
		expires:   zset.New(),
	}
}

// Get 获取缓存的值
func (c *Cache) Get(key string) (Value, bool) {
	element, ok := c.cache[key]
	if !ok {
		return nil, false
	}
	ent := element.Value.(*entry)
	// 移除过期的键
	if !ent.value.Expire().IsZero() && ent.value.Expire().Before(time.Now()) {
		c.removeElement(element)
		return nil, false
	}
	c.ll.MoveToBack(element)
	return ent.value, true
}

// RemoveOldest 移除最近最少访问的数据
func (c *Cache) RemoveOldest() {
	front := c.ll.Front()
	if front != nil {
		c.removeElement(front)
	}
}

// Add 添加数据到缓存
func (c *Cache) Add(key string, value Value) {
	if element, ok := c.cache[key]; ok {
		c.ll.MoveToBack(element)
		ent := element.Value.(*entry)
		c.nBytes += value.Len() - ent.value.Len()
		ent.value = value
	} else {
		ent := &entry{
			key:   key,
			value: value,
		}
		element := c.ll.PushBack(ent)
		c.cache[key] = element
		c.nBytes += len(key) + value.Len()
	}
	// 如果有超时时间则设置
	if !value.Expire().IsZero() {
		c.expires.ZAdd(expiresZSetKey, value.Expire().UnixNano(), key)
	} else {
		// 没有则删除
		c.expires.ZRem(expiresZSetKey, key)
	}
	for c.maxBytes != 0 && c.nBytes > c.maxBytes {
		c.RemoveOldest()
	}
}

// Remove 移除某个键
func (c *Cache) Remove(key string) {
	if element, ok := c.cache[key]; ok {
		c.removeElement(element)
	}
}

// Len 返回数据数量
func (c *Cache) Len() int {
	return c.ll.Len()
}

// 移除某个键，并删除链表里面的节点，减少lru缓存大小，删除过期时间，调用回调函数
func (c *Cache) removeElement(e *list.Element) {
	c.ll.Remove(e)
	kv := e.Value.(*entry)
	delete(c.cache, kv.key)
	c.nBytes -= len(kv.key) + kv.value.Len()
	// 移除过期键
	if !kv.value.Expire().IsZero() {
		c.expires.ZRem(expiresZSetKey, kv.key)
	}
	if c.onEvicted != nil {
		c.onEvicted(kv.key, kv.value)
	}
}

// RemoveExpire 移除过期的键
func (c *Cache) RemoveExpire(n int) {
	for n > 0 && c.expires.ZCard(expiresZSetKey) > 0 {
		values := c.expires.ZRangeWithScores(expiresZSetKey, 0, 0)
		key, expireNano := values[0].(string), values[1].(int64)
		// 第一个键都没超时，结果循环
		if expireNano > time.Now().UnixNano() {
			return
		}
		c.Remove(key)
		n--
	}
}
