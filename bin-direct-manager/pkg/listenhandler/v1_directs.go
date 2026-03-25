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

	"monorepo/bin-direct-manager/models/direct"
	"monorepo/bin-direct-manager/pkg/listenhandler/models/request"
)

// processV1DirectsGet handles GET /v1/directs request
func (h *listenHandler) processV1DirectsGet(ctx context.Context, req *sock.Request) (*sock.Response, error) {

	u, err := url.Parse(req.URI)
	if err != nil {
		return nil, err
	}

	// parse the pagination params
	tmpSize, _ := strconv.Atoi(u.Query().Get(PageSize))
	pageSize := uint64(tmpSize)
	pageToken := u.Query().Get(PageToken)

	log := logrus.WithFields(logrus.Fields{
		"func":  "processV1DirectsGet",
		"size":  pageSize,
		"token": pageToken,
	})
	log.WithField("request", req).Debug("Received request.")

	// get filters from request body
	tmpFilters, err := utilhandler.ParseFiltersFromRequestBody(req.Data)
	if err != nil {
		log.Errorf("Could not parse filters. err: %v", err)
		return simpleResponse(400), nil
	}

	// convert to typed filters
	filters, err := utilhandler.ConvertFilters[direct.FieldStruct, direct.Field](direct.FieldStruct{}, tmpFilters)
	if err != nil {
		log.Errorf("Could not convert filters. err: %v", err)
		return simpleResponse(400), nil
	}

	tmp, err := h.directHandler.Gets(ctx, pageSize, pageToken, filters)
	if err != nil {
		log.Errorf("Could not get directs info. err:%v", err)
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

// processV1DirectsIDGet handles GET /v1/directs/{direct-id} request
func (h *listenHandler) processV1DirectsIDGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"func":      "processV1DirectsIDGet",
			"direct_id": id,
		})
	log.WithField("request", m).Debug("Received request.")

	tmp, err := h.directHandler.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get a direct info. err: %v", err)
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

// processV1DirectsByHashGet handles GET /v1/directs/by-hash/{hash} request
func (h *listenHandler) processV1DirectsByHashGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		return simpleResponse(400), nil
	}

	hash := uriItems[4]
	log := logrus.WithFields(
		logrus.Fields{
			"func": "processV1DirectsByHashGet",
			"hash": hash,
		})
	log.WithField("request", m).Debug("Received request.")

	tmp, err := h.directHandler.GetByHash(ctx, hash)
	if err != nil {
		log.Errorf("Could not get a direct by hash. err: %v", err)
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

// processV1DirectsPost handles POST /v1/directs request
func (h *listenHandler) processV1DirectsPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "processV1DirectsPost",
		})
	log.WithField("request", m).Debug("Received request.")

	var reqData request.V1DataDirectsPost
	if err := json.Unmarshal([]byte(m.Data), &reqData); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}
	log = log.WithFields(logrus.Fields{
		"customer_id": reqData.CustomerID,
	})
	log.WithField("request", reqData).Debug("Creating a direct.")

	tmp, err := h.directHandler.Create(
		ctx,
		reqData.CustomerID,
		reqData.ResourceType,
		reqData.ResourceID,
	)
	if err != nil {
		log.Errorf("Could not create a direct info. err: %v", err)
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

// processV1DirectsIDDelete handles DELETE /v1/directs/{direct-id} request
func (h *listenHandler) processV1DirectsIDDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"func":      "processV1DirectsIDDelete",
			"direct_id": id,
		})
	log.WithField("request", m).Debug("Received request.")

	tmp, err := h.directHandler.Delete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete the direct info. err: %v", err)
		return simpleResponse(400), nil
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

// processV1DirectsIDRegenerate handles POST /v1/directs/{direct-id}/regenerate request
func (h *listenHandler) processV1DirectsIDRegenerate(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"func":      "processV1DirectsIDRegenerate",
			"direct_id": id,
		})
	log.WithField("request", m).Debug("Received request.")

	tmp, err := h.directHandler.Regenerate(ctx, id)
	if err != nil {
		log.Errorf("Could not regenerate the direct hash. err: %v", err)
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
