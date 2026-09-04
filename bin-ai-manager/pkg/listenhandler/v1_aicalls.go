package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/pkg/listenhandler/models/request"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// processV1AIcallsGet handles GET /v1/aicall request
func (h *listenHandler) processV1AIcallsGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"handler": "processV1AIcallsGet",
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
	typedFilters, err := utilhandler.ConvertFilters[aicall.FieldStruct, aicall.Field](aicall.FieldStruct{}, tmpFilters)
	if err != nil {
		log.Errorf("Could not convert filters. err: %v", err)
		return simpleResponse(400), nil
	}

	log = log.WithFields(logrus.Fields{
		"size":    pageSize,
		"token":   pageToken,
		"filters": typedFilters,
	})

	tmp, err := h.aicallHandler.List(ctx, pageSize, pageToken, typedFilters)
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

// processV1AIcallsPost handles POST /v1/aicalls request
func (h *listenHandler) processV1AIcallsPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"handler": "processV1AIcallsPost",
		"request": m,
	})

	var req request.V1DataAIcallsPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Errorf("Could not unmarshal the requested data. err: %v", err)
		return simpleResponse(400), nil
	}

	tmp, err := h.aicallHandler.Start(ctx, req.AssistanceType, req.AssistanceID, req.ActiveflowID, req.ReferenceType, req.ReferenceID)
	if err != nil {
		log.Errorf("Could not create aicall. err: %v", err)
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

// processV1AIcallsIDGet handles GET /v1/aicalls/<aicall-id> request.
//
// The optional skip_cache=true query parameter forces a database-authoritative
// read, bypassing ai-manager's own Redis snapshot cache. Its one caller today is
// messagehandler's stale-reply guard, which must not drop the agent's genuine
// answer because of a transiently-stale cached PipecatcallID. See
// docs/plans/2026-09-03-insight-ai-realtime-listen-design.md §5.4.4(b).
func (h *listenHandler) processV1AIcallsIDGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"handler": "processV1AIcallsIDGet",
		"request": m,
	})

	// url.Parse, not strings.Split(m.URI, "/"): with a query string attached,
	// splitting on "/" yields "<uuid>?skip_cache=true" as the id element, which
	// parses to uuid.Nil.
	u, err := url.Parse(m.URI)
	if err != nil {
		log.Errorf("Could not parse the uri. err: %v", err)
		return simpleResponse(400), nil
	}

	uriItems := strings.Split(u.Path, "/")
	if len(uriItems) < 4 {
		log.Errorf("Wrong uri item count. uri_items: %d", len(uriItems))
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])
	if id == uuid.Nil {
		log.Errorf("Invalid AIcall ID.")
		return simpleResponse(400), nil
	}

	skipCache := u.Query().Get("skip_cache") == "true"

	var tmp *aicall.AIcall
	if skipCache {
		tmp, err = h.aicallHandler.GetSkipCache(ctx, id)
	} else {
		tmp, err = h.aicallHandler.Get(ctx, id)
	}
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

// processV1AIcallsIDDelete handles DELETE /v1/aicalls/<aicall-id> request
func (h *listenHandler) processV1AIcallsIDDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"handler": "processV1AIcallsIDDelete",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		log.Errorf("Wrong uri item count. uri_items: %d", len(uriItems))
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])
	if id == uuid.Nil {
		log.Errorf("Invalid AIcall ID.")
		return simpleResponse(400), nil
	}

	tmp, err := h.aicallHandler.Delete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete aicall. err: %v", err)
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

// processV1AIcallsPost handles POST /v1/aicalls/<aicall-id>/terminate request
func (h *listenHandler) processV1AIcallsIDTerminatePost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"handler": "processV1AIcallsIDTerminatePost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		log.Errorf("Wrong uri item count. uri_items: %d", len(uriItems))
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])
	if id == uuid.Nil {
		log.Errorf("Invalid AIcall ID.")
		return simpleResponse(400), nil
	}

	tmp, err := h.aicallHandler.ProcessTerminate(ctx, id)
	if err != nil {
		log.Errorf("Could not terminate aicall. err: %v", err)
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

// processV1AIcallsIDListenPost handles POST /v1/aicalls/<aicall-id>/listen request.
//
// Deliberately thin -- parse the id, one business-handler call, marshal, 200 --
// matching processV1AIcallsIDTerminatePost's shape. No orchestration logic
// belongs in listenhandler (design review round 13 finding MEDIUM-4), and the
// handler returns a domain *aicall.AIcall which this layer marshals directly
// (root CLAUDE.md layering style (A)), with no response.* DTO.
//
// Note the path: ai-manager's own RPC surface keeps the plain
// /v1/aicalls/{id}/listen route. Only the PUBLIC, api-manager-facing path is
// /service_agents/aicalls/{id}/listen. Two services, two routes, one shared
// trailing segment -- do not "fix" this one to match the public path.
func (h *listenHandler) processV1AIcallsIDListenPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"handler": "processV1AIcallsIDListenPost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		log.Errorf("Wrong uri item count. uri_items: %d", len(uriItems))
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])
	if id == uuid.Nil {
		log.Errorf("Invalid AIcall ID.")
		return simpleResponse(400), nil
	}

	tmp, err := h.aicallHandler.ProcessListen(ctx, id)
	if err != nil {
		log.Errorf("Could not start listening. err: %v", err)
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

// processV1AIcallsIDToolExecutePost handles POST /v1/aicalls/<aicall-id>/tool_execute request
func (h *listenHandler) processV1AIcallsIDToolExecutePost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"handler": "processV1AIcallsIDToolExecutePost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		log.Errorf("Wrong uri item count. uri_items: %d", len(uriItems))
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])

	var req request.V1DataAIcallsIDToolExecutePost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Errorf("Could not unmarshal the requested data. err: %v", err)
		return simpleResponse(400), nil
	}

	tmp, err := h.aicallHandler.ToolHandle(ctx, id, req.ID, req.Type, req.Function, req.PipecatcallID)
	if err != nil {
		log.Errorf("Could not execute tool. err: %v", err)
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
