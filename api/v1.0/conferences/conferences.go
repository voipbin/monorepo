package conferences

import (
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/api-manager/models/api"
	"gitlab.com/voipbin/bin-manager/api-manager/models/conference"
	"gitlab.com/voipbin/bin-manager/api-manager/models/user"
	"gitlab.com/voipbin/bin-manager/api-manager/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/api-manager/servicehandler"
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
		Type   conference.Type `json:"type" binding:"required"`
		Name   string          `json:"name"`
		Detail string          `json:"detail"`
	}
	var requestBody RequestBody

	if err := c.BindJSON(&requestBody); err != nil {
		c.AbortWithStatus(400)
		return
	}

	tmp, exists := c.Get("user")
	if exists != true {
		logrus.Errorf("Could not find user info.")
		c.AbortWithStatus(400)
		return
	}
	u := tmp.(user.User)

	servicehandler := c.MustGet(api.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := servicehandler.ConferenceCreate(&u, requestBody.Type, requestBody.Name, requestBody.Detail)
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
	res, err := requestHandler.CMConferenceGet(ID)
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
	if err := requestHandler.CMConferenceDelete(ID); err != nil {
		c.AbortWithStatus(400)
		return
	}

	c.AbortWithStatus(200)
}
