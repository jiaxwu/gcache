package gcache

import (
	"errors"
	"fmt"
	pb "github.com/jiaxwu/gcache/gcachepb"
	"golang.org/x/sync/singleflight"
	"log"
	"sync"
	"time"
)

// Getter 用于加载数据
type Getter interface {
	Get(key string) (ByteView, error)
}

type GetterFunc func(key string) (ByteView, error)

func (f GetterFunc) Get(key string) (ByteView, error) {
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
	loadGroup *singleflight.Group
	// 避免对同一个key多次删除
	removeGroup *singleflight.Group
	// getter返回error时对应空值key的过期时间
	emptyKeyDuration time.Duration
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
		loadGroup:   &singleflight.Group{},
		removeGroup: &singleflight.Group{},
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

// SetEmptyWhenError 当getter返回error时设置空值，缓解缓存穿透问题
// 为0表示该机制不生效
func (g *Group) SetEmptyWhenError(duration time.Duration) {
	g.emptyKeyDuration = duration
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

// Remove 从缓存删除key
func (g *Group) Remove(key string) error {
	_, err, _ := g.loadGroup.Do(key, func() (interface{}, error) {
		// 先判断是否需要从远程删除
		if g.peers != nil {
			if peer, ok := g.peers.PickPeer(key); ok {
				return nil, g.removeFromPeer(peer, key)
			}
		}
		// 否则从本地删除
		g.removeLocally(key)
		return nil, nil
	})
	return err
}

// 加载缓存
func (g *Group) load(key string) (ByteView, error) {
	view, err, _ := g.loadGroup.Do(key, func() (interface{}, error) {
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
	value, err := g.getter.Get(key)
	if err != nil {
		if g.emptyKeyDuration == 0 {
			return ByteView{}, err
		}
		// 走缓存空值机制
		value = ByteView{
			expire: time.Now().Add(g.emptyKeyDuration),
		}
	}
	g.populateCache(key, value)
	return value, nil
}

// 从本地节点删除缓存
func (g *Group) removeLocally(key string) {
	g.mainCache.remove(key)
}

// 发布到缓存
func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}

// 从远程节点加载缓存值
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
	var expire time.Time
	if res.Expire != 0 {
		expire = time.Unix(res.Expire/int64(time.Second), res.Expire%int64(time.Second))
		if time.Now().After(expire) {
			return ByteView{}, errors.New("peer returned expired value")
		}
	}
	return ByteView{b: res.Value, expire: expire}, nil
}

// 从远程节点删除缓存值
func (g *Group) removeFromPeer(peer PeerGetter, key string) error {
	req := &pb.Request{
		Group: g.name,
		Key:   key,
	}
	return peer.Remove(req)
}
