package transcripthandler

import (
	"context"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcript"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/dbhandler"
)

func Test_dbDelete(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseTranscript *transcript.Transcript
	}{
		{
			"normal",

			uuid.FromStringOrNil("2adc1780-f260-11ee-9069-8bb772a86cf3"),

			&transcript.Transcript{
				ID:       uuid.FromStringOrNil("2adc1780-f260-11ee-9069-8bb772a86cf3"),
				TMDelete: dbhandler.DefaultTimeStamp,
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

			h := &transcriptHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().TranscriptDelete(ctx, tt.id).Return(nil)
			mockDB.EXPECT().TranscriptGet(ctx, tt.id).Return(tt.responseTranscript, nil)
			mockNotify.EXPECT().PublishEvent(ctx, transcript.EventTypeTranscriptCreated, tt.responseTranscript)

			res, err := h.dbDelete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseTranscript, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseTranscript, res)
			}
		})
	}
}
