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

	"monorepo/bin-tag-manager/models/tag"
	"monorepo/bin-tag-manager/pkg/listenhandler/models/request"
)

// processV1TagsGet handles GET /v1/tags request
func (h *listenHandler) processV1TagsGet(ctx context.Context, req *sock.Request) (*sock.Response, error) {

	u, err := url.Parse(req.URI)
	if err != nil {
		return nil, err
	}

	// parse the pagination params
	tmpSize, _ := strconv.Atoi(u.Query().Get(PageSize))
	pageSize := uint64(tmpSize)
	pageToken := u.Query().Get(PageToken)

	log := logrus.WithFields(logrus.Fields{
		"func":  "processV1TagsGet",
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
	filters, err := utilhandler.ConvertFilters[tag.FieldStruct, tag.Field](tag.FieldStruct{}, tmpFilters)
	if err != nil {
		log.Errorf("Could not convert filters. err: %v", err)
		return simpleResponse(400), nil
	}

	tmp, err := h.tagHandler.Gets(ctx, pageSize, pageToken, filters)
	if err != nil {
		log.Errorf("Could not get tags info. err:%v", err)
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

// processV1TagsIDGet handles Get /v1/tags/<tag-id> request
func (h *listenHandler) processV1TagsIDGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"func":   "processV1TagsIDGet",
			"tag_id": id,
		})
	log.WithField("request", m).Debug("Received request.")

	tmp, err := h.tagHandler.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get an tag info. err: %v", err)
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

// processV1TagsIDPut handles Put /v1/tags/<tag-id> request
func (h *listenHandler) processV1TagsIDPut(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"func":   "processV1TagsIDPut",
			"tag_id": id,
		})
	log.WithField("request", m).Debug("Received request.")

	var reqData request.V1DataTagsIDPut
	if err := json.Unmarshal([]byte(m.Data), &reqData); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}
	log.WithField("request", reqData).Debug("Updating the tag.")

	tmp, err := h.tagHandler.UpdateBasicInfo(ctx, id, reqData.Name, reqData.Detail)
	if err != nil {
		log.Errorf("Could not update the tag info. err: %v", err)
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

// processV1TagPost handles Post /v1/tags request
func (h *listenHandler) processV1TagsPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "processV1TagPost",
		})
	log.WithField("request", m).Debug("Received request.")

	var reqData request.V1DataTagsPost
	if err := json.Unmarshal([]byte(m.Data), &reqData); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}
	log = log.WithFields(logrus.Fields{
		"customer_id": reqData.CustomerID,
	})
	log.WithField("request", reqData).Debug("Creating a tag.")

	// create an agent
	tmp, err := h.tagHandler.Create(
		ctx,
		reqData.CustomerID,
		reqData.Name,
		reqData.Detail,
	)
	if err != nil {
		log.Errorf("Could not create a tag info. err: %v", err)
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

// processV1TagsIDDelete handles Delete /v1/tag/<tag_id> request
func (h *listenHandler) processV1TagsIDDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"func":   "processV1TagsIDDelete",
			"tag_id": id,
		})
	log.WithField("request", m).Debug("Received request.")

	tmp, err := h.tagHandler.Delete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete the tag info. err: %v", err)
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
