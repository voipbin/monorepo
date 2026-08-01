package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/sirupsen/logrus"

	"monorepo/bin-schedule-manager/models/execution"
	"monorepo/bin-schedule-manager/pkg/listenhandler/models/response"
)

// processV1ExecutionsGet handles GET /v1/executions request
func (h *listenHandler) processV1ExecutionsGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	// parse the pagination params
	tmpSize, _ := strconv.Atoi(u.Query().Get(PageSize))
	pageSize := uint64(tmpSize)
	pageToken := u.Query().Get(PageToken)

	log := logrus.WithFields(logrus.Fields{
		"func":  "processV1ExecutionsGet",
		"size":  pageSize,
		"token": pageToken,
	})
	log.WithField("request", m).Debug("Received request.")

	// get filters from request body
	tmpFilters, err := utilhandler.ParseFiltersFromRequestBody(m.Data)
	if err != nil {
		log.Errorf("Could not parse filters. err: %v", err)
		return simpleResponse(400), nil
	}

	// convert to typed filters
	filters, err := utilhandler.ConvertFilters[execution.FieldStruct, execution.Field](execution.FieldStruct{}, tmpFilters)
	if err != nil {
		log.Errorf("Could not convert filters. err: %v", err)
		return simpleResponse(400), nil
	}

	tmp, err := h.scheduleHandler.ExecutionGets(ctx, pageSize, pageToken, filters)
	if err != nil {
		log.Errorf("Could not get executions info. err: %v", err)
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

// processV1ExecutionsPrunePost handles POST /v1/executions/prune request.
// It removes execution rows older than the configured retention (design §6:
// batched deletes, invoked by the seeded execution-retention schedule).
func (h *listenHandler) processV1ExecutionsPrunePost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "processV1ExecutionsPrunePost",
		"retention_days": h.executionRetentionDays,
	})
	log.WithField("request", m).Debug("Received request.")

	removed, err := h.scheduleHandler.ExecutionPrune(ctx, h.executionRetentionDays)
	if err != nil {
		log.Errorf("Could not prune the executions. err: %v", err)
		return errorResponse(err), nil
	}

	tmp := &response.V1ResponseExecutionsPrune{
		Removed: removed,
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
