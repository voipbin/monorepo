package middleware

import (
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
