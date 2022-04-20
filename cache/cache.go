package cache

type Cache[K comparable, V any] interface {
	// Get 获取元素
	Get(key K) (val V, exist bool)
	// Set 设置元素
	Set(key K, val V) (origVal V, origExist bool)
	// Del 删除元素
	Del(key K) (origVal V, origExist bool)
	// Evict 驱逐元素
	Evict() (evictedKey K, evictedVal V, evicted bool)
	// Len 缓存元素个数
	Len() int
}
