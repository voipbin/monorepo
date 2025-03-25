package transcribehandler

import (
	"context"
	reflect "reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-transcribe-manager/models/streaming"
	"monorepo/bin-transcribe-manager/models/transcribe"
	"monorepo/bin-transcribe-manager/pkg/dbhandler"
	"monorepo/bin-transcribe-manager/pkg/streaminghandler"
	"monorepo/bin-transcribe-manager/pkg/transcripthandler"
)

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
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("e28b21dc-8218-11ed-b54f-d394b81cda3b"),
				},
				ReferenceType: transcribe.ReferenceTypeCall,
				Status:        transcribe.StatusProgressing,
			},

			&transcribe.Transcribe{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("e28b21dc-8218-11ed-b54f-d394b81cda3b"),
				},
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
			}

			ctx := context.Background()

			mockDB.EXPECT().TranscribeGet(ctx, tt.id).Return(tt.responseTranscribe, nil)

			// streamingTranscribeStop
			mockDB.EXPECT().TranscribeSetStatus(ctx, gomock.Any(), gomock.Any()).Return(nil)
			mockDB.EXPECT().TranscribeGet(gomock.Any(), gomock.Any()).Return(tt.responseTranscribe, nil)
			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())

			res, err := h.Stop(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_stopLive(t *testing.T) {

	tests := []struct {
		name string

		transcribe *transcribe.Transcribe
	}{
		{
			"normal",

			&transcribe.Transcribe{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("58ad260c-8789-11ec-87ad-63d573434c69"),
				},
				StreamingIDs: []uuid.UUID{
					uuid.FromStringOrNil("d5824a14-8788-11ec-9e71-a7cedf6ca3e1"),
					uuid.FromStringOrNil("df402f8a-8788-11ec-a14b-af9efb78ed6a"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockStreaming := streaminghandler.NewMockStreamingHandler(mc)
			mockTranscript := transcripthandler.NewMockTranscriptHandler(mc)

			h := &transcribeHandler{
				reqHandler:        mockReq,
				db:                mockDB,
				notifyHandler:     mockNotify,
				transcriptHandler: mockTranscript,
				streamingHandler:  mockStreaming,
			}

			ctx := context.Background()

			for _, stID := range tt.transcribe.StreamingIDs {
				mockStreaming.EXPECT().Stop(ctx, stID).Return(&streaming.Streaming{}, nil)
			}

			mockDB.EXPECT().TranscribeSetStatus(ctx, tt.transcribe.ID, transcribe.StatusDone).Return(nil)
			mockDB.EXPECT().TranscribeGet(gomock.Any(), tt.transcribe.ID).Return(tt.transcribe, nil)
			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), tt.transcribe.CustomerID, transcribe.EventTypeTranscribeDone, tt.transcribe)

			res, err := h.stopLive(ctx, tt.transcribe)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.transcribe, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.transcribe, res)
			}
		})
	}
}
