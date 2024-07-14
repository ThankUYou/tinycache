package lru

import (
	"container/list"
	"time"
)

type Value interface {
	Len() int
} // 为了通用性，我们允许值是实现了 Value 接口的任意类型，该接口只包含了一个方法 Len() int，用于返回值所占用的内存大小。

// LRUCache 定义了一个结构体，用来实现lru缓存淘汰算法
type LRUCache struct {
	maxBytes   int64                         // 允许使用的最大内存
	nBytes     int64                         // 当前已经使用的内存
	ll         *list.List                    // 双向链表常用于维护缓存中各个数据的访问顺序，以便在淘汰数据时能够方便地找到最近最少使用的数据
	cache      map[string]*list.Element      // 键是字符串，值是双向链表中对应节点的指针
	OnEvicted  func(key string, value Value) // 某条记录被移除时的回调函数，可以为 nil
	defaultTTL time.Duration                 // 记录在缓存中的默认过期时间
}

// 键值对 entry 是双向链表节点的数据类型，在链表中仍保存每个值对应的 key 的好处在于，淘汰队首节点时，需要用 key 从字典中删除对应的映射。
type entry struct {
	key    string
	value  Value
	expire time.Time //节点的过期时间
}

// New 通过传入maxBytes,onEvicted,defaultTTL这些参数，返回一个LRUCache结构体。
func New(maxBytes int64, onEvicted func(string, Value), defaultTTL time.Duration) *LRUCache {
	return &LRUCache{
		maxBytes:   maxBytes,
		ll:         list.New(),
		cache:      make(map[string]*list.Element),
		OnEvicted:  onEvicted,
		defaultTTL: defaultTTL,
	}
}

func (c *LRUCache) Get(key string) (value Value, ok bool) {
	if ele, ok := c.cache[key]; ok {
		kv := ele.Value.(*entry)
		if kv.expire.Before(time.Now()) {

		}
		c.ll.MoveToFront(ele)
		return kv.value, true
	}
	return
}

// Add 方法用于向缓存中添加新的键值对。如果键已存在，则更新对应的值，并将节点移动到链表的最前面；
// 如果键不存在，则在链表头部插入新的节点，并更新已占用的容量。
// 如果添加新的键值对后超出了最大存储容量，则会连续移除最久未使用的记录，直到满足容量要求。
func (c *LRUCache) Add(key string, value Value, ttl time.Duration) {
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		c.nBytes += int64(value.Len()) - int64(kv.value.Len())
		kv.value = value
		// 更新过期时间时，判断是否应该保留原本的过期时间
		if kv.expire.Before(time.Now().Add(ttl)) {
			kv.expire = time.Now().Add(ttl)
		}
	} else {
		ele = c.ll.PushFront(&entry{key: key, value: value, expire: time.Now().Add(ttl)})
		c.cache[key] = ele
		c.nBytes += int64(len(key)) + int64(value.Len())
	}
	for c.maxBytes != 0 && c.maxBytes < c.nBytes {
		c.RemoveOldest()
	}
}

// Len 方法返回当前缓存中的记录数量。
func (c *LRUCache) Len() int {
	return c.ll.Len()
}

// RemoveOldest 方法用于移除最近最少访问的节点（队首节点）
func (c *LRUCache) RemoveOldest() {
	for e := c.ll.Back(); e != nil; e = e.Prev() {
		kv := e.Value.(*entry)
		if kv.expire.Before(time.Now()) {
			c.RemoveElement(e)
			break
		}
	}
}

// RemoveElement 函数用于删除某个节点
func (c *LRUCache) RemoveElement(e *list.Element) {
	c.ll.Remove(e)
	kv := e.Value.(*entry)
	delete(c.cache, kv.key)                                //删除key-节点这对映射
	c.nBytes -= int64(len(kv.key)) + int64(kv.value.Len()) //重新计算已用容量
	if c.OnEvicted != nil {
		c.OnEvicted(kv.key, kv.value) //调用对应的回调函数
	}
}
