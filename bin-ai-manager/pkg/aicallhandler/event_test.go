package aicallhandler

import (
	"context"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/pkg/aihandler"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/messagehandler"
	cmconfbridge "monorepo/bin-call-manager/models/confbridge"
	cmdtmf "monorepo/bin-call-manager/models/dtmf"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	pmpipecatcall "monorepo/bin-pipecat-manager/models/pipecatcall"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_EventDTMFRecevied(t *testing.T) {

	tests := []struct {
		name string

		evt *cmdtmf.DTMF

		responseAIcall *aicall.AIcall

		responsePipecatcall *pmpipecatcall.Pipecatcall
		responseConfbridge  *cmconfbridge.Confbridge

		expectedReferenceID   uuid.UUID
		expectedPipecatcallID uuid.UUID
		expectedHostID        string
		expectedMessageText   string
	}{
		{
			name: "normal",

			evt: &cmdtmf.DTMF{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c3f4df40-919e-11f0-b323-c35a63a7c2ea"),
				},
				CallID:   uuid.FromStringOrNil("8660e752-b86b-11f0-978b-476bcd1ad7a6"),
				Digit:    "9",
				Duration: 100,
			},

			responseAIcall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("8660e752-b86b-11f0-978b-476bcd1ad7a6"),
				},
				Status:        aicall.StatusTerminating,
				PipecatcallID: uuid.FromStringOrNil("868ec1ea-b86b-11f0-8293-57474c75fb86"),
			},

			responsePipecatcall: &pmpipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("868ec1ea-b86b-11f0-8293-57474c75fb86"),
				},
				HostID: "1.2.3.4",
			},
			responseConfbridge: &cmconfbridge.Confbridge{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("6be1c6c8-919f-11f0-aa05-6fa11ae38c9a"),
				},
			},

			expectedReferenceID:   uuid.FromStringOrNil("8660e752-b86b-11f0-978b-476bcd1ad7a6"),
			expectedPipecatcallID: uuid.FromStringOrNil("868ec1ea-b86b-11f0-8293-57474c75fb86"),
			expectedHostID:        "1.2.3.4",
			expectedMessageText:   "type: DTMF_EVENT\ndigit: 9\nduration: 100\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockAI := aihandler.NewMockAIHandler(mc)
			mockMessage := messagehandler.NewMockMessageHandler(mc)

			h := &aicallHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				notifyHandler:  mockNotify,
				db:             mockDB,
				aiHandler:      mockAI,
				messageHandler: mockMessage,
			}
			ctx := context.Background()

			mockDB.EXPECT().AIcallGetByReferenceID(ctx, tt.expectedReferenceID).Return(tt.responseAIcall, nil)
			mockReq.EXPECT().PipecatV1PipecatcallGet(ctx, tt.expectedPipecatcallID).Return(tt.responsePipecatcall, nil)
			mockMessage.EXPECT().Create(ctx, tt.responseAIcall.CustomerID, tt.responseAIcall.ID, message.DirectionOutgoing, message.RoleUser, tt.expectedMessageText, nil, "").Return(&message.Message{}, nil)
			mockReq.EXPECT().PipecatV1MessageSend(ctx, tt.responsePipecatcall.HostID, tt.expectedPipecatcallID, "", tt.expectedMessageText, true, false).Return(nil, nil)

			h.EventDTMFRecevied(ctx, tt.evt)
		})
	}
}
