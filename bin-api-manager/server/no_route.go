package server

import (
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/gin-gonic/gin"
)

// NoRoute returns a Gin handler registered via app.NoRoute(...) that
// emits the canonical error envelope for paths the router does not
// match. Without this, unrouted paths fall through to Gin's default
// "404 page not found" plain-text response, which breaks the
// consistent error contract documented in restful_api.rst.
//
// The reason is ROUTE_NOT_FOUND (not RESOURCE_NOT_FOUND) to distinguish
// a wrong URL path from a valid path whose underlying resource does
// not exist.
//
// Known gap: Gin's HandleMethodNotAllowed defaults to false, so a
// request with a valid path but wrong method (e.g. POST /v1.0/me when
// only GET is registered) also falls through to this handler and gets
// ROUTE_NOT_FOUND. Proper 405 handling requires an 11th canonical
// METHOD_NOT_ALLOWED status in bin-common-handler/models/errors, which
// is a coordinated schema update deferred to a future PR. See
// docs/plans/2026-04-24-api-error-response-codes-design.md §6 for the
// canonical status list.
func NoRoute() gin.HandlerFunc {
	return func(c *gin.Context) {
		abortWithError(c, cerrors.NotFound(
			commonoutline.ServiceNameAPIManager,
			"ROUTE_NOT_FOUND",
			"The requested endpoint does not exist.",
		))
	}
}
