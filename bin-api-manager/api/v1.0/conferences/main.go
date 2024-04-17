package conferences

import "github.com/gin-gonic/gin"

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	conferences := r.Group("/conferences")

	conferences.POST("", conferencesPOST)
	conferences.GET("", conferencesGET)
	conferences.GET("/:id", conferencesIDGET)
	conferences.DELETE("/:id", conferencesIDDELETE)
	conferences.PUT("/:id", conferencesIDPUT)
	conferences.GET("/:id/media_stream", conferencesIDMediaStreamGET)
	conferences.POST("/:id/recording_start", conferencesIDRecordingStartPOST)
	conferences.POST("/:id/recording_stop", conferencesIDRecordingStopPOST)
	conferences.POST("/:id/transcribe_start", conferencesIDTranscribeStartPOST)
	conferences.POST("/:id/transcribe_stop", conferencesIDTranscribeStopPOST)
}
