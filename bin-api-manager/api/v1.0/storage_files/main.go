package storage_files

import "github.com/gin-gonic/gin"

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	storage_files := r.Group("/storage_files")

	storage_files.GET("", storageFilesGET)
	storage_files.POST("", storageFilesPOST)
	storage_files.DELETE("/:id", storageFilesIDDELETE)
	storage_files.GET("/:id", storageFilesIDGET)
}

const (
	constMaxFileSize = int64(30 << 20) // Max upload file size. 30 MB.
)
