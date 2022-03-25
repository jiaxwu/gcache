package gcache

import (
	"fmt"
	"log"
	"sync"
)

// Getter 用于加载数据
type Getter interface {
	Get(key string) ([]byte, error)
}

type GetterFunc func(key string) ([]byte, error)

func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

// Group 一个缓存命名空间
type Group struct {
	name      string
	getter    Getter
	mainCache cache
}

var (
	// 对全局group操作的锁
	mu sync.RWMutex
	// 缓存全局的group
	groups = make(map[string]*Group)
)

// NewGroup 创建一个Group
func NewGroup(name string, cacheBytes int, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter")
	}
	mu.Lock()
	defer mu.Unlock()
	g := &Group{
		name:   name,
		getter: getter,
		mainCache: cache{
			cacheBytes: cacheBytes,
		},
	}
	groups[name] = g
	return g
}

// GetGroup 从全局缓存获取Group
func GetGroup(name string) *Group {
	mu.RLock()
	defer mu.RUnlock()
	return groups[name]
}

// Get 从缓存获取key对应的value
func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}

	if v, ok := g.mainCache.get(key); ok {
		log.Println("[Cache] hit")
		return v, nil
	}
	return g.load(key)
}

// 加载缓存
func (g *Group) load(key string) (ByteView, error) {
	return g.loadLocally(key)
}

// 从本地节点加载缓存值
func (g *Group) loadLocally(key string) (ByteView, error) {
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err
	}
	value := ByteView{b: bytes}
	g.populateCache(key, value)
	return value, nil
}

// 发布到缓存
func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}
