package chats

import "github.com/gin-gonic/gin"

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	chats := r.Group("/chats")

	chats.GET("", chatsGET)
	chats.POST("", chatsPOST)

	chats.DELETE("/:id", chatsIDDELETE)
	chats.GET("/:id", chatsIDGET)
	chats.PUT("/:id", chatsIDPUT)

	chats.PUT("/:id/room_owner_id", chatsIDRoomOwnerIDPUT)
	chats.POST("/:id/participant_ids", chatsIDParticipantIDsPOST)
	chats.DELETE("/:id/participant_ids/:participant_id", chatsIDParticipantIDsIDDELETE)
}
