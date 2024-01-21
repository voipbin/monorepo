package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	uuid "github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/chat-manager.git/models/chatroom"
	"gitlab.com/voipbin/bin-manager/chat-manager.git/pkg/cachehandler"
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
				ID: uuid.FromStringOrNil("5772b2d8-0247-4214-9588-56d6792bc15b"),
			},

			&chatroom.Chatroom{
				ID: uuid.FromStringOrNil("5772b2d8-0247-4214-9588-56d6792bc15b"),
			},
		},
		{
			"all items",

			&chatroom.Chatroom{
				ID:         uuid.FromStringOrNil("9d4ea39d-e2dc-4254-99aa-04f05a7bedc7"),
				CustomerID: uuid.FromStringOrNil("90b2d956-9d0a-426d-90e6-1e60585833fe"),

				Type:   chatroom.TypeNormal,
				ChatID: uuid.FromStringOrNil("78464d96-d6f3-4688-9f9d-9db0a4de5694"),

				OwnerID: uuid.FromStringOrNil("6d6150d4-1121-4ec9-9f9d-2387b918c35e"),
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
				ID:         uuid.FromStringOrNil("9d4ea39d-e2dc-4254-99aa-04f05a7bedc7"),
				CustomerID: uuid.FromStringOrNil("90b2d956-9d0a-426d-90e6-1e60585833fe"),

				Type:   chatroom.TypeNormal,
				ChatID: uuid.FromStringOrNil("78464d96-d6f3-4688-9f9d-9db0a4de5694"),

				OwnerID: uuid.FromStringOrNil("6d6150d4-1121-4ec9-9f9d-2387b918c35e"),
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

func Test_ChatroomGetsByCustomerID(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		size       uint64
		filters    map[string]string

		chatrooms []*chatroom.Chatroom
	}{
		{
			"normal",
			uuid.FromStringOrNil("63331078-4431-43ed-96ac-5975fa9e6749"),
			10,
			map[string]string{
				"deleted": "false",
			},

			[]*chatroom.Chatroom{
				{
					ID:         uuid.FromStringOrNil("f17903aa-1c48-4ce5-80da-5e62fc0e7951"),
					CustomerID: uuid.FromStringOrNil("63331078-4431-43ed-96ac-5975fa9e6749"),
					TMCreate:   "2020-04-19T03:22:17.995000",
					TMDelete:   DefaultTimeStamp,
				},
				{
					ID:         uuid.FromStringOrNil("16d46a60-79b5-42af-99d2-e9c7404ab642"),
					CustomerID: uuid.FromStringOrNil("63331078-4431-43ed-96ac-5975fa9e6749"),
					TMCreate:   "2020-04-18T03:22:17.995000",
					TMDelete:   DefaultTimeStamp,
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

			for _, c := range tt.chatrooms {
				mockCache.EXPECT().ChatroomSet(ctx, gomock.Any())
				if err := h.ChatroomCreate(ctx, c); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			cs, err := h.ChatroomGetsByCustomerID(ctx, tt.customerID, GetCurTime(), tt.size, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(cs, tt.chatrooms) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.chatrooms, cs)
			}
		})
	}
}

func Test_ChatroomGetsByOwnerID(t *testing.T) {

	tests := []struct {
		name string

		ownerID uuid.UUID
		size    uint64
		filters map[string]string

		chatrooms []*chatroom.Chatroom
	}{
		{
			"normal",
			uuid.FromStringOrNil("df474d84-b82c-11ee-93f6-d3ff927a1060"),
			10,
			map[string]string{
				"deleted": "false",
			},

			[]*chatroom.Chatroom{
				{
					ID:       uuid.FromStringOrNil("5ba01d6a-b82e-11ee-8699-bfdaee561985"),
					OwnerID:  uuid.FromStringOrNil("df474d84-b82c-11ee-93f6-d3ff927a1060"),
					TMCreate: "2020-04-19T03:22:17.995000",
					TMDelete: DefaultTimeStamp,
				},
				{
					ID:       uuid.FromStringOrNil("5bddcc6e-b82e-11ee-933a-5f95f3a20894"),
					OwnerID:  uuid.FromStringOrNil("df474d84-b82c-11ee-93f6-d3ff927a1060"),
					TMCreate: "2020-04-18T03:22:17.995000",
					TMDelete: DefaultTimeStamp,
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

			for _, c := range tt.chatrooms {
				mockCache.EXPECT().ChatroomSet(ctx, gomock.Any())
				if err := h.ChatroomCreate(ctx, c); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			cs, err := h.ChatroomGetsByOwnerID(ctx, tt.ownerID, GetCurTime(), tt.size, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(cs, tt.chatrooms) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.chatrooms, cs)
			}
		})
	}
}

func Test_ChatroomGetsByType(t *testing.T) {

	tests := []struct {
		name         string
		customerID   uuid.UUID
		chatroomType chatroom.Type
		limit        uint64
		chatrooms    []*chatroom.Chatroom
	}{
		{
			"type single",
			uuid.FromStringOrNil("cf1c35b0-5185-4fca-9160-d069e7caab91"),
			chatroom.TypeNormal,
			10,
			[]*chatroom.Chatroom{
				{
					ID:         uuid.FromStringOrNil("64180150-c68f-417d-acca-d84805738ee0"),
					CustomerID: uuid.FromStringOrNil("cf1c35b0-5185-4fca-9160-d069e7caab91"),
					Type:       chatroom.TypeNormal,
					TMCreate:   "2020-04-19T03:22:17.995000",
					TMDelete:   DefaultTimeStamp,
				},
				{
					ID:         uuid.FromStringOrNil("5ace269b-63e9-4ec9-bf77-fdf0e293fcb9"),
					CustomerID: uuid.FromStringOrNil("cf1c35b0-5185-4fca-9160-d069e7caab91"),
					Type:       chatroom.TypeNormal,
					TMCreate:   "2020-04-18T03:22:17.995000",
					TMDelete:   DefaultTimeStamp,
				},
			},
		},
		{
			"type group",
			uuid.FromStringOrNil("21845d03-9343-477d-a0ee-86af9634ba6d"),
			chatroom.TypeGroup,
			10,
			[]*chatroom.Chatroom{
				{
					ID:         uuid.FromStringOrNil("c09108f6-023d-445c-875f-6ac19174090a"),
					CustomerID: uuid.FromStringOrNil("21845d03-9343-477d-a0ee-86af9634ba6d"),
					Type:       chatroom.TypeGroup,
					TMCreate:   "2020-04-19T03:22:17.995000",
					TMDelete:   DefaultTimeStamp,
				},
				{
					ID:         uuid.FromStringOrNil("18c8e1ca-d8d9-41af-b291-adb3d1b76f33"),
					CustomerID: uuid.FromStringOrNil("21845d03-9343-477d-a0ee-86af9634ba6d"),
					Type:       chatroom.TypeGroup,
					TMCreate:   "2020-04-18T03:22:17.995000",
					TMDelete:   DefaultTimeStamp,
				},
			},
		}}

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

			for _, c := range tt.chatrooms {
				mockCache.EXPECT().ChatroomSet(ctx, gomock.Any())
				if err := h.ChatroomCreate(ctx, c); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			res, err := h.ChatroomGetsByType(ctx, tt.customerID, tt.chatroomType, GetCurTime(), tt.limit)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.chatrooms) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.chatrooms, res)
			}
		})
	}
}

func Test_ChatroomGetsByChatID(t *testing.T) {

	tests := []struct {
		name      string
		chatID    uuid.UUID
		limit     uint64
		chatrooms []*chatroom.Chatroom
	}{
		{
			"type single",
			uuid.FromStringOrNil("798a20dd-a589-436d-9e6e-f497d8f18de1"),
			10,
			[]*chatroom.Chatroom{
				{
					ID:         uuid.FromStringOrNil("7296e48b-49c9-49f5-9957-f19e28540a5c"),
					CustomerID: uuid.FromStringOrNil("5602c905-5492-4e06-8c9d-4859e617fb52"),
					ChatID:     uuid.FromStringOrNil("798a20dd-a589-436d-9e6e-f497d8f18de1"),
					Type:       chatroom.TypeNormal,
					TMCreate:   "2020-04-19T03:22:17.995000",
					TMDelete:   DefaultTimeStamp,
				},
				{
					ID:         uuid.FromStringOrNil("2efb2bd9-9e85-4040-9386-f021fb165c49"),
					CustomerID: uuid.FromStringOrNil("5602c905-5492-4e06-8c9d-4859e617fb52"),
					ChatID:     uuid.FromStringOrNil("798a20dd-a589-436d-9e6e-f497d8f18de1"),
					Type:       chatroom.TypeNormal,
					TMCreate:   "2020-04-18T03:22:17.995000",
					TMDelete:   DefaultTimeStamp,
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

			for _, c := range tt.chatrooms {
				mockCache.EXPECT().ChatroomSet(ctx, gomock.Any())
				if err := h.ChatroomCreate(ctx, c); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			res, err := h.ChatroomGetsByChatID(ctx, tt.chatID, GetCurTime(), tt.limit)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.chatrooms) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.chatrooms, res)
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
				ID: uuid.FromStringOrNil("62b08684-6a3b-4416-910f-d77843d41932"),
			},

			uuid.FromStringOrNil("62b08684-6a3b-4416-910f-d77843d41932"),
			"update name",
			"update detail",

			&chatroom.Chatroom{
				ID:     uuid.FromStringOrNil("62b08684-6a3b-4416-910f-d77843d41932"),
				Name:   "update name",
				Detail: "update detail",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := &handler{
				db:    dbTest,
				cache: mockCache,
			}

			ctx := context.Background()

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
				ID: uuid.FromStringOrNil("0c1fde2b-1985-42b4-8eb3-96888e0477b5"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := &handler{
				db:    dbTest,
				cache: mockCache,
			}

			ctx := context.Background()

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
