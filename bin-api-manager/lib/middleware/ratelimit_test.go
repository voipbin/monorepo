package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRateLimit_AllowsNormalTraffic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RateLimit(10, 20))
	r.POST("/auth/login", func(c *gin.Context) {
		c.Status(200)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/auth/login", nil)
	req.RemoteAddr = "1.2.3.4:1234"
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestRateLimit_BlocksExcessiveTraffic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RateLimit(1, 2)) // very restrictive for testing
	r.POST("/auth/login", func(c *gin.Context) {
		c.Status(200)
	})

	var lastCode int
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/auth/login", nil)
		req.RemoteAddr = "1.2.3.4:1234"
		r.ServeHTTP(w, req)
		lastCode = w.Code
	}

	if lastCode != 429 {
		t.Errorf("expected 429 after exceeding rate limit, got %d", lastCode)
	}
}

func TestRateLimit_DifferentIPsIndependent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RateLimit(1, 1))
	r.POST("/auth/login", func(c *gin.Context) {
		c.Status(200)
	})

	// Exhaust rate limit for IP 1
	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/auth/login", nil)
		req.RemoteAddr = "1.2.3.4:1234"
		r.ServeHTTP(w, req)
	}

	// IP 2 should still be allowed
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/auth/login", nil)
	req.RemoteAddr = "5.6.7.8:1234"
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("different IP should not be rate limited, got %d", w.Code)
	}
}

func TestRateLimit_EnvelopeShape(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID())
	r.Use(RateLimit(0.001, 1)) // 1 request per ~1000s, burst 1 — second request is blocked.

	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	// First request — allowed.
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, httptest.NewRequest(http.MethodGet, "/", nil))

	// Second request — blocked with new envelope.
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/", nil))

	if w2.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d want 429", w2.Code)
	}

	var body struct {
		Error struct {
			Status    string `json:"status"`
			Reason    string `json:"reason"`
			Message   string `json:"message"`
			RequestID string `json:"request_id"`
		} `json:"error"`
	}
	if err := json.Unmarshal(w2.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v; body: %s", err, w2.Body.String())
	}
	if body.Error.Status != "RESOURCE_EXHAUSTED" {
		t.Errorf("wrong status: %q", body.Error.Status)
	}
	if body.Error.Reason != "RATE_LIMIT_EXCEEDED" {
		t.Errorf("wrong reason: %q", body.Error.Reason)
	}
	if body.Error.Message == "" {
		t.Error("message missing")
	}
	if body.Error.RequestID == "" {
		t.Error("request_id missing")
	}
	// Structural check: verify the "domain" key is absent from the
	// external response (see bin-api-manager/lib/apierror).
	var full map[string]any
	if err := json.Unmarshal(w2.Body.Bytes(), &full); err != nil {
		t.Fatalf("unmarshal full body for domain check: %v; body=%s", err, w2.Body.String())
	}
	errObj, ok := full["error"].(map[string]any)
	if !ok {
		t.Fatalf("body.error is not an object: %+v", full)
	}
	if _, hasDomain := errObj["domain"]; hasDomain {
		t.Errorf("domain key MUST be absent from external response; body=%s", w2.Body.String())
	}
}
