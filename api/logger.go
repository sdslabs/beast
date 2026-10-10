package api

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/core"
	log "github.com/sirupsen/logrus"
)

type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w bodyLogWriter) WriteString(s string) (int, error) {
	w.body.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}

// isPayloadLoggingEnabled returns true if HTTP payload logging is enabled.
// If LOG_HTTP_PAYLOADS is explicitly set to false/0/no in the environment or .env,
// it returns false (even if DEBUG mode is enabled).
func isPayloadLoggingEnabled() bool {
	// 1. Check explicit OS environment variable first
	if envVal := os.Getenv("LOG_HTTP_PAYLOADS"); envVal != "" {
		val := strings.ToLower(strings.TrimSpace(envVal))
		if val == "false" || val == "0" || val == "no" {
			return false
		}
		if val == "true" || val == "1" || val == "yes" {
			return true
		}
	}

	// 2. Check .env files (.env in working directory and ~/.beast/.env)
	paths := []string{
		".env",
		filepath.Join(core.BEAST_GLOBAL_DIR, ".env"),
	}
	for _, p := range paths {
		if data, err := os.ReadFile(p); err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "#") {
					continue
				}
				if strings.HasPrefix(line, "LOG_HTTP_PAYLOADS=") {
					parts := strings.SplitN(line, "=", 2)
					if len(parts) == 2 {
						v := strings.ToLower(strings.Trim(strings.TrimSpace(parts[1]), "\"'"))
						if v == "false" || v == "0" || v == "no" {
							return false
						}
						if v == "true" || v == "1" || v == "yes" {
							return true
						}
					}
				}
			}
		}
	}

	// 3. Fallback to DEBUG mode only if LOG_HTTP_PAYLOADS was not specified anywhere
	return os.Getenv("DEBUG") != ""
}

// ginLoggerMiddleware returns a Gin logging middleware that ignores OPTIONS preflight requests.
func ginLoggerMiddleware() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		if param.Request.Method == "OPTIONS" {
			return ""
		}
		var statusColor, methodColor, resetColor string
		if param.IsOutputColor() {
			statusColor = param.StatusCodeColor()
			methodColor = param.MethodColor()
			resetColor = param.ResetColor()
		}
		return fmt.Sprintf("[GIN] %v |%s %3d %s| %13v | %15s |%s %-7s %s %#v\n%s",
			param.TimeStamp.Format("2006/01/02 - 15:04:05"),
			statusColor, param.StatusCode, resetColor,
			param.Latency,
			param.ClientIP,
			methodColor, param.Method, resetColor,
			param.Path,
			param.ErrorMessage,
		)
	})
}

// requestResponseLogger logs detailed information about incoming HTTP requests and responses,
// including method, path, client IP, status, latency, request body, and response body.
// Note: CORS preflight (OPTIONS) requests are ignored to avoid log noise.
func requestResponseLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !isPayloadLoggingEnabled() || c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}

		startTime := time.Now()
		path := c.Request.URL.Path
		rawQuery := c.Request.URL.RawQuery
		if rawQuery != "" {
			path = path + "?" + rawQuery
		}

		contentType := c.GetHeader("Content-Type")
		isMultipart := strings.HasPrefix(contentType, "multipart/")

		var reqBodyBytes []byte
		if !isMultipart && c.Request.Body != nil {
			var err error
			reqBodyBytes, err = io.ReadAll(c.Request.Body)
			if err == nil {
				c.Request.Body = io.NopCloser(bytes.NewBuffer(reqBodyBytes))
			}
		}

		reqBodyStr := "<empty>"
		if isMultipart {
			reqBodyStr = "<multipart form data>"
		} else if len(reqBodyBytes) > 0 {
			reqBodyStr = string(reqBodyBytes)
			if len(reqBodyStr) > 4096 {
				reqBodyStr = reqBodyStr[:4096] + " ...[truncated]"
			}
		}

		log.Infof("[HTTP REQ] %s %s | ClientIP: %s | Content-Type: %s | Body: %s",
			c.Request.Method,
			path,
			c.ClientIP(),
			contentType,
			reqBodyStr,
		)

		blw := &bodyLogWriter{
			body:           bytes.NewBufferString(""),
			ResponseWriter: c.Writer,
		}
		c.Writer = blw

		c.Next()

		duration := time.Since(startTime)
		statusCode := c.Writer.Status()
		respContentType := c.Writer.Header().Get("Content-Type")

		respBodyStr := "<empty>"
		if strings.HasPrefix(respContentType, "image/") ||
			strings.HasPrefix(respContentType, "application/octet-stream") ||
			strings.HasPrefix(respContentType, "application/zip") {
			respBodyStr = "<binary content>"
		} else if blw.body.Len() > 0 {
			respBodyStr = blw.body.String()
			if len(respBodyStr) > 4096 {
				respBodyStr = respBodyStr[:4096] + " ...[truncated]"
			}
		}

		log.Infof("[HTTP RESP] %s %s | Status: %d | Latency: %v | Content-Type: %s | Body: %s\n",
			c.Request.Method,
			path,
			statusCode,
			duration,
			respContentType,
			respBodyStr,
		)
	}
}
