package persistence

import (
	"testing"
	"time"

	"github.com/memcachier/mc/v3"
)

// These tests require memcached running on localhost:11211 (the default)
const localhost = "localhost:11211"

var newMcStore = func(t *testing.T, defaultExpiration time.Duration) CacheStore {
	mcStore := NewMemcachedBinaryStore(localhost, "", "", defaultExpiration)
	err := mcStore.Flush()
	if err == nil {
		return mcStore
	}
	t.Errorf("Failed to connect to memcached on %s with %s", localhost, err)
	t.FailNow()
	panic("")
}

func TestMemcachedBinary_TypicalGetSet(t *testing.T) {
	typicalGetSet(t, newMcStore)
}

func TestMemcachedBinary_IncrDecr(t *testing.T) {
	incrDecr(t, newMcStore)
}

func TestMemcachedBinary_Expiration(t *testing.T) {
	expiration(t, newMcStore)
}

func TestMemcachedBinary_EmptyCache(t *testing.T) {
	emptyCache(t, newMcStore)
}

func TestMemcachedBinary_Replace(t *testing.T) {
	testReplace(t, newMcStore)
}

func TestMemcachedBinary_Add(t *testing.T) {
	testAdd(t, newMcStore)
}

var newMcStoreWithConfig = func(t *testing.T, defaultExpiration time.Duration) CacheStore {
	config := mc.DefaultConfig()
	config.PoolSize = 2
	mcStore := NewMemcachedBinaryStoreWithConfig(localhost, "", "", defaultExpiration, config)
	err := mcStore.Flush()
	if err == nil {
		return mcStore
	}
	t.Errorf("Failed to connect to memcached on %s with %s", localhost, err)
	t.FailNow()
	panic("")
}

func TestMemcachedBinaryWithConfig_TypicalGetSet(t *testing.T) {
	typicalGetSet(t, newMcStoreWithConfig)
}

func TestMemcachedBinaryWithConfig_IncrDecr(t *testing.T) {
	incrDecr(t, newMcStoreWithConfig)
}

func TestMemcachedBinaryWithConfig_Expiration(t *testing.T) {
	expiration(t, newMcStoreWithConfig)
}

func TestMemcachedBinaryWithConfig_EmptyCache(t *testing.T) {
	emptyCache(t, newMcStoreWithConfig)
}

func TestMemcachedBinaryWithConfig_Replace(t *testing.T) {
	testReplace(t, newMcStoreWithConfig)
}

func TestMemcachedBinaryWithConfig_Add(t *testing.T) {
	testAdd(t, newMcStoreWithConfig)
}
