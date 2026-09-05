package aicallhandler

import (
	"context"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/pkg/aihandler"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/messagehandler"
	cmcall "monorepo/bin-call-manager/models/call"
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

func Test_EventCMDTMFReceived(t *testing.T) {

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
			mockMessage.EXPECT().Create(ctx, uuid.Nil, tt.responseAIcall.CustomerID, tt.responseAIcall.ID, tt.responseAIcall.ActiveflowID, message.DirectionOutgoing, message.RoleUser, tt.expectedMessageText, nil, "", gomock.Any()).Return(tt.responseMessage, nil)
			mockReq.EXPECT().PipecatV1Ping(gomock.Any(), tt.responsePipecatcall.HostID).Return(nil)
			mockReq.EXPECT().PipecatV1MessageSend(ctx, tt.responsePipecatcall.HostID, tt.expectedPipecatcallID, tt.responseMessage.ID.String(), tt.expectedMessageText, true, true).Return(nil, nil)

			h.EventCMDTMFReceived(ctx, tt.evt)
		})
	}
}

// Test_EventCMCallHangup_TerminateFailureIsLoggedNotFatal pins review round 2's
// LOW-2 and the contract around it.
//
// The ProcessTerminate call on this path is deliberately NON-BLOCKING -- the
// listening-AIcall sweep below it must run whether or not the terminate
// succeeded, because the call is gone either way. What changed is only that the
// error is no longer discarded silently; the sweep still runs, which is what
// this test actually asserts (a log line is not observable from here, but a
// swallowed-and-then-returned error would be).
func Test_EventCMCallHangup_TerminateFailureIsLoggedNotFatal(t *testing.T) {
	callID := uuid.FromStringOrNil("2b1a7f1c-0000-4000-8000-000000000001")
	aicallID := uuid.FromStringOrNil("2b1a7f1c-0000-4000-8000-000000000002")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &aicallHandler{db: mockDB}
	ctx := context.Background()

	// The AIcall whose reference IS this call is found...
	mockDB.EXPECT().AIcallGetByReferenceID(ctx, callID).Return(&aicall.AIcall{
		Identity: commonidentity.Identity{ID: aicallID},
		Status:   aicall.StatusProgressing,
	}, nil)
	// ...but ProcessTerminate's own read fails, so the terminate errors out.
	mockDB.EXPECT().AIcallGet(ctx, aicallID).Return(nil, dbhandler.ErrNotFound)

	// THE SWEEP STILL RUNS. This is the property the non-blocking discard was
	// protecting and that adding the log must not change.
	mockDB.EXPECT().AIcallList(ctx, uint64(listenStopPageSize), "", gomock.Any()).
		Return([]*aicall.AIcall{}, nil).Times(1)

	h.EventCMCallHangup(ctx, &cmcall.Call{Identity: commonidentity.Identity{ID: callID}})
}

func Test_EventPMPipecatcallInitialized(t *testing.T) {

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

			h.EventPMPipecatcallInitialized(ctx, tt.evt)
		})
	}
}
