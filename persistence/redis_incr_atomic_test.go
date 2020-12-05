package persistence

import (
	"net"
	"testing"
	"time"
)

type redisStoreFactory func(*testing.T, time.Duration) *RedisStore

var newRawRedisStore = func(t *testing.T, defaultExpiration time.Duration) *RedisStore {
	c, err := net.Dial("tcp", redisTestServer)
	if err == nil {
		_, err := c.Write([]byte("flush_all\r\n"))
		if err != nil {
			t.Fatalf("Unexpected error: %s", err.Error())
		}
		c.Close()
		redisCache := NewRedisCache(redisTestServer, "", defaultExpiration)
		redisCache.Flush()
		return redisCache
	}
	t.Errorf("couldn't connect to redis on %s", redisTestServer)
	t.FailNow()
	panic("")
}

// Test the increment-decrement cases
func incrAtomic(t *testing.T, newStore redisStoreFactory) {
	var err error
	store := newStore(t, time.Hour)

	// Normal increment / decrement operation.
	if err = store.Set("int", 10, DEFAULT); err != nil {
		t.Errorf("Error setting int: %s", err)
	}

	newValue, err := store.IncrementCheckSet("int", 50)
	if err != nil {
		t.Errorf("Error incrementing int: %s", err)
	}
	if newValue != 60 {
		t.Errorf("Expected 60, was %d", newValue)
	}
	newValue, err = store.IncrementCheckSet("int", 50)
	if err != nil {
		t.Errorf("Error incrementing int: %s", err)
	}
	if newValue != 110 {
		t.Errorf("Expected 110, was %d", newValue)
	}
	_, err = store.IncrementCheckSet("badkey", 50)
	if err != ErrCacheMiss {
		t.Errorf("Error incrementing badkey.. should have been ErrCacheMiss")
	}
	newValue, err = store.IncrementAtomic("newInt", 2)
	if err != nil {
		t.Errorf("Error incrementing int: %s", err)
	}
	if newValue != 2 {
		t.Errorf("Expected 2, was %d", newValue)
	}

	err = store.ExpireAt("int", uint64(time.Now().Unix()+10))
	if err != nil {
		t.Errorf("Error setting expire at: %s", err.Error())
	}
	var value int
	err = store.Get("int", &value)
	if newValue != 2 {
		t.Errorf("Expected 2, was %d", newValue)
	}
	err = store.ExpireAt("int", uint64(time.Now().Unix()+1))
	time.Sleep(2 * time.Second)
	err = store.Get("int", &value)
	if newValue != 2 {
		t.Errorf("Expected 2, was %d", newValue)
	}
	if err != ErrCacheMiss {
		t.Errorf("Expected to NOT get the value, but got: %v", value)
	}

}
