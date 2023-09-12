package conferencecalls

import "github.com/gin-gonic/gin"

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	conferencecalls := r.Group("/conferencecalls")

	conferencecalls.GET("", conferencecallsGET)
	conferencecalls.GET("/:id", conferencecallsIDGET)
	conferencecalls.DELETE("/:id", conferencecallsIDDELETE)
}
