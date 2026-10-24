package listenhandler

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-pipecat-manager/models/pipecatcall"
	"monorepo/bin-pipecat-manager/pkg/dbhandler"
	"monorepo/bin-pipecat-manager/pkg/pipecatcallhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_processRequest_errorMapping(t *testing.T) {
	id := uuid.FromStringOrNil("5b5bb704-b48c-11f0-819e-2ff9e60d5c3c")

	typedConflict := cerrors.AlreadyExists(commonoutline.ServiceNamePipecatManager, "PIPECATCALL_EXISTS", "exists")

	tests := []struct {
		name string

		request *sock.Request
		// the error returned by the business handler (Get or Terminate).
		handlerErr error

		expectStatus int
	}{
		{
			name:         "get: typed error keeps its own status",
			request:      &sock.Request{URI: "/v1/pipecatcalls/" + id.String(), Method: sock.RequestMethodGet},
			handlerErr:   typedConflict,
			expectStatus: 409,
		},
		{
			name:         "get: legacy not found is 404",
			request:      &sock.Request{URI: "/v1/pipecatcalls/" + id.String(), Method: sock.RequestMethodGet},
			handlerErr:   dbhandler.ErrNotFound,
			expectStatus: 404,
		},
		{
			name:         "get: wrapped legacy not found is 404",
			request:      &sock.Request{URI: "/v1/pipecatcalls/" + id.String(), Method: sock.RequestMethodGet},
			handlerErr:   fmt.Errorf("could not get: %w", dbhandler.ErrNotFound),
			expectStatus: 404,
		},
		{
			name:         "get: unclassified error is 400",
			request:      &sock.Request{URI: "/v1/pipecatcalls/" + id.String(), Method: sock.RequestMethodGet},
			handlerErr:   errors.New("boom"),
			expectStatus: 400,
		},
		{
			name:         "get: typed error wrapping legacy not found keeps the typed status",
			request:      &sock.Request{URI: "/v1/pipecatcalls/" + id.String(), Method: sock.RequestMethodGet},
			handlerErr:   typedConflict.Wrap(dbhandler.ErrNotFound),
			expectStatus: 409,
		},
		{
			name:         "stop: typed error keeps its own status",
			request:      &sock.Request{URI: "/v1/pipecatcalls/" + id.String() + "/stop", Method: sock.RequestMethodPost},
			handlerErr:   typedConflict,
			expectStatus: 409,
		},
		{
			name:         "stop: legacy not found is 404",
			request:      &sock.Request{URI: "/v1/pipecatcalls/" + id.String() + "/stop", Method: sock.RequestMethodPost},
			handlerErr:   dbhandler.ErrNotFound,
			expectStatus: 404,
		},
		{
			name:         "stop: unclassified error is 400",
			request:      &sock.Request{URI: "/v1/pipecatcalls/" + id.String() + "/stop", Method: sock.RequestMethodPost},
			handlerErr:   errors.New("boom"),
			expectStatus: 400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockPipecatcall := pipecatcallhandler.NewMockPipecatcallHandler(mc)
			h := &listenHandler{
				sockHandler:        sockhandler.NewMockSockHandler(mc),
				pipecatcallHandler: mockPipecatcall,
			}

			if tt.request.Method == sock.RequestMethodGet {
				mockPipecatcall.EXPECT().Get(gomock.Any(), id).Return(nil, tt.handlerErr)
			} else {
				mockPipecatcall.EXPECT().Terminate(gomock.Any(), id).Return(nil, tt.handlerErr)
			}

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Fatalf("Wrong error. expect: nil, got: %v", err)
			}
			if res.StatusCode != tt.expectStatus {
				t.Errorf("Wrong status. expect: %d, got: %d", tt.expectStatus, res.StatusCode)
			}
		})
	}
}

func Test_processRequest_ping(t *testing.T) {
	tests := []struct {
		name string

		pingResult   *pipecatcall.PingResult
		expectStatus int
	}{
		{
			name:         "normal",
			pingResult:   &pipecatcall.PingResult{HostID: "host-a", Timestamp: time.Date(2020, 4, 18, 3, 22, 17, 0, time.UTC)},
			expectStatus: 200,
		},
		{
			// time.Time.MarshalJSON rejects a year outside [0, 9999].
			name:         "marshal failure is 500",
			pingResult:   &pipecatcall.PingResult{HostID: "host-a", Timestamp: time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)},
			expectStatus: 500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockPipecatcall := pipecatcallhandler.NewMockPipecatcallHandler(mc)
			h := &listenHandler{
				sockHandler:        sockhandler.NewMockSockHandler(mc),
				pipecatcallHandler: mockPipecatcall,
			}

			mockPipecatcall.EXPECT().Ping(gomock.Any()).Return(tt.pingResult, nil)

			res, err := h.processRequest(&sock.Request{URI: "/v1/ping", Method: sock.RequestMethodGet})
			if err != nil {
				t.Fatalf("Wrong error. expect: nil, got: %v", err)
			}
			if res.StatusCode != tt.expectStatus {
				t.Errorf("Wrong status. expect: %d, got: %d", tt.expectStatus, res.StatusCode)
			}
			if tt.expectStatus == 200 && len(res.Data) == 0 {
				t.Errorf("Wrong data. expect: the ping result body, got: empty")
			}
		})
	}
}

// waitDone waits for the consume goroutines started by the listen queues.
func waitDone(t *testing.T, wg *sync.WaitGroup) {
	t.Helper()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for the consume goroutines")
	}
}

func Test_Run(t *testing.T) {
	const (
		queue         = "test.queue"
		queueVolatile = "test.queue.volatile"
	)

	t.Run("creates both queues and consumes each", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		h := &listenHandler{sockHandler: mockSock}

		var wg sync.WaitGroup
		wg.Add(2)

		gomock.InOrder(
			mockSock.EXPECT().QueueCreate(queue, "normal").Return(nil),
			mockSock.EXPECT().QueueCreate(queueVolatile, "volatile").Return(nil),
		)
		// the callback is a method value and cannot be compared, so it is matched by type.
		mockSock.EXPECT().ConsumeRPC(gomock.Any(), queue, constConsumerName, false, false, false, 10, gomock.AssignableToTypeOf(sock.CbMsgRPC(nil))).
			DoAndReturn(func(context.Context, string, string, bool, bool, bool, int, sock.CbMsgRPC) error {
				defer wg.Done()
				return nil
			})
		mockSock.EXPECT().ConsumeRPC(gomock.Any(), queueVolatile, constConsumerName, false, false, false, 10, gomock.AssignableToTypeOf(sock.CbMsgRPC(nil))).
			DoAndReturn(func(context.Context, string, string, bool, bool, bool, int, sock.CbMsgRPC) error {
				defer wg.Done()
				return errors.New("consume failed")
			})

		if err := h.Run(queue, queueVolatile, "delay"); err != nil {
			t.Fatalf("Wrong error. expect: nil, got: %v", err)
		}
		waitDone(t, &wg)
	})

	t.Run("queue create failure stops before the volatile queue", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		h := &listenHandler{sockHandler: mockSock}

		// no other expectation: neither ConsumeRPC nor the volatile queue may be touched.
		mockSock.EXPECT().QueueCreate(queue, "normal").Return(errors.New("declare failed"))

		if err := h.Run(queue, queueVolatile, "delay"); err == nil {
			t.Errorf("Wrong error. expect: an error, got: nil")
		}
	})

	t.Run("volatile queue create failure is returned", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		h := &listenHandler{sockHandler: mockSock}

		var wg sync.WaitGroup
		wg.Add(1)

		mockSock.EXPECT().QueueCreate(queue, "normal").Return(nil)
		mockSock.EXPECT().ConsumeRPC(gomock.Any(), queue, constConsumerName, false, false, false, 10, gomock.AssignableToTypeOf(sock.CbMsgRPC(nil))).
			DoAndReturn(func(context.Context, string, string, bool, bool, bool, int, sock.CbMsgRPC) error {
				defer wg.Done()
				return nil
			})
		mockSock.EXPECT().QueueCreate(queueVolatile, "volatile").Return(errors.New("declare failed"))

		if err := h.Run(queue, queueVolatile, "delay"); err == nil {
			t.Errorf("Wrong error. expect: an error, got: nil")
		}
		waitDone(t, &wg)
	})
}
