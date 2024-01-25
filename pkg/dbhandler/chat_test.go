package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	uuid "github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/chat-manager.git/models/chat"
	"gitlab.com/voipbin/bin-manager/chat-manager.git/pkg/cachehandler"
)

func Test_ChatCreate(t *testing.T) {

	tests := []struct {
		name string

		data      *chat.Chat
		expectRes *chat.Chat
	}{
		{
			"normal",

			&chat.Chat{
				ID:         uuid.FromStringOrNil("d649c8b5-c5f6-4740-a28c-653a47195a1d"),
				CustomerID: uuid.FromStringOrNil("4dbf893c-82fe-4dea-8079-272670aea7b4"),
				Type:       chat.TypeNormal,
				OwnerID:    uuid.FromStringOrNil("c4111534-e222-4e0a-9485-f012ea0a9e02"),
				ParticipantIDs: []uuid.UUID{
					uuid.FromStringOrNil("47adb891-80aa-4ba4-af47-446faa6cf386"),
					uuid.FromStringOrNil("3beef418-cf3a-4239-adff-6f096cebacae"),
				},
				Name:     "test name",
				Detail:   "test detail",
				TMCreate: "2020-04-18T03:22:17.995000",
				TMUpdate: "2020-04-18T03:22:17.995000",
				TMDelete: DefaultTimeStamp,
			},

			&chat.Chat{
				ID:         uuid.FromStringOrNil("d649c8b5-c5f6-4740-a28c-653a47195a1d"),
				CustomerID: uuid.FromStringOrNil("4dbf893c-82fe-4dea-8079-272670aea7b4"),
				Type:       chat.TypeNormal,
				OwnerID:    uuid.FromStringOrNil("c4111534-e222-4e0a-9485-f012ea0a9e02"),
				ParticipantIDs: []uuid.UUID{
					uuid.FromStringOrNil("3beef418-cf3a-4239-adff-6f096cebacae"),
					uuid.FromStringOrNil("47adb891-80aa-4ba4-af47-446faa6cf386"),
				},
				Name:     "test name",
				Detail:   "test detail",
				TMCreate: "2020-04-18T03:22:17.995000",
				TMUpdate: "2020-04-18T03:22:17.995000",
				TMDelete: DefaultTimeStamp,
			},
		},
		{
			"empty",

			&chat.Chat{
				ID:             uuid.FromStringOrNil("6127ed80-8f71-4f93-883c-bbeab83f0a10"),
				ParticipantIDs: []uuid.UUID{},
				TMDelete:       DefaultTimeStamp,
			},

			&chat.Chat{
				ID:             uuid.FromStringOrNil("6127ed80-8f71-4f93-883c-bbeab83f0a10"),
				ParticipantIDs: []uuid.UUID{},
				TMDelete:       DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				db:    dbTest,
				cache: mockCache,
			}
			ctx := context.Background()

			mockCache.EXPECT().ChatSet(ctx, gomock.Any())
			if err := h.ChatCreate(ctx, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ChatGet(ctx, tt.data.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ChatSet(ctx, gomock.Any())

			res, err := h.ChatGet(ctx, tt.data.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.data, res)
			}
		})
	}
}

func Test_ChatGets(t *testing.T) {

	tests := []struct {
		name string
		data []*chat.Chat

		customerID uuid.UUID
		limit      uint64
		filters    map[string]string
	}{
		{
			"normal",
			[]*chat.Chat{
				{
					ID:         uuid.FromStringOrNil("837117d8-0c31-11eb-9f9e-6b4ac01a7e66"),
					Type:       chat.TypeNormal,
					CustomerID: uuid.FromStringOrNil("20db856a-051f-49d3-986e-3715af514273"),
					OwnerID:    uuid.FromStringOrNil("e90b7010-b936-11ee-8da8-bb61becf0b57"),
					ParticipantIDs: []uuid.UUID{
						uuid.FromStringOrNil("e90b7010-b936-11ee-8da8-bb61becf0b57"),
						uuid.FromStringOrNil("e9e04548-b958-11ee-af60-1721a820a3d2"),
					},
					TMCreate: "2020-04-19T03:22:17.995000",
					TMDelete: DefaultTimeStamp,
				},
				{
					ID:         uuid.FromStringOrNil("845e04f8-0c31-11eb-a8cf-6f8836b86b2b"),
					CustomerID: uuid.FromStringOrNil("20db856a-051f-49d3-986e-3715af514273"),
					Type:       chat.TypeNormal,
					OwnerID:    uuid.FromStringOrNil("e90b7010-b936-11ee-8da8-bb61becf0b57"),
					ParticipantIDs: []uuid.UUID{
						uuid.FromStringOrNil("e90b7010-b936-11ee-8da8-bb61becf0b57"),
						uuid.FromStringOrNil("ea09d3fe-b958-11ee-8b3b-a3ed291a64df"),
					},
					TMCreate: "2020-04-18T03:22:17.995000",
					TMDelete: DefaultTimeStamp,
				},
			},

			uuid.FromStringOrNil("20db856a-051f-49d3-986e-3715af514273"),
			10,
			map[string]string{
				"participant_ids": "",
				"customer_id":     "20db856a-051f-49d3-986e-3715af514273",
				"deleted":         "false",
				"type":            string(chat.TypeNormal),
				"owner_id":        "e90b7010-b936-11ee-8da8-bb61becf0b57",
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

			for _, c := range tt.data {
				mockCache.EXPECT().ChatSet(ctx, gomock.Any())
				if err := h.ChatCreate(ctx, c); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			cs, err := h.ChatGets(ctx, utilhandler.TimeGetCurTime(), tt.limit, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(cs, tt.data) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.data, cs)
			}
		})
	}
}

func Test_ChatUpdateBasic(t *testing.T) {

	tests := []struct {
		name string
		chat *chat.Chat

		id       uuid.UUID
		chatName string
		detail   string

		expectRes *chat.Chat
	}{
		{
			"normal",
			&chat.Chat{
				ID: uuid.FromStringOrNil("599b43c2-bb10-45fb-a4f4-00e04ecf4c28"),
			},

			uuid.FromStringOrNil("599b43c2-bb10-45fb-a4f4-00e04ecf4c28"),
			"update name",
			"update detail",

			&chat.Chat{
				ID:             uuid.FromStringOrNil("599b43c2-bb10-45fb-a4f4-00e04ecf4c28"),
				Name:           "update name",
				Detail:         "update detail",
				ParticipantIDs: []uuid.UUID{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := NewHandler(dbTest, mockCache)

			ctx := context.Background()

			mockCache.EXPECT().ChatSet(ctx, gomock.Any())
			if err := h.ChatCreate(ctx, tt.chat); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ChatSet(ctx, gomock.Any())
			if err := h.ChatUpdateBasicInfo(ctx, tt.id, tt.chatName, tt.detail); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ChatGet(ctx, tt.chat.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ChatSet(ctx, gomock.Any())
			res, err := h.ChatGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res.TMUpdate = ""
			res.TMCreate = ""
			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ChatDelete(t *testing.T) {
	tests := []struct {
		name string

		id uuid.UUID

		chat *chat.Chat
	}{
		{
			"normal",

			uuid.FromStringOrNil("66edc48d-d2c0-475f-ad10-9e2fffec8626"),

			&chat.Chat{
				ID: uuid.FromStringOrNil("66edc48d-d2c0-475f-ad10-9e2fffec8626"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := &handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}

			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(utilhandler.TimeGetCurTime())
			mockCache.EXPECT().ChatSet(ctx, gomock.Any())
			if err := h.ChatCreate(ctx, tt.chat); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ChatSet(ctx, gomock.Any())
			if err := h.ChatDelete(ctx, tt.id); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ChatGet(ctx, tt.id).Return(nil, fmt.Errorf("error"))
			mockCache.EXPECT().ChatSet(ctx, gomock.Any()).Return(nil)
			res, err := h.ChatGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res.TMDelete == DefaultTimeStamp {
				t.Errorf("Wrong match. expect: any other, got: %s", res.TMDelete)
			}
		})
	}
}

func Test_ChatUpdateOwnerID(t *testing.T) {

	tests := []struct {
		name string
		chat *chat.Chat

		id      uuid.UUID
		ownerID uuid.UUID

		expectRes *chat.Chat
	}{
		{
			"normal",
			&chat.Chat{
				ID:      uuid.FromStringOrNil("48703865-5b0f-4d82-b7de-9267e040996b"),
				Name:    "name",
				Detail:  "detail",
				OwnerID: uuid.FromStringOrNil("0497e2dc-4af5-4baa-a986-8ca50bf001fb"),
			},

			uuid.FromStringOrNil("48703865-5b0f-4d82-b7de-9267e040996b"),
			uuid.FromStringOrNil("4b25fe52-02c7-4201-9fa8-91b1bfce068e"),

			&chat.Chat{
				ID:             uuid.FromStringOrNil("48703865-5b0f-4d82-b7de-9267e040996b"),
				Name:           "name",
				Detail:         "detail",
				OwnerID:        uuid.FromStringOrNil("4b25fe52-02c7-4201-9fa8-91b1bfce068e"),
				ParticipantIDs: []uuid.UUID{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := NewHandler(dbTest, mockCache)

			ctx := context.Background()

			mockCache.EXPECT().ChatSet(ctx, gomock.Any())
			if err := h.ChatCreate(ctx, tt.chat); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ChatSet(ctx, gomock.Any())
			if err := h.ChatUpdateOwnerID(ctx, tt.id, tt.ownerID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ChatGet(ctx, tt.id).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ChatSet(ctx, gomock.Any())
			res, err := h.ChatGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res.TMUpdate = ""
			res.TMCreate = ""
			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ChatUpdateParticipantID(t *testing.T) {

	tests := []struct {
		name string
		data *chat.Chat

		id             uuid.UUID
		ownerID        uuid.UUID
		praticipantIDs []uuid.UUID

		responseCurTime string

		expectRes *chat.Chat
	}{
		{
			"normal",
			&chat.Chat{
				ID:     uuid.FromStringOrNil("cf217ce0-b953-11ee-be3e-5fa75128cdbf"),
				Name:   "name",
				Detail: "detail",
				ParticipantIDs: []uuid.UUID{
					uuid.FromStringOrNil("52d0b714-b958-11ee-bb4f-cf04464c6d32"),
				},
			},

			uuid.FromStringOrNil("cf217ce0-b953-11ee-be3e-5fa75128cdbf"),
			uuid.FromStringOrNil("4b25fe52-02c7-4201-9fa8-91b1bfce068e"),
			[]uuid.UUID{
				uuid.FromStringOrNil("94f569d2-b953-11ee-9a0d-5b787552edb3"),
				uuid.FromStringOrNil("95237778-b953-11ee-8e5d-9fd34365d05b"),
			},

			"2024-01-22 18:27:22.363616708",

			&chat.Chat{
				ID:     uuid.FromStringOrNil("cf217ce0-b953-11ee-be3e-5fa75128cdbf"),
				Name:   "name",
				Detail: "detail",
				ParticipantIDs: []uuid.UUID{
					uuid.FromStringOrNil("94f569d2-b953-11ee-9a0d-5b787552edb3"),
					uuid.FromStringOrNil("95237778-b953-11ee-8e5d-9fd34365d05b"),
				},
				TMUpdate: "2024-01-22 18:27:22.363616708",
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

			mockCache.EXPECT().ChatSet(ctx, gomock.Any())
			if err := h.ChatCreate(ctx, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().ChatSet(ctx, gomock.Any())
			if err := h.ChatUpdateParticipantID(ctx, tt.id, tt.praticipantIDs); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ChatGet(ctx, tt.id).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ChatSet(ctx, gomock.Any())
			res, err := h.ChatGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_chatFilterParseParticipantIDs(t *testing.T) {
	tests := []struct {
		name string

		participantIDs string
		expectRes      string
	}{
		{
			"normal",

			"db66f260-b95c-11ee-9aca-c3699d079c51",
			`["db66f260-b95c-11ee-9aca-c3699d079c51"]`,
		},
		{
			"3 items and sort",

			"71baa77c-b95b-11ee-a207-87f226f90d55,2d754c5c-b95b-11ee-a6a8-9bf04c2855cc,2db40618-b95b-11ee-b575-b37b01a3f3d3",
			`["2d754c5c-b95b-11ee-a6a8-9bf04c2855cc","2db40618-b95b-11ee-b575-b37b01a3f3d3","71baa77c-b95b-11ee-a207-87f226f90d55"]`,
		},
		{
			"empty",

			"",
			``,
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

			res := h.chatFilterParseParticipantIDs(tt.participantIDs)
			if res != tt.expectRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}

}
