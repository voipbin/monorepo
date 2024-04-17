package transcripts

import "github.com/gin-gonic/gin"

// ApplyRoutes applies router to the gin engine
func ApplyRoutes(r *gin.RouterGroup) {
	transcripts := r.Group("/transcripts")

	transcripts.GET("", transcriptsGET)

}
