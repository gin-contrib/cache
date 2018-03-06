package cache

import (
	"fmt"
	"log"
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
		c.String(200, "pong "+fmt.Sprint(time.Now().UnixNano()))
	}))

	w1 := performRequest("GET", "/cache_ping", router)
	time.Sleep(10 * time.Millisecond)
	w2 := performRequest("GET", "/cache_ping", router)

	assert.Equal(t, 200, w1.Code)
	assert.Equal(t, 200, w2.Code)
	assert.Equal(t, w1.Body.String(), w2.Body.String())
}

func TestCachePageError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := persistence.NewInMemoryStore(60 * time.Second)

	router := gin.New()
	router.GET("/cache_ping", CachePage(store, time.Second*5, func(c *gin.Context) {
		now := "pong "+fmt.Sprint(time.Now().UnixNano())
		log.Println(now)
		c.String(500, now)
	}))

	w1 := performRequest("GET", "/cache_ping", router)
	time.Sleep(10 * time.Millisecond)
	w2 := performRequest("GET", "/cache_ping", router)

	assert.Equal(t, 500, w1.Code)
	assert.Equal(t, 500, w2.Code)
	assert.NotEqual(t, w1.Body.String(), w2.Body.String())
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

func TestCacheHtmlFile(t *testing.T) {
	store := persistence.NewInMemoryStore(60 * time.Second)

	router := gin.New()
	router.LoadHTMLFiles("example/template.html")
	router.GET("/cache_html", CachePage(store, time.Second*3, func(c *gin.Context) {
		c.HTML(
      .StatusOK, "template.html", gin.H{"values": fmt.Sprint(time.Now().Unix())})
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
