package ordernumbers

import (
	"github.com/gin-gonic/gin"
)

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	orderNumbers := r.Group("/order_numbers")

	orderNumbers.GET("", orderNumbersGET)
	orderNumbers.POST("", orderNumbersPOST)

	orderNumbers.GET("/:id", orderNumbersIDGET)
	orderNumbers.DELETE("/:id", orderNumbersIDDELETE)

}
