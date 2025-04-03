package persistence

import (
	"net"
	"testing"
	"time"
)

// These tests require redis server running on localhost:6379 (the default)
const redisTestServer = "localhost:6379"

var newRedisStore = func(t *testing.T, defaultExpiration time.Duration) CacheStore {
	c, err := net.Dial("tcp", redisTestServer)
	if err == nil {
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
	t.Errorf("couldn't connect to redis on %s", redisTestServer)
	t.FailNow()
	panic("")
}

func TestRedisCache_TypicalGetSet(t *testing.T) {
	typicalGetSet(t, newRedisStore)
}

func TestRedisCache_IncrDecr(t *testing.T) {
	incrDecr(t, newRedisStore)
}

func TestRedisCache_Expiration(t *testing.T) {
	expiration(t, newRedisStore)
}

func TestRedisCache_EmptyCache(t *testing.T) {
	emptyCache(t, newRedisStore)
}

func TestRedisCache_Replace(t *testing.T) {
	testReplace(t, newRedisStore)
}

func TestRedisCache_Add(t *testing.T) {
	testAdd(t, newRedisStore)
}
