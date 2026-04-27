package server

import (
	"monorepo/bin-api-manager/lib/apierror"
	"monorepo/bin-api-manager/lib/middleware"
	cerrors "monorepo/bin-common-handler/models/errors"

	"github.com/gin-gonic/gin"
)

// abortWithError writes the VoipbinError as a JSON body, sets the
// correct HTTP status code, and aborts the Gin context. The request
// ID is read from middleware.RequestIDFromContext, so that middleware
// must run before any handler that calls this.
//
// The external envelope omits the internal Domain field — see
// bin-api-manager/lib/apierror for the boundary.
//
// A nil VoipbinError falls back to a StatusInternal response so the
// helper never panics on a caller oversight (handled inside
// apierror.EnvelopeFor).
func abortWithError(c *gin.Context, e *cerrors.VoipbinError) {
	status := cerrors.StatusInternal
	if e != nil {
		status = e.Status
	}
	c.AbortWithStatusJSON(
		cerrors.HTTPStatusFor(status),
		apierror.EnvelopeFor(e, middleware.RequestIDFromContext(c)),
	)
}

// abortWithServiceError runs any error returned from servicehandler
// through the translator (see server/error_translate.go), then aborts
// with the resulting VoipbinError.
func abortWithServiceError(c *gin.Context, err error) {
	abortWithError(c, translateToVoipbinError(err))
}
