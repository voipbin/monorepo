package service_agents

import "github.com/gin-gonic/gin"

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	targets := r.Group("/service_agents")

	// agents
	targets.GET("/agents", agentsGET)
	targets.GET("/agents/:id", agentIDGET)

	// calls
	targets.GET("/calls", callsGET)
	targets.GET("/calls/:id", callsIDGET)

	// chatrooms
	targets.GET("/chatrooms", chatroomsGET)
	targets.POST("/chatrooms", chatroomsPOST)
	targets.GET("/chatrooms/:id", chatroomsIDGET)
	targets.PUT("/chatrooms/:id", chatroomsIDPUT)
	targets.DELETE("/chatrooms/:id", chatroomsIDDELETE)

	// chatroom messages
	targets.GET("/chatroommessages", chatroommessagesGET)
	targets.POST("/chatroommessages", chatroommessagesPOST)
	targets.GET("/chatroommessages/:id", chatroommessagesIDGET)
	targets.DELETE("/chatroommessages/:id", chatroommessagesIDDELETE)

	// conversations
	targets.GET("/conversations", conversationsGET)
	targets.GET("/conversations/:id", conversationsIDGET)
	targets.GET("/conversations/:id/messages", conversationsIDMessagesGet)
	targets.POST("/conversations/:id/messages", conversationsIDMessagesPost)

	// extensions
	targets.GET("/extensions", extensionsGET)
	targets.GET("/extensions/:id", extensionsIDGET)

	// me
	targets.GET("/me", meGET)
	targets.PUT("/me", mePUT)
	targets.PUT("/me/addresses", meAddressesPUT)
	targets.PUT("/me/status", meStatusPUT)
	targets.PUT("/me/password", mePasswordPUT)

	// ws
	targets.GET("/ws", wsGET)
}
