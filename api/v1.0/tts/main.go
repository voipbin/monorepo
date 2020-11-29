package tts

import "github.com/gin-gonic/gin"

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	tts := r.Group("/tts")

	tts.GET("/:filename", ttsGET)
}
