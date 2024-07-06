package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"monorepo/bin-common-handler/pkg/utilhandler"

	commonidentity "monorepo/bin-common-handler/models/identity"

	uuid "github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"monorepo/bin-chat-manager/models/chatroom"
	"monorepo/bin-chat-manager/pkg/cachehandler"
)

func Test_ChatroomCreate(t *testing.T) {

	tests := []struct {
		name string

		chatroom *chatroom.Chatroom

		expectRes *chatroom.Chatroom
	}{
		{
			"empty all",

			&chatroom.Chatroom{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("5772b2d8-0247-4214-9588-56d6792bc15b"),
				},
			},

			&chatroom.Chatroom{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("5772b2d8-0247-4214-9588-56d6792bc15b"),
				},
			},
		},
		{
			"all items",

			&chatroom.Chatroom{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("9d4ea39d-e2dc-4254-99aa-04f05a7bedc7"),
					CustomerID: uuid.FromStringOrNil("90b2d956-9d0a-426d-90e6-1e60585833fe"),
					OwnerID:    uuid.FromStringOrNil("6512dac0-da31-11ee-95b1-b7241292ed37"),
				},

				Type:   chatroom.TypeNormal,
				ChatID: uuid.FromStringOrNil("78464d96-d6f3-4688-9f9d-9db0a4de5694"),

				RoomOwnerID: uuid.FromStringOrNil("6d6150d4-1121-4ec9-9f9d-2387b918c35e"),
				ParticipantIDs: []uuid.UUID{
					uuid.FromStringOrNil("ca06a988-df95-412b-9b5f-8ebbce71c0e8"),
					uuid.FromStringOrNil("2b754dfc-03fe-4a17-b1a0-c0d95b2cc1be"),
				},
				Name:     "test name",
				Detail:   "test detail",
				TMCreate: "2020-04-18T03:22:17.995000",
				TMUpdate: "2020-04-18T03:22:17.995000",
				TMDelete: DefaultTimeStamp,
			},

			&chatroom.Chatroom{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("9d4ea39d-e2dc-4254-99aa-04f05a7bedc7"),
					CustomerID: uuid.FromStringOrNil("90b2d956-9d0a-426d-90e6-1e60585833fe"),
					OwnerID:    uuid.FromStringOrNil("6512dac0-da31-11ee-95b1-b7241292ed37"),
				},

				Type:   chatroom.TypeNormal,
				ChatID: uuid.FromStringOrNil("78464d96-d6f3-4688-9f9d-9db0a4de5694"),

				RoomOwnerID: uuid.FromStringOrNil("6d6150d4-1121-4ec9-9f9d-2387b918c35e"),
				ParticipantIDs: []uuid.UUID{
					uuid.FromStringOrNil("ca06a988-df95-412b-9b5f-8ebbce71c0e8"),
					uuid.FromStringOrNil("2b754dfc-03fe-4a17-b1a0-c0d95b2cc1be"),
				},
				Name:     "test name",
				Detail:   "test detail",
				TMCreate: "2020-04-18T03:22:17.995000",
				TMUpdate: "2020-04-18T03:22:17.995000",
				TMDelete: DefaultTimeStamp,
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

			mockCache.EXPECT().ChatroomSet(ctx, gomock.Any())
			if err := h.ChatroomCreate(ctx, tt.chatroom); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ChatroomGet(ctx, tt.chatroom.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ChatroomSet(ctx, gomock.Any())
			res, err := h.ChatroomGet(ctx, tt.chatroom.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ChatroomGets(t *testing.T) {

	tests := []struct {
		name string
		data []*chatroom.Chatroom

		size    uint64
		filters map[string]string

		expectRes []*chatroom.Chatroom
	}{
		{
			"normal",
			[]*chatroom.Chatroom{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("f17903aa-1c48-4ce5-80da-5e62fc0e7951"),
						CustomerID: uuid.FromStringOrNil("63331078-4431-43ed-96ac-5975fa9e6749"),
					},
					Type:        chatroom.TypeNormal,
					ChatID:      uuid.FromStringOrNil("152d7e64-bac6-11ee-a61f-67fba233c588"),
					RoomOwnerID: uuid.FromStringOrNil("155d19da-bac6-11ee-8ed8-8fff6c8907d5"),
					TMCreate:    "2020-04-19T03:22:17.995000",
					TMDelete:    DefaultTimeStamp,
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("16d46a60-79b5-42af-99d2-e9c7404ab642"),
						CustomerID: uuid.FromStringOrNil("63331078-4431-43ed-96ac-5975fa9e6749"),
					},
					Type:     chatroom.TypeNormal,
					TMCreate: "2020-04-18T03:22:17.995000",
					TMDelete: DefaultTimeStamp,
				},
			},

			10,
			map[string]string{
				"customer_id":   "63331078-4431-43ed-96ac-5975fa9e6749",
				"deleted":       "false",
				"type":          string(chatroom.TypeNormal),
				"room_owner_id": "155d19da-bac6-11ee-8ed8-8fff6c8907d5",
				"chat_id":       "152d7e64-bac6-11ee-a61f-67fba233c588",
			},

			[]*chatroom.Chatroom{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("f17903aa-1c48-4ce5-80da-5e62fc0e7951"),
						CustomerID: uuid.FromStringOrNil("63331078-4431-43ed-96ac-5975fa9e6749"),
					},
					Type:        chatroom.TypeNormal,
					ChatID:      uuid.FromStringOrNil("152d7e64-bac6-11ee-a61f-67fba233c588"),
					RoomOwnerID: uuid.FromStringOrNil("155d19da-bac6-11ee-8ed8-8fff6c8907d5"),
					TMCreate:    "2020-04-19T03:22:17.995000",
					TMDelete:    DefaultTimeStamp,
				},
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

			for _, c := range tt.data {
				mockCache.EXPECT().ChatroomSet(ctx, gomock.Any())
				if err := h.ChatroomCreate(ctx, c); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			cs, err := h.ChatroomGets(ctx, utilhandler.TimeGetCurTime(), tt.size, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(cs, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.data, cs)
			}
		})
	}
}

func Test_ChatroomUpdateBasicInfo(t *testing.T) {

	tests := []struct {
		name     string
		chatroom *chatroom.Chatroom

		id           uuid.UUID
		chatroomName string
		detail       string

		expectRes *chatroom.Chatroom
	}{
		{
			"normal",
			&chatroom.Chatroom{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("62b08684-6a3b-4416-910f-d77843d41932"),
				},
			},

			uuid.FromStringOrNil("62b08684-6a3b-4416-910f-d77843d41932"),
			"update name",
			"update detail",

			&chatroom.Chatroom{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("62b08684-6a3b-4416-910f-d77843d41932"),
				},
				Name:   "update name",
				Detail: "update detail",
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
			mockCache.EXPECT().ChatroomSet(ctx, gomock.Any())
			if err := h.ChatroomCreate(ctx, tt.chatroom); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ChatroomSet(ctx, gomock.Any())
			if err := h.ChatroomUpdateBasicInfo(ctx, tt.id, tt.chatroomName, tt.detail); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ChatroomGet(ctx, tt.chatroom.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ChatroomSet(ctx, gomock.Any())
			res, err := h.ChatroomGet(ctx, tt.id)
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

func Test_ChatroomDelete(t *testing.T) {
	tests := []struct {
		name string

		id uuid.UUID

		chatroom *chatroom.Chatroom
	}{
		{
			"normal",

			uuid.FromStringOrNil("0c1fde2b-1985-42b4-8eb3-96888e0477b5"),

			&chatroom.Chatroom{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("0c1fde2b-1985-42b4-8eb3-96888e0477b5"),
				},
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
			mockCache.EXPECT().ChatroomSet(ctx, gomock.Any())
			if err := h.ChatroomCreate(ctx, tt.chatroom); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ChatroomSet(ctx, gomock.Any())
			if err := h.ChatroomDelete(ctx, tt.id); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ChatroomGet(ctx, tt.id).Return(nil, fmt.Errorf("error"))
			mockCache.EXPECT().ChatroomSet(ctx, gomock.Any()).Return(nil)
			res, err := h.ChatroomGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res.TMDelete == DefaultTimeStamp {
				t.Errorf("Wrong match. expect: any other, got: %s", res.TMDelete)
			}
		})
	}
}
