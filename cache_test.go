package cache

import (
	"fmt"
	"net/http"
	"net/http/httptest"
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
	//TODO:unit test
}

func TestWrite(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	store := persistence.NewInMemoryStore(60 * time.Second)
	writer := newCachedWriter(store, time.Second*3, c.Writer, "mykey")
	c.Writer = writer

	c.Writer.WriteHeader(204)
	c.Writer.WriteHeaderNow()
	c.Writer.Write([]byte("foo"))
	assert.Equal(t, 204, c.Writer.Status())
	assert.Equal(t, "foo", w.Body.String())
	assert.True(t, c.Writer.Written())
}

func TestCachePage(t *testing.T) {
	store := persistence.NewInMemoryStore(60 * time.Second)

	router := gin.New()
	router.GET("/cache_ping", CachePage(store, time.Second*3, func(c *gin.Context) {
		c.String(200, "pong "+fmt.Sprint(time.Now().Unix()))
	}))

	w1 := performRequest("GET", "/cache_ping", router)
	w2 := performRequest("GET", "/cache_ping", router)

	assert.Equal(t, 200, w1.Code)
	assert.Equal(t, 200, w2.Code)
	assert.Equal(t, w1.Body.String(), w2.Body.String())
}

func TestCachePageExpire(t *testing.T) {
	store := persistence.NewInMemoryStore(60 * time.Second)

	router := gin.New()
	router.GET("/cache_ping", CachePage(store, time.Second, func(c *gin.Context) {
		c.String(200, "pong "+fmt.Sprint(time.Now().Unix()))
	}))

	w1 := performRequest("GET", "/cache_ping", router)
	time.Sleep(time.Second * 2)
	w2 := performRequest("GET", "/cache_ping", router)

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
			resp := performRequest("GET", "/atomic", router)
			outp <- resp.Body.String()
		}()
	}
	time.Sleep(time.Millisecond * 500)
	for i := 0; i < 5; i++ {
		go func() {
			resp := performRequest("GET", "/atomic", router)
			outp <- resp.Body.String()
		}()
	}
	time.Sleep(time.Millisecond * 500)

	for i := 0; i < 10; i++ {
		v := <-outp
		assert.Equal(t, "OK", v)
	}
}

func TestCacheHtmlFile(t *testing.T) {
	store := persistence.NewInMemoryStore(60 * time.Second)

	router := gin.New()
	router.LoadHTMLFiles("example/template.html")
	router.GET("/cache_html", CachePage(store, time.Second*3, func(c *gin.Context) {
		c.HTML(http.StatusOK, "template.html", gin.H{"values": fmt.Sprint(time.Now().Unix())})
	}))

	w1 := performRequest("GET", "/cache_html", router)
	w2 := performRequest("GET", "/cache_html", router)

	assert.Equal(t, 200, w1.Code)
	assert.Equal(t, 200, w2.Code)
	assert.Equal(t, w1.Body.String(), w2.Body.String())
}

func TestCacheHtmlFileExpire(t *testing.T) {
	store := persistence.NewInMemoryStore(60 * time.Second)

	router := gin.New()
	router.LoadHTMLFiles("example/template.html")
	router.GET("/cache_html", CachePage(store, time.Second*1, func(c *gin.Context) {
		c.HTML(http.StatusOK, "template.html", gin.H{"values": fmt.Sprint(time.Now().Unix())})
	}))

	w1 := performRequest("GET", "/cache_html", router)
	time.Sleep(time.Second * 2)
	w2 := performRequest("GET", "/cache_html", router)

	assert.Equal(t, 200, w1.Code)
	assert.Equal(t, 200, w2.Code)
	assert.NotEqual(t, w1.Body.String(), w2.Body.String())
}

func performRequest(method, target string, router *gin.Engine) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, target, nil)
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

func (c *memoryDelayStore) Set(key string, value interface{}, expires time.Duration) error {
	time.Sleep(time.Millisecond * 3)
	return c.InMemoryStore.Set(key, value, expires)
}

func (c *memoryDelayStore) Add(key string, value interface{}, expires time.Duration) error {
	time.Sleep(time.Millisecond * 3)
	return c.InMemoryStore.Add(key, value, expires)
}
