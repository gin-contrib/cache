package cache

import (
	"bytes"
	"crypto/sha1"
	"encoding/gob"
	"io"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gin-contrib/cache/persistence"
	"github.com/gin-gonic/gin"
)

const (
	CACHE_MIDDLEWARE_KEY = "gincontrib.cache"
)

var (
	PageCachePrefix = "gincontrib.page.cache"
)

type responseCache struct {
	Status int
	Header http.Header
	Data   []byte
}

// RegisterResponseCacheGob registers the responseCache type with the encoding/gob package
func RegisterResponseCacheGob() {
	gob.Register(responseCache{})
}

type cachedWriter struct {
	gin.ResponseWriter
	status  int
	written bool
	store   persistence.CacheStore
	expire  time.Duration
	key     string
}

var _ gin.ResponseWriter = &cachedWriter{}

// CreateKey creates a package specific key for a given string
func CreateKey(u string) string {
	return urlEscape(PageCachePrefix, u)
}

func urlEscape(prefix string, u string) string {
	key := url.QueryEscape(u)
	if len(key) > 200 {
		h := sha1.New()
		_, _ = io.WriteString(h, u)
		key = string(h.Sum(nil))
	}
	var buffer bytes.Buffer
	buffer.WriteString(prefix)
	buffer.WriteString(":")
	buffer.WriteString(key)
	return buffer.String()
}

func newCachedWriter(store persistence.CacheStore, expire time.Duration, writer gin.ResponseWriter, key string) *cachedWriter {
	return &cachedWriter{writer, 0, false, store, expire, key}
}

func (w *cachedWriter) WriteHeader(code int) {
	w.status = code
	w.written = true
	w.ResponseWriter.WriteHeader(code)
}

func (w *cachedWriter) Status() int {
	return w.ResponseWriter.Status()
}

func (w *cachedWriter) Written() bool {
	return w.ResponseWriter.Written()
}

func (w *cachedWriter) Write(data []byte) (int, error) {
	ret, err := w.ResponseWriter.Write(data)
	if err == nil {
		store := w.store
		var cache responseCache
		if err := store.Get(w.key, &cache); err == nil {
			data = append(cache.Data, data...)
		}

		//cache responses with a status code < 300
		if w.Status() < 300 {
			val := responseCache{
				w.Status(),
				w.Header(),
				data,
			}
			err = store.Set(w.key, val, w.expire)
			// if err != nil {
			// 	// need logger
			// }
		}
	}
	return ret, err
}

func (w *cachedWriter) WriteString(data string) (n int, err error) {
	ret, err := w.ResponseWriter.WriteString(data)
	//cache responses with a status code < 300
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

// Cache Middleware
func Cache(store *persistence.CacheStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(CACHE_MIDDLEWARE_KEY, store)
		c.Next()
	}
}

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

// CachePage Decorator
func CachePage(store persistence.CacheStore, expire time.Duration, handle gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		var cache responseCache
		url := c.Request.URL
		key := CreateKey(url.RequestURI())
		if err := store.Get(key, &cache); err != nil {
			if err != persistence.ErrCacheMiss {
				log.Println(err.Error())
			}
			// replace writer
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


// CachePage Decorator to work with http header
func CachePageWithHeader(store persistence.CacheStore, expire time.Duration, handle gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		var cache responseCache
		url := c.Request.URL
		header := strings.Join(c.Request.Header["X-Session-Data"], "-")
		key := CreateKey(url.RequestURI() + header)
		if err := store.Get(key, &cache); err != nil {
			if err != persistence.ErrCacheMiss {
				log.Println(err.Error())
			}
			// replace writer
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
// CachePageWithoutQuery add ability to ignore GET query parameters.
func CachePageWithoutQuery(store persistence.CacheStore, expire time.Duration, handle gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		var cache responseCache
		key := CreateKey(c.Request.URL.Path)
		if err := store.Get(key, &cache); err != nil {
			if err != persistence.ErrCacheMiss {
				log.Println(err.Error())
			}
			// replace writer
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

// CachePageAtomic Decorator
func CachePageAtomic(store persistence.CacheStore, expire time.Duration, handle gin.HandlerFunc) gin.HandlerFunc {
	var m sync.Mutex
	p := CachePage(store, expire, handle)
	return func(c *gin.Context) {
		m.Lock()
		defer m.Unlock()
		p(c)
	}
}

func CachePageWithoutHeader(store persistence.CacheStore, expire time.Duration, handle gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		var cache responseCache
		url := c.Request.URL
		key := CreateKey(url.RequestURI())
		if err := store.Get(key, &cache); err != nil {
			if err != persistence.ErrCacheMiss {
				log.Println(err.Error())
			}
			// replace writer
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
