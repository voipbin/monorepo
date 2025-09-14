package aicallhandler

import (
	"context"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/pkg/aihandler"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	cmconfbridge "monorepo/bin-call-manager/models/confbridge"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	tmmessage "monorepo/bin-tts-manager/models/message"

	tmtranscribe "monorepo/bin-transcribe-manager/models/transcribe"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_EventTMPlayFinished(t *testing.T) {

	tests := []struct {
		name string

		evt *tmmessage.Message

		responseAIcall *aicall.AIcall

		responseTranscribe *tmtranscribe.Transcribe
		responseConfbridge *cmconfbridge.Confbridge

		expectedStreamingID  uuid.UUID
		expectedTranscribeID uuid.UUID
		expectedConfbridgeID uuid.UUID
	}{
		{
			name: "normal",

			evt: &tmmessage.Message{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c3f4df40-919e-11f0-b323-c35a63a7c2ea"),
				},
				StreamingID: uuid.FromStringOrNil("c4b0b544-919e-11f0-aadc-1736b1bcda1b"),
			},

			responseAIcall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c4d46818-919e-11f0-a76a-73b2d524da9b"),
				},
				Status:       aicall.StatusTerminating,
				TranscribeID: uuid.FromStringOrNil("c4eb82d2-919e-11f0-a6c2-5f2b4c268b25"),
				ConfbridgeID: uuid.FromStringOrNil("6be1c6c8-919f-11f0-aa05-6fa11ae38c9a"),
			},

			responseTranscribe: &tmtranscribe.Transcribe{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c4eb82d2-919e-11f0-a6c2-5f2b4c268b25"),
				},
			},
			responseConfbridge: &cmconfbridge.Confbridge{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("6be1c6c8-919f-11f0-aa05-6fa11ae38c9a"),
				},
			},

			expectedStreamingID:  uuid.FromStringOrNil("c4b0b544-919e-11f0-aadc-1736b1bcda1b"),
			expectedTranscribeID: uuid.FromStringOrNil("c4eb82d2-919e-11f0-a6c2-5f2b4c268b25"),
			expectedConfbridgeID: uuid.FromStringOrNil("6be1c6c8-919f-11f0-aa05-6fa11ae38c9a"),
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

			mockDB.EXPECT().AIcallGetByStreamingID(ctx, tt.expectedStreamingID).Return(tt.responseAIcall, nil)
			mockReq.EXPECT().TranscribeV1TranscribeStop(ctx, tt.expectedTranscribeID).Return(tt.responseTranscribe, nil)
			mockReq.EXPECT().CallV1ConfbridgeTerminate(ctx, tt.expectedConfbridgeID).Return(tt.responseConfbridge, nil)
			mockDB.EXPECT().AIcallUpdateStatusTerminated(ctx, tt.responseAIcall.ID).Return(nil)
			mockDB.EXPECT().AIcallGet(ctx, tt.responseAIcall.ID).Return(tt.responseAIcall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseAIcall.CustomerID, aicall.EventTypeStatusTerminated, tt.responseAIcall)

			h.EventTMPlayFinished(ctx, tt.evt)
		})
	}
}
