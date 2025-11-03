package subscribehandler

import (
	"monorepo/bin-ai-manager/pkg/aicallhandler"
	cmdtmf "monorepo/bin-call-manager/models/dtmf"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_processEventCMDTMFReceived(t *testing.T) {

	tests := []struct {
		name  string
		event *sock.Event

		expectedEvent *cmdtmf.DTMF
	}{
		{
			name: "normal",

			event: &sock.Event{
				Publisher: "call-manager",
				Type:      "dtmf_received",
				DataType:  "application/json",
				Data:      []byte(`{"id":"db993672-b873-11f0-bccf-37a302bcc930"}`),
			},

			expectedEvent: &cmdtmf.DTMF{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("db993672-b873-11f0-bccf-37a302bcc930"),
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

			mockAIcall.EXPECT().EventDTMFReceived(gomock.Any(), tt.expectedEvent)

			h.processEvent(tt.event)

			time.Sleep(100 * time.Millisecond)
		})
	}
}
