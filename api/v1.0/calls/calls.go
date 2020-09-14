package calls

import (
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/api"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"
	"gitlab.com/voipbin/bin-manager/api-manager.git/servicehandler"
)

// callsPOST handles POST /calls request.
// It creates a temp flow and create a call with temp flow.
func callsPOST(c *gin.Context) {

	var requestBody RequestBodyCallsPOST

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

	// get service
	serviceHandler := c.MustGet(api.OBJServiceHandler).(servicehandler.ServiceHandler)

	// create flow
	flow, err := serviceHandler.FlowCreate(&u, uuid.Nil, "temp", "tmp outbound flow", requestBody.Actions, false)
	if err != nil {
		logrus.Errorf("Could not create a flow for outoing call. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	// create call
	res, err := serviceHandler.CallCreate(&u, flow.ID, requestBody.Source, requestBody.Destination)
	if err != nil {
		logrus.Errorf("Could not create a call for outgoing. err; %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// callsIDDelete handles DELETE /calls/<call-id> request.
// It creates a temp flow and create a call with temp flow.
func callsIDDelete(c *gin.Context) {

	// get id
	id := uuid.FromStringOrNil(c.Params.ByName("id"))

	tmp, exists := c.Get("user")
	if exists != true {
		logrus.Errorf("Could not find user info.")
		c.AbortWithStatus(400)
		return
	}
	u := tmp.(user.User)

	// get service
	serviceHandler := c.MustGet(api.OBJServiceHandler).(servicehandler.ServiceHandler)

	// get call
	res, err := serviceHandler.CallGet(&u, id)
	if err != nil {
		logrus.Infof("Could not get the call info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
