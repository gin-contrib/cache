package cache

import (
	"bytes"
	"crypto/sha1"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
	"github.com/lox/httpcache"
	"net/http/httputil"
	"encoding/json"
	"io/ioutil"
	"fmt"
	"errors"

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

type cachedWriter struct {
	gin.ResponseWriter
	status  int
	written bool
	store   persistence.CacheStore
	expire  time.Duration
	key     string
}

var _ gin.ResponseWriter = &cachedWriter{}

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

func (w *cachedWriter) IsSuccess() bool {
	return w.Status() == 200 || w.Status() == 202
}

func (w *cachedWriter) Write(data []byte) (int, error) {
	ret, err := w.ResponseWriter.Write(data)
	if err == nil  &&  w.IsSuccess() {
		store := w.store
		var cache responseCache
		if err := store.Get(w.key, &cache); err == nil {
			data = append(cache.Data, data...)
		}

		//cache response
		val := responseCache{
			w.status,
			w.Header(),
			data,
		}
		err = store.Set(w.key, val, w.expire)
		if err != nil {
			// need logger
		}
	}
	return ret, err
}

func (w *cachedWriter) WriteString(data string) (n int, err error) {
	ret, err := w.ResponseWriter.WriteString(data)
	if err == nil && w.IsSuccess() {
		//cache response
		store := w.store
		val := responseCache{
			w.status,
			w.Header(),
			[]byte(data),
		}
		store.Set(w.key, val, w.expire)
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

		key :=  httpcache.NewKey(c.Request.Method, c.Request.URL, c.Request.Header).String()
		if err := store.Get(key, &cache); err != nil {
			c.Next()
		} else {
			c.Writer.WriteHeader(cache.Status)
			for k, vals := range cache.Header {
				for _, v := range vals {
					c.Writer.Header().Add(k, v)
				}
			}
			c.Writer.Write(cache.Data)
		}
	}
}

// CachePage Decorator
func CachePage(store persistence.CacheStore, expire time.Duration, handle gin.HandlerFunc) gin.HandlerFunc {

	return func(c *gin.Context) {
		var cache responseCache
		
		key :=  httpcache.NewKey(c.Request.Method, c.Request.URL, c.Request.Header).String()
		if err := store.Get(key, &cache); err != nil {
			log.Println(err.Error())
			// replace writer
			writer := newCachedWriter(store, expire, c.Writer, key)
			c.Writer = writer
			handle(c)
		} else {
			c.Writer.WriteHeader(cache.Status)
			for k, vals := range cache.Header {
				for _, v := range vals {
					c.Writer.Header().Add(k, v)
				}
			}
			c.Writer.Write(cache.Data)
		}
	}
}


func CachePageIncludeBodyAsKey(store persistence.CacheStore, expire time.Duration, handle gin.HandlerFunc) gin.HandlerFunc {

	return func(c *gin.Context) {
		var cache responseCache
		
		key, err :=  newKeyWithBody(c.Request)
		if err != nil {
			log.Println(err.Error())
			writer := newCachedWriter(store, expire, c.Writer, key)
			c.Writer = writer
			handle(c)
		} else { 
			if err = store.Get(key, &cache); err != nil {
				log.Println(err.Error())
				// replace writer
				writer := newCachedWriter(store, expire, c.Writer, key)
				c.Writer = writer
				handle(c)
			} else {
				c.Writer.WriteHeader(cache.Status)
				for k, vals := range cache.Header {
					for _, v := range vals {
						c.Writer.Header().Add(k, v)
					}
				}
				c.Writer.Write(cache.Data)
			}
		}
	}


}

func newKeyWithBody(r *http.Request) (string, error) {

	if r.Body == nil {
		return "", errors.New("no body")
	}

	dump, err := httputil.DumpRequest(r, false)
	if err != nil {
		return "", err
	} else {
		sortedBytes, err := sortBody(r)
		if err == nil {
			out := fmt.Sprintf("%v:%v", dump, sortedBytes)
			//fmt.Printf("key = %s\n", out)
			return out, nil
		} else {
			return "", err
		}
	}
}


func sortBody(r *http.Request) ([]byte, error) {
	var buf bytes.Buffer
  	if _, err := buf.ReadFrom(r.Body); err != nil {
  		return nil, err
  	}
  		
  	if err := r.Body.Close(); err != nil {
  		return  nil, err
  	}


	// Note : json key order (including maps) is undefined. 
	// but https://github.com/golang/go/issues/15424 says go sorts keys
	// 
	
	// but we might get calls from non-go clients
	// to get around this we marshall and unmarshall


	var res interface{}
	b := buf.Bytes()
	err := json.Unmarshal(b, &res)
	if err != nil {
		// if it's not json, it will be caught here
		return nil, err
	}
	bs, err := json.Marshal(res)
	if err != nil {
		return nil, err
	}

	r.Body = ioutil.NopCloser(bytes.NewReader(bs))
	return bs, nil 
}



