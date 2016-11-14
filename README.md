# Cache gin's middleware

[![Build Status](https://travis-ci.org/dpordomingo/go-gingonic-cache.svg)](https://travis-ci.org/dpordomingo/go-gingonic-cache)
[![codecov](https://codecov.io/gh/dpordomingo/go-gingonic-cache/branch/master/graph/badge.svg)](https://codecov.io/gh/dpordomingo/go-gingonic-cache)
[![Go Report Card](https://goreportcard.com/badge/github.com/dpordomingo/go-gingonic-cache)](https://goreportcard.com/report/github.com/dpordomingo/go-gingonic-cache)
[![GoDoc](https://godoc.org/github.com/dpordomingo/go-gingonic-cache?status.svg)](https://godoc.org/github.com/dpordomingo/go-gingonic-cache)


Gin middleware/handler to enable Cache.

## Usage

### Start using it

Download and install it:

```sh
$ go get github.com/dpordomingo/go-gingonic-cache
```

Import it in your code:

```go
import "github.com/dpordomingo/go-gingonic-cache"
```

### Canonical example:

```go
package main

import (
	"time"

	"github.com/dpordomingo/go-gingonic-cache"
	"gopkg.in/gin-gonic/gin.v1"
)

func main() {
	router := gin.Default()

	store := cache.NewInMemoryStore(time.Second)
	// Cached Page
	router.GET("/ping", func(c *gin.Context) {
		c.String(200, "pong "+fmt.Sprint(time.Now().Unix()))
	})

	router.GET("/cache_ping", cache.CachePage(store, time.Minute, func(c *gin.Context) {
		c.String(200, "pong "+fmt.Sprint(time.Now().Unix()))
	}))

	// Listen and Server in 0.0.0.0:8080
	router.Run(":8080")
}
```
