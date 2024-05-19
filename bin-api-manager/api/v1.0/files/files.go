package files

import (
	"fmt"
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-api-manager/api/models/common"
	"monorepo/bin-api-manager/api/models/request"
	"monorepo/bin-api-manager/api/models/response"
	"monorepo/bin-api-manager/pkg/servicehandler"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// filesPOST handles POST /files request.
// It creates a temp flow and create a call with temp flow.
//
//	@Summary		Make an outbound call
//	@Description	dialing to destination
//	@Produce		json
//	@Param			call	body		request.BodyCallsPOST	true	"The call detail"
//	@Success		200		{object}	call.Call
//	@Router			/v1.0/calls [post]
func filesPOST(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "filesPOST",
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

	f, err := c.FormFile("file")
	if err != nil {
		log.Errorf("Could not get file. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	// save the file
	tmpFilepath := fmt.Sprintf("/tmp/%s", utilhandler.UUIDCreate())
	if errSave := c.SaveUploadedFile(f, tmpFilepath); errSave != nil {
		log.Errorf("Could not save uploaded file. err: %v", errSave)
		c.AbortWithStatus(400)
		return
	}

	var req request.BodyFilesPOST
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
