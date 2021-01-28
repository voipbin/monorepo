package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/conferencehandler/models/conference"
)

func TestConferenceCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	type test struct {
		name string

		conference       *conference.Conference
		expectConference *conference.Conference
	}

	tests := []test{
		{
			"type conference",
			&conference.Conference{
				ID:     uuid.FromStringOrNil("26a42912-9163-11ea-93ca-bf5915635f88"),
				Type:   conference.TypeConference,
				Name:   "test type conference",
				Detail: "test type conference detail",
			},
			&conference.Conference{
				ID:           uuid.FromStringOrNil("26a42912-9163-11ea-93ca-bf5915635f88"),
				Type:         conference.TypeConference,
				Name:         "test type conference",
				Detail:       "test type conference detail",
				CallIDs:      []uuid.UUID{},
				RecordingIDs: []uuid.UUID{},
			},
		},
		{
			"added user ID",
			&conference.Conference{
				ID:     uuid.FromStringOrNil("132d3c9e-f08f-11ea-8ed9-6f27c201eff3"),
				UserID: 1,
				Type:   conference.TypeConference,
				Name:   "test type conference",
				Detail: "test type conference detail",
			},
			&conference.Conference{
				ID:           uuid.FromStringOrNil("132d3c9e-f08f-11ea-8ed9-6f27c201eff3"),
				UserID:       1,
				Type:         conference.TypeConference,
				Name:         "test type conference",
				Detail:       "test type conference detail",
				CallIDs:      []uuid.UUID{},
				RecordingIDs: []uuid.UUID{},
			},
		},
		{
			"added record id",
			&conference.Conference{
				ID:          uuid.FromStringOrNil("218aa220-2c19-11eb-905f-1b9d4d0da185"),
				UserID:      1,
				Type:        conference.TypeConference,
				Name:        "test type conference",
				Detail:      "test type conference detail",
				RecordingID: uuid.FromStringOrNil("37962c54-6122-11eb-a8b2-4ff0062b4c1b"),
			},
			&conference.Conference{
				ID:           uuid.FromStringOrNil("218aa220-2c19-11eb-905f-1b9d4d0da185"),
				UserID:       1,
				Type:         conference.TypeConference,
				Name:         "test type conference",
				Detail:       "test type conference detail",
				CallIDs:      []uuid.UUID{},
				RecordingID:  uuid.FromStringOrNil("37962c54-6122-11eb-a8b2-4ff0062b4c1b"),
				RecordingIDs: []uuid.UUID{},
			},
		},
		{
			"added record ids",
			&conference.Conference{
				ID:           uuid.FromStringOrNil("21d33d64-2c19-11eb-be7d-1ff9387bed0e"),
				UserID:       1,
				Type:         conference.TypeConference,
				Name:         "test type conference",
				Detail:       "test type conference detail",
				RecordingIDs: []uuid.UUID{uuid.FromStringOrNil("515f79ce-6122-11eb-b3ca-db50409503c4")},
			},
			&conference.Conference{
				ID:           uuid.FromStringOrNil("21d33d64-2c19-11eb-be7d-1ff9387bed0e"),
				UserID:       1,
				Type:         conference.TypeConference,
				Name:         "test type conference",
				Detail:       "test type conference detail",
				CallIDs:      []uuid.UUID{},
				RecordingIDs: []uuid.UUID{uuid.FromStringOrNil("515f79ce-6122-11eb-b3ca-db50409503c4")},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest, mockCache)

			mockCache.EXPECT().ConferenceSet(gomock.Any(), gomock.Any())
			if err := h.ConferenceCreate(context.Background(), tt.conference); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ConferenceGet(gomock.Any(), tt.conference.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ConferenceSet(gomock.Any(), gomock.Any())
			res, err := h.ConferenceGet(context.Background(), tt.conference.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res.TMCreate = ""
			if reflect.DeepEqual(tt.expectConference, res) == false {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectConference, res)
			}
		})
	}
}

func TestConferenceSetRecordID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	type test struct {
		name             string
		conference       *conference.Conference
		recordID         uuid.UUID
		expectConference *conference.Conference
	}

	tests := []test{
		{
			"test normal",
			&conference.Conference{
				ID: uuid.FromStringOrNil("2f7b0ee4-2834-11eb-9a6d-5beea5795ea6"),
			},
			uuid.FromStringOrNil("2fb4b446-2834-11eb-b864-1fdb13777d08"),
			&conference.Conference{
				ID:           uuid.FromStringOrNil("2f7b0ee4-2834-11eb-9a6d-5beea5795ea6"),
				CallIDs:      []uuid.UUID{},
				RecordingID:  uuid.FromStringOrNil("2fb4b446-2834-11eb-b864-1fdb13777d08"),
				RecordingIDs: []uuid.UUID{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest, mockCache)

			mockCache.EXPECT().ConferenceSet(gomock.Any(), gomock.Any())
			if err := h.ConferenceCreate(context.Background(), tt.conference); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ConferenceSet(gomock.Any(), gomock.Any())
			if err := h.ConferenceSetRecordID(context.Background(), tt.conference.ID, tt.recordID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ConferenceGet(gomock.Any(), tt.conference.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ConferenceSet(gomock.Any(), gomock.Any())
			res, err := h.ConferenceGet(context.Background(), tt.conference.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res.TMUpdate = ""
			res.TMCreate = ""
			if reflect.DeepEqual(tt.expectConference, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectConference, res)
			}
		})
	}
}
