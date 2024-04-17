package transcripts

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

// transcriptsGET handles GET /transcripts request.
// It returns list of transcripts of the given customer.
//	@Summary		Get list of transcripts
//	@Description	get transcripts of the customer
//	@Produce		json
//	@Param			page_size	query		int		false	"The size of results. Max 100"
//	@Param			page_token	query		string	false	"The token. tm_create"
//	@Success		200			{object}	response.BodyTranscribesGET
//	@Router			/v1.0/transcripts [get]
func transcriptsGET(c *gin.Context) {
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

	var requestParam request.ParamTranscriptsGET
	if err := c.BindQuery(&requestParam); err != nil {
		log.Errorf("Could not parse the reqeust parameter. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log.Debugf("Received request detail. transcribe_id: %s, page_size: %d, page_token: %s", requestParam.TranscribeID, requestParam.PageSize, requestParam.PageToken)

	id := uuid.FromStringOrNil(requestParam.TranscribeID)

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// get tmps
	tmps, err := serviceHandler.TranscriptGets(c.Request.Context(), &a, id)
	if err != nil {
		logrus.Errorf("Could not get transcribes info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(tmps) > 0 {
		nextToken = tmps[len(tmps)-1].TMCreate
	}
	res := response.BodyTranscriptsGET{
		Result: tmps,
		Pagination: response.Pagination{
			NextPageToken: nextToken,
		},
	}

	c.JSON(200, res)
}
