# Cache gin's middleware

[![Build Status](https://travis-ci.org/gin-contrib/cache.svg)](https://travis-ci.org/gin-contrib/cache)
[![codecov](https://codecov.io/gh/gin-contrib/cache/branch/master/graph/badge.svg)](https://codecov.io/gh/gin-contrib/cache)
[![Go Report Card](https://goreportcard.com/badge/github.com/gin-contrib/cache)](https://goreportcard.com/report/github.com/gin-contrib/cache)
[![GoDoc](https://godoc.org/github.com/gin-contrib/cache?status.svg)](https://godoc.org/github.com/gin-contrib/cache)

Gin middleware/handler to enable Cache.

## Usage

### Start using it

Download and install it:

```sh
$ go get github.com/gin-contrib/cache
```

Import it in your code:

```go
import "github.com/gin-contrib/cache"
```

### Canonical example:

See the [example](example/example.go)

```go
package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-contrib/cache"
	"github.com/gin-contrib/cache/persistence"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	opts := persistence.Options{
		Adapter:           persistence.AdapterInMemoryStore,
		DefaultExpiration: 60 * time.Second,
	}
	store, err := persistence.NewCacheStore(opts)
	if err != nil {
		log.Fatalln(err)
	}

	// Use cache.Cache middleware
	r.Use(cache.Cache(store))

	ping := "ping"

	// Store data to cache store
	r.GET("/cache_set", func(ctx *gin.Context) {
		store, exist := ctx.Get(cache.CACHE_MIDDLEWARE_KEY)
		if !exist {
			ctx.String(http.StatusInternalServerError, "cache middleware not found")
			return
		}

		cacheStore := store.(persistence.CacheStore)
		if err := cacheStore.Set(ping, "pong", time.Minute); err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}

		ctx.String(http.StatusOK, "value set to cache `%s: pong`\n", ping)
	})

	// Read data from cache store
	r.GET("/cache_get", func(ctx *gin.Context) {
		store, exist := ctx.Get(cache.CACHE_MIDDLEWARE_KEY)
		if !exist {
			ctx.String(http.StatusInternalServerError, "cache middleware not found")
			return
		}
		cacheStore := store.(persistence.CacheStore)

		var value string
		if err := cacheStore.Get(ping, &value); err != nil {
			ctx.String(http.StatusNotFound, err.Error())
			return
		}
		ctx.String(http.StatusOK, "value read from cache `%s: %s`\n", ping, value)
	})

	// Listen and Server in 0.0.0.0:8080
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
```
