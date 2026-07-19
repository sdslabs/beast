package cache

import (
	"crypto/tls"
	"slices"
	"testing"
)

func TestRedisAddressSupportsIPv6(t *testing.T) {
	got := redisAddress(RedisConfig{Host: "2001:db8::1", Port: "6379"})
	if got != "[2001:db8::1]:6379" {
		t.Fatalf("redisAddress() = %q", got)
	}
}

func TestRedisCLIConnectionArgsIncludeTLS(t *testing.T) {
	previous := cacheConfig
	cacheConfig = RedisConfig{Host: "redis.example.com", Port: "6380", User: "beast", DB: 2, TLS: true, CAFile: "/ca.pem", ServerName: "redis.example.com"}
	defer func() { cacheConfig = previous }()

	arguments := redisCLIConnectionArgs()
	for _, expected := range []string{"--tls", "--cacert", "/ca.pem", "--sni", "redis.example.com"} {
		if !slices.Contains(arguments, expected) {
			t.Fatalf("missing %q from Redis CLI arguments: %v", expected, arguments)
		}
	}
}

func TestRedisTLSConfigRequiresTLS12(t *testing.T) {
	config, err := NewTLSConfig(true, "", "redis.example.com", "redis.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if config == nil || config.MinVersion != tls.VersionTLS12 || config.ServerName != "redis.example.com" {
		t.Fatalf("unexpected TLS config: %+v", config)
	}
}
