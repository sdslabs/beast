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
	coreUtils "github.com/sdslabs/beastv4/core/utils"
	log "github.com/sirupsen/logrus"
)

const (
	ansiReset  = "\033[0m"
	ansiDim    = "\033[2m\033[90m" // Faint + dark gray
	ansiYellow = "\033[33m"        // Yellow for 4xx errors
	ansiRed    = "\033[31m"        // Red for 5xx errors
)

const maxLoggedPayloadBytes = 4096

type payloadPreview struct {
	bytes.Buffer
	truncated bool
}

func (preview *payloadPreview) Write(data []byte) (int, error) {
	remaining := maxLoggedPayloadBytes - preview.Len()
	if len(data) > remaining {
		preview.truncated = true
	}
	_, _ = preview.Buffer.Write(data[:min(len(data), remaining)])
	return len(data), nil
}

func (preview *payloadPreview) String() string {
	if preview.truncated {
		return preview.Buffer.String() + " ...[truncated]"
	}
	if preview.Len() == 0 {
		return "<empty>"
	}
	return preview.Buffer.String()
}

type requestPreviewReader struct {
	io.Reader
	io.Closer
}

type bodyLogWriter struct {
	gin.ResponseWriter
	body *payloadPreview
}

func (w bodyLogWriter) Write(b []byte) (int, error) {
	if isTextPayload(w.Header().Get("Content-Type")) {
		_, _ = w.body.Write(b)
	}
	return w.ResponseWriter.Write(b)
}

func (w bodyLogWriter) WriteString(s string) (int, error) {
	if isTextPayload(w.Header().Get("Content-Type")) {
		remaining := maxLoggedPayloadBytes - w.body.Len()
		if len(s) > remaining {
			w.body.truncated = true
		}
		_, _ = w.body.Buffer.WriteString(s[:min(len(s), remaining)])
	}
	return w.ResponseWriter.WriteString(s)
}

func isTextPayload(contentType string) bool {
	mediaType, _, _ := strings.Cut(contentType, ";")
	return mediaType == "application/json" || mediaType == "application/x-www-form-urlencoded" || mediaType == "text/plain"
}

func isSensitivePayloadPath(path string) bool {
	for _, prefix := range []string{"/auth", "/api/check", "/api/submit", "/api/config", "/api/instances", "/api/admin", "/api/manage", "/api/remote", "/api/notification"} {
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return true
		}
	}
	return false
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

// ginLoggerMiddleware returns a Gin logging middleware that dims OPTIONS requests.
func ginLoggerMiddleware() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		if param.Request.Method == "OPTIONS" {
			if param.StatusCode >= 400 {
				return fmt.Sprintf("%s[GIN] %v | %3d | %13v | %15s | %-7s %#v%s\n%s",
					ansiYellow,
					param.TimeStamp.Format("2006/01/02 - 15:04:05"),
					param.StatusCode,
					param.Latency,
					param.ClientIP,
					param.Method,
					param.Request.URL.Path,
					ansiReset,
					param.ErrorMessage,
				)
			}
			// Dim the OPTIONS preflight request in faint gray
			return fmt.Sprintf("%s[GIN] %v | %3d | %13v | %15s | %-7s %#v%s\n%s",
				ansiDim,
				param.TimeStamp.Format("2006/01/02 - 15:04:05"),
				param.StatusCode,
				param.Latency,
				param.ClientIP,
				param.Method,
				param.Request.URL.Path,
				ansiReset,
				param.ErrorMessage,
			)
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
			param.Request.URL.Path,
			param.ErrorMessage,
		)
	})
}

// requestResponseLogger records errors and optional bounded previews of text payloads.
func requestResponseLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()
		path := c.Request.URL.Path
		contentType := c.GetHeader("Content-Type")
		payloadsEnabled := isPayloadLoggingEnabled()
		capture := payloadsEnabled && !isSensitivePayloadPath(path)

		var requestBody, responseBody payloadPreview
		if capture {
			if c.Request.Body != nil && isTextPayload(contentType) {
				body := c.Request.Body
				c.Request.Body = requestPreviewReader{Reader: io.TeeReader(body, &requestBody), Closer: body}
			}
			c.Writer = &bodyLogWriter{ResponseWriter: c.Writer, body: &responseBody}
		}

		c.Next()

		status := c.Writer.Status()
		if !payloadsEnabled && status < 400 {
			return
		}
		user := ""
		if authHeader := c.GetHeader("Authorization"); authHeader != "" {
			if username, err := coreUtils.GetUser(authHeader); err == nil && username != "" {
				user = fmt.Sprintf(" | User: %q", username)
			}
		}

		if capture {
			log.Infof("[HTTP REQ] %s %s%s | ClientIP: %s | Content-Type: %s | Body: %q",
				c.Request.Method, path, user, c.ClientIP(), contentType, requestBody.String())
		}

		level := log.InfoLevel
		color := ""
		switch {
		case status >= 500:
			level, color = log.ErrorLevel, ansiRed
		case status >= 400:
			level, color = log.WarnLevel, ansiYellow
		case c.Request.Method == "OPTIONS":
			color = ansiDim
		}
		payload := ""
		if capture {
			payload = fmt.Sprintf(" | Body: %q", responseBody.String())
		}
		reset := ""
		if color != "" {
			reset = ansiReset
		}
		log.StandardLogger().Logf(level, "%s[HTTP RESP] %s %s%s | Status: %d | Latency: %v | Content-Type: %s%s%s",
			color, c.Request.Method, path, user, status, time.Since(startTime), c.Writer.Header().Get("Content-Type"), payload, reset)
	}
}
