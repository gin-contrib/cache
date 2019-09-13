package cache

import (
	"bytes"
	"encoding/gob"
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
	opts := persistence.Options{
		Adapter:           persistence.AdapterInMemoryStore,
		DefaultExpiration: 60 * time.Second,
	}
	store, err := persistence.NewCacheStore(opts)
	assert.NoError(t, err)

	router := gin.New()
	router.Use(Cache(store))

	ping := "ping"
	router.POST("/cache_set", func(ctx *gin.Context) {
		store, exist := ctx.Get(CACHE_MIDDLEWARE_KEY)
		assert.ObjectsAreEqualValues(true, exist)
		cacheStore := store.(persistence.CacheStore)

		err := cacheStore.Set(ping, "pong", time.Minute)
		assert.NoError(t, err)
		ctx.String(200, "pong")
	})

	router.GET("/cache_get", func(ctx *gin.Context) {
		store, exist := ctx.Get(CACHE_MIDDLEWARE_KEY)
		assert.ObjectsAreEqualValues(true, exist)
		cacheStore := store.(persistence.CacheStore)

		var value string
		err := cacheStore.Get(ping, &value)
		assert.NoError(t, err)
		ctx.String(200, value)
	})

	w1 := performRequest("POST", "/cache_set", router)
	w2 := performRequest("GET", "/cache_get", router)

	assert.Equal(t, 200, w1.Code)
	assert.Equal(t, 200, w2.Code)
	assert.Equal(t, w1.Body.String(), w2.Body.String())
}

func TestWrite(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	opts := persistence.Options{
		Adapter:           persistence.AdapterInMemoryStore,
		DefaultExpiration: 60 * time.Second,
	}
	store, err := persistence.NewCacheStore(opts)
	assert.NoError(t, err)

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
	opts := persistence.Options{
		Adapter:           persistence.AdapterInMemoryStore,
		DefaultExpiration: 60 * time.Second,
	}
	store, err := persistence.NewCacheStore(opts)
	assert.NoError(t, err)

	router := gin.New()
	router.GET("/cache_ping", CachePage(store, time.Second*3, func(c *gin.Context) {
		c.String(200, "pong "+fmt.Sprint(time.Now().UnixNano()))
	}))

	w1 := performRequest("GET", "/cache_ping", router)
	w2 := performRequest("GET", "/cache_ping", router)

	assert.Equal(t, 200, w1.Code)
	assert.Equal(t, 200, w2.Code)
	assert.Equal(t, w1.Body.String(), w2.Body.String())
}

func TestCachePageExpire(t *testing.T) {
	opts := persistence.Options{
		Adapter:           persistence.AdapterInMemoryStore,
		DefaultExpiration: 60 * time.Second,
	}
	store, err := persistence.NewCacheStore(opts)
	assert.NoError(t, err)

	router := gin.New()
	router.GET("/cache_ping", CachePage(store, time.Second, func(c *gin.Context) {
		c.String(200, "pong "+fmt.Sprint(time.Now().UnixNano()))
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
	store, err := newDelayStore(60 * time.Second)
	assert.NoError(t, err)

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

func TestCachePageWithoutHeader(t *testing.T) {
	opts := persistence.Options{
		Adapter:           persistence.AdapterInMemoryStore,
		DefaultExpiration: 60 * time.Second,
	}
	store, err := persistence.NewCacheStore(opts)
	assert.NoError(t, err)

	router := gin.New()
	router.GET("/cache_ping", CachePageWithoutHeader(store, time.Second*3, func(c *gin.Context) {
		c.String(200, "pong "+fmt.Sprint(time.Now().UnixNano()))
	}))

	w1 := performRequest("GET", "/cache_ping", router)
	w2 := performRequest("GET", "/cache_ping", router)

	assert.Equal(t, 200, w1.Code)
	assert.Equal(t, 200, w2.Code)
	assert.NotNil(t, w1.Header()["Content-Type"])
	assert.Nil(t, w2.Header()["Content-Type"])
	assert.Equal(t, w1.Body.String(), w2.Body.String())
}

func TestCachePageWithoutHeaderExpire(t *testing.T) {
	opts := persistence.Options{
		Adapter:           persistence.AdapterInMemoryStore,
		DefaultExpiration: 60 * time.Second,
	}
	store, err := persistence.NewCacheStore(opts)
	assert.NoError(t, err)

	router := gin.New()
	router.GET("/cache_ping", CachePage(store, time.Second, func(c *gin.Context) {
		c.String(200, "pong "+fmt.Sprint(time.Now().UnixNano()))
	}))

	w1 := performRequest("GET", "/cache_ping", router)
	time.Sleep(time.Second * 2)
	w2 := performRequest("GET", "/cache_ping", router)

	assert.Equal(t, 200, w1.Code)
	assert.Equal(t, 200, w2.Code)
	assert.NotNil(t, w1.Header()["Content-Type"])
	assert.NotNil(t, w2.Header()["Content-Type"])
	assert.NotEqual(t, w1.Body.String(), w2.Body.String())
}

func TestCacheHtmlFile(t *testing.T) {
	opts := persistence.Options{
		Adapter:           persistence.AdapterInMemoryStore,
		DefaultExpiration: 60 * time.Second,
	}
	store, err := persistence.NewCacheStore(opts)
	assert.NoError(t, err)

	router := gin.New()
	router.LoadHTMLFiles("example/template.html")
	router.GET("/cache_html", CachePage(store, time.Second*3, func(c *gin.Context) {
		c.HTML(http.StatusOK, "template.html", gin.H{"values": fmt.Sprint(time.Now().UnixNano())})
	}))

	w1 := performRequest("GET", "/cache_html", router)
	w2 := performRequest("GET", "/cache_html", router)

	assert.Equal(t, 200, w1.Code)
	assert.Equal(t, 200, w2.Code)
	assert.Equal(t, w1.Body.String(), w2.Body.String())
}

func TestCacheHtmlFileExpire(t *testing.T) {
	opts := persistence.Options{
		Adapter:           persistence.AdapterInMemoryStore,
		DefaultExpiration: 60 * time.Second,
	}
	store, err := persistence.NewCacheStore(opts)
	assert.NoError(t, err)

	router := gin.New()
	router.LoadHTMLFiles("example/template.html")
	router.GET("/cache_html", CachePage(store, time.Second*1, func(c *gin.Context) {
		c.HTML(http.StatusOK, "template.html", gin.H{"values": fmt.Sprint(time.Now().UnixNano())})
	}))

	w1 := performRequest("GET", "/cache_html", router)
	time.Sleep(time.Second * 2)
	w2 := performRequest("GET", "/cache_html", router)

	assert.Equal(t, 200, w1.Code)
	assert.Equal(t, 200, w2.Code)
	assert.NotEqual(t, w1.Body.String(), w2.Body.String())
}

func TestCachePageAborted(t *testing.T) {
	opts := persistence.Options{
		Adapter:           persistence.AdapterInMemoryStore,
		DefaultExpiration: 60 * time.Second,
	}
	store, err := persistence.NewCacheStore(opts)
	assert.NoError(t, err)

	router := gin.New()
	router.GET("/cache_aborted", CachePage(store, time.Second*3, func(c *gin.Context) {
		c.AbortWithStatusJSON(200, map[string]int64{"time": time.Now().UnixNano()})
	}))

	w1 := performRequest("GET", "/cache_aborted", router)
	time.Sleep(time.Millisecond * 500)
	w2 := performRequest("GET", "/cache_aborted", router)

	assert.Equal(t, 200, w1.Code)
	assert.Equal(t, 200, w2.Code)
	assert.NotEqual(t, w1.Body.String(), w2.Body.String())
}

func TestCachePage400(t *testing.T) {
	opts := persistence.Options{
		Adapter:           persistence.AdapterInMemoryStore,
		DefaultExpiration: 60 * time.Second,
	}
	store, err := persistence.NewCacheStore(opts)
	assert.NoError(t, err)

	router := gin.New()
	router.GET("/cache_400", CachePage(store, time.Second*3, func(c *gin.Context) {
		c.String(400, fmt.Sprint(time.Now().UnixNano()))
	}))

	w1 := performRequest("GET", "/cache_400", router)
	time.Sleep(time.Millisecond * 500)
	w2 := performRequest("GET", "/cache_400", router)

	assert.Equal(t, 400, w1.Code)
	assert.Equal(t, 400, w2.Code)
	assert.NotEqual(t, w1.Body.String(), w2.Body.String())
}

func TestCachePageWithoutHeaderAborted(t *testing.T) {
	opts := persistence.Options{
		Adapter:           persistence.AdapterInMemoryStore,
		DefaultExpiration: 60 * time.Second,
	}
	store, err := persistence.NewCacheStore(opts)
	assert.NoError(t, err)

	router := gin.New()
	router.GET("/cache_aborted", CachePage(store, time.Second*3, func(c *gin.Context) {
		c.AbortWithStatusJSON(200, map[string]int64{"time": time.Now().UnixNano()})
	}))

	w1 := performRequest("GET", "/cache_aborted", router)
	time.Sleep(time.Millisecond * 500)
	w2 := performRequest("GET", "/cache_aborted", router)

	assert.Equal(t, 200, w1.Code)
	assert.Equal(t, 200, w2.Code)
	assert.NotNil(t, w1.Header()["Content-Type"])
	assert.NotNil(t, w2.Header()["Content-Type"])
	assert.NotEqual(t, w1.Body.String(), w2.Body.String())
}

func TestCachePageWithoutHeader400(t *testing.T) {
	opts := persistence.Options{
		Adapter:           persistence.AdapterInMemoryStore,
		DefaultExpiration: 60 * time.Second,
	}
	store, err := persistence.NewCacheStore(opts)
	assert.NoError(t, err)

	router := gin.New()
	router.GET("/cache_400", CachePage(store, time.Second*3, func(c *gin.Context) {
		c.String(400, fmt.Sprint(time.Now().UnixNano()))
	}))

	w1 := performRequest("GET", "/cache_400", router)
	time.Sleep(time.Millisecond * 500)
	w2 := performRequest("GET", "/cache_400", router)

	assert.Equal(t, 400, w1.Code)
	assert.Equal(t, 400, w2.Code)
	assert.NotNil(t, w1.Header()["Content-Type"])
	assert.NotNil(t, w2.Header()["Content-Type"])
	assert.NotEqual(t, w1.Body.String(), w2.Body.String())
}

func TestCachePageStatus207(t *testing.T) {
	opts := persistence.Options{
		Adapter:           persistence.AdapterInMemoryStore,
		DefaultExpiration: 60 * time.Second,
	}
	store, err := persistence.NewCacheStore(opts)
	assert.NoError(t, err)

	router := gin.New()
	router.GET("/cache_207", CachePage(store, time.Second*3, func(c *gin.Context) {
		c.String(207, fmt.Sprint(time.Now().UnixNano()))
	}))

	w1 := performRequest("GET", "/cache_207", router)
	time.Sleep(time.Millisecond * 500)
	w2 := performRequest("GET", "/cache_207", router)

	assert.Equal(t, 207, w1.Code)
	assert.Equal(t, 207, w2.Code)
	assert.Equal(t, w1.Body.String(), w2.Body.String())
}

func TestCachePageWithoutQuery(t *testing.T) {
	opts := persistence.Options{
		Adapter:           persistence.AdapterInMemoryStore,
		DefaultExpiration: 60 * time.Second,
	}
	store, err := persistence.NewCacheStore(opts)
	assert.NoError(t, err)

	router := gin.New()
	router.GET("/cache_without_query", CachePageWithoutQuery(store, time.Second*3, func(c *gin.Context) {
		c.String(200, "pong "+fmt.Sprint(time.Now().UnixNano()))
	}))

	w1 := performRequest("GET", "/cache_without_query?foo=1", router)
	w2 := performRequest("GET", "/cache_without_query?foo=2", router)

	assert.Equal(t, 200, w1.Code)
	assert.Equal(t, 200, w2.Code)
	assert.Equal(t, w1.Body.String(), w2.Body.String())
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
func performRequest(method, target string, router *gin.Engine) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, target, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	return w
}

type memoryDelayStore struct {
	persistence.CacheStore
}

func newDelayStore(defaultExpiration time.Duration) (*memoryDelayStore, error) {
	opts := persistence.Options{
		Adapter:           persistence.AdapterInMemoryStore,
		DefaultExpiration: defaultExpiration,
	}
	store, err := persistence.NewCacheStore(opts)
	if err != nil {
		return nil, err
	}
	v := memoryDelayStore{
		CacheStore: store,
	}
	return &v, nil
}

func (c *memoryDelayStore) Set(key string, value interface{}, expires time.Duration) error {
	time.Sleep(time.Millisecond * 3)
	return c.CacheStore.Set(key, value, expires)
}

func (c *memoryDelayStore) Add(key string, value interface{}, expires time.Duration) error {
	time.Sleep(time.Millisecond * 3)
	return c.CacheStore.Add(key, value, expires)
}
