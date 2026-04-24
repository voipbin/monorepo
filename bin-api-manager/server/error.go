package server

import (
	stderrors "errors"

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
	c.AbortWithStatusJSON(cerrors.HTTPStatusFor(e.Status), gin.H{
		"error": gin.H{
			"status":     string(e.Status),
			"reason":     e.Reason,
			"domain":     e.Domain,
			"message":    e.Message,
			"request_id": middleware.RequestIDFromContext(c),
		},
	})
}

// abortWithServiceError runs any error returned from servicehandler
// through the translator, then aborts with the resulting VoipbinError.
// The translator is replaced with a richer fallback chain in
// server/error_translate.go (next commit); this file only includes
// the typed-passthrough shortcut and a default Internal return.
func abortWithServiceError(c *gin.Context, err error) {
	abortWithError(c, translateToVoipbinError(err))
}

// translateToVoipbinError is the minimal translator used until Task 5
// lands the full fallback chain. For now it recognises typed errors
// (via errors.As) and defaults everything else to Internal. Do not
// extend this stub — the replacement in error_translate.go handles
// sentinels, transport errors, and substring fallback.
func translateToVoipbinError(err error) *cerrors.VoipbinError {
	if err == nil {
		return cerrors.Internal(commonoutline.ServiceNameAPIManager, "INTERNAL", "An internal error occurred.")
	}
	var ve *cerrors.VoipbinError
	if stderrors.As(err, &ve) {
		return ve
	}
	return cerrors.Internal(commonoutline.ServiceNameAPIManager, "INTERNAL", "An internal error occurred.").Wrap(err)
}
