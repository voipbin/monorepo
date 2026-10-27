package listenhandler

import (
	"context"
	"net/http"
	"testing"
	"time"

	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-schedule-manager/models/execution"
)

// Test_executions_response_marshal_failure verifies that an execution that
// cannot be encoded gives a bare 500, while the same request with an
// encodable execution gives 200 (positive control).
func Test_executions_response_marshal_failure(t *testing.T) {
	const scheduleID = "d0e14e34-6f21-11f0-9b93-9316cd2b6d9a"

	okTime := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	// a year above 9999 cannot be encoded as RFC 3339
	badTime := time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		request *sock.Request
		setup   func(f *handlerFixture, tm *time.Time)
		call    func(h *listenHandler, ctx context.Context, m *sock.Request) (*sock.Response, error)
	}{
		{
			name:    "list",
			request: &sock.Request{URI: "/v1/executions?page_size=10", Method: sock.RequestMethodGet, Data: []byte(`{}`)},
			setup: func(f *handlerFixture, tm *time.Time) {
				f.mockSchedule.EXPECT().ExecutionGets(gomock.Any(), uint64(10), gomock.Any(), gomock.Any()).Return([]*execution.Execution{{TMCreate: tm}}, nil)
			},
			call: (*listenHandler).processV1ExecutionsGet,
		},
		{
			name:    "manual execute",
			request: &sock.Request{URI: "/v1/schedules/" + scheduleID + "/execute", Method: sock.RequestMethodPost},
			setup: func(f *handlerFixture, tm *time.Time) {
				f.mockDispatch.EXPECT().ExecuteManual(gomock.Any(), uuid.FromStringOrNil(scheduleID)).Return(&execution.Execution{TMCreate: tm}, nil)
			},
			call: (*listenHandler).processV1SchedulesIDExecutePost,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, c := range []struct {
				label      string
				tm         time.Time
				expectCode int
				expectData bool
			}{
				{"encodable", okTime, http.StatusOK, true},
				{"unencodable", badTime, http.StatusInternalServerError, false},
			} {
				mc := gomock.NewController(t)
				f := newHandlerFixture(mc)
				tm := c.tm
				tt.setup(f, &tm)

				res, err := tt.call(f.h, context.Background(), tt.request)
				if err != nil {
					t.Fatalf("%s: Wrong match. expect: nil error, got: %v", c.label, err)
				}
				if res.StatusCode != c.expectCode || (len(res.Data) != 0) != c.expectData {
					t.Errorf("%s: Wrong match. expect: %d (data: %v), got: %d with %d bytes", c.label, c.expectCode, c.expectData, res.StatusCode, len(res.Data))
				}
				mc.Finish()
			}
		})
	}
}
