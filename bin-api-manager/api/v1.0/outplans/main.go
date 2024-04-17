package outplans

import "github.com/gin-gonic/gin"

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	outplans := r.Group("/outplans")

	outplans.GET("", outplansGET)
	outplans.POST("", outplansPOST)

	outplans.DELETE("/:id", outplansIDDELETE)
	outplans.GET("/:id", outplansIDGET)
	outplans.PUT("/:id", outplansIDPUT)

	outplans.PUT("/:id/dial_info", outplansIDDialInfoPUT)
}
