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
			"have all",
			&conference.Conference{
				ID:           uuid.FromStringOrNil("26a42912-9163-11ea-93ca-bf5915635f88"),
				CustomerID:   uuid.FromStringOrNil("361de3de-7f45-11ec-b641-5358ec38b5e2"),
				ConfbridgeID: uuid.FromStringOrNil("6a8ea5c6-98b8-11ed-87e2-33728f9ec79e"),
				FlowID:       uuid.FromStringOrNil("6ad5c136-98b8-11ed-84cb-07e86f64e72c"),
				Type:         conference.TypeConference,
				Status:       conference.StatusProgressing,
				Name:         "test type conference",
				Detail:       "test type conference detail",
				Data: map[string]interface{}{
					"key1": "string value",
					"key2": "string value",
				},
				Timeout: 100,
				PreActions: []fmaction.Action{
					{
						Type: fmaction.TypeAnswer,
					},
				},
				PostActions: []fmaction.Action{
					{
						Type: fmaction.TypeHangup,
					},
				},
				ConferencecallIDs: []uuid.UUID{
					uuid.FromStringOrNil("6b00d1c8-98b8-11ed-86cf-13921488e8d1"),
					uuid.FromStringOrNil("6b28b4d6-98b8-11ed-a21b-7fb97887cf8c"),
				},
				RecordingID: uuid.FromStringOrNil("6b556288-98b8-11ed-b567-23ed7bb9222b"),
				RecordingIDs: []uuid.UUID{
					uuid.FromStringOrNil("b197268c-98b8-11ed-964e-779dfe986c16"),
					uuid.FromStringOrNil("b1bd7eea-98b8-11ed-af70-23caf714c591"),
				},
				TranscribeID: uuid.FromStringOrNil("b1e3cf6e-98b8-11ed-9638-8f594b2cc533"),
				TranscribeIDs: []uuid.UUID{
					uuid.FromStringOrNil("b20ff346-98b8-11ed-9fa7-f74b9cee92b5"),
					uuid.FromStringOrNil("b238edc8-98b8-11ed-9ce4-73af14c3e8ff"),
				},
			},

			"2023-01-03 21:35:02.809",
			&conference.Conference{
				ID:           uuid.FromStringOrNil("26a42912-9163-11ea-93ca-bf5915635f88"),
				CustomerID:   uuid.FromStringOrNil("361de3de-7f45-11ec-b641-5358ec38b5e2"),
				ConfbridgeID: uuid.FromStringOrNil("6a8ea5c6-98b8-11ed-87e2-33728f9ec79e"),
				FlowID:       uuid.FromStringOrNil("6ad5c136-98b8-11ed-84cb-07e86f64e72c"),
				Type:         conference.TypeConference,
				Status:       conference.StatusProgressing,
				Name:         "test type conference",
				Detail:       "test type conference detail",
				Data: map[string]interface{}{
					"key1": "string value",
					"key2": "string value",
				},
				Timeout: 100,
				PreActions: []fmaction.Action{
					{
						Type: fmaction.TypeAnswer,
					},
				},
				PostActions: []fmaction.Action{
					{
						Type: fmaction.TypeHangup,
					},
				},
				ConferencecallIDs: []uuid.UUID{
					uuid.FromStringOrNil("6b00d1c8-98b8-11ed-86cf-13921488e8d1"),
					uuid.FromStringOrNil("6b28b4d6-98b8-11ed-a21b-7fb97887cf8c"),
				},
				RecordingID: uuid.FromStringOrNil("6b556288-98b8-11ed-b567-23ed7bb9222b"),
				RecordingIDs: []uuid.UUID{
					uuid.FromStringOrNil("b197268c-98b8-11ed-964e-779dfe986c16"),
					uuid.FromStringOrNil("b1bd7eea-98b8-11ed-af70-23caf714c591"),
				},
				TranscribeID: uuid.FromStringOrNil("b1e3cf6e-98b8-11ed-9638-8f594b2cc533"),
				TranscribeIDs: []uuid.UUID{
					uuid.FromStringOrNil("b20ff346-98b8-11ed-9fa7-f74b9cee92b5"),
					uuid.FromStringOrNil("b238edc8-98b8-11ed-9ce4-73af14c3e8ff"),
				},
				TMEnd:    DefaultTimeStamp,
				TMCreate: "2023-01-03 21:35:02.809",
				TMUpdate: DefaultTimeStamp,
				TMDelete: DefaultTimeStamp,
			},
		},
		{
			"empty",
			&conference.Conference{
				ID: uuid.FromStringOrNil("a9f69592-98b9-11ed-947e-0f7ac40639b6"),
			},

			"2023-01-03 21:35:02.809",
			&conference.Conference{
				ID:                uuid.FromStringOrNil("a9f69592-98b9-11ed-947e-0f7ac40639b6"),
				Data:              map[string]interface{}{},
				PreActions:        []fmaction.Action{},
				PostActions:       []fmaction.Action{},
				ConferencecallIDs: []uuid.UUID{},
				RecordingIDs:      []uuid.UUID{},
				TranscribeIDs:     []uuid.UUID{},
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

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().ConferenceSet(ctx, gomock.Any())
			if err := h.ConferenceCreate(ctx, tt.conference); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ConferenceGet(ctx, tt.conference.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ConferenceSet(ctx, gomock.Any())
			res, err := h.ConferenceGet(ctx, tt.conference.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
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
				Data:              map[string]interface{}{},
				PreActions:        []fmaction.Action{},
				PostActions:       []fmaction.Action{},
				ConferencecallIDs: []uuid.UUID{},
				RecordingIDs:      []uuid.UUID{},
				TranscribeIDs:     []uuid.UUID{},
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

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
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

func Test_ConferenceSetRecordingID(t *testing.T) {

	tests := []struct {
		name        string
		conference  *conference.Conference
		recordingID uuid.UUID

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
				Data:              map[string]interface{}{},
				PreActions:        []fmaction.Action{},
				PostActions:       []fmaction.Action{},
				ConferencecallIDs: []uuid.UUID{},
				RecordingID:       uuid.FromStringOrNil("2fb4b446-2834-11eb-b864-1fdb13777d08"),
				RecordingIDs:      []uuid.UUID{},
				TranscribeIDs:     []uuid.UUID{},
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

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().ConferenceSet(ctx, gomock.Any())
			if err := h.ConferenceCreate(ctx, tt.conference); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().ConferenceSet(ctx, gomock.Any())
			if err := h.ConferenceSetRecordingID(ctx, tt.conference.ID, tt.recordingID); err != nil {
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
				ID: uuid.FromStringOrNil("0a64e234-675d-11eb-92c7-13f0c9a0e28b"),
				Data: map[string]interface{}{
					"key1": "string value",
				},
				PreActions:        []fmaction.Action{},
				PostActions:       []fmaction.Action{},
				ConferencecallIDs: []uuid.UUID{},
				RecordingIDs:      []uuid.UUID{},
				TranscribeIDs:     []uuid.UUID{},
				TMEnd:             DefaultTimeStamp,
				TMCreate:          "2023-01-03 21:35:02.809",
				TMUpdate:          "2023-01-03 21:35:02.809",
				TMDelete:          DefaultTimeStamp,
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
				ID: uuid.FromStringOrNil("d54bf5b4-675d-11eb-b133-9b06996a9b99"),
				Data: map[string]interface{}{
					"key1": "string value",
					"key2": "string value",
				},
				PreActions:        []fmaction.Action{},
				PostActions:       []fmaction.Action{},
				ConferencecallIDs: []uuid.UUID{},
				RecordingIDs:      []uuid.UUID{},
				TranscribeIDs:     []uuid.UUID{},
				TMEnd:             DefaultTimeStamp,
				TMCreate:          "2023-01-03 21:35:02.809",
				TMUpdate:          "2023-01-03 21:35:02.809",
				TMDelete:          DefaultTimeStamp,
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
				ID: uuid.FromStringOrNil("efa1ec2a-675d-11eb-b854-ffe06d0fc488"),
				Data: map[string]interface{}{
					"key1": "string value",
					"key2": float64(123),
				},
				PreActions:        []fmaction.Action{},
				PostActions:       []fmaction.Action{},
				ConferencecallIDs: []uuid.UUID{},
				RecordingIDs:      []uuid.UUID{},
				TranscribeIDs:     []uuid.UUID{},
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

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().ConferenceSet(ctx, gomock.Any())
			if err := h.ConferenceCreate(ctx, tt.conference); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
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

func Test_ConferenceGets(t *testing.T) {

	tests := []struct {
		name        string
		conferences []*conference.Conference

		count   int
		filters map[string]string

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

			10,
			map[string]string{
				"customer_id": "91f25410-7f45-11ec-97d1-8b4f8cee4768",
				"deleted":     "false",
			},

			"2023-01-03 21:35:02.809",
			[]*conference.Conference{
				{
					ID:                uuid.FromStringOrNil("ac54ebd4-94c9-11ed-b4aa-4f7da8f9741a"),
					CustomerID:        uuid.FromStringOrNil("91f25410-7f45-11ec-97d1-8b4f8cee4768"),
					Data:              map[string]interface{}{},
					PreActions:        []fmaction.Action{},
					PostActions:       []fmaction.Action{},
					ConferencecallIDs: []uuid.UUID{},
					RecordingIDs:      []uuid.UUID{},
					TranscribeIDs:     []uuid.UUID{},
					TMEnd:             DefaultTimeStamp,
					TMCreate:          "2023-01-03 21:35:02.809",
					TMUpdate:          DefaultTimeStamp,
					TMDelete:          DefaultTimeStamp,
				},
				{
					ID:                uuid.FromStringOrNil("ad76ec88-94c9-11ed-9651-df2f9c2178aa"),
					CustomerID:        uuid.FromStringOrNil("91f25410-7f45-11ec-97d1-8b4f8cee4768"),
					Data:              map[string]interface{}{},
					PreActions:        []fmaction.Action{},
					PostActions:       []fmaction.Action{},
					ConferencecallIDs: []uuid.UUID{},
					RecordingIDs:      []uuid.UUID{},
					TranscribeIDs:     []uuid.UUID{},
					TMEnd:             DefaultTimeStamp,
					TMCreate:          "2023-01-03 21:35:02.809",
					TMUpdate:          DefaultTimeStamp,
					TMDelete:          DefaultTimeStamp,
				},
			},
		},

		{
			"gets conference type only",
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

			10,
			map[string]string{
				"customer_id": "80a965e0-7f45-11ec-a078-7f296665fa3d",
				"deleted":     "false",
				"type":        string(conference.TypeConference),
			},

			"2023-01-03 21:35:02.809",
			[]*conference.Conference{
				{
					ID:                uuid.FromStringOrNil("418ea85a-94b8-11ed-9cf4-5f71d1d56a86"),
					CustomerID:        uuid.FromStringOrNil("80a965e0-7f45-11ec-a078-7f296665fa3d"),
					Type:              conference.TypeConference,
					Data:              map[string]interface{}{},
					PreActions:        []fmaction.Action{},
					PostActions:       []fmaction.Action{},
					ConferencecallIDs: []uuid.UUID{},
					RecordingIDs:      []uuid.UUID{},
					TranscribeIDs:     []uuid.UUID{},
					TMEnd:             DefaultTimeStamp,
					TMCreate:          "2023-01-03 21:35:02.809",
					TMUpdate:          DefaultTimeStamp,
					TMDelete:          DefaultTimeStamp,
				},
				{
					ID:                uuid.FromStringOrNil("4b0feace-94b8-11ed-a8a7-f3ffb3124f95"),
					CustomerID:        uuid.FromStringOrNil("80a965e0-7f45-11ec-a078-7f296665fa3d"),
					Type:              conference.TypeConference,
					Data:              map[string]interface{}{},
					PreActions:        []fmaction.Action{},
					PostActions:       []fmaction.Action{},
					ConferencecallIDs: []uuid.UUID{},
					RecordingIDs:      []uuid.UUID{},
					TranscribeIDs:     []uuid.UUID{},
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

			0,
			map[string]string{
				"customer_id": "3f84e9f4-ed84-11ee-9bfb-2bce0d221d0b",
			},

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
				mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
				mockCache.EXPECT().ConferenceSet(ctx, gomock.Any())
				if errCreate := h.ConferenceCreate(ctx, cf); errCreate != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", errCreate)
				}
			}

			res, err := h.ConferenceGets(ctx, 10, utilhandler.TimeGetCurTime(), tt.filters)
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
				Data:              map[string]interface{}{},
				PreActions:        []fmaction.Action{},
				PostActions:       []fmaction.Action{},
				ConferencecallIDs: []uuid.UUID{},
				RecordingIDs:      []uuid.UUID{},
				TranscribeIDs:     []uuid.UUID{},
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

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().ConferenceSet(ctx, gomock.Any())
			if err := h.ConferenceCreate(ctx, tt.conference); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
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
				Data:              map[string]interface{}{},
				PreActions:        []fmaction.Action{},
				PostActions:       []fmaction.Action{},
				ConferencecallIDs: []uuid.UUID{},
				RecordingIDs:      []uuid.UUID{},
				TranscribeIDs:     []uuid.UUID{},
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

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().ConferenceSet(ctx, gomock.Any())
			if err := h.ConferenceCreate(ctx, tt.conference); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
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

func Test_ConferenceSetTranscribeID(t *testing.T) {

	tests := []struct {
		name         string
		conference   *conference.Conference
		transcribeID uuid.UUID

		responseCurTime  string
		expectConference *conference.Conference
	}{
		{
			"normal",
			&conference.Conference{
				ID: uuid.FromStringOrNil("000ca104-98c1-11ed-bde2-9badb79a7365"),
			},
			uuid.FromStringOrNil("003eb216-98c1-11ed-9789-ff71dbeab66e"),

			"2023-01-03 21:35:02.809",
			&conference.Conference{
				ID:                uuid.FromStringOrNil("000ca104-98c1-11ed-bde2-9badb79a7365"),
				Data:              map[string]interface{}{},
				PreActions:        []fmaction.Action{},
				PostActions:       []fmaction.Action{},
				ConferencecallIDs: []uuid.UUID{},
				RecordingIDs:      []uuid.UUID{},
				TranscribeID:      uuid.FromStringOrNil("003eb216-98c1-11ed-9789-ff71dbeab66e"),
				TranscribeIDs:     []uuid.UUID{},
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

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().ConferenceSet(ctx, gomock.Any())
			if err := h.ConferenceCreate(ctx, tt.conference); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().ConferenceSet(ctx, gomock.Any())
			if err := h.ConferenceSetTranscribeID(ctx, tt.conference.ID, tt.transcribeID); err != nil {
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
