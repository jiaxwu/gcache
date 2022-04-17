# gcache

缓存框架对比：https://cloud.tencent.com/developer/article/1967978

- 实现基于HTTP+protobuf的分布式缓存节点通信机制
- 使用一致性哈希算法解决Key路由和缓存雪崩问题
- 使用SingleFlight算法防止缓存击穿问题
- 实现缓存空值机制，解决缓存穿透问题
- 实现LRU缓存淘汰机制，避免内存无限增长
- 实现TTL机制，基于ZSet的惰性删除
- 实现基于ETCD的服务注册和发现，解决需要手动处理集群变化问题
- 实现远程HotKey的本地缓存机制，避免HotKey频繁网络请求带来的性能问题
