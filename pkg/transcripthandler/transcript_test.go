package transcripthandler

import (
	"context"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcript"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/dbhandler"
)

func Test_Create(t *testing.T) {

	tests := []struct {
		name string

		customerID   uuid.UUID
		transcribeID uuid.UUID
		direction    transcript.Direction
		message      string
		tmTranscript string

		responseUUID       uuid.UUID
		responseTranscript *transcript.Transcript

		expectReqCreate *transcript.Transcript
	}{
		{
			"normal",

			uuid.FromStringOrNil("0d662fb2-7eb5-11ed-9d23-d724d9c70f65"),
			uuid.FromStringOrNil("0da54c10-7eb5-11ed-b190-43412cc32f80"),
			transcript.DirectionIn,
			"test transcript",
			"0000-00-00 00:00:01.00000",

			uuid.FromStringOrNil("494f5bfc-7eb5-11ed-a6d7-07162f18f28e"),
			&transcript.Transcript{
				ID:           uuid.FromStringOrNil("494f5bfc-7eb5-11ed-a6d7-07162f18f28e"),
				CustomerID:   uuid.FromStringOrNil("0d662fb2-7eb5-11ed-9d23-d724d9c70f65"),
				TranscribeID: uuid.FromStringOrNil("0da54c10-7eb5-11ed-b190-43412cc32f80"),
				Direction:    transcript.DirectionIn,
				Message:      "test transcript",
				TMTranscript: "0000-00-00 00:00:01.00000",
			},

			&transcript.Transcript{
				ID:           uuid.FromStringOrNil("494f5bfc-7eb5-11ed-a6d7-07162f18f28e"),
				CustomerID:   uuid.FromStringOrNil("0d662fb2-7eb5-11ed-9d23-d724d9c70f65"),
				TranscribeID: uuid.FromStringOrNil("0da54c10-7eb5-11ed-b190-43412cc32f80"),
				Direction:    transcript.DirectionIn,
				Message:      "test transcript",
				TMTranscript: "0000-00-00 00:00:01.00000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &transcriptHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockUtil.EXPECT().CreateUUID().Return(tt.responseUUID)
			mockDB.EXPECT().TranscriptCreate(ctx, tt.expectReqCreate).Return(nil)
			mockDB.EXPECT().TranscriptGet(ctx, tt.responseUUID).Return(tt.responseTranscript, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseTranscript.CustomerID, transcript.EventTypeTranscriptCreated, tt.responseTranscript)

			res, err := h.Create(ctx, tt.customerID, tt.transcribeID, tt.direction, tt.message, tt.tmTranscript)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseTranscript) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseTranscript, res)
			}
		})
	}
}

func Test_Gets(t *testing.T) {

	tests := []struct {
		name string

		transcribeID uuid.UUID

		responseTranscripts []*transcript.Transcript

		expectReqCreate []*transcript.Transcript
	}{
		{
			"normal",

			uuid.FromStringOrNil("87d87c60-821a-11ed-a6e2-4f8ea6cd0a1d"),

			[]*transcript.Transcript{
				{
					ID: uuid.FromStringOrNil("c33a05bc-821a-11ed-91e6-b34b3f52cdf9"),
				},
				{
					ID: uuid.FromStringOrNil("c36d26b8-821a-11ed-aeae-83a320a63874"),
				},
			},

			[]*transcript.Transcript{
				{
					ID: uuid.FromStringOrNil("c33a05bc-821a-11ed-91e6-b34b3f52cdf9"),
				},
				{
					ID: uuid.FromStringOrNil("c36d26b8-821a-11ed-aeae-83a320a63874"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &transcriptHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().TranscriptGetsByTranscribeID(ctx, tt.transcribeID).Return(tt.responseTranscripts, nil)

			res, err := h.Gets(ctx, tt.transcribeID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectReqCreate) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectReqCreate, res)
			}
		})
	}
}
