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

	"monorepo/bin-schedule-manager/models/schedule"
	"monorepo/bin-schedule-manager/pkg/listenhandler/models/request"
)

// processV1SchedulesPost handles POST /v1/schedules request
func (h *listenHandler) processV1SchedulesPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "processV1SchedulesPost",
		})
	log.WithField("request", m).Debug("Received request.")

	var reqData request.V1DataSchedulesPost
	if err := json.Unmarshal([]byte(m.Data), &reqData); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}
	log = log.WithFields(logrus.Fields{
		"customer_id": reqData.CustomerID,
		"name":        reqData.Name,
	})
	log.WithField("request", reqData).Debug("Creating a schedule.")

	tmp, err := h.scheduleHandler.Create(
		ctx,
		reqData.CustomerID,
		reqData.Name,
		reqData.Detail,
		reqData.Cron,
		reqData.TargetQueue,
		reqData.TargetURI,
		reqData.TargetMethod,
		reqData.TargetDataType,
		reqData.TargetData,
		reqData.TimeoutMS,
		reqData.RetryMax,
		reqData.Enabled,
	)
	if err != nil {
		log.Errorf("Could not create the schedule. err: %v", err)
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

// processV1SchedulesGet handles GET /v1/schedules request
func (h *listenHandler) processV1SchedulesGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	// parse the pagination params
	tmpSize, _ := strconv.Atoi(u.Query().Get(PageSize))
	pageSize := uint64(tmpSize)
	pageToken := u.Query().Get(PageToken)

	log := logrus.WithFields(logrus.Fields{
		"func":  "processV1SchedulesGet",
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
	filters, err := utilhandler.ConvertFilters[schedule.FieldStruct, schedule.Field](schedule.FieldStruct{}, tmpFilters)
	if err != nil {
		log.Errorf("Could not convert filters. err: %v", err)
		return simpleResponse(400), nil
	}

	tmp, err := h.scheduleHandler.Gets(ctx, pageSize, pageToken, filters)
	if err != nil {
		log.Errorf("Could not get schedules info. err: %v", err)
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

// processV1SchedulesIDGet handles GET /v1/schedules/<schedule-id> request
func (h *listenHandler) processV1SchedulesIDGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "processV1SchedulesIDGet",
			"schedule_id": id,
		})
	log.WithField("request", m).Debug("Received request.")

	tmp, err := h.scheduleHandler.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get the schedule info. err: %v", err)
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

// updateFieldsFromRequest rebuilds the partial update field set from the
// pointer-typed wire DTO — only present keys are forwarded to the handler.
func updateFieldsFromRequest(reqData *request.V1DataSchedulesIDPut) map[schedule.Field]any {
	fields := map[schedule.Field]any{}

	if reqData.Name != nil {
		fields[schedule.FieldName] = *reqData.Name
	}
	if reqData.Detail != nil {
		fields[schedule.FieldDetail] = *reqData.Detail
	}
	if reqData.Cron != nil {
		fields[schedule.FieldCron] = *reqData.Cron
	}
	if reqData.TargetQueue != nil {
		fields[schedule.FieldTargetQueue] = *reqData.TargetQueue
	}
	if reqData.TargetURI != nil {
		fields[schedule.FieldTargetURI] = *reqData.TargetURI
	}
	if reqData.TargetMethod != nil {
		fields[schedule.FieldTargetMethod] = *reqData.TargetMethod
	}
	if reqData.TargetDataType != nil {
		fields[schedule.FieldTargetDataType] = *reqData.TargetDataType
	}
	if reqData.TargetData != nil {
		fields[schedule.FieldTargetData] = *reqData.TargetData
	}
	if reqData.TimeoutMS != nil {
		fields[schedule.FieldTimeoutMS] = *reqData.TimeoutMS
	}
	if reqData.RetryMax != nil {
		fields[schedule.FieldRetryMax] = *reqData.RetryMax
	}
	if reqData.Enabled != nil {
		fields[schedule.FieldEnabled] = *reqData.Enabled
	}

	return fields
}

// processV1SchedulesIDPut handles PUT /v1/schedules/<schedule-id> request
func (h *listenHandler) processV1SchedulesIDPut(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "processV1SchedulesIDPut",
			"schedule_id": id,
		})
	log.WithField("request", m).Debug("Received request.")

	var reqData request.V1DataSchedulesIDPut
	if err := json.Unmarshal([]byte(m.Data), &reqData); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}
	log.WithField("request", reqData).Debug("Updating the schedule.")

	fields := updateFieldsFromRequest(&reqData)

	tmp, err := h.scheduleHandler.Update(ctx, id, fields)
	if err != nil {
		log.Errorf("Could not update the schedule info. err: %v", err)
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

// processV1SchedulesIDDelete handles DELETE /v1/schedules/<schedule-id> request
func (h *listenHandler) processV1SchedulesIDDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "processV1SchedulesIDDelete",
			"schedule_id": id,
		})
	log.WithField("request", m).Debug("Received request.")

	tmp, err := h.scheduleHandler.Delete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete the schedule info. err: %v", err)
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

// processV1SchedulesIDExecutePost handles POST /v1/schedules/<schedule-id>/execute request.
// A concurrent run in flight is refused with a 409 (FailedPrecondition,
// design §6 manual execute semantics) via errorResponse.
func (h *listenHandler) processV1SchedulesIDExecutePost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "processV1SchedulesIDExecutePost",
			"schedule_id": id,
		})
	log.WithField("request", m).Debug("Received request.")

	tmp, err := h.dispatchHandler.ExecuteManual(ctx, id)
	if err != nil {
		log.Errorf("Could not execute the schedule manually. err: %v", err)
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
