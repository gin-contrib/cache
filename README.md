# Cache middleware

[![Build Status](https://github.com/gin-contrib/cache/actions/workflows/testing.yml/badge.svg)](https://github.com/gin-contrib/cache/actions/workflows/testing.yml)
[![Trivy Security Scan](https://github.com/gin-contrib/cache/actions/workflows/trivy-scan.yml/badge.svg)](https://github.com/gin-contrib/cache/actions/workflows/trivy-scan.yml)
[![codecov](https://codecov.io/gh/gin-contrib/cache/branch/master/graph/badge.svg)](https://codecov.io/gh/gin-contrib/cache)
[![Go Report Card](https://goreportcard.com/badge/github.com/gin-contrib/cache)](https://goreportcard.com/report/github.com/gin-contrib/cache)
[![GoDoc](https://godoc.org/github.com/gin-contrib/cache?status.svg)](https://godoc.org/github.com/gin-contrib/cache)

Gin middleware/handler to enable Cache.

- [Cache middleware](#cache-middleware)
  - [Usage](#usage)
    - [Start using it](#start-using-it)
    - [InMemory Example](#inmemory-example)
    - [Redis Example](#redis-example)

## Usage

### Start using it

Download and install it:

```sh
go get github.com/gin-contrib/cache
```

Import it in your code:

```go
import "github.com/gin-contrib/cache"
```

### InMemory Example

See the [example](_example/example.go)

```go
package main

import (
  "fmt"
  "time"

  "github.com/gin-contrib/cache"
  "github.com/gin-contrib/cache/persistence"
  "github.com/gin-gonic/gin"
)

func main() {
  r := gin.Default()

  store := persistence.NewInMemoryStore(time.Second)

  r.GET("/ping", func(c *gin.Context) {
    c.String(200, "pong "+fmt.Sprint(time.Now().Unix()))
  })
  // Cached Page
  r.GET("/cache_ping", cache.CachePage(store, time.Minute, func(c *gin.Context) {
    c.String(200, "pong "+fmt.Sprint(time.Now().Unix()))
  }))

  // Listen and Server in 0.0.0.0:8080
  r.Run(":8080")
}
```

You can also use the `Delete` and `Flush` methods with the InMemory store:

```go
// Delete a specific cache entry by key
err := store.Delete("your-cache-key")
if err != nil {
  // handle error
}

// Flush all cache entries
err = store.Flush()
if err != nil {
  // handle error
}
```

### Redis Example

Here is a complete example using Redis as the cache backend with `NewRedisCacheWithURL`:

```go
package main

import (
  "fmt"
  "time"

  "github.com/gin-contrib/cache"
  "github.com/gin-contrib/cache/persistence"
  "github.com/gin-gonic/gin"
)

func main() {
  r := gin.Default()

  // Basic usage:
  store := persistence.NewRedisCacheWithURL("redis://localhost:6379", time.Minute)

  // Advanced configuration with password and DB number:
  // store := persistence.NewRedisCacheWithURL("redis://:password@localhost:6379/0", time.Minute)

  r.GET("/ping", func(c *gin.Context) {
    c.String(200, "pong "+fmt.Sprint(time.Now().Unix()))
  })
  // Cached Page
  r.GET("/cache_ping", cache.CachePage(store, time.Minute, func(c *gin.Context) {
    c.String(200, "pong "+fmt.Sprint(time.Now().Unix()))
  }))

  // Listen and serve on 0.0.0.0:8080
  r.Run(":8080")
}
```
