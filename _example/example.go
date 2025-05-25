package main

import (
	"embed"
	"fmt"
	"html/template"
	"time"

	"github.com/gin-contrib/cache"
	"github.com/gin-contrib/cache/persistence"
	"github.com/gin-gonic/gin"
)

//go:embed template.html
var tmplFS embed.FS

func main() {
	r := gin.Default()

	// Parse embedded template
	tmpl, err := template.ParseFS(tmplFS, "template.html")
	if err != nil {
		panic(err)
	}
	r.SetHTMLTemplate(tmpl)

	store := persistence.NewInMemoryStore(60 * time.Second)
	// Non-cached Page
	r.GET("/ping", func(c *gin.Context) {
		c.String(200, "pong "+fmt.Sprint(time.Now().Unix()))
	})

	// Cached Page
	r.GET("/cache_ping", cache.CachePage(store, time.Minute, func(c *gin.Context) {
		c.String(200, "pong "+fmt.Sprint(time.Now().Unix()))
	}))

	// HTML Page using embedded template
	r.GET("/html", cache.CachePage(store, time.Minute, func(c *gin.Context) {
		name := c.DefaultQuery("name", "guest")
		c.HTML(200, "template.html", gin.H{
			"title":     "Cached HTML Page",
			"values":    "Hello from embed (" + name + ")!",
			"timestamp": time.Now().Format(time.RFC3339),
		})
	}))

	// Listen and Server in 0.0.0.0:8080
	_ = r.Run(":8080")
}
