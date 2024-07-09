package strategy

import "container/list"

// 当访问次数达到K次后，将数据索引从历史队列移到缓存队列中（缓存队列时间降序）
// 缓存数据中被访问后重新排序；需要淘汰数据时，淘汰缓存队列中排在末尾的数据
// 假使history和cache的缓存大小均为设置的maxBytes

type historyEntry struct {
	key   string
	value Value
	count int
}

type LRUKCache struct {
	K               int // the K setting
	historyMaxBytes int64
	historyNBytes   int64
	maxBytes        int64
	nBytes          int64
	historyList     *list.List
	cacheList       *list.List
	historyCache    map[string]*list.Element
	cache           map[string]*list.Element
}

func NewLRUKCache(k int, maxBytes int64) *LRUKCache {
	if k <= 1 {
		k = 2
	}

	return &LRUKCache{
		K:               k,
		historyMaxBytes: maxBytes,
		maxBytes:        maxBytes,
		historyNBytes:   0,
		nBytes:          0,
		historyList:     list.New(),
		cacheList:       list.New(),
		historyCache:    make(map[string]*list.Element),
		cache:           make(map[string]*list.Element),
	}
}
func (l *LRUKCache) Set(key string, value Value) {
	if ele, ok := l.cache[key]; ok {
		kv := ele.Value.(*historyEntry)
		kv.value = value
		l.cacheList.MoveToFront(ele)
		l.nBytes += int64(value.Len()) - int64(kv.value.Len())
		for l.nBytes >= l.maxBytes {
			l.RemoveFromCache()
		}
		return
	}

	if ele, ok := l.historyCache[key]; ok {
		kv := ele.Value.(*historyEntry)
		kv.value = value
		kv.count++
		if kv.count >= l.K {
			l.changeEntry(ele)
		}
	} else {
		item := l.historyList.PushFront(&historyEntry{key: key, value: value, count: 1})
		l.historyCache[key] = item
		l.historyNBytes += int64(len(key)) + int64(value.Len())
		for l.historyNBytes >= l.historyMaxBytes {
			l.RemoveFromHistory()
		}
	}

	return
}

func (l *LRUKCache) Get(key string) (Value, error) {
	if ele, ok := l.cache[key]; ok {
		kv := ele.Value.(*historyEntry)
		l.cacheList.MoveToFront(ele)
		return kv.value, nil
	}

	if ele, ok := l.historyCache[key]; ok {
		kv := ele.Value.(*historyEntry)
		kv.count++
		if kv.count >= l.K {
			l.changeEntry(ele)
		}
		return kv.value, nil
	}

	return nil, ErrKeyNotFound
}

func (l *LRUKCache) Remove(key string) error {
	if ele, ok := l.cache[key]; ok {
		kv := ele.Value.(*historyEntry)
		l.cacheList.Remove(ele)
		delete(l.cache, kv.key)
		l.nBytes -= int64(len(kv.key)) + int64(kv.value.Len())
		return nil
	}

	if ele, ok := l.historyCache[key]; ok {
		kv := ele.Value.(*historyEntry)
		l.historyList.Remove(ele)
		delete(l.historyCache, kv.key)
		l.historyNBytes -= int64(len(kv.key)) + int64(kv.value.Len())
		return nil
	}

	return ErrKeyNotFound
}

func (l *LRUKCache) Purge() {
	l.cache = nil
	l.historyList = nil
	l.historyCache = nil
	l.cacheList = nil
}

func (l *LRUKCache) Size() int64 {
	return l.nBytes
}

func (l *LRUKCache) MaxSize() int64 {
	return l.maxBytes
}

func (l *LRUKCache) RemoveFromCache() {
	if l.cacheList == nil || l.cacheList.Len() == 0 {
		return
	}

	ele := l.cacheList.Back()
	kv := ele.Value.(*historyEntry)
	l.cacheList.Remove(ele)
	delete(l.cache, kv.key)
	l.nBytes -= int64(len(kv.key)) + int64(kv.value.Len())

}

func (l *LRUKCache) RemoveFromHistory() {
	if l.historyList == nil || l.historyList.Len() == 0 {
		return
	}

	ele := l.historyList.Back()
	kv := ele.Value.(*historyEntry)
	l.historyList.Remove(ele)
	delete(l.historyCache, kv.key)
	l.historyNBytes -= int64(len(kv.key)) + int64(kv.value.Len())

}

// change from the item from the historyList to cacheList
// item is already in the historyList
func (l *LRUKCache) changeEntry(item *list.Element) {
	kv := item.Value.(*historyEntry)

	// delete item from the historyList
	l.historyList.Remove(item)
	delete(l.historyCache, kv.key)
	l.historyNBytes -= int64(len(kv.key)) + int64(kv.value.Len())

	// add item to the cacheList
	ele := l.cacheList.PushFront(kv)
	l.cache[kv.key] = ele
	l.nBytes += int64(len(kv.key)) + int64(kv.value.Len())
}
