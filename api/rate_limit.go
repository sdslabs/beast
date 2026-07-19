package api

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/core/cache"
)

const loginRateWindow = 5 * time.Minute

const incrementWithExpiryScript = `
local count = redis.call("INCR", KEYS[1])
if count == 1 then
  redis.call("PEXPIRE", KEYS[1], ARGV[1])
end
return count
`

func rateLimitKey(kind, value string) string {
	digest := sha256.Sum256([]byte(kind + "\x00" + value))
	return fmt.Sprintf("beast:rate:%s:%x", kind, digest)
}

func incrementRateLimit(ctx context.Context, key string, window time.Duration) (int64, error) {
	if cache.Cache == nil {
		return 0, fmt.Errorf("cache is unavailable")
	}
	return cache.Cache.Eval(ctx, incrementWithExpiryScript, []string{key}, window.Milliseconds()).Int64()
}

func enforceLoginRateLimit(c *gin.Context, username string) bool {
	host := requestPeerHost(c)
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	usernameCount, err := incrementRateLimit(ctx, rateLimitKey("login-user", username), loginRateWindow)
	if err == nil {
		var addressCount int64
		addressCount, err = incrementRateLimit(ctx, rateLimitKey("login-address", host), loginRateWindow)
		if err == nil && usernameCount <= 10 && addressCount <= 50 {
			return true
		}
	}
	if err != nil {
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, HTTPErrorResp{Error: "Authentication service unavailable"})
		return false
	}
	c.Header("Retry-After", "300")
	c.AbortWithStatusJSON(http.StatusTooManyRequests, HTTPErrorResp{Error: "Too many login attempts"})
	return false
}

func enforceOTPSendRateLimit(c *gin.Context) bool {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()
	count, err := incrementRateLimit(ctx, rateLimitKey("otp-address", requestPeerHost(c)), 10*time.Minute)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, HTTPErrorResp{Error: "OTP service unavailable"})
		return false
	}
	if count > 10 {
		c.Header("Retry-After", "600")
		c.AbortWithStatusJSON(http.StatusTooManyRequests, HTTPErrorResp{Error: "Too many OTP requests"})
		return false
	}
	return true
}

func requestPeerHost(c *gin.Context) string {
	host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		return c.Request.RemoteAddr
	}
	return host
}
