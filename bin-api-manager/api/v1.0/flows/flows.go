package flows

import (
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	fmflow "gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/request"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/response"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

// flowsPOST handles POST /flows request.
// It creates a new flow with the given info and returns created flow info.
//	@Summary		Create a new flow and returns detail created flow info.
//	@Description	Create a new flow and returns detail created flow info.
//	@Produce		json
//	@Param			flow	body		request.BodyFlowsPOST	true	"flow info."
//	@Success		200		{object}	flow.Flow
//	@Router			/v1.0/flows [post]
func flowsPOST(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "flowsPOST",
		"request_address": c.ClientIP,
	})

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)
	log = log.WithFields(logrus.Fields{
		"agent": a,
	})

	var req request.BodyFlowsPOST
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log.WithField("request", req).Debug("Executing flowsPOST.")

	// create a flow
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.FlowCreate(c.Request.Context(), &a, req.Name, req.Detail, req.Actions, true)
	if err != nil {
		log.Errorf("Could not create a flow. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// flowsGET handles GET /flows request.
// It gets a list of flows with the given info.
//	@Summary		Gets a list of flows.
//	@Description	Gets a list of flows
//	@Produce		json
//	@Param			page_size	query		int		false	"The size of results. Max 100"
//	@Param			page_token	query		string	false	"The token. tm_create"
//	@Success		200			{object}	response.BodyFlowsGET
//	@Router			/v1.0/flows [get]
func flowsGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "flowsGET",
		"request_address": c.ClientIP,
		"request":         c.Request,
	})

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)
	log = log.WithFields(logrus.Fields{
		"agent": a,
	})

	var req request.ParamFlowsGET
	if err := c.BindQuery(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	// set max page size
	pageSize := req.PageSize
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 100
		log.Debugf("Invalid requested page size. Set to default. page_size: %d", pageSize)
	}
	log.Debugf("flowsGET. Received request detail. page_size: %d, page_token: %s", pageSize, req.PageToken)

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// get flows
	flows, err := serviceHandler.FlowGets(c.Request.Context(), &a, pageSize, req.PageToken)
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
//	@Summary		Returns detail flow info.
//	@Description	Returns detail flow info of the given flow id.
//	@Produce		json
//	@Param			id	path		string	true	"The ID of the flow"
//	@Success		200	{object}	flow.Flow
//	@Router			/v1.0/flows/{id} [get]
func flowsIDGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "flowsIDGET",
		"request_address": c.ClientIP,
	})

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)
	log = log.WithFields(logrus.Fields{
		"agent": a,
	})

	// get id
	id := uuid.FromStringOrNil(c.Params.ByName("id"))
	log = log.WithField("flow_id", id)
	log.Debug("Executing flowsIDGET.")

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.FlowGet(c.Request.Context(), &a, id)
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
//	@Summary		Update a flow and reuturns updated flow info.
//	@Description	Update a flow and returns detail updated flow info.
//	@Produce		json
//	@Param			id			query		string					true	"The flow's id"
//	@Param			update_info	body		request.BodyFlowsIDPUT	true	"The update info"
//	@Success		200			{object}	flow.Flow
//	@Router			/v1.0/flows/{id} [put]
func flowsIDPUT(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "flowsIDPUT",
		"request_address": c.ClientIP,
	})

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)
	log = log.WithFields(logrus.Fields{
		"agent": a,
	})

	// get id
	id := uuid.FromStringOrNil(c.Params.ByName("id"))
	log = log.WithField("flow_id", id)

	var req request.BodyFlowsIDPUT
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log.WithField("request", req).Debug("Executing flowsIDPUT.")

	f := &fmflow.Flow{
		ID:      id,
		Name:    req.Name,
		Detail:  req.Detail,
		Actions: req.Actions,
	}

	// update a flow
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.FlowUpdate(c.Request.Context(), &a, f)
	if err != nil {
		log.Errorf("Could not update the flow. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// flowsIDDELETE handles DELETE /flows/{id} request.
// It deletes a exist flow info.
//	@Summary		Delete a existing flow.
//	@Description	Delete a existing flow.
//	@Produce		json
//	@Param			id	query	string	true	"The flow's id"
//	@Success		200
//	@Router			/v1.0/flows/{id} [delete]
func flowsIDDELETE(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "flowsIDDELETE",
		"request_address": c.ClientIP,
	})

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)
	log = log.WithFields(logrus.Fields{
		"agent": a,
	})

	// get id
	id := uuid.FromStringOrNil(c.Params.ByName("id"))
	log = log.WithField("flow_id", id)
	log.Debug("Executing flowsIDDELETE.")

	// delete a flow
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.FlowDelete(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not delete the flow. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
