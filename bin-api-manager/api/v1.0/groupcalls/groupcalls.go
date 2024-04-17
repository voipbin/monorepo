package groupcalls

import (
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/request"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/response"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

// groupcallsPOST handles POST /groupcalls request.
// It creates a new groupcall with the given info and returns created groupcall info.
//	@Summary		Create a new groupcall and returns detail created groupcall info.
//	@Description	Create a new groupcall and returns detail created groupcall info.
//	@Produce		json
//	@Param			flow	body		request.BodyGroupcallsPOST	true	"groupcall info."
//	@Success		200		{object}	groupcall.Groupcall
//	@Router			/v1.0/groupcalls [post]
func groupcallsPOST(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "groupcallsPOST",
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

	var req request.BodyGroupcallsPOST
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	// create a groupcall
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.GroupcallCreate(c.Request.Context(), &a, req.Source, req.Destinations, req.FlowID, req.Actions, req.RingMethod, req.AnswerMethod)
	if err != nil {
		log.Errorf("Could not create a groupcall. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// groupcallsGET handles GET /groupcalls request.
// It gets a list of groupcalls with the given info.
//	@Summary		Gets a list of groupcalls.
//	@Description	Gets a list of groupcalls
//	@Produce		json
//	@Param			page_size	query		int		false	"The size of results. Max 100"
//	@Param			page_token	query		string	false	"The token. tm_create"
//	@Success		200			{object}	response.BodyGroupcallsGET
//	@Router			/v1.0/groupcalls [get]
func groupcallsGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "groupcallsGET",
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

	var req request.ParamGroupcallsGET
	if err := c.BindQuery(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	// set max page size
	pageSize := req.PageSize
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 10
		log.Debugf("Invalid requested page size. Set to default. page_size: %d", pageSize)
	}
	log.Debugf("Received request detail. page_size: %d, page_token: %s", pageSize, req.PageToken)

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// get groupcalls
	tmps, err := serviceHandler.GroupcallGets(c.Request.Context(), &a, pageSize, req.PageToken)
	if err != nil {
		log.Errorf("Could not get a groupcall list. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(tmps) > 0 {
		nextToken = tmps[len(tmps)-1].TMCreate
	}
	res := response.BodyGroupcallsGET{
		Result: tmps,
		Pagination: response.Pagination{
			NextPageToken: nextToken,
		},
	}

	c.JSON(200, res)
}

// groupcallsIDGET handles GET /groupcalls/{id} request.
// It returns detail groupcall info.
//	@Summary		Returns detail groupcall info.
//	@Description	Returns detail groupcall info of the given groupcall id.
//	@Produce		json
//	@Param			id	path		string	true	"The ID of the groupcall"
//	@Success		200	{object}	groupcall.Groupcall
//	@Router			/v1.0/groupcalls/{id} [get]
func groupcallsIDGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "groupcallsIDGET",
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
	log = log.WithField("groupcall_id", id)

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.GroupcallGet(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not get a groupcall. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// groupcallsIDHangupPOST handles POST /groupcalls/{id}/hangup request.
// It hangup the groupcall.
//	@Summary		Hangup the groupcall.
//	@Description	Hangup the groupcall of the given groupcall id.
//	@Produce		json
//	@Param			id	path		string	true	"The ID of the groupcall"
//	@Success		200	{object}	groupcall.Groupcall
//	@Router			/v1.0/groupcalls/{id}/hangup [post]
func groupcallsIDHangupPOST(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "groupcallsIDHangupPOST",
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
	log = log.WithField("groupcall_id", id)

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.GroupcallHangup(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not hangup the groupcall. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// groupcallsIDDELETE handles DELETE /groupcalls/{id} request.
// It deletes the groupcall.
//	@Summary		Delete the groupcall.
//	@Description	Delete the groupcall of the given groupcall id.
//	@Produce		json
//	@Param			id	path		string	true	"The ID of the groupcall"
//	@Success		200	{object}	groupcall.Groupcall
//	@Router			/v1.0/groupcalls/{id} [delete]
func groupcallsIDDELETE(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "groupcallsIDDELETE",
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
	log = log.WithField("groupcall_id", id)

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.GroupcallDelete(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not delete the groupcall. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
