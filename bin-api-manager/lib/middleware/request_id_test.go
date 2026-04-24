package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequestIDGeneratesID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID())

	var seen string
	r.GET("/", func(c *gin.Context) {
		seen = RequestIDFromContext(c)
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w, req)

	if seen == "" {
		t.Fatal("RequestIDFromContext returned empty")
	}
	if !strings.HasPrefix(seen, "req_") {
		t.Errorf("id should start with req_ prefix, got %q", seen)
	}
	if got := w.Header().Get("X-Request-Id"); got != seen {
		t.Errorf("response header X-Request-Id = %q want %q", got, seen)
	}
}

func TestRequestIDEchoesInboundHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID())

	var seen string
	r.GET("/", func(c *gin.Context) {
		seen = RequestIDFromContext(c)
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-Id", "req_client_supplied")
	r.ServeHTTP(w, req)

	if seen != "req_client_supplied" {
		t.Errorf("did not echo inbound header, got %q", seen)
	}
	if got := w.Header().Get("X-Request-Id"); got != "req_client_supplied" {
		t.Errorf("response header should echo, got %q", got)
	}
}

func TestRequestIDFromContextEmptyWhenAbsent(t *testing.T) {
	ctx := context.Background()
	if got := requestIDFromStdContext(ctx); got != "" {
		t.Errorf("unset context should yield empty, got %q", got)
	}
}
