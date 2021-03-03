package numbers

import (
	"github.com/gin-gonic/gin"
)

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	numbers := r.Group("/numbers")

	numbers.GET("", numbersGET)
	numbers.POST("", numbersPOST)

	numbers.GET("/:id", numbersIDGET)
	numbers.DELETE("/:id", numbersIDDELETE)

}
