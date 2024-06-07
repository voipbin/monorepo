package listenhandler

import (
	"context"
	"encoding/json"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
	"net/url"
	"strconv"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// processV1ResourcesGet handles GET /v1/resources request
func (h *listenHandler) processV1ResourcesGet(ctx context.Context, req *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {

	u, err := url.Parse(req.URI)
	if err != nil {
		return nil, err
	}

	// parse the pagination params
	tmpSize, _ := strconv.Atoi(u.Query().Get(PageSize))
	pageSize := uint64(tmpSize)
	pageToken := u.Query().Get(PageToken)

	// parse the filters
	filters := h.utilHandler.URLParseFilters(u)

	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1ResourcesGet",
		"size":    pageSize,
		"token":   pageToken,
		"filters": filters,
	})

	tmp, err := h.resourceHandler.Gets(ctx, pageSize, pageToken, filters)
	if err != nil {
		log.Errorf("Could not get resources info. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1ResourcesIDGet handles Get /v1/resources/<resource-id> request
func (h *listenHandler) processV1ResourcesIDGet(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(logrus.Fields{
		"func":        "processV1ResourcesIDGet",
		"resource_id": id,
	})
	log.Debug("Executing processV1ResourcesIDGet.")

	tmp, err := h.resourceHandler.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get an resource info. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1ResourcesIDDelete handles Delete /v1/resources/<resource-id> request
func (h *listenHandler) processV1ResourcesIDDelete(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(logrus.Fields{
		"func":        "processV1ResourcesIDDelete",
		"resource_id": id,
	})
	log.Debug("Executing processV1ResourcesIDDelete.")

	tmp, err := h.resourceHandler.Delete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete the resource info. err: %v", err)
		return simpleResponse(400), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
