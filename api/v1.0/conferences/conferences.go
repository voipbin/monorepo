package conferences

import (
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/api-manager/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/api-manager/pkg/requesthandler/models/conference"
)

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	conferences := r.Group("/conferences")

	conferences.POST("", conferencesPOST)
	conferences.GET("/:id", conferencesIDGET)
	conferences.DELETE("/:id", conferencesIDDELETE)
}

func conferencesPOST(c *gin.Context) {

	type RequestBody struct {
		Type conference.Type `json:"type" binding:"required"`
	}
	var requestBody RequestBody

	if err := c.BindJSON(&requestBody); err != nil {
		c.AbortWithStatus(400)
		return
	}

	// send a request to call
	requestHandler := c.MustGet("requestHandler").(requesthandler.RequestHandler)
	res, err := requestHandler.CallConferenceCreate(requestBody.Type)
	if err != nil || res == nil {
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func conferencesIDGET(c *gin.Context) {
	// get id
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

func conferencesIDDELETE(c *gin.Context) {
	// get id
	ID := uuid.FromStringOrNil(c.Params.ByName("id"))

	// send a request to call
	requestHandler := c.MustGet("requestHandler").(requesthandler.RequestHandler)
	if err := requestHandler.CallConferenceDelete(ID); err != nil {
		c.AbortWithStatus(400)
		return
	}

	c.AbortWithStatus(200)
}
