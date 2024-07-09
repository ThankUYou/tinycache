package strategy

import "container/list"

type lfuEntry struct {
	key   string
	value Value
	freq  int
}

type LFUCache struct {
	maxBytes int64
	nBytes   int64 // 已使用内存
	cache    map[string]*list.Element
	freq     map[int]*list.List
	minFreq  int
}

func NewLFUCache(maxBytes int64) *LFUCache {
	return &LFUCache{
		maxBytes: maxBytes,
		nBytes:   0,
		cache:    make(map[string]*list.Element),
		freq:     make(map[int]*list.List),
	}
}

func (lfu *LFUCache) Set(key string, value Value) {
	if ele, ok := lfu.cache[key]; ok {
		e := ele.Value.(*lfuEntry)
		lfu.nBytes += int64(value.Len()) - int64(e.value.Len())
		e.value = value
		lfu.Increment(ele)
	} else {
		item := &lfuEntry{key: key, value: value, freq: 1}
		if lfu.freq[1] == nil {
			lfu.freq[1] = list.New()
		}
		ele := lfu.freq[1].PushFront(item)
		lfu.cache[key] = ele
		lfu.minFreq = 1
		lfu.nBytes += int64(len(key)) + int64(item.value.Len())
	}

	if lfu.nBytes > lfu.maxBytes {
		lfu.RemoveLast()
	}
}

func (lfu *LFUCache) Get(key string) (Value, error) {
	if ele, ok := lfu.cache[key]; ok {
		e := ele.Value.(*lfuEntry)
		lfu.Increment(ele)
		return e.value, nil
	}
	return nil, ErrKeyNotFound
}

func (lfu *LFUCache) Remove(key string) error {
	if lfu.cache == nil {
		return ErrInvalidCache
	}

	if ele, ok := lfu.cache[key]; ok {
		e := ele.Value.(*lfuEntry)
		delete(lfu.cache, key)
		lfu.nBytes -= int64(len(e.key)) + int64(e.value.Len())
		lfu.freq[e.freq].Remove(ele)
		ele = nil
		return nil
	}
	return ErrKeyNotFound
}

func (lfu *LFUCache) Purge() {
	lfu.nBytes = 0
	lfu.cache = nil
	lfu.freq = nil
}

func (lfu *LFUCache) Size() int64 {
	return lfu.nBytes
}

func (lfu *LFUCache) MaxSize() int64 {
	return lfu.maxBytes
}

func (lfu *LFUCache) Increment(item *list.Element) {
	e := item.Value.(*lfuEntry)
	freq := e.freq
	lfu.freq[freq].Remove(item)

	if lfu.minFreq == freq && lfu.freq[freq].Len() == 0 {
		lfu.minFreq++
		delete(lfu.freq, freq)
	}

	e.freq++
	if lfu.freq[e.freq] == nil {
		lfu.freq[e.freq] = list.New()
	}
	lfu.freq[e.freq].PushFront(e)
}

func (lfu *LFUCache) RemoveLast() {
	if lfu.minFreq == 0 || lfu.cache == nil || lfu.freq == nil {
		return
	}
	// TODO: fix bug
	defer func() {
		lfu.minFreq++
	}()

	if l, ok := lfu.freq[lfu.minFreq]; ok {
		if l == nil {
			return
		}

		if l.Len() == 0 {
			delete(lfu.freq, lfu.minFreq)
			return
		}

		l.Remove(l.Back()) // delete the oldest entry
		return
	}
}
