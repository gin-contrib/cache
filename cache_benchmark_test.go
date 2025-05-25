package cache

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"io"
	"strings"
	"testing"
)

func BenchmarkGenerateCacheKey_Short(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()
	key := "/api/v1/resource?id=123"
	for i := 0; i < b.N; i++ {
		_ = generateCacheKey("prefix", key)
	}
}

func BenchmarkGenerateCacheKey_Long(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()
	longKey := "/api/v1/resource?" + generateLongQuery(1000)
	for i := 0; i < b.N; i++ {
		_ = generateCacheKey("prefix", longKey)
	}
}

func generateLongQuery(n int) string {
	var builder strings.Builder
	for i := 0; i < n; i++ {
		builder.WriteString(fmt.Sprintf("k%d=v%d&", i, i))
	}
	return builder.String()
}

func BenchmarkGenerateURLEscapeKey_Short(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()
	key := "/api/v1/resource?id=123"
	for i := 0; i < b.N; i++ {
		_ = urlEscape("prefix", key)
	}
}

func BenchmarkGenerateURLEscapeKey_Long(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()
	longKey := "/api/v1/resource?" + generateLongQuery(1000)
	for i := 0; i < b.N; i++ {
		_ = urlEscape("prefix", longKey)
	}
}

func urlEscape(prefix string, u string) string {
	h := sha1.New()
	_, _ = io.WriteString(h, u)
	key := string(h.Sum(nil))
	var buffer bytes.Buffer
	buffer.WriteString(prefix)
	buffer.WriteString(":")
	buffer.WriteString(key)
	return buffer.String()
}
