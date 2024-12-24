package storage_files

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	"net/http"

	"monorepo/bin-api-manager/api/models/common"
	"monorepo/bin-api-manager/api/models/request"
	"monorepo/bin-api-manager/api/models/response"
	"monorepo/bin-api-manager/pkg/servicehandler"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// storageFilesPOST handles POST /files request.
// It creates a temp file and create a call with temp file.
//
//	@Summary		Make an outbound call
//	@Description	dialing to destination
//	@Produce		json
//	@Param			call	body		request.BodyCallsPOST	true	"The call detail"
//	@Success		200		{object}	call.Call
//	@Router			/v1.0/storage_files [post]
func storageFilesPOST(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "storageFilesPOST",
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

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// create call
	res, err := serviceHandler.StorageFileCreate(c.Request.Context(), &a, f, "", "", header.Filename)
	if err != nil {
		log.Errorf("Could not create a call for outgoing. err; %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// storageFilesGET handles GET /storage_files request.
// It gets a list of files with the given info.
//
//	@Summary		Gets a list of files.
//	@Description	Gets a list of files
//	@Produce		json
//	@Param			page_size	query		int		false	"The size of results. Max 100"
//	@Param			page_token	query		string	false	"The token. tm_create"
//	@Success		200			{object}	response.BodyFilesGET
//	@Router			/v1.0/storage_files [get]
func storageFilesGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "storageFilesGET",
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

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// get files
	files, err := serviceHandler.StorageFileGets(c.Request.Context(), &a, pageSize, req.PageToken)
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

// storageFilesIDGET handles GET /storage_files/{id} request.
// It returns detail file info.
//
//	@Summary		Returns detail file info.
//	@Description	Returns detail file info of the given file id.
//	@Produce		json
//	@Param			id	path		string	true	"The ID of the file"
//	@Success		200	{object}	file.File
//	@Router			/v1.0/storage_files/{id} [get]
func storageFilesIDGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "storageFilesIDGET",
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
	res, err := serviceHandler.StorageFileGet(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not get a file. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// storageFilesIDDELETE handles DELETE /files/{id} request.
// It deletes a exist file info.
//
//	@Summary		Delete a file.
//	@Description	Delete a file.
//	@Produce		json
//	@Param			id	query	string	true	"The file's id"
//	@Success		200
//	@Router			/v1.0/storage_files/{id} [delete]
func storageFilesIDDELETE(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "storageFilesIDDELETE",
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

	// delete a file
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.ServiceAgentFileDelete(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not delete the file. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
