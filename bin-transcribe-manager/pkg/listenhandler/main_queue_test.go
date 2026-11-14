package listenhandler

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"monorepo/bin-transcribe-manager/pkg/transcribehandler"
	"monorepo/bin-transcribe-manager/pkg/transcripthandler"
)

func Test_NewListenHandler_keepsDependencies(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	hostID := uuid.FromStringOrNil("a14f0d10-0000-11ee-8c11-000000000101")
	mockSock := sockhandler.NewMockSockHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockTranscribe := transcribehandler.NewMockTranscribeHandler(mc)
	mockTranscript := transcripthandler.NewMockTranscriptHandler(mc)

	res := NewListenHandler(hostID, mockSock, mockReq, mockTranscribe, mockTranscript)

	h, ok := res.(*listenHandler)
	if !ok {
		t.Fatalf("Wrong match. expect: *listenHandler, got: %T", res)
	}
	if h.hostID != hostID || h.sockHandler != mockSock || h.reqHandler != mockReq || h.transcribeHandler != mockTranscribe || h.transcriptHandler != mockTranscript {
		t.Errorf("Wrong match. the dependencies were not kept: %+v", h)
	}
	if h.utilHandler == nil {
		t.Errorf("Wrong match. expect a default util handler")
	}
}

func Test_Run(t *testing.T) {
	queue := "bin-manager.transcribe-manager.request"
	queueVolatile := "bin-manager.transcribe-manager-host.request"

	tests := []struct {
		name string

		queueErr         error
		volatileQueueErr error

		// the number of consumers expected to start
		expectConsumers int
		expectVolatile  bool
		expectErr       bool
	}{
		{
			name:            "both queues are declared and consumed",
			expectConsumers: 2,
			expectVolatile:  true,
		},
		{
			name:      "the normal queue declaration fails and the volatile queue is never touched",
			queueErr:  fmt.Errorf("declare failed"),
			expectErr: true,
		},
		{
			name:             "the volatile queue declaration fails",
			volatileQueueErr: fmt.Errorf("declare failed"),
			expectConsumers:  1,
			expectVolatile:   true,
			expectErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			h := &listenHandler{sockHandler: mockSock}

			consumed := make(chan string, 2)
			consume := func(queueName string) {
				mockSock.EXPECT().ConsumeRPC(gomock.Any(), queueName, constCosumerName, false, false, false, 10, gomock.Any()).DoAndReturn(
					func(_ context.Context, name string, _ string, _ bool, _ bool, _ bool, _ int, _ sock.CbMsgRPC) error {
						consumed <- name
						return nil
					})
			}

			mockSock.EXPECT().QueueCreate(queue, "normal").Return(tt.queueErr)
			if tt.queueErr == nil {
				consume(queue)
			}
			if tt.expectVolatile {
				mockSock.EXPECT().QueueCreate(queueVolatile, "volatile").Return(tt.volatileQueueErr)
				if tt.volatileQueueErr == nil {
					consume(queueVolatile)
				}
			}

			err := h.Run(queue, queueVolatile, "")
			if (err != nil) != tt.expectErr {
				t.Fatalf("Wrong match. expect error: %v, got: %v", tt.expectErr, err)
			}

			// each consumer runs in its own goroutine; wait for them before the controller is finished
			got := map[string]bool{}
			for i := 0; i < tt.expectConsumers; i++ {
				select {
				case name := <-consumed:
					got[name] = true
				case <-time.After(5 * time.Second):
					t.Fatalf("Wrong match. expect %d consumers, got: %v", tt.expectConsumers, got)
				}
			}
			if tt.expectConsumers == 2 && (!got[queue] || !got[queueVolatile]) {
				t.Errorf("Wrong match. expect both queues consumed, got: %v", got)
			}
		})
	}
}
