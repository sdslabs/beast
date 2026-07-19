package cache

import (
	"testing"
)

func TestRedisAddressSupportsIPv6(t *testing.T) {
	got := redisAddress(RedisConfig{Host: "2001:db8::1", Port: "6379"})
	if got != "[2001:db8::1]:6379" {
		t.Fatalf("redisAddress() = %q", got)
	}
}
