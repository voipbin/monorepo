package transcribehandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/common"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/streaming"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/sttgoogle"
)

func TestStreamingTranscribeStart(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockGoogle := sttgoogle.NewMockSTTGoogle(mc)

	h := &transcribeHandler{
		reqHandler:    mockReq,
		db:            mockDB,
		notifyHandler: mockNotify,
		sttGoogle:     mockGoogle,

		transcribeStreamingsMap: map[uuid.UUID][]*streaming.Streaming{},
	}

	tests := []struct {
		name string

		customerID     uuid.UUID
		referenceID    uuid.UUID
		transcribeType transcribe.Type
		language       string

		responseTranscribe *transcribe.Transcribe

		expectRes *transcribe.Transcribe
	}{
		{
			"normal",

			uuid.FromStringOrNil("469b200c-8786-11ec-bd4f-bb7ae5541d57"),
			uuid.FromStringOrNil("47b30720-8786-11ec-ac47-f37c07bbbef5"),
			transcribe.TypeCall,
			"en-US",

			&transcribe.Transcribe{
				ID: uuid.FromStringOrNil("49a3529c-8786-11ec-928e-bb8e9b925697"),
			},

			&transcribe.Transcribe{
				ID: uuid.FromStringOrNil("49a3529c-8786-11ec-928e-bb8e9b925697"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockDB.EXPECT().TranscribeCreate(gomock.Any(), gomock.Any()).Return(nil)
			mockDB.EXPECT().TranscribeGet(gomock.Any(), gomock.Any()).Return(tt.responseTranscribe, nil)
			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), tt.responseTranscribe.CustomerID, transcribe.EventTypeTranscribeCreated, tt.responseTranscribe)
			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), tt.responseTranscribe.CustomerID, transcribe.EventTypeTranscribeStarted, tt.responseTranscribe)

			for _, direction := range []common.Direction{common.DirectionIn, common.DirectionOut} {
				mockGoogle.EXPECT().Start(gomock.Any(), tt.responseTranscribe, direction).Return(&streaming.Streaming{}, nil)
			}

			res, err := h.StreamingTranscribeStart(ctx, tt.customerID, tt.referenceID, tt.transcribeType, tt.language)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func TestStreamingTranscribeStop(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockGoogle := sttgoogle.NewMockSTTGoogle(mc)

	h := &transcribeHandler{
		reqHandler:    mockReq,
		db:            mockDB,
		notifyHandler: mockNotify,
		sttGoogle:     mockGoogle,

		transcribeStreamingsMap: map[uuid.UUID][]*streaming.Streaming{},
	}

	tests := []struct {
		name string

		transcribeID uuid.UUID
		streamings   []*streaming.Streaming

		responseTranscribe *transcribe.Transcribe
	}{
		{
			"normal",

			uuid.FromStringOrNil("58ad260c-8789-11ec-87ad-63d573434c69"),
			[]*streaming.Streaming{
				{
					ID: uuid.FromStringOrNil("d5824a14-8788-11ec-9e71-a7cedf6ca3e1"),
				},
				{
					ID: uuid.FromStringOrNil("df402f8a-8788-11ec-a14b-af9efb78ed6a"),
				},
			},

			&transcribe.Transcribe{
				ID: uuid.FromStringOrNil("58ad260c-8789-11ec-87ad-63d573434c69"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			h.addTranscribeStreamings(tt.transcribeID, tt.streamings)

			for _, st := range tt.streamings {
				mockGoogle.EXPECT().Stop(gomock.Any(), st).Return(nil)
			}
			mockDB.EXPECT().TranscribeGet(gomock.Any(), tt.transcribeID).Return(tt.responseTranscribe, nil)
			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), tt.responseTranscribe.CustomerID, transcribe.EventTypeTranscribeStopped, tt.responseTranscribe)

			if err := h.StreamingTranscribeStop(ctx, tt.transcribeID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
