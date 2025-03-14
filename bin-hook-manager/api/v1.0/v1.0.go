package apiv1

import (
	"github.com/gin-gonic/gin"

	"monorepo/bin-hook-manager/api/v1.0/conversation"
	"monorepo/bin-hook-manager/api/v1.0/emails"
	"monorepo/bin-hook-manager/api/v1.0/messages"
)

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	v1 := r.Group("/v1.0")

	emails.ApplyRoutes(v1)
	messages.ApplyRoutes(v1)
	conversation.ApplyRoutes(v1)
}
