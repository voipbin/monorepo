package subscribehandler

import (
	"monorepo/bin-ai-manager/pkg/aicallhandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	pmpipecatcall "monorepo/bin-pipecat-manager/models/pipecatcall"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_Test_processEventPMPipecatcallInitialized(t *testing.T) {

	tests := []struct {
		name  string
		event *sock.Event

		expectedEvent *pmpipecatcall.Pipecatcall
	}{
		{
			name: "normal",

			event: &sock.Event{
				Publisher: "pipecat-manager",
				Type:      pmpipecatcall.EventTypeInitialized,
				DataType:  "application/json",
				Data:      []byte(`{"id":"cbfc5f0e-cb5c-11f0-80f1-23452c78fe7c"}`),
			},

			expectedEvent: &pmpipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cbfc5f0e-cb5c-11f0-80f1-23452c78fe7c"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockAIcall := aicallhandler.NewMockAIcallHandler(mc)

			h := subscribeHandler{
				sockHandler:   mockSock,
				aicallHandler: mockAIcall,
			}

			mockAIcall.EXPECT().EventPMPipecatcallInitialized(gomock.Any(), tt.expectedEvent)

			h.processEvent(tt.event)

			time.Sleep(100 * time.Millisecond)
		})
	}
}
