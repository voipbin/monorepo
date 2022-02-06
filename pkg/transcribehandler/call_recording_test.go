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

	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcript"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/sttgoogle"
)

func TestCallRecording(t *testing.T) {
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
	}

	tests := []struct {
		name string

		customerID uuid.UUID
		callID     uuid.UUID
		language   string

		responseCall        *cmcall.Call
		responseTranscribes []*transcribe.Transcribe

		expectRes []*transcribe.Transcribe
	}{
		{
			"normal",

			uuid.FromStringOrNil("419841c6-825d-11ec-823f-13ee3d677a1b"),
			uuid.FromStringOrNil("74582ca6-877c-11ec-937d-b3dc9da5953a"),
			"en-US",

			&cmcall.Call{
				RecordingIDs: []uuid.UUID{
					uuid.FromStringOrNil("a9e88118-877c-11ec-a30b-b7af76bdce58"),
				},
			},
			[]*transcribe.Transcribe{
				{
					ID: uuid.FromStringOrNil("564bbd4e-877d-11ec-84cc-978116c3fab9"),
				},
			},

			[]*transcribe.Transcribe{
				{
					ID: uuid.FromStringOrNil("564bbd4e-877d-11ec-84cc-978116c3fab9"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			lang := getBCP47LanguageCode(tt.language)
			mockReq.EXPECT().CMV1CallGet(gomock.Any(), tt.callID).Return(tt.responseCall, nil)

			for i, recordingID := range tt.responseCall.RecordingIDs {
				mockGoogle.EXPECT().Recording(gomock.Any(), recordingID, lang).Return(&transcript.Transcript{}, nil)
				mockDB.EXPECT().TranscribeCreate(gomock.Any(), gomock.Any()).Return(nil)
				mockDB.EXPECT().TranscribeGet(gomock.Any(), gomock.Any()).Return(tt.responseTranscribes[i], nil)
				mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), tt.responseTranscribes[i].CustomerID, transcribe.EventTypeTranscribeCreated, tt.responseTranscribes[i])
			}

			res, err := h.CallRecording(ctx, tt.customerID, tt.callID, tt.language)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
