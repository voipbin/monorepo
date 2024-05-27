package storage_accounts

import "github.com/gin-gonic/gin"

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	storageAccounts := r.Group("/storage_account")

	storageAccounts.GET("", storageAccountGet)
}
