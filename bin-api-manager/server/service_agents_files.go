package server

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

func (h *server) PostServiceAgentsFiles(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostServiceAgentsFiles",
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

	// set limit for max file siz
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, constMaxFileSize)

	f, header, err := c.Request.FormFile("file")
	if err != nil {
		log.Errorf("Could not get file. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log.WithField("file", header).Debugf("Checking uploaded file header. filename: %s", header.Filename)

	res, err := h.serviceHandler.ServiceAgentFileCreate(c.Request.Context(), &a, f, "", "", header.Filename)
	if err != nil {
		log.Errorf("Could not upload the file. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) GetServiceAgentsFiles(c *gin.Context, params openapi_server.GetServiceAgentsFilesParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "filesGET",
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

	tmps, err := h.serviceHandler.ServiceAgentFileGets(c.Request.Context(), &a, pageSize, pageToken)
	if err != nil {
		log.Errorf("Could not get a file list. err: %v", err)
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

func (h *server) GetServiceAgentsFilesId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetServiceAgentsFilesId",
		"request_address": c.ClientIP,
		"file_id":         id,
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

	res, err := h.serviceHandler.ServiceAgentFileGet(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not get a file. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) DeleteServiceAgentsFilesId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteServiceAgentsFilesId",
		"request_address": c.ClientIP,
		"file_id":         id,
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

	res, err := h.serviceHandler.ServiceAgentFileDelete(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not delete the file. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
