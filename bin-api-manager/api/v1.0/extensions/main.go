package extensions

import (
	"github.com/gin-gonic/gin"
)

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	exts := r.Group("/extensions")

	exts.GET("", extensionsGET)
	exts.POST("", extensionsPOST)

	exts.DELETE("/:id", extensionsIDDELETE)
	exts.GET("/:id", extensionsIDGET)
	exts.PUT("/:id", extensionsIDPUT)
}
