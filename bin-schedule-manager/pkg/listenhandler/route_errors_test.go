package listenhandler

import (
	"errors"
	"net/http"
	"testing"

	"monorepo/bin-common-handler/models/sock"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-schedule-manager/pkg/dbhandler"
	"monorepo/bin-schedule-manager/pkg/dispatchhandler"
	"monorepo/bin-schedule-manager/pkg/schedulehandler"
)

// Test_processRequest_routeErrors verifies how each route maps a failure of
// the downstream handler or a malformed request to a status code.
func Test_processRequest_routeErrors(t *testing.T) {
	const idPath = "/v1/schedules/d0e14e34-6f21-11f0-9b93-9316cd2b6d9a"
	errBoom := errors.New("boom")

	tests := []struct {
		name    string
		request *sock.Request
		setup   func(s *schedulehandler.MockScheduleHandler, d *dispatchhandler.MockDispatchHandler)

		expectStatus int
	}{
		{
			name:    "schedules post: handler failure is 500",
			request: &sock.Request{URI: "/v1/schedules", Method: sock.RequestMethodPost, Data: []byte(`{"name":"a"}`)},
			setup: func(s *schedulehandler.MockScheduleHandler, _ *dispatchhandler.MockDispatchHandler) {
				s.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errBoom)
			},
			expectStatus: http.StatusInternalServerError,
		},
		{
			name:         "schedules get: unparsable filter body is 400 and the handler is not called",
			request:      &sock.Request{URI: "/v1/schedules?page_size=10", Method: sock.RequestMethodGet, Data: []byte(`not-json`)},
			setup:        func(_ *schedulehandler.MockScheduleHandler, _ *dispatchhandler.MockDispatchHandler) {},
			expectStatus: http.StatusBadRequest,
		},
		{
			name:         "schedules get: malformed uuid filter is 400 and the handler is not called",
			request:      &sock.Request{URI: "/v1/schedules?page_size=10", Method: sock.RequestMethodGet, Data: []byte(`{"customer_id":"not-a-uuid"}`)},
			setup:        func(_ *schedulehandler.MockScheduleHandler, _ *dispatchhandler.MockDispatchHandler) {},
			expectStatus: http.StatusBadRequest,
		},
		{
			name:    "schedules get: handler failure is 500",
			request: &sock.Request{URI: "/v1/schedules?page_size=10", Method: sock.RequestMethodGet, Data: []byte(`{}`)},
			setup: func(s *schedulehandler.MockScheduleHandler, _ *dispatchhandler.MockDispatchHandler) {
				s.EXPECT().Gets(gomock.Any(), uint64(10), gomock.Any(), gomock.Any()).Return(nil, errBoom)
			},
			expectStatus: http.StatusInternalServerError,
		},
		{
			name:    "schedules id get: handler failure is 500",
			request: &sock.Request{URI: idPath, Method: sock.RequestMethodGet},
			setup: func(s *schedulehandler.MockScheduleHandler, _ *dispatchhandler.MockDispatchHandler) {
				s.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, errBoom)
			},
			expectStatus: http.StatusInternalServerError,
		},
		{
			name:    "schedules id put: record not found is 404",
			request: &sock.Request{URI: idPath, Method: sock.RequestMethodPut, Data: []byte(`{"detail":"d"}`)},
			setup: func(s *schedulehandler.MockScheduleHandler, _ *dispatchhandler.MockDispatchHandler) {
				s.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, dbhandler.ErrNotFound)
			},
			expectStatus: http.StatusNotFound,
		},
		{
			name:    "schedules id put: handler failure is 500",
			request: &sock.Request{URI: idPath, Method: sock.RequestMethodPut, Data: []byte(`{"detail":"d"}`)},
			setup: func(s *schedulehandler.MockScheduleHandler, _ *dispatchhandler.MockDispatchHandler) {
				s.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errBoom)
			},
			expectStatus: http.StatusInternalServerError,
		},
		{
			name:    "schedules id delete: record not found is 404",
			request: &sock.Request{URI: idPath, Method: sock.RequestMethodDelete},
			setup: func(s *schedulehandler.MockScheduleHandler, _ *dispatchhandler.MockDispatchHandler) {
				s.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil, dbhandler.ErrNotFound)
			},
			expectStatus: http.StatusNotFound,
		},
		{
			name:    "schedules id delete: handler failure is 500",
			request: &sock.Request{URI: idPath, Method: sock.RequestMethodDelete},
			setup: func(s *schedulehandler.MockScheduleHandler, _ *dispatchhandler.MockDispatchHandler) {
				s.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil, errBoom)
			},
			expectStatus: http.StatusInternalServerError,
		},
		{
			name:         "executions get: unparsable filter body is 400 and the handler is not called",
			request:      &sock.Request{URI: "/v1/executions?page_size=10", Method: sock.RequestMethodGet, Data: []byte(`not-json`)},
			setup:        func(_ *schedulehandler.MockScheduleHandler, _ *dispatchhandler.MockDispatchHandler) {},
			expectStatus: http.StatusBadRequest,
		},
		{
			name:         "executions get: malformed uuid filter is 400 and the handler is not called",
			request:      &sock.Request{URI: "/v1/executions?page_size=10", Method: sock.RequestMethodGet, Data: []byte(`{"schedule_id":"not-a-uuid"}`)},
			setup:        func(_ *schedulehandler.MockScheduleHandler, _ *dispatchhandler.MockDispatchHandler) {},
			expectStatus: http.StatusBadRequest,
		},
		{
			name:    "executions get: handler failure is 500",
			request: &sock.Request{URI: "/v1/executions?page_size=10", Method: sock.RequestMethodGet, Data: []byte(`{}`)},
			setup: func(s *schedulehandler.MockScheduleHandler, _ *dispatchhandler.MockDispatchHandler) {
				s.EXPECT().ExecutionGets(gomock.Any(), uint64(10), gomock.Any(), gomock.Any()).Return(nil, errBoom)
			},
			expectStatus: http.StatusInternalServerError,
		},
		{
			name:    "executions prune: handler failure is 500 with the configured retention",
			request: &sock.Request{URI: "/v1/executions/prune", Method: sock.RequestMethodPost},
			setup: func(s *schedulehandler.MockScheduleHandler, _ *dispatchhandler.MockDispatchHandler) {
				s.EXPECT().ExecutionPrune(gomock.Any(), 90).Return(int64(0), errBoom)
			},
			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc, h, mockSchedule, mockDispatch := setupListenHandler(t)
			defer mc.Finish()

			tt.setup(mockSchedule, mockDispatch)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if res.StatusCode != tt.expectStatus {
				t.Errorf("status = %d, expect: %d", res.StatusCode, tt.expectStatus)
			}
		})
	}
}
