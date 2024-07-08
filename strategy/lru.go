package strategy

import "container/list"

// lruEntry 键值对
type lruEntry struct {
	key   string
	value Value
}

type LRUCache struct {
	maxBytes int64 //允许使用的最大内存
	nBytes   int64 //当前已经使用的内存
	ll       *list.List
	cache    map[string]*list.Element
	//OnEvicted func(key string, value Value) //回调函数
}

// NewLRUCache 实例化
func NewLRUCache(maxBytes int64) *LRUCache {
	return &LRUCache{
		maxBytes: maxBytes, //允许使用的最大内存
		nBytes:   0,
		ll:       list.New(),
		cache:    make(map[string]*list.Element),
	}
}

// Set 插入
// 如果键存在，则更新对应节点的值，并将该节点移到队尾。
// 不存在则是新增场景，首先队尾添加新节点 &entry{key, value}, 并字典中添加 key 和节点的映射关系。
// 更新 c.nbytes，如果超过了设定的最大值 c.maxBytes，则移除最少访问的节点。
func (lru *LRUCache) Set(key string, value Value) {
	if ele, ok := lru.cache[key]; ok {
		lru.ll.MoveToFront(ele)
		kv := ele.Value.(*lruEntry)
		lru.nBytes += int64(value.Len()) - int64(kv.value.Len())
		kv.value = value
	} else {
		ele := lru.ll.PushFront(&lruEntry{key: key, value: value})
		lru.cache[key] = ele
		lru.nBytes += int64(len(key)) + int64(value.Len())
	}

	for lru.maxBytes <= lru.nBytes {
		lru.removeOldest()
	}
}

// Get 查询
// 如果键对应的链表节点存在，则将对应节点移动到队尾，并返回查找到的值。
// 将链表中的节点 ele 移动到队尾（双向链表作为队列，队首队尾是相对的，在这里约定 front 为队尾）
func (lru *LRUCache) Get(key string) (Value, error) {
	if ele, ok := lru.cache[key]; ok {
		lru.ll.MoveToFront(ele)
		kv := (ele.Value).(*lruEntry)
		return kv.value, nil
	}
	return nil, ErrKeyNotFound
}

// Remove 删除
func (lru *LRUCache) Remove(key string) error {
	if ele, ok := lru.cache[key]; ok {
		lru.ll.Remove(ele)
		kv := ele.Value.(*lruEntry)
		delete(lru.cache, kv.key)
		lru.nBytes -= int64(len(kv.key)) + int64(kv.value.Len()) // 更新已使用内存
		return nil
	}
	return ErrKeyNotFound
}

func (lru *LRUCache) Size() int64 {
	return lru.nBytes
}

func (lru *LRUCache) MaxSize() int64 {
	return lru.maxBytes
}

func (lru *LRUCache) Purge() {
	lru.ll = nil
	lru.cache = nil
}

// Removes 移除最近最少访问的节点（队首）
func (lru *LRUCache) removeOldest() {
	ele := lru.ll.Back() // 取到队首节点，从链表中删除
	if ele != nil {
		lru.ll.Remove(ele)
		kv := ele.Value.(*lruEntry)
		delete(lru.cache, kv.key)
		lru.nBytes -= int64(len(kv.key)) + int64(kv.value.Len())
	}
}
