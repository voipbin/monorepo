package transcribes

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

// transcribesPOST handles POST /transcribes request.
// It creates a transcribe of the recording and returns the result.
//
//	@Summary		Create a transcribe
//	@Description	transcribe a recording
//	@Produce		json
//	@Param			transcribe	body		request.BodyTranscribesPOST	true	"Creating transcribe info."
//	@Success		200			{object}	transcribe.Transcribe
//	@Router			/v1.0/transcribes [post]
func transcribesPOST(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "transcribesPOST",
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

	var req request.BodyTranscribesPOST
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log.WithField("request", req).Debug("Executing transcribesPOST.")

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// create a transcribe
	res, err := serviceHandler.TranscribeStart(c.Request.Context(), &a, req.ReferenceType, req.ReferenceID, req.Language, req.Direction)
	if err != nil {
		log.Errorf("Could not create a transcribe. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// transcribesGET handles GET /transcribes request.
// It returns list of transcribes of the given customer.
//
//	@Summary		Get list of transcribes
//	@Description	get transcribes of the customer
//	@Produce		json
//	@Param			page_size	query		int		false	"The size of results. Max 100"
//	@Param			page_token	query		string	false	"The token. tm_create"
//	@Success		200			{object}	response.BodyTranscribesGET
//	@Router			/v1.0/transcribes [get]
func transcribesGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "transcribesGET",
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

	var requestParam request.ParamTranscribesGET
	if err := c.BindQuery(&requestParam); err != nil {
		log.Errorf("Could not parse the reqeust parameter. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log.Debugf("Received request detail. page_size: %d, page_token: %s", requestParam.PageSize, requestParam.PageToken)

	// set max page size
	pageSize := requestParam.PageSize
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 10
		log.Debugf("Invalid requested page size. Set to default. page_size: %d", pageSize)
	}

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// get tmps
	tmps, err := serviceHandler.TranscribeGets(c.Request.Context(), &a, pageSize, requestParam.PageToken)
	if err != nil {
		logrus.Errorf("Could not get transcribes info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(tmps) > 0 {
		nextToken = tmps[len(tmps)-1].TMCreate
	}
	res := response.BodyTranscribesGET{
		Result: tmps,
		Pagination: response.Pagination{
			NextPageToken: nextToken,
		},
	}

	c.JSON(200, res)
}

// transcribesIDGET handles GET /transcribes/{id} request.
// It returns detail transcribe info.
//
//	@Summary		Get detail transcribe info.
//	@Description	Returns detail transcribe info of the given transcribe id.
//	@Produce		json
//	@Param			id	path		string	true	"The ID of the transcribe"
//	@Success		200	{object}	transcribe.Transcribe
//	@Router			/v1.0/transcribe/{id} [get]
func transcribesIDGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "transcribesIDGET",
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
	log = log.WithField("transcribe_id", id)
	log.Debug("Executing transcribesIDGET.")

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.TranscribeGet(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not get a transcribe. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// transcribesIDDelete handles DELETE /transcribes/<transcribe-id> request.
// It deletes the transcribe.

// @Summary		Delete the transcribe.
// @Description	Delete the transcribe of the given id
// @Produce		json
// @Param			id	path		string	true	"The ID of the transcribe"
// @Success		200	{object}	transcribe.Transcribe
// @Router			/v1.0/transcribes/{id} [delete]
func transcribesIDDelete(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "transcribesIDDelete",
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
	log.Debug("Executing transcribesIDDelete.")

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// delete
	res, err := serviceHandler.TranscribeDelete(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not delete the transcribe. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// transcribesIDStopPOST handles POST /transcribes/<transcribe-id>/stop request.
// It creates a transcribe of the recording and returns the result.
//
//	@Summary		Create a transcribe
//	@Description	transcribe a recording
//	@Produce		json
//	@Param			transcribe	body		request.BodyTranscribesPOST	true	"Creating transcribe info."
//	@Success		200			{object}	transcribe.Transcribe
//	@Router			/v1.0/transcribes/{id}/stop [post]
func transcribesIDStopPOST(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "transcribesIDStopPOST",
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
	log.Debug("Executing transcribesIDStopPOST.")

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// create a transcribe
	res, err := serviceHandler.TranscribeStop(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not stop the transcribe. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
