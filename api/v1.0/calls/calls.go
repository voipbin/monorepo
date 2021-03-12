package calls

import (
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/request"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/response"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

// callsPOST handles POST /calls request.
// It creates a temp flow and create a call with temp flow.

// @Summary Make an outbound call
// @Description dialing to destination
// @Produce  json
// @Param call body request.BodyCallsPOST true "The call detail"
// @Success 200 {object} models.Call
// @Router /v1.0/calls [post]
func callsPOST(c *gin.Context) {

	var requestBody request.BodyCallsPOST

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
	u := tmp.(models.User)

	// get service
	serviceHandler := c.MustGet(models.OBJServiceHandler).(servicehandler.ServiceHandler)

	// create flow
	f := &models.Flow{
		Name:       "temp",
		Detail:     "tmp outbound flow",
		Actions:    requestBody.Actions,
		Persist:    false,
		WebhookURI: requestBody.WebhookURI,
	}
	flow, err := serviceHandler.FlowCreate(&u, f)
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
// It hangup the call.

// @Summary Hangup the call
// @Description Hangup the call of the given id
// @Produce json
// @Param id path string true "The ID of the call"
// @Success 200 {object} models.Call
// @Router /v1.0/calls/{id} [delete]
func callsIDDelete(c *gin.Context) {

	// get id
	id := uuid.FromStringOrNil(c.Params.ByName("id"))

	tmp, exists := c.Get("user")
	if exists != true {
		logrus.Errorf("Could not find user info.")
		c.AbortWithStatus(400)
		return
	}
	u := tmp.(models.User)

	// get service
	serviceHandler := c.MustGet(models.OBJServiceHandler).(servicehandler.ServiceHandler)

	// hangup the call
	err := serviceHandler.CallDelete(&u, id)
	if err != nil {
		logrus.Infof("Could not get the call info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.AbortWithStatus(200)
}

// callsGET handles GET /calls request.
// It returns list of calls of the given user.

// @Summary List calls
// @Description get calls of the user
// @Produce  json
// @Param page_size query int false "The size of results. Max 100"
// @Param page_token query string false "The token. tm_create"
// @Success 200 {object} response.BodyCallsGET
// @Router /v1.0/calls [get]
func callsGET(c *gin.Context) {

	var requestParam request.ParamCallsGET

	if err := c.BindQuery(&requestParam); err != nil {
		c.AbortWithStatus(400)
		return
	}
	log := logrus.WithFields(
		logrus.Fields{
			"request_address": c.ClientIP,
		},
	)
	log.Debugf("callsGET. Received request detail. page_size: %d, page_token: %s", requestParam.PageSize, requestParam.PageToken)

	tmp, exists := c.Get("user")
	if exists != true {
		logrus.Errorf("Could not find user info.")
		c.AbortWithStatus(400)
		return
	}
	u := tmp.(models.User)

	// get service
	serviceHandler := c.MustGet(models.OBJServiceHandler).(servicehandler.ServiceHandler)

	// set max page size
	pageSize := requestParam.PageSize
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 10
		log.Debugf("Invalid requested page size. Set to default. page_size: %d", pageSize)
	}

	// get calls
	calls, err := serviceHandler.CallGets(&u, pageSize, requestParam.PageToken)
	if err != nil {
		logrus.Errorf("Could not get calls info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(calls) > 0 {
		nextToken = calls[len(calls)-1].TMCreate
	}
	res := response.BodyCallsGET{
		Result: calls,
		Pagination: response.Pagination{
			NextPageToken: nextToken,
		},
	}

	c.JSON(200, res)
}

// callsIDGET handles GET /calls/{id} request.
// It returns detail call info.

// @Summary Returns detail call info.
// @Description Returns detail call info of the given call id.
// @Produce json
// @Param id path string true "The ID of the call"
// @Param token query string true "JWT token"
// @Success 200 {object} models.Call
// @Router /v1.0/calls/{id} [get]
func callsIDGET(c *gin.Context) {
	// get id
	id := uuid.FromStringOrNil(c.Params.ByName("id"))

	tmp, exists := c.Get("user")
	if exists != true {
		logrus.Errorf("Could not find user info.")
		c.AbortWithStatus(400)
		return
	}
	u := tmp.(models.User)
	log := logrus.WithFields(logrus.Fields{
		"id":         u.ID,
		"username":   u.Username,
		"permission": u.Permission,
	})
	log.Debug("Executing callsIDGET.")

	serviceHandler := c.MustGet(models.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.CallGet(&u, id)
	if err != nil {
		log.Errorf("Could not get a call. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
