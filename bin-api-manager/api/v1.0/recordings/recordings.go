package recordings

import (
	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-api-manager/api/models/common"
	"monorepo/bin-api-manager/api/models/request"
	"monorepo/bin-api-manager/api/models/response"
	"monorepo/bin-api-manager/pkg/servicehandler"
)

// recordingsGET handles GET /recordings request.
// It returns list of calls of the given customer.
//
//	@Summary		List recordings
//	@Description	get recordings of the customer
//	@Produce		json
//	@Param			page_size	query		int		false	"The size of results. Max 100"
//	@Param			page_token	query		string	false	"The token. tm_create"
//	@Success		200			{object}	response.BodyRecordingsGET
//	@Router			/v1.0/recordings [get]
func recordingsGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "recordingsGET",
		"request_address": c.ClientIP,
	})

	tmpAgent, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmpAgent.(amagent.Agent)
	log = log.WithFields(logrus.Fields{
		"agent": a,
	})

	var requestParam request.ParamRecordingsGET
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
	log.Debugf("recordingsGET. Received request detail. page_size: %d, page_token: %s", pageSize, requestParam.PageToken)

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// get recordings
	recordings, err := serviceHandler.RecordingGets(c.Request.Context(), &a, pageSize, requestParam.PageToken)
	if err != nil {
		logrus.Errorf("Could not get a recordings. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(recordings) > 0 {
		nextToken = recordings[len(recordings)-1].TMCreate
	}
	res := response.BodyRecordingsGET{
		Result: recordings,
		Pagination: response.Pagination{
			NextPageToken: nextToken,
		},
	}

	c.JSON(200, res)
}

// recordingsIDGET handles GET /recordings/<id> request.
// It returns a detail recording info.
//
//	@Summary		Returns a detail recording information.
//	@Description	Returns a detial recording information of the given recording id.
//	@Produce		json
//	@Param			id	query		string	true	"The recording's id."
//	@Success		200	{object}	recording.Recording
//	@Router			/v1.0/recordings/{id} [get]
func recordingsIDGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "recordingsIDGET",
		"request_address": c.ClientIP,
	})

	tmpAgent, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmpAgent.(amagent.Agent)
	log = log.WithFields(logrus.Fields{
		"agent": a,
	})

	// get id
	id := uuid.FromStringOrNil(c.Params.ByName("id"))
	log = log.WithField("recording_id", id)
	log.Debug("Executing recordingsIDGET.")

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.RecordingGet(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not get a recording info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// recordingsIDDELETE handles DELETE /recordings/<id> request.
// It deletes a recording info.
//
//	@Summary		Deletes a recording and returns a deleted recording information.
//	@Description	Deletes a recording and returns a deleted recording information of the given recording id.
//	@Produce		json
//	@Param			id	query		string	true	"The recording's id."
//	@Success		200	{object}	recording.Recording
//	@Router			/v1.0/recordings/{id} [delete]
func recordingsIDDELETE(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "recordingsIDDELETE",
		"request_address": c.ClientIP,
	})

	tmpAgent, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmpAgent.(amagent.Agent)
	log = log.WithFields(logrus.Fields{
		"agent": a,
	})

	// get id
	id := uuid.FromStringOrNil(c.Params.ByName("id"))
	log = log.WithField("recording_id", id)
	log.Debug("Executing recordingsIDDELETE.")

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.RecordingDelete(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not get a recording info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
