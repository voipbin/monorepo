package server

import (
	"monorepo/bin-api-manager/gens/openapi_server"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	fmaction "monorepo/bin-flow-manager/models/action"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/sirupsen/logrus"
)

func (h *server) PostFlows(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostFlows",
		"request_address": c.ClientIP,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithField("agent", a)

	var req openapi_server.PostFlowsJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON."))
		return
	}

	actions := []fmaction.Action{}
	for _, v := range req.Actions {
		actions = append(actions, ConvertFlowManagerAction(v))
	}

	onCompleteFlowId := uuid.Nil
	if req.OnCompleteFlowId != nil {
		onCompleteFlowId = uuid.FromStringOrNil(*req.OnCompleteFlowId)
	}
	res, err := h.serviceHandler.FlowCreate(c.Request.Context(), a, req.Name, req.Detail, actions, onCompleteFlowId, true)
	if err != nil {
		log.Errorf("Could not create data. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) GetFlows(c *gin.Context, params openapi_server.GetFlowsParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetFlows",
		"request_address": c.ClientIP,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithField("agent", a)

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

	tmps, err := h.serviceHandler.FlowList(c.Request.Context(), a, pageSize, pageToken)
	if err != nil {
		log.Errorf("Could not get data list. err: %v", err)
		abortWithServiceError(c, err)
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

func (h *server) GetFlowsId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetFlowsId",
		"request_address": c.ClientIP,
		"target_id":       id,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithField("agent", a)

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID."))
		return
	}

	res, err := h.serviceHandler.FlowGet(c.Request.Context(), a, target)
	if err != nil {
		log.Errorf("Could not get data. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutFlowsId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutFlowsId",
		"request_address": c.ClientIP,
		"target_id":       id,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithField("agent", a)

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID."))
		return
	}

	var req openapi_server.PutFlowsIdJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON."))
		return
	}

	actions := []fmaction.Action{}
	for _, v := range req.Actions {
		actions = append(actions, ConvertFlowManagerAction(v))
	}

	onCompleteFlowID := uuid.Nil
	if req.OnCompleteFlowId != nil {
		onCompleteFlowID = uuid.FromStringOrNil(*req.OnCompleteFlowId)
	}

	res, err := h.serviceHandler.FlowUpdate(c.Request.Context(), a, target, req.Name, req.Detail, actions, onCompleteFlowID)
	if err != nil {
		log.Errorf("Could not update data. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) DeleteFlowsId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteFlowsId",
		"request_address": c.ClientIP,
		"target_id":       id,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithField("agent", a)

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID."))
		return
	}

	res, err := h.serviceHandler.FlowDelete(c.Request.Context(), a, target)
	if err != nil {
		log.Errorf("Could not delete data. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) PostFlowsIdDirectHashRegenerate(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostFlowsIdDirectHashRegenerate",
		"request_address": c.ClientIP(),
		"flow_id":         id,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithField("agent", a)

	// Convert openapi_types.UUID to uuid.UUID
	flowID, err := uuid.FromString(id.String())
	if err != nil {
		log.Errorf("Invalid flow ID format. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID."))
		return
	}

	res, err := h.serviceHandler.FlowDirectHashRegenerate(c.Request.Context(), a, flowID)
	if err != nil {
		log.Errorf("Could not regenerate flow direct hash. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}
