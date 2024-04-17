package transcripthandler

import (
	"context"
	reflect "reflect"
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"monorepo/bin-transcribe-manager/models/transcript"
	"monorepo/bin-transcribe-manager/pkg/dbhandler"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
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
