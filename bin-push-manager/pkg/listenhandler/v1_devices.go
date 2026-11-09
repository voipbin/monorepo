package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-push-manager/models/device"
	"monorepo/bin-push-manager/pkg/listenhandler/models/request"
)

// jsonResponse returns the 200 response of the data.
func jsonResponse(data any) *sock.Response {
	m, err := json.Marshal(data)
	if err != nil {
		logrus.WithField("func", "jsonResponse").Errorf("Could not marshal the response message. err: %v", err)
		return simpleResponse(500)
	}

	return &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       m,
	}
}

// processV1DevicesPost handles POST /v1/devices request
func (h *listenHandler) processV1DevicesPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithField("func", "processV1DevicesPost")

	var req request.V1DataDevicesPost
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not unmarshal the request. err: %v", err)
		return simpleResponse(400), nil
	}

	// the registration errors are typed (400) or internal: both flow through the dispatcher
	res, err := h.deviceHandler.Register(ctx, req.CustomerID, req.AgentID, req.Platform, req.Token)
	if err != nil {
		return nil, err
	}

	return jsonResponse(res), nil
}

// processV1DevicesGet handles GET /v1/devices request
func (h *listenHandler) processV1DevicesGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	// parse the pagination params
	tmpSize, _ := strconv.Atoi(u.Query().Get(PageSize))
	pageSize := uint64(tmpSize)
	pageToken := u.Query().Get(PageToken)

	log := logrus.WithFields(logrus.Fields{
		"func":  "processV1DevicesGet",
		"size":  pageSize,
		"token": pageToken,
	})

	// get filters from request body
	tmpFilters, err := utilhandler.ParseFiltersFromRequestBody(m.Data)
	if err != nil {
		log.Errorf("Could not parse filters. err: %v", err)
		return simpleResponse(400), nil
	}

	// convert to typed filters
	filters, err := utilhandler.ConvertFilters[device.FieldStruct, device.Field](device.FieldStruct{}, tmpFilters)
	if err != nil {
		log.Errorf("Could not convert filters. err: %v", err)
		return simpleResponse(400), nil
	}

	tmp, err := h.deviceHandler.List(ctx, pageSize, pageToken, filters)
	if err != nil {
		log.Errorf("Could not get devices info. err: %v", err)
		return simpleResponse(500), nil
	}

	return jsonResponse(tmp), nil
}

// parseID returns the device id of the uri /v1/devices/<device-id>.
func parseID(uri string) (uuid.UUID, bool) {
	items := strings.Split(uri, "/")
	if len(items) < 4 {
		return uuid.Nil, false
	}

	return uuid.FromStringOrNil(items[3]), true
}

// processV1DevicesIDGet handles GET /v1/devices/<device-id> request
func (h *listenHandler) processV1DevicesIDGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	id, ok := parseID(m.URI)
	if !ok {
		return simpleResponse(400), nil
	}

	res, err := h.deviceHandler.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	return jsonResponse(res), nil
}

// processV1DevicesIDDelete handles DELETE /v1/devices/<device-id> request
func (h *listenHandler) processV1DevicesIDDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	id, ok := parseID(m.URI)
	if !ok {
		return simpleResponse(400), nil
	}

	res, err := h.deviceHandler.Delete(ctx, id)
	if err != nil {
		return nil, err
	}

	return jsonResponse(res), nil
}
