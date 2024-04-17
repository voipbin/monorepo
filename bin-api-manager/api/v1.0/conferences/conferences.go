package conferences

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

// conferencesGET handles GET /conferences request.
// It returns list of conferences of the given customer.
//
//	@Summary		Get list of conferences
//	@Description	get conferences of the customer
//	@Produce		json
//	@Param			page_size	query		int		false	"The size of results. Max 100"
//	@Param			page_token	query		string	false	"The token. tm_create"
//	@Success		200			{object}	response.BodyCallsGET
//	@Router			/v1.0/conferences [get]
func conferencesGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "conferencesGET",
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

	var requestParam request.ParamConferencesGET
	if err := c.BindQuery(&requestParam); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	// set max page size
	pageSize := requestParam.PageSize
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 10
		log.Debugf("Invalid requested page size. Set to default. page_size: %d", pageSize)
	}
	log.Debugf("conferencesGET. Received request detail. page_size: %d, page_token: %s", requestParam.PageSize, requestParam.PageToken)

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// get conferences
	confs, err := serviceHandler.ConferenceGets(c.Request.Context(), &a, pageSize, requestParam.PageToken)
	if err != nil {
		logrus.Errorf("Could not get conferences. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(confs) > 0 {
		nextToken = confs[len(confs)-1].TMCreate
	}

	res := response.BodyConferencesGET{
		Result: confs,
		Pagination: response.Pagination{
			NextPageToken: nextToken,
		},
	}

	c.JSON(200, res)
}

// conferencesPOST handles POST /conferences request.
// It creates a new conference and returns the created conferences of the given customer.
//
//	@Summary		Create a new conferences
//	@Description	Create a new conference with the given information.
//	@Produce		json
//	@Param			conference	body		request.BodyConferencesPOST	true	"The conference detail"
//	@Success		200			{object}	conference.Conference
//	@Router			/v1.0/conferences [post]
func conferencesPOST(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "conferencesPOST",
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

	var req request.BodyConferencesPOST
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	servicehandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := servicehandler.ConferenceCreate(
		c.Request.Context(),
		&a,
		req.Type,
		req.Name,
		req.Detail,
		req.Timeout,
		req.Data,
		req.PreActions,
		req.PostActions,
	)
	if err != nil || res == nil {
		log.Errorf("Could not create the conference. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// conferencesIDGET handles GET /conferences/{id} request.
// It returns detail conference info.
//
//	@Summary		Returns detail conference info.
//	@Description	Returns detail conference info of the given conference id.
//	@Produce		json
//	@Param			id		path		string	true	"The ID of the conference"
//	@Param			token	query		string	true	"JWT token"
//	@Success		200		{object}	conference.Conference
//	@Router			/v1.0/conferences/{id} [get]
func conferencesIDGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "conferencesIDGET",
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
	log = log.WithField("conference_id", id)
	log.Debug("Executing conferencesIDGET.")

	servicehandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := servicehandler.ConferenceGet(c.Request.Context(), &a, id)
	if err != nil || res == nil {
		log.Errorf("Could not get the conference. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// conferencesIDPUT handles PUT /conferences/{id} request.
// It updates the conference and returns updated conference info.
//
//	@Summary		Update conference info.
//	@Description	Update conference info of the given conference id.
//	@Produce		json
//	@Param			id		path		string	true	"The ID of the conference"
//	@Param			token	query		string	true	"JWT token"
//	@Success		200		{object}	conference.Conference
//	@Router			/v1.0/conferences/{id} [put]
func conferencesIDPUT(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "conferencesIDPUT",
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
	log = log.WithField("conference_id", id)
	log.Debug("Executing conferencesIDPUT.")

	var req request.BodyConferencesIDPUT
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	servicehandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := servicehandler.ConferenceUpdate(c.Request.Context(), &a, id, req.Name, req.Detail, req.Tiemout, req.PreActions, req.PostActions)
	if err != nil || res == nil {
		log.Errorf("Could not update the conference. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// conferencesIDDELETE handles DELETE /conferences/{id} request.
// It deletes the conference.
//
//	@Summary		Delete the conference.
//	@Description	Delete the conference. All the participants in the conference will be kicked out.
//	@Produce		json
//	@Param			id	path	string	true	"The ID of the conference"
//	@Success		200
//	@Router			/v1.0/conferences/{id} [delete]
func conferencesIDDELETE(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "conferencesIDDELETE",
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
	log = log.WithField("conference_id", id)
	log.Debug("Executing conferencesIDDELETE.")

	servicehandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := servicehandler.ConferenceDelete(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not delete the conference. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// conferencesIDRecordingStartPOST handles DELETE /conferences/{id}/recording_start request.
// It starts the conference recording.
//
//	@Summary		Starts the conference recording.
//	@Description	Start the conference recording.
//	@Produce		json
//	@Param			id	path	string	true	"The ID of the conference"
//	@Success		200
//	@Router			/v1.0/conferences/{id}/recording_start [post]
func conferencesIDRecordingStartPOST(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "conferencesIDRecordingStartPOST",
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
	log = log.WithField("conference_id", id)
	log.Debug("Executing conferencesIDDELETE.")

	servicehandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := servicehandler.ConferenceRecordingStart(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not start the conference recording. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// conferencesIDRecordingStopPOST handles DELETE /conferences/{id}/recording_stop request.
// It stops the conference recording.
//
//	@Summary		Stops the conference recording.
//	@Description	Stops the conference recording.
//	@Produce		json
//	@Param			id	path	string	true	"The ID of the conference"
//	@Success		200
//	@Router			/v1.0/conferences/{id}/recording_stop [post]
func conferencesIDRecordingStopPOST(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "conferencesIDRecordingStopPOST",
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
	log = log.WithField("conference_id", id)
	log.Debug("Executing conferencesIDRecordingStopPOST.")

	servicehandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := servicehandler.ConferenceRecordingStop(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not stop the conference recording. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// conferencesIDTranscribeStartPOST handles DELETE /conferences/{id}/transcribe_start request.
// It starts the conference transcribe.
//
//	@Summary		Starts the conference transcribe.
//	@Description	Start the conference transcribe.
//	@Produce		json
//	@Param			id			path	string											true	"The ID of the conference"
//	@Param			conference	body	request.BodyConferencesIDTranscribeStartPOST	true	"conference info"
//	@Success		200
//	@Router			/v1.0/conferences/{id}/transcribe_start [post]
func conferencesIDTranscribeStartPOST(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "conferencesIDTranscribeStartPOST",
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
	log = log.WithField("conference_id", id)

	var req request.BodyConferencesIDTranscribeStartPOST
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	log.Debug("Executing conferencesIDTranscribeStartPOST.")

	servicehandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := servicehandler.ConferenceTranscribeStart(c.Request.Context(), &a, id, req.Language)
	if err != nil {
		log.Errorf("Could not start the conference recording. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// conferencesIDTranscribeStopPOST handles DELETE /conferences/{id}/transcribe_stop request.
// It stops the conference transcribe.
//
//	@Summary		Stops the conference transcribe.
//	@Description	Stops the conference transcribe.
//	@Produce		json
//	@Param			id	path	string	true	"The ID of the conference"
//	@Success		200
//	@Router			/v1.0/conferences/{id}/transcribe_stop [post]
func conferencesIDTranscribeStopPOST(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "conferencesIDTranscribeStopPOST",
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
	log = log.WithField("conference_id", id)
	log.Debug("Executing conferencesIDTranscribeStopPOST.")

	servicehandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := servicehandler.ConferenceTranscribeStop(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not stop the conference transcribe. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// conferencesIDMediaStreamGET handles GET /conferences/{id}/media_stream request.
// It starts the in/out media streaming of the call.
//
//	@Summary		Start the conference media streaming.
//	@Description	Start the conference media streaming.
//	@Produce		json
//	@Param			id	path	string	true	"The ID of the conference"
//	@Success		200
//	@Router			/v1.0/conferences/{id}/meida_stream [get]
func conferencesIDMediaStreamGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "conferencesIDMediaStreamGET",
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
	log = log.WithField("conference_id", id)

	var requestParam request.ParamConferencesIDMediaStreamGET
	if err := c.BindQuery(&requestParam); err != nil {
		log.Errorf("Could not parse the reqeust parameter. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log.WithField("parameter", requestParam).Debugf("conferencesIDMediaStreamGET. Received request detail. id: %S", id)

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	if err := serviceHandler.ConferenceMediaStreamStart(c.Request.Context(), &a, id, requestParam.Encapsulation, c.Writer, c.Request); err != nil {
		log.Errorf("Could not start the conference media streaming. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.AbortWithStatus(200)
}
