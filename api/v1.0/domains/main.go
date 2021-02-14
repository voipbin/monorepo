package domains

import (
	"github.com/gin-gonic/gin"
)

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	domains := r.Group("/domains")

	domains.GET("", domainsGET)
	domains.POST("", domainsPOST)

	domains.DELETE("/:id", domainsIDDELETE)
	domains.GET("/:id", domainsIDGET)
	domains.PUT("/:id", domainsIDPUT)
}
