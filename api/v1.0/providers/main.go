package providers

import "github.com/gin-gonic/gin"

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	targets := r.Group("/providers")

	targets.GET("", providersGET)
	targets.POST("", providersPOST)
	targets.GET("/:id", providersIDGet)
	targets.DELETE("/:id", providersIDDelete)
	targets.PUT("/:id", providersIDPUT)
}
