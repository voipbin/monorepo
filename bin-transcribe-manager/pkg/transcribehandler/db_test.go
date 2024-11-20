package transcribehandler

import (
	"context"
	reflect "reflect"
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"monorepo/bin-transcribe-manager/models/transcribe"
	"monorepo/bin-transcribe-manager/pkg/dbhandler"
	"monorepo/bin-transcribe-manager/pkg/transcripthandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_dbDelete(t *testing.T) {

	tests := []struct {
		name string

		id                 uuid.UUID
		responseTranscribe *transcribe.Transcribe

		expectRes *transcribe.Transcribe
	}{
		{
			"normal",

			uuid.FromStringOrNil("4452ca84-8781-11ec-a486-c77bd5b20dc8"),
			&transcribe.Transcribe{
				ID: uuid.FromStringOrNil("4452ca84-8781-11ec-a486-c77bd5b20dc8"),
			},

			&transcribe.Transcribe{
				ID: uuid.FromStringOrNil("4452ca84-8781-11ec-a486-c77bd5b20dc8"),
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
			mockGoogle := transcripthandler.NewMockTranscriptHandler(mc)

			h := &transcribeHandler{
				reqHandler:        mockReq,
				db:                mockDB,
				notifyHandler:     mockNotify,
				transcriptHandler: mockGoogle,
			}

			ctx := context.Background()

			mockDB.EXPECT().TranscribeDelete(ctx, tt.id).Return(nil)
			mockDB.EXPECT().TranscribeGet(ctx, tt.id).Return(tt.responseTranscribe, nil)
			mockNotify.EXPECT().PublishEvent(ctx, transcribe.EventTypeTranscribeDeleted, gomock.Any())

			res, err := h.dbDelete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}
