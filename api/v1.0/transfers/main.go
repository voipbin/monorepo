package transfers

import "github.com/gin-gonic/gin"

// ApplyRoutes applies router to the gin engine
func ApplyRoutes(r *gin.RouterGroup) {
	transfers := r.Group("/transfers")

	transfers.POST("", transfersPOST)

}
