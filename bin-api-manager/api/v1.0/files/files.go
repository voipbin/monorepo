package files

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	"net/http"

	"monorepo/bin-api-manager/api/models/common"
	"monorepo/bin-api-manager/api/models/request"
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

	// set limit for max file sizw. 30M
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, int64(30<<20))

	f, header, err := c.Request.FormFile("file")
	if err != nil {
		log.Errorf("Could not get file. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log.WithField("file", header).Debugf("Checking uploaded file header. filename: %s", header.Filename)

	// f, err := c.FormFile("file")
	// if err != nil {
	// 	log.Errorf("Could not get file. err: %v", err)
	// 	c.AbortWithStatus(400)
	// 	return
	// }

	// if f.Size == 0 {
	// 	// no file
	// 	log.Errorf("Invalid file size. size: %d", f.Size)
	// 	c.AbortWithStatus(400)
	// 	return
	// }

	var req request.BodyFilesPOST
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// create call
	res, err := serviceHandler.FileCreate(c.Request.Context(), &a, f, req.Name, req.Detail)
	if err != nil {
		log.Errorf("Could not create a call for outgoing. err; %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
