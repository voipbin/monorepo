package aicallhandler

import (
	"context"
	"reflect"
	"testing"

	cmconfbridge "monorepo/bin-call-manager/models/confbridge"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	pmpipecatcall "monorepo/bin-pipecat-manager/models/pipecatcall"

	tmtranscribe "monorepo/bin-transcribe-manager/models/transcribe"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/pkg/aihandler"
	"monorepo/bin-ai-manager/pkg/dbhandler"
)

func Test_ProcessStart(t *testing.T) {

	tests := []struct {
		name string

		aicall *aicall.AIcall

		responseTranscribe *tmtranscribe.Transcribe
	}{
		{
			name: "normal",

			aicall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("6ed69462-a705-11ed-a47b-cfb979f9f07d"),
					CustomerID: uuid.FromStringOrNil("6f12ea52-a705-11ed-86d3-8b796a5da603"),
				},
				ActiveflowID:  uuid.FromStringOrNil("5b2ad484-093a-11f0-928b-3bd49b19fd87"),
				ReferenceType: aicall.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("6f69db50-a705-11ed-bc35-177b3c1673d4"),
				Language:      "en-US",
			},

			responseTranscribe: &tmtranscribe.Transcribe{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("6f40a2c6-a705-11ed-8981-d78afab8acba"),
				},
			},
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

			h := &aicallHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				db:            mockDB,
				aiHandler:     mockAI,
			}

			ctx := context.Background()

			mockDB.EXPECT().AIcallUpdateStatus(ctx, tt.aicall.ID, aicall.StatusProgressing).Return(nil)
			mockDB.EXPECT().AIcallGet(ctx, tt.aicall.ID).Return(tt.aicall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.aicall.CustomerID, aicall.EventTypeStatusProgressing, tt.aicall)

			res, err := h.ProcessStart(ctx, tt.aicall)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.aicall) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.aicall, res)
			}
		})
	}
}

func Test_ProcessTerminate(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseAicall      *aicall.AIcall
		responsePipecatcall *pmpipecatcall.Pipecatcall
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("dd188916-d791-11f0-b284-4359b8729dde"),

			responseAicall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("dd188916-d791-11f0-b284-4359b8729dde"),
				},
				ConfbridgeID:  uuid.FromStringOrNil("a213c5f8-d794-11f0-9e01-3738b1dbf1d6"),
				PipecatcallID: uuid.FromStringOrNil("a2460c84-d794-11f0-a9e3-8ffda8bbf25b"),
			},
			responsePipecatcall: &pmpipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a2460c84-d794-11f0-a9e3-8ffda8bbf25b"),
				},
				HostID: "host-12345",
			},
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

			h := &aicallHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				db:            mockDB,
				aiHandler:     mockAI,
			}
			ctx := context.Background()

			mockDB.EXPECT().AIcallGet(ctx, tt.id).Return(tt.responseAicall, nil)
			mockReq.EXPECT().FlowV1ActiveflowServiceStop(ctx, tt.responseAicall.ActiveflowID, tt.responseAicall.ID, 0).Return(nil)
			if tt.responseAicall.ReferenceType != aicall.ReferenceTypeCall {
				mockReq.EXPECT().FlowV1ActiveflowContinue(ctx, tt.responseAicall.ActiveflowID, tt.responseAicall.ID).Return(nil)
			}

			if tt.responseAicall.PipecatcallID != uuid.Nil {
				mockReq.EXPECT().PipecatV1PipecatcallGet(ctx, tt.responseAicall.PipecatcallID).Return(tt.responsePipecatcall, nil)
				mockReq.EXPECT().PipecatV1PipecatcallTerminate(ctx, tt.responsePipecatcall.HostID, tt.responsePipecatcall.ID).Return(tt.responsePipecatcall, nil)
			}

			if tt.responseAicall.ConfbridgeID != uuid.Nil {
				mockReq.EXPECT().CallV1ConfbridgeTerminate(ctx, tt.responseAicall.ConfbridgeID).Return(&cmconfbridge.Confbridge{}, nil)
			}

			mockDB.EXPECT().AIcallUpdateStatus(ctx, tt.responseAicall.ID, aicall.StatusTerminated).Return(nil)
			mockDB.EXPECT().AIcallGet(ctx, tt.responseAicall.ID).Return(tt.responseAicall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseAicall.CustomerID, aicall.EventTypeStatusTerminated, tt.responseAicall)

			res, err := h.ProcessTerminate(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseAicall) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseAicall, res)
			}
		})
	}
}
