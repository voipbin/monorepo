package me

import (
	"github.com/gin-gonic/gin"
)

func ApplyRoutes(r *gin.RouterGroup) {
	targets := r.Group("/me")

	targets.GET("", meGET)
}
