package server

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	rmrag "monorepo/bin-rag-manager/models/rag"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/sirupsen/logrus"
)

func (h *server) PostRags(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostRags",
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

	var req openapi_server.PostRagsJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	description := ""
	if req.Description != nil {
		description = *req.Description
	}

	storageFileIDs := []uuid.UUID{}
	if req.StorageFileIds != nil {
		for _, id := range *req.StorageFileIds {
			uid, err := uuid.FromString(id.String())
			if err != nil {
				log.Errorf("Invalid storage_file_id format. err: %v", err)
				c.AbortWithStatus(400)
				return
			}
			storageFileIDs = append(storageFileIDs, uid)
		}
	}

	sourceURLs := []string{}
	if req.SourceUrls != nil {
		sourceURLs = *req.SourceUrls
	}

	res, err := h.serviceHandler.RagCreate(c.Request.Context(), &a, req.Name, description, storageFileIDs, sourceURLs)
	if err != nil {
		log.Errorf("Could not create data. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) GetRags(c *gin.Context, params openapi_server.GetRagsParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetRags",
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

	tmps, err := h.serviceHandler.RagGets(c.Request.Context(), &a, pageSize, pageToken)
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

func (h *server) GetRagsId(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetRagsId",
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

	res, err := h.serviceHandler.RagGet(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not get data. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutRagsId(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutRagsId",
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

	var req openapi_server.PutRagsIdJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	fields := map[rmrag.Field]any{}
	if req.Name != nil {
		fields[rmrag.FieldName] = *req.Name
	}
	if req.Description != nil {
		fields[rmrag.FieldDescription] = *req.Description
	}

	if len(fields) == 0 {
		log.Errorf("No fields to update.")
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.RagUpdate(c.Request.Context(), &a, target, fields)
	if err != nil {
		log.Errorf("Could not update data. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PostRagsIdSources(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostRagsIdSources",
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

	var req openapi_server.PostRagsIdSourcesJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	storageFileIDs := []uuid.UUID{}
	if req.StorageFileIds != nil {
		for _, fid := range *req.StorageFileIds {
			uid, err := uuid.FromString(fid.String())
			if err != nil {
				log.Errorf("Invalid storage_file_id format. err: %v", err)
				c.AbortWithStatus(400)
				return
			}
			storageFileIDs = append(storageFileIDs, uid)
		}
	}

	sourceURLs := []string{}
	if req.SourceUrls != nil {
		sourceURLs = *req.SourceUrls
	}

	res, err := h.serviceHandler.RagAddSources(c.Request.Context(), &a, target, storageFileIDs, sourceURLs)
	if err != nil {
		log.Errorf("Could not add sources. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) DeleteRagsIdSourcesSourceId(c *gin.Context, id openapi_types.UUID, sourceId openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteRagsIdSourcesSourceId",
		"request_address": c.ClientIP(),
		"rag_id":          id,
		"source_id":       sourceId,
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

	ragID, err := uuid.FromString(id.String())
	if err != nil {
		log.Errorf("Invalid rag ID format. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	sourceID, err := uuid.FromString(sourceId.String())
	if err != nil {
		log.Errorf("Invalid source ID format. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.RagRemoveSource(c.Request.Context(), &a, ragID, sourceID)
	if err != nil {
		log.Errorf("Could not remove source. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) DeleteRagsId(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteRagsId",
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

	res, err := h.serviceHandler.RagDelete(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not delete data. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
