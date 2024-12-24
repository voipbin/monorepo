package service_agents

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/api/models/common"
	"monorepo/bin-api-manager/api/models/request"
	"monorepo/bin-api-manager/api/models/response"
	"monorepo/bin-api-manager/pkg/servicehandler"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

const (
	constMaxFileSize = int64(30 << 20) // Max upload file size. 30 MB.
)

// filesPOST handles POST /service_agents/files request.
// It uploads a file.
//
//	@Summary		Upload a file
//	@Description	Upload a file
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			file	formData	file	true	"The file to upload"
//	@Success		200		{object}	response.FileResponse
//	@Router			/v1.0/service_agents/files [post]
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

	// set limit for max file siz
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, constMaxFileSize)

	f, header, err := c.Request.FormFile("file")
	if err != nil {
		log.Errorf("Could not get file. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log.WithField("file", header).Debugf("Checking uploaded file header. filename: %s", header.Filename)

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.ServiceAgentFileCreate(c.Request.Context(), &a, f, "", "", header.Filename)
	if err != nil {
		log.Errorf("Could not upload the file. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// filesGET handles GET /service_agents/files request.
// It gets a list of files with the given info.
//
//	@Summary		Gets a list of files.
//	@Description	Gets a list of files
//	@Produce		json
//	@Param			page_size	query		int		false	"The size of results. Max 100"
//	@Param			page_token	query		string	false	"The token. tm_create"
//	@Success		200			{object}	response.BodyFilesGET
//	@Router			/v1.0/service_agents/files [get]
func filesGET(c *gin.Context) {
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

	var req request.ParamFilesGET
	if err := c.BindQuery(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	// set max page size
	pageSize := req.PageSize
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 100
		log.Debugf("Invalid requested page size. Set to default. page_size: %d", pageSize)
	}
	log.Debugf("filesGET. Received request detail. page_size: %d, page_token: %s", pageSize, req.PageToken)

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	files, err := serviceHandler.ServiceAgentFileGets(c.Request.Context(), &a, pageSize, req.PageToken)
	if err != nil {
		log.Errorf("Could not get a file list. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(files) > 0 {
		nextToken = files[len(files)-1].TMCreate
	}
	res := response.BodyFilesGET{
		Result: files,
		Pagination: response.Pagination{
			NextPageToken: nextToken,
		},
	}

	c.JSON(200, res)
}

// filesIDGET handles GET /service_agents/files/{id} request.
// It returns detail file info.
//
//	@Summary		Returns detail file info.
//	@Description	Returns detail file info of the given file id.
//	@Produce		json
//	@Param			id	path		string	true	"The ID of the file"
//	@Success		200	{object}	file.File
//	@Router			/v1.0/service_agents/files/{id} [get]
func filesIDGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "filesIDGET",
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
	log = log.WithField("file_id", id)
	log.Debug("Executing filesIDGET.")

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.ServiceAgentFileGet(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not get a file. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// filesIDDELETE handles DELETE /service_agents/files/{id} request.
// It deletes a exist file info.
//
//	@Summary		Delete a file.
//	@Description	Delete a file.
//	@Produce		json
//	@Param			id	query	string	true	"The file's id"
//	@Success		200
//	@Router			/v1.0/service_agents/files/{id} [delete]
func filesIDDELETE(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "filesIDDELETE",
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
	log = log.WithField("file_id", id)
	log.Debug("Executing storageFilesIDDELETE.")

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.ServiceAgentFileDelete(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not delete the file. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
