package server

import (
	"monorepo/bin-api-manager/gens/openapi_server"
	cmgroupcall "monorepo/bin-call-manager/models/groupcall"
	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	fmaction "monorepo/bin-flow-manager/models/action"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

func (h *server) PostGroupcalls(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostGroupcalls",
		"request_address": c.ClientIP,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(
			commonoutline.ServiceNameAPIManager,
			"AUTHENTICATION_REQUIRED",
			"Authentication is required.",
		))
		return
	}
	log = log.WithField("agent", a)

	var req openapi_server.PostGroupcallsJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(
			commonoutline.ServiceNameAPIManager,
			"INVALID_JSON_BODY",
			"The request body is not valid JSON.",
		))
		return
	}

	source := ConvertCommonAddress(req.Source)
	destinations := []commonaddress.Address{}
	for _, v := range req.Destinations {
		destinations = append(destinations, ConvertCommonAddress(v))
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

	res, err := h.serviceHandler.GroupcallCreate(c.Request.Context(), a, source, destinations, flowID, actions, cmgroupcall.RingMethod(req.RingMethod), cmgroupcall.AnswerMethod(req.AnswerMethod))
	if err != nil {
		log.Errorf("Could not create a groupcall. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) GetGroupcalls(c *gin.Context, params openapi_server.GetGroupcallsParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetGroupcalls",
		"request_address": c.ClientIP,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(
			commonoutline.ServiceNameAPIManager,
			"AUTHENTICATION_REQUIRED",
			"Authentication is required.",
		))
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

	tmps, err := h.serviceHandler.GroupcallList(c.Request.Context(), a, pageSize, pageToken)
	if err != nil {
		log.Errorf("Could not get a groupcall list. err: %v", err)
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

func (h *server) GetGroupcallsId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetGroupcallsId",
		"request_address": c.ClientIP,
		"groupcall_id":    id,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(
			commonoutline.ServiceNameAPIManager,
			"AUTHENTICATION_REQUIRED",
			"Authentication is required.",
		))
		return
	}
	log = log.WithField("agent", a)

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(
			commonoutline.ServiceNameAPIManager,
			"INVALID_ID",
			"The provided id is not a valid UUID.",
		))
		return
	}

	res, err := h.serviceHandler.GroupcallGet(c.Request.Context(), a, target)
	if err != nil {
		log.Errorf("Could not get a groupcall. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) PostGroupcallsIdHangup(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostGroupcallsIdHangup",
		"request_address": c.ClientIP,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(
			commonoutline.ServiceNameAPIManager,
			"AUTHENTICATION_REQUIRED",
			"Authentication is required.",
		))
		return
	}
	log = log.WithField("agent", a)

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(
			commonoutline.ServiceNameAPIManager,
			"INVALID_ID",
			"The provided id is not a valid UUID.",
		))
		return
	}

	res, err := h.serviceHandler.GroupcallHangup(c.Request.Context(), a, target)
	if err != nil {
		log.Errorf("Could not hangup the groupcall. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) DeleteGroupcallsId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteGroupcallsId",
		"request_address": c.ClientIP,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(
			commonoutline.ServiceNameAPIManager,
			"AUTHENTICATION_REQUIRED",
			"Authentication is required.",
		))
		return
	}
	log = log.WithField("agent", a)

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(
			commonoutline.ServiceNameAPIManager,
			"INVALID_ID",
			"The provided id is not a valid UUID.",
		))
		return
	}

	res, err := h.serviceHandler.GroupcallDelete(c.Request.Context(), a, target)
	if err != nil {
		log.Errorf("Could not delete the groupcall. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}
