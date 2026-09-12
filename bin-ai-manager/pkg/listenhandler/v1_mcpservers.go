package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-ai-manager/models/mcpserver"
	"monorepo/bin-ai-manager/pkg/listenhandler/models/request"
	"monorepo/bin-ai-manager/pkg/listenhandler/models/response"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

// processV1McpServersGet handles GET /v1/mcp_servers request
func (h *listenHandler) processV1McpServersGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1McpServersGet",
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
	typedFilters, err := utilhandler.ConvertFilters[mcpserver.FieldStruct, mcpserver.Field](mcpserver.FieldStruct{}, tmpFilters)
	if err != nil {
		log.Errorf("Could not convert filters. err: %v", err)
		return simpleResponse(400), nil
	}

	log = log.WithFields(logrus.Fields{
		"size":    pageSize,
		"token":   pageToken,
		"filters": typedFilters,
	})

	tmp, err := h.mcpServerHandler.List(ctx, pageSize, pageToken, typedFilters)
	if err != nil {
		log.Debugf("Could not get mcp servers. err: %v", err)
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

// processV1McpServersPost handles POST /v1/mcp_servers request
func (h *listenHandler) processV1McpServersPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"handler": "processV1McpServersPost",
		"request": m,
	})

	var req request.V1DataMcpServersPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Errorf("Could not unmarshal the requested data. err: %v", err)
		return simpleResponse(400), nil
	}

	tmp, err := h.mcpServerHandler.Create(
		ctx,
		req.CustomerID,
		req.Name,
		req.Detail,
		req.URL,
		mcpserver.StatusActive,
		req.AuthType,
		req.APIKeyHeader,
		req.Secret,
	)
	if err != nil {
		log.Errorf("Could not create mcp server. err: %v", err)
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

// processV1McpServersIDGet handles GET /v1/mcp_servers/<mcp-server-id> request
func (h *listenHandler) processV1McpServersIDGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"handler": "processV1McpServersIDGet",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		log.Errorf("Wrong uri item count. uri_items: %d", len(uriItems))
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])
	if id == uuid.Nil {
		log.Errorf("Invalid mcp server ID.")
		return simpleResponse(400), nil
	}

	tmp, err := h.mcpServerHandler.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get mcp server. err: %v", err)
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

// processV1McpServersIDPut handles PUT /v1/mcp_servers/<mcp-server-id> request
func (h *listenHandler) processV1McpServersIDPut(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"handler": "processV1McpServersIDPut",
		"request": m,
	})

	var req request.V1DataMcpServersIDPut
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
		log.Errorf("Invalid mcp server ID.")
		return simpleResponse(400), nil
	}

	tmp, err := h.mcpServerHandler.Update(
		ctx,
		id,
		req.Name,
		req.Detail,
		req.URL,
		req.Status,
		req.AuthType,
		req.APIKeyHeader,
		req.Secret,
	)
	if err != nil {
		log.Errorf("Could not update mcp server. err: %v", err)
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

// processV1McpServersOAuthStartPost handles POST /v1/mcp_servers/oauth/start request
func (h *listenHandler) processV1McpServersOAuthStartPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"handler": "processV1McpServersOAuthStartPost",
		"request": m,
	})

	var req request.V1DataMcpServersOAuthStartPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Errorf("Could not unmarshal the requested data. err: %v", err)
		return simpleResponse(400), nil
	}

	authorizeURL, linkToken, err := h.mcpOAuthHandler.Start(ctx, req.CustomerID, req.Vendor, req.McpServerID)
	if err != nil {
		log.Errorf("Could not start mcp oauth flow. err: %v", err)
		return errorResponse(err), nil
	}

	res := response.V1ResponseMcpServersOAuthStartPost{
		AuthorizeURL: authorizeURL,
		LinkToken:    linkToken,
	}

	data, err := json.Marshal(res)
	if err != nil {
		log.Errorf("Could not marshal the response message. message: %v, err: %v", res, err)
		return simpleResponse(500), nil
	}

	return &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}, nil
}

// processV1McpServersOAuthCallbackGet handles GET /v1/mcp_servers/oauth/callback request
func (h *listenHandler) processV1McpServersOAuthCallbackGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"handler": "processV1McpServersOAuthCallbackGet",
		"request": m,
	})

	u, err := url.Parse(m.URI)
	if err != nil {
		log.Errorf("Could not parse the request uri. err: %v", err)
		return simpleResponse(400), nil
	}
	state := u.Query().Get("state")

	exists, err := h.mcpOAuthHandler.CallbackExists(ctx, state)
	if err != nil {
		log.Errorf("Could not check mcp oauth callback state. err: %v", err)
		return errorResponse(err), nil
	}

	res := response.V1ResponseMcpServersOAuthCallbackGet{
		Exists: exists,
	}

	data, err := json.Marshal(res)
	if err != nil {
		log.Errorf("Could not marshal the response message. message: %v, err: %v", res, err)
		return simpleResponse(500), nil
	}

	return &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}, nil
}

// processV1McpServersOAuthCompletePost handles POST /v1/mcp_servers/oauth/complete request
func (h *listenHandler) processV1McpServersOAuthCompletePost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"handler": "processV1McpServersOAuthCompletePost",
		"request": m,
	})

	var req request.V1DataMcpServersOAuthCompletePost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Errorf("Could not unmarshal the requested data. err: %v", err)
		return simpleResponse(400), nil
	}

	tmp, err := h.mcpOAuthHandler.Complete(ctx, req.CustomerID, req.State, req.Code)
	if err != nil {
		log.Errorf("Could not complete mcp oauth flow. err: %v", err)
		return errorResponse(err), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	return &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}, nil
}

// processV1McpServersIDDelete handles DELETE /v1/mcp_servers/<mcp-server-id> request
func (h *listenHandler) processV1McpServersIDDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"handler": "processV1McpServersIDDelete",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		log.Errorf("Wrong uri item count. uri_items: %d", len(uriItems))
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])
	if id == uuid.Nil {
		log.Errorf("Invalid mcp server ID.")
		return simpleResponse(400), nil
	}

	tmp, err := h.mcpServerHandler.Delete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete mcp server. err: %v", err)
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
