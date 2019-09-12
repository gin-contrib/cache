package persistence

import (
	"errors"
	"fmt"
	"time"
)

const (
	DEFAULT = time.Duration(0)
	FOREVER = time.Duration(-1)

	// CacheStore adapter names
	AdapterRedisStore           = "redis"
	AdapterInMemoryStore        = "memory"
	AdapterMemcachedStore       = "memcache"
	AdapterMemcachedBinaryStore = "memcachebinary"
)

var (
	PageCachePrefix = "gincontrib.page.cache"
	ErrCacheMiss    = errors.New("cache: key not found")
	ErrNotStored    = errors.New("cache: not stored")
	ErrNotSupport   = errors.New("cache: not support")
)

// CacheStore is the interface of a cache backend
type CacheStore interface {
	// New returns a new CacheStore associated with the configuration
	New(opts Options) CacheStore

	// Get retrieves an item from the cache. Returns the item or nil, and a bool indicating
	// whether the key was found.
	Get(key string, value interface{}) error

	// Set sets an item to the cache, replacing any existing item.
	Set(key string, value interface{}, expire time.Duration) error

	// Add adds an item to the cache only if an item doesn't already exist for the given
	// key, or if the existing item has expired. Returns an error otherwise.
	Add(key string, value interface{}, expire time.Duration) error

	// Replace sets a new value for the cache key only if it already exists. Returns an
	// error if it does not.
	Replace(key string, data interface{}, expire time.Duration) error

	// Delete removes an item from the cache. Does nothing if the key is not in the cache.
	Delete(key string) error

	// Increment increments a real number, and returns error if the value is not real
	Increment(key string, data uint64) (uint64, error)

	// Decrement decrements a real number, and returns error if the value is not real
	Decrement(key string, data uint64) (uint64, error)

	// Flush seletes all items from the cache.
	Flush() error
}

// Options contains configuration for desired CacheStore
type Options struct {
	Adapter string
	AdapterConfig
	DefaultExpiration time.Duration
}

// AdapterConfig contains CacheStore specific configuration
type AdapterConfig struct {
	MemCachedConfig       *MemCachedConfig
	MemcachedBinaryConfig *MemcachedBinaryConfig
	RedisConfig           *RedisConfig
}

// NewCacheStore creates and returns a new CacheStore
// associated with the adapter name and configuration
// It returns error if the adapter isn't registered
func NewCacheStore(opts Options) (CacheStore, error) {
	if len(opts.Adapter) == 0 {
		opts.Adapter = AdapterInMemoryStore
	}
	adapter, ok := adapters[opts.Adapter]
	if !ok {
		return nil, fmt.Errorf("cache: unknown adapter '%s'", opts.Adapter)
	}
	if opts.DefaultExpiration == time.Duration(0) {
		opts.DefaultExpiration = time.Hour
	}
	return adapter.New(opts), nil
}

var adapters = make(map[string]CacheStore)

// Register registers a cache store
func Register(name string, adapter CacheStore) {
	if adapter == nil {
		panic("cache: can't register nil cache adapter")
	}
	if _, exist := adapters[name]; exist {
		panic("cache: adapter '%s' already registered")
	}
	adapters[name] = adapter
}
