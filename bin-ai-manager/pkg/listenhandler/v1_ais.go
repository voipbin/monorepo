package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-ai-manager/pkg/listenhandler/models/request"
)

// processV1AIsGet handles GET /v1/ais request
func (h *listenHandler) processV1AIsGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1AIsGet",
		"request": m,
	})

	u, err := url.Parse(m.URI)
	if err != nil {
		log.Errorf("Could not parse the request uri. err: %v", err)
		return simpleResponse(400), nil
	}

	// parse the pagination params
	tmpSize, _ := strconv.Atoi(u.Query().Get(PageSize))
	pageSize := uint64(tmpSize)
	pageToken := u.Query().Get(PageToken)

	// get filters from request body
	tmpFilters, err := utilhandler.ParseFiltersFromRequestBody(m.Data)
	if err != nil {
		log.Errorf("Could not parse filters. err: %v", err)
		return simpleResponse(400), nil
	}

	// convert to typed filters
	typedFilters, err := utilhandler.ConvertFilters[ai.FieldStruct, ai.Field](ai.FieldStruct{}, tmpFilters)
	if err != nil {
		log.Errorf("Could not convert filters. err: %v", err)
		return simpleResponse(400), nil
	}

	log = log.WithFields(logrus.Fields{
		"size":    pageSize,
		"token":   pageToken,
		"filters": typedFilters,
	})

	tmp, err := h.aiHandler.List(ctx, pageSize, pageToken, typedFilters)
	if err != nil {
		log.Debugf("Could not get conferences. err: %v", err)
		return errorResponse(err), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1AIsPost handles POST /v1/ais request
func (h *listenHandler) processV1AIsPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"handler": "processV1AIsPost",
		"request": m,
	})

	var req request.V1DataAIsPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Errorf("Could not unmarshal the requested data. err: %v", err)
		return simpleResponse(400), nil
	}

	// Validate the whitelist BEFORE creating the AI. Validating afterwards
	// still returns 400 but leaves an orphaned AI behind: the customer sees a
	// rejected request and an extra AI they never asked for. The request's own
	// customer id is the right owner to check against -- it is what Create is
	// about to persist.
	//
	// No stored whitelist to exempt: the AI does not exist yet, so every id
	// here is newly added and a deleted one is rejected.
	if req.McpServerIDs != nil {
		if err := h.aiHandler.ValidateMcpServerIDs(ctx, req.CustomerID, *req.McpServerIDs, nil); err != nil {
			log.Errorf("Could not validate mcp_server_ids. err: %v", err)
			return errorResponse(err), nil
		}
	}

	tmp, err := h.aiHandler.Create(
		ctx,
		req.CustomerID,
		req.Name,
		req.Detail,
		req.Type,
		req.EngineModel,
		req.Parameter,
		req.EngineKey,
		req.RagID,
		req.InitPrompt,
		req.TTSType,
		req.TTSVoiceID,
		req.STTType,
		req.STTLanguage,
		req.ToolNames,
		req.VADConfig,
		req.SmartTurnEnabled,
		req.AutoAICallAuditEnabled,
	)
	if err != nil {
		log.Errorf("Could not create ai. err: %v", err)
		return errorResponse(err), nil
	}

	if req.McpServerIDs != nil {
		tmp, err = h.aiHandler.UpdateMcpServerIDs(ctx, tmp.ID, *req.McpServerIDs)
		if err != nil {
			log.Errorf("Could not update ai mcp_server_ids. err: %v", err)
			return errorResponse(err), nil
		}
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1AIsIDGet handles GET /v1/ais/<ai-id> request
func (h *listenHandler) processV1AIsIDGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"handler": "processV1AIsIDGet",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		log.Errorf("Wrong uri item count. uri_items: %d", len(uriItems))
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])
	if id == uuid.Nil {
		log.Errorf("Invalid AI ID.")
		return simpleResponse(400), nil
	}

	tmp, err := h.aiHandler.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get ai. err: %v", err)
		return errorResponse(err), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1AIsIDDelete handles DELETE /v1/ais/<ai-id> request
func (h *listenHandler) processV1AIsIDDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"handler": "processV1AIsIDDelete",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		log.Errorf("Wrong uri item count. uri_items: %d", len(uriItems))
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])
	if id == uuid.Nil {
		log.Errorf("Invalid AI ID.")
		return simpleResponse(400), nil
	}

	tmp, err := h.aiHandler.Delete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete ai. err: %v", err)
		return errorResponse(err), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1AIsIDPut handles PUT /v1/ais/<ai-id> request
func (h *listenHandler) processV1AIsIDPut(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"handler": "processV1AIsIDPut",
		"request": m,
	})

	var req request.V1DataAIsIDPut
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Errorf("Could not unmarshal the requested data. err: %v", err)
		return simpleResponse(400), nil
	}

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		log.Errorf("Wrong uri item count. uri_items: %d", len(uriItems))
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])
	if id == uuid.Nil {
		log.Errorf("Invalid AI ID.")
		return simpleResponse(400), nil
	}

	// Validate the whitelist BEFORE mutating the AI, for the same reason as
	// the POST path: a rejected whitelist must not leave the AI already
	// half-updated with the request's other fields.
	//
	// The owner has to come from a read here. Unlike POST, a PUT body carries
	// no customer id, and taking it from the updated AI (as this did before)
	// means the mutation has already committed by the time the check runs.
	//
	// The same read supplies the stored whitelist, so exempting an already
	// stored (since-deleted) id costs no extra query -- Update does not take
	// McpServerIDs, so this list is still the unmodified stored one.
	if req.McpServerIDs != nil {
		preUpdateAI, err := h.aiHandler.Get(ctx, id)
		if err != nil {
			log.Errorf("Could not get ai for mcp_server_ids validation. err: %v", err)
			return errorResponse(err), nil
		}

		if err := h.aiHandler.ValidateMcpServerIDs(ctx, preUpdateAI.CustomerID, *req.McpServerIDs, preUpdateAI.McpServerIDs); err != nil {
			log.Errorf("Could not validate mcp_server_ids. err: %v", err)
			return errorResponse(err), nil
		}
	}

	tmp, err := h.aiHandler.Update(
		ctx,
		id,
		req.Name,
		req.Detail,
		req.Type,
		req.EngineModel,
		req.Parameter,
		req.EngineKey,
		req.RagID,
		req.InitPrompt,
		req.TTSType,
		req.TTSVoiceID,
		req.STTType,
		req.STTLanguage,
		req.ToolNames,
		req.VADConfig,
		req.SmartTurnEnabled,
		req.AutoAICallAuditEnabled,
	)
	if err != nil {
		log.Errorf("Could not update ai. err: %v", err)
		return errorResponse(err), nil
	}

	if req.McpServerIDs != nil {
		tmp, err = h.aiHandler.UpdateMcpServerIDs(ctx, tmp.ID, *req.McpServerIDs)
		if err != nil {
			log.Errorf("Could not update ai mcp_server_ids. err: %v", err)
			return errorResponse(err), nil
		}
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
