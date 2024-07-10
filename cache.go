package tinycache

import (
	"math"
	"sync"
	"tinycache/strategy"
)

// 缓存策略
const (
	TYPE_FIFO = "fifo"
	TYPE_LRU  = "lru"
	TYPE_LRUK = "lruk"
	TYPE_LFU  = "lfu"
	TYPE_ARC  = "arc"
	TYPE_2Q   = "2q"
)

// Cache 接口，支持多种缓存策略
type Cache interface {

	// Set or Update the key-value pair
	Set(key string, value strategy.Value)

	// Get returns the value for the specific key-value pair
	Get(key string) (strategy.Value, error)

	// Remove removes the specific key from the cache if the key is present
	Remove(key string) error

	// Purge is used to completely clear the cache
	Purge()

	// Size returns the space used by the cache
	Size() int64

	// MaxSize returns the maxSize of the cache
	MaxSize() int64
}

// mainCache 并发安全的cache
type mainCache struct {
	mu    sync.RWMutex
	cache Cache
	//cacheBytes int64
	cacheType string
	OnEvicted func(key string, value strategy.Value)
}

type Option struct {
	cacheType string
	maxBytes  int64
	k         int
	OnEvicted func(key string, value strategy.Value)
}

var DefaultOption = Option{
	cacheType: "lru",
	maxBytes:  int64(math.MaxInt32),
	k:         strategy.DefaultLRUK,
	OnEvicted: nil,
}

type ModOption func(option *Option)

func New(opts ...Option) *mainCache {
	return NewByOption(DefaultOption.cacheType, DefaultOption.maxBytes)
}

func NewByOption(cacheType string, maxBytes int64, opts ...Option) *mainCache {
	option := DefaultOption
	option.cacheType = cacheType

	var cache Cache
	switch cacheType {
	case TYPE_LRU:
		cache = strategy.NewLRUCache(maxBytes)
	case TYPE_FIFO:
		cache = strategy.NewFIFOCache(maxBytes)
	case TYPE_LFU:
		cache = strategy.NewLFUCache(maxBytes)
	case TYPE_LRUK:
		cache = strategy.NewLRUKCache(option.k, maxBytes)
	default:
		cache = strategy.NewLRUCache(maxBytes)
	}

	return &mainCache{
		cache:     cache,
		cacheType: cacheType,
	}
}

func (m *mainCache) add(key string, value ByteView) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cache == nil {
		m.cache = New(DefaultOption)
	}
	m.cache.Set(key, value)
}

func (m *mainCache) get(key string) (value ByteView, ok bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cache == nil {
		return
	}
	if v, err := m.cache.Get(key); err == nil {
		return v.(ByteView), true
	}
	return
}

// Set 支持并发的缓存写入
func (m *mainCache) Set(key string, value strategy.Value) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cache.Set(key, value)
}

// Get 支持并发的缓存读取
func (m *mainCache) Get(key string) (strategy.Value, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cache.Get(key)
}

// Remove 支持并发的缓存删除
func (m *mainCache) Remove(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.cache.Remove(key)
}

func (m *mainCache) Purge() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cache.Purge()
}

func (m *mainCache) Size() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cache.Size()
}

func (m *mainCache) MaxSize() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cache.MaxSize()
}
