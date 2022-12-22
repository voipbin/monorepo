package transcribehandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/streaming"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/transcripthandler"
)

func Test_TranscribingStart_call(t *testing.T) {

	tests := []struct {
		name string

		customerID    uuid.UUID
		referenceType transcribe.ReferenceType
		referenceID   uuid.UUID
		language      string
		direction     transcribe.Direction

		responseCall       *cmcall.Call
		responseTranscribe *transcribe.Transcribe

		expectRes *transcribe.Transcribe
	}{
		{
			"normal",

			uuid.FromStringOrNil("0e259c1c-8211-11ed-a907-5bf5bd61fa6a"),
			transcribe.ReferenceTypeCall,
			uuid.FromStringOrNil("0e5ecd0c-8211-11ed-9c0a-4fa1d29f93c2"),
			"en-US",
			transcribe.DirectionBoth,

			&cmcall.Call{
				ID:     uuid.FromStringOrNil("0e5ecd0c-8211-11ed-9c0a-4fa1d29f93c2"),
				Status: cmcall.StatusProgressing,
			},

			&transcribe.Transcribe{
				ID: uuid.FromStringOrNil("5241c614-8216-11ed-9e05-ab1368296bbd"),
			},
			&transcribe.Transcribe{
				ID: uuid.FromStringOrNil("5241c614-8216-11ed-9e05-ab1368296bbd"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockTranscript := transcripthandler.NewMockTranscriptHandler(mc)

			h := &transcribeHandler{
				utilHandler:       mockUtil,
				reqHandler:        mockReq,
				db:                mockDB,
				notifyHandler:     mockNotify,
				transcriptHandler: mockTranscript,

				transcribeStreamingsMap: map[uuid.UUID][]*streaming.Streaming{},
			}

			ctx := context.Background()

			mockReq.EXPECT().CallV1CallGet(ctx, tt.referenceID).Return(tt.responseCall, nil)

			// streaming start
			mockUtil.EXPECT().CreateUUID().Return(utilhandler.CreateUUID())
			mockDB.EXPECT().TranscribeCreate(ctx, gomock.Any()).Return(nil)
			mockDB.EXPECT().TranscribeGet(ctx, gomock.Any()).Return(tt.responseTranscribe, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, gomock.Any(), gomock.Any(), gomock.Any())
			mockTranscript.EXPECT().Start(ctx, gomock.Any(), gomock.Any()).Return(&streaming.Streaming{}, nil).AnyTimes()

			res, err := h.TranscribingStart(ctx, tt.customerID, tt.referenceType, tt.referenceID, tt.language, tt.direction)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_TranscribingStop_call(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseTranscribe *transcribe.Transcribe

		expectRes *transcribe.Transcribe
	}{
		{
			"normal",

			uuid.FromStringOrNil("e28b21dc-8218-11ed-b54f-d394b81cda3b"),

			&transcribe.Transcribe{
				ID:            uuid.FromStringOrNil("e28b21dc-8218-11ed-b54f-d394b81cda3b"),
				ReferenceType: transcribe.ReferenceTypeCall,
				Status:        transcribe.StatusProgressing,
			},
			&transcribe.Transcribe{
				ID:            uuid.FromStringOrNil("e28b21dc-8218-11ed-b54f-d394b81cda3b"),
				ReferenceType: transcribe.ReferenceTypeCall,
				Status:        transcribe.StatusProgressing,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockTranscript := transcripthandler.NewMockTranscriptHandler(mc)

			h := &transcribeHandler{
				utilHandler:       mockUtil,
				reqHandler:        mockReq,
				db:                mockDB,
				notifyHandler:     mockNotify,
				transcriptHandler: mockTranscript,

				transcribeStreamingsMap: map[uuid.UUID][]*streaming.Streaming{},
			}

			ctx := context.Background()

			mockDB.EXPECT().TranscribeGet(ctx, tt.id).Return(tt.responseTranscribe, nil)

			// streamingTranscribeStop
			mockDB.EXPECT().TranscribeSetStatus(ctx, gomock.Any(), gomock.Any()).Return(nil)
			mockDB.EXPECT().TranscribeGet(gomock.Any(), gomock.Any()).Return(tt.responseTranscribe, nil)
			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())

			// mockReq.EXPECT().CallV1CallGet(ctx, tt.referenceID).Return(tt.responseCall, nil)

			// // streaming start
			// mockUtil.EXPECT().CreateUUID().Return(utilhandler.CreateUUID())
			// mockDB.EXPECT().TranscribeCreate(ctx, gomock.Any()).Return(nil)
			// mockDB.EXPECT().TranscribeGet(ctx, gomock.Any()).Return(tt.responseTranscribe, nil)
			// mockNotify.EXPECT().PublishWebhookEvent(ctx, gomock.Any(), gomock.Any(), gomock.Any())
			// mockTranscript.EXPECT().Start(ctx, gomock.Any(), gomock.Any()).Return(&streaming.Streaming{}, nil).AnyTimes()

			res, err := h.TranscribingStop(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
