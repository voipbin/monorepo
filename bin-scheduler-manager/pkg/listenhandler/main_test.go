package listenhandler

import (
	"reflect"
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-scheduler-manager/pkg/backuphandler"
	"monorepo/bin-scheduler-manager/pkg/dispatchhandler"
	"monorepo/bin-scheduler-manager/pkg/schedulehandler"
)

func Test_NewListenHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockSchedule := schedulehandler.NewMockScheduleHandler(mc)
	mockDispatch := dispatchhandler.NewMockDispatchHandler(mc)
	mockBackup := backuphandler.NewMockBackupHandler(mc)

	h := NewListenHandler(mockSock, mockSchedule, mockDispatch, mockBackup, 90)
	if h == nil {
		t.Errorf("Expected handler, got nil")
	}
}

func Test_processRequest_notFound(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request
	}{
		{
			name: "unknown uri",
			request: &sock.Request{
				URI:    "/v1/unknown",
				Method: sock.RequestMethodGet,
			},
		},
		{
			name: "wrong method on backups",
			request: &sock.Request{
				URI:    "/v1/backups",
				Method: sock.RequestMethodGet,
			},
		},
		{
			name: "wrong method on executions prune",
			request: &sock.Request{
				URI:    "/v1/executions/prune",
				Method: sock.RequestMethodGet,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockSchedule := schedulehandler.NewMockScheduleHandler(mc)
			mockDispatch := dispatchhandler.NewMockDispatchHandler(mc)
			mockBackup := backuphandler.NewMockBackupHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,

				scheduleHandler: mockSchedule,
				dispatchHandler: mockDispatch,
				backupHandler:   mockBackup,
			}

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			expectRes := simpleResponse(404)
			if !reflect.DeepEqual(res, expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expectRes, res)
			}
		})
	}
}
