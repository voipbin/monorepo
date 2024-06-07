package listenhandler

import (
	"context"
	"encoding/json"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
	"net/url"
	"strconv"

	"github.com/sirupsen/logrus"
)

// processV1AgentsGet handles GET /v1/agents request
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
		log.Errorf("Could not get agents info. err: %v", err)
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
