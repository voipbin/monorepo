package listenhandler

import (
	"encoding/json"
	"net/http"
	"reflect"
	"testing"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-schedule-manager/models/execution"
	"monorepo/bin-schedule-manager/models/schedule"
	"monorepo/bin-schedule-manager/pkg/dispatchhandler"
	"monorepo/bin-schedule-manager/pkg/schedulehandler"
)

func setupListenHandler(t *testing.T) (*gomock.Controller, *listenHandler, *schedulehandler.MockScheduleHandler, *dispatchhandler.MockDispatchHandler) {
	mc := gomock.NewController(t)

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockSchedule := schedulehandler.NewMockScheduleHandler(mc)
	mockDispatch := dispatchhandler.NewMockDispatchHandler(mc)

	h := &listenHandler{
		sockHandler: mockSock,

		scheduleHandler: mockSchedule,
		dispatchHandler: mockDispatch,

		executionRetentionDays: 90,
	}

	return mc, h, mockSchedule, mockDispatch
}

func Test_processV1SchedulesPost(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		customerID     uuid.UUID
		scheduleName   string
		detail         string
		cronExpr       string
		targetQueue    string
		targetURI      string
		targetMethod   string
		targetDataType string
		targetData     json.RawMessage
		timeoutMS      int
		retryMax       int
		enabled        bool

		responseSchedule *schedule.Schedule
		expectRes        *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/schedules",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				// pinned client shape: requesthandler.ScheduleV1ScheduleCreate
				Data: []byte(`{"customer_id":"00000000-0000-0000-0000-000000000000","name":"number-renew","detail":"daily number renew","cron":"0 1 * * *","target_queue":"bin-manager.number-manager.request","target_uri":"/v1/numbers/renew","target_method":"POST","target_data_type":"application/json","target_data":{"days":28},"timeout_ms":300000,"retry_max":0,"enabled":true}`),
			},

			customerID:     uuid.Nil,
			scheduleName:   "number-renew",
			detail:         "daily number renew",
			cronExpr:       "0 1 * * *",
			targetQueue:    "bin-manager.number-manager.request",
			targetURI:      "/v1/numbers/renew",
			targetMethod:   "POST",
			targetDataType: "application/json",
			targetData:     json.RawMessage(`{"days":28}`),
			timeoutMS:      300000,
			retryMax:       0,
			enabled:        true,

			responseSchedule: &schedule.Schedule{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cc8c1af8-6f20-11f0-8f4a-3f8a53bd0fd9"),
				},
				Name: "number-renew",
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"cc8c1af8-6f20-11f0-8f4a-3f8a53bd0fd9","customer_id":"00000000-0000-0000-0000-000000000000","name":"number-renew","detail":"","type":"","cron":"","target_queue":"","target_uri":"","target_method":"","target_data_type":"","target_data":null,"timeout_ms":0,"retry_max":0,"enabled":false,"tm_next_run":null,"tm_last_run":null,"tm_create":null,"tm_update":null,"tm_delete":null}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc, h, mockSchedule, _ := setupListenHandler(t)
			defer mc.Finish()

			mockSchedule.EXPECT().Create(
				gomock.Any(),
				tt.customerID,
				tt.scheduleName,
				tt.detail,
				tt.cronExpr,
				tt.targetQueue,
				tt.targetURI,
				tt.targetMethod,
				tt.targetDataType,
				tt.targetData,
				tt.timeoutMS,
				tt.retryMax,
				tt.enabled,
			).Return(tt.responseSchedule, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1SchedulesPost_badJSON(t *testing.T) {
	mc, h, _, _ := setupListenHandler(t)
	defer mc.Finish()

	res, err := h.processRequest(&sock.Request{
		URI:      "/v1/schedules",
		Method:   sock.RequestMethodPost,
		DataType: "application/json",
		Data:     []byte(`{invalid json`),
	})
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("Wrong match. expect: 400, got: %d", res.StatusCode)
	}
}

func Test_processV1SchedulesGet(t *testing.T) {
	customerID := uuid.FromStringOrNil("6e3a0dfe-6f21-11f0-84ea-7feed12f6cb2")

	tests := []struct {
		name    string
		request *sock.Request

		pageSize  uint64
		pageToken string
		filters   map[schedule.Field]any

		responseSchedules []*schedule.Schedule
		expectRes         *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/schedules?page_token=2026-08-01T00:00:00.000000Z&page_size=10",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"6e3a0dfe-6f21-11f0-84ea-7feed12f6cb2","deleted":false}`),
			},

			pageSize:  10,
			pageToken: "2026-08-01T00:00:00.000000Z",
			filters: map[schedule.Field]any{
				schedule.FieldCustomerID: customerID,
				schedule.FieldDeleted:    false,
			},

			responseSchedules: []*schedule.Schedule{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("6ec0a4b2-6f21-11f0-8dc9-6b46ff17f7b3"),
						CustomerID: customerID,
					},
				},
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"6ec0a4b2-6f21-11f0-8dc9-6b46ff17f7b3","customer_id":"6e3a0dfe-6f21-11f0-84ea-7feed12f6cb2","name":"","detail":"","type":"","cron":"","target_queue":"","target_uri":"","target_method":"","target_data_type":"","target_data":null,"timeout_ms":0,"retry_max":0,"enabled":false,"tm_next_run":null,"tm_last_run":null,"tm_create":null,"tm_update":null,"tm_delete":null}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc, h, mockSchedule, _ := setupListenHandler(t)
			defer mc.Finish()

			mockSchedule.EXPECT().Gets(gomock.Any(), tt.pageSize, tt.pageToken, tt.filters).Return(tt.responseSchedules, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1SchedulesIDGet(t *testing.T) {
	scheduleID := uuid.FromStringOrNil("b74b6f4c-6f21-11f0-8c86-2b3f1f10a2f8")

	mc, h, mockSchedule, _ := setupListenHandler(t)
	defer mc.Finish()

	responseSchedule := &schedule.Schedule{
		Identity: commonidentity.Identity{
			ID: scheduleID,
		},
	}

	mockSchedule.EXPECT().Get(gomock.Any(), scheduleID).Return(responseSchedule, nil)

	res, err := h.processRequest(&sock.Request{
		URI:    "/v1/schedules/b74b6f4c-6f21-11f0-8c86-2b3f1f10a2f8",
		Method: sock.RequestMethodGet,
	})
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	expectRes := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       []byte(`{"id":"b74b6f4c-6f21-11f0-8c86-2b3f1f10a2f8","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","type":"","cron":"","target_queue":"","target_uri":"","target_method":"","target_data_type":"","target_data":null,"timeout_ms":0,"retry_max":0,"enabled":false,"tm_next_run":null,"tm_last_run":null,"tm_create":null,"tm_update":null,"tm_delete":null}`),
	}
	if !reflect.DeepEqual(res, expectRes) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expectRes, res)
	}
}

func Test_processV1SchedulesIDGet_notFound(t *testing.T) {
	scheduleID := uuid.FromStringOrNil("d0e14e34-6f21-11f0-9b93-9316cd2b6d9a")

	mc, h, mockSchedule, _ := setupListenHandler(t)
	defer mc.Finish()

	mockSchedule.EXPECT().Get(gomock.Any(), scheduleID).Return(nil, cerrors.NotFound(commonoutline.ServiceNameScheduleManager, "SCHEDULE_NOT_FOUND", "The schedule was not found."))

	res, err := h.processRequest(&sock.Request{
		URI:    "/v1/schedules/d0e14e34-6f21-11f0-9b93-9316cd2b6d9a",
		Method: sock.RequestMethodGet,
	})
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	if res.StatusCode != http.StatusNotFound {
		t.Errorf("Wrong match. expect: 404, got: %d", res.StatusCode)
	}
}

func Test_processV1SchedulesIDPut(t *testing.T) {
	scheduleID := uuid.FromStringOrNil("e8e3902a-6f21-11f0-b1a1-2f74a58c4c0e")

	tests := []struct {
		name    string
		request *sock.Request

		fields map[schedule.Field]any

		responseSchedule *schedule.Schedule
		expectRes        *sock.Response
	}{
		{
			name: "cron and enabled",
			request: &sock.Request{
				URI:      "/v1/schedules/e8e3902a-6f21-11f0-b1a1-2f74a58c4c0e",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				// pinned client shape: requesthandler.ScheduleV1ScheduleUpdate
				// marshals the field map directly
				Data: []byte(`{"cron":"0 4 * * *","enabled":true}`),
			},

			fields: map[schedule.Field]any{
				schedule.FieldCron:    "0 4 * * *",
				schedule.FieldEnabled: true,
			},

			responseSchedule: &schedule.Schedule{
				Identity: commonidentity.Identity{
					ID: scheduleID,
				},
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"e8e3902a-6f21-11f0-b1a1-2f74a58c4c0e","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","type":"","cron":"","target_queue":"","target_uri":"","target_method":"","target_data_type":"","target_data":null,"timeout_ms":0,"retry_max":0,"enabled":false,"tm_next_run":null,"tm_last_run":null,"tm_create":null,"tm_update":null,"tm_delete":null}`),
			},
		},
		{
			name: "all updatable fields",
			request: &sock.Request{
				URI:      "/v1/schedules/e8e3902a-6f21-11f0-b1a1-2f74a58c4c0e",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"name":"new-name","detail":"new detail","cron":"30 2 * * *","target_queue":"bin-manager.number-manager.request","target_uri":"/v1/numbers/renew","target_method":"POST","target_data_type":"application/json","target_data":{"days":30},"timeout_ms":60000,"retry_max":2,"enabled":false}`),
			},

			fields: map[schedule.Field]any{
				schedule.FieldName:           "new-name",
				schedule.FieldDetail:         "new detail",
				schedule.FieldCron:           "30 2 * * *",
				schedule.FieldTargetQueue:    "bin-manager.number-manager.request",
				schedule.FieldTargetURI:      "/v1/numbers/renew",
				schedule.FieldTargetMethod:   "POST",
				schedule.FieldTargetDataType: "application/json",
				schedule.FieldTargetData:     json.RawMessage(`{"days":30}`),
				schedule.FieldTimeoutMS:      60000,
				schedule.FieldRetryMax:       2,
				schedule.FieldEnabled:        false,
			},

			responseSchedule: &schedule.Schedule{
				Identity: commonidentity.Identity{
					ID: scheduleID,
				},
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"e8e3902a-6f21-11f0-b1a1-2f74a58c4c0e","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","type":"","cron":"","target_queue":"","target_uri":"","target_method":"","target_data_type":"","target_data":null,"timeout_ms":0,"retry_max":0,"enabled":false,"tm_next_run":null,"tm_last_run":null,"tm_create":null,"tm_update":null,"tm_delete":null}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc, h, mockSchedule, _ := setupListenHandler(t)
			defer mc.Finish()

			mockSchedule.EXPECT().Update(gomock.Any(), scheduleID, tt.fields).Return(tt.responseSchedule, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1SchedulesIDPut_badJSON(t *testing.T) {
	mc, h, _, _ := setupListenHandler(t)
	defer mc.Finish()

	res, err := h.processRequest(&sock.Request{
		URI:      "/v1/schedules/e8e3902a-6f21-11f0-b1a1-2f74a58c4c0e",
		Method:   sock.RequestMethodPut,
		DataType: "application/json",
		Data:     []byte(`not json`),
	})
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("Wrong match. expect: 400, got: %d", res.StatusCode)
	}
}

func Test_processV1SchedulesIDDelete(t *testing.T) {
	scheduleID := uuid.FromStringOrNil("1cfa9a4a-6f22-11f0-b64f-1bd3a3e13d3f")

	mc, h, mockSchedule, _ := setupListenHandler(t)
	defer mc.Finish()

	responseSchedule := &schedule.Schedule{
		Identity: commonidentity.Identity{
			ID: scheduleID,
		},
	}

	mockSchedule.EXPECT().Delete(gomock.Any(), scheduleID).Return(responseSchedule, nil)

	res, err := h.processRequest(&sock.Request{
		URI:    "/v1/schedules/1cfa9a4a-6f22-11f0-b64f-1bd3a3e13d3f",
		Method: sock.RequestMethodDelete,
	})
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	expectRes := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       []byte(`{"id":"1cfa9a4a-6f22-11f0-b64f-1bd3a3e13d3f","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","type":"","cron":"","target_queue":"","target_uri":"","target_method":"","target_data_type":"","target_data":null,"timeout_ms":0,"retry_max":0,"enabled":false,"tm_next_run":null,"tm_last_run":null,"tm_create":null,"tm_update":null,"tm_delete":null}`),
	}
	if !reflect.DeepEqual(res, expectRes) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expectRes, res)
	}
}

func Test_processV1SchedulesIDExecutePost(t *testing.T) {
	scheduleID := uuid.FromStringOrNil("4b1b409c-6f22-11f0-8f4e-eb0a7fbfbd52")
	executionID := uuid.FromStringOrNil("4b96a97c-6f22-11f0-b7a1-6f0d9f2c1c1a")

	mc, h, _, mockDispatch := setupListenHandler(t)
	defer mc.Finish()

	responseExecution := &execution.Execution{
		Identity: commonidentity.Identity{
			ID: executionID,
		},
		ScheduleID:  scheduleID,
		TriggerType: execution.TriggerTypeManual,
		Status:      execution.StatusRunning,
	}

	mockDispatch.EXPECT().ExecuteManual(gomock.Any(), scheduleID).Return(responseExecution, nil)

	res, err := h.processRequest(&sock.Request{
		URI:    "/v1/schedules/4b1b409c-6f22-11f0-8f4e-eb0a7fbfbd52/execute",
		Method: sock.RequestMethodPost,
	})
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	expectRes := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       []byte(`{"id":"4b96a97c-6f22-11f0-b7a1-6f0d9f2c1c1a","customer_id":"00000000-0000-0000-0000-000000000000","schedule_id":"4b1b409c-6f22-11f0-8f4e-eb0a7fbfbd52","trigger_type":"manual","status":"running","status_code":0,"error":"","result":"","attempt_count":0,"duration_ms":0,"tm_scheduled":null,"tm_deadline":null,"tm_start":null,"tm_end":null,"tm_create":null,"tm_update":null,"tm_delete":null}`),
	}
	if !reflect.DeepEqual(res, expectRes) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expectRes, res)
	}
}

func Test_processV1SchedulesIDExecutePost_error(t *testing.T) {
	scheduleID := uuid.FromStringOrNil("79a1e7f4-6f22-11f0-9e93-83fcb0a4c31a")

	tests := []struct {
		name string

		responseErr error

		expectStatusCode int
	}{
		{
			name: "409 while a run is in flight",

			responseErr: cerrors.FailedPrecondition(commonoutline.ServiceNameScheduleManager, "EXECUTION_IN_PROGRESS", "The schedule already has a running execution."),

			expectStatusCode: http.StatusConflict,
		},
		{
			name: "404 unknown schedule",

			responseErr: cerrors.NotFound(commonoutline.ServiceNameScheduleManager, "SCHEDULE_NOT_FOUND", "The schedule was not found."),

			expectStatusCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc, h, _, mockDispatch := setupListenHandler(t)
			defer mc.Finish()

			mockDispatch.EXPECT().ExecuteManual(gomock.Any(), scheduleID).Return(nil, tt.responseErr)

			res, err := h.processRequest(&sock.Request{
				URI:    "/v1/schedules/79a1e7f4-6f22-11f0-9e93-83fcb0a4c31a/execute",
				Method: sock.RequestMethodPost,
			})
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res.StatusCode != tt.expectStatusCode {
				t.Errorf("Wrong match. expect: %d, got: %d", tt.expectStatusCode, res.StatusCode)
			}
			if res.DataType != cerrors.DataTypeVoipbinError {
				t.Errorf("Wrong match. expect: %s, got: %s", cerrors.DataTypeVoipbinError, res.DataType)
			}
		})
	}
}
