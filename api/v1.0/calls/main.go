package calls

import (
	"github.com/gin-gonic/gin"
)

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	calls := r.Group("/calls")

	calls.GET("", callsGET)
	calls.POST("", callsPOST)
	calls.GET("/:id", callsIDDelete)
	// calls.DELETE("/:id", conferencesIDDELETE)
}
