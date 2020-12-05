package persistence

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var newInMemoryStore = func(t *testing.T, defaultExpiration time.Duration) CacheStore {
	opts := Options{
		Adapter:           AdapterInMemoryStore,
		DefaultExpiration: defaultExpiration,
	}
	inMemoryStore, err := NewCacheStore(opts)
	assert.NoError(t, err)
	return inMemoryStore
}

// Test typical cache interactions
func TestInMemoryCache_TypicalGetSet(t *testing.T) {
	typicalGetSet(t, newInMemoryStore)
}

func TestInMemoryCache_IncrDecr(t *testing.T) {
	incrDecr(t, newInMemoryStore)
}

func TestInMemoryCache_Expiration(t *testing.T) {
	expiration(t, newInMemoryStore)
}

func TestInMemoryCache_EmptyCache(t *testing.T) {
	emptyCache(t, newInMemoryStore)
}

func TestInMemoryCache_Replace(t *testing.T) {
	testReplace(t, newInMemoryStore)
}

func TestInMemoryCache_Add(t *testing.T) {
	testAdd(t, newInMemoryStore)
}
