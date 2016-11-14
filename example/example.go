package main

import (
	"fmt"
	"time"

	"github.com/dpordomingo/go-gingonic-cache"
	"github.com/dpordomingo/go-gingonic-cache/persistence"
	"gopkg.in/gin-gonic/gin.v1"
)

func main() {
	r := gin.Default()

	store := persistence.NewInMemoryStore(time.Second)
	// Cached Page
	r.GET("/ping", func(c *gin.Context) {
		c.String(200, "pong "+fmt.Sprint(time.Now().Unix()))
	})

	r.GET("/cache_ping", cache.CachePage(store, time.Minute, func(c *gin.Context) {
		c.String(200, "pong "+fmt.Sprint(time.Now().Unix()))
	}))

	// Listen and Server in 0.0.0.0:8080
	r.Run(":8080")
}
