package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-conference-manager/models/conference"
	"monorepo/bin-conference-manager/pkg/cachehandler"
)

func Test_ConferenceCreate(t *testing.T) {

	curTime := time.Date(2023, 1, 3, 21, 35, 2, 809000000, time.UTC)

	tests := []struct {
		name string

		conference *conference.Conference

		responseCurTime *time.Time

		expectRes *conference.Conference
	}{
		{
			"have all",
			&conference.Conference{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("26a42912-9163-11ea-93ca-bf5915635f88"),
					CustomerID: uuid.FromStringOrNil("361de3de-7f45-11ec-b641-5358ec38b5e2"),
				},
				ConfbridgeID: uuid.FromStringOrNil("6a8ea5c6-98b8-11ed-87e2-33728f9ec79e"),
				Type:         conference.TypeConference,
				Status:       conference.StatusProgressing,
				Name:         "test type conference",
				Detail:       "test type conference detail",
				Data: map[string]interface{}{
					"key1": "string value",
					"key2": "string value",
				},
				Timeout:    100,
				PreFlowID:  uuid.FromStringOrNil("83894532-1e07-11f0-a8d1-1fa5e177dd4b"),
				PostFlowID: uuid.FromStringOrNil("83b9adda-1e07-11f0-a265-37149ee71b54"),
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

			&curTime,
			&conference.Conference{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("26a42912-9163-11ea-93ca-bf5915635f88"),
					CustomerID: uuid.FromStringOrNil("361de3de-7f45-11ec-b641-5358ec38b5e2"),
				},
				ConfbridgeID: uuid.FromStringOrNil("6a8ea5c6-98b8-11ed-87e2-33728f9ec79e"),
				Type:         conference.TypeConference,
				Status:       conference.StatusProgressing,
				Name:         "test type conference",
				Detail:       "test type conference detail",
				Data: map[string]interface{}{
					"key1": "string value",
					"key2": "string value",
				},
				Timeout:    100,
				PreFlowID:  uuid.FromStringOrNil("83894532-1e07-11f0-a8d1-1fa5e177dd4b"),
				PostFlowID: uuid.FromStringOrNil("83b9adda-1e07-11f0-a265-37149ee71b54"),
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
				TMEnd:    nil,
				TMCreate: &curTime,
				TMUpdate: nil,
				TMDelete: nil,
			},
		},
		{
			"empty",
			&conference.Conference{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a9f69592-98b9-11ed-947e-0f7ac40639b6"),
				},
			},

			&curTime,
			&conference.Conference{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a9f69592-98b9-11ed-947e-0f7ac40639b6"),
				},
				Data:              map[string]interface{}{},
				ConferencecallIDs: []uuid.UUID{},
				RecordingIDs:      []uuid.UUID{},
				TranscribeIDs:     []uuid.UUID{},
				TMEnd:             nil,
				TMCreate:          &curTime,
				TMUpdate:          nil,
				TMDelete:          nil,
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

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
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

	curTime := time.Date(2023, 1, 3, 21, 35, 2, 809000000, time.UTC)

	tests := []struct {
		name string

		conference *conference.Conference

		responseCurTime  *time.Time
		expectConference *conference.Conference
	}{
		{
			"normal",
			&conference.Conference{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("1ac9f480-9861-11ec-8e29-c7820822026e"),
					CustomerID: uuid.FromStringOrNil("1afc3ce2-9861-11ec-90b1-d76e949c3805"),
				},
				ConfbridgeID: uuid.FromStringOrNil("1b280016-9861-11ec-999c-5f70848e711d"),
				Type:         conference.TypeConference,
				Name:         "test type conference",
				Detail:       "test type conference detail",
			},

			&curTime,
			&conference.Conference{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("1ac9f480-9861-11ec-8e29-c7820822026e"),
					CustomerID: uuid.FromStringOrNil("1afc3ce2-9861-11ec-90b1-d76e949c3805"),
				},
				ConfbridgeID:      uuid.FromStringOrNil("1b280016-9861-11ec-999c-5f70848e711d"),
				Type:              conference.TypeConference,
				Name:              "test type conference",
				Detail:            "test type conference detail",
				Data:              map[string]interface{}{},
				ConferencecallIDs: []uuid.UUID{},
				RecordingIDs:      []uuid.UUID{},
				TranscribeIDs:     []uuid.UUID{},
				TMEnd:             nil,
				TMCreate:          &curTime,
				TMUpdate:          nil,
				TMDelete:          nil,
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

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
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

func Test_ConferenceUpdate(t *testing.T) {

	curTime := time.Date(2023, 1, 3, 21, 35, 2, 809000000, time.UTC)

	tests := []struct {
		name       string
		conference *conference.Conference

		id     uuid.UUID
		fields map[conference.Field]any

		responseCurTime *time.Time
		expectRes       *conference.Conference
	}{
		{
			name: "test normal",
			conference: &conference.Conference{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("90d83f46-1e0b-11f0-881e-db8cc51c453b"),
				},
			},

			id: uuid.FromStringOrNil("90d83f46-1e0b-11f0-881e-db8cc51c453b"),
			fields: map[conference.Field]any{
				conference.FieldName:   "update name",
				conference.FieldDetail: "update detail",
				conference.FieldData: map[string]interface{}{
					"key1": "string value",
				},
				conference.FieldTimeout:    100,
				conference.FieldPreFlowID:  uuid.FromStringOrNil("910adef6-1e0b-11f0-be96-9b8635c520c0"),
				conference.FieldPostFlowID: uuid.FromStringOrNil("91345970-1e0b-11f0-a446-bf10e1f783b5"),
			},

			responseCurTime: &curTime,
			expectRes: &conference.Conference{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("90d83f46-1e0b-11f0-881e-db8cc51c453b"),
				},
				Name:   "update name",
				Detail: "update detail",
				Data: map[string]interface{}{
					"key1": "string value",
				},
				Timeout:           100,
				ConferencecallIDs: []uuid.UUID{},
				PreFlowID:         uuid.FromStringOrNil("910adef6-1e0b-11f0-be96-9b8635c520c0"),
				PostFlowID:        uuid.FromStringOrNil("91345970-1e0b-11f0-a446-bf10e1f783b5"),
				RecordingIDs:      []uuid.UUID{},
				TranscribeIDs:     []uuid.UUID{},
				TMEnd:             nil,
				TMCreate:          &curTime,
				TMUpdate:          &curTime,
				TMDelete:          nil,
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

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().ConferenceSet(ctx, gomock.Any())
			if err := h.ConferenceCreate(ctx, tt.conference); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().ConferenceSet(ctx, gomock.Any())
			if err := h.ConferenceUpdate(ctx, tt.conference.ID, tt.fields); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ConferenceGet(ctx, tt.conference.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ConferenceSet(ctx, gomock.Any())
			res, err := h.ConferenceGet(ctx, tt.conference.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ConferenceUpdateRecordingID(t *testing.T) {

	curTime := time.Date(2023, 1, 3, 21, 35, 2, 809000000, time.UTC)

	tests := []struct {
		name        string
		conference  *conference.Conference
		recordingID uuid.UUID

		responseCurTime  *time.Time
		expectConference *conference.Conference
	}{
		{
			"test normal",
			&conference.Conference{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2f7b0ee4-2834-11eb-9a6d-5beea5795ea6"),
				},
			},
			uuid.FromStringOrNil("2fb4b446-2834-11eb-b864-1fdb13777d08"),

			&curTime,
			&conference.Conference{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2f7b0ee4-2834-11eb-9a6d-5beea5795ea6"),
				},
				Data:              map[string]interface{}{},
				ConferencecallIDs: []uuid.UUID{},
				RecordingID:       uuid.FromStringOrNil("2fb4b446-2834-11eb-b864-1fdb13777d08"),
				RecordingIDs:      []uuid.UUID{},
				TranscribeIDs:     []uuid.UUID{},
				TMEnd:             nil,
				TMCreate:          &curTime,
				TMUpdate:          &curTime,
				TMDelete:          nil,
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

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().ConferenceSet(ctx, gomock.Any())
			if err := h.ConferenceCreate(ctx, tt.conference); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().ConferenceSet(ctx, gomock.Any())
			fields := map[conference.Field]any{
				conference.FieldRecordingID: tt.recordingID,
			}
			if err := h.ConferenceUpdate(ctx, tt.conference.ID, fields); err != nil {
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

func Test_ConferenceUpdateData(t *testing.T) {

	curTime := time.Date(2023, 1, 3, 21, 35, 2, 809000000, time.UTC)

	tests := []struct {
		name       string
		conference *conference.Conference

		data map[string]interface{}

		responseCurTime  *time.Time
		expectConference *conference.Conference
	}{
		{
			"test normal",
			&conference.Conference{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("0a64e234-675d-11eb-92c7-13f0c9a0e28b"),
				},
			},

			map[string]interface{}{
				"key1": "string value",
			},

			&curTime,
			&conference.Conference{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("0a64e234-675d-11eb-92c7-13f0c9a0e28b"),
				},
				Data: map[string]interface{}{
					"key1": "string value",
				},
				ConferencecallIDs: []uuid.UUID{},
				RecordingIDs:      []uuid.UUID{},
				TranscribeIDs:     []uuid.UUID{},
				TMEnd:             nil,
				TMCreate:          &curTime,
				TMUpdate:          &curTime,
				TMDelete:          nil,
			},
		},
		{
			"update 2 datas",
			&conference.Conference{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("d54bf5b4-675d-11eb-b133-9b06996a9b99"),
				},
			},
			map[string]interface{}{
				"key1": "string value",
				"key2": "string value",
			},

			&curTime,
			&conference.Conference{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("d54bf5b4-675d-11eb-b133-9b06996a9b99"),
				},
				Data: map[string]interface{}{
					"key1": "string value",
					"key2": "string value",
				},
				ConferencecallIDs: []uuid.UUID{},
				RecordingIDs:      []uuid.UUID{},
				TranscribeIDs:     []uuid.UUID{},
				TMEnd:             nil,
				TMCreate:          &curTime,
				TMUpdate:          &curTime,
				TMDelete:          nil,
			},
		},
		{
			"update mixed data types",
			&conference.Conference{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("efa1ec2a-675d-11eb-b854-ffe06d0fc488"),
				},
			},
			map[string]interface{}{
				"key1": "string value",
				"key2": 123,
			},

			&curTime,
			&conference.Conference{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("efa1ec2a-675d-11eb-b854-ffe06d0fc488"),
				},
				Data: map[string]interface{}{
					"key1": "string value",
					"key2": float64(123),
				},
				ConferencecallIDs: []uuid.UUID{},
				RecordingIDs:      []uuid.UUID{},
				TranscribeIDs:     []uuid.UUID{},
				TMEnd:             nil,
				TMCreate:          &curTime,
				TMUpdate:          &curTime,
				TMDelete:          nil,
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

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().ConferenceSet(ctx, gomock.Any())
			if err := h.ConferenceCreate(ctx, tt.conference); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().ConferenceSet(ctx, gomock.Any())
			fields := map[conference.Field]any{
				conference.FieldData: tt.data,
			}
			if err := h.ConferenceUpdate(ctx, tt.conference.ID, fields); err != nil {
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

func Test_ConferenceList(t *testing.T) {

	curTime := time.Date(2023, 1, 3, 21, 35, 2, 809000000, time.UTC)

	tests := []struct {
		name        string
		conferences []*conference.Conference

		count   int
		filters map[conference.Field]any

		responseCurTime *time.Time
		expectRes       []*conference.Conference
	}{
		{
			"normal",
			[]*conference.Conference{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("ac54ebd4-94c9-11ed-b4aa-4f7da8f9741a"),
						CustomerID: uuid.FromStringOrNil("91f25410-7f45-11ec-97d1-8b4f8cee4768"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("ad76ec88-94c9-11ed-9651-df2f9c2178aa"),
						CustomerID: uuid.FromStringOrNil("91f25410-7f45-11ec-97d1-8b4f8cee4768"),
					},
				},
			},

			10,
			map[conference.Field]any{
				conference.FieldCustomerID: uuid.FromStringOrNil("91f25410-7f45-11ec-97d1-8b4f8cee4768"),
				conference.FieldDeleted:    false,
			},

			&curTime,
			[]*conference.Conference{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("ac54ebd4-94c9-11ed-b4aa-4f7da8f9741a"),
						CustomerID: uuid.FromStringOrNil("91f25410-7f45-11ec-97d1-8b4f8cee4768"),
					},
					Data:              map[string]interface{}{},
					ConferencecallIDs: []uuid.UUID{},
					RecordingIDs:      []uuid.UUID{},
					TranscribeIDs:     []uuid.UUID{},
					TMEnd:             nil,
					TMCreate:          &curTime,
					TMUpdate:          nil,
					TMDelete:          nil,
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("ad76ec88-94c9-11ed-9651-df2f9c2178aa"),
						CustomerID: uuid.FromStringOrNil("91f25410-7f45-11ec-97d1-8b4f8cee4768"),
					},
					Data:              map[string]interface{}{},
					ConferencecallIDs: []uuid.UUID{},
					RecordingIDs:      []uuid.UUID{},
					TranscribeIDs:     []uuid.UUID{},
					TMEnd:             nil,
					TMCreate:          &curTime,
					TMUpdate:          nil,
					TMDelete:          nil,
				},
			},
		},

		{
			"gets conference type only",
			[]*conference.Conference{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("418ea85a-94b8-11ed-9cf4-5f71d1d56a86"),
						CustomerID: uuid.FromStringOrNil("80a965e0-7f45-11ec-a078-7f296665fa3d"),
					},
					Type: conference.TypeConference,
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("4b0feace-94b8-11ed-a8a7-f3ffb3124f95"),
						CustomerID: uuid.FromStringOrNil("80a965e0-7f45-11ec-a078-7f296665fa3d"),
					},
					Type: conference.TypeConference,
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("7dec4d52-94b8-11ed-9d79-ff4a5e22e54f"),
						CustomerID: uuid.FromStringOrNil("80a965e0-7f45-11ec-a078-7f296665fa3d"),
					},
					Type: conference.TypeConnect,
				},
			},

			10,
			map[conference.Field]any{
				conference.FieldCustomerID: uuid.FromStringOrNil("80a965e0-7f45-11ec-a078-7f296665fa3d"),
				conference.FieldDeleted:    false,
				conference.FieldType:       conference.TypeConference,
			},

			&curTime,
			[]*conference.Conference{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("418ea85a-94b8-11ed-9cf4-5f71d1d56a86"),
						CustomerID: uuid.FromStringOrNil("80a965e0-7f45-11ec-a078-7f296665fa3d"),
					},
					Type:              conference.TypeConference,
					Data:              map[string]interface{}{},
					ConferencecallIDs: []uuid.UUID{},
					RecordingIDs:      []uuid.UUID{},
					TranscribeIDs:     []uuid.UUID{},
					TMEnd:             nil,
					TMCreate:          &curTime,
					TMUpdate:          nil,
					TMDelete:          nil,
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("4b0feace-94b8-11ed-a8a7-f3ffb3124f95"),
						CustomerID: uuid.FromStringOrNil("80a965e0-7f45-11ec-a078-7f296665fa3d"),
					},
					Type:              conference.TypeConference,
					Data:              map[string]interface{}{},
					ConferencecallIDs: []uuid.UUID{},
					RecordingIDs:      []uuid.UUID{},
					TranscribeIDs:     []uuid.UUID{},
					TMEnd:             nil,
					TMCreate:          &curTime,
					TMUpdate:          nil,
					TMDelete:          nil,
				},
			},
		},

		{
			"empty",
			[]*conference.Conference{},

			0,
			map[conference.Field]any{
				conference.FieldCustomerID: uuid.FromStringOrNil("3f84e9f4-ed84-11ee-9bfb-2bce0d221d0b"),
			},

			&curTime,
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
				mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
				mockCache.EXPECT().ConferenceSet(ctx, gomock.Any())
				if errCreate := h.ConferenceCreate(ctx, cf); errCreate != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", errCreate)
				}
			}

			res, err := h.ConferenceList(ctx, 10, utilhandler.TimeGetCurTime(), tt.filters)
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

	curTime := time.Date(2023, 1, 3, 21, 35, 2, 809000000, time.UTC)

	tests := []struct {
		name       string
		conference *conference.Conference

		id uuid.UUID

		responseCurTime *time.Time
		expectRes       *conference.Conference
	}{
		{
			"normal",
			&conference.Conference{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("722c7822-94ca-11ed-b0a9-ef969fc8348d"),
				},
			},

			uuid.FromStringOrNil("722c7822-94ca-11ed-b0a9-ef969fc8348d"),

			&curTime,
			&conference.Conference{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("722c7822-94ca-11ed-b0a9-ef969fc8348d"),
				},
				Status:            conference.StatusTerminated,
				Data:              map[string]interface{}{},
				ConferencecallIDs: []uuid.UUID{},
				RecordingIDs:      []uuid.UUID{},
				TranscribeIDs:     []uuid.UUID{},
				TMEnd:             &curTime,
				TMCreate:          &curTime,
				TMUpdate:          &curTime,
				TMDelete:          nil,
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

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().ConferenceSet(ctx, gomock.Any())
			if err := h.ConferenceCreate(ctx, tt.conference); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			// ConferenceEnd calls TimeNow once, and internally ConferenceUpdate calls it again
			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime).Times(2)
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

	curTime := time.Date(2023, 1, 3, 21, 35, 2, 809000000, time.UTC)

	tests := []struct {
		name       string
		conference *conference.Conference

		id uuid.UUID

		responseCurTime *time.Time
		expectRes       *conference.Conference
	}{
		{
			"normal",
			&conference.Conference{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("7a23bfa0-94e2-11ed-8dd9-0b374780e823"),
				},
			},

			uuid.FromStringOrNil("7a23bfa0-94e2-11ed-8dd9-0b374780e823"),

			&curTime,
			&conference.Conference{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("7a23bfa0-94e2-11ed-8dd9-0b374780e823"),
				},
				Data:              map[string]interface{}{},
				ConferencecallIDs: []uuid.UUID{},
				RecordingIDs:      []uuid.UUID{},
				TranscribeIDs:     []uuid.UUID{},
				TMEnd:             nil,
				TMCreate:          &curTime,
				TMUpdate:          &curTime,
				TMDelete:          &curTime,
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

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().ConferenceSet(ctx, gomock.Any())
			if err := h.ConferenceCreate(ctx, tt.conference); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
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

func Test_ConferenceUpdateTranscribeID(t *testing.T) {

	curTime := time.Date(2023, 1, 3, 21, 35, 2, 809000000, time.UTC)

	tests := []struct {
		name         string
		conference   *conference.Conference
		transcribeID uuid.UUID

		responseCurTime  *time.Time
		expectConference *conference.Conference
	}{
		{
			"normal",
			&conference.Conference{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("000ca104-98c1-11ed-bde2-9badb79a7365"),
				},
			},
			uuid.FromStringOrNil("003eb216-98c1-11ed-9789-ff71dbeab66e"),

			&curTime,
			&conference.Conference{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("000ca104-98c1-11ed-bde2-9badb79a7365"),
				},
				Data:              map[string]interface{}{},
				ConferencecallIDs: []uuid.UUID{},
				RecordingIDs:      []uuid.UUID{},
				TranscribeID:      uuid.FromStringOrNil("003eb216-98c1-11ed-9789-ff71dbeab66e"),
				TranscribeIDs:     []uuid.UUID{},
				TMEnd:             nil,
				TMCreate:          &curTime,
				TMUpdate:          &curTime,
				TMDelete:          nil,
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

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().ConferenceSet(ctx, gomock.Any())
			if err := h.ConferenceCreate(ctx, tt.conference); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().ConferenceSet(ctx, gomock.Any())
			fields := map[conference.Field]any{
				conference.FieldTranscribeID: tt.transcribeID,
			}
			if err := h.ConferenceUpdate(ctx, tt.conference.ID, fields); err != nil {
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

func Test_ConferenceCountByCustomerID(t *testing.T) {

	curTime := time.Date(2023, 1, 3, 21, 35, 2, 809000000, time.UTC)
	customerID := uuid.FromStringOrNil("8512e56c-cb08-46fa-96de-7855d0889577")

	tests := []struct {
		name        string
		conferences []*conference.Conference
		customerID  uuid.UUID
		expected    int
	}{
		{
			name: "count two conferences",
			conferences: []*conference.Conference{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("43b02684-94ce-11ed-95e6-3727def0e4fd"),
						CustomerID: customerID,
					},
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("50be6c64-94ce-11ed-9def-dfcdc44a112d"),
						CustomerID: customerID,
					},
				},
			},
			customerID: customerID,
			expected:   2,
		},
		{
			name:        "count zero conferences",
			conferences: []*conference.Conference{},
			customerID:  uuid.FromStringOrNil("9c61ef24-b396-465b-9705-44b420f2dc5d"),
			expected:    0,
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
				mockUtil.EXPECT().TimeNow().Return(&curTime)
				mockCache.EXPECT().ConferenceSet(ctx, gomock.Any())
				if err := h.ConferenceCreate(ctx, cf); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			count, err := h.ConferenceCountByCustomerID(ctx, tt.customerID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if count != tt.expected {
				t.Errorf("Wrong count. expect: %d, got: %d", tt.expected, count)
			}
		})
	}
}
