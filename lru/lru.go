package lru

import (
	"container/list"
	"time"
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
}

type entry struct {
	key   string
	value Value
	// 过期时间
	expire time.Time
}

// Value 用于计算值占用了多少字节和过期时间
type Value interface {
	Len() int
}

func New(maxBytes int, onEvicted func(key string, value Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		onEvicted: onEvicted,
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
	if !ent.expire.IsZero() && ent.expire.Before(time.Now()) {
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
func (c *Cache) Add(key string, value Value, expire time.Time) {
	if element, ok := c.cache[key]; ok {
		c.ll.MoveToBack(element)
		kv := element.Value.(*entry)
		c.nBytes += value.Len() - kv.value.Len()
		kv.value = value
	} else {
		element := c.ll.PushBack(&entry{
			key:    key,
			value:  value,
			expire: expire,
		})
		c.cache[key] = element
		c.nBytes += len(key) + value.Len()
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

// 移除某个键，并删除链表里面的节点，减少lru缓存大小，调用回调函数
func (c *Cache) removeElement(e *list.Element) {
	c.ll.Remove(e)
	kv := e.Value.(*entry)
	delete(c.cache, kv.key)
	c.nBytes -= len(kv.key) + kv.value.Len()
	if c.onEvicted != nil {
		c.onEvicted(kv.key, kv.value)
	}
}

// Len 返回数据数量
func (c *Cache) Len() int {
	return c.ll.Len()
}
