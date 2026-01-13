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

	"monorepo/bin-conference-manager/models/conferencecall"
	"monorepo/bin-conference-manager/pkg/listenhandler/models/request"
)

// processV1ConferencecallsGet handles GET /v1/conferencecalls request
func (h *listenHandler) processV1ConferencecallsGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1ConferencecallsGet",
		"request": m,
	})

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	// parse the pagination params
	tmpSize, _ := strconv.Atoi(u.Query().Get(PageSize))
	pageSize := uint64(tmpSize)
	pageToken := u.Query().Get(PageToken)

	// get filters
	tmpFilters := h.utilHandler.URLParseFilters(u)
	filters, err := conferencecall.ConvertStringMapToFieldMap(tmpFilters)
	if err != nil {
		log.Errorf("Could not convert filters. err: %v", err)
		return simpleResponse(400), nil
	}

	confs, err := h.conferencecallHandler.Gets(ctx, pageSize, pageToken, filters)
	if err != nil {
		log.Debugf("Could not get conferencecalls. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(confs)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", confs, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1ConferencecallsIDGet handles /v1/conferencecalls/<id> GET request
func (h *listenHandler) processV1ConferencecallsIDGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1ConferencecallsIDGet",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		log.Errorf("Wrong uri item count. uri_items: %d", len(uriItems))
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])

	cc, err := h.conferencecallHandler.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not remove the call from the conference. err: %v", err)
		return simpleResponse(400), nil
	}

	tmp, err := json.Marshal(cc)
	if err != nil {
		log.Errorf("Could not marshal the conferencecall. err: %v", err)
		return simpleResponse(400), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       tmp,
	}

	return res, nil
}

// processV1ConferencecallsIDDelete handles /v1/conferencecalls/<id> DELETE request
func (h *listenHandler) processV1ConferencecallsIDDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1ConferencecallsIDDelete",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		log.Errorf("Wrong uri item count. uri_items: %d", len(uriItems))
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])

	cc, err := h.conferencecallHandler.Terminate(ctx, id)
	if err != nil {
		log.Errorf("Could not remove the call from the conference. err: %v", err)
		return simpleResponse(400), nil
	}

	tmp, err := json.Marshal(cc)
	if err != nil {
		log.Errorf("Could not marshal the conferencecall. err: %v", err)
		return simpleResponse(400), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       tmp,
	}

	return res, nil
}

// processV1ConferencecallsIDHealthCheckPost handles /v1/conferencecalls/<id>/health-check POST request
func (h *listenHandler) processV1ConferencecallsIDHealthCheckPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1ConferencecallsIDHealthCheckPost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		log.Errorf("Wrong uri item count. uri_items: %d", len(uriItems))
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])

	var data request.V1DataConferencecallsIDHealthCheckPost
	if err := json.Unmarshal([]byte(m.Data), &data); err != nil {
		log.Errorf("Could not unmarshal the requested data. err: %v", err)
		return nil, err
	}

	// health check run in a go routine
	go h.conferencecallHandler.HealthCheck(ctx, id, data.RetryCount)

	res := &sock.Response{
		StatusCode: 200,
	}

	return res, nil
}
