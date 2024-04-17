package recordingfiles

import (
	"github.com/gin-gonic/gin"
)

// ApplyRoutes applies router to the gin engine
func ApplyRoutes(r *gin.RouterGroup) {
	recordings := r.Group("/recordingfiles")

	recordings.GET("/:id", recordingfilesIDGET)

}
