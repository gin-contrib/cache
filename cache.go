/*
Package cache provides middleware and utilities for HTTP response caching in Gin web applications.
It supports pluggable cache stores, cache key generation, and decorators for page and site-level caching.
*/
package cache

import (
	"encoding/gob"
	"encoding/hex"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/cespare/xxhash/v2"

	"github.com/gin-contrib/cache/persistence"
	"github.com/gin-gonic/gin"
)

const (
	CACHE_MIDDLEWARE_KEY = "gincontrib.cache"
)

var PageCachePrefix = "gincontrib.page.cache"

/*
responseCache stores the HTTP response status, headers, and body data for caching.
*/
type responseCache struct {
	Status int
	Header http.Header
	Data   []byte
}

/*
RegisterResponseCacheGob registers the responseCache type with the encoding/gob package.
This is required for gob-based cache stores to serialize/deserialize cached responses.
*/
func RegisterResponseCacheGob() {
	gob.Register(responseCache{})
}

/*
cachedWriter is a Gin ResponseWriter wrapper that intercepts writes to cache the response.
It stores the response status, headers, and body, and writes them to the configured cache store.
*/
type cachedWriter struct {
	gin.ResponseWriter
	status  int
	written bool
	store   persistence.CacheStore
	expire  time.Duration
	key     string
}

var _ gin.ResponseWriter = &cachedWriter{}

/*
CreateKey generates a cache key for the given string using the package-specific prefix.
*/
func CreateKey(u string) string {
	return generateCacheKey(PageCachePrefix, u)
}

/*
hasherPool is a sync.Pool of xxhash.Digest objects for efficient hash computation.
*/
var hasherPool = sync.Pool{
	New: func() interface{} {
		return xxhash.New()
	},
}

/*
builderPool is a sync.Pool of strings.Builder objects for efficient string concatenation.
*/
var builderPool = sync.Pool{
	New: func() interface{} {
		return &strings.Builder{}
	},
}

/*
generateCacheKey creates a cache key by hashing the input string and prepending the given prefix.
It uses object pools for hashers and string builders to reduce allocations.
*/
func generateCacheKey(prefix string, u string) string {
	h := hasherPool.Get().(*xxhash.Digest)
	h.Reset()
	_, _ = io.WriteString(h, u)
	key := hex.EncodeToString(h.Sum(nil))
	hasherPool.Put(h)

	builder := builderPool.Get().(*strings.Builder)
	builder.Reset()
	builder.WriteString(prefix)
	builder.WriteString(":")
	builder.WriteString(key)
	result := builder.String()
	builderPool.Put(builder)
	return result
}

/*
newCachedWriter constructs a new cachedWriter wrapping the given Gin ResponseWriter.
*/
func newCachedWriter(store persistence.CacheStore, expire time.Duration, writer gin.ResponseWriter, key string) *cachedWriter {
	return &cachedWriter{writer, 0, false, store, expire, key}
}

/*
WriteHeader sets the HTTP status code and marks the response as written.
*/
func (w *cachedWriter) WriteHeader(code int) {
	w.status = code
	w.written = true
	w.ResponseWriter.WriteHeader(code)
}

/*
Status returns the current HTTP status code of the response.
*/
func (w *cachedWriter) Status() int {
	return w.ResponseWriter.Status()
}

/*
Written reports whether the response has been written.
*/
func (w *cachedWriter) Written() bool {
	return w.ResponseWriter.Written()
}

/*
Write writes data to the underlying ResponseWriter and caches the response if status < 300.
If a previous cache entry exists, it appends the new data to the cached data.
*/
func (w *cachedWriter) Write(data []byte) (int, error) {
	ret, err := w.ResponseWriter.Write(data)
	if err == nil {
		store := w.store
		var cache responseCache

		if err := store.Get(w.key, &cache); err == nil {
			cache.Data = append(cache.Data, data...)
		}else{
			cache.Data = make([]byte,0)
			cache.Data = append(cache.Data, data...)
		}

		// Cache responses with a status code < 300
		if w.Status() < 300 {
			val := responseCache{
				w.Status(),
				w.Header(),
				cache.Data,
			}
			err = store.Set(w.key, val, w.expire)
			// if err != nil {
			// 	// need logger
			// }
		}
	}
	return ret, err
}

/*
WriteString writes a string to the underlying ResponseWriter and caches the response if status < 300.
*/
func (w *cachedWriter) WriteString(data string) (n int, err error) {
	ret, err := w.ResponseWriter.WriteString(data)
	// Cache responses with a status code < 300
	if err == nil && w.Status() < 300 {
		store := w.store
		val := responseCache{
			w.Status(),
			w.Header(),
			[]byte(data),
		}
		_ = store.Set(w.key, val, w.expire)
	}
	return ret, err
}

/*
Cache is a Gin middleware that injects the cache store into the request context.
*/
func Cache(store *persistence.CacheStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(CACHE_MIDDLEWARE_KEY, store)
		c.Next()
	}
}

/*
SiteCache is a Gin middleware that caches entire site responses based on the request URI.
If a cached response exists, it is written directly; otherwise, the request proceeds as normal.
*/
func SiteCache(store persistence.CacheStore, expire time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		var cache responseCache
		url := c.Request.URL
		key := CreateKey(url.RequestURI())
		if err := store.Get(key, &cache); err != nil {
			c.Next()
		} else {
			c.Writer.WriteHeader(cache.Status)
			for k, vals := range cache.Header {
				for _, v := range vals {
					c.Writer.Header().Set(k, v)
				}
			}
			_, _ = c.Writer.Write(cache.Data)
		}
	}
}

// CachePage is a decorator that caches the response of the given handler based on the request URI.
// If a cached response exists, it is served directly. Otherwise, the handler is executed and its response is cached.
// If the context is aborted, the cache entry is deleted.
func CachePage(store persistence.CacheStore, expire time.Duration, handle gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		var cache responseCache
		url := c.Request.URL
		key := CreateKey(url.RequestURI())
		if err := store.Get(key, &cache); err != nil {
			if err != persistence.ErrCacheMiss {
				log.Println(err.Error())
			}
			// Replace writer with cachedWriter to intercept response
			writer := newCachedWriter(store, expire, c.Writer, key)
			c.Writer = writer
			handle(c)

			// Drop caches of aborted contexts
			if c.IsAborted() {
				_ = store.Delete(key)
			}
		} else {
			c.Writer.WriteHeader(cache.Status)
			for k, vals := range cache.Header {
				for _, v := range vals {
					c.Writer.Header().Set(k, v)
				}
			}
			_, _ = c.Writer.Write(cache.Data)
		}
	}
}

// CachePageWithoutQuery is a decorator that caches responses ignoring GET query parameters.
// The cache key is based only on the request path, so all queries to the same path share the cache.
func CachePageWithoutQuery(store persistence.CacheStore, expire time.Duration, handle gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		var cache responseCache
		key := CreateKey(c.Request.URL.Path)
		if err := store.Get(key, &cache); err != nil {
			if err != persistence.ErrCacheMiss {
				log.Println(err.Error())
			}
			// Replace writer with cachedWriter to intercept response
			writer := newCachedWriter(store, expire, c.Writer, key)
			c.Writer = writer
			handle(c)
		} else {
			c.Writer.WriteHeader(cache.Status)
			for k, vals := range cache.Header {
				for _, v := range vals {
					c.Writer.Header().Set(k, v)
				}
			}
			_, _ = c.Writer.Write(cache.Data)
		}
	}
}

// CachePageAtomic is a decorator that wraps CachePage with a mutex to ensure atomic access.
// This prevents concurrent requests from generating duplicate cache entries for the same resource.
func CachePageAtomic(store persistence.CacheStore, expire time.Duration, handle gin.HandlerFunc) gin.HandlerFunc {
	var m sync.Mutex
	p := CachePage(store, expire, handle)
	return func(c *gin.Context) {
		m.Lock()
		defer m.Unlock()
		p(c)
	}
}

/*
CachePageWithoutHeader is a decorator that caches responses without restoring headers from the cache.
Only the status and body are restored from the cache.
*/
func CachePageWithoutHeader(store persistence.CacheStore, expire time.Duration, handle gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		var cache responseCache
		url := c.Request.URL
		key := CreateKey(url.RequestURI())
		if err := store.Get(key, &cache); err != nil {
			if err != persistence.ErrCacheMiss {
				log.Println(err.Error())
			}
			// Replace writer with cachedWriter to intercept response
			writer := newCachedWriter(store, expire, c.Writer, key)
			c.Writer = writer
			handle(c)

			// Drop caches of aborted contexts
			if c.IsAborted() {
				_ = store.Delete(key)
			}
		} else {
			c.Writer.WriteHeader(cache.Status)
			_, _ = c.Writer.Write(cache.Data)
		}
	}
}
