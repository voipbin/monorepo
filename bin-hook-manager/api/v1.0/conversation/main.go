package conversation

import "github.com/gin-gonic/gin"

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {

	conversation := r.Group("/conversation")

	conversation.Any("*any", conversationPOST)
}
