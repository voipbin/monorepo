package listenhandler

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"net/http"
	"testing"
	"time"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-schedule-manager/models/schedule"
	"monorepo/bin-schedule-manager/pkg/backuphandler"
	"monorepo/bin-schedule-manager/pkg/dispatchhandler"
	"monorepo/bin-schedule-manager/pkg/schedulehandler"
)

const testWait = 5 * time.Second

type handlerFixture struct {
	h            *listenHandler
	mockSock     *sockhandler.MockSockHandler
	mockSchedule *schedulehandler.MockScheduleHandler
	mockDispatch *dispatchhandler.MockDispatchHandler
}

func newHandlerFixture(mc *gomock.Controller) *handlerFixture {
	mockSock := sockhandler.NewMockSockHandler(mc)
	mockSchedule := schedulehandler.NewMockScheduleHandler(mc)
	mockDispatch := dispatchhandler.NewMockDispatchHandler(mc)
	mockBackup := backuphandler.NewMockBackupHandler(mc)

	return &handlerFixture{
		h: &listenHandler{
			sockHandler:     mockSock,
			scheduleHandler: mockSchedule,
			dispatchHandler: mockDispatch,
			backupHandler:   mockBackup,
		},
		mockSock:     mockSock,
		mockSchedule: mockSchedule,
		mockDispatch: mockDispatch,
	}
}

func Test_Run(t *testing.T) {
	queue := "test.queue"

	tests := []struct {
		name string

		queueErr   error
		consumeErr error

		expectErr bool
	}{
		{
			name: "queue declared and consumer started",
		},
		{
			name: "consumer failure is only logged",

			consumeErr: stderrors.New("synthetic consume failure"),
		},
		{
			name: "queue declare fails",

			queueErr: stderrors.New("synthetic queue failure"),

			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			f := newHandlerFixture(mc)

			f.mockSock.EXPECT().QueueCreate(queue, "normal").Return(tt.queueErr)

			consumed := make(chan struct{})
			if tt.queueErr == nil {
				f.mockSock.EXPECT().ConsumeRPC(gomock.Any(), queue, gomock.Any(), false, false, false, 10, gomock.Any()).
					DoAndReturn(func(_ context.Context, _ string, _ string, _ bool, _ bool, _ bool, _ int, _ sock.CbMsgRPC) error {
						defer close(consumed)
						return tt.consumeErr
					})
			}

			err := f.h.Run(queue, "test.delay")
			if tt.expectErr {
				if !stderrors.Is(err, tt.queueErr) {
					t.Errorf("Wrong match. expect: the queue error reachable, got: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			// the consumer runs in a goroutine; wait for the call before the controller finishes
			select {
			case <-consumed:
			case <-time.After(testWait):
				t.Fatalf("Wrong match. expect: the consumer call, got: timeout")
			}
		})
	}
}

// brokenSchedule returns a schedule whose raw target data is not valid JSON, so
// marshaling the response body fails.
func brokenSchedule() *schedule.Schedule {
	return &schedule.Schedule{
		Identity:   commonidentity.Identity{ID: uuid.FromStringOrNil("d0e14e34-6f21-11f0-9b93-9316cd2b6d9a")},
		Name:       "broken-target-data",
		TargetData: json.RawMessage(`{bad`),
	}
}

func Test_schedules_response_marshal_failure(t *testing.T) {
	const idPath = "/v1/schedules/d0e14e34-6f21-11f0-9b93-9316cd2b6d9a"

	tests := []struct {
		name    string
		request *sock.Request
		setup   func(f *handlerFixture)
		call    func(h *listenHandler, ctx context.Context, m *sock.Request) (*sock.Response, error)
	}{
		{
			name:    "create",
			request: &sock.Request{URI: "/v1/schedules", Method: sock.RequestMethodPost, Data: []byte(`{"name":"a"}`)},
			setup: func(f *handlerFixture) {
				f.mockSchedule.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(brokenSchedule(), nil)
			},
			call: (*listenHandler).processV1SchedulesPost,
		},
		{
			name:    "list",
			request: &sock.Request{URI: "/v1/schedules?page_size=10", Method: sock.RequestMethodGet, Data: []byte(`{}`)},
			setup: func(f *handlerFixture) {
				f.mockSchedule.EXPECT().Gets(gomock.Any(), uint64(10), gomock.Any(), gomock.Any()).Return([]*schedule.Schedule{brokenSchedule()}, nil)
			},
			call: (*listenHandler).processV1SchedulesGet,
		},
		{
			name:    "get",
			request: &sock.Request{URI: idPath, Method: sock.RequestMethodGet},
			setup: func(f *handlerFixture) {
				f.mockSchedule.EXPECT().Get(gomock.Any(), gomock.Any()).Return(brokenSchedule(), nil)
			},
			call: (*listenHandler).processV1SchedulesIDGet,
		},
		{
			name:    "update",
			request: &sock.Request{URI: idPath, Method: sock.RequestMethodPut, Data: []byte(`{}`)},
			setup: func(f *handlerFixture) {
				f.mockSchedule.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).Return(brokenSchedule(), nil)
			},
			call: (*listenHandler).processV1SchedulesIDPut,
		},
		{
			name:    "delete",
			request: &sock.Request{URI: idPath, Method: sock.RequestMethodDelete},
			setup: func(f *handlerFixture) {
				f.mockSchedule.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(brokenSchedule(), nil)
			},
			call: (*listenHandler).processV1SchedulesIDDelete,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			f := newHandlerFixture(mc)
			tt.setup(f)

			res, err := tt.call(f.h, context.Background(), tt.request)
			if err != nil {
				t.Fatalf("Wrong match. expect: nil error, got: %v", err)
			}
			if res.StatusCode != http.StatusInternalServerError || len(res.Data) != 0 {
				t.Errorf("Wrong match. expect: bare 500, got: %d with %d bytes", res.StatusCode, len(res.Data))
			}
		})
	}
}

// Test_id_handlers_short_uri calls the handlers directly: the router regexes
// never let a short uri through, so only a direct call reaches the length guard.
func Test_id_handlers_short_uri(t *testing.T) {
	tests := []struct {
		name string
		uri  string
		call func(h *listenHandler, ctx context.Context, m *sock.Request) (*sock.Response, error)
	}{
		{"get", "/v1/schedules", (*listenHandler).processV1SchedulesIDGet},
		{"put", "/v1/schedules", (*listenHandler).processV1SchedulesIDPut},
		{"delete", "/v1/schedules", (*listenHandler).processV1SchedulesIDDelete},
		{"execute", "/v1/schedules/d0e14e34-6f21-11f0-9b93-9316cd2b6d9a", (*listenHandler).processV1SchedulesIDExecutePost},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// the mocks have no expectations: any downstream call fails the test
			f := newHandlerFixture(mc)

			res, err := tt.call(f.h, context.Background(), &sock.Request{URI: tt.uri, Data: []byte(`{}`)})
			if err != nil {
				t.Fatalf("Wrong match. expect: nil error, got: %v", err)
			}
			if res.StatusCode != http.StatusBadRequest {
				t.Errorf("Wrong match. expect: 400, got: %d", res.StatusCode)
			}
		})
	}
}

// Test_processRequest_unparsable_uri uses a fragment with a bad escape: the
// regex route still matches, but url.Parse fails and the router answers 400.
func Test_processRequest_unparsable_uri(t *testing.T) {
	tests := []struct {
		name string
		uri  string
	}{
		{"schedules", "/v1/schedules?a=1#%zz"},
		{"executions", "/v1/executions?a=1#%zz"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			f := newHandlerFixture(mc)

			res, err := f.h.processRequest(&sock.Request{URI: tt.uri, Method: sock.RequestMethodGet, Data: []byte(`{}`)})
			if err != nil {
				t.Fatalf("Wrong match. expect: nil error, got: %v", err)
			}
			if res.StatusCode != http.StatusBadRequest {
				t.Errorf("Wrong match. expect: 400, got: %d", res.StatusCode)
			}
		})
	}
}

func Test_errorResponse_typed_nil_conversion_failure(t *testing.T) {
	var typedNil *cerrors.VoipbinError

	res := errorResponse(typedNil)
	if res.StatusCode != http.StatusInternalServerError || len(res.Data) != 0 {
		t.Errorf("Wrong match. expect: bare 500, got: %d with %d bytes", res.StatusCode, len(res.Data))
	}
}
