package listenhandler

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-pipecat-manager/models/pipecatcall"
	"monorepo/bin-pipecat-manager/pkg/pipecatcallhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

// unserializableLLMMessages returns llm messages that json can not encode, so the handler can
// not build the response body of the pipecatcall it got.
func unserializableLLMMessages() []map[string]any {
	return []map[string]any{{"content": make(chan int)}}
}

func Test_pipecatcallsResponse_marshalFailure(t *testing.T) {
	id := uuid.FromStringOrNil("5b5bb704-b48c-11f0-819e-2ff9e60d5c3c")
	broken := &pipecatcall.Pipecatcall{
		Identity:    commonidentity.Identity{ID: id},
		LLMMessages: unserializableLLMMessages(),
	}

	tests := []struct {
		name string

		request *sock.Request
		expect  func(m *pipecatcallhandler.MockPipecatcallHandler)
	}{
		{
			name: "post",
			request: &sock.Request{
				URI:    "/v1/pipecatcalls",
				Method: sock.RequestMethodPost,
				Data:   []byte(`{"id":"5b5bb704-b48c-11f0-819e-2ff9e60d5c3c"}`),
			},
			expect: func(m *pipecatcallhandler.MockPipecatcallHandler) {
				m.EXPECT().Start(gomock.Any(), id, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(broken, nil)
			},
		},
		{
			name: "get",
			request: &sock.Request{
				URI:    "/v1/pipecatcalls/5b5bb704-b48c-11f0-819e-2ff9e60d5c3c",
				Method: sock.RequestMethodGet,
			},
			expect: func(m *pipecatcallhandler.MockPipecatcallHandler) {
				m.EXPECT().Get(gomock.Any(), id).Return(broken, nil)
			},
		},
		{
			name: "stop",
			request: &sock.Request{
				URI:    "/v1/pipecatcalls/5b5bb704-b48c-11f0-819e-2ff9e60d5c3c/stop",
				Method: sock.RequestMethodPost,
			},
			expect: func(m *pipecatcallhandler.MockPipecatcallHandler) {
				m.EXPECT().Terminate(gomock.Any(), id).Return(broken, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockPipecatcall := pipecatcallhandler.NewMockPipecatcallHandler(mc)
			h := &listenHandler{pipecatcallHandler: mockPipecatcall}
			tt.expect(mockPipecatcall)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Fatalf("Wrong error. expect: nil, got: %v", err)
			}
			if res.StatusCode != http.StatusInternalServerError {
				t.Errorf("Wrong status. expect: %d, got: %d", http.StatusInternalServerError, res.StatusCode)
			}
			if len(res.Data) != 0 {
				t.Errorf("Wrong data. expect: no body, got: %s", res.Data)
			}
		})
	}
}

func Test_errorResponse_unserializableDetails(t *testing.T) {
	typed := cerrors.NotFound(commonoutline.ServiceNamePipecatManager, "PIPECATCALL_NOT_FOUND", "not found")
	typed.Details = unserializableLLMMessages()

	res := errorResponse(typed)
	if res.StatusCode != http.StatusInternalServerError {
		t.Errorf("Wrong status. expect: %d, got: %d", http.StatusInternalServerError, res.StatusCode)
	}
	if res.DataType == cerrors.DataTypeVoipbinError {
		t.Errorf("Wrong data type. a response that could not be built must not claim the typed error body")
	}
}

func Test_Run_consumeFailure(t *testing.T) {
	const (
		queue         = "test.queue"
		queueVolatile = "test.queue.volatile"
	)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	h := &listenHandler{sockHandler: mockSock}

	var wg sync.WaitGroup
	wg.Add(2)

	// a failing consume of the first queue is only logged: the run still succeeds and the
	// volatile queue is consumed.
	mockSock.EXPECT().QueueCreate(queue, "normal").Return(nil)
	mockSock.EXPECT().QueueCreate(queueVolatile, "volatile").Return(nil)
	mockSock.EXPECT().ConsumeRPC(gomock.Any(), queue, constConsumerName, false, false, false, 10, gomock.AssignableToTypeOf(sock.CbMsgRPC(nil))).
		DoAndReturn(func(context.Context, string, string, bool, bool, bool, int, sock.CbMsgRPC) error {
			defer wg.Done()
			return errors.New("consume failed")
		})
	mockSock.EXPECT().ConsumeRPC(gomock.Any(), queueVolatile, constConsumerName, false, false, false, 10, gomock.AssignableToTypeOf(sock.CbMsgRPC(nil))).
		DoAndReturn(func(context.Context, string, string, bool, bool, bool, int, sock.CbMsgRPC) error {
			defer wg.Done()
			return nil
		})

	if err := h.Run(queue, queueVolatile, "delay"); err != nil {
		t.Fatalf("Wrong error. expect: nil, got: %v", err)
	}
	waitDone(t, &wg)
}
