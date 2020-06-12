package conferences

import (
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/api-manager/pkg/requesthandler"
)

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	conferences := r.Group("/conferences")

	conferences.GET("/:id", conferencesIDGET)
}

func conferencesIDGET(c *gin.Context) {
	// send a request to call-manager
	ID := uuid.FromStringOrNil(c.Params.ByName("id"))

	// send a request to call
	requestHandler := c.MustGet("requestHandler").(requesthandler.RequestHandler)
	res, err := requestHandler.CallConferenceGet(ID)
	if err != nil || res == nil {
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
