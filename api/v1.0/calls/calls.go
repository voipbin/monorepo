package calls

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/api-manager/models/action"
	"gitlab.com/voipbin/bin-manager/api-manager/models/user"
	"gitlab.com/voipbin/bin-manager/api-manager/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/api-manager/pkg/requesthandler/models/cmconference"
)

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	conferences := r.Group("/conferences")

	conferences.POST("", callsPOST)
	// conferences.GET("/:id", conferencesIDGET)
	// conferences.DELETE("/:id", conferencesIDDELETE)
}

func callsPOST(c *gin.Context) {

	type RequestBody struct {
		Source      string          `json:"type"`
		Destination string          `json:"destination" binding:"required"`
		Actions     []action.Action `json:"actions"`
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
	user := tmp.(user.User)

	// servicehandler := c.MustGet("servicehandler").(servicehandler.ServiceHandler)

	// create flow

	// create call

	// servicehandler.CallCreate(user.ID, )

	// send a request to call
	requestHandler := c.MustGet("requestHandler").(requesthandler.RequestHandler)
	res, err := requestHandler.CallConferenceCreate(user.ID, cmconference.TypeConference, "", "")
	if err != nil || res == nil {
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
