package server

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	cmcall "monorepo/bin-call-manager/models/call"
	cmgroupcall "monorepo/bin-call-manager/models/groupcall"
	commonaddress "monorepo/bin-common-handler/models/address"
	fmaction "monorepo/bin-flow-manager/models/action"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

func (h *server) PostCalls(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostCalls",
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

	var req openapi_server.PostCallsJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	flowID := uuid.Nil
	if req.FlowId != nil {
		flowID = uuid.FromStringOrNil(*req.FlowId)
	}

	actions := []fmaction.Action{}
	if req.Actions != nil {
		for _, v := range *req.Actions {
			actions = append(actions, ConvertFlowManagerAction(v))
		}
	}

	source := commonaddress.Address{}
	if req.Source != nil {
		source = ConvertCommonAddress(*req.Source)
	}

	destinations := []commonaddress.Address{}
	if req.Destinations != nil {

		for _, v := range *req.Destinations {
			destinations = append(destinations, ConvertCommonAddress(v))
		}
	}

	tmpCalls, tmpGroupcalls, err := h.serviceHandler.CallCreate(c.Request.Context(), &a, flowID, actions, &source, destinations)
	if err != nil {
		log.Errorf("Could not create a call for outgoing. err; %v", err)
		c.AbortWithStatus(400)
		return
	}

	res := struct {
		Calls      []*cmcall.WebhookMessage      `json:"calls"`
		Groupcalls []*cmgroupcall.WebhookMessage `json:"groupcalls"`
	}{
		Calls:      tmpCalls,
		Groupcalls: tmpGroupcalls,
	}

	c.JSON(200, res)
}

func (h *server) DeleteCallsId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteCallsId",
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

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.CallDelete(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not delete the call. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) GetCalls(c *gin.Context, params openapi_server.GetCallsParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetCalls",
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

	pageSize := uint64(100)
	if params.PageSize != nil {
		pageSize = uint64(*params.PageSize)
	}

	pageToken := ""
	if params.PageToken != nil {
		pageToken = *params.PageToken
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 100
		log.Debugf("Invalid requested page size. Set to default. page_size: %d", pageSize)
	}

	tmps, err := h.serviceHandler.CallGets(c.Request.Context(), &a, pageSize, pageToken)
	if err != nil {
		logrus.Errorf("Could not get calls info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(tmps) > 0 {
		nextToken = tmps[len(tmps)-1].TMCreate
	}

	res := GenerateListResponse(tmps, nextToken)
	c.JSON(200, res)
}

func (h *server) GetCallsId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetCallsId",
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

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.CallGet(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not get a call. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PostCallsIdHangup(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostCallsIdHangup",
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

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.CallHangup(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not get a call. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PostCallsIdTalk(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostCallsIdTalk",
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

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	var req openapi_server.PostCallsIdTalkJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	text := ""
	if req.Text != nil {
		text = *req.Text
	}

	gender := ""
	if req.Gender != nil {
		gender = *req.Gender
	}

	language := ""
	if req.Language != nil {
		language = *req.Language
	}

	if errTalk := h.serviceHandler.CallTalk(c.Request.Context(), &a, target, text, gender, language); errTalk != nil {
		log.Errorf("Could not talk to the call. err: %v", errTalk)
		c.AbortWithStatus(400)
		return
	}

	c.AbortWithStatus(200)
}

func (h *server) PostCallsIdHold(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostCallsIdHold",
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

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	if err := h.serviceHandler.CallHoldOn(c.Request.Context(), &a, target); err != nil {
		log.Errorf("Could not hold the call. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.AbortWithStatus(200)
}

func (h *server) DeleteCallsIdHold(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteCallsIdHold",
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

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	if errHold := h.serviceHandler.CallHoldOff(c.Request.Context(), &a, target); errHold != nil {
		log.Errorf("Could not unhold the call. err: %v", errHold)
		c.AbortWithStatus(400)
		return
	}

	c.AbortWithStatus(200)
}

func (h *server) PostCallsIdMute(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostCallsIdMute",
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

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	var req openapi_server.PostCallsIdMuteJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the reqeust parameter. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	direction := cmcall.MuteDirectionNone
	if req.Direction != nil {
		direction = cmcall.MuteDirection(*req.Direction)
	}

	if errMute := h.serviceHandler.CallMuteOn(c.Request.Context(), &a, target, direction); errMute != nil {
		log.Errorf("Could not mute the call. err: %v", errMute)
		c.AbortWithStatus(400)
		return
	}

	c.AbortWithStatus(200)
}

func (h *server) DeleteCallsIdMute(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteCallsIdMute",
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

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	var req openapi_server.DeleteCallsIdMuteJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the reqeust parameter. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	direction := cmcall.MuteDirectionNone
	if req.Direction != nil {
		direction = cmcall.MuteDirection(*req.Direction)
	}

	if errMute := h.serviceHandler.CallMuteOff(c.Request.Context(), &a, target, direction); errMute != nil {
		log.Errorf("Could not unmute the call. err: %v", errMute)
		c.AbortWithStatus(400)
		return
	}

	c.AbortWithStatus(200)
}

func (h *server) PostCallsIdMoh(c *gin.Context, id string) {
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

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	if errMoh := h.serviceHandler.CallMOHOn(c.Request.Context(), &a, target); errMoh != nil {
		log.Errorf("Could not moh on the call. err: %v", errMoh)
		c.AbortWithStatus(400)
		return
	}

	c.AbortWithStatus(200)
}

func (h *server) DeleteCallsIdMoh(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteCallsIdMoh",
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

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	if errMoh := h.serviceHandler.CallMOHOff(c.Request.Context(), &a, target); errMoh != nil {
		log.Errorf("Could not moh off the call. err: %v", errMoh)
		c.AbortWithStatus(400)
		return
	}

	c.AbortWithStatus(200)
}

func (h *server) PostCallsIdSilence(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostCallsIdSilence",
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

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	if errSilence := h.serviceHandler.CallSilenceOn(c.Request.Context(), &a, target); errSilence != nil {
		log.Errorf("Could not silence on the call. err: %v", errSilence)
		c.AbortWithStatus(400)
		return
	}

	c.AbortWithStatus(200)
}

func (h *server) DeleteCallsIdSilence(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteCallsIdSilence",
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

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	if errSilence := h.serviceHandler.CallSilenceOff(c.Request.Context(), &a, target); errSilence != nil {
		log.Errorf("Could not silence off the call. err: %v", errSilence)
		c.AbortWithStatus(400)
		return
	}

	c.AbortWithStatus(200)
}

func (h *server) GetCallsIdMediaStream(c *gin.Context, id string, params openapi_server.GetCallsIdMediaStreamParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetCallsIdMediaStream",
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

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	encapsulation := ""
	if params.Encapsulation != nil {
		encapsulation = *params.Encapsulation
	}

	if err := h.serviceHandler.CallMediaStreamStart(c.Request.Context(), &a, target, encapsulation, c.Writer, c.Request); err != nil {
		log.Errorf("Could not start the call media streaming. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.AbortWithStatus(200)
}
