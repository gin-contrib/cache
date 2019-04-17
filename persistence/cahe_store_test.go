package persistence

import (
	"math"
	"testing"
	"time"
)

type cacheFactory func(*testing.T, time.Duration) CacheStore

// Test typical cache interactions
func typicalGetSet(t *testing.T, newCache cacheFactory) {
	var err error
	cache := newCache(t, time.Hour)

	value := "foo"
	if err = cache.Set("value", value, DEFAULT); err != nil {
		t.Errorf("Error setting a value: %s", err)
	}

	value = ""
	err = cache.Get("value", &value)
	if err != nil {
		t.Errorf("Error getting a value: %s", err)
	}
	if value != "foo" {
		t.Errorf("Expected to get foo back, got %s", value)
	}
}

func simpleMgetTwoKeys(t *testing.T, newCache cacheFactory) {
	cache := newCache(t, time.Hour)
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
	cache := newCache(t, time.Hour)
	k1 := "test1"
	v1 := "value2"
	k2 := "test2"
	v2 := "value2"

	cache.Delete(k1)
	cache.Delete(k2)
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
	cache := newCache(t, time.Hour)
	k1 := "test1"
	v1 := "value2"
	k2 := "test2"
	v2 := "value2"
	k3 := "test3"
	v3 := "value3"

	cache.Delete(k1)
	cache.Delete(k2)
	cache.Delete(k3)
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

// Test the increment-decrement cases
func incrDecr(t *testing.T, newCache cacheFactory) {
	var err error
	cache := newCache(t, time.Hour)

	// Normal increment / decrement operation.
	if err = cache.Set("int", 10, DEFAULT); err != nil {
		t.Errorf("Error setting int: %s", err)
	}
	newValue, err := cache.Increment("int", 50)
	if err != nil {
		t.Errorf("Error incrementing int: %s", err)
	}
	if newValue != 60 {
		t.Errorf("Expected 60, was %d", newValue)
	}

	if newValue, err = cache.Decrement("int", 50); err != nil {
		t.Errorf("Error decrementing: %s", err)
	}
	if newValue != 10 {
		t.Errorf("Expected 10, was %d", newValue)
	}

	// Increment wraparound
	newValue, err = cache.Increment("int", math.MaxUint64-5)
	if err != nil {
		t.Errorf("Error wrapping around: %s", err)
	}
	if newValue != 4 {
		t.Errorf("Expected wraparound 4, got %d", newValue)
	}

	// Decrement capped at 0
	newValue, err = cache.Decrement("int", 25)
	if err != nil {
		t.Errorf("Error decrementing below 0: %s", err)
	}
	if newValue != 0 {
		t.Errorf("Expected capped at 0, got %d", newValue)
	}
}

func expiration(t *testing.T, newCache cacheFactory) {
	// memcached does not support expiration times less than 1 second.
	var err error
	cache := newCache(t, time.Second)
	// Test Set w/ DEFAULT
	value := 10
	cache.Set("int", value, DEFAULT)
	time.Sleep(2 * time.Second)
	err = cache.Get("int", &value)
	if err != ErrCacheMiss {
		t.Errorf("Expected CacheMiss, but got: %s", err)
	}

	// Test Set w/ short time
	cache.Set("int", value, time.Second)
	time.Sleep(2 * time.Second)
	err = cache.Get("int", &value)
	if err != ErrCacheMiss {
		t.Errorf("Expected CacheMiss, but got: %s", err)
	}

	// Test Set w/ longer time.
	cache.Set("int", value, time.Hour)
	time.Sleep(2 * time.Second)
	err = cache.Get("int", &value)
	if err != nil {
		t.Errorf("Expected to get the value, but got: %s", err)
	}

	// Test Set w/ forever.
	cache.Set("int", value, FOREVER)
	time.Sleep(2 * time.Second)
	err = cache.Get("int", &value)
	if err != nil {
		t.Errorf("Expected to get the value, but got: %s", err)
	}
}

func emptyCache(t *testing.T, newCache cacheFactory) {
	var err error
	cache := newCache(t, time.Hour)

	err = cache.Get("notexist", 0)
	if err == nil {
		t.Errorf("Error expected for non-existent key")
	}
	if err != ErrCacheMiss {
		t.Errorf("Expected ErrCacheMiss for non-existent key: %s", err)
	}

	err = cache.Delete("notexist")
	if err != ErrCacheMiss {
		t.Errorf("Expected ErrCacheMiss for non-existent key: %s", err)
	}

	_, err = cache.Increment("notexist", 1)
	if err != ErrCacheMiss {
		t.Errorf("Expected cache miss incrementing non-existent key: %s", err)
	}

	_, err = cache.Decrement("notexist", 1)
	if err != ErrCacheMiss {
		t.Errorf("Expected cache miss decrementing non-existent key: %s", err)
	}
}

func testReplace(t *testing.T, newCache cacheFactory) {
	var err error
	cache := newCache(t, time.Hour)

	// Replace in an empty cache.
	if err = cache.Replace("notexist", 1, FOREVER); err != ErrNotStored && err != ErrCacheMiss {
		t.Errorf("Replace in empty cache: expected ErrNotStored or ErrCacheMiss, got: %s", err)
	}

	// Set a value of 1, and replace it with 2
	if err = cache.Set("int", 1, time.Second); err != nil {
		t.Errorf("Unexpected error: %s", err)
	}

	if err = cache.Replace("int", 2, time.Second); err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	var i int
	if err = cache.Get("int", &i); err != nil {
		t.Errorf("Unexpected error getting a replaced item: %s", err)
	}
	if i != 2 {
		t.Errorf("Expected 2, got %d", i)
	}

	// Wait for it to expire and replace with 3 (unsuccessfully).
	time.Sleep(2 * time.Second)
	if err = cache.Replace("int", 3, time.Second); err != ErrNotStored && err != ErrCacheMiss {
		t.Errorf("Expected ErrNotStored or ErrCacheMiss, got: %s", err)
	}
	if err = cache.Get("int", &i); err != ErrCacheMiss {
		t.Errorf("Expected cache miss, got: %s", err)
	}
}

func testAdd(t *testing.T, newCache cacheFactory) {
	var err error
	cache := newCache(t, time.Hour)
	// Add to an empty cache.
	if err = cache.Add("int", 1, time.Second); err != nil {
		t.Errorf("Unexpected error adding to empty cache: %s", err)
	}

	// Try to add again. (fail)
	if err = cache.Add("int", 2, time.Second); err != ErrNotStored {
		t.Errorf("Expected ErrNotStored adding dupe to cache: %s", err)
	}

	// Wait for it to expire, and add again.
	time.Sleep(2 * time.Second)
	if err = cache.Add("int", 3, time.Second); err != nil {
		t.Errorf("Unexpected error adding to cache: %s", err)
	}

	// Get and verify the value.
	var i int
	if err = cache.Get("int", &i); err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if i != 3 {
		t.Errorf("Expected 3, got: %d", i)
	}
}
