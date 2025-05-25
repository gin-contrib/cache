package persistence

import (
	"net"
	"testing"
	"time"
)

// These tests require redis server running on localhost:6379 (the default)
const (
	redisTestServer = "localhost:6379"
	redisTestURL    = "redis://localhost:6379"
)

func newRedisStore(t *testing.T, defaultExpiration time.Duration) CacheStore {
	c, err := net.Dial("tcp", redisTestServer)
	if err != nil {
		t.Errorf("couldn't connect to redis on %s", redisTestServer)
		t.FailNow()
	}
	_, _ = c.Write([]byte("flush_all\r\n"))
	if err := c.Close(); err != nil {
		t.Errorf("Error closing connection: %v", err)
	}
	redisCache := NewRedisCache(redisTestServer, "", defaultExpiration)
	if err := redisCache.Flush(); err != nil {
		t.Errorf("Error flushing cache: %v", err)
	}
	return redisCache
}

func newRedisStoreWithURL(t *testing.T, defaultExpiration time.Duration) CacheStore {
	c, err := net.Dial("tcp", redisTestServer)
	if err != nil {
		t.Skipf("couldn't connect to redis on %s, skipping URL-based tests", redisTestServer)
	}
	_, _ = c.Write([]byte("flush_all\r\n"))
	if err := c.Close(); err != nil {
		t.Errorf("Error closing connection: %v", err)
	}
	redisCache := NewRedisCacheWithURL(redisTestURL, defaultExpiration)
	if err := redisCache.Flush(); err != nil {
		t.Errorf("Error flushing cache: %v", err)
	}
	return redisCache
}

func runCommonTests(t *testing.T, factory cacheFactory) {
	t.Run("TypicalGetSet", func(t *testing.T) { typicalGetSet(t, factory) })
	t.Run("IncrDecr", func(t *testing.T) { incrDecr(t, factory) })
	t.Run("Expiration", func(t *testing.T) { expiration(t, factory) })
	t.Run("EmptyCache", func(t *testing.T) { emptyCache(t, factory) })
	t.Run("Replace", func(t *testing.T) { testReplace(t, factory) })
	t.Run("Add", func(t *testing.T) { testAdd(t, factory) })
}

func TestRedisCache(t *testing.T) {
	t.Run("Standard", func(t *testing.T) { runCommonTests(t, newRedisStore) })
	t.Run("WithURL", func(t *testing.T) { runCommonTests(t, newRedisStoreWithURL) })
}
