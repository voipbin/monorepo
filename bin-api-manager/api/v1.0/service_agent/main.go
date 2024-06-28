package service_talk

import "github.com/gin-gonic/gin"

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	targets := r.Group("/service_agent")

	targets.GET("/calls", callsGET)

	// targets.GET("", routesGET)
	// targets.POST("", routesPOST)
	// targets.GET("/:id", routesIDGet)
	// targets.DELETE("/:id", routesIDDelete)
	// targets.PUT("/:id", routesIDPUT)
}
