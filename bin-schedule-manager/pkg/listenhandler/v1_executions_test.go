package listenhandler

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-schedule-manager/models/execution"
)

func Test_processV1ExecutionsGet(t *testing.T) {
	scheduleID := uuid.FromStringOrNil("9c17e21a-6f22-11f0-9d59-b3f1cf2b7a1e")

	tests := []struct {
		name    string
		request *sock.Request

		pageSize  uint64
		pageToken string
		filters   map[execution.Field]any

		responseExecutions []*execution.Execution
		expectRes          *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/executions?page_token=2026-08-01T00:00:00.000000Z&page_size=10",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     []byte(`{"schedule_id":"9c17e21a-6f22-11f0-9d59-b3f1cf2b7a1e","status":"failed"}`),
			},

			pageSize:  10,
			pageToken: "2026-08-01T00:00:00.000000Z",
			filters: map[execution.Field]any{
				execution.FieldScheduleID: scheduleID,
				execution.FieldStatus:     "failed",
			},

			responseExecutions: []*execution.Execution{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("9caf7d16-6f22-11f0-92f6-1f0d1e05f3ba"),
					},
					ScheduleID:  scheduleID,
					TriggerType: execution.TriggerTypeCron,
					Status:      execution.StatusFailed,
				},
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"9caf7d16-6f22-11f0-92f6-1f0d1e05f3ba","customer_id":"00000000-0000-0000-0000-000000000000","schedule_id":"9c17e21a-6f22-11f0-9d59-b3f1cf2b7a1e","trigger_type":"cron","status":"failed","status_code":0,"error":"","result":"","attempt_count":0,"duration_ms":0,"tm_scheduled":null,"tm_deadline":null,"tm_start":null,"tm_end":null,"tm_create":null,"tm_update":null,"tm_delete":null}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc, h, mockSchedule, _ := setupListenHandler(t)
			defer mc.Finish()

			mockSchedule.EXPECT().ExecutionGets(gomock.Any(), tt.pageSize, tt.pageToken, tt.filters).Return(tt.responseExecutions, nil)

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

func Test_processV1ExecutionsPrunePost(t *testing.T) {
	mc, h, mockSchedule, _ := setupListenHandler(t)
	defer mc.Finish()

	// executionRetentionDays is 90 in setupListenHandler
	mockSchedule.EXPECT().ExecutionPrune(gomock.Any(), 90).Return(int64(1250), nil)

	res, err := h.processRequest(&sock.Request{
		URI:    "/v1/executions/prune",
		Method: sock.RequestMethodPost,
	})
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	expectRes := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       []byte(`{"removed":1250}`),
	}
	if !reflect.DeepEqual(res, expectRes) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expectRes, res)
	}
}

func Test_processV1ExecutionsPrunePost_error(t *testing.T) {
	mc, h, mockSchedule, _ := setupListenHandler(t)
	defer mc.Finish()

	mockSchedule.EXPECT().ExecutionPrune(gomock.Any(), 90).Return(int64(0), fmt.Errorf("db error"))

	res, err := h.processRequest(&sock.Request{
		URI:    "/v1/executions/prune",
		Method: sock.RequestMethodPost,
	})
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	if res.StatusCode != http.StatusInternalServerError {
		t.Errorf("Wrong match. expect: 500, got: %d", res.StatusCode)
	}
}
