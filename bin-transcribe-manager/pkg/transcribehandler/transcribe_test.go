package transcribehandler

import (
	"context"
	"reflect"
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"monorepo/bin-transcribe-manager/models/transcribe"
	"monorepo/bin-transcribe-manager/models/transcript"
	"monorepo/bin-transcribe-manager/pkg/dbhandler"
	"monorepo/bin-transcribe-manager/pkg/transcripthandler"
)

func Test_Get(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID
	}{
		{
			"normal",

			uuid.FromStringOrNil("5d0166e6-877f-11ec-b42f-4f6a59ece023"),
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

			mockDB.EXPECT().TranscribeGet(gomock.Any(), tt.id).Return(&transcribe.Transcribe{}, nil)
			_, err := h.Get(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}

func Test_GetByReferenceIDAndLanguage(t *testing.T) {

	tests := []struct {
		name string

		referenceID uuid.UUID
		language    string

		responseTranscribe *transcribe.Transcribe
	}{
		{
			"normal",

			uuid.FromStringOrNil("2fd5bd08-7f6c-11ed-8d71-67bb37305dd8"),
			"en-US",

			&transcribe.Transcribe{
				ID: uuid.FromStringOrNil("300196f8-7f6c-11ed-95d7-1f1fecd1ebc5"),
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

			mockDB.EXPECT().TranscribeGetByReferenceIDAndLanguage(ctx, tt.referenceID, tt.language).Return(tt.responseTranscribe, nil)
			_, err := h.GetByReferenceIDAndLanguage(ctx, tt.referenceID, tt.language)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_Gets(t *testing.T) {

	tests := []struct {
		name string

		size    uint64
		token   string
		filters map[string]string

		responseTranscribes []*transcribe.Transcribe
	}{
		{
			"normal",

			10,
			"2020-05-03%2021:35:02.809",
			map[string]string{
				"customer_id": "2fd5bd08-7f6c-11ed-8d71-67bb37305dd8",
			},

			[]*transcribe.Transcribe{
				{
					ID: uuid.FromStringOrNil("300196f8-7f6c-11ed-95d7-1f1fecd1ebc5"),
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
			mockGoogle := transcripthandler.NewMockTranscriptHandler(mc)

			h := &transcribeHandler{
				reqHandler:        mockReq,
				db:                mockDB,
				notifyHandler:     mockNotify,
				transcriptHandler: mockGoogle,
			}
			ctx := context.Background()

			mockDB.EXPECT().TranscribeGets(ctx, tt.size, tt.token, tt.filters).Return(tt.responseTranscribes, nil)
			_, err := h.Gets(ctx, tt.size, tt.token, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}

func Test_Create(t *testing.T) {

	tests := []struct {
		name string

		id            uuid.UUID
		customerID    uuid.UUID
		referenceType transcribe.ReferenceType
		referenceID   uuid.UUID
		language      string
		direction     transcribe.Direction
		streamingIDs  []uuid.UUID

		responseTranscribe *transcribe.Transcribe

		expectTranscribe *transcribe.Transcribe
	}{
		{
			"normal type call",

			uuid.FromStringOrNil("0afbb01e-986c-11ed-9fdb-d3bf0303c51c"),
			uuid.FromStringOrNil("5d0166e6-877f-11ec-b42f-4f6a59ece023"),
			transcribe.ReferenceTypeCall,
			uuid.FromStringOrNil("8a9bc0b2-7f6b-11ed-8cad-5b6ec2832ff4"),
			"en-US",
			transcribe.DirectionBoth,
			[]uuid.UUID{
				uuid.FromStringOrNil("fbd2802c-986b-11ed-83d3-e34b7b277be6"),
				uuid.FromStringOrNil("fc071828-986b-11ed-ab88-07e9d45c9d0f"),
			},

			&transcribe.Transcribe{
				ID: uuid.FromStringOrNil("0afbb01e-986c-11ed-9fdb-d3bf0303c51c"),
			},

			&transcribe.Transcribe{
				ID:            uuid.FromStringOrNil("0afbb01e-986c-11ed-9fdb-d3bf0303c51c"),
				CustomerID:    uuid.FromStringOrNil("5d0166e6-877f-11ec-b42f-4f6a59ece023"),
				ReferenceType: transcribe.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("8a9bc0b2-7f6b-11ed-8cad-5b6ec2832ff4"),
				Status:        transcribe.StatusProgressing,
				HostID:        testHostID,
				Language:      "en-US",
				Direction:     transcribe.DirectionBoth,
				StreamingIDs: []uuid.UUID{
					uuid.FromStringOrNil("fbd2802c-986b-11ed-83d3-e34b7b277be6"),
					uuid.FromStringOrNil("fc071828-986b-11ed-ab88-07e9d45c9d0f"),
				},
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

				hostID: testHostID,
			}

			ctx := context.Background()

			mockDB.EXPECT().TranscribeCreate(ctx, tt.expectTranscribe).Return(nil)
			mockDB.EXPECT().TranscribeGet(ctx, tt.id).Return(tt.responseTranscribe, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseTranscribe.CustomerID, transcribe.EventTypeTranscribeCreated, tt.responseTranscribe)
			res, err := h.Create(ctx, tt.id, tt.customerID, transcribe.ReferenceTypeCall, tt.referenceID, "en-US", transcribe.DirectionBoth, tt.streamingIDs)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseTranscribe, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseTranscribe, res)
			}
		})
	}
}

func Test_Delete(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseTranscribe *transcribe.Transcribe
		expectRes          *transcribe.Transcribe
	}{
		{
			"normal",

			uuid.FromStringOrNil("4452ca84-8781-11ec-a486-c77bd5b20dc8"),

			&transcribe.Transcribe{
				ID:            uuid.FromStringOrNil("4452ca84-8781-11ec-a486-c77bd5b20dc8"),
				Status:        transcribe.StatusDone,
				ReferenceType: transcribe.ReferenceTypeCall,
				TMDelete:      dbhandler.DefaultTimeStamp,
			},

			&transcribe.Transcribe{
				ID:            uuid.FromStringOrNil("4452ca84-8781-11ec-a486-c77bd5b20dc8"),
				Status:        transcribe.StatusDone,
				ReferenceType: transcribe.ReferenceTypeCall,
				TMDelete:      dbhandler.DefaultTimeStamp,
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
			mockTranscript := transcripthandler.NewMockTranscriptHandler(mc)

			h := &transcribeHandler{
				reqHandler:        mockReq,
				db:                mockDB,
				notifyHandler:     mockNotify,
				transcriptHandler: mockTranscript,
			}
			ctx := context.Background()

			mockDB.EXPECT().TranscribeGet(ctx, tt.id).Return(tt.responseTranscribe, nil)

			// deleteTranscripts
			mockTranscript.EXPECT().Gets(ctx, uint64(1000), "", gomock.Any()).Return([]*transcript.Transcript{}, nil)

			// dbDelete
			mockDB.EXPECT().TranscribeDelete(ctx, tt.id).Return(nil)
			mockDB.EXPECT().TranscribeGet(ctx, tt.id).Return(tt.responseTranscribe, nil)
			mockNotify.EXPECT().PublishEvent(ctx, transcribe.EventTypeTranscribeDeleted, gomock.Any())

			res, err := h.Delete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func Test_deleteTranscripts(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseTranscripts []*transcript.Transcript

		expectFilters map[string]string
	}{
		{
			"normal",

			uuid.FromStringOrNil("98a1e9ea-f25e-11ee-b2b9-03b097a87225"),

			[]*transcript.Transcript{
				{
					ID: uuid.FromStringOrNil("98e53588-f25e-11ee-9b2c-cb8f088fb4a0"),
				},
				{
					ID: uuid.FromStringOrNil("99090b48-f25e-11ee-a595-47b42745925b"),
				},
			},

			map[string]string{
				"transcribe_id": "98a1e9ea-f25e-11ee-b2b9-03b097a87225",
				"deleted":       "false",
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
			mockTranscript := transcripthandler.NewMockTranscriptHandler(mc)

			h := &transcribeHandler{
				reqHandler:        mockReq,
				db:                mockDB,
				notifyHandler:     mockNotify,
				transcriptHandler: mockTranscript,
			}
			ctx := context.Background()

			mockTranscript.EXPECT().Gets(ctx, uint64(1000), "", tt.expectFilters).Return(tt.responseTranscripts, nil)
			for _, tr := range tt.responseTranscripts {
				mockTranscript.EXPECT().Delete(ctx, tr.ID).Return(&transcript.Transcript{}, nil)
			}

			if err := h.deleteTranscripts(ctx, tt.id); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}

func Test_UpdateStatus(t *testing.T) {

	tests := []struct {
		name string

		id     uuid.UUID
		status transcribe.Status

		responseTranscribe *transcribe.Transcribe
		expectRes          *transcribe.Transcribe
	}{
		{
			"normal",

			uuid.FromStringOrNil("bec8dbda-7f6c-11ed-846e-bb48973f24fa"),
			transcribe.StatusProgressing,

			&transcribe.Transcribe{
				ID: uuid.FromStringOrNil("bec8dbda-7f6c-11ed-846e-bb48973f24fa"),
			},
			&transcribe.Transcribe{
				ID: uuid.FromStringOrNil("bec8dbda-7f6c-11ed-846e-bb48973f24fa"),
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

			mockDB.EXPECT().TranscribeSetStatus(ctx, tt.id, tt.status).Return(nil)
			mockDB.EXPECT().TranscribeGet(ctx, tt.id).Return(tt.responseTranscribe, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseTranscribe.CustomerID, transcribe.EventTypeTranscribeProgressing, tt.responseTranscribe)

			res, err := h.UpdateStatus(ctx, tt.id, tt.status)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}
