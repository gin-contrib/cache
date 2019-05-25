package persistence

import (
	"errors"
	"fmt"

	//"github.com/gin-contrib/cache/utils"
	"time"

	"github.com/Jim-Lambert-Bose/cache/utils"
	"github.com/gomodule/redigo/redis"
)

var (
	ErrCacheNoTTL = errors.New("cache: key has no TTL.")
)

// RedisStore represents the cache with redis persistence
type RedisStore struct {
	pool              *redis.Pool
	defaultExpiration time.Duration
}

// NewRedisCache returns a RedisStore
// until redigo supports sharding/clustering, only one host will be in hostList
func NewRedisCache(host string, password string, defaultExpiration time.Duration) *RedisStore {
	var pool = &redis.Pool{
		MaxIdle:     5,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			// the redis protocol should probably be made sett-able
			c, err := redis.Dial("tcp", host)
			if err != nil {
				return nil, err
			}
			if len(password) > 0 {
				if _, err := c.Do("AUTH", password); err != nil {
					c.Close()
					return nil, err
				}
			} else {
				// check with PING
				if _, err := c.Do("PING"); err != nil {
					c.Close()
					return nil, err
				}
			}
			return c, err
		},
		// custom connection test method
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if _, err := c.Do("PING"); err != nil {
				return err
			}
			return nil
		},
	}
	return &RedisStore{pool, defaultExpiration}
}

// NewRedisCacheWithPool returns a RedisStore using the provided pool
// until redigo supports sharding/clustering, only one host will be in hostList
func NewRedisCacheWithPool(pool *redis.Pool, defaultExpiration time.Duration) *RedisStore {
	return &RedisStore{pool, defaultExpiration}
}

// Set (see CacheStore interface)
func (c *RedisStore) Set(key string, value interface{}, expires time.Duration) error {
	conn := c.pool.Get()
	defer conn.Close()
	return c.invoke(conn.Do, key, value, expires)
}

// MSET add multiple items to redis cache if none of them already exists for the given keys. Return error otherwise.
// kv is a list of key value pairs: k1, v1, k2, v2, ...
func (c *RedisStore) MSetNX(expires time.Duration, kv ...interface{}) error {
	l := len(kv)
	if l%2 != 0 {
		return fmt.Errorf("Got %v keys but %v values", l/2, l/2+1)
	}
	keys := []string{}
	values := []interface{}{}
	for i := 0; i < l; i += 2 {
		if k, ok := kv[i].(string); !ok {
			return fmt.Errorf("key %v: %v is not string", i, kv[i])
		} else {
			keys = append(keys, k)
			values = append(values, kv[i+1])
		}
	}

	ex := c.translateExpire(expires)

	conn := c.pool.Get()
	defer conn.Close()

	if err := conn.Send("MULTI"); err != nil {
		return err
	}
	for i := 0; i < len(keys); i++ {
		b, err := utils.Serialize(values[i])
		if err != nil {
			return fmt.Errorf("Failed to serialize value %v: %v", i, values[i])
		}
		if err := conn.Send("SETNX", keys[i], b); err != nil {
			return err
		}
		if ex > 0 {
			if err := conn.Send("EXPIRE", keys[i], ex); err != nil {
				return err
			}
		}
	}
	_, err := conn.Do("EXEC")
	if err != nil {
		return err
	}
	return nil
}

// Add (see CacheStore interface)
func (c *RedisStore) Add(key string, value interface{}, expires time.Duration) error {
	conn := c.pool.Get()
	defer conn.Close()
	if exists(conn, key) {
		return ErrNotStored
	}
	return c.invoke(conn.Do, key, value, expires)
}

// Replace (see CacheStore interface)
func (c *RedisStore) Replace(key string, value interface{}, expires time.Duration) error {
	conn := c.pool.Get()
	defer conn.Close()
	if !exists(conn, key) {
		return ErrNotStored
	}
	err := c.invoke(conn.Do, key, value, expires)
	if value == nil {
		return ErrNotStored
	}

	return err

}

// Get (see CacheStore interface)
func (c *RedisStore) Get(key string, ptrValue interface{}) error {
	conn := c.pool.Get()
	defer conn.Close()
	raw, err := conn.Do("GET", key)
	if raw == nil {
		return ErrCacheMiss
	}
	item, err := redis.Bytes(raw, err)
	if err != nil {
		return err
	}
	return utils.Deserialize(item, ptrValue)
}

// MGet retrieves a list of items for the list of keys provided. If an item does not exist, an ErrCacheMiss is returned.
func (c *RedisStore) Mget(ptrValue []interface{}, keys ...string) error {
	if len(ptrValue) != len(keys) {
		return fmt.Errorf("Length of value array is different from number of keys. Got %v, requires %v", len(ptrValue), len(keys))
	}
	conn := c.pool.Get()
	defer conn.Close()
	var ks []interface{}
	for _, k := range keys {
		ks = append(ks, k)
	}

	raw, err := redis.Values(conn.Do("MGET", ks...))
	if err != nil {
		return err
	}
	if raw == nil {
		return ErrCacheMiss
	}
	for idx, r := range raw {
		item, err := redis.Bytes(r, err)
		if err != nil {
			return err
		}
		err = utils.Deserialize(item, ptrValue[idx])
		if err != nil {
			return err
		}
	}
	return nil
}

func exists(conn redis.Conn, key string) bool {
	retval, _ := redis.Bool(conn.Do("EXISTS", key))
	return retval
}

// Delete (see CacheStore interface)
func (c *RedisStore) Delete(key string) error {
	conn := c.pool.Get()
	defer conn.Close()
	if !exists(conn, key) {
		return ErrCacheMiss
	}
	_, err := conn.Do("DEL", key)
	return err
}

// Increment (see CacheStore interface)
func (c *RedisStore) Increment(key string, delta uint64) (uint64, error) {
	conn := c.pool.Get()
	defer conn.Close()
	// Check for existance *before* increment as per the cache contract.
	// redis will auto create the key, and we don't want that. Since we need to do increment
	// ourselves instead of natively via INCRBY (redis doesn't support wrapping), we get the value
	// and do the exists check this way to minimize calls to Redis
	val, err := conn.Do("GET", key)
	if val == nil {
		return 0, ErrCacheMiss
	}
	if err == nil {
		currentVal, err := redis.Int64(val, nil)
		if err != nil {
			return 0, err
		}
		sum := currentVal + int64(delta)
		_, err = conn.Do("SET", key, sum)
		if err != nil {
			return 0, err
		}
		return uint64(sum), nil
	}

	return 0, err
}

// IncrementCheckSet - special case where you want to increment a value ONLY if it doesn't change between your GET and SET
func (c *RedisStore) IncrementCheckSet(key string, delta uint64) (uint64, error) {
	conn := c.pool.Get()
	defer conn.Close()
	if _, err := conn.Do("WATCH", key); err != nil {
		return 0, err
	}
	defer func() {
		_, _ = conn.Do("UNWATCH", key)
	}()
	val, err := conn.Do("GET", key)
	if val == nil {
		return 0, ErrCacheMiss
	}
	if err == nil {
		currentVal, err := redis.Int64(val, nil)
		if err != nil {
			return 0, err
		}
		sum := currentVal + int64(delta)
		_, err = conn.Do("SET", key, sum)
		if err != nil {
			return 0, err
		}
		return uint64(sum), nil
	}
	return 0, err
}

// IncrementAtomic - special case for Redis storage to handle the need for atomic increments without a data race problem when
// a consumer wants to use this storage for something outside the standard cache contract.
func (c *RedisStore) IncrementAtomic(key string, delta uint64) (uint64, error) {
	conn := c.pool.Get()
	defer conn.Close()

	newValue, err := conn.Do("INCRBY", key, delta)
	if err != nil {
		return 0, err
	}
	return uint64(newValue.(int64)), nil
}

// ExpireAt - special case for Redis storage to handle updating the TTL for the entry for when
// a consumer wants to use this storage for something outside the standard cache contract.
func (c *RedisStore) ExpireAt(key string, epoc uint64) error {
	conn := c.pool.Get()
	defer conn.Close()
	ret, err := conn.Do("EXPIREAT", key, epoc)
	if ret == 0 {
		return ErrCacheMiss
	}
	if err != nil {
		return err
	}
	return nil
}

// GetExpiresIn returns the number of milliseconds until the key expires
// returns ErrCacheNoTTL if no expiration is set on the entry
func (c *RedisStore) GetExpiresIn(key string) (int64, error) {
	conn := c.pool.Get()
	defer conn.Close()
	ret, err := conn.Do("PTTL", key)
	if err != nil {
		return 0, err
	}
	ttl := ret.(int64)
	if ttl == -2 {
		return 0, ErrCacheMiss
	}
	if ttl == -1 {
		return 0, ErrCacheNoTTL
	}
	return ret.(int64), nil
}

// Decrement (see CacheStore interface)
func (c *RedisStore) Decrement(key string, delta uint64) (newValue uint64, err error) {
	conn := c.pool.Get()
	defer conn.Close()
	// Check for existance *before* increment as per the cache contract.
	// redis will auto create the key, and we don't want that, hence the exists call
	if !exists(conn, key) {
		return 0, ErrCacheMiss
	}
	// Decrement contract says you can only go to 0
	// so we go fetch the value and if the delta is greater than the amount,
	// 0 out the value
	currentVal, err := redis.Int64(conn.Do("GET", key))
	if err == nil && delta > uint64(currentVal) {
		tempint, err := redis.Int64(conn.Do("DECRBY", key, currentVal))
		return uint64(tempint), err
	}
	tempint, err := redis.Int64(conn.Do("DECRBY", key, delta))
	return uint64(tempint), err
}

// Flush (see CacheStore interface)
func (c *RedisStore) Flush() error {
	conn := c.pool.Get()
	defer conn.Close()
	_, err := conn.Do("FLUSHALL")
	return err
}

func (c *RedisStore) invoke(f func(string, ...interface{}) (interface{}, error),
	key string, value interface{}, expires time.Duration) error {

	switch expires {
	case DEFAULT:
		expires = c.defaultExpiration
	case FOREVER:
		expires = time.Duration(0)
	}

	b, err := utils.Serialize(value)
	if err != nil {
		return err
	}

	if expires > 0 {
		_, err := f("SETEX", key, int32(expires/time.Second), b)
		return err
	}

	_, err = f("SET", key, b)
	return err

}

// translate time duration to int32
func (c *RedisStore) translateExpire(expires time.Duration) int32 {
	result := expires
	switch expires {
	case DEFAULT:
		result = c.defaultExpiration
	case FOREVER:
		result = time.Duration(0)
	}
	return int32(result / time.Second)
}
