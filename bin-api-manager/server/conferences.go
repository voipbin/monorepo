package server

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	cmrecording "monorepo/bin-call-manager/models/recording"
	cfconference "monorepo/bin-conference-manager/models/conference"
	fmaction "monorepo/bin-flow-manager/models/action"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

func (h *server) GetConferences(c *gin.Context, params openapi_server.GetConferencesParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetConferences",
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
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 100
		log.Debugf("Invalid requested page size. Set to default. page_size: %d", pageSize)
	}

	pageToken := ""
	if params.PageToken != nil {
		pageToken = *params.PageToken
	}

	tmps, err := h.serviceHandler.ConferenceGets(c.Request.Context(), &a, pageSize, pageToken)
	if err != nil {
		logrus.Errorf("Could not get conferences. err: %v", err)
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

func (h *server) PostConferences(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostConferences",
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

	var req openapi_server.PostConferencesJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	preActions := []fmaction.Action{}
	for _, v := range req.PreActions {
		preActions = append(preActions, ConvertFlowManagerAction(v))
	}

	postActions := []fmaction.Action{}
	for _, v := range req.PostActions {
		postActions = append(postActions, ConvertFlowManagerAction(v))
	}

	res, err := h.serviceHandler.ConferenceCreate(
		c.Request.Context(),
		&a,
		cfconference.Type(req.Type),
		req.Name,
		req.Detail,
		req.Timeout,
		req.Data,
		preActions,
		postActions,
	)
	if err != nil || res == nil {
		log.Errorf("Could not create the conference. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) GetConferencesId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetConferencesId",
		"request_address": c.ClientIP,
		"conference_id":   id,
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

	res, err := h.serviceHandler.ConferenceGet(c.Request.Context(), &a, target)
	if err != nil || res == nil {
		log.Errorf("Could not get the conference. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutConferencesId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "conferencesIDPUT",
		"request_address": c.ClientIP,
		"conference_id":   id,
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

	var req openapi_server.PutConferencesIdJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	preActions := []fmaction.Action{}
	for _, v := range req.PreActions {
		preActions = append(preActions, ConvertFlowManagerAction(v))
	}

	postActions := []fmaction.Action{}
	for _, v := range req.PostActions {
		postActions = append(postActions, ConvertFlowManagerAction(v))
	}

	res, err := h.serviceHandler.ConferenceUpdate(c.Request.Context(), &a, target, req.Name, req.Detail, req.Timeout, preActions, postActions)
	if err != nil || res == nil {
		log.Errorf("Could not update the conference. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) DeleteConferencesId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteConferencesId",
		"request_address": c.ClientIP,
		"conference_id":   id,
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

	res, err := h.serviceHandler.ConferenceDelete(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not delete the conference. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PostConferencesIdRecordingStart(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostConferencesIdRecordingStart",
		"request_address": c.ClientIP,
		"conference_id":   id,
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

	var req openapi_server.PostConferencesIdRecordingStartJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	onEndFlowID := uuid.FromStringOrNil(req.OnEndFlowId)

	res, err := h.serviceHandler.ConferenceRecordingStart(c.Request.Context(), &a, target, cmrecording.Format(req.Format), req.Duration, onEndFlowID)
	if err != nil {
		log.Errorf("Could not start the conference recording. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PostConferencesIdRecordingStop(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostConferencesIdRecordingStop",
		"request_address": c.ClientIP,
		"conference_id":   id,
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

	res, err := h.serviceHandler.ConferenceRecordingStop(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not stop the conference recording. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PostConferencesIdTranscribeStart(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostConferencesIdTranscribeStart",
		"request_address": c.ClientIP,
		"conference_id":   id,
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

	var req openapi_server.PostConferencesIdTranscribeStartJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.ConferenceTranscribeStart(c.Request.Context(), &a, target, req.Language)
	if err != nil {
		log.Errorf("Could not start the conference recording. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PostConferencesIdTranscribeStop(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostConferencesIdTranscribeStop",
		"request_address": c.ClientIP,
		"conference_id":   id,
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

	res, err := h.serviceHandler.ConferenceTranscribeStop(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not stop the conference transcribe. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) GetConferencesIdMediaStream(c *gin.Context, id string, params openapi_server.GetConferencesIdMediaStreamParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetConferencesIdMediaStream",
		"request_address": c.ClientIP,
		"conference_id":   id,
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

	if errMedia := h.serviceHandler.ConferenceMediaStreamStart(c.Request.Context(), &a, target, params.Encapsulation, c.Writer, c.Request); errMedia != nil {
		log.Errorf("Could not start the conference media streaming. err: %v", errMedia)
		c.AbortWithStatus(400)
		return
	}

	c.AbortWithStatus(200)
}
