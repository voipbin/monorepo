package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"monorepo/bin-ai-manager/pkg/listenhandler/models/request"
	"monorepo/bin-common-handler/models/sock"

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

	// get filters
	filters := getFilters(u)

	log = log.WithFields(logrus.Fields{
		"size":    pageSize,
		"token":   pageToken,
		"filters": filters,
	})

	tmp, err := h.aicallHandler.Gets(ctx, pageSize, pageToken, filters)
	if err != nil {
		log.Debugf("Could not get conferences. err: %v", err)
		return simpleResponse(500), nil
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

	tmp, err := h.aicallHandler.Start(ctx, req.AIID, req.ActiveflowID, req.ReferenceType, req.ReferenceID, req.Gender, req.Language, false)
	if err != nil {
		log.Errorf("Could not create aicall. err: %v", err)
		return simpleResponse(500), nil
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

// processV1AIcallsIDGet handles GET /v1/aicalls/<aicall-id> request
func (h *listenHandler) processV1AIcallsIDGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"handler": "processV1AIcallsIDGet",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		log.Errorf("Wrong uri item count. uri_items: %d", len(uriItems))
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])

	tmp, err := h.aicallHandler.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get ai. err: %v", err)
		return simpleResponse(500), nil
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

	tmp, err := h.aicallHandler.Delete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete aicall. err: %v", err)
		return simpleResponse(500), nil
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
