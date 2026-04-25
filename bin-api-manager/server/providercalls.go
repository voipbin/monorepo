package server

import (
	"monorepo/bin-api-manager/gens/openapi_server"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonaddress "monorepo/bin-common-handler/models/address"
	commonoutline "monorepo/bin-common-handler/models/outline"
	fmaction "monorepo/bin-flow-manager/models/action"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/sirupsen/logrus"
)

// GetProvidercalls handles GET /v1/providercalls — list providercall audit records.
// Gated by PermissionProjectSuperAdmin; because that role is platform-level,
// the list is cross-customer (matches GET /v1/providercalls/{id} and DELETE
// /v1/providercalls/{id}). Optional provider_id filter narrows to one provider.
func (h *server) GetProvidercalls(c *gin.Context, params openapi_server.GetProvidercallsParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetProvidercalls",
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
	}

	pageToken := ""
	if params.PageToken != nil {
		pageToken = *params.PageToken
	}

	providerID := uuid.Nil
	if params.ProviderId != nil {
		providerID = uuid.FromStringOrNil(params.ProviderId.String())
	}

	tmps, err := h.serviceHandler.ProviderCallGets(c.Request.Context(), a, pageSize, pageToken, providerID)
	if err != nil {
		log.Errorf("Could not get providercalls. err: %v", err)
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

// PostProvidercalls handles POST /v1/providercalls — create a provider call.
func (h *server) PostProvidercalls(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostProvidercalls",
		"request_address": c.ClientIP,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithField("agent", a)

	var req openapi_server.PostProvidercallsJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON."))
		return
	}

	providerID := uuid.FromStringOrNil(req.ProviderId.String())
	if providerID == uuid.Nil {
		log.Error("provider_id is required.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ARGUMENT", "provider_id is required."))
		return
	}

	flowID := uuid.Nil
	if req.FlowId != nil {
		flowID = uuid.FromStringOrNil(req.FlowId.String())
	}

	actions := []fmaction.Action{}
	if req.Actions != nil {
		for _, v := range *req.Actions {
			actions = append(actions, ConvertFlowManagerAction(v))
		}
	}

	var source *commonaddress.Address
	if req.Source != nil {
		tmp := ConvertCommonAddress(*req.Source)
		source = &tmp
	}

	destinations := []commonaddress.Address{}
	if req.Destinations != nil {
		for _, v := range req.Destinations {
			destinations = append(destinations, ConvertCommonAddress(v))
		}
	}
	// OpenAPI marks destinations as required with minItems: 1, but BindJSON
	// doesn't enforce minItems. Guard explicitly to avoid an orphaned
	// ProviderCall record with empty call_ids.
	if len(destinations) == 0 {
		log.Error("destinations is required and must not be empty.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ARGUMENT", "destinations is required and must not be empty."))
		return
	}

	// OpenAPI response schema allows only yes/no/auto on ProviderCall.Anonymous;
	// default to "auto" when the caller omits the field so the persisted record
	// conforms to the schema.
	anonymous := "auto"
	if req.Anonymous != nil {
		anonymous = string(*req.Anonymous)
	}

	res, err := h.serviceHandler.ProviderCallCreate(c.Request.Context(), a, providerID, flowID, actions, source, destinations, anonymous)
	if err != nil {
		log.Errorf("Could not create the providercall. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

// GetProvidercallsId handles GET /v1/providercalls/{id} — fetch a single record.
func (h *server) GetProvidercallsId(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetProvidercallsId",
		"request_address": c.ClientIP,
		"providercall_id": id,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithField("agent", a)

	target := uuid.FromStringOrNil(id.String())
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID."))
		return
	}

	res, err := h.serviceHandler.ProviderCallGet(c.Request.Context(), a, target)
	if err != nil {
		log.Infof("Could not get the providercall. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

// DeleteProvidercallsId handles DELETE /v1/providercalls/{id} — soft-delete the record.
func (h *server) DeleteProvidercallsId(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteProvidercallsId",
		"request_address": c.ClientIP,
		"providercall_id": id,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithField("agent", a)

	target := uuid.FromStringOrNil(id.String())
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID."))
		return
	}

	res, err := h.serviceHandler.ProviderCallDelete(c.Request.Context(), a, target)
	if err != nil {
		log.Infof("Could not delete the providercall. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}
