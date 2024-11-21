package conferencehandler

import (
	"context"
	"reflect"
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	tmtranscribe "monorepo/bin-transcribe-manager/models/transcribe"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-conference-manager/models/conference"
	"monorepo/bin-conference-manager/pkg/dbhandler"
)

func Test_TranscribeStart(t *testing.T) {

	tests := []struct {
		name string

		id   uuid.UUID
		lang string

		responseConference *conference.Conference
		responseTranscribe *tmtranscribe.Transcribe
	}{
		{
			name: "normal",

			id:   uuid.FromStringOrNil("dd016dfa-98c2-11ed-8725-2712895395fe"),
			lang: "en-US",

			responseConference: &conference.Conference{
				ID:           uuid.FromStringOrNil("dd016dfa-98c2-11ed-8725-2712895395fe"),
				CustomerID:   uuid.FromStringOrNil("dd5a2b20-98c2-11ed-b9f1-d77ec9cc4ee1"),
				ConfbridgeID: uuid.FromStringOrNil("dd7fe02c-98c2-11ed-b244-53a1c8648a76"),
				Type:         conference.TypeConference,
				Status:       conference.StatusProgressing,
			},
			responseTranscribe: &tmtranscribe.Transcribe{
				ID: uuid.FromStringOrNil("dd2c7b4e-98c2-11ed-a286-9b582399a47e"),
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := conferenceHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().ConferenceGet(ctx, tt.id).Return(tt.responseConference, nil)
			mockReq.EXPECT().TranscribeV1TranscribeStart(ctx, tt.responseConference.CustomerID, tmtranscribe.ReferenceTypeConfbridge, tt.responseConference.ConfbridgeID, tt.lang, tmtranscribe.DirectionIn).Return(tt.responseTranscribe, nil)
			mockDB.EXPECT().ConferenceSetTranscribeID(ctx, tt.responseConference.ID, tt.responseTranscribe.ID).Return(nil)
			mockDB.EXPECT().ConferenceAddTranscribeIDs(ctx, tt.id, tt.responseTranscribe.ID).Return(nil)
			mockDB.EXPECT().ConferenceGet(ctx, tt.id).Return(tt.responseConference, nil)

			res, err := h.TranscribeStart(ctx, tt.id, tt.lang)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseConference, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseConference, res)
			}
		})
	}
}

func Test_TranscribeStop(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseConference *conference.Conference
		responseTranscribe *tmtranscribe.Transcribe
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("cfa6500c-98c3-11ed-9f10-c3f50b15365f"),

			responseConference: &conference.Conference{
				ID:           uuid.FromStringOrNil("cfa6500c-98c3-11ed-9f10-c3f50b15365f"),
				Type:         conference.TypeConference,
				Status:       conference.StatusProgressing,
				TranscribeID: uuid.FromStringOrNil("d00b7324-98c3-11ed-aa8f-0ff9d0b64c91"),
			},
			responseTranscribe: &tmtranscribe.Transcribe{
				ID: uuid.FromStringOrNil("d00b7324-98c3-11ed-aa8f-0ff9d0b64c91"),
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := conferenceHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().ConferenceGet(ctx, tt.id).Return(tt.responseConference, nil)
			mockReq.EXPECT().TranscribeV1TranscribeStop(ctx, tt.responseConference.TranscribeID).Return(tt.responseTranscribe, nil)
			mockDB.EXPECT().ConferenceSetTranscribeID(ctx, tt.id, uuid.Nil).Return(nil)
			mockDB.EXPECT().ConferenceGet(ctx, tt.id).Return(tt.responseConference, nil)

			res, err := h.TranscribeStop(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseConference, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseConference, res)
			}
		})
	}
}
