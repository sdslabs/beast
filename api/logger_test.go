package api

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

func TestRequestResponseLoggerPayloads(t *testing.T) {
	t.Setenv("LOG_HTTP_PAYLOADS", "true")

	var output bytes.Buffer
	logger := log.StandardLogger()
	previousOutput := logger.Out
	logger.SetOutput(&output)
	t.Cleanup(func() { logger.SetOutput(previousOutput) })

	router := gin.New()
	router.Use(requestResponseLogger())
	echo := func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
		}
		c.String(http.StatusBadRequest, "%s", body)
	}
	router.POST("/ordinary", echo)
	router.POST("/auth/login", echo)

	request := func(path, payload string) {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(payload))
		req.Header.Set("Content-Type", "text/plain")
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)
		if recorder.Code != http.StatusBadRequest || recorder.Body.String() != payload {
			t.Fatalf("unexpected response status %d or altered request body", recorder.Code)
		}
	}

	t.Run("caps body previews and omits query values", func(t *testing.T) {
		output.Reset()
		request("/ordinary?item=private-query", "ordinary-preview "+strings.Repeat("x", maxLoggedPayloadBytes)+"private-tail")
		if !strings.Contains(output.String(), "ordinary-preview") || !strings.Contains(output.String(), "[truncated]") {
			t.Fatal("expected a truncated payload preview")
		}
		if strings.Contains(output.String(), "private-query") || strings.Contains(output.String(), "private-tail") {
			t.Fatal("logger included query values or data beyond the preview limit")
		}
	})

	t.Run("omits sensitive payloads", func(t *testing.T) {
		output.Reset()
		request("/auth/login", "private-credential")
		if !strings.Contains(output.String(), "Status: 400") || strings.Contains(output.String(), "private-credential") {
			t.Fatal("expected error metadata and no credential material")
		}
	})

	t.Run("honors disabled logging on errors", func(t *testing.T) {
		t.Setenv("LOG_HTTP_PAYLOADS", "false")
		output.Reset()
		request("/ordinary", "private-disabled")
		if !strings.Contains(output.String(), "Status: 400") || strings.Contains(output.String(), "private-disabled") {
			t.Fatal("expected error metadata and no disabled payload preview")
		}
	})
}
