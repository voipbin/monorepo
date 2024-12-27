package customer

import (
	"github.com/gin-gonic/gin"
)

func ApplyRoutes(r *gin.RouterGroup) {
	targets := r.Group("/customer")

	targets.GET("", customerGET)
	targets.PUT("", customerPut)
}
