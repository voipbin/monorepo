package middleware

import (
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"golang.org/x/time/rate"
)

func TestRateLimit_AllowsNormalTraffic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RateLimit("test_allows_normal_traffic", 10, 20))
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
	r.Use(RateLimit("test_blocks_excessive_traffic", 1, 2)) // very restrictive for testing
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
	r.Use(RateLimit("test_different_ips_independent", 1, 1))
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
	r.Use(RateLimit("test_envelope_shape", 0.001, 1)) // 1 request per ~1000s, burst 1 — second request is blocked.

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

// TestRateLimit_DisabledZeroBurst verifies that a burst <= 0 disables the
// tier (unlimited pass-through) instead of denying every request.
func TestRateLimit_DisabledZeroBurst(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RateLimit("test_disabled_zero_burst", 10, 0))
	r.GET("/", func(c *gin.Context) { c.Status(200) })

	for i := 0; i < 20; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "1.2.3.4:1234"
		r.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Fatalf("request %d: expected 200 (disabled tier), got %d", i, w.Code)
		}
	}
}

// TestRateLimit_DisabledNonPositiveRate verifies that zero, negative, and
// NaN rates all resolve to the disabled/pass-through path rather than
// denying every request.
func TestRateLimit_DisabledNonPositiveRate(t *testing.T) {
	cases := []struct {
		name string
		rate float64
	}{
		{"zero", 0},
		{"negative", -5},
		{"nan", math.NaN()},
		{"negative_inf", math.Inf(-1)},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			r := gin.New()
			r.Use(RateLimit("test_disabled_"+tt.name, tt.rate, 20))
			r.GET("/", func(c *gin.Context) { c.Status(200) })

			for i := 0; i < 20; i++ {
				w := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				req.RemoteAddr = "1.2.3.4:1234"
				r.ServeHTTP(w, req)
				if w.Code != 200 {
					t.Fatalf("request %d: expected 200 (disabled tier), got %d", i, w.Code)
				}
			}
		})
	}
}

// TestRateLimit_PositiveInfNormalized verifies that a +Inf rate is
// normalized to the library's actual Inf sentinel (allow-all via the
// intended fast path) rather than left as IEEE +Inf, which would silently
// miss golang.org/x/time/rate's internal fast path and fall into
// unspecified floating-point behavior in the limiter's token accounting.
func TestRateLimit_PositiveInfNormalized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RateLimit("test_positive_inf", math.Inf(1), 1))
	r.GET("/", func(c *gin.Context) { c.Status(200) })

	// A burst-1 limiter with a finite rate would block after the first
	// request; with the Inf sentinel correctly applied, every request in
	// quick succession is allowed.
	for i := 0; i < 50; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "1.2.3.4:1234"
		r.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Fatalf("request %d: expected 200 (Inf-normalized tier), got %d", i, w.Code)
		}
	}
}

// TestRateLimit_MetricsIncrement verifies the allowed/rejected CounterVecs
// increment on the correct branch, and that a series is pre-initialized at
// construction time (present at 0 before any request is served).
func TestRateLimit_MetricsIncrement(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handler := RateLimit("test_metrics_increment", 1, 1)
	r.Use(handler)
	r.GET("/", func(c *gin.Context) { c.Status(200) })

	// Pre-initialized zero-value series must be readable immediately,
	// before any request is served.
	if v := testutil.ToFloat64(promRateLimitAllowedTotal.WithLabelValues("test_metrics_increment")); v != 0 {
		t.Errorf("expected pre-initialized allowed series at 0, got %v", v)
	}
	if v := testutil.ToFloat64(promRateLimitRejectedTotal.WithLabelValues("test_metrics_increment")); v != 0 {
		t.Errorf("expected pre-initialized rejected series at 0, got %v", v)
	}

	allowedBefore := testutil.ToFloat64(promRateLimitAllowedTotal.WithLabelValues("test_metrics_increment"))
	rejectedBefore := testutil.ToFloat64(promRateLimitRejectedTotal.WithLabelValues("test_metrics_increment"))

	// First request consumes the sole burst token — allowed.
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	req1.RemoteAddr = "9.9.9.9:1234"
	r.ServeHTTP(w1, req1)

	// Second request — rejected.
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.RemoteAddr = "9.9.9.9:1234"
	r.ServeHTTP(w2, req2)

	allowedAfter := testutil.ToFloat64(promRateLimitAllowedTotal.WithLabelValues("test_metrics_increment"))
	rejectedAfter := testutil.ToFloat64(promRateLimitRejectedTotal.WithLabelValues("test_metrics_increment"))

	if allowedAfter-allowedBefore != 1 {
		t.Errorf("expected allowed counter to increment by 1, got %v", allowedAfter-allowedBefore)
	}
	if rejectedAfter-rejectedBefore != 1 {
		t.Errorf("expected rejected counter to increment by 1, got %v", rejectedAfter-rejectedBefore)
	}
}

// TestRateLimit_DisabledMetricsPreInitialized verifies the disabled path
// also pre-initializes both series at 0, so a disabled tier is
// distinguishable in metrics from a tier that has simply seen no traffic.
func TestRateLimit_DisabledMetricsPreInitialized(t *testing.T) {
	_ = RateLimit("test_disabled_metrics", 0, 0)

	if v := testutil.ToFloat64(promRateLimitAllowedTotal.WithLabelValues("test_disabled_metrics")); v != 0 {
		t.Errorf("expected pre-initialized allowed series at 0, got %v", v)
	}
	if v := testutil.ToFloat64(promRateLimitRejectedTotal.WithLabelValues("test_disabled_metrics")); v != 0 {
		t.Errorf("expected pre-initialized rejected series at 0, got %v", v)
	}
}

// sanity check that rate.Inf is in fact the MaxFloat64 sentinel, not IEEE
// +Inf, to guard against a future golang.org/x/time/rate upgrade silently
// changing the semantics this file's normalization logic depends on.
func TestRateLimit_LibraryInfSentinelAssumption(t *testing.T) {
	if math.IsInf(float64(rate.Inf), 1) {
		t.Fatalf("golang.org/x/time/rate.Inf is now IEEE +Inf; RateLimit's +Inf normalization logic may be redundant or incorrect, re-check ratelimit.go")
	}
}
