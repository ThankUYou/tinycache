package hash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

type Hash func(data []byte) uint32 //便于选择不同的哈希算法

type Map struct {
	hash     Hash           //一致性哈希的哈希算法
	replicas int            //虚拟节点倍数
	keys     []int          //哈希环
	hashmap  map[int]string //虚拟节点和真实节点的映射关系
}

func NewConsistentHash(replicas int, hash Hash) *Map {
	m := &Map{
		hash:     hash,
		replicas: replicas,
		hashmap:  make(map[int]string),
	}

	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE
	}
	return m
}

// Add 添加缓存节点
func (m *Map) Add(keys ...string) { // 真实节点的地址
	for _, key := range keys {
		for i := 0; i < m.replicas; i++ { // 对于每一个真实节点的地址，创建对应的虚拟节点的名字， strconv.Itoa(i) + key
			hash := int(m.hash([]byte(strconv.Itoa(i) + key)))
			m.keys = append(m.keys, hash) // 将虚拟节点添加到哈希环上
			m.hashmap[hash] = key         // 添加虚拟节点和真实节点的映射关系，可以通过虚拟节点找到真实节点
		}
	}
	sort.Ints(m.keys)
}

// Get 获取缓存节点
func (m *Map) Get(key string) string {
	if len(key) == 0 {
		return ""
	}

	hash := int(m.hash([]byte(key)))

	idx := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hash
	}) // 在哈希环上顺时针找到第一个大于等于这个哈希值的虚拟节点的下标

	return m.hashmap[m.keys[idx%len(m.keys)]] // 找到真是节点映射
}

// Remove 移除缓存节点
func (m *Map) Remove(key string) {
	for i := 0; i < m.replicas; i++ {
		hash := int(m.hash([]byte(strconv.Itoa(i) + key)))
		idx := sort.SearchInts(m.keys, hash)
		m.keys = append(m.keys[:idx], m.keys[idx+1:]...)
		delete(m.hashmap, hash)
	}
}
