package campaigncalls

import "github.com/gin-gonic/gin"

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	campaigncalls := r.Group("/campaigncalls")

	campaigncalls.GET("", campaigncallsGET)
	campaigncalls.DELETE("/:id", campaigncallsIDDELETE)
	campaigncalls.GET("/:id", campaigncallsIDGET)
}
