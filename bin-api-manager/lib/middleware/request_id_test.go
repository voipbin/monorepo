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

func TestRequestIDEchoesSafeInboundHeader(t *testing.T) {
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
		t.Errorf("did not echo safe inbound header, got %q", seen)
	}
	if got := w.Header().Get("X-Request-Id"); got != "req_client_supplied" {
		t.Errorf("response header should echo, got %q", got)
	}
}

func TestRequestIDRejectsUnsafeInboundHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name   string
		header string
	}{
		{"newline", "req_foo\nSet-Cookie: x=y"},
		{"carriage_return", "req_foo\rbad"},
		{"null_byte", "req_foo\x00bar"},
		{"space", "req foo"},
		{"special", "req_foo;rm -rf /"},
		{"too_long", strings.Repeat("a", 65)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.Use(RequestID())

			var seen string
			r.GET("/", func(c *gin.Context) {
				seen = RequestIDFromContext(c)
				c.String(http.StatusOK, "ok")
			})

			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("X-Request-Id", tt.header)
			r.ServeHTTP(w, req)

			if seen == tt.header {
				t.Errorf("unsafe header %q must not be echoed", tt.header)
			}
			if !strings.HasPrefix(seen, "req_") {
				t.Errorf("rejected inbound should fall back to generated id, got %q", seen)
			}
		})
	}
}

func TestRequestIDAcceptsMaxLenHeader(t *testing.T) {
	// Exactly maxRequestIDLen (64) and all-safe chars should be accepted.
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID())

	var seen string
	r.GET("/", func(c *gin.Context) {
		seen = RequestIDFromContext(c)
		c.String(http.StatusOK, "ok")
	})

	maxLen := strings.Repeat("a", 64)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-Id", maxLen)
	r.ServeHTTP(w, req)

	if seen != maxLen {
		t.Errorf("64-char safe header must be echoed, got %q", seen)
	}
}

func TestRequestIDFromStdContextEmptyWhenAbsent(t *testing.T) {
	if got := RequestIDFromStdContext(context.Background()); got != "" {
		t.Errorf("unset context should yield empty, got %q", got)
	}
	// Intentionally pass a nil context.Context to verify the nil-safety
	// guarantee documented on RequestIDFromStdContext.
	//nolint:staticcheck // SA1012: intentional nil to exercise nil-safety path
	if got := RequestIDFromStdContext(nil); got != "" {
		t.Errorf("nil context should yield empty, got %q", got)
	}
}

func TestRequestIDFromStdContextPropagated(t *testing.T) {
	// Full middleware run — the ID must be readable both from the Gin
	// context AND from the std context the handler receives.
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID())

	var fromGin, fromStd string
	r.GET("/", func(c *gin.Context) {
		fromGin = RequestIDFromContext(c)
		fromStd = RequestIDFromStdContext(c.Request.Context())
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w, req)

	if fromGin == "" || fromStd == "" {
		t.Fatalf("both should be set: gin=%q std=%q", fromGin, fromStd)
	}
	if fromGin != fromStd {
		t.Errorf("gin and std contexts disagree: gin=%q std=%q", fromGin, fromStd)
	}
}
