package listenhandler

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/pkg/callhandler"
	"monorepo/bin-call-manager/pkg/outboundconfighandler"
)

func Test_processV1OutboundConfigsPost_createError(t *testing.T) {
	customerID := uuid.FromStringOrNil(testResID)

	tests := []struct {
		name string

		createErr error

		expectStatus int
	}{
		{
			name:         "duplicate entry is a conflict",
			createErr:    fmt.Errorf("Error 1062: Duplicate entry for key 'customer_id'"),
			expectStatus: 409,
		},
		{
			name:         "other failure is an internal error",
			createErr:    errTestBackend,
			expectStatus: 500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newResourceErrorHandler(mc)
			m.outboundConfig.EXPECT().Create(gomock.Any(), customerID, gomock.Any()).Return(nil, tt.createErr)

			res, err := h.processV1OutboundConfigsPost(context.Background(), &sock.Request{
				URI:  "/v1/outbound_configs",
				Data: []byte(`{"customer_id":"` + testResID + `","name":"cfg"}`),
			})
			if err != nil || res == nil || res.StatusCode != tt.expectStatus {
				t.Errorf("Wrong match. expect: %d, got: %v, %v", tt.expectStatus, res, err)
			}
		})
	}
}

func Test_listenHandler_Run(t *testing.T) {
	newHandler := func(mockSock sockhandler.SockHandler, mc *gomock.Controller) *listenHandler {
		return &listenHandler{
			sockHandler:           mockSock,
			callHandler:           callhandler.NewMockCallHandler(mc),
			outboundConfigHandler: outboundconfighandler.NewMockOutboundConfigHandler(mc),
		}
	}

	t.Run("queue declare failure is returned and nothing is consumed", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		mockSock.EXPECT().QueueCreate("test.queue", "normal").Return(errTestBackend)

		err := newHandler(mockSock, mc).Run("test.queue", "test.delay")
		if err == nil || !strings.Contains(err.Error(), "could not declare the queue") {
			t.Errorf("Wrong match. expect: declare error, got: %v", err)
		}
	})

	t.Run("requests are consumed from the declared queue", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		consumed := make(chan string, 1)

		mockSock.EXPECT().QueueCreate("test.queue", "normal").Return(nil)
		mockSock.EXPECT().ConsumeRPC(gomock.Any(), "test.queue", gomock.Any(), false, false, false, 10, gomock.Any()).
			DoAndReturn(func(_ context.Context, queue, _ string, _, _, _ bool, _ int, _ any) error {
				consumed <- queue
				return nil
			})

		if err := newHandler(mockSock, mc).Run("test.queue", "test.delay"); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}

		select {
		case queue := <-consumed:
			if queue != "test.queue" {
				t.Errorf("Wrong match. expect: test.queue, got: %s", queue)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("Wrong match. expect: consumer started, got: timeout")
		}
	})
}
