package gcache

import (
	"fmt"
	pb "github.com/jiaxwu/gcache/gcachepb"
	"golang.org/x/sync/singleflight"
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
	// 用于获取远程节点请求客户端
	peers PeerPicker
	// 避免对同一个key多次加载
	loader *singleflight.Group
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
		loader: &singleflight.Group{},
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

// RegisterPeers 注册获取远程节点请求客户端的PeerPicker
func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("register peer picker called more than once")
	}
	g.peers = peers
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
	view, err, _ := g.loader.Do(key, func() (interface{}, error) {
		// 先判断是否需要从远程加载
		if g.peers != nil {
			if peer, ok := g.peers.PickPeer(key); ok {
				value, err := g.loadFromPeer(peer, key)
				if err == nil {
					return value, nil
				}
				log.Printf("[Cache] failed to get from peer key=%s, err=%v\n", key, err)
			}
		}
		// 否则从本地加载
		return g.loadLocally(key)
	})
	if err != nil {
		return ByteView{}, err
	}
	return view.(ByteView), nil
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

// 从远程加载缓存值
func (g *Group) loadFromPeer(peer PeerGetter, key string) (ByteView, error) {
	req := &pb.Request{
		Group: g.name,
		Key:   key,
	}
	var res pb.Response
	err := peer.Get(req, &res)
	if err != nil {
		return ByteView{}, err
	}
	return ByteView{b: res.Value}, nil
}
