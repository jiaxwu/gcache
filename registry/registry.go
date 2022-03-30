package registry

import (
	"context"
	"fmt"
	"go.etcd.io/etcd/api/v3/mvccpb"
	etcd "go.etcd.io/etcd/client/v3"
	"sync"
	"time"
)

const (
	// 续约间隔，单位秒
	keepAliveTTL = 10
	// 事件通道缓冲区大小
	eventChanSize = 10
)

// Event 服务变化事件
type Event struct {
	AddAddr    string
	DeleteAddr string
}

// Registry 名字服务
type Registry struct {
	// etcd服务器地址
	endpoints []string
	mu        sync.Mutex
	client    *etcd.Client
	// etcd名字服务key前缀
	prefix string
}

func New(prefix string, endpoints []string) (*Registry, error) {
	client, err := etcd.New(etcd.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, err
	}
	return &Registry{
		endpoints: endpoints,
		client:    client,
		prefix:    prefix,
	}, nil
}

// Register 注册服务
func (r *Registry) Register(ctx context.Context, addr string) error {
	kv := etcd.NewKV(r.client)
	lease := etcd.NewLease(r.client)
	grant, err := lease.Grant(ctx, keepAliveTTL)
	if err != nil {
		return err
	}
	key := fmt.Sprintf("%s%s", r.prefix, addr)
	if _, err := kv.Put(ctx, key, addr, etcd.WithLease(grant.ID)); err != nil {
		return err
	}
	ch, err := lease.KeepAlive(ctx, grant.ID)
	if err != nil {
		return err
	}
	go func() {
		for range ch {
		}
	}()
	return nil
}

// GetAddrs 获取节点地址列表
func (r *Registry) GetAddrs(ctx context.Context) ([]string, error) {
	kv := etcd.NewKV(r.client)
	resp, err := kv.Get(ctx, r.prefix, etcd.WithPrefix())
	if err != nil {
		return nil, err
	}
	addrs := make([]string, len(resp.Kvs))
	for i, kv := range resp.Kvs {
		addrs[i] = string(kv.Value)
	}
	return addrs, nil
}

// Watch 发现服务
func (r *Registry) Watch(ctx context.Context) <-chan Event {
	watcher := etcd.NewWatcher(r.client)
	watchChan := watcher.Watch(ctx, r.prefix, etcd.WithPrefix())
	ch := make(chan Event, eventChanSize)
	go func() {
		for watchRsp := range watchChan {
			for _, event := range watchRsp.Events {
				switch event.Type {
				case mvccpb.PUT:
					ch <- Event{AddAddr: string(event.Kv.Value)}
				case mvccpb.DELETE:
					ch <- Event{DeleteAddr: string(event.Kv.Key[len(r.prefix):])}
				}
			}
		}
		close(ch)
	}()
	return ch
}
