package cache

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-contrib/cache/persistence"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestCache(t *testing.T) {
	// TODO:unit test
}

func TestWrite(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	store := persistence.NewInMemoryStore(60 * time.Second)
	writer := newCachedWriter(store, time.Second*3, c.Writer, "mykey")
	c.Writer = writer

	c.Writer.WriteHeader(204)
	c.Writer.WriteHeaderNow()
	_, _ = c.Writer.Write([]byte("foo"))
	assert.Equal(t, 204, c.Writer.Status())
	assert.Equal(t, "foo", w.Body.String())
	assert.True(t, c.Writer.Written())
}

func TestCachePage(t *testing.T) {
	store := persistence.NewInMemoryStore(60 * time.Second)

	router := gin.New()
	router.GET("/cache_ping", CachePage(store, time.Second*3, func(c *gin.Context) {
		c.String(200, "pong "+fmt.Sprint(time.Now().UnixNano()))
	}))

	w1 := performRequest("GET", "/cache_ping", nil, router)
	w2 := performRequest("GET", "/cache_ping", nil, router)

	assert.Equal(t, 200, w1.Code)
	assert.Equal(t, 200, w2.Code)
	assert.Equal(t, w1.Body.String(), w2.Body.String())
}

func TestCachePageExpire(t *testing.T) {
	store := persistence.NewInMemoryStore(60 * time.Second)

	router := gin.New()
	router.GET("/cache_ping", CachePage(store, time.Second, func(c *gin.Context) {
		c.String(200, "pong "+fmt.Sprint(time.Now().UnixNano()))
	}))

	w1 := performRequest("GET", "/cache_ping", nil, router)
	time.Sleep(time.Second * 2)
	w2 := performRequest("GET", "/cache_ping", nil, router)

	assert.Equal(t, 200, w1.Code)
	assert.Equal(t, 200, w2.Code)
	assert.NotEqual(t, w1.Body.String(), w2.Body.String())
}

func TestCachePageAtomic(t *testing.T) {
	// memoryDelayStore is a wrapper of a InMemoryStore
	// designed to simulate data race (by doing a delayed write)
	store := newDelayStore(60 * time.Second)

	router := gin.New()
	router.GET("/atomic", CachePageAtomic(store, time.Second*5, func(c *gin.Context) {
		c.String(200, "OK")
	}))

	outp := make(chan string, 10)

	for i := 0; i < 5; i++ {
		go func() {
			resp := performRequest("GET", "/atomic", nil, router)
			outp <- resp.Body.String()
		}()
	}
	time.Sleep(time.Millisecond * 500)
	for i := 0; i < 5; i++ {
		go func() {
			resp := performRequest("GET", "/atomic", nil, router)
			outp <- resp.Body.String()
		}()
	}
	time.Sleep(time.Millisecond * 500)

	for i := 0; i < 10; i++ {
		v := <-outp
		assert.Equal(t, "OK", v)
	}
}

func TestCachePageWithoutHeader(t *testing.T) {
	store := persistence.NewInMemoryStore(60 * time.Second)

	router := gin.New()
	router.GET("/cache_ping", CachePageWithoutHeader(store, time.Second*3, func(c *gin.Context) {
		c.String(200, "pong "+fmt.Sprint(time.Now().UnixNano()))
	}))

	w1 := performRequest("GET", "/cache_ping", nil, router)
	w2 := performRequest("GET", "/cache_ping", nil, router)

	assert.Equal(t, 200, w1.Code)
	assert.Equal(t, 200, w2.Code)
	assert.NotNil(t, w1.Header()["Content-Type"])
	assert.Nil(t, w2.Header()["Content-Type"])
	assert.Equal(t, w1.Body.String(), w2.Body.String())
}

func TestCachePageWithoutHeaderExpire(t *testing.T) {
	store := persistence.NewInMemoryStore(60 * time.Second)

	router := gin.New()
	router.GET("/cache_ping", CachePage(store, time.Second, func(c *gin.Context) {
		c.String(200, "pong "+fmt.Sprint(time.Now().UnixNano()))
	}))

	w1 := performRequest("GET", "/cache_ping", nil, router)
	time.Sleep(time.Second * 2)
	w2 := performRequest("GET", "/cache_ping", nil, router)

	assert.Equal(t, 200, w1.Code)
	assert.Equal(t, 200, w2.Code)
	assert.NotNil(t, w1.Header()["Content-Type"])
	assert.NotNil(t, w2.Header()["Content-Type"])
	assert.NotEqual(t, w1.Body.String(), w2.Body.String())
}

func TestCacheHtmlFile(t *testing.T) {
	store := persistence.NewInMemoryStore(60 * time.Second)

	router := gin.New()
	router.LoadHTMLFiles("_example/template.html")
	router.GET("/cache_html", CachePage(store, time.Second*3, func(c *gin.Context) {
		c.HTML(http.StatusOK, "template.html", gin.H{"values": fmt.Sprint(time.Now().UnixNano())})
	}))

	w1 := performRequest("GET", "/cache_html", nil, router)
	w2 := performRequest("GET", "/cache_html", nil, router)

	assert.Equal(t, 200, w1.Code)
	assert.Equal(t, 200, w2.Code)
	assert.Equal(t, w1.Body.String(), w2.Body.String())
}

func TestCacheHtmlFileExpire(t *testing.T) {
	store := persistence.NewInMemoryStore(60 * time.Second)

	router := gin.New()
	router.LoadHTMLFiles("_example/template.html")
	router.GET("/cache_html", CachePage(store, time.Second*1, func(c *gin.Context) {
		c.HTML(http.StatusOK, "template.html", gin.H{"values": fmt.Sprint(time.Now().UnixNano())})
	}))

	w1 := performRequest("GET", "/cache_html", nil, router)
	time.Sleep(time.Second * 2)
	w2 := performRequest("GET", "/cache_html", nil, router)

	assert.Equal(t, 200, w1.Code)
	assert.Equal(t, 200, w2.Code)
	assert.NotEqual(t, w1.Body.String(), w2.Body.String())
}

func TestCachePageAborted(t *testing.T) {
	store := persistence.NewInMemoryStore(60 * time.Second)

	router := gin.New()
	router.GET("/cache_aborted", CachePage(store, time.Second*3, func(c *gin.Context) {
		c.AbortWithStatusJSON(200, map[string]int64{"time": time.Now().UnixNano()})
	}))

	w1 := performRequest("GET", "/cache_aborted", nil, router)
	time.Sleep(time.Millisecond * 500)
	w2 := performRequest("GET", "/cache_aborted", nil, router)

	assert.Equal(t, 200, w1.Code)
	assert.Equal(t, 200, w2.Code)
	assert.NotEqual(t, w1.Body.String(), w2.Body.String())
}

func TestCachePage400(t *testing.T) {
	store := persistence.NewInMemoryStore(60 * time.Second)

	router := gin.New()
	router.GET("/cache_400", CachePage(store, time.Second*3, func(c *gin.Context) {
		c.String(400, fmt.Sprint(time.Now().UnixNano()))
	}))

	w1 := performRequest("GET", "/cache_400", nil, router)
	time.Sleep(time.Millisecond * 500)
	w2 := performRequest("GET", "/cache_400", nil, router)

	assert.Equal(t, 400, w1.Code)
	assert.Equal(t, 400, w2.Code)
	assert.NotEqual(t, w1.Body.String(), w2.Body.String())
}

func TestCachePageWithoutHeaderAborted(t *testing.T) {
	store := persistence.NewInMemoryStore(60 * time.Second)

	router := gin.New()
	router.GET("/cache_aborted", CachePage(store, time.Second*3, func(c *gin.Context) {
		c.AbortWithStatusJSON(200, map[string]int64{"time": time.Now().UnixNano()})
	}))

	w1 := performRequest("GET", "/cache_aborted", nil, router)
	time.Sleep(time.Millisecond * 500)
	w2 := performRequest("GET", "/cache_aborted", nil, router)

	assert.Equal(t, 200, w1.Code)
	assert.Equal(t, 200, w2.Code)
	assert.NotNil(t, w1.Header()["Content-Type"])
	assert.NotNil(t, w2.Header()["Content-Type"])
	assert.NotEqual(t, w1.Body.String(), w2.Body.String())
}

func TestCachePageWithoutHeader400(t *testing.T) {
	store := persistence.NewInMemoryStore(60 * time.Second)

	router := gin.New()
	router.GET("/cache_400", CachePage(store, time.Second*3, func(c *gin.Context) {
		c.String(400, fmt.Sprint(time.Now().UnixNano()))
	}))

	w1 := performRequest("GET", "/cache_400", nil, router)
	time.Sleep(time.Millisecond * 500)
	w2 := performRequest("GET", "/cache_400", nil, router)

	assert.Equal(t, 400, w1.Code)
	assert.Equal(t, 400, w2.Code)
	assert.NotNil(t, w1.Header()["Content-Type"])
	assert.NotNil(t, w2.Header()["Content-Type"])
	assert.NotEqual(t, w1.Body.String(), w2.Body.String())
}

func TestCachePageStatus207(t *testing.T) {
	store := persistence.NewInMemoryStore(60 * time.Second)

	router := gin.New()
	router.GET("/cache_207", CachePage(store, time.Second*3, func(c *gin.Context) {
		c.String(207, fmt.Sprint(time.Now().UnixNano()))
	}))

	w1 := performRequest("GET", "/cache_207", nil, router)
	time.Sleep(time.Millisecond * 500)
	w2 := performRequest("GET", "/cache_207", nil, router)

	assert.Equal(t, 207, w1.Code)
	assert.Equal(t, 207, w2.Code)
	assert.Equal(t, w1.Body.String(), w2.Body.String())
}

func TestCachePageWithoutQuery(t *testing.T) {
	store := persistence.NewInMemoryStore(60 * time.Second)

	router := gin.New()
	router.GET("/cache_without_query", CachePageWithoutQuery(store, time.Second*3, func(c *gin.Context) {
		c.String(200, "pong "+fmt.Sprint(time.Now().UnixNano()))
	}))

	w1 := performRequest("GET", "/cache_without_query?foo=1", nil, router)
	w2 := performRequest("GET", "/cache_without_query?foo=2", nil, router)

	assert.Equal(t, 200, w1.Code)
	assert.Equal(t, 200, w2.Code)
	assert.Equal(t, w1.Body.String(), w2.Body.String())
}

func TestCachePageWithRequestBody(t *testing.T) {
	store := persistence.NewInMemoryStore(60 * time.Second)

	router := gin.New()

	router.POST("/cache_req_body", CachePageWithRequestBody(store, time.Second*3, func(c *gin.Context) {
		c.String(200, "pong "+fmt.Sprint(time.Now().UnixNano()))
	}))

	w1 := performRequest("POST", "/cache_req_body", strings.NewReader(`{"name":"John","age":30}`), router)
	w2 := performRequest("POST", "/cache_req_body", strings.NewReader(`{"name":"John","age":30}`), router)
	w3 := performRequest("POST", "/cache_req_body", strings.NewReader(`{"name":"John","age":31}`), router)

	assert.Equal(t, 200, w1.Code)
	assert.Equal(t, 200, w2.Code)
	assert.Equal(t, 200, w3.Code)

	assert.Equal(t, w1.Body.String(), w2.Body.String())
	assert.NotEqual(t, w1.Body.String(), w3.Body.String())
	assert.NotEqual(t, w2.Body.String(), w3.Body.String())

}

func TestRegisterResponseCacheGob(t *testing.T) {
	RegisterResponseCacheGob()
	r := responseCache{Status: 200, Data: []byte("test")}
	mCache := new(bytes.Buffer)
	encCache := gob.NewEncoder(mCache)
	err := encCache.Encode(r)
	assert.Nil(t, err)

	var decodedResp responseCache
	pCache := bytes.NewBuffer(mCache.Bytes())
	decCache := gob.NewDecoder(pCache)
	err = decCache.Decode(&decodedResp)
	assert.Nil(t, err)
}

func performRequest(method, target string, body io.Reader, router *gin.Engine) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, target, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	return w
}

type memoryDelayStore struct {
	*persistence.InMemoryStore
}

func newDelayStore(defaultExpiration time.Duration) *memoryDelayStore {
	v := &memoryDelayStore{}
	v.InMemoryStore = persistence.NewInMemoryStore(defaultExpiration)
	return v
}

func (c *memoryDelayStore) Set(key string, value any, expires time.Duration) error {
	time.Sleep(time.Millisecond * 3)
	return c.InMemoryStore.Set(key, value, expires)
}

func (c *memoryDelayStore) Add(key string, value any, expires time.Duration) error {
	time.Sleep(time.Millisecond * 3)
	return c.InMemoryStore.Add(key, value, expires)
}
