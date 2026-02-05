package transcripthandler

import (
	"context"
	reflect "reflect"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-transcribe-manager/models/transcript"
	"monorepo/bin-transcribe-manager/pkg/dbhandler"
)

func Test_Create(t *testing.T) {

	tmTranscript := time.Date(0, 0, 0, 0, 0, 1, 0, time.UTC)

	tests := []struct {
		name string

		customerID   uuid.UUID
		transcribeID uuid.UUID
		direction    transcript.Direction
		message      string
		tmTranscript *time.Time

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
			&tmTranscript,

			uuid.FromStringOrNil("494f5bfc-7eb5-11ed-a6d7-07162f18f28e"),
			&transcript.Transcript{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("494f5bfc-7eb5-11ed-a6d7-07162f18f28e"),
					CustomerID: uuid.FromStringOrNil("0d662fb2-7eb5-11ed-9d23-d724d9c70f65"),
				},
				TranscribeID: uuid.FromStringOrNil("0da54c10-7eb5-11ed-b190-43412cc32f80"),
				Direction:    transcript.DirectionIn,
				Message:      "test transcript",
				TMTranscript: &tmTranscript,
			},

			&transcript.Transcript{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("494f5bfc-7eb5-11ed-a6d7-07162f18f28e"),
					CustomerID: uuid.FromStringOrNil("0d662fb2-7eb5-11ed-9d23-d724d9c70f65"),
				},
				TranscribeID: uuid.FromStringOrNil("0da54c10-7eb5-11ed-b190-43412cc32f80"),
				Direction:    transcript.DirectionIn,
				Message:      "test transcript",
				TMTranscript: &tmTranscript,
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

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
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

func Test_List(t *testing.T) {

	tests := []struct {
		name string

		size    uint64
		token   string
		filters map[transcript.Field]any

		responseTranscripts []*transcript.Transcript
	}{
		{
			name: "normal",

			size:  10,
			token: "2020-05-03T21:35:02.809Z",
			filters: map[transcript.Field]any{
				transcript.FieldCustomerID: uuid.FromStringOrNil("cf322d78-ed98-11ee-813d-1ff686765c1f"),
			},

			responseTranscripts: []*transcript.Transcript{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("cf8dc02a-ed98-11ee-bc86-53c66222068a"),
					},
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

			h := &transcriptHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().TranscriptList(ctx, tt.size, tt.token, tt.filters).Return(tt.responseTranscripts, nil)
			_, err := h.List(ctx, tt.size, tt.token, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}

func Test_Delete(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseTranscript *transcript.Transcript
	}{
		{
			"normal",

			uuid.FromStringOrNil("87cf2e7e-f25f-11ee-81cd-1f9ea9d83ffb"),

			&transcript.Transcript{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("87cf2e7e-f25f-11ee-81cd-1f9ea9d83ffb"),
				},
				TMDelete: nil,
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

			mockDB.EXPECT().TranscriptGet(ctx, tt.id).Return(tt.responseTranscript, nil)

			// dbDelete
			mockDB.EXPECT().TranscriptDelete(ctx, tt.id).Return(nil)
			mockDB.EXPECT().TranscriptGet(ctx, tt.id).Return(tt.responseTranscript, nil)
			mockNotify.EXPECT().PublishEvent(ctx, transcript.EventTypeTranscriptCreated, tt.responseTranscript)

			res, err := h.Delete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseTranscript, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseTranscript, res)
			}
		})
	}
}
