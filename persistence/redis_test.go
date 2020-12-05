package persistence

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// These tests require redis server running on localhost:6379 (the default)
const redisTestServer = "localhost:6379"

var newRedisStore = func(t *testing.T, defaultExpiration time.Duration) CacheStore {
	c, err := net.Dial("tcp", redisTestServer)
	if err == nil {
		c.Write([]byte("flush_all\r\n"))
		c.Close()
		opts := Options{
			Adapter: AdapterRedisStore,
			AdapterConfig: AdapterConfig{
				RedisConfig: &RedisConfig{
					Host:     redisTestServer,
					Password: "",
				},
			},
			DefaultExpiration: defaultExpiration,
		}
		redisStore, err := NewCacheStore(opts)
		assert.NoError(t, err)
		redisStore.Flush()
		return redisStore
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
