package aicallhandler

import (
	"context"
	"reflect"
	"testing"

	cmconfbridge "monorepo/bin-call-manager/models/confbridge"

	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

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
			"normal",

			&aicall.AIcall{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("6ed69462-a705-11ed-a47b-cfb979f9f07d"),
					CustomerID: uuid.FromStringOrNil("6f12ea52-a705-11ed-86d3-8b796a5da603"),
				},
				ReferenceType: aicall.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("6f69db50-a705-11ed-bc35-177b3c1673d4"),
				Language:      "en-US",
			},

			&tmtranscribe.Transcribe{
				ID: uuid.FromStringOrNil("6f40a2c6-a705-11ed-8981-d78afab8acba"),
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

			mockReq.EXPECT().TranscribeV1TranscribeStart(ctx, tt.aicall.CustomerID, tmtranscribe.ReferenceTypeCall, tt.aicall.ReferenceID, tt.aicall.Language, tmtranscribe.DirectionIn).Return(tt.responseTranscribe, nil)
			mockDB.EXPECT().AIcallUpdateStatusProgressing(ctx, tt.aicall.ID, tt.responseTranscribe.ID).Return(nil)
			mockDB.EXPECT().AIcallGet(ctx, tt.aicall.ID).Return(tt.aicall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.aicall.CustomerID, aicall.EventTypeProgressing, tt.aicall)

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

func Test_ProcessEnd(t *testing.T) {

	tests := []struct {
		name string

		aicall *aicall.AIcall

		responseTranscribe *tmtranscribe.Transcribe
	}{
		{
			"normal",

			&aicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("a7c462f8-a706-11ed-9461-cbd173399722"),
				},
				ConfbridgeID: uuid.FromStringOrNil("fe18ea48-e12d-43cb-8b40-48caeed6d67b"),
				TranscribeID: uuid.FromStringOrNil("a7f1d814-a706-11ed-9af7-3f37982d3546"),
			},

			&tmtranscribe.Transcribe{
				ID: uuid.FromStringOrNil("a7f1d814-a706-11ed-9af7-3f37982d3546"),
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

			mockReq.EXPECT().TranscribeV1TranscribeStop(ctx, tt.aicall.TranscribeID).Return(&tmtranscribe.Transcribe{}, nil)
			mockDB.EXPECT().AIcallUpdateStatusEnd(ctx, tt.aicall.ID).Return(nil)
			mockDB.EXPECT().AIcallGet(ctx, tt.aicall.ID).Return(tt.aicall, nil)
			mockReq.EXPECT().CallV1ConfbridgeDelete(ctx, tt.aicall.ConfbridgeID).Return(&cmconfbridge.Confbridge{}, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.aicall.CustomerID, aicall.EventTypeEnd, tt.aicall)

			res, err := h.ProcessEnd(ctx, tt.aicall)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.aicall) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.aicall, res)
			}
		})
	}
}
