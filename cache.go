package cache

import (
	"bytes"
	"compress/gzip"
	"crypto/sha1"
	"encoding/gob"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
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
	buffer  *bytes.Buffer
	status  int
	written bool
	aborted bool
	store   persistence.CacheStore
	expire  time.Duration
	key     string
}

type KeyGenerator func(c *gin.Context) string

type WriterHook func(c *gin.Context, cache responseCache)

type Middleware func(store persistence.CacheStore, expire time.Duration, handle gin.HandlerFunc) gin.HandlerFunc

var _ gin.ResponseWriter = &cachedWriter{}

// CreateKey creates a package specific key for a given string
func CreateKey(u string) string {
	return urlEscape(PageCachePrefix, u)
}

func newCachedWriter(
	store persistence.CacheStore,
	expire time.Duration,
	writer gin.ResponseWriter,
	key string,
) *cachedWriter {
	return &cachedWriter{
		writer,
		bytes.NewBuffer([]byte{}),
		0,
		false,
		false,
		store,
		expire,
		key,
	}
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
	return w.buffer.Write(data)
}

func (w *cachedWriter) WriteString(data string) (n int, err error) {
	return w.buffer.WriteString(data)
}

func (w *cachedWriter) Close() error {
	val := responseCache{
		w.Status(),
		w.Header(),
		w.readCompressed(w.buffer.Bytes()),
	}
	w.ResponseWriter.Write(val.Data)
	if w.Status() >= 300 || w.aborted {
		return nil
	}
	return w.store.Set(w.key, val, w.expire)
}

func (w *cachedWriter) Abort() {
	w.aborted = true
}

func (w *cachedWriter) readCompressed(data []byte) []byte {
	if strings.Contains(w.ResponseWriter.Header().Get("Content-Encoding"), "gzip") {
		if reader, err := gzip.NewReader(bytes.NewReader(data)); err != nil {
			fmt.Println(err.Error())
		} else if b, err := ioutil.ReadAll(reader); err != nil {
			fmt.Println(err.Error())
		} else {
			return b
		}
	}
	return data
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
		if err := store.Get(RequestURIKey(c), &cache); err != nil {
			c.Next()
		} else {
			WriteWithHeaders(c, cache)
		}
	}
}

func CacheCustom(
	store persistence.CacheStore,
	expire time.Duration,
	handle gin.HandlerFunc,
	keyGenerator KeyGenerator,
	writerHook WriterHook,
) gin.HandlerFunc {
	return handleCache(store, expire, handle, keyGenerator, writerHook)
}

// CachePage Decorator
func CachePage(store persistence.CacheStore, expire time.Duration, handle gin.HandlerFunc) gin.HandlerFunc {
	return handleCache(store, expire, handle, RequestURIKey, WriteWithHeaders)
}

// CachePageWithoutQuery add ability to ignore GET query parameters.
func CachePageWithoutQuery(store persistence.CacheStore, expire time.Duration, handle gin.HandlerFunc) gin.HandlerFunc {
	return handleCache(store, expire, handle, WithoutParamKey, WriteWithHeaders)
}

func CachePageWithoutHeader(store persistence.CacheStore, expire time.Duration, handle gin.HandlerFunc) gin.HandlerFunc {
	return handleCache(store, expire, handle, RequestURIKey, WriteWithoutHeaders)
}

func CachePageAtomic(store persistence.CacheStore, expire time.Duration, handle gin.HandlerFunc) gin.HandlerFunc {
	return CachePageAtomicDecorator(CachePage, store, expire, handle)
}

// CachePageAtomic Decorator
func CachePageAtomicDecorator(
	middleware Middleware,
	store persistence.CacheStore,
	expire time.Duration,
	handle gin.HandlerFunc,
) gin.HandlerFunc {
	var m sync.Mutex
	p := middleware(store, expire, handle)
	return func(c *gin.Context) {
		m.Lock()
		defer m.Unlock()
		p(c)
	}
}

func WriteWithHeaders(c *gin.Context, cache responseCache) {
	for k, vals := range cache.Header {
		for _, v := range vals {
			c.Writer.Header().Set(k, v)
		}
	}
	WriteWithoutHeaders(c, cache)
}

func WriteWithoutHeaders(c *gin.Context, cache responseCache) {
	c.Writer.WriteHeader(cache.Status)
	c.Writer.Write(cache.Data)
}

func RequestURIKey(c *gin.Context) string {
	return CreateKey(c.Request.URL.RequestURI())
}

func WithoutParamKey(c *gin.Context) string {
	return CreateKey(c.Request.URL.Path)
}

func handleCache(
	store persistence.CacheStore,
	expire time.Duration,
	handle gin.HandlerFunc,
	keyGenerator KeyGenerator,
	writerHook WriterHook,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		var cache responseCache
		key := keyGenerator(c)
		err := store.Get(key, &cache)
		if err != nil && err != persistence.ErrCacheMiss {
			log.Println(err.Error())
		} else if err == nil {
			writerHook(c, cache)
		} else {
			callHandle(c, handle, store, expire, key)
		}
	}
}

func callHandle(
	c *gin.Context,
	handle gin.HandlerFunc,
	store persistence.CacheStore,
	expire time.Duration,
	key string,
) {
	writer := newCachedWriter(store, expire, c.Writer, key)
	c.Writer = writer
	handle(c)

	c.Next()
	if c.IsAborted() {
		writer.Abort()
	}
	writer.Close()
}

func urlEscape(prefix string, u string) string {
	key := url.QueryEscape(u)
	if len(key) > 200 {
		h := sha1.New()
		io.WriteString(h, u)
		key = string(h.Sum(nil))
	}
	var buffer bytes.Buffer
	buffer.WriteString(prefix)
	buffer.WriteString(":")
	buffer.WriteString(key)
	return buffer.String()
}
