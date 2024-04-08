package billings

import "github.com/gin-gonic/gin"

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	billings := r.Group("/billings")

	billings.GET("", billingsGET)
}
