package persistence

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type redisTestContainer struct {
	Container testcontainers.Container
	Host      string
	Port      string
	URL       string
}

func setupRedisContainer(t *testing.T, imageTag string) *redisTestContainer {
	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        fmt.Sprintf("redis:%s", imageTag),
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForLog("Ready to accept connections"),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err, "failed to start redis container")

	host, err := container.Host(ctx)
	require.NoError(t, err, "failed to get redis container host")

	port, err := container.MappedPort(ctx, "6379")
	require.NoError(t, err, "failed to get redis container port")

	url := fmt.Sprintf("redis://%s:%s", host, port.Port())

	return &redisTestContainer{
		Container: container,
		Host:      fmt.Sprintf("%s:%s", host, port.Port()),
		Port:      port.Port(),
		URL:       url,
	}
}

func teardownRedisContainer(t *testing.T, c *redisTestContainer) {
	ctx := context.Background()
	err := c.Container.Terminate(ctx)
	require.NoError(t, err, "failed to terminate redis container")
}

func newRedisStore(t *testing.T, defaultExpiration time.Duration, imageTag string) CacheStore {
	c := setupRedisContainer(t, imageTag)
	t.Cleanup(func() { teardownRedisContainer(t, c) })

	redisCache := NewRedisCache(c.Host, "", defaultExpiration)
	if err := redisCache.Flush(); err != nil {
		t.Errorf("Error flushing cache: %v", err)
	}
	return redisCache
}

func newRedisStoreWithURL(t *testing.T, defaultExpiration time.Duration, imageTag string) CacheStore {
	c := setupRedisContainer(t, imageTag)
	t.Cleanup(func() { teardownRedisContainer(t, c) })

	redisCache := NewRedisCacheWithURL(c.URL, defaultExpiration)
	if err := redisCache.Flush(); err != nil {
		t.Errorf("Error flushing cache: %v", err)
	}
	return redisCache
}

func runCommonTests(t *testing.T, factory func(*testing.T, time.Duration, string) CacheStore, imageTag string) {
	t.Run("TypicalGetSet", func(t *testing.T) {
		typicalGetSet(t, func(t *testing.T, d time.Duration) CacheStore { return factory(t, d, imageTag) })
	})
	t.Run("IncrDecr", func(t *testing.T) {
		incrDecr(t, func(t *testing.T, d time.Duration) CacheStore { return factory(t, d, imageTag) })
	})
	t.Run("Expiration", func(t *testing.T) {
		expiration(t, func(t *testing.T, d time.Duration) CacheStore { return factory(t, d, imageTag) })
	})
	t.Run("EmptyCache", func(t *testing.T) {
		emptyCache(t, func(t *testing.T, d time.Duration) CacheStore { return factory(t, d, imageTag) })
	})
	t.Run("Replace", func(t *testing.T) {
		testReplace(t, func(t *testing.T, d time.Duration) CacheStore { return factory(t, d, imageTag) })
	})
	t.Run("Add", func(t *testing.T) {
		testAdd(t, func(t *testing.T, d time.Duration) CacheStore { return factory(t, d, imageTag) })
	})
}

func TestRedisCache(t *testing.T) {
	versions := []string{"8.0-alpine", "7.2-alpine", "6.2-alpine"}
	for _, version := range versions {
		t.Run(fmt.Sprintf("Standard_Redis_%s", version), func(t *testing.T) {
			runCommonTests(t, newRedisStore, version)
		})
		t.Run(fmt.Sprintf("WithURL_Redis_%s", version), func(t *testing.T) {
			runCommonTests(t, newRedisStoreWithURL, version)
		})
	}
}
