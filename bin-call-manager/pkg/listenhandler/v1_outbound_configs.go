package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	outboundconfig "monorepo/bin-call-manager/models/outboundconfig"
	"monorepo/bin-call-manager/pkg/listenhandler/models/request"
)

// processV1OutboundConfigsPost handles POST /v1/outbound_configs
func (h *listenHandler) processV1OutboundConfigsPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1OutboundConfigsPost",
		"request": m,
	})

	var req request.V1DataOutboundConfigsPost
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	c, err := h.outboundConfigHandler.Create(ctx, req.CustomerID, &req.Request)
	if err != nil {
		log.Errorf("Could not create outbound config. err: %v", err)
		if strings.Contains(err.Error(), "Duplicate entry") {
			return simpleResponse(409), nil
		}
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(outboundconfig.ConvertWebhookMessage(c))
	if err != nil {
		log.Errorf("Could not marshal the response message. message: %v, err: %v", c, err)
		return simpleResponse(500), nil
	}

	return &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}, nil
}

// processV1OutboundConfigsGet handles GET /v1/outbound_configs?customer_id=<uuid>&page_size=10&page_token=...
func (h *listenHandler) processV1OutboundConfigsGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1OutboundConfigsGet",
		"request": m,
	})

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	// parse pagination params
	tmpSize, _ := strconv.Atoi(u.Query().Get(PageSize))
	pageSize := uint64(tmpSize)
	pageToken := u.Query().Get(PageToken)

	// parse customer_id
	customerID := uuid.FromStringOrNil(u.Query().Get("customer_id"))

	configs, err := h.outboundConfigHandler.List(ctx, customerID, pageSize, pageToken)
	if err != nil {
		log.Errorf("Could not list outbound configs. err: %v", err)
		return simpleResponse(500), nil
	}

	// convert to webhook messages
	wms := make([]*outboundconfig.WebhookMessage, len(configs))
	for i, c := range configs {
		wms[i] = outboundconfig.ConvertWebhookMessage(c)
	}

	data, err := json.Marshal(wms)
	if err != nil {
		log.Errorf("Could not marshal the response message. message: %v, err: %v", wms, err)
		return simpleResponse(500), nil
	}

	return &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}, nil
}

// processV1OutboundConfigsIDGet handles GET /v1/outbound_configs/<uuid>
func (h *listenHandler) processV1OutboundConfigsIDGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1OutboundConfigsIDGet",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	c, err := h.outboundConfigHandler.GetByID(ctx, id)
	if err != nil {
		log.Errorf("Could not get outbound config. err: %v", err)
		return errorResponse(err), nil
	}

	data, err := json.Marshal(outboundconfig.ConvertWebhookMessage(c))
	if err != nil {
		log.Errorf("Could not marshal the response message. message: %v, err: %v", c, err)
		return simpleResponse(500), nil
	}

	return &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}, nil
}

// processV1OutboundConfigsIDDelete handles DELETE /v1/outbound_configs/<uuid>
func (h *listenHandler) processV1OutboundConfigsIDDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1OutboundConfigsIDDelete",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	c, err := h.outboundConfigHandler.Delete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete outbound config. err: %v", err)
		return errorResponse(err), nil
	}

	data, err := json.Marshal(outboundconfig.ConvertWebhookMessage(c))
	if err != nil {
		log.Errorf("Could not marshal the response message. message: %v, err: %v", c, err)
		return simpleResponse(500), nil
	}

	return &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}, nil
}

// processV1OutboundConfigsIDPut handles PUT /v1/outbound_configs/<uuid>
func (h *listenHandler) processV1OutboundConfigsIDPut(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1OutboundConfigsIDPut",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	var req request.V1DataOutboundConfigsIDPut
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	c, err := h.outboundConfigHandler.Update(ctx, id, &req.Request)
	if err != nil {
		log.Errorf("Could not update outbound config. err: %v", err)
		return errorResponse(err), nil
	}

	data, err := json.Marshal(outboundconfig.ConvertWebhookMessage(c))
	if err != nil {
		log.Errorf("Could not marshal the response message. message: %v, err: %v", c, err)
		return simpleResponse(500), nil
	}

	return &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}, nil
}
