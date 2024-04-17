package trunks

import (
	"github.com/gin-gonic/gin"
)

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	trunks := r.Group("/trunks")

	trunks.GET("", trunksGET)
	trunks.POST("", trunksPOST)

	trunks.DELETE("/:id", trunksIDDELETE)
	trunks.GET("/:id", trunksIDGET)
	trunks.PUT("/:id", trunksIDPUT)
}
