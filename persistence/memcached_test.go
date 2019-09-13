package persistence

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// These tests require memcached running on localhost:11211 (the default)
const testServer = "localhost:11211"

var newMemcachedStore = func(t *testing.T, defaultExpiration time.Duration) CacheStore {
	c, err := net.Dial("tcp", testServer)
	if err == nil {
		c.Write([]byte("flush_all\r\n"))
		c.Close()
		opts := Options{
			Adapter: AdapterMemcachedStore,
			AdapterConfig: AdapterConfig{
				MemCachedConfig: &MemCachedConfig{
					HostList: []string{testServer},
				},
			},
			DefaultExpiration: defaultExpiration,
		}
		mcStore, err := NewCacheStore(opts)
		assert.NoError(t, err)
		return mcStore
	}
	t.Errorf("couldn't connect to memcached on %s", testServer)
	t.FailNow()
	panic("")
}

func TestMemcachedCache_TypicalGetSet(t *testing.T) {
	typicalGetSet(t, newMemcachedStore)
}

func TestMemcachedCache_IncrDecr(t *testing.T) {
	incrDecr(t, newMemcachedStore)
}

func TestMemcachedCache_Expiration(t *testing.T) {
	expiration(t, newMemcachedStore)
}

func TestMemcachedCache_EmptyCache(t *testing.T) {
	emptyCache(t, newMemcachedStore)
}

func TestMemcachedCache_Replace(t *testing.T) {
	testReplace(t, newMemcachedStore)
}

func TestMemcachedCache_Add(t *testing.T) {
	testAdd(t, newMemcachedStore)
}
