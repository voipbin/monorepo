package calls

import (
	"github.com/gin-gonic/gin"
)

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	calls := r.Group("/calls")

	calls.GET("", callsGET)
	calls.POST("", callsPOST)
	calls.DELETE("/:id", callsIDDelete)
	calls.GET("/:id", callsIDGET)
	calls.POST("/:id/hangup", callsIDHangupPOST)
	calls.POST("/:id/talk", callsIDTalkPOST)
	calls.POST("/:id/hold", callsIDHoldPOST)
	calls.DELETE("/:id/hold", callsIDHoldDELETE)
	calls.GET("/:id/media_stream", callsIDMediaStreamGET)
	calls.POST("/:id/moh", callsIDMOHPOST)
	calls.DELETE("/:id/moh", callsIDMOHDELETE)
	calls.POST("/:id/mute", callsIDMutePOST)
	calls.DELETE("/:id/mute", callsIDMuteDELETE)
	calls.POST("/:id/silence", callsIDSilencePOST)
	calls.DELETE("/:id/silence", callsIDSilenceDELETE)
}
