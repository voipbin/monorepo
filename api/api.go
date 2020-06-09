package api

import (
	"github.com/gin-gonic/gin"
	apiv1 "gitlab.com/voipbin/bin-manager/api-manager/api/v1.0"
)

// ApplyRoutes applies router to gin Router
func ApplyRoutes(r *gin.Engine) {
	api := r.Group("/")

	apiv1.ApplyRoutes(api)
}
