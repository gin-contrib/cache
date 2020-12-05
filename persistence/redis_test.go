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

func TestRedisStoreSelectDatabase(t *testing.T) {
	c, err := net.Dial("tcp", redisTestServer)
	if err != nil {
		t.Errorf("couldn't connect to redis on %s", redisTestServer)
	}
	_, err = c.Write([]byte("flush_all\r\n"))
	if err != nil {
		t.Fatalf("Unexpected error: %s", err.Error())
	}
	c.Close()
	redisCache := NewRedisCache(redisTestServer, "", 1*time.Second, WithSelectDatabase(1))
	err = redisCache.Flush()
	if err != nil {
		t.Errorf("couldn't connect to redis on %s", redisTestServer)
	}
}
func TestRedisCache_MgetTwoKeys(t *testing.T) {
	simpleMgetTwoKeys(t, newRedisStore)
}

func TestRedisCache_MsetNXTwoKeys(t *testing.T) {
	simpleMsetNXTwoKeys(t, newRedisStore)
}

func TestRedisCache_MsetNXThenMgetThreeKeys(t *testing.T) {
	msetNXThenMgetThreeKeys(t, newRedisStore)
}

func TestRedisCache_TypicalGetSet(t *testing.T) {
	typicalGetSet(t, newRedisStore)
}

func TestRedisCache_IncrDecr(t *testing.T) {
	incrDecr(t, newRedisStore)
}
func TestRedis_IncrAtomic(t *testing.T) {
	incrAtomic(t, newRawRedisStore)
}

func TestRedis_GetExpiresIn(t *testing.T) {
	getExpiresIn(t, newRawRedisStore)
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

// The following tests are specific to RedisStore.
func simpleMgetTwoKeys(t *testing.T, newCache cacheFactory) {
	cache := newCache(t, time.Hour).(*RedisStore)
	// set two keys and make sure set is successful
	value := "foo1"
	err := cache.Set("test1", value, DEFAULT)
	if err != nil {
		t.Errorf("Error setting a value: %s", err)
	}
	value = ""
	err = cache.Get("test1", &value)
	if err != nil {
		t.Errorf("Error getting a value: %s", err)
	}
	if value != "foo1" {
		t.Errorf("Expected to get foo back, got %s", value)
	}
	value = "foo2"
	if err = cache.Set("test2", value, DEFAULT); err != nil {
		t.Errorf("Error setting a value: %s", err)
	}
	value = ""
	err = cache.Get("test2", &value)
	if err != nil {
		t.Errorf("Error getting a value: %s", err)
	}
	if value != "foo2" {
		t.Errorf("Expected to get foo back, got %s", value)
	}

	// mget and verify
	s1 := ""
	s2 := ""
	result := []interface{}{&s1, &s2}
	err = cache.Mget(result, "test1", "test2")
	if err != nil {
		t.Errorf("Error while doing mget: %v", err)
	}
	if s1 != "foo1" {
		t.Errorf("Expected to get foo1 for key test1, got %v", s1)
	}
	if s2 != "foo2" {
		t.Errorf("Expected to get foo2 for key test2, got %v", s2)
	}
	// shows another way to get the value
	t.Logf("test1: %v, test2: %v, err: %v", *(result[0].(*string)), *(result[1].(*string)), err)
}

func simpleMsetNXTwoKeys(t *testing.T, newCache cacheFactory) {
	cache := newCache(t, time.Hour).(*RedisStore)
	k1 := "test1"
	v1 := "value2"
	k2 := "test2"
	v2 := "value2"

	if err := cache.Delete(k1); err != nil {
		t.Fatalf("Unexpected error: %s", err.Error())
	}
	if err := cache.Delete(k2); err != nil {
		t.Fatalf("Unexpected error: %s", err.Error())
	}
	// mset two keys
	err := cache.MSetNX(time.Hour, k1, v1, k2, v2)
	if err != nil {
		t.Errorf("Error mset: %v", err)
	}

	// verify values
	value := ""
	err = cache.Get(k1, &value)
	if err != nil {
		t.Errorf("Error getting a key %s: %v", k1, err)
	}
	if value != v1 {
		t.Errorf("Error getting value for key %s. Got %v, expect %v", k1, value, v1)
	}
	value = ""
	err = cache.Get(k2, &value)
	if err != nil {
		t.Errorf("Error getting a key %s: %v", k2, err)
	}
	if value != v2 {
		t.Errorf("Error getting value for key %s. Got %v, expect %v", k2, value, v2)
	}
}

func msetNXThenMgetThreeKeys(t *testing.T, newCache cacheFactory) {
	cache := newCache(t, time.Hour).(*RedisStore)
	k1 := "test1"
	v1 := "value2"
	k2 := "test2"
	v2 := "value2"
	k3 := "test3"
	v3 := "value3"

	if err := cache.Delete(k1); err != nil {
		t.Fatalf("Unexpected error: %s", err.Error())
	}
	if err := cache.Delete(k2); err != nil {
		t.Fatalf("Unexpected error: %s", err.Error())
	}
	if err := cache.Delete(k3); err != nil {
		t.Fatalf("Unexpected error: %s", err.Error())
	}
	// mset two keys
	err := cache.MSetNX(time.Hour, k1, v1, k2, v2, k3, v3)
	if err != nil {
		t.Errorf("Error mset: %v", err)
	}

	r1 := ""
	r2 := ""
	r3 := ""
	result := []interface{}{&r1, &r2, &r3}
	err = cache.Mget(result, k1, k2, k3)
	if err != nil {
		t.Errorf("Error while doing mget: %v", err)
	}
	if r1 != v1 {
		t.Errorf("Expected to get %v for key %v, got %v", v1, k1, r1)
	}
	if r2 != v2 {
		t.Errorf("Expected to get %v for key %v, got %v", v2, k2, r2)
	}
	if r3 != v3 {
		t.Errorf("Expected to get %v for key %v, got %v", v3, k3, r3)
	}
}
