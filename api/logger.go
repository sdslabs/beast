package api

import (
	"bytes"
	"io"
	"os"

	"github.com/gin-gonic/gin"
	"google.golang.org/appengine/log"
	// plus the same `log` import your current logger file uses
)

func isPayloadLoggingEnabled() bool {
	return os.Getenv("LOG_HTTP_PAYLOADS") == "true"
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

func requestResponseLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if !isPayloadLoggingEnabled() {
			return
		}

		reqBody, _ := io.ReadAll(c.Request.Body)
		c.Request.Body = io.NopCloser(bytes.NewReader(reqBody))

		w := &bodyLogWriter{ResponseWriter: c.Writer}
		c.Writer = w
		c.Next()

		log.Infof("[HTTP] %s %s | Req: %s | Resp: %s", c.Request.Method, path, reqBody, w.body.String())
	}
}
