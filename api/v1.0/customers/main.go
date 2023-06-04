package customers

import "github.com/gin-gonic/gin"

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	customers := r.Group("/customers")

	customers.GET("", customersGet)
	customers.POST("", customersPost)

	customers.DELETE("/:id", customersIDDelete)
	customers.GET("/:id", customersIDGet)
	customers.PUT("/:id", customersIDPut)

	customers.PUT("/:id/permission_ids", customersIDPermissionIDsPut)
	customers.PUT("/:id/password", customersIDPasswordPut)
}
