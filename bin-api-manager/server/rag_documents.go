package server

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	rmdocument "monorepo/bin-rag-manager/models/document"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/sirupsen/logrus"
)

func (h *server) PostRagDocuments(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostRagDocuments",
		"request_address": c.ClientIP(),
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

	var req openapi_server.PostRagDocumentsJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	ragID, err := uuid.FromString(req.RagId.String())
	if err != nil {
		log.Errorf("Invalid rag_id format. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	docType := rmdocument.DocType(string(req.DocType))

	sourceURL := ""
	if req.SourceUrl != nil {
		sourceURL = *req.SourceUrl
	}

	storageFileID := uuid.Nil
	if req.StorageFileId != nil {
		storageFileID, err = uuid.FromString(req.StorageFileId.String())
		if err != nil {
			log.Errorf("Invalid storage_file_id format. err: %v", err)
			c.AbortWithStatus(400)
			return
		}
	}

	res, err := h.serviceHandler.RagDocumentCreate(c.Request.Context(), &a, ragID, req.Name, docType, sourceURL, storageFileID)
	if err != nil {
		log.Errorf("Could not create data. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) GetRagDocuments(c *gin.Context, params openapi_server.GetRagDocumentsParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetRagDocuments",
		"request_address": c.ClientIP(),
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

	ragID := uuid.Nil
	if params.RagId != nil {
		var err error
		ragID, err = uuid.FromString(params.RagId.String())
		if err != nil {
			log.Errorf("Invalid rag_id format. err: %v", err)
			c.AbortWithStatus(400)
			return
		}
	}

	tmps, err := h.serviceHandler.RagDocumentGets(c.Request.Context(), &a, ragID, pageSize, pageToken)
	if err != nil {
		log.Errorf("Could not get data list. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(tmps) > 0 {
		if tmps[len(tmps)-1].TMCreate != nil {
			nextToken = tmps[len(tmps)-1].TMCreate.UTC().Format("2006-01-02T15:04:05.000000Z")
		}
	}

	res := GenerateListResponse(tmps, nextToken)
	c.JSON(200, res)
}

func (h *server) GetRagDocumentsId(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetRagDocumentsId",
		"request_address": c.ClientIP(),
		"target_id":       id,
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

	target, err := uuid.FromString(id.String())
	if err != nil {
		log.Errorf("Invalid ID format. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.RagDocumentGet(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not get data. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) DeleteRagDocumentsId(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteRagDocumentsId",
		"request_address": c.ClientIP(),
		"target_id":       id,
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

	target, err := uuid.FromString(id.String())
	if err != nil {
		log.Errorf("Invalid ID format. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.RagDocumentDelete(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not delete data. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
