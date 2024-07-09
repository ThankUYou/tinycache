package strategy

import "container/list"

type fifoEntry struct {
	key   string
	value Value
}

type FIFOCache struct {
	maxBytes int64
	nBytes   int64
	ll       *list.List
	cache    map[string]*list.Element
}

func NewFIFOCache(maxBytes int64) *FIFOCache {
	return &FIFOCache{
		maxBytes: maxBytes,
		nBytes:   0,
		ll:       list.New(),
		cache:    make(map[string]*list.Element),
	}
}

func (fifo *FIFOCache) Set(key string, value Value) {
	if ele, ok := fifo.cache[key]; ok {
		kv := ele.Value.(*fifoEntry)
		kv.value = value
		fifo.nBytes += int64(value.Len()) - int64(kv.value.Len())
	} else {
		tmp := fifo.ll.PushFront(&fifoEntry{key: key, value: value})
		fifo.cache[key] = tmp
		fifo.nBytes += int64(len(key)) + int64(value.Len())
	}

	for fifo.nBytes >= fifo.maxBytes {
		// log.Printf("nBytes: %d, maxBytes: %d\n", f.nbytes, f.maxBytes)
		fifo.RemoveLast()
	}
}

// Get 查询
func (fifo *FIFOCache) Get(key string) (Value, error) {
	if ele, ok := fifo.cache[key]; ok {
		kv := ele.Value.(*fifoEntry)
		return kv.value, nil
	}
	return nil, ErrKeyNotFound
}

// Remove 删除
func (fifo *FIFOCache) Remove(key string) error {
	if ele, ok := fifo.cache[key]; ok {
		kv := ele.Value.(*fifoEntry)
		fifo.ll.Remove(ele)
		delete(fifo.cache, kv.key)
		fifo.nBytes -= int64(len(kv.key)) + int64(kv.value.Len())
	}
	return ErrKeyNotFound
}

// Purge 清空
func (fifo *FIFOCache) Purge() {
	fifo.ll = nil
	fifo.cache = nil
	fifo.nBytes = 0
}

func (fifo *FIFOCache) MaxSize() int64 {
	return fifo.maxBytes
}

func (fifo *FIFOCache) Size() int64 {
	return fifo.nBytes
}

func (fifo *FIFOCache) RemoveLast() {
	if fifo.ll == nil || fifo.ll.Len() == 0 {
		return
	}

	ele := fifo.ll.Back()
	kv := ele.Value.(*fifoEntry)
	fifo.ll.Remove(ele)
	delete(fifo.cache, kv.key)
	fifo.nBytes -= int64(len(kv.key)) + int64(kv.value.Len())
}
