package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	// webchatDeprecationSunset is the RFC 8594 Sunset date advertised to
	// clients still calling the legacy underscore-named webchat_* routes
	// (/webchat_widgets, /webchat_sessions, /webchat_messages and their
	// subpaths). It is a fixed value tracked in code -- not computed at
	// request time -- so any change to the announced removal date is a
	// deliberate, reviewed code change.
	//
	// This is a Phase 1 placeholder, not a committed removal date. Phase 3
	// (actual removal of the legacy routes) is a separate, later decision
	// gated on Phase 2 usage monitoring -- see
	// docs/plans/2026-08-05-voip-1311-webchat-resource-path-rename-design.md.
	webchatDeprecationSunset = "Wed, 31 Dec 2026 23:59:59 GMT"

	headerDeprecation = "Deprecation"
	headerSunset      = "Sunset"
)

// promWebchatDeprecatedRequestsTotal counts requests served by the legacy
// underscore-named webchat_* route aliases, broken down by route template
// and HTTP method.
//
// The route label MUST be populated from gin.Context.FullPath() (the
// registered route template, e.g. "/v1.0/webchat_widgets/:id"), never from
// c.Request.URL.Path -- the latter contains the literal request path with
// UUIDs substituted in, which would blow up cardinality on this metric.
var promWebchatDeprecatedRequestsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: metricsNamespace,
		Name:      "webchat_deprecated_requests_total",
		Help:      "Total number of requests served by the legacy underscore-named webchat_* route aliases, by route template and method.",
	},
	[]string{"route", "method"},
)

func init() {
	prometheus.MustRegister(promWebchatDeprecatedRequestsTotal)
}

// WebchatDeprecationMarker returns a Gin middleware that marks requests to
// the legacy underscore-named webchat_* routes (/webchat_widgets,
// /webchat_sessions, /webchat_messages and their subpaths) as deprecated.
//
// It does two things, then always calls c.Next() -- the legacy routes
// remain fully functional, this middleware only observes and annotates:
//  1. Increments a Prometheus counter labeled by route template + method,
//     for Phase 2 usage monitoring ahead of an eventual Phase 3 removal.
//  2. Sets the "Deprecation" and "Sunset" response headers (RFC 8594) so
//     well-behaved clients can detect the deprecation programmatically.
func WebchatDeprecationMarker() gin.HandlerFunc {
	return func(c *gin.Context) {
		promWebchatDeprecatedRequestsTotal.WithLabelValues(c.FullPath(), c.Request.Method).Inc()

		c.Header(headerDeprecation, "true")
		c.Header(headerSunset, webchatDeprecationSunset)

		c.Next()
	}
}
