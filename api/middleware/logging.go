package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/core/config"
)

const (
	defaultPayloadLimit = 4 * 1024
	hardPayloadLimit    = 64 * 1024
)

// JSON logs must be valid JSON lines, with no standard logger prefix.
var jsonLogger = log.New(os.Stdout, "", 0)

func getLoggingConfig() config.LoggingConfig {
	if config.IsConfigured() {
		return config.Cfg.Logging
	}
	return config.DefaultConfig().Logging
}

func IsPayloadLoggingEnabled() bool {
	payloadEnv := os.Getenv("BEAST_LOG_HTTP_PAYLOADS")
	if payloadEnv == "" {
		payloadEnv = os.Getenv("LOG_HTTP_PAYLOADS")
	}
	if payloadEnv != "" {
		if b, err := strconv.ParseBool(payloadEnv); err == nil {
			return b
		}
	}
	return getLoggingConfig().LogHTTPPayloads
}

// When payload logging is on, the custom logger handles every request.
// Otherwise, Gin logs errors only.
func LoggerSkipFunc(c *gin.Context) bool {
	return IsPayloadLoggingEnabled() || c.Writer.Status() < http.StatusBadRequest
}

func isRedactedPath(path string, prefixes []string) bool {
	for _, p := range prefixes {
		if p != "" && strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

func isUnbufferedPath(path string, suffixes []string) bool {
	for _, s := range suffixes {
		if s != "" && strings.HasSuffix(path, s) {
			return true
		}
	}
	return false
}

func payloadLimit(configured int) int {
	if configured <= 0 {
		return defaultPayloadLimit
	}
	if configured > hardPayloadLimit {
		return hardPayloadLimit
	}
	return configured
}

// boundedCapture stores at most limit bytes, recording whether data was lost.
type boundedCapture struct {
	buffer    bytes.Buffer
	limit     int
	truncated bool
}

func (b *boundedCapture) appendBytes(p []byte) {
	remaining := b.limit - b.buffer.Len()
	if len(p) > remaining {
		b.truncated = true
		if remaining <= 0 {
			return
		}
		p = p[:remaining]
	}
	if len(p) != 0 {
		_, _ = b.buffer.Write(p)
	}
}

func (b *boundedCapture) appendString(s string) {
	remaining := b.limit - b.buffer.Len()
	if len(s) > remaining {
		b.truncated = true
		if remaining <= 0 {
			return
		}
		s = s[:remaining]
	}
	if len(s) != 0 {
		_, _ = b.buffer.WriteString(s)
	}
}

func (b *boundedCapture) String() string {
	if b.truncated {
		return b.buffer.String() + "...[truncated]"
	}
	return b.buffer.String()
}

// Capture happens only as the handler reads the request body.
type loggingReadCloser struct {
	io.ReadCloser
	capture *boundedCapture
}

func (r *loggingReadCloser) Read(p []byte) (int, error) {
	n, err := r.ReadCloser.Read(p)
	if n > 0 {
		r.capture.appendBytes(p[:n])
	}
	return n, err
}

// Embedding gin.ResponseWriter preserves Gin's Flush, Hijack, etc.
type bodyLogWriter struct {
	gin.ResponseWriter
	capture *boundedCapture
}

func (w *bodyLogWriter) Write(p []byte) (int, error) {
	n, err := w.ResponseWriter.Write(p)
	if n > 0 {
		w.capture.appendBytes(p[:n])
	}
	return n, err
}

func (w *bodyLogWriter) WriteString(s string) (int, error) {
	n, err := w.ResponseWriter.WriteString(s)
	if n > 0 {
		w.capture.appendString(s[:n])
	}
	return n, err
}

func isTextContentType(raw string) bool {
	mediaType, _, err := mime.ParseMediaType(raw)
	if err != nil {
		return false
	}
	mediaType = strings.ToLower(mediaType)
	return strings.HasPrefix(mediaType, "text/") ||
		strings.HasSuffix(mediaType, "+json") ||
		strings.HasSuffix(mediaType, "+xml") ||
		mediaType == "application/json" ||
		mediaType == "application/xml" ||
		mediaType == "application/x-www-form-urlencoded" ||
		mediaType == "application/javascript" ||
		mediaType == "application/graphql"
}

func isIdentityEncoding(raw string) bool {
	return raw == "" || strings.EqualFold(strings.TrimSpace(raw), "identity")
}

func responsePayload(w gin.ResponseWriter, capture *boundedCapture) string {
	if capture.buffer.Len() == 0 && !capture.truncated {
		return ""
	}
	if !isIdentityEncoding(w.Header().Get("Content-Encoding")) {
		return "[skipped: encoded response]"
	}
	contentType := w.Header().Get("Content-Type")
	if contentType == "" {
		// net/http may infer a Content-Type without updating Header().
		contentType = http.DetectContentType(capture.buffer.Bytes())
	}
	if !isTextContentType(contentType) {
		return "[skipped: non-text response]"
	}
	return capture.String()
}

type httpLogEntry struct {
	Time      string `json:"time"`
	Level     string `json:"level"`
	Method    string `json:"method"`
	Path      string `json:"path"`
	Status    int    `json:"status"`
	LatencyMS int64  `json:"latency_ms"`
	Request   string `json:"request"`
	Response  string `json:"response"`
	RequestID string `json:"request_id,omitempty"`
}

// RequestResponseLogger logs bounded, textual request/response payloads.
func RequestResponseLogger(c *gin.Context) {
	if !IsPayloadLoggingEnabled() {
		c.Next()
		return
	}

	cfg := getLoggingConfig()
	limit := payloadLimit(cfg.MaxPayloadBytes)
	path := c.Request.URL.Path
	start := time.Now()

	redacted := isRedactedPath(path, cfg.RedactedPathPrefixes)
	unbuffered := isUnbufferedPath(path, cfg.UnbufferedPathSuffixes) ||
		strings.Contains(c.GetHeader("Accept"), "text/event-stream") ||
		strings.EqualFold(c.GetHeader("Upgrade"), "websocket")

	var requestCapture *boundedCapture
	requestText := ""

	switch {
	case redacted:
		requestText = "[redacted]"
	case c.Request.Body == nil || c.Request.Body == http.NoBody:
		// No body to capture.
	case !isIdentityEncoding(c.Request.Header.Get("Content-Encoding")) ||
		!isTextContentType(c.Request.Header.Get("Content-Type")):
		requestText = "[skipped: non-text or encoded request]"
	default:
		requestCapture = &boundedCapture{limit: limit}
		c.Request.Body = &loggingReadCloser{
			ReadCloser: c.Request.Body,
			capture:    requestCapture,
		}
	}

	var responseCapture *boundedCapture
	if !redacted && !unbuffered {
		responseCapture = &boundedCapture{limit: limit}
		c.Writer = &bodyLogWriter{
			ResponseWriter: c.Writer,
			capture:        responseCapture,
		}
	}

	c.Next()

	if requestCapture != nil {
		requestText = requestCapture.String()
	}

	responseText := ""
	switch {
	case redacted:
		responseText = "[redacted]"
	case unbuffered:
		responseText = "[unbuffered]"
	case responseCapture != nil:
		responseText = responsePayload(c.Writer, responseCapture)
	}

	latency := time.Since(start)
	status := c.Writer.Status()

	if strings.EqualFold(cfg.Format, "json") {
		entry := httpLogEntry{
			Time:      time.Now().UTC().Format(time.RFC3339Nano),
			Level:     cfg.Level,
			Method:    c.Request.Method,
			Path:      path,
			Status:    status,
			LatencyMS: latency.Milliseconds(),
			Request:   requestText,
			Response:  responseText,
			RequestID: c.GetString("request_id"),
		}
		if data, err := json.Marshal(entry); err == nil {
			jsonLogger.Print(string(data))
			return
		}
	}

	// %q prevents payload newlines and control characters from forging log lines.
	log.Printf("[HTTP] %s %q %d %s | Req: %q | Resp: %q",
		c.Request.Method, path, status, latency.Round(time.Millisecond),
		requestText, responseText)
}
