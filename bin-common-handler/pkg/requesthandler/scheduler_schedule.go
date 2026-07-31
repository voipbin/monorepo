package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"monorepo/bin-common-handler/models/sock"
	smexecution "monorepo/bin-scheduler-manager/models/execution"
	smschedule "monorepo/bin-scheduler-manager/models/schedule"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

// SchedulerV1ScheduleCreate sends a request to scheduler-manager
// to creating a schedule.
// it returns created schedule if it succeed.
func (r *requestHandler) SchedulerV1ScheduleCreate(
	ctx context.Context,
	customerID uuid.UUID,
	name string,
	detail string,
	cronExpr string,
	targetQueue string,
	targetURI string,
	targetMethod string,
	targetDataType string,
	targetData []byte,
	timeoutMS int,
	retryMax int,
	enabled bool,
) (*smschedule.Schedule, error) {
	uri := "/v1/schedules"

	data := struct {
		CustomerID     uuid.UUID       `json:"customer_id"`
		Name           string          `json:"name"`
		Detail         string          `json:"detail"`
		Cron           string          `json:"cron"`
		TargetQueue    string          `json:"target_queue"`
		TargetURI      string          `json:"target_uri"`
		TargetMethod   string          `json:"target_method"`
		TargetDataType string          `json:"target_data_type"`
		TargetData     json.RawMessage `json:"target_data"`
		TimeoutMS      int             `json:"timeout_ms"`
		RetryMax       int             `json:"retry_max"`
		Enabled        bool            `json:"enabled"`
	}{
		CustomerID:     customerID,
		Name:           name,
		Detail:         detail,
		Cron:           cronExpr,
		TargetQueue:    targetQueue,
		TargetURI:      targetURI,
		TargetMethod:   targetMethod,
		TargetDataType: targetDataType,
		TargetData:     targetData,
		TimeoutMS:      timeoutMS,
		RetryMax:       retryMax,
		Enabled:        enabled,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestScheduler(ctx, uri, sock.RequestMethodPost, "scheduler/schedules", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res smschedule.Schedule
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// SchedulerV1ScheduleGet sends a request to scheduler-manager
// to getting the schedule.
func (r *requestHandler) SchedulerV1ScheduleGet(ctx context.Context, id uuid.UUID) (*smschedule.Schedule, error) {
	uri := fmt.Sprintf("/v1/schedules/%s", id)

	tmp, err := r.sendRequestScheduler(ctx, uri, sock.RequestMethodGet, "scheduler/schedules", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res smschedule.Schedule
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// SchedulerV1ScheduleGets sends a request to scheduler-manager
// to getting the list of schedules.
func (r *requestHandler) SchedulerV1ScheduleGets(ctx context.Context, pageToken string, pageSize uint64, filters map[smschedule.Field]any) ([]smschedule.Schedule, error) {
	uri := fmt.Sprintf("/v1/schedules?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	m, err := json.Marshal(filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal filters")
	}

	tmp, err := r.sendRequestScheduler(ctx, uri, sock.RequestMethodGet, "scheduler/schedules", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res []smschedule.Schedule
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}

// SchedulerV1ScheduleUpdate sends a request to scheduler-manager
// to update the schedule info.
// it returns updated schedule if it succeed.
func (r *requestHandler) SchedulerV1ScheduleUpdate(ctx context.Context, id uuid.UUID, fields map[smschedule.Field]any) (*smschedule.Schedule, error) {
	uri := fmt.Sprintf("/v1/schedules/%s", id)

	m, err := json.Marshal(fields)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestScheduler(ctx, uri, sock.RequestMethodPut, "scheduler/schedules", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res smschedule.Schedule
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// SchedulerV1ScheduleDelete sends a request to scheduler-manager
// to deleting the schedule.
func (r *requestHandler) SchedulerV1ScheduleDelete(ctx context.Context, id uuid.UUID) (*smschedule.Schedule, error) {
	uri := fmt.Sprintf("/v1/schedules/%s", id)

	tmp, err := r.sendRequestScheduler(ctx, uri, sock.RequestMethodDelete, "scheduler/schedules", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res smschedule.Schedule
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// SchedulerV1ScheduleExecute sends a request to scheduler-manager
// to execute the schedule manually.
// it returns created execution if it succeed.
func (r *requestHandler) SchedulerV1ScheduleExecute(ctx context.Context, id uuid.UUID) (*smexecution.Execution, error) {
	uri := fmt.Sprintf("/v1/schedules/%s/execute", id)

	tmp, err := r.sendRequestScheduler(ctx, uri, sock.RequestMethodPost, "scheduler/schedules", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res smexecution.Execution
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// SchedulerV1ExecutionGets sends a request to scheduler-manager
// to getting the list of executions.
func (r *requestHandler) SchedulerV1ExecutionGets(ctx context.Context, pageToken string, pageSize uint64, filters map[smexecution.Field]any) ([]smexecution.Execution, error) {
	uri := fmt.Sprintf("/v1/executions?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	m, err := json.Marshal(filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal filters")
	}

	tmp, err := r.sendRequestScheduler(ctx, uri, sock.RequestMethodGet, "scheduler/executions", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res []smexecution.Execution
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}
