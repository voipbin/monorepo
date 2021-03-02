package availablenumbers

import (
	"github.com/gin-gonic/gin"
)

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	availableNumbers := r.Group("/available_numbers")

	availableNumbers.GET("", availableNumbersGET)
}
