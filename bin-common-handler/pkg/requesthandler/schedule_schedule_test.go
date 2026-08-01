package requesthandler

import (
	"context"
	"reflect"
	"testing"

	smexecution "monorepo/bin-schedule-manager/models/execution"
	smschedule "monorepo/bin-schedule-manager/models/schedule"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

func Test_ScheduleV1ScheduleCreate(t *testing.T) {

	tests := []struct {
		name string

		customerID     uuid.UUID
		scheduleName   string
		detail         string
		cronExpr       string
		targetQueue    string
		targetURI      string
		targetMethod   string
		targetDataType string
		targetData     []byte
		timeoutMS      int
		retryMax       int
		enabled        bool

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response

		expectRes *smschedule.Schedule
	}{
		{
			name: "normal",

			customerID:     uuid.FromStringOrNil("15edb1c6-6f15-11f0-b0f2-cb2c53b2f552"),
			scheduleName:   "number-renew",
			detail:         "daily number renewal",
			cronExpr:       "0 2 * * *",
			targetQueue:    "bin-manager.number-manager.request",
			targetURI:      "/v1/numbers/renew",
			targetMethod:   "POST",
			targetDataType: "application/json",
			targetData:     []byte(`{"days":3}`),
			timeoutMS:      300000,
			retryMax:       0,
			enabled:        true,

			expectTarget: "bin-manager.schedule-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/schedules",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"customer_id":"15edb1c6-6f15-11f0-b0f2-cb2c53b2f552","name":"number-renew","detail":"daily number renewal","cron":"0 2 * * *","target_queue":"bin-manager.number-manager.request","target_uri":"/v1/numbers/renew","target_method":"POST","target_data_type":"application/json","target_data":{"days":3},"timeout_ms":300000,"retry_max":0,"enabled":true}`),
			},
			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"161f83e0-6f15-11f0-8f8e-2f39d3b2e552"}`),
			},
			expectRes: &smschedule.Schedule{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("161f83e0-6f15-11f0-8f8e-2f39d3b2e552"),
				},
			},
		},
		{
			name: "empty target data",

			customerID:   uuid.FromStringOrNil("164fce88-6f15-11f0-b552-3f8cf3b2e552"),
			scheduleName: "database-backup",
			detail:       "nightly database backup",
			cronExpr:     "0 3 * * *",
			targetQueue:  "bin-manager.schedule-manager.request",
			targetURI:    "/v1/backups",
			targetMethod: "POST",
			timeoutMS:    1800000,
			retryMax:     1,
			enabled:      false,

			expectTarget: "bin-manager.schedule-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/schedules",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"customer_id":"164fce88-6f15-11f0-b552-3f8cf3b2e552","name":"database-backup","detail":"nightly database backup","cron":"0 3 * * *","target_queue":"bin-manager.schedule-manager.request","target_uri":"/v1/backups","target_method":"POST","target_data_type":"","target_data":null,"timeout_ms":1800000,"retry_max":1,"enabled":false}`),
			},
			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"1680a3f2-6f15-11f0-9b2e-af5cf3b2e552"}`),
			},
			expectRes: &smschedule.Schedule{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("1680a3f2-6f15-11f0-9b2e-af5cf3b2e552"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.ScheduleV1ScheduleCreate(ctx, tt.customerID, tt.scheduleName, tt.detail, tt.cronExpr, tt.targetQueue, tt.targetURI, tt.targetMethod, tt.targetDataType, tt.targetData, tt.timeoutMS, tt.retryMax, tt.enabled)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectRes, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectRes, *res)
			}
		})
	}
}

func Test_ScheduleV1ScheduleGet(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response

		expectRes *smschedule.Schedule
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("3e9b12a4-6f15-11f0-8f52-eb2cf3b2e552"),

			expectTarget: "bin-manager.schedule-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/schedules/3e9b12a4-6f15-11f0-8f52-eb2cf3b2e552",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeNone,
			},
			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"3e9b12a4-6f15-11f0-8f52-eb2cf3b2e552"}`),
			},
			expectRes: &smschedule.Schedule{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("3e9b12a4-6f15-11f0-8f52-eb2cf3b2e552"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.ScheduleV1ScheduleGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectRes, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectRes, *res)
			}
		})
	}
}

func Test_ScheduleV1ScheduleGets(t *testing.T) {

	tests := []struct {
		name string

		pageToken string
		pageSize  uint64
		filters   map[smschedule.Field]any

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response

		expectRes []smschedule.Schedule
	}{
		{
			name: "normal",

			pageToken: "2026-08-01 03:30:17.000000",
			pageSize:  10,
			filters: map[smschedule.Field]any{
				smschedule.FieldCustomerID: "5d31556e-6f15-11f0-b552-6f2cf3b2e552",
				smschedule.FieldDeleted:    false,
			},

			expectTarget: "bin-manager.schedule-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/schedules?page_token=2026-08-01+03%3A30%3A17.000000&page_size=10",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"customer_id":"5d31556e-6f15-11f0-b552-6f2cf3b2e552","deleted":false}`),
			},
			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"5d688a4c-6f15-11f0-9c3e-1b2cf3b2e552"}]`),
			},
			expectRes: []smschedule.Schedule{
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("5d688a4c-6f15-11f0-9c3e-1b2cf3b2e552"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.ScheduleV1ScheduleGets(ctx, tt.pageToken, tt.pageSize, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ScheduleV1ScheduleUpdate(t *testing.T) {

	tests := []struct {
		name string

		id     uuid.UUID
		fields map[smschedule.Field]any

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response

		expectRes *smschedule.Schedule
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("7f4a10b2-6f15-11f0-b7dc-cf2cf3b2e552"),
			fields: map[smschedule.Field]any{
				smschedule.FieldCron:    "0 4 * * *",
				smschedule.FieldEnabled: true,
			},

			expectTarget: "bin-manager.schedule-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/schedules/7f4a10b2-6f15-11f0-b7dc-cf2cf3b2e552",
				Method:   sock.RequestMethodPut,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"cron":"0 4 * * *","enabled":true}`),
			},
			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"7f4a10b2-6f15-11f0-b7dc-cf2cf3b2e552"}`),
			},
			expectRes: &smschedule.Schedule{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("7f4a10b2-6f15-11f0-b7dc-cf2cf3b2e552"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.ScheduleV1ScheduleUpdate(ctx, tt.id, tt.fields)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectRes, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectRes, *res)
			}
		})
	}
}

func Test_ScheduleV1ScheduleDelete(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response

		expectRes *smschedule.Schedule
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("9c2b34d6-6f15-11f0-8a52-af2cf3b2e552"),

			expectTarget: "bin-manager.schedule-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/schedules/9c2b34d6-6f15-11f0-8a52-af2cf3b2e552",
				Method:   sock.RequestMethodDelete,
				DataType: ContentTypeNone,
			},
			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"9c2b34d6-6f15-11f0-8a52-af2cf3b2e552"}`),
			},
			expectRes: &smschedule.Schedule{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("9c2b34d6-6f15-11f0-8a52-af2cf3b2e552"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.ScheduleV1ScheduleDelete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectRes, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectRes, *res)
			}
		})
	}
}

func Test_ScheduleV1ScheduleExecute(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response

		expectRes *smexecution.Execution
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("b8e410c2-6f15-11f0-9d15-1f2cf3b2e552"),

			expectTarget: "bin-manager.schedule-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/schedules/b8e410c2-6f15-11f0-9d15-1f2cf3b2e552/execute",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeNone,
			},
			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"b91a54c8-6f15-11f0-8d38-3f2cf3b2e552","schedule_id":"b8e410c2-6f15-11f0-9d15-1f2cf3b2e552","trigger_type":"manual","status":"running"}`),
			},
			expectRes: &smexecution.Execution{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("b91a54c8-6f15-11f0-8d38-3f2cf3b2e552"),
				},
				ScheduleID:  uuid.FromStringOrNil("b8e410c2-6f15-11f0-9d15-1f2cf3b2e552"),
				TriggerType: smexecution.TriggerTypeManual,
				Status:      smexecution.StatusRunning,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.ScheduleV1ScheduleExecute(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectRes, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectRes, *res)
			}
		})
	}
}

func Test_ScheduleV1ExecutionGets(t *testing.T) {

	tests := []struct {
		name string

		pageToken string
		pageSize  uint64
		filters   map[smexecution.Field]any

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response

		expectRes []smexecution.Execution
	}{
		{
			name: "normal",

			pageToken: "2026-08-01 03:30:17.000000",
			pageSize:  10,
			filters: map[smexecution.Field]any{
				smexecution.FieldScheduleID: "d61a54c8-6f15-11f0-b7cd-5f2cf3b2e552",
				smexecution.FieldStatus:     string(smexecution.StatusFailed),
			},

			expectTarget: "bin-manager.schedule-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/executions?page_token=2026-08-01+03%3A30%3A17.000000&page_size=10",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"schedule_id":"d61a54c8-6f15-11f0-b7cd-5f2cf3b2e552","status":"failed"}`),
			},
			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"d64c81a2-6f15-11f0-9138-7f2cf3b2e552"}]`),
			},
			expectRes: []smexecution.Execution{
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("d64c81a2-6f15-11f0-9138-7f2cf3b2e552"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.ScheduleV1ExecutionGets(ctx, tt.pageToken, tt.pageSize, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
