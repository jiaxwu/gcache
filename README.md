# gcache

- 实现LRU缓存淘汰机制，避免内存无限增长
- 实现基于HTTP+protobuf的分布式缓存节点通信机制
- 使用一致性哈希算法解决Key路由和缓存雪崩问题
- 使用SingleFlight算法防止缓存击穿问题
- 实现Key的TTL机制，基于Key和ZSet的惰性删除