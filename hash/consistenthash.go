package hash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

// Hash 定义了函数类型，采取依赖注入的方式，允许用于替换成自定义的 Hash 函数，也方便测试时替换，默认为 crc32.ChecksumIEEE 算法。
type Hash func(data []byte) uint32

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

// Add 添加缓存节点，允许传入 0 或 多个真实节点的名称。
// 对每一个真实节点 key，对应创建 m.replicas 个虚拟节点，虚拟节点的名称是：strconv.Itoa(i) + key，即通过添加编号的方式区分不同虚拟节点。
// 使用 m.hash() 计算虚拟节点的哈希值，使用 append(m.keys, hash) 添加到环上。
// 在 hashMap 中增加虚拟节点和真实节点的映射关系。
// 最后一步，环上的哈希值排序。
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

// Get 获取缓存节点，通过key获取真实节点
// 第一步，计算 key 的哈希值。
// 第二步，顺时针找到第一个匹配的虚拟节点的下标 idx，从 m.keys 中获取到对应的哈希值。
// 如果 idx == len(m.keys)，说明应选择 m.keys[0]，因为 m.keys 是一个环状结构，所以用取余数的方式来处理这种情况。
// 第三步，通过 hashMap 映射得到真实的节点。
func (m *Map) Get(key string) string {
	if len(key) == 0 {
		return ""
	}

	hash := int(m.hash([]byte(key)))

	idx := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hash
	}) // 在哈希环上顺时针找到第一个大于等于这个哈希值的虚拟节点的下标

	return m.hashmap[m.keys[idx%len(m.keys)]] // 找到真实节点映射
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
