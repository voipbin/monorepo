package aicallhandler

import (
	"context"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/pkg/aihandler"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/messagehandler"
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

func Test_EventDTMFReceived(t *testing.T) {

	tests := []struct {
		name string

		evt *cmdtmf.DTMF

		responseAIcall      *aicall.AIcall
		responseMessage     *message.Message
		responsePipecatcall *pmpipecatcall.Pipecatcall

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
			responseMessage: &message.Message{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("da41abd4-b87c-11f0-924e-47f942b42bf4"),
				},
			},
			responsePipecatcall: &pmpipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("868ec1ea-b86b-11f0-8293-57474c75fb86"),
				},
				HostID: "1.2.3.4",
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
			mockMessage.EXPECT().Create(ctx, tt.responseAIcall.CustomerID, tt.responseAIcall.ID, message.DirectionOutgoing, message.RoleUser, tt.expectedMessageText, nil, "").Return(tt.responseMessage, nil)
			mockReq.EXPECT().PipecatV1MessageSend(ctx, tt.responsePipecatcall.HostID, tt.expectedPipecatcallID, tt.responseMessage.ID.String(), tt.expectedMessageText, true, true).Return(nil, nil)

			h.EventCMDTMFReceived(ctx, tt.evt)
		})
	}
}

func Test_EventPMPipecatcallInitialzided(t *testing.T) {

	tests := []struct {
		name string

		evt *pmpipecatcall.Pipecatcall

		responseAIcall *aicall.AIcall

		expectedAICallID uuid.UUID
		expectedCallID   uuid.UUID
	}{
		{
			name: "normal",

			evt: &pmpipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("01d966e0-cb5c-11f0-be7e-774d531e6ec8"),
				},
				ReferenceType: pmpipecatcall.ReferenceTypeAICall,
				ReferenceID:   uuid.FromStringOrNil("021532d8-cb5c-11f0-8f38-df7986b6fe53"),
			},

			responseAIcall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("021532d8-cb5c-11f0-8f38-df7986b6fe53"),
				},
				ReferenceType: aicall.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("0246703c-cb5c-11f0-ba32-e30e51dfb4e2"),
			},

			expectedAICallID: uuid.FromStringOrNil("021532d8-cb5c-11f0-8f38-df7986b6fe53"),
			expectedCallID:   uuid.FromStringOrNil("0246703c-cb5c-11f0-ba32-e30e51dfb4e2"),
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

			mockDB.EXPECT().AIcallGet(ctx, tt.expectedAICallID).Return(tt.responseAIcall, nil)
			mockReq.EXPECT().CallV1CallMediaStop(ctx, tt.expectedCallID).Return(nil)

			h.EventPMPipecatcallInitialzided(ctx, tt.evt)
		})
	}
}
