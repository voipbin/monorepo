package transcribehandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/common"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/streaming"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/transcirpthandler"
)

func Test_StreamingTranscribeStart(t *testing.T) {

	tests := []struct {
		name string

		customerID    uuid.UUID
		referenceType transcribe.ReferenceType
		referenceID   uuid.UUID
		language      string
		direction     transcribe.Direction

		responseTranscribe *transcribe.Transcribe

		expectRes *transcribe.Transcribe
	}{
		{
			"normal",

			uuid.FromStringOrNil("469b200c-8786-11ec-bd4f-bb7ae5541d57"),
			transcribe.ReferenceTypeCall,
			uuid.FromStringOrNil("47b30720-8786-11ec-ac47-f37c07bbbef5"),
			"en-US",
			transcribe.DirectionBoth,

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
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockGoogle := transcirpthandler.NewMockTranscriptHandler(mc)

			h := &transcribeHandler{
				utilHandler:       mockUtil,
				reqHandler:        mockReq,
				db:                mockDB,
				notifyHandler:     mockNotify,
				transcriptHandler: mockGoogle,

				transcribeStreamingsMap: map[uuid.UUID][]*streaming.Streaming{},
			}

			ctx := context.Background()

			// create
			mockUtil.EXPECT().CreateUUID().Return(utilhandler.CreateUUID())
			mockDB.EXPECT().TranscribeCreate(ctx, gomock.Any()).Return(nil)
			mockDB.EXPECT().TranscribeGet(ctx, gomock.Any()).Return(tt.responseTranscribe, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseTranscribe.CustomerID, transcribe.EventTypeTranscribeCreated, tt.responseTranscribe)

			// // update status
			// mockDB.EXPECT().TranscribeSetStatus(ctx, tt.responseTranscribe.ID, transcribe.StatusProgressing).Return(nil)
			// mockDB.EXPECT().TranscribeGet(ctx, gomock.Any()).Return(tt.responseTranscribe, nil)
			// mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseTranscribe.CustomerID, transcribe.EventTypeTranscribeProgressing, tt.responseTranscribe)

			for _, direction := range []common.Direction{common.DirectionIn, common.DirectionOut} {
				mockGoogle.EXPECT().Start(ctx, tt.responseTranscribe, direction).Return(&streaming.Streaming{}, nil)
			}

			res, err := h.StreamingTranscribeStart(ctx, tt.customerID, tt.referenceType, tt.referenceID, tt.language, tt.direction)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func Test_StreamingTranscribeStop(t *testing.T) {

	tests := []struct {
		name string

		transcribeID uuid.UUID
		streamings   []*streaming.Streaming

		responseTranscribe *transcribe.Transcribe

		expectRes *transcribe.Transcribe
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

			&transcribe.Transcribe{
				ID: uuid.FromStringOrNil("58ad260c-8789-11ec-87ad-63d573434c69"),
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
			mockGoogle := transcirpthandler.NewMockTranscriptHandler(mc)

			h := &transcribeHandler{
				reqHandler:        mockReq,
				db:                mockDB,
				notifyHandler:     mockNotify,
				transcriptHandler: mockGoogle,

				transcribeStreamingsMap: map[uuid.UUID][]*streaming.Streaming{},
			}

			ctx := context.Background()

			h.addTranscribeStreamings(tt.transcribeID, tt.streamings)

			for _, st := range tt.streamings {
				mockGoogle.EXPECT().Stop(gomock.Any(), st).Return(nil)
			}

			mockDB.EXPECT().TranscribeSetStatus(ctx, tt.transcribeID, transcribe.StatusDone).Return(nil)
			mockDB.EXPECT().TranscribeGet(gomock.Any(), tt.transcribeID).Return(tt.responseTranscribe, nil)
			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), tt.responseTranscribe.CustomerID, transcribe.EventTypeTranscribeDone, tt.responseTranscribe)

			res, err := h.StreamingTranscribeStop(ctx, tt.transcribeID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
