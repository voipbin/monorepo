package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"

	"gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/pkg/cachehandler"
)

func Test_ConferenceCreate(t *testing.T) {

	tests := []struct {
		name string

		conference *conference.Conference

		responseCurTime string

		expectRes *conference.Conference
	}{

		{
			"type conference",
			&conference.Conference{
				ID:         uuid.FromStringOrNil("26a42912-9163-11ea-93ca-bf5915635f88"),
				CustomerID: uuid.FromStringOrNil("361de3de-7f45-11ec-b641-5358ec38b5e2"),
				Type:       conference.TypeConference,
				Name:       "test type conference",
				Detail:     "test type conference detail",
			},

			"2023-01-03 21:35:02.809",
			&conference.Conference{
				ID:                uuid.FromStringOrNil("26a42912-9163-11ea-93ca-bf5915635f88"),
				CustomerID:        uuid.FromStringOrNil("361de3de-7f45-11ec-b641-5358ec38b5e2"),
				Type:              conference.TypeConference,
				Name:              "test type conference",
				Detail:            "test type conference detail",
				PreActions:        []fmaction.Action{},
				PostActions:       []fmaction.Action{},
				ConferencecallIDs: []uuid.UUID{},
				RecordingIDs:      []uuid.UUID{},
				TMEnd:             DefaultTimeStamp,
				TMCreate:          "2023-01-03 21:35:02.809",
				TMUpdate:          DefaultTimeStamp,
				TMDelete:          DefaultTimeStamp,
			},
		},
		{
			"added user ID",
			&conference.Conference{
				ID:         uuid.FromStringOrNil("132d3c9e-f08f-11ea-8ed9-6f27c201eff3"),
				CustomerID: uuid.FromStringOrNil("3fccf15e-7f45-11ec-bae1-7344ecec79bd"),
				Type:       conference.TypeConference,
				Name:       "test type conference",
				Detail:     "test type conference detail",
			},

			"2023-01-03 21:35:02.809",
			&conference.Conference{
				ID:                uuid.FromStringOrNil("132d3c9e-f08f-11ea-8ed9-6f27c201eff3"),
				CustomerID:        uuid.FromStringOrNil("3fccf15e-7f45-11ec-bae1-7344ecec79bd"),
				Type:              conference.TypeConference,
				Name:              "test type conference",
				Detail:            "test type conference detail",
				PreActions:        []fmaction.Action{},
				PostActions:       []fmaction.Action{},
				ConferencecallIDs: []uuid.UUID{},
				RecordingIDs:      []uuid.UUID{},
				TMEnd:             DefaultTimeStamp,
				TMCreate:          "2023-01-03 21:35:02.809",
				TMUpdate:          DefaultTimeStamp,
				TMDelete:          DefaultTimeStamp,
			},
		},
		{
			"added record id",
			&conference.Conference{
				ID:          uuid.FromStringOrNil("218aa220-2c19-11eb-905f-1b9d4d0da185"),
				CustomerID:  uuid.FromStringOrNil("48974ac8-7f45-11ec-9ba9-1f3e79c1ba60"),
				Type:        conference.TypeConference,
				Name:        "test type conference",
				Detail:      "test type conference detail",
				RecordingID: uuid.FromStringOrNil("37962c54-6122-11eb-a8b2-4ff0062b4c1b"),
			},

			"2023-01-03 21:35:02.809",
			&conference.Conference{
				ID:                uuid.FromStringOrNil("218aa220-2c19-11eb-905f-1b9d4d0da185"),
				CustomerID:        uuid.FromStringOrNil("48974ac8-7f45-11ec-9ba9-1f3e79c1ba60"),
				Type:              conference.TypeConference,
				Name:              "test type conference",
				Detail:            "test type conference detail",
				PreActions:        []fmaction.Action{},
				PostActions:       []fmaction.Action{},
				ConferencecallIDs: []uuid.UUID{},
				RecordingID:       uuid.FromStringOrNil("37962c54-6122-11eb-a8b2-4ff0062b4c1b"),
				RecordingIDs:      []uuid.UUID{},
				TMEnd:             DefaultTimeStamp,
				TMCreate:          "2023-01-03 21:35:02.809",
				TMUpdate:          DefaultTimeStamp,
				TMDelete:          DefaultTimeStamp,
			},
		},
		{
			"added record ids",
			&conference.Conference{
				ID:           uuid.FromStringOrNil("21d33d64-2c19-11eb-be7d-1ff9387bed0e"),
				CustomerID:   uuid.FromStringOrNil("50b72fca-7f45-11ec-8734-6749711ab0ed"),
				Type:         conference.TypeConference,
				Name:         "test type conference",
				Detail:       "test type conference detail",
				RecordingIDs: []uuid.UUID{uuid.FromStringOrNil("515f79ce-6122-11eb-b3ca-db50409503c4")},
			},

			"2023-01-03 21:35:02.809",
			&conference.Conference{
				ID:                uuid.FromStringOrNil("21d33d64-2c19-11eb-be7d-1ff9387bed0e"),
				CustomerID:        uuid.FromStringOrNil("50b72fca-7f45-11ec-8734-6749711ab0ed"),
				Type:              conference.TypeConference,
				Name:              "test type conference",
				Detail:            "test type conference detail",
				PreActions:        []fmaction.Action{},
				PostActions:       []fmaction.Action{},
				ConferencecallIDs: []uuid.UUID{},
				RecordingIDs:      []uuid.UUID{uuid.FromStringOrNil("515f79ce-6122-11eb-b3ca-db50409503c4")},
				TMEnd:             DefaultTimeStamp,
				TMCreate:          "2023-01-03 21:35:02.809",
				TMUpdate:          DefaultTimeStamp,
				TMDelete:          DefaultTimeStamp,
			},
		},
		{
			"set pre actions",
			&conference.Conference{
				ID:         uuid.FromStringOrNil("3824d0e4-3be7-11ec-b046-674e88e91f56"),
				CustomerID: uuid.FromStringOrNil("6342318a-7f45-11ec-b41d-af17676583fe"),
				Type:       conference.TypeConference,
				Name:       "test type conference",
				Detail:     "test type conference detail",
				PreActions: []fmaction.Action{
					{
						Type: "answer",
					},
				},
			},

			"2023-01-03 21:35:02.809",
			&conference.Conference{
				ID:         uuid.FromStringOrNil("3824d0e4-3be7-11ec-b046-674e88e91f56"),
				CustomerID: uuid.FromStringOrNil("6342318a-7f45-11ec-b41d-af17676583fe"),
				Type:       conference.TypeConference,
				Name:       "test type conference",
				Detail:     "test type conference detail",
				PreActions: []fmaction.Action{
					{
						Type: "answer",
					},
				},
				PostActions:       []fmaction.Action{},
				ConferencecallIDs: []uuid.UUID{},
				RecordingIDs:      []uuid.UUID{},
				TMEnd:             DefaultTimeStamp,
				TMCreate:          "2023-01-03 21:35:02.809",
				TMUpdate:          DefaultTimeStamp,
				TMDelete:          DefaultTimeStamp,
			},
		},
		{
			"set post actions",
			&conference.Conference{
				ID:         uuid.FromStringOrNil("e3d6c112-3bed-11ec-99be-5bc49af8efc2"),
				CustomerID: uuid.FromStringOrNil("6aa95b9c-7f45-11ec-b6fe-cbd78c2e625c"),
				Type:       conference.TypeConference,
				Name:       "test type conference",
				Detail:     "test type conference detail",
				PostActions: []fmaction.Action{
					{
						Type: "answer",
					},
				},
			},

			"2023-01-03 21:35:02.809",
			&conference.Conference{
				ID:         uuid.FromStringOrNil("e3d6c112-3bed-11ec-99be-5bc49af8efc2"),
				CustomerID: uuid.FromStringOrNil("6aa95b9c-7f45-11ec-b6fe-cbd78c2e625c"),
				Type:       conference.TypeConference,
				Name:       "test type conference",
				Detail:     "test type conference detail",
				PreActions: []fmaction.Action{},
				PostActions: []fmaction.Action{
					{
						Type: "answer",
					},
				},
				ConferencecallIDs: []uuid.UUID{},
				RecordingIDs:      []uuid.UUID{},
				TMEnd:             DefaultTimeStamp,
				TMCreate:          "2023-01-03 21:35:02.809",
				TMUpdate:          DefaultTimeStamp,
				TMDelete:          DefaultTimeStamp,
			},
		},
		{
			"set timeout",
			&conference.Conference{
				ID:         uuid.FromStringOrNil("05d6758c-3bee-11ec-a2f7-e78825f985e2"),
				CustomerID: uuid.FromStringOrNil("6b84f04e-7f45-11ec-be45-3f95176833cc"),
				Type:       conference.TypeConference,
				Name:       "test type conference",
				Detail:     "test type conference detail",
				Timeout:    86400,
			},

			"2023-01-03 21:35:02.809",
			&conference.Conference{
				ID:                uuid.FromStringOrNil("05d6758c-3bee-11ec-a2f7-e78825f985e2"),
				CustomerID:        uuid.FromStringOrNil("6b84f04e-7f45-11ec-be45-3f95176833cc"),
				Type:              conference.TypeConference,
				Name:              "test type conference",
				Detail:            "test type conference detail",
				Timeout:           86400,
				PreActions:        []fmaction.Action{},
				PostActions:       []fmaction.Action{},
				ConferencecallIDs: []uuid.UUID{},
				RecordingIDs:      []uuid.UUID{},
				TMEnd:             DefaultTimeStamp,
				TMCreate:          "2023-01-03 21:35:02.809",
				TMUpdate:          DefaultTimeStamp,
				TMDelete:          DefaultTimeStamp,
			},
		},
		{
			"having confbridge",
			&conference.Conference{
				ID:           uuid.FromStringOrNil("ba8a9474-0c2f-11ed-8002-37c00264d0b5"),
				CustomerID:   uuid.FromStringOrNil("6b84f04e-7f45-11ec-be45-3f95176833cc"),
				ConfbridgeID: uuid.FromStringOrNil("bc976166-0c2f-11ed-a0ff-6fe0610a390e"),
				Type:         conference.TypeConference,
				Name:         "test type conference",
				Detail:       "test type conference detail",
				Timeout:      86400,
			},

			"2023-01-03 21:35:02.809",
			&conference.Conference{
				ID:                uuid.FromStringOrNil("ba8a9474-0c2f-11ed-8002-37c00264d0b5"),
				CustomerID:        uuid.FromStringOrNil("6b84f04e-7f45-11ec-be45-3f95176833cc"),
				ConfbridgeID:      uuid.FromStringOrNil("bc976166-0c2f-11ed-a0ff-6fe0610a390e"),
				Type:              conference.TypeConference,
				Name:              "test type conference",
				Detail:            "test type conference detail",
				Timeout:           86400,
				PreActions:        []fmaction.Action{},
				PostActions:       []fmaction.Action{},
				ConferencecallIDs: []uuid.UUID{},
				RecordingIDs:      []uuid.UUID{},
				TMEnd:             DefaultTimeStamp,
				TMCreate:          "2023-01-03 21:35:02.809",
				TMUpdate:          DefaultTimeStamp,
				TMDelete:          DefaultTimeStamp,
			},
		},
		{
			"have conferencecall ids",
			&conference.Conference{
				ID:           uuid.FromStringOrNil("dc31f848-14d5-11ed-a65a-13e7fcbe84bb"),
				CustomerID:   uuid.FromStringOrNil("6b84f04e-7f45-11ec-be45-3f95176833cc"),
				ConfbridgeID: uuid.FromStringOrNil("bc976166-0c2f-11ed-a0ff-6fe0610a390e"),
				Type:         conference.TypeConference,
				Name:         "test type conference",
				Detail:       "test type conference detail",
				ConferencecallIDs: []uuid.UUID{
					uuid.FromStringOrNil("dc060896-14d5-11ed-8fb1-6fece6a05aa5"),
				},
				Timeout: 86400,
			},

			"2023-01-03 21:35:02.809",
			&conference.Conference{
				ID:           uuid.FromStringOrNil("dc31f848-14d5-11ed-a65a-13e7fcbe84bb"),
				CustomerID:   uuid.FromStringOrNil("6b84f04e-7f45-11ec-be45-3f95176833cc"),
				ConfbridgeID: uuid.FromStringOrNil("bc976166-0c2f-11ed-a0ff-6fe0610a390e"),
				Type:         conference.TypeConference,
				Name:         "test type conference",
				Detail:       "test type conference detail",
				Timeout:      86400,
				PreActions:   []fmaction.Action{},
				PostActions:  []fmaction.Action{},
				ConferencecallIDs: []uuid.UUID{
					uuid.FromStringOrNil("dc060896-14d5-11ed-8fb1-6fece6a05aa5"),
				},
				RecordingIDs: []uuid.UUID{},
				TMEnd:        DefaultTimeStamp,
				TMCreate:     "2023-01-03 21:35:02.809",
				TMUpdate:     DefaultTimeStamp,
				TMDelete:     DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}

			mockUtil.EXPECT().GetCurTime().Return(tt.responseCurTime)
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

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ConferenceGetByConfbridgeID(t *testing.T) {

	tests := []struct {
		name string

		conference *conference.Conference

		responseCurTime  string
		expectConference *conference.Conference
	}{
		{
			"normal",
			&conference.Conference{
				ID:           uuid.FromStringOrNil("1ac9f480-9861-11ec-8e29-c7820822026e"),
				CustomerID:   uuid.FromStringOrNil("1afc3ce2-9861-11ec-90b1-d76e949c3805"),
				ConfbridgeID: uuid.FromStringOrNil("1b280016-9861-11ec-999c-5f70848e711d"),
				Type:         conference.TypeConference,
				Name:         "test type conference",
				Detail:       "test type conference detail",
			},

			"2023-01-03 21:35:02.809",
			&conference.Conference{
				ID:                uuid.FromStringOrNil("1ac9f480-9861-11ec-8e29-c7820822026e"),
				CustomerID:        uuid.FromStringOrNil("1afc3ce2-9861-11ec-90b1-d76e949c3805"),
				ConfbridgeID:      uuid.FromStringOrNil("1b280016-9861-11ec-999c-5f70848e711d"),
				Type:              conference.TypeConference,
				Name:              "test type conference",
				Detail:            "test type conference detail",
				PreActions:        []fmaction.Action{},
				PostActions:       []fmaction.Action{},
				ConferencecallIDs: []uuid.UUID{},
				RecordingIDs:      []uuid.UUID{},
				TMEnd:             DefaultTimeStamp,
				TMCreate:          "2023-01-03 21:35:02.809",
				TMUpdate:          DefaultTimeStamp,
				TMDelete:          DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			mockUtil.EXPECT().GetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().ConferenceSet(ctx, gomock.Any())
			if err := h.ConferenceCreate(ctx, tt.conference); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.ConferenceGetByConfbridgeID(ctx, tt.conference.ConfbridgeID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectConference, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectConference, res)
			}
		})
	}
}

func Test_ConferenceSetRecordID(t *testing.T) {

	tests := []struct {
		name       string
		conference *conference.Conference
		recordID   uuid.UUID

		responseCurTime  string
		expectConference *conference.Conference
	}{
		{
			"test normal",
			&conference.Conference{
				ID: uuid.FromStringOrNil("2f7b0ee4-2834-11eb-9a6d-5beea5795ea6"),
			},
			uuid.FromStringOrNil("2fb4b446-2834-11eb-b864-1fdb13777d08"),

			"2023-01-03 21:35:02.809",
			&conference.Conference{
				ID:                uuid.FromStringOrNil("2f7b0ee4-2834-11eb-9a6d-5beea5795ea6"),
				PreActions:        []fmaction.Action{},
				PostActions:       []fmaction.Action{},
				ConferencecallIDs: []uuid.UUID{},
				RecordingID:       uuid.FromStringOrNil("2fb4b446-2834-11eb-b864-1fdb13777d08"),
				RecordingIDs:      []uuid.UUID{},
				TMEnd:             DefaultTimeStamp,
				TMCreate:          "2023-01-03 21:35:02.809",
				TMUpdate:          "2023-01-03 21:35:02.809",
				TMDelete:          DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}

			ctx := context.Background()

			mockUtil.EXPECT().GetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().ConferenceSet(ctx, gomock.Any())
			if err := h.ConferenceCreate(ctx, tt.conference); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().GetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().ConferenceSet(ctx, gomock.Any())
			if err := h.ConferenceSetRecordingID(ctx, tt.conference.ID, tt.recordID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ConferenceGet(ctx, tt.conference.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ConferenceSet(ctx, gomock.Any())
			res, err := h.ConferenceGet(ctx, tt.conference.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectConference, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectConference, res)
			}
		})
	}
}

func Test_ConferenceSetData(t *testing.T) {

	tests := []struct {
		name       string
		conference *conference.Conference

		data map[string]interface{}

		responseCurTime  string
		expectConference *conference.Conference
	}{
		{
			"test normal",
			&conference.Conference{
				ID: uuid.FromStringOrNil("0a64e234-675d-11eb-92c7-13f0c9a0e28b"),
			},

			map[string]interface{}{
				"key1": "string value",
			},

			"2023-01-03 21:35:02.809",
			&conference.Conference{
				ID:                uuid.FromStringOrNil("0a64e234-675d-11eb-92c7-13f0c9a0e28b"),
				PreActions:        []fmaction.Action{},
				PostActions:       []fmaction.Action{},
				ConferencecallIDs: []uuid.UUID{},
				RecordingIDs:      []uuid.UUID{},
				Data: map[string]interface{}{
					"key1": "string value",
				},
				TMEnd:    DefaultTimeStamp,
				TMCreate: "2023-01-03 21:35:02.809",
				TMUpdate: "2023-01-03 21:35:02.809",
				TMDelete: DefaultTimeStamp,
			},
		},
		{
			"update 2 datas",
			&conference.Conference{
				ID: uuid.FromStringOrNil("d54bf5b4-675d-11eb-b133-9b06996a9b99"),
			},
			map[string]interface{}{
				"key1": "string value",
				"key2": "string value",
			},

			"2023-01-03 21:35:02.809",
			&conference.Conference{
				ID:                uuid.FromStringOrNil("d54bf5b4-675d-11eb-b133-9b06996a9b99"),
				PreActions:        []fmaction.Action{},
				PostActions:       []fmaction.Action{},
				ConferencecallIDs: []uuid.UUID{},
				RecordingIDs:      []uuid.UUID{},
				Data: map[string]interface{}{
					"key1": "string value",
					"key2": "string value",
				},
				TMEnd:    DefaultTimeStamp,
				TMCreate: "2023-01-03 21:35:02.809",
				TMUpdate: "2023-01-03 21:35:02.809",
				TMDelete: DefaultTimeStamp,
			},
		},
		{
			"update mixed data types",
			&conference.Conference{
				ID: uuid.FromStringOrNil("efa1ec2a-675d-11eb-b854-ffe06d0fc488"),
			},
			map[string]interface{}{
				"key1": "string value",
				"key2": 123,
			},

			"2023-01-03 21:35:02.809",
			&conference.Conference{
				ID:                uuid.FromStringOrNil("efa1ec2a-675d-11eb-b854-ffe06d0fc488"),
				PreActions:        []fmaction.Action{},
				PostActions:       []fmaction.Action{},
				ConferencecallIDs: []uuid.UUID{},
				RecordingIDs:      []uuid.UUID{},
				Data: map[string]interface{}{
					"key1": "string value",
					"key2": float64(123),
				},
				TMEnd:    DefaultTimeStamp,
				TMCreate: "2023-01-03 21:35:02.809",
				TMUpdate: "2023-01-03 21:35:02.809",
				TMDelete: DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}

			ctx := context.Background()

			mockUtil.EXPECT().GetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().ConferenceSet(ctx, gomock.Any())
			if err := h.ConferenceCreate(ctx, tt.conference); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().GetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().ConferenceSet(ctx, gomock.Any())
			if err := h.ConferenceSetData(ctx, tt.conference.ID, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ConferenceGet(ctx, tt.conference.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ConferenceSet(ctx, gomock.Any())
			res, err := h.ConferenceGet(ctx, tt.conference.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectConference, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectConference, res)
			}
		})
	}
}

func Test_ConferenceGetsWithType(t *testing.T) {
	tests := []struct {
		name        string
		conferences []*conference.Conference

		customerID     uuid.UUID
		conferenceType conference.Type

		responseCurTime string

		expectRes []*conference.Conference
	}{
		{
			"normal",
			[]*conference.Conference{
				{
					ID:         uuid.FromStringOrNil("418ea85a-94b8-11ed-9cf4-5f71d1d56a86"),
					CustomerID: uuid.FromStringOrNil("80a965e0-7f45-11ec-a078-7f296665fa3d"),
					Type:       conference.TypeConference,
				},
				{
					ID:         uuid.FromStringOrNil("4b0feace-94b8-11ed-a8a7-f3ffb3124f95"),
					CustomerID: uuid.FromStringOrNil("80a965e0-7f45-11ec-a078-7f296665fa3d"),
					Type:       conference.TypeConference,
				},
				{
					ID:         uuid.FromStringOrNil("7dec4d52-94b8-11ed-9d79-ff4a5e22e54f"),
					CustomerID: uuid.FromStringOrNil("80a965e0-7f45-11ec-a078-7f296665fa3d"),
					Type:       conference.TypeConnect,
				},
			},

			uuid.FromStringOrNil("80a965e0-7f45-11ec-a078-7f296665fa3d"),
			conference.TypeConference,

			"2023-01-03 21:35:02.809",

			[]*conference.Conference{
				{
					ID:                uuid.FromStringOrNil("418ea85a-94b8-11ed-9cf4-5f71d1d56a86"),
					CustomerID:        uuid.FromStringOrNil("80a965e0-7f45-11ec-a078-7f296665fa3d"),
					Type:              conference.TypeConference,
					PreActions:        []fmaction.Action{},
					PostActions:       []fmaction.Action{},
					ConferencecallIDs: []uuid.UUID{},
					RecordingIDs:      []uuid.UUID{},
					TMEnd:             DefaultTimeStamp,
					TMCreate:          "2023-01-03 21:35:02.809",
					TMUpdate:          DefaultTimeStamp,
					TMDelete:          DefaultTimeStamp,
				},
				{
					ID:                uuid.FromStringOrNil("4b0feace-94b8-11ed-a8a7-f3ffb3124f95"),
					CustomerID:        uuid.FromStringOrNil("80a965e0-7f45-11ec-a078-7f296665fa3d"),
					Type:              conference.TypeConference,
					PreActions:        []fmaction.Action{},
					PostActions:       []fmaction.Action{},
					ConferencecallIDs: []uuid.UUID{},
					RecordingIDs:      []uuid.UUID{},
					TMEnd:             DefaultTimeStamp,
					TMCreate:          "2023-01-03 21:35:02.809",
					TMUpdate:          DefaultTimeStamp,
					TMDelete:          DefaultTimeStamp,
				},
			},
		},
		{
			"empty",
			[]*conference.Conference{},

			uuid.FromStringOrNil("2060b9be-94c9-11ed-8e3e-2363706d4c8a"),
			conference.TypeConference,

			"2023-01-03 21:35:02.809",
			[]*conference.Conference{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}

			ctx := context.Background()

			for _, cf := range tt.conferences {
				mockUtil.EXPECT().GetCurTime().Return(tt.responseCurTime)
				mockCache.EXPECT().ConferenceSet(ctx, gomock.Any())
				if errCreate := h.ConferenceCreate(ctx, cf); errCreate != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", errCreate)
				}
			}

			res, err := h.ConferenceGetsWithType(ctx, tt.customerID, tt.conferenceType, 10, utilhandler.GetCurTime())
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ConferenceGets(t *testing.T) {

	tests := []struct {
		name        string
		conferences []*conference.Conference

		customerID uuid.UUID
		count      int

		responseCurTime string
		expectRes       []*conference.Conference
	}{
		{
			"normal",
			[]*conference.Conference{
				{
					ID:         uuid.FromStringOrNil("ac54ebd4-94c9-11ed-b4aa-4f7da8f9741a"),
					CustomerID: uuid.FromStringOrNil("91f25410-7f45-11ec-97d1-8b4f8cee4768"),
				},
				{
					ID:         uuid.FromStringOrNil("ad76ec88-94c9-11ed-9651-df2f9c2178aa"),
					CustomerID: uuid.FromStringOrNil("91f25410-7f45-11ec-97d1-8b4f8cee4768"),
				},
			},

			uuid.FromStringOrNil("91f25410-7f45-11ec-97d1-8b4f8cee4768"),
			10,
			"2023-01-03 21:35:02.809",
			[]*conference.Conference{
				{
					ID:                uuid.FromStringOrNil("ac54ebd4-94c9-11ed-b4aa-4f7da8f9741a"),
					CustomerID:        uuid.FromStringOrNil("91f25410-7f45-11ec-97d1-8b4f8cee4768"),
					PreActions:        []fmaction.Action{},
					PostActions:       []fmaction.Action{},
					ConferencecallIDs: []uuid.UUID{},
					RecordingIDs:      []uuid.UUID{},
					TMEnd:             DefaultTimeStamp,
					TMCreate:          "2023-01-03 21:35:02.809",
					TMUpdate:          DefaultTimeStamp,
					TMDelete:          DefaultTimeStamp,
				},
				{
					ID:                uuid.FromStringOrNil("ad76ec88-94c9-11ed-9651-df2f9c2178aa"),
					CustomerID:        uuid.FromStringOrNil("91f25410-7f45-11ec-97d1-8b4f8cee4768"),
					PreActions:        []fmaction.Action{},
					PostActions:       []fmaction.Action{},
					ConferencecallIDs: []uuid.UUID{},
					RecordingIDs:      []uuid.UUID{},
					TMEnd:             DefaultTimeStamp,
					TMCreate:          "2023-01-03 21:35:02.809",
					TMUpdate:          DefaultTimeStamp,
					TMDelete:          DefaultTimeStamp,
				},
			},
		},
		{
			"empty",
			[]*conference.Conference{},

			uuid.FromStringOrNil("b31d32ae-7f45-11ec-82c6-936e22306376"),
			0,
			"2023-01-03 21:35:02.809",
			[]*conference.Conference{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}

			ctx := context.Background()

			for _, cf := range tt.conferences {
				mockUtil.EXPECT().GetCurTime().Return(tt.responseCurTime)
				mockCache.EXPECT().ConferenceSet(ctx, gomock.Any())
				if errCreate := h.ConferenceCreate(ctx, cf); errCreate != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", errCreate)
				}
			}

			res, err := h.ConferenceGets(ctx, tt.customerID, 10, utilhandler.GetCurTime())
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func Test_ConferenceEnd(t *testing.T) {

	tests := []struct {
		name       string
		conference *conference.Conference

		id uuid.UUID

		responseCurTime string
		expectRes       *conference.Conference
	}{
		{
			"normal",
			&conference.Conference{
				ID: uuid.FromStringOrNil("722c7822-94ca-11ed-b0a9-ef969fc8348d"),
			},

			uuid.FromStringOrNil("722c7822-94ca-11ed-b0a9-ef969fc8348d"),

			"2023-01-03 21:35:02.809",
			&conference.Conference{
				ID:                uuid.FromStringOrNil("722c7822-94ca-11ed-b0a9-ef969fc8348d"),
				Status:            conference.StatusTerminated,
				PreActions:        []fmaction.Action{},
				PostActions:       []fmaction.Action{},
				ConferencecallIDs: []uuid.UUID{},
				RecordingIDs:      []uuid.UUID{},
				TMEnd:             "2023-01-03 21:35:02.809",
				TMCreate:          "2023-01-03 21:35:02.809",
				TMUpdate:          "2023-01-03 21:35:02.809",
				TMDelete:          DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}

			ctx := context.Background()

			mockUtil.EXPECT().GetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().ConferenceSet(ctx, gomock.Any())
			if err := h.ConferenceCreate(ctx, tt.conference); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().GetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().ConferenceSet(ctx, gomock.Any())
			if errDel := h.ConferenceEnd(ctx, tt.id); errDel != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errDel)
			}

			mockCache.EXPECT().ConferenceGet(ctx, tt.id).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ConferenceSet(ctx, gomock.Any())
			res, err := h.ConferenceGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func Test_ConferenceDelete(t *testing.T) {

	tests := []struct {
		name       string
		conference *conference.Conference

		id uuid.UUID

		responseCurTime string
		expectRes       *conference.Conference
	}{
		{
			"normal",
			&conference.Conference{
				ID: uuid.FromStringOrNil("7a23bfa0-94e2-11ed-8dd9-0b374780e823"),
			},

			uuid.FromStringOrNil("7a23bfa0-94e2-11ed-8dd9-0b374780e823"),

			"2023-01-03 21:35:02.809",
			&conference.Conference{
				ID:                uuid.FromStringOrNil("7a23bfa0-94e2-11ed-8dd9-0b374780e823"),
				PreActions:        []fmaction.Action{},
				PostActions:       []fmaction.Action{},
				ConferencecallIDs: []uuid.UUID{},
				RecordingIDs:      []uuid.UUID{},
				TMEnd:             DefaultTimeStamp,
				TMCreate:          "2023-01-03 21:35:02.809",
				TMUpdate:          "2023-01-03 21:35:02.809",
				TMDelete:          "2023-01-03 21:35:02.809",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}

			ctx := context.Background()

			mockUtil.EXPECT().GetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().ConferenceSet(ctx, gomock.Any())
			if err := h.ConferenceCreate(ctx, tt.conference); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().GetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().ConferenceSet(ctx, gomock.Any())
			if errDel := h.ConferenceDelete(ctx, tt.id); errDel != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errDel)
			}

			mockCache.EXPECT().ConferenceGet(ctx, tt.id).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ConferenceSet(ctx, gomock.Any())
			res, err := h.ConferenceGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}
