package server

import (
	"monorepo/bin-api-manager/lib/middleware"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/gin-gonic/gin"
)

// abortWithError writes the VoipbinError as a JSON body, sets the
// correct HTTP status code, and aborts the Gin context. The request
// ID is read from middleware.RequestIDFromContext, so that middleware
// must run before any handler that calls this.
//
// A nil VoipbinError falls back to a StatusInternal response so the
// helper never panics on a caller oversight.
func abortWithError(c *gin.Context, e *cerrors.VoipbinError) {
	if e == nil {
		e = cerrors.Internal(commonoutline.ServiceNameAPIManager, "INTERNAL", "An internal error occurred.")
	}
	body := gin.H{
		"status":     string(e.Status),
		"reason":     e.Reason,
		"domain":     e.Domain,
		"message":    e.Message,
		"request_id": middleware.RequestIDFromContext(c),
	}
	if len(e.Details) > 0 {
		body["details"] = e.Details
	}
	c.AbortWithStatusJSON(cerrors.HTTPStatusFor(e.Status), gin.H{"error": body})
}

// abortWithServiceError runs any error returned from servicehandler
// through the translator (see server/error_translate.go), then aborts
// with the resulting VoipbinError.
func abortWithServiceError(c *gin.Context, err error) {
	abortWithError(c, translateToVoipbinError(err))
}
