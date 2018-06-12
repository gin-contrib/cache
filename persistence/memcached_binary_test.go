package persistence

import (
	"testing"
	"time"
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
