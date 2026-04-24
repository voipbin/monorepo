package middleware

import (
	"context"
	"crypto/rand"
	"encoding/base32"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

const (
	headerRequestID  = "X-Request-Id"
	ctxKeyRequestID  = "request_id"
	requestIDPrefix  = "req_"
	requestIDRawBits = 16 // 16 bytes -> 26 base32 chars -> 30 total w/ prefix
	// maxRequestIDLen caps the client-supplied X-Request-Id length.
	// Anything longer is treated as malformed and the middleware
	// generates a fresh ID instead.
	maxRequestIDLen = 64
)

// RequestID returns a Gin middleware that ensures every request has a
// unique correlation ID. If the client sends a safe X-Request-Id, that
// value is echoed; otherwise a fresh ULID-like ID is generated.
//
// An inbound header is "safe" when it is non-empty, at most 64 bytes,
// and contains only characters in [A-Za-z0-9_-]. Any violation causes
// the middleware to silently discard the inbound value and generate a
// fresh ID. This prevents log injection via newlines / control
// characters and bounds memory pressure from hostile clients.
//
// The ID is stored in the Gin context, propagated to
// c.Request.Context(), and attached to the X-Request-Id response
// header.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(headerRequestID)
		if !isSafeRequestID(id) {
			id = newRequestID()
		}
		c.Set(ctxKeyRequestID, id)
		c.Request = c.Request.WithContext(
			context.WithValue(c.Request.Context(), requestIDCtxKey{}, id),
		)
		c.Writer.Header().Set(headerRequestID, id)

		// Make logrus.WithContext pick up the request_id automatically.
		c.Set("logger", logrus.WithField("request_id", id))

		c.Next()
	}
}

// RequestIDFromContext returns the request ID stored in the Gin
// context, or the empty string if RequestID middleware was not run.
func RequestIDFromContext(c *gin.Context) string {
	if c == nil {
		return ""
	}
	if v, ok := c.Get(ctxKeyRequestID); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

type requestIDCtxKey struct{}

// RequestIDFromStdContext returns the request ID attached to a
// context.Context by this middleware, or "" if not present. Used by
// non-Gin consumers such as the reqHandler RPC wrapper that forwards
// the ID into sock.Request.RequestID for downstream log correlation.
func RequestIDFromStdContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Value(requestIDCtxKey{}).(string); ok {
		return v
	}
	return ""
}

// isSafeRequestID reports whether an inbound X-Request-Id value is
// suitable for echoing: non-empty, bounded length, and limited to
// characters that cannot inject into logs (no CR/LF or control bytes).
func isSafeRequestID(id string) bool {
	if id == "" || len(id) > maxRequestIDLen {
		return false
	}
	for i := 0; i < len(id); i++ {
		c := id[i]
		switch {
		case c >= 'A' && c <= 'Z':
		case c >= 'a' && c <= 'z':
		case c >= '0' && c <= '9':
		case c == '_' || c == '-':
		default:
			return false
		}
	}
	return true
}

func newRequestID() string {
	buf := make([]byte, requestIDRawBits)
	if _, err := rand.Read(buf); err != nil {
		// Cryptographically impossible in practice; fall back to a
		// full-length placeholder so downstream format checks still
		// see req_ + 26 chars.
		return requestIDPrefix + "AAAAAAAAAAAAAAAAAAAAAAAAAA"
	}
	return requestIDPrefix + base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(buf)
}
