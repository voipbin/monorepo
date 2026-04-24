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
)

// RequestID returns a Gin middleware that ensures every request has a
// unique correlation ID. If the client sends X-Request-Id, that value
// is echoed; otherwise a fresh ULID-like ID is generated. The ID is
// stored in the Gin context, propagated to c.Request.Context(), and
// attached to the X-Request-Id response header.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(headerRequestID)
		if id == "" {
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

// requestIDFromStdContext returns the request ID attached to a
// context.Context by this middleware. Used by non-Gin consumers such
// as the reqHandler wrapper that needs to forward the ID into RPC.
func requestIDFromStdContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Value(requestIDCtxKey{}).(string); ok {
		return v
	}
	return ""
}

func newRequestID() string {
	buf := make([]byte, requestIDRawBits)
	if _, err := rand.Read(buf); err != nil {
		// Cryptographically impossible in practice; fall back to a
		// fixed sentinel so downstream code never sees an empty ID.
		return requestIDPrefix + "deadbeef"
	}
	return requestIDPrefix + base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(buf)
}
