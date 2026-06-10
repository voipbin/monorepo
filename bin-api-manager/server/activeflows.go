package server

import (
	"monorepo/bin-api-manager/gens/openapi_server"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	fmaction "monorepo/bin-flow-manager/models/action"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

func (h *server) GetActiveflows(c *gin.Context, params openapi_server.GetActiveflowsParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "activeflowsGET",
		"request_address": c.ClientIP,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithFields(logrus.Fields{
		"auth": a,
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

	tmps, err := h.serviceHandler.ActiveflowList(c.Request.Context(), a, pageSize, pageToken)
	if err != nil {
		log.Errorf("Could not get calls info. err: %v", err)
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

func (h *server) PostActiveflows(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostActiveflows",
		"request_address": c.ClientIP,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithFields(logrus.Fields{
		"auth": a,
	})

	var req openapi_server.PostActiveflowsJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON."))
		return
	}

	id := uuid.Nil
	if req.Id != nil {
		id = uuid.FromStringOrNil(*req.Id)
	}

	flowID := uuid.Nil
	if req.FlowId != nil {
		flowID = uuid.FromStringOrNil(*req.FlowId)
	}

	actions := []fmaction.Action{}
	if req.Actions != nil {
		for _, v := range *req.Actions {
			actions = append(actions, ConvertFlowManagerAction(v))
		}
	}

	variables := convertVariables(req.Variables)
	if err := validateVariables(variables); err != nil {
		log.Errorf("Invalid variables. err: %v", err)
		abortWithError(c, err)
		return
	}

	webhookURI := ""
	if req.WebhookUri != nil {
		webhookURI = *req.WebhookUri
	}
	if err := validateWebhookURI(webhookURI); err != nil {
		log.Errorf("Invalid webhook uri. err: %v", err)
		abortWithError(c, err)
		return
	}

	webhookMethodStr := ""
	if req.WebhookMethod != nil {
		webhookMethodStr = string(*req.WebhookMethod)
	}
	webhookMethod, errMethod := validateWebhookMethod(webhookMethodStr)
	if errMethod != nil {
		log.Errorf("Invalid webhook method. err: %v", errMethod)
		abortWithError(c, errMethod)
		return
	}

	res, err := h.serviceHandler.ActiveflowCreate(c.Request.Context(), a, id, flowID, actions, variables, webhookURI, webhookMethod)
	if err != nil {
		log.Errorf("Could not create a call for outgoing. err; %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) GetActiveflowsId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetActiveflowsId",
		"request_address": c.ClientIP,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithFields(logrus.Fields{
		"auth": a,
	})

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID."))
		return
	}
	log = log.WithField("activeflow_id", target)

	res, err := h.serviceHandler.ActiveflowGet(c.Request.Context(), a, target)
	if err != nil {
		log.Errorf("Could not get a activeflow. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) DeleteActiveflowsId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteActiveflowsId",
		"request_address": c.ClientIP,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithFields(logrus.Fields{
		"auth": a,
	})

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID."))
		return
	}
	log = log.WithField("activeflow_id", target)

	res, err := h.serviceHandler.ActiveflowDelete(c.Request.Context(), a, target)
	if err != nil {
		log.Errorf("Could not delete the activeflow. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) PostActiveflowsIdStop(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostActiveflowsIdStop",
		"request_address": c.ClientIP,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithFields(logrus.Fields{
		"auth": a,
	})

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID."))
		return
	}
	log = log.WithField("activeflow_id", target)

	res, err := h.serviceHandler.ActiveflowStop(c.Request.Context(), a, target)
	if err != nil {
		log.Errorf("Could not stop the activeflow. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}
