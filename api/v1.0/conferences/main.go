package conferences

import "github.com/gin-gonic/gin"

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	conferences := r.Group("/conferences")

	conferences.POST("", conferencesPOST)
	conferences.GET("", conferencesGET)
	conferences.GET("/:id", conferencesIDGET)
	conferences.DELETE("/:id", conferencesIDDELETE)
}
