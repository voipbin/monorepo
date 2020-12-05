package recordings

import (
	"github.com/gin-gonic/gin"
)

// ApplyRoutes applies router to the gin engine
func ApplyRoutes(r *gin.RouterGroup) {
	recordings := r.Group("/recordings")

	recordings.GET("/:id", recordingsIDGET)

}
