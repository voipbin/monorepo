package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"

	"gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/pkg/cachehandler"
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
				UserID: 1,
				Type:   conference.TypeConference,
				Name:   "test type conference",
				Detail: "test type conference detail",
			},
			&conference.Conference{
				ID:           uuid.FromStringOrNil("26a42912-9163-11ea-93ca-bf5915635f88"),
				UserID:       1,
				Type:         conference.TypeConference,
				Name:         "test type conference",
				Detail:       "test type conference detail",
				PreActions:   []fmaction.Action{},
				PostActions:  []fmaction.Action{},
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
				PreActions:   []fmaction.Action{},
				PostActions:  []fmaction.Action{},
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
				PreActions:   []fmaction.Action{},
				PostActions:  []fmaction.Action{},
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
				PreActions:   []fmaction.Action{},
				PostActions:  []fmaction.Action{},
				CallIDs:      []uuid.UUID{},
				RecordingIDs: []uuid.UUID{uuid.FromStringOrNil("515f79ce-6122-11eb-b3ca-db50409503c4")},
			},
		},
		{
			"set webhook uri",
			&conference.Conference{
				ID:           uuid.FromStringOrNil("6cbf3216-1ff6-11ec-874c-9fdc6af9a2e1"),
				UserID:       1,
				Type:         conference.TypeConference,
				Name:         "test type conference",
				Detail:       "test type conference detail",
				RecordingIDs: []uuid.UUID{uuid.FromStringOrNil("71aac0ec-1ff6-11ec-bfd0-af46a0a99821")},
				WebhookURI:   "test.com/webhook",
			},
			&conference.Conference{
				ID:           uuid.FromStringOrNil("6cbf3216-1ff6-11ec-874c-9fdc6af9a2e1"),
				UserID:       1,
				Type:         conference.TypeConference,
				Name:         "test type conference",
				Detail:       "test type conference detail",
				PreActions:   []fmaction.Action{},
				PostActions:  []fmaction.Action{},
				CallIDs:      []uuid.UUID{},
				RecordingIDs: []uuid.UUID{uuid.FromStringOrNil("71aac0ec-1ff6-11ec-bfd0-af46a0a99821")},
				WebhookURI:   "test.com/webhook",
			},
		},
		{
			"set pre actions",
			&conference.Conference{
				ID:     uuid.FromStringOrNil("3824d0e4-3be7-11ec-b046-674e88e91f56"),
				UserID: 1,
				Type:   conference.TypeConference,
				Name:   "test type conference",
				Detail: "test type conference detail",
				PreActions: []fmaction.Action{
					{
						Type: "answer",
					},
				},
			},
			&conference.Conference{
				ID:     uuid.FromStringOrNil("3824d0e4-3be7-11ec-b046-674e88e91f56"),
				UserID: 1,
				Type:   conference.TypeConference,
				Name:   "test type conference",
				Detail: "test type conference detail",
				PreActions: []fmaction.Action{
					{
						Type: "answer",
					},
				},
				PostActions:  []fmaction.Action{},
				CallIDs:      []uuid.UUID{},
				RecordingIDs: []uuid.UUID{},
			},
		},
		{
			"set post actions",
			&conference.Conference{
				ID:     uuid.FromStringOrNil("e3d6c112-3bed-11ec-99be-5bc49af8efc2"),
				UserID: 1,
				Type:   conference.TypeConference,
				Name:   "test type conference",
				Detail: "test type conference detail",
				PostActions: []fmaction.Action{
					{
						Type: "answer",
					},
				},
			},
			&conference.Conference{
				ID:         uuid.FromStringOrNil("e3d6c112-3bed-11ec-99be-5bc49af8efc2"),
				UserID:     1,
				Type:       conference.TypeConference,
				Name:       "test type conference",
				Detail:     "test type conference detail",
				PreActions: []fmaction.Action{},
				PostActions: []fmaction.Action{
					{
						Type: "answer",
					},
				},
				CallIDs:      []uuid.UUID{},
				RecordingIDs: []uuid.UUID{},
			},
		},
		{
			"set timeout",
			&conference.Conference{
				ID:      uuid.FromStringOrNil("05d6758c-3bee-11ec-a2f7-e78825f985e2"),
				UserID:  1,
				Type:    conference.TypeConference,
				Name:    "test type conference",
				Detail:  "test type conference detail",
				Timeout: 86400,
			},
			&conference.Conference{
				ID:           uuid.FromStringOrNil("05d6758c-3bee-11ec-a2f7-e78825f985e2"),
				UserID:       1,
				Type:         conference.TypeConference,
				Name:         "test type conference",
				Detail:       "test type conference detail",
				Timeout:      86400,
				PreActions:   []fmaction.Action{},
				PostActions:  []fmaction.Action{},
				CallIDs:      []uuid.UUID{},
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

			mockCache.EXPECT().ConferenceGet(gomock.Any(), tt.conference.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ConferenceSet(gomock.Any(), gomock.Any())
			res, err := h.ConferenceGet(context.Background(), tt.conference.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res.TMCreate = ""
			if reflect.DeepEqual(tt.expectConference, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectConference, res)
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
				PreActions:   []fmaction.Action{},
				PostActions:  []fmaction.Action{},
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

func TestConferenceSetData(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	type test struct {
		name             string
		conference       *conference.Conference
		data             map[string]interface{}
		expectConference *conference.Conference
	}

	tests := []test{
		{
			"test normal",
			&conference.Conference{
				ID: uuid.FromStringOrNil("0a64e234-675d-11eb-92c7-13f0c9a0e28b"),
			},
			map[string]interface{}{
				"key1": "string value",
			},
			&conference.Conference{
				ID:           uuid.FromStringOrNil("0a64e234-675d-11eb-92c7-13f0c9a0e28b"),
				PreActions:   []fmaction.Action{},
				PostActions:  []fmaction.Action{},
				CallIDs:      []uuid.UUID{},
				RecordingIDs: []uuid.UUID{},
				Data: map[string]interface{}{
					"key1": "string value",
				},
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
			&conference.Conference{
				ID:           uuid.FromStringOrNil("d54bf5b4-675d-11eb-b133-9b06996a9b99"),
				PreActions:   []fmaction.Action{},
				PostActions:  []fmaction.Action{},
				CallIDs:      []uuid.UUID{},
				RecordingIDs: []uuid.UUID{},
				Data: map[string]interface{}{
					"key1": "string value",
					"key2": "string value",
				},
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
			&conference.Conference{
				ID:           uuid.FromStringOrNil("efa1ec2a-675d-11eb-b854-ffe06d0fc488"),
				PreActions:   []fmaction.Action{},
				PostActions:  []fmaction.Action{},
				CallIDs:      []uuid.UUID{},
				RecordingIDs: []uuid.UUID{},
				Data: map[string]interface{}{
					"key1": "string value",
					"key2": float64(123),
				},
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
			if err := h.ConferenceSetData(context.Background(), tt.conference.ID, tt.data); err != nil {
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

func TestConferenceGetsWithType(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{
		db:    dbTest,
		cache: mockCache,
	}

	tests := []struct {
		name string

		userID uint64
		count  int
	}{
		{
			"normal",
			99,
			10,
		},
		{
			"empty",
			98,
			0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			ctx := context.Background()

			for i := 0; i < tt.count; i++ {
				cf := &conference.Conference{
					ID:       uuid.Must(uuid.NewV4()),
					UserID:   tt.userID,
					Type:     conference.TypeConference,
					TMDelete: defaultTimeStamp,
				}

				mockCache.EXPECT().ConferenceSet(gomock.Any(), gomock.Any())
				_ = h.ConferenceCreate(ctx, cf)
			}

			res, err := h.ConferenceGetsWithType(ctx, tt.userID, conference.TypeConference, 10, getCurTime())
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if len(res) != tt.count {
				t.Errorf("Wrong match. expect: %d, got: %v", tt.count, len(res))
			}
		})
	}
}

func TestConferenceGets(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{
		db:    dbTest,
		cache: mockCache,
	}

	tests := []struct {
		name string

		userID uint64
		count  int
	}{
		{
			"normal",
			97,
			10,
		},
		{
			"empty",
			96,
			0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			ctx := context.Background()

			for i := 0; i < tt.count; i++ {
				cf := &conference.Conference{
					ID:       uuid.Must(uuid.NewV4()),
					UserID:   tt.userID,
					TMDelete: defaultTimeStamp,
				}

				mockCache.EXPECT().ConferenceSet(gomock.Any(), gomock.Any())
				_ = h.ConferenceCreate(ctx, cf)
			}

			res, err := h.ConferenceGets(ctx, tt.userID, 10, getCurTime())
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if len(res) != tt.count {
				t.Errorf("Wrong match. expect: %d, got: %v", tt.count, len(res))
			}
		})
	}
}
