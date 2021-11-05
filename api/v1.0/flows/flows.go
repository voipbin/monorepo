package flows

import (
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/request"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/response"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/flow"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

// flowsPOST handles POST /flows request.
// It creates a new flow with the given info and returns created flow info.
// @Summary Create a new flow and returns detail created flow info.
// @Description Create a new flow and returns detail created flow info.
// @Produce json
// @Success 200 {object} flow.Flow
// @Router /v1.0/flows [post]
func flowsPOST(c *gin.Context) {

	var body request.BodyFlowsPOST
	if err := c.BindJSON(&body); err != nil {
		c.AbortWithStatus(400)
		return
	}

	tmp, exists := c.Get("user")
	if !exists {
		logrus.Errorf("Could not find user info.")
		c.AbortWithStatus(400)
		return
	}
	u := tmp.(user.User)
	log := logrus.WithFields(logrus.Fields{
		"id":         u.ID,
		"username":   u.Username,
		"permission": u.Permission,
	})

	// create a flow
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	f := &flow.Flow{
		Name:       body.Name,
		Detail:     body.Detail,
		Actions:    body.Actions,
		Persist:    true,
		WebhookURI: body.WebhookURI,
	}
	res, err := serviceHandler.FlowCreate(&u, f)
	if err != nil {
		log.Errorf("Could not create a flow. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// flowsGET handles GET /flows request.
// It gets a list of flows with the given info.
// @Summary Gets a list of flows.
// @Description Gets a list of flows
// @Produce json
// @Success 200 {array} flow.Flow
// @Router /v1.0/flows [get]
func flowsGET(c *gin.Context) {

	var requestParam request.ParamFlowsGET

	if err := c.BindQuery(&requestParam); err != nil {
		c.AbortWithStatus(400)
		return
	}
	log := logrus.WithFields(
		logrus.Fields{
			"request_address": c.ClientIP,
		},
	)
	log.Debugf("flowsGET. Received request detail. page_size: %d, page_token: %s", requestParam.PageSize, requestParam.PageToken)

	tmp, exists := c.Get("user")
	if !exists {
		logrus.Errorf("Could not find user info.")
		c.AbortWithStatus(400)
		return
	}

	u := tmp.(user.User)
	log = log.WithFields(logrus.Fields{
		"id":         u.ID,
		"username":   u.Username,
		"permission": u.Permission,
	})

	// set max page size
	pageSize := requestParam.PageSize
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 10
		log.Debugf("Invalid requested page size. Set to default. page_size: %d", pageSize)
	}

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// get flows
	flows, err := serviceHandler.FlowGets(&u, pageSize, requestParam.PageToken)
	if err != nil {
		log.Errorf("Could not get a flow list. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(flows) > 0 {
		nextToken = flows[len(flows)-1].TMCreate
	}
	res := response.BodyFlowsGET{
		Result: flows,
		Pagination: response.Pagination{
			NextPageToken: nextToken,
		},
	}

	c.JSON(200, res)
}

// flowsIDGET handles GET /flows/{id} request.
// It returns detail flow info.
// @Summary Returns detail flow info.
// @Description Returns detail flow info of the given flow id.
// @Produce json
// @Param id path string true "The ID of the flow"
// @Param token query string true "JWT token"
// @Success 200 {object} flow.Flow
// @Router /v1.0/flows/{id} [get]
func flowsIDGET(c *gin.Context) {
	// get id
	id := uuid.FromStringOrNil(c.Params.ByName("id"))

	tmp, exists := c.Get("user")
	if !exists {
		logrus.Errorf("Could not find user info.")
		c.AbortWithStatus(400)
		return
	}
	u := tmp.(user.User)
	log := logrus.WithFields(logrus.Fields{
		"id":         u.ID,
		"username":   u.Username,
		"permission": u.Permission,
	})
	log.Debug("Executing flowsIDGET.")

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.FlowGet(&u, id)
	if err != nil {
		log.Errorf("Could not get a flow. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// flowsIDPUT handles PUT /flows/{id} request.
// It updates a exist flow info with the given flow info.
// And returns updated flow info if it succeed.
// @Summary Update a flow and reuturns updated flow info.
// @Description Update a flow and returns detail updated flow info.
// @Produce json
// @Success 200 {object} flow.Flow
// @Router /v1.0/flows/{id} [put]
func flowsIDPUT(c *gin.Context) {

	// get id
	id := uuid.FromStringOrNil(c.Params.ByName("id"))

	var body request.BodyFlowsIDPUT
	if err := c.BindJSON(&body); err != nil {
		c.AbortWithStatus(400)
		return
	}

	tmp, exists := c.Get("user")
	if !exists {
		logrus.Errorf("Could not find user info.")
		c.AbortWithStatus(400)
		return
	}
	u := tmp.(user.User)
	log := logrus.WithFields(logrus.Fields{
		"id":         u.ID,
		"username":   u.Username,
		"permission": u.Permission,
	})

	f := &flow.Flow{
		ID:      id,
		Name:    body.Name,
		Detail:  body.Detail,
		Actions: body.Actions,
	}

	// update a flow
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.FlowUpdate(&u, f)
	if err != nil {
		log.Errorf("Could not create a flow. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// flowsIDDELETE handles DELETE /flows/{id} request.
// It deletes a exist flow info.
// @Summary Delete a existing flow.
// @Description Delete a existing flow.
// @Produce json
// @Success 200
// @Router /v1.0/flows/{id} [delete]
func flowsIDDELETE(c *gin.Context) {

	// get id
	id := uuid.FromStringOrNil(c.Params.ByName("id"))

	tmp, exists := c.Get("user")
	if !exists {
		logrus.Errorf("Could not find user info.")
		c.AbortWithStatus(400)
		return
	}
	u := tmp.(user.User)
	log := logrus.WithFields(logrus.Fields{
		"id":         u.ID,
		"username":   u.Username,
		"permission": u.Permission,
	})

	// delete a flow
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	if err := serviceHandler.FlowDelete(&u, id); err != nil {
		log.Errorf("Could not create a flow. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.AbortWithStatus(200)
}
