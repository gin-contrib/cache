package persistence

import (
	"time"

	"github.com/gin-contrib/cache/utils"
	"github.com/memcachier/mc"
)

// MemcachedBinaryStore represents the cache with memcached persistence using
// the binary protocol
type MemcachedBinaryStore struct {
	*mc.Client
	defaultExpiration time.Duration
}

// NewMemcachedBinaryStore returns a MemcachedBinaryStore
func NewMemcachedBinaryStore(hostList, username, password string, defaultExpiration time.Duration) *MemcachedBinaryStore {
	return &MemcachedBinaryStore{mc.NewMC(hostList, username, password), defaultExpiration}
}

// NewMemcachedBinaryStoreWithConfig returns a MemcachedBinaryStore using the provided configuration
func NewMemcachedBinaryStoreWithConfig(hostList, username, password string, defaultExpiration time.Duration, config *mc.Config) *MemcachedBinaryStore {
	return &MemcachedBinaryStore{mc.NewMCwithConfig(hostList, username, password, config), defaultExpiration}
}

// Set (see CacheStore interface)
func (s *MemcachedBinaryStore) Set(key string, value interface{}, expires time.Duration) error {
	exp := s.getExpiration(expires)
	b, err := utils.Serialize(value)
	if err != nil {
		return err
	}
	_, err = s.Client.Set(key, string(b), 0, exp, 0)
	return convertMcError(err)
}

// Add (see CacheStore interface)
func (s *MemcachedBinaryStore) Add(key string, value interface{}, expires time.Duration) error {
	exp := s.getExpiration(expires)
	b, err := utils.Serialize(value)
	if err != nil {
		return err
	}
	_, err = s.Client.Add(key, string(b), 0, exp)
	return convertMcError(err)
}

// Replace (see CacheStore interface)
func (s *MemcachedBinaryStore) Replace(key string, value interface{}, expires time.Duration) error {
	exp := s.getExpiration(expires)
	b, err := utils.Serialize(value)
	if err != nil {
		return err
	}
	_, err = s.Client.Replace(key, string(b), 0, exp, 0)
	return convertMcError(err)
}

// Get (see CacheStore interface)
func (s *MemcachedBinaryStore) Get(key string, value interface{}) error {
	val, _, _, err := s.Client.Get(key)
	if err != nil {
		return convertMcError(err)
	}
	return utils.Deserialize([]byte(val), value)
}

// Delete (see CacheStore interface)
func (s *MemcachedBinaryStore) Delete(key string) error {
	return convertMcError(s.Client.Del(key))
}

// Increment (see CacheStore interface)
func (s *MemcachedBinaryStore) Increment(key string, delta uint64) (uint64, error) {
	n, _, err := s.Client.Incr(key, delta, 0, 0xffffffff, 0)
	return n, convertMcError(err)
}

// Decrement (see CacheStore interface)
func (s *MemcachedBinaryStore) Decrement(key string, delta uint64) (uint64, error) {
	n, _, err := s.Client.Decr(key, delta, 0, 0xffffffff, 0)
	return n, convertMcError(err)
}

// Flush (see CacheStore interface)
func (s *MemcachedBinaryStore) Flush() error {
	return convertMcError(s.Client.Flush(0))
}

// getExpiration converts a gin-contrib/cache expiration in the form of a
// time.Duration to a valid memcached expiration either in seconds (<30 days)
// or a Unix timestamp (>30 days)
func (s *MemcachedBinaryStore) getExpiration(expires time.Duration) uint32 {
	switch expires {
	case DEFAULT:
		expires = s.defaultExpiration
	case FOREVER:
		expires = time.Duration(0)
	}
	exp := uint32(expires.Seconds())
	if exp > 60*60*24*30 { // > 30 days
		exp += uint32(time.Now().Unix())
	}
	return exp
}

func convertMcError(err error) error {
	switch err {
	case nil:
		return nil
	case mc.ErrNotFound:
		return ErrCacheMiss
	case mc.ErrValueNotStored:
		return ErrNotStored
	case mc.ErrKeyExists:
		return ErrNotStored
	}
	return err
}
