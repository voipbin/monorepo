package calls

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

// callsPOST handles POST /calls request.
// It creates a temp flow and create a call with temp flow.
//	@Summary		Make an outbound call
//	@Description	dialing to destination
//	@Produce		json
//	@Param			call	body		request.BodyCallsPOST	true	"The call detail"
//	@Success		200		{object}	call.Call
//	@Router			/v1.0/calls [post]
func callsPOST(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "callsPOST",
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

	var req request.BodyCallsPOST
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// create call
	tmpCalls, tmpGroupcalls, err := serviceHandler.CallCreate(c.Request.Context(), &a, req.FlowID, req.Actions, &req.Source, req.Destinations)
	if err != nil {
		log.Errorf("Could not create a call for outgoing. err; %v", err)
		c.AbortWithStatus(400)
		return
	}

	res := &response.BodyCallsPOST{
		Calls:      tmpCalls,
		Groupcalls: tmpGroupcalls,
	}

	c.JSON(200, res)
}

// callsIDDelete handles DELETE /calls/<call-id> request.
// It deletes the call.
//	@Summary		Hangup the call
//	@Description	Hangup the call of the given id
//	@Produce		json
//	@Param			id	path		string	true	"The ID of the call"
//	@Success		200	{object}	call.Call
//	@Router			/v1.0/calls/{id} [delete]
func callsIDDelete(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "callsIDDelete",
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
	log.Debug("Executing callsIDDelete.")

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// hangup the call
	res, err := serviceHandler.CallDelete(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not hangup the call. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// callsGET handles GET /calls request.
// It returns list of calls of the given customer.

//	@Summary		Get list of calls
//	@Description	get calls of the customer
//	@Produce		json
//	@Param			page_size	query		int		false	"The size of results. Max 100"
//	@Param			page_token	query		string	false	"The token. tm_create"
//	@Success		200			{object}	response.BodyCallsGET
//	@Router			/v1.0/calls [get]
func callsGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "callsGET",
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

	var requestParam request.ParamCallsGET
	if err := c.BindQuery(&requestParam); err != nil {
		log.Errorf("Could not parse the reqeust parameter. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log.Debugf("callsGET. Received request detail. page_size: %d, page_token: %s", requestParam.PageSize, requestParam.PageToken)

	// set max page size
	pageSize := requestParam.PageSize
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 10
		log.Debugf("Invalid requested page size. Set to default. page_size: %d", pageSize)
	}

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// get calls
	calls, err := serviceHandler.CallGets(c.Request.Context(), &a, pageSize, requestParam.PageToken)
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
//	@Summary		Get detail call info.
//	@Description	Returns detail call info of the given call id.
//	@Produce		json
//	@Param			id	path		string	true	"The ID of the call"
//	@Success		200	{object}	call.Call
//	@Router			/v1.0/calls/{id} [get]
func callsIDGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "callsIDGET",
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
	log = log.WithField("call_id", id)
	log.Debug("Executing callsIDGET.")

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.CallGet(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not get a call. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// callsIDHangupPOST handles GET /calls/{id}/hangup request.
// It returns detail call info.
//	@Summary		Hangup the call.
//	@Description	Returns detail call info of the given call id.
//	@Produce		json
//	@Param			id	path		string	true	"The ID of the call"
//	@Success		200	{object}	call.Call
//	@Router			/v1.0/calls/{id}/hangup [post]
func callsIDHangupPOST(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "callsIDHangupPOST",
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
	log = log.WithField("call_id", id)
	log.Debug("Executing callsIDHangupPOST.")

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.CallHangup(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not get a call. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// callsIDTalkPOST handles GET /calls/{id}/talk request.
// It talks to the call.
//	@Summary		Talk to the call.
//	@Description	Talks to the call.
//	@Produce		json
//	@Param			id	path	string	true	"The ID of the call"
//	@Success		200
//	@Router			/v1.0/calls/{id}/talk [post]
func callsIDTalkPOST(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "callsIDTalkPOST",
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
	log = log.WithField("call_id", id)
	log.Debug("Executing callsIDTalkPOST.")

	var req request.BodyCallsIDTalkPOST
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the reqeust parameter. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	if err := serviceHandler.CallTalk(c.Request.Context(), &a, id, req.Text, req.Gender, req.Language); err != nil {
		log.Errorf("Could not talk to the call. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.AbortWithStatus(200)
}

// callsIDHoldPOST handles GET /calls/{id}/hold request.
// It holds the call.
//	@Summary		Hold the call.
//	@Description	Hold the call.
//	@Produce		json
//	@Param			id	path	string	true	"The ID of the call"
//	@Success		200
//	@Router			/v1.0/calls/{id}/hold [post]
func callsIDHoldPOST(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "callsIDHoldPOST",
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
	log = log.WithField("call_id", id)
	log.Debug("Executing callsIDHoldPOST.")

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	if err := serviceHandler.CallHoldOn(c.Request.Context(), &a, id); err != nil {
		log.Errorf("Could not hold the call. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.AbortWithStatus(200)
}

// callsIDHoldDELETE handles DELETE /calls/{id}/hold request.
// It unholds the call.
//	@Summary		Unhold the call.
//	@Description	Unhold the call.
//	@Produce		json
//	@Param			id	path	string	true	"The ID of the call"
//	@Success		200
//	@Router			/v1.0/calls/{id}/hold [delete]
func callsIDHoldDELETE(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "callsIDHoldDELETE",
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
	log = log.WithField("call_id", id)
	log.Debug("Executing callsIDUnholdPOST.")

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	if err := serviceHandler.CallHoldOff(c.Request.Context(), &a, id); err != nil {
		log.Errorf("Could not unhold the call. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.AbortWithStatus(200)
}

// callsIDMutePOST handles POST /calls/{id}/mute request.
// It mutes the call.
//	@Summary		Mute the call.
//	@Description	Mute the call.
//	@Produce		json
//	@Param			id	path	string	true	"The ID of the call"
//	@Success		200
//	@Router			/v1.0/calls/{id}/mute [post]
func callsIDMutePOST(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "callsIDMutePOST",
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
	log = log.WithField("call_id", id)

	var req request.BodyCallsIDMutePost
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	log.Debug("Executing callsIDMutePOST.")

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	if err := serviceHandler.CallMuteOn(c.Request.Context(), &a, id, req.Direction); err != nil {
		log.Errorf("Could not mute the call. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.AbortWithStatus(200)
}

// callsIDMuteDELETE handles DELETE /calls/{id}/mute request.
// It unmutes the call.
//	@Summary		Unmute the call.
//	@Description	Unmute the call.
//	@Produce		json
//	@Param			id	path	string	true	"The ID of the call"
//	@Success		200
//	@Router			/v1.0/calls/{id}/mute [delete]
func callsIDMuteDELETE(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "callsIDMuteDELETE",
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
	log = log.WithField("call_id", id)

	var req request.BodyCallsIDMuteDelete
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	log.Debug("Executing callsIDMuteDELETE.")

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	if err := serviceHandler.CallMuteOff(c.Request.Context(), &a, id, req.Direction); err != nil {
		log.Errorf("Could not unmute the call. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.AbortWithStatus(200)
}

// callsIDMOHPOST handles POST /calls/{id}/moh request.
// It moh the call.
//	@Summary		MOH the call.
//	@Description	MOH the call.
//	@Produce		json
//	@Param			id	path	string	true	"The ID of the call"
//	@Success		200
//	@Router			/v1.0/calls/{id}/moh [post]
func callsIDMOHPOST(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "callsIDMOHPOST",
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
	log = log.WithField("call_id", id)
	log.Debug("Executing callsIDMOHPOST.")

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	if err := serviceHandler.CallMOHOn(c.Request.Context(), &a, id); err != nil {
		log.Errorf("Could not moh on the call. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.AbortWithStatus(200)
}

// callsIDMOHDELETE handles DELETE /calls/{id}/moh request.
// It moh the call.
//	@Summary		MOH off the call.
//	@Description	MOH off the call.
//	@Produce		json
//	@Param			id	path	string	true	"The ID of the call"
//	@Success		200
//	@Router			/v1.0/calls/{id}/moh [delete]
func callsIDMOHDELETE(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "callsIDMOHDELETE",
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
	log = log.WithField("call_id", id)
	log.Debug("Executing callsIDMOHDELETE.")

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	if err := serviceHandler.CallMOHOff(c.Request.Context(), &a, id); err != nil {
		log.Errorf("Could not moh off the call. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.AbortWithStatus(200)
}

// callsIDSilencePOST handles POST /calls/{id}/silence request.
// It silence the call.
//	@Summary		Silence the call.
//	@Description	Silence the call.
//	@Produce		json
//	@Param			id	path	string	true	"The ID of the call"
//	@Success		200
//	@Router			/v1.0/calls/{id}/silence [post]
func callsIDSilencePOST(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "callsIDSilencePOST",
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
	log = log.WithField("call_id", id)
	log.Debug("Executing callsIDSilencePOST.")

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	if err := serviceHandler.CallSilenceOn(c.Request.Context(), &a, id); err != nil {
		log.Errorf("Could not moh on the call. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.AbortWithStatus(200)
}

// callsIDSilenceDELETE handles DELETE /calls/{id}/silence request.
// It silence off the call.
//	@Summary		Silence off the call.
//	@Description	Silence off the call.
//	@Produce		json
//	@Param			id	path	string	true	"The ID of the call"
//	@Success		200
//	@Router			/v1.0/calls/{id}/silence [delete]
func callsIDSilenceDELETE(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "callsIDSilenceDELETE",
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
	log = log.WithField("call_id", id)
	log.Debug("Executing callsIDSilenceDELETE.")

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	if err := serviceHandler.CallSilenceOff(c.Request.Context(), &a, id); err != nil {
		log.Errorf("Could not moh off the call. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.AbortWithStatus(200)
}

// callsIDMediaStreamGET handles GET /calls/{id}/media_stream request.
// It starts the in/out media streaming of the call.
//	@Summary		Start the call media streaming.
//	@Description	Start the call media streaming.
//	@Produce		json
//	@Param			id	path	string	true	"The ID of the call"
//	@Success		200
//	@Router			/v1.0/calls/{id}/meida_stream [get]
func callsIDMediaStreamGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "callsIDMediaStreamGET",
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
	log = log.WithField("call_id", id)

	var requestParam request.ParamCallsIDMediaStreamGET
	if err := c.BindQuery(&requestParam); err != nil {
		log.Errorf("Could not parse the reqeust parameter. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log.WithField("parameter", requestParam).Debugf("callsIDMediaStreamGET. Received request detail. id: %S", id)

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	if err := serviceHandler.CallMediaStreamStart(c.Request.Context(), &a, id, requestParam.Encapsulation, c.Writer, c.Request); err != nil {
		log.Errorf("Could not start the call media streaming. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.AbortWithStatus(200)
}
