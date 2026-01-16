package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
	"time"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-customer-manager/models/accesskey"
	"monorepo/bin-customer-manager/pkg/listenhandler/models/request"
)

// processV1AccesskeysGet handles GET /v1/accesskeys request
func (h *listenHandler) processV1AccesskeysGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1AccesskeysGet",
		"request": m,
	})
	log.Debug("Executing processV1AccesskeysGet.")

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
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
	filters, err := utilhandler.ConvertFilters[accesskey.FieldStruct, accesskey.Field](accesskey.FieldStruct{}, tmpFilters)
	if err != nil {
		log.Errorf("Could not convert filters. err: %v", err)
		return simpleResponse(400), nil
	}

	tmp, err := h.accesskeyHandler.List(ctx, pageSize, pageToken, filters)
	if err != nil {
		log.Errorf("Could not get accesskyes info. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}
	log.Debugf("Sending result: %v", data)

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1AccesskeysPost handles Post /v1/accesskeys request
func (h *listenHandler) processV1AccesskeysPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1AccesskeysPost",
		"request": m,
	})
	log.Debug("Executing processV1AccesskeysPost.")

	var reqData request.V1DataAccesskeysPost
	if err := json.Unmarshal([]byte(m.Data), &reqData); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	expire := time.Second * time.Duration(reqData.Expire)
	tmp, err := h.accesskeyHandler.Create(
		ctx,
		reqData.CustomerID,
		reqData.Name,
		reqData.Detail,
		expire,
	)
	if err != nil {
		log.Errorf("Could not create the accesskye info. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the result data. data: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}
	log.Debugf("Sending result: %v", data)

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1AccesskeysIDGet handles Get /v1/accesskeys/<accesskeys-id> request
func (h *listenHandler) processV1AccesskeysIDGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(logrus.Fields{
		"func":         "processV1AccesskeysIDGet",
		"accesskey_id": id,
	})
	log.Debug("Executing processV1AccesskeysIDGet.")

	tmp, err := h.accesskeyHandler.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not update the accesskey info. err: %v", err)
		return simpleResponse(400), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the result data. data: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}
	log.Debugf("Sending result: %v", data)

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1AccesskeysIDDelete handles Delete /v1/accesskeys/<accesskey-id> request
func (h *listenHandler) processV1AccesskeysIDDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(logrus.Fields{
		"func":         "processV1AccesskeysIDDelete",
		"accesskey_id": id,
	})

	tmp, err := h.accesskeyHandler.Delete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete the accesskey info. err: %v", err)
		return simpleResponse(400), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the result data. data: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}
	log.Debugf("Sending result: %v", data)

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1AccesskeysIDPut handles Put /v1/accesskeys/<accesskey-id> request
func (h *listenHandler) processV1AccesskeysIDPut(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	log := logrus.WithFields(logrus.Fields{
		"func":         "processV1AccesskeysIDPut",
		"accesskey_id": id,
	})

	var reqData request.V1DataAccesskeysIDPut
	if err := json.Unmarshal([]byte(m.Data), &reqData); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	tmp, err := h.accesskeyHandler.UpdateBasicInfo(
		ctx,
		id,
		reqData.Name,
		reqData.Detail,
	)
	if err != nil {
		log.Errorf("Could not update the accesskey info. err: %v", err)
		return simpleResponse(400), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the result data. data: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}
	log.Debugf("Sending result: %v", data)

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
