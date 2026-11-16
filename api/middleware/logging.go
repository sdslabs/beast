package middleware

import (
	"bytes"
	"io"
	"log"
	"os"

	"github.com/gin-gonic/gin"
)

// IsPayloadLoggingEnabled reports whether full HTTP request/response payloads should be logged.
func IsPayloadLoggingEnabled() bool {
	return os.Getenv("LOG_HTTP_PAYLOADS") == "true"
}

// LoggerSkipFunc determines whether gin.LoggerWithConfig should skip logging this request.
func LoggerSkipFunc(c *gin.Context) bool {
	return !IsPayloadLoggingEnabled() && c.Writer.Status() < 400
}

type bodyLogWriter struct {
	gin.ResponseWriter
	body bytes.Buffer
}

func (w *bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *bodyLogWriter) WriteString(s string) (int, error) {
	w.body.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}

// RequestResponseLogger intercepts and logs payload details when enabled.
func RequestResponseLogger(c *gin.Context) {
	path := c.Request.URL.Path
	if !IsPayloadLoggingEnabled() {
		return
	}

	reqBody, _ := io.ReadAll(c.Request.Body)
	c.Request.Body = io.NopCloser(bytes.NewReader(reqBody))

	w := &bodyLogWriter{ResponseWriter: c.Writer}
	c.Writer = w
	c.Next()

	log.Printf("[HTTP] %s %s | Req: %s | Resp: %s", c.Request.Method, path, reqBody, w.body.String())
}
