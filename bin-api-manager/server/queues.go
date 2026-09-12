package server

import (
	"monorepo/bin-api-manager/gens/openapi_server"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	qmqueue "monorepo/bin-queue-manager/models/queue"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/sirupsen/logrus"
)

func (h *server) GetQueues(c *gin.Context, params openapi_server.GetQueuesParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetQueues",
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

	tmps, err := h.serviceHandler.QueueList(c.Request.Context(), a, pageSize, pageToken)
	if err != nil {
		logrus.Errorf("Could not get queues info. err: %v", err)
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

func (h *server) PostQueues(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostQueues",
		"request_address": c.ClientIP,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithField("agent", a)

	var req openapi_server.PostQueuesJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON.").Wrap(err))
		return
	}

	tagIDs := []uuid.UUID{}
	for _, v := range req.TagIds {
		tagIDs = append(tagIDs, uuid.FromStringOrNil(v))
	}

	waitFlowID := uuid.FromStringOrNil(req.WaitFlowId)

	res, err := h.serviceHandler.QueueCreate(
		c.Request.Context(),
		a,
		req.Name,
		req.Detail,
		qmqueue.RoutingMethod(req.RoutingMethod),
		tagIDs,
		waitFlowID,
		req.WaitTimeout,
		req.ServiceTimeout,
	)
	if err != nil {
		log.Errorf("Could not create a queue. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) DeleteQueuesId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteQueuesId",
		"request_address": c.ClientIP,
		"queue_id":        id,
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

	res, err := h.serviceHandler.QueueDelete(c.Request.Context(), a, target)
	if err != nil {
		log.Infof("Could not get the delete the queue info. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) GetQueuesId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetQueuesId",
		"request_address": c.ClientIP,
		"queue_id":        id,
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

	res, err := h.serviceHandler.QueueGet(c.Request.Context(), a, target)
	if err != nil {
		log.Infof("Could not get the queue info. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutQueuesId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutQueuesId",
		"request_address": c.ClientIP,
		"queue_id":        id,
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

	var req openapi_server.PutQueuesIdJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON.").Wrap(err))
		return
	}

	var routingMethodPtr *qmqueue.RoutingMethod
	if req.RoutingMethod != nil {
		v := qmqueue.RoutingMethod(*req.RoutingMethod)
		routingMethodPtr = &v
	}

	var tagIDsPtr *[]uuid.UUID
	if req.TagIds != nil {
		tagIDs := make([]uuid.UUID, 0, len(*req.TagIds))
		for _, v := range *req.TagIds {
			parsed, errParse := uuid.FromString(v)
			if errParse != nil {
				log.Errorf("Could not parse the tag id. tag_id: %s, err: %v", v, errParse)
				abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_TAG_ID", "One of the provided tag_ids is not a valid UUID.").Wrap(errParse))
				return
			}
			tagIDs = append(tagIDs, parsed)
		}
		tagIDsPtr = &tagIDs
	}

	var waitFlowIDPtr *uuid.UUID
	if req.WaitFlowId != nil {
		parsed, errParse := uuid.FromString(*req.WaitFlowId)
		if errParse != nil {
			log.Errorf("Could not parse the wait_flow_id. wait_flow_id: %s, err: %v", *req.WaitFlowId, errParse)
			abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_WAIT_FLOW_ID", "The provided wait_flow_id is not a valid UUID.").Wrap(errParse))
			return
		}
		waitFlowIDPtr = &parsed
	}

	res, err := h.serviceHandler.QueueUpdate(c.Request.Context(), a, target, req.Name, req.Detail, routingMethodPtr, tagIDsPtr, waitFlowIDPtr, req.WaitTimeout, req.ServiceTimeout)
	if err != nil {
		log.Errorf("Could not update the queue. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutQueuesIdTagIds(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutQueuesIdTagIds",
		"request_address": c.ClientIP,
		"queue_id":        id,
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

	var req openapi_server.PutQueuesIdTagIdsJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON.").Wrap(err))
		return
	}

	tagIDs := []uuid.UUID{}
	for _, v := range req.TagIds {
		tagIDs = append(tagIDs, uuid.FromStringOrNil(v))
	}

	res, err := h.serviceHandler.QueueUpdateTagIDs(c.Request.Context(), a, target, tagIDs)
	if err != nil {
		log.Errorf("Could not update the agent. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutQueuesIdRoutingMethod(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutQueuesIdRoutingMethod",
		"request_address": c.ClientIP,
		"queue_id":        id,
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

	var req openapi_server.PutQueuesIdRoutingMethodJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON.").Wrap(err))
		return
	}

	res, err := h.serviceHandler.QueueUpdateRoutingMethod(c.Request.Context(), a, target, qmqueue.RoutingMethod(req.RoutingMethod))
	if err != nil {
		log.Errorf("Could not update the queue. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) PostQueuesIdDirectHashRegenerate(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostQueuesIdDirectHashRegenerate",
		"request_address": c.ClientIP(),
		"queue_id":        id,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithField("agent", a)

	// Convert openapi_types.UUID to uuid.UUID
	queueID, err := uuid.FromString(id.String())
	if err != nil {
		log.Errorf("Invalid queue ID format. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID.").Wrap(err))
		return
	}

	res, err := h.serviceHandler.QueueDirectHashRegenerate(c.Request.Context(), a, queueID)
	if err != nil {
		log.Errorf("Could not regenerate queue direct hash. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}
