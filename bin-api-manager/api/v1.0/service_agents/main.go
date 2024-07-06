package service_agents

import "github.com/gin-gonic/gin"

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	targets := r.Group("/service_agents")

	// calls
	targets.GET("/calls", callsGET)
	targets.GET("/calls/:id", callsIDGET)

	// chatrooms
	targets.GET("/chatrooms", chatroomsGET)
	targets.POST("/chatrooms", chatroomsPOST)
	targets.GET("/chatrooms/:id", chatroomsIDGET)
	targets.PUT("/chatrooms/:id", chatroomsIDPUT)
	targets.DELETE("/chatrooms/:id", chatroomsIDDELETE)
}
