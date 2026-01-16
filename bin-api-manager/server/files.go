package server

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

const (
	constMaxFileSize = int64(30 << 20) // Max upload file size. 30 MB.
)

func (h *server) PostFiles(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostFiles",
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

	res, err := h.serviceHandler.StorageFileCreate(c.Request.Context(), &a, f, "", "", header.Filename)
	if err != nil {
		log.Errorf("Could not create a call for outgoing. err; %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) GetFiles(c *gin.Context, params openapi_server.GetFilesParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetFiles",
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

	// get tmps
	tmps, err := h.serviceHandler.StorageFileList(c.Request.Context(), &a, pageSize, pageToken)
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

func (h *server) GetFilesId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetFilesId",
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

	res, err := h.serviceHandler.StorageFileGet(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not get a file. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) DeleteFilesId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteFilesId",
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

	res, err := h.serviceHandler.StorageFileDelete(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not delete the file. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
