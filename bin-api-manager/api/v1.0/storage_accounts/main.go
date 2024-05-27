package storage_accounts

import "github.com/gin-gonic/gin"

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	storageAccounts := r.Group("/storage_accounts")

	storageAccounts.GET("", storageAccountsGet)
	storageAccounts.POST("", storageAccountsPost)

	storageAccounts.GET("/:id", storageAccountsIDGet)
	storageAccounts.DELETE("/:id", storageAccountsIDDelete)
}
