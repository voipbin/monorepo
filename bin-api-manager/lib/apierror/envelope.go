// Package apierror builds the external HTTP error envelope for the
// VoIPbin public API. The envelope intentionally omits the internal
// Domain (originating service name) carried by VoipbinError — that field
// is internal-only and must not cross the public API boundary. The
// VoipbinError.Domain field stays available for server-side logs and
// internal RPC; only this external serialization strips it.
package apierror

import (
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/gin-gonic/gin"
)

// EnvelopeFor returns the JSON body for an external HTTP error response.
// Pass the request ID extracted from the gin context. A nil VoipbinError
// falls back to a generic INTERNAL envelope so callers never panic on a
// missed nil check.
func EnvelopeFor(e *cerrors.VoipbinError, requestID string) gin.H {
	if e == nil {
		e = cerrors.Internal(
			commonoutline.ServiceNameAPIManager,
			"INTERNAL",
			"An internal error occurred.",
		)
	}
	body := gin.H{
		"status":     string(e.Status),
		"reason":     e.Reason,
		"message":    e.Message,
		"request_id": requestID,
	}
	if len(e.Details) > 0 {
		body["details"] = e.Details
	}
	return gin.H{"error": body}
}
