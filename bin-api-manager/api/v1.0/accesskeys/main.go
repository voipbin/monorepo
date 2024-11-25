package accesskeys

import "github.com/gin-gonic/gin"

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	targets := r.Group("/accesskeys")

	targets.POST("", accesskeysPOST)
	targets.GET("", accesskeysGET)

	targets.DELETE("/:id", accesskeysIDDelete)
	targets.GET("/:id", accesskeysIDGet)
	targets.PUT("/:id", accesskeysIDPUT)
}
