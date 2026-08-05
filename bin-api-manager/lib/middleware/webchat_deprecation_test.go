package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

// TestWebchatDeprecationMarker_SetsHeadersAndAlwaysCallsNext proves the
// middleware sets the RFC 8594 Deprecation/Sunset headers and never blocks
// the request -- it is purely observational, the legacy routes must keep
// working normally.
func TestWebchatDeprecationMarker_SetsHeadersAndAlwaysCallsNext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/webchat_widgets", WebchatDeprecationMarker(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/webchat_widgets", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d want 200 -- middleware must not block the request", w.Code)
	}
	if got := w.Header().Get(headerDeprecation); got != "true" {
		t.Errorf("Deprecation header = %q want %q", got, "true")
	}
	if got := w.Header().Get(headerSunset); got != webchatDeprecationSunset {
		t.Errorf("Sunset header = %q want %q", got, webchatDeprecationSunset)
	}
}

// TestWebchatDeprecationMarker_CountsByRouteTemplateAndMethod proves the
// Prometheus counter is labeled by the route TEMPLATE (c.FullPath(), e.g.
// "/webchat_widgets/:id") and HTTP method, not by the literal request path
// -- using a UUID path parameter would otherwise blow up cardinality.
func TestWebchatDeprecationMarker_CountsByRouteTemplateAndMethod(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/webchat_widgets/:id", WebchatDeprecationMarker(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	before := testutil.ToFloat64(promWebchatDeprecatedRequestsTotal.WithLabelValues("/webchat_widgets/:id", http.MethodGet))

	// Two different literal UUIDs hitting the SAME route template: the
	// counter must accumulate under one label pair, not create two.
	for _, id := range []string{
		"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
	} {
		req := httptest.NewRequest(http.MethodGet, "/webchat_widgets/"+id, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d want 200", w.Code)
		}
	}

	after := testutil.ToFloat64(promWebchatDeprecatedRequestsTotal.WithLabelValues("/webchat_widgets/:id", http.MethodGet))
	if after != before+2 {
		t.Errorf("counter for route=/webchat_widgets/:id method=GET: before=%v after=%v, want exactly +2", before, after)
	}

	// A different, unrelated route template must not have been touched by
	// the above requests.
	unrelated := testutil.ToFloat64(promWebchatDeprecatedRequestsTotal.WithLabelValues("/webchat_sessions", http.MethodGet))
	if unrelated != 0 {
		t.Errorf("unrelated route/method label was incremented: got %v want 0", unrelated)
	}
}

// TestWebchatDeprecationMarker_DoesNotFireForUnrelatedRoutes documents that
// the middleware only ever counts routes it is explicitly attached to --
// it has no global request interception, so routes that never register it
// (i.e. every hyphenated /webchat-* route) are structurally unaffected.
func TestWebchatDeprecationMarker_DoesNotFireForUnrelatedRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/webchat-widgets", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	before := testutil.ToFloat64(promWebchatDeprecatedRequestsTotal.WithLabelValues("/webchat-widgets", http.MethodGet))

	req := httptest.NewRequest(http.MethodGet, "/webchat-widgets", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d want 200", w.Code)
	}
	if got := w.Header().Get(headerDeprecation); got != "" {
		t.Errorf("Deprecation header = %q, want empty -- WebchatDeprecationMarker was never attached to this route", got)
	}

	after := testutil.ToFloat64(promWebchatDeprecatedRequestsTotal.WithLabelValues("/webchat-widgets", http.MethodGet))
	if after != before {
		t.Errorf("counter changed for a route the middleware was never attached to: before=%v after=%v", before, after)
	}
}
