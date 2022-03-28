package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

// Hash 映射bytes到uint32，用于散列键
type Hash func(date []byte) uint32

// Map 包含所有被散列的键
type Map struct {
	hash Hash
	// 虚拟节点倍数
	replicas int
	// 哈希环，存的都是虚拟节点的hash
	keys []int
	// 虚拟节点hash到真实节点名称的映射
	hashMap map[int]string
}

// New 创建一个一致性哈希
func New(replicas int, fn Hash) *Map {
	m := &Map{
		replicas: replicas,
		hash:     fn,
		hashMap:  make(map[int]string),
	}
	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE
	}
	return m
}

// Add 添加节点到一致性哈希里
func (m *Map) Add(keys ...string) {
	m.resetKeys(func(key string, hash int) {
		m.hashMap[hash] = key
	}, keys)
}

// Delete 从一致性哈希删除节点
func (m *Map) Delete(keys ...string) {
	m.resetKeys(func(_ string, hash int) {
		delete(m.hashMap, hash)
	}, keys)
}

// 重置keys
func (m *Map) resetKeys(fn func(key string, hash int), keys []string) {
	for _, key := range keys {
		for i := 0; i < m.replicas; i++ {
			hash := int(m.hash([]byte(strconv.Itoa(i) + key)))
			fn(key, hash)
		}
	}
	newKeys := make([]int, 0, len(m.hashMap))
	for key := range m.hashMap {
		newKeys = append(newKeys, key)
	}
	m.keys = newKeys
	sort.Ints(m.keys)
}

// Get 获取第一个哈希值大于等于键的节点
func (m *Map) Get(key string) string {
	if len(m.keys) == 0 {
		return ""
	}
	hash := int(m.hash([]byte(key)))
	idx := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hash
	})
	virtualNode := m.keys[idx%len(m.keys)]
	return m.hashMap[virtualNode]
}
