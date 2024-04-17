package conversationaccounts

import "github.com/gin-gonic/gin"

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	conversationAccounts := r.Group("/conversation_accounts")

	conversationAccounts.GET("", conversationAccountsGet)
	conversationAccounts.POST("", conversationAccountsPost)

	conversationAccounts.GET("/:id", conversationAccountsIDGet)
	conversationAccounts.PUT("/:id", conversationAccountsIDPut)
	conversationAccounts.DELETE("/:id", conversationAccountsIDDelete)
}
