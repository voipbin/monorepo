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
func NoRoute() gin.HandlerFunc {
	return func(c *gin.Context) {
		abortWithError(c, cerrors.NotFound(
			commonoutline.ServiceNameAPIManager,
			"ROUTE_NOT_FOUND",
			"The requested endpoint does not exist.",
		))
	}
}
