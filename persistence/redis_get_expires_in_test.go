
package persistence

import (
	"testing"
	"time"
	// "github.com/Jim-Lambert-Bose/cache/persistence"

)


// Test the increment-decrement cases
func getExpiresIn(t *testing.T, newStore redisStoreFactory) {
	var err error
	store := newStore(t, time.Hour)

	key := "expires-in-int"
	if err = store.Set(key, 10, 1*time.Second); err != nil {
		t.Errorf("Error setting int: %s", err)
	}
	exIn, err := store.GetExpiresIn(key)
	
	if err != nil {
		t.Errorf("unexpected error: %s", err.Error())
		if err == ErrCacheMiss {
			t.Errorf("expected to find entry: %s", key )
		}
		if err == ErrCacheNoTTL {
			t.Errorf("expected to find ttl on entry: %s", key)
		}
	}
	if exIn < 500 || exIn > 1000 {
		t.Errorf("unexpected value for ttl ms: %d", exIn)
	}
	t.Log(err, exIn)
	
	
}

