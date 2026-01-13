package chathandler

import (
	"context"
	reflect "reflect"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-chat-manager/models/chat"
	"monorepo/bin-chat-manager/models/chatroom"
	"monorepo/bin-chat-manager/pkg/chatroomhandler"
	"monorepo/bin-chat-manager/pkg/dbhandler"
)

func Test_Get(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseChat *chat.Chat
	}{
		{
			"normal",

			uuid.FromStringOrNil("e8427fa8-17b2-4e9e-8855-90e516bcf1d3"),

			&chat.Chat{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("e8427fa8-17b2-4e9e-8855-90e516bcf1d3"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &chatHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().ChatGet(ctx, tt.id).Return(tt.responseChat, nil)

			res, err := h.Get(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.responseChat) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.responseChat, res)
			}
		})
	}
}

func Test_GetsByCustomerID(t *testing.T) {

	tests := []struct {
		name string

		token   string
		limit   uint64
		filters map[chat.Field]any

		responseChat []*chat.Chat
	}{
		{
			"normal",

			"2022-04-18 03:22:17.995000",
			10,
			map[chat.Field]any{
				chat.FieldCustomerID: uuid.FromStringOrNil("809656e2-305e-43cd-8d7b-ccb44373dddb"),
				chat.FieldDeleted:    false,
			},

			[]*chat.Chat{
				{
					Identity: commonidentity.Identity{
						CustomerID: uuid.FromStringOrNil("809656e2-305e-43cd-8d7b-ccb44373dddb"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &chatHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().ChatGets(ctx, tt.token, tt.limit, tt.filters).Return(tt.responseChat, nil)

			res, err := h.Gets(ctx, tt.token, tt.limit, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.responseChat) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.responseChat, res)
			}
		})
	}
}

func Test_create(t *testing.T) {

	tests := []struct {
		name string

		customerID     uuid.UUID
		chatType       chat.Type
		ownerID        uuid.UUID
		participantIDs []uuid.UUID
		chatName       string
		detail         string

		responseUUID uuid.UUID
		responseChat *chat.Chat
	}{
		{
			"normal",

			uuid.FromStringOrNil("ba3ad8aa-cb0d-47fe-beef-f7c76c61a9f4"),
			chat.TypeNormal,
			uuid.FromStringOrNil("fbfd1fe6-bde3-4fa2-9f7b-e460807100a6"),
			[]uuid.UUID{
				uuid.FromStringOrNil("d8e0c187-44c8-4376-abab-2831f1754f5d"),
				uuid.FromStringOrNil("fbfd1fe6-bde3-4fa2-9f7b-e460807100a6"),
			},
			"test name",
			"test detail",

			uuid.FromStringOrNil("31536998-da36-11ee-976a-b31b049d62c2"),
			&chat.Chat{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("04bc94c1-9cc1-4ce8-8559-39d6f1892109"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &chatHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockUtil.EXPECT().TimeGetCurTime().Return(utilhandler.TimeGetCurTime())
			mockDB.EXPECT().ChatCreate(ctx, gomock.Any()).Return(nil)
			mockDB.EXPECT().ChatGet(ctx, gomock.Any()).Return(tt.responseChat, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseChat.CustomerID, chat.EventTypeChatCreated, tt.responseChat)

			res, err := h.create(ctx, tt.customerID, tt.chatType, tt.ownerID, tt.participantIDs, tt.chatName, tt.detail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.responseChat) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.responseChat, res)
			}
		})
	}
}

func Test_Create(t *testing.T) {

	tests := []struct {
		name string

		customerID     uuid.UUID
		chatType       chat.Type
		ownerID        uuid.UUID
		participantIDs []uuid.UUID
		chatName       string
		detail         string

		responseUUID uuid.UUID
		responseChat *chat.Chat

		expectFilters map[chat.Field]any
	}{
		{
			"normal",

			uuid.FromStringOrNil("ba3ad8aa-cb0d-47fe-beef-f7c76c61a9f4"),
			chat.TypeNormal,
			uuid.FromStringOrNil("fbfd1fe6-bde3-4fa2-9f7b-e460807100a6"),
			[]uuid.UUID{
				uuid.FromStringOrNil("d8e0c187-44c8-4376-abab-2831f1754f5d"),
				uuid.FromStringOrNil("fbfd1fe6-bde3-4fa2-9f7b-e460807100a6"),
			},
			"test name",
			"test detail",

			uuid.FromStringOrNil("05edc666-da33-11ee-b970-ffcf954442da"),
			&chat.Chat{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("04bc94c1-9cc1-4ce8-8559-39d6f1892109"),
				},
			},

			map[chat.Field]any{
				chat.FieldCustomerID:     uuid.FromStringOrNil("ba3ad8aa-cb0d-47fe-beef-f7c76c61a9f4"),
				chat.FieldDeleted:        false,
				chat.FieldParticipantIDs: "d8e0c187-44c8-4376-abab-2831f1754f5d,fbfd1fe6-bde3-4fa2-9f7b-e460807100a6",
				chat.FieldType:           chat.TypeNormal,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockChatroom := chatroomhandler.NewMockChatroomHandler(mc)

			h := &chatHandler{
				utilHandler:     mockUtil,
				db:              mockDB,
				reqHandler:      mockReq,
				notifyHandler:   mockNotify,
				chatroomHandler: mockChatroom,
			}
			ctx := context.Background()

			mockDB.EXPECT().ChatGets(ctx, gomock.Any(), gomock.Any(), tt.expectFilters).Return([]*chat.Chat{}, nil)
			for _, participantID := range tt.participantIDs {
				tmp := &amagent.Agent{
					Identity: commonidentity.Identity{
						ID:         participantID,
						CustomerID: tt.customerID,
					},
				}
				mockReq.EXPECT().AgentV1AgentGet(ctx, participantID).Return(tmp, nil)
			}

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockUtil.EXPECT().TimeGetCurTime().Return(utilhandler.TimeGetCurTime())
			mockDB.EXPECT().ChatCreate(ctx, gomock.Any()).Return(nil)
			mockDB.EXPECT().ChatGet(ctx, gomock.Any()).Return(tt.responseChat, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseChat.CustomerID, chat.EventTypeChatCreated, tt.responseChat)

			chatroomType := chatroom.ConvertType(tt.chatType)
			for _, participantID := range tt.participantIDs {
				mockChatroom.EXPECT().Create(
					ctx,
					tt.customerID,
					participantID,
					chatroomType,
					tt.responseChat.ID,
					participantID,
					tt.participantIDs,
					tt.chatName,
					tt.detail,
				).Return(&chatroom.Chatroom{}, nil)
			}

			res, err := h.Create(ctx, tt.customerID, tt.chatType, tt.ownerID, tt.participantIDs, tt.chatName, tt.detail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.responseChat) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.responseChat, res)
			}
		})
	}
}

func Test_UpdateBasicInfo(t *testing.T) {

	tests := []struct {
		name string

		id       uuid.UUID
		chatName string
		detail   string

		responseChat *chat.Chat
	}{
		{
			"normal",

			uuid.FromStringOrNil("62b0e2b7-0583-4f78-9406-45b00d17a9b4"),
			"update name",
			"update detail",

			&chat.Chat{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("62b0e2b7-0583-4f78-9406-45b00d17a9b4"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &chatHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().ChatUpdateBasicInfo(ctx, tt.id, tt.chatName, tt.detail).Return(nil)
			mockDB.EXPECT().ChatGet(ctx, tt.responseChat.ID).Return(tt.responseChat, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseChat.CustomerID, chat.EventTypeChatUpdated, tt.responseChat)

			res, err := h.UpdateBasicInfo(ctx, tt.id, tt.chatName, tt.detail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.responseChat) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.responseChat, res)
			}
		})
	}
}

func Test_UpdateRoomOwnerID(t *testing.T) {

	tests := []struct {
		name string

		id      uuid.UUID
		ownerID uuid.UUID

		responseChat *chat.Chat
	}{
		{
			"normal",

			uuid.FromStringOrNil("6a9a0ed0-1bcb-46de-a225-e638bbaf2fc1"),
			uuid.FromStringOrNil("41b0f472-da9f-4a2d-8729-d456686d3930"),

			&chat.Chat{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("6a9a0ed0-1bcb-46de-a225-e638bbaf2fc1"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &chatHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().ChatUpdateRoomOwnerID(ctx, tt.id, tt.ownerID).Return(nil)
			mockDB.EXPECT().ChatGet(ctx, tt.responseChat.ID).Return(tt.responseChat, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseChat.CustomerID, chat.EventTypeChatUpdated, tt.responseChat)

			res, err := h.UpdateRoomOwnerID(ctx, tt.id, tt.ownerID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.responseChat) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.responseChat, res)
			}
		})
	}
}

func Test_addParticipantID(t *testing.T) {

	tests := []struct {
		name string

		id            uuid.UUID
		participantID uuid.UUID

		responseChat *chat.Chat

		expectParticipantIDs []uuid.UUID
	}{
		{
			"normal",

			uuid.FromStringOrNil("02d7e497-8825-4f80-934e-cb01d93270e9"),
			uuid.FromStringOrNil("2cd8e3b8-200c-4ab3-a9d2-b14788d3e41d"),

			&chat.Chat{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("02d7e497-8825-4f80-934e-cb01d93270e9"),
				},
				ParticipantIDs: []uuid.UUID{
					uuid.FromStringOrNil("e4391882-b957-11ee-aaa9-bb3cdf1e2651"),
					uuid.FromStringOrNil("e4610d9c-b957-11ee-91f3-07311eda30a4"),
					uuid.FromStringOrNil("e494310e-b957-11ee-ac35-876aad11c488"),
				},
			},

			[]uuid.UUID{
				uuid.FromStringOrNil("e4391882-b957-11ee-aaa9-bb3cdf1e2651"),
				uuid.FromStringOrNil("e4610d9c-b957-11ee-91f3-07311eda30a4"),
				uuid.FromStringOrNil("e494310e-b957-11ee-ac35-876aad11c488"),
				uuid.FromStringOrNil("2cd8e3b8-200c-4ab3-a9d2-b14788d3e41d"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &chatHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().ChatGet(ctx, tt.responseChat.ID).Return(tt.responseChat, nil)
			mockDB.EXPECT().ChatUpdateParticipantID(ctx, tt.id, tt.expectParticipantIDs).Return(nil)
			mockDB.EXPECT().ChatGet(ctx, tt.responseChat.ID).Return(tt.responseChat, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseChat.CustomerID, chat.EventTypeChatUpdated, tt.responseChat)

			res, err := h.addParticipantID(ctx, tt.id, tt.participantID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.responseChat) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.responseChat, res)
			}
		})
	}
}

func Test_removeParticipantID(t *testing.T) {

	tests := []struct {
		name string

		id            uuid.UUID
		participantID uuid.UUID

		responseChat *chat.Chat

		expectParticipantID []uuid.UUID
	}{
		{
			"normal",

			uuid.FromStringOrNil("61a94935-5fd4-4091-ab23-49ebc69b9d66"),
			uuid.FromStringOrNil("5f16f38b-5d9d-40af-94c2-2bd50d939c28"),

			&chat.Chat{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("61a94935-5fd4-4091-ab23-49ebc69b9d66"),
				},
				ParticipantIDs: []uuid.UUID{
					uuid.FromStringOrNil("3bbc8f4a-b957-11ee-b921-8f4469c59011"),
					uuid.FromStringOrNil("5f16f38b-5d9d-40af-94c2-2bd50d939c28"),
				},
			},

			[]uuid.UUID{
				uuid.FromStringOrNil("3bbc8f4a-b957-11ee-b921-8f4469c59011"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &chatHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().ChatGet(ctx, tt.responseChat.ID).Return(tt.responseChat, nil)
			mockDB.EXPECT().ChatUpdateParticipantID(ctx, tt.id, tt.expectParticipantID).Return(nil)
			mockDB.EXPECT().ChatGet(ctx, tt.responseChat.ID).Return(tt.responseChat, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseChat.CustomerID, chat.EventTypeChatUpdated, tt.responseChat)

			res, err := h.removeParticipantID(ctx, tt.id, tt.participantID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.responseChat) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.responseChat, res)
			}
		})
	}
}

func Test_Delete(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseChat *chat.Chat
	}{
		{
			"normal",

			uuid.FromStringOrNil("af243cbc-de04-4705-ad2b-78350d0a4fba"),

			&chat.Chat{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("af243cbc-de04-4705-ad2b-78350d0a4fba"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &chatHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().ChatDelete(ctx, tt.id).Return(nil)
			mockDB.EXPECT().ChatGet(ctx, tt.responseChat.ID).Return(tt.responseChat, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseChat.CustomerID, chat.EventTypeChatDeleted, tt.responseChat)

			res, err := h.Delete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.responseChat) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.responseChat, res)
			}
		})
	}
}

func Test_AddParticipantID(t *testing.T) {

	tests := []struct {
		name string

		id            uuid.UUID
		participantID uuid.UUID

		responseChat      *chat.Chat
		responseChatrooms []*chatroom.Chatroom

		expectFilters        map[chatroom.Field]any
		expectParticipantIDs []uuid.UUID
	}{
		{
			"normal",

			uuid.FromStringOrNil("2442834a-312e-11ed-8306-87672e5154fb"),
			uuid.FromStringOrNil("253abd3a-312e-11ed-9393-0ff58ef4c53f"),

			&chat.Chat{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2442834a-312e-11ed-8306-87672e5154fb"),
				},
				ParticipantIDs: []uuid.UUID{
					uuid.FromStringOrNil("612a6f48-312e-11ed-ac1d-ab725d46bc95"),
				},
			},
			[]*chatroom.Chatroom{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("02e9531c-312f-11ed-989c-cf70d99dbb1e"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("0343ca40-312f-11ed-8b56-971508921afc"),
					},
				},
			},

			map[chatroom.Field]any{
				chatroom.FieldDeleted: false,
				chatroom.FieldChatID:  uuid.FromStringOrNil("2442834a-312e-11ed-8306-87672e5154fb"),
			},
			[]uuid.UUID{
				uuid.FromStringOrNil("612a6f48-312e-11ed-ac1d-ab725d46bc95"),
				uuid.FromStringOrNil("253abd3a-312e-11ed-9393-0ff58ef4c53f"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockChatroom := chatroomhandler.NewMockChatroomHandler(mc)

			h := &chatHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,

				chatroomHandler: mockChatroom,
			}
			ctx := context.Background()

			// addParticipantID
			mockDB.EXPECT().ChatGet(ctx, tt.responseChat.ID).Return(tt.responseChat, nil)
			mockDB.EXPECT().ChatUpdateParticipantID(ctx, tt.id, tt.expectParticipantIDs).Return(nil)
			mockDB.EXPECT().ChatGet(ctx, tt.responseChat.ID).Return(tt.responseChat, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseChat.CustomerID, chat.EventTypeChatUpdated, tt.responseChat)

			mockChatroom.EXPECT().Gets(ctx, gomock.Any(), gomock.Any(), tt.expectFilters).Return(tt.responseChatrooms, nil)
			for _, cr := range tt.responseChatrooms {
				mockChatroom.EXPECT().AddParticipantID(ctx, cr.ID, tt.participantID).Return(&chatroom.Chatroom{}, nil)
			}

			mockChatroom.EXPECT().Create(
				ctx,
				tt.responseChat.CustomerID,
				tt.participantID,
				gomock.Any(),
				tt.responseChat.ID,
				tt.participantID,
				tt.responseChat.ParticipantIDs,
				tt.responseChat.Name,
				tt.responseChat.Detail,
			).Return(&chatroom.Chatroom{}, nil)

			mockUtil.EXPECT().TimeGetCurTime().Return(utilhandler.TimeGetCurTime())
			res, err := h.AddParticipantID(ctx, tt.id, tt.participantID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.responseChat) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.responseChat, res)
			}
		})
	}
}

func Test_RemoveParticipantID(t *testing.T) {

	tests := []struct {
		name string

		id            uuid.UUID
		participantID uuid.UUID

		responseChat      *chat.Chat
		responseChatrooms []*chatroom.Chatroom

		expectFilters        map[chatroom.Field]any
		expectParticipantIDs []uuid.UUID
	}{
		{
			"normal",

			uuid.FromStringOrNil("25d04f5c-3134-11ed-ac20-6f4780413d87"),
			uuid.FromStringOrNil("25fa6e86-3134-11ed-be21-27cdde31883c"),

			&chat.Chat{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("25d04f5c-3134-11ed-ac20-6f4780413d87"),
				},
				ParticipantIDs: []uuid.UUID{
					uuid.FromStringOrNil("2622af18-3134-11ed-9fde-a709c229f85c"),
					uuid.FromStringOrNil("25fa6e86-3134-11ed-be21-27cdde31883c"),
				},
			},
			[]*chatroom.Chatroom{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("264e8764-3134-11ed-9f9d-e3e2f588f17a"),
					},
					RoomOwnerID: uuid.FromStringOrNil("25fa6e86-3134-11ed-be21-27cdde31883c"),
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("336e3386-3134-11ed-b1df-e38814f71100"),
					},
				},
			},

			map[chatroom.Field]any{
				chatroom.FieldDeleted: false,
				chatroom.FieldChatID:  uuid.FromStringOrNil("25d04f5c-3134-11ed-ac20-6f4780413d87"),
			},
			[]uuid.UUID{
				uuid.FromStringOrNil("2622af18-3134-11ed-9fde-a709c229f85c"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockChatroom := chatroomhandler.NewMockChatroomHandler(mc)

			h := &chatHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,

				chatroomHandler: mockChatroom,
			}

			ctx := context.Background()

			// removeParticipantID
			mockDB.EXPECT().ChatGet(ctx, tt.responseChat.ID).Return(tt.responseChat, nil)
			mockDB.EXPECT().ChatUpdateParticipantID(ctx, tt.id, tt.expectParticipantIDs).Return(nil)
			mockDB.EXPECT().ChatGet(ctx, tt.responseChat.ID).Return(tt.responseChat, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseChat.CustomerID, chat.EventTypeChatUpdated, tt.responseChat)

			mockChatroom.EXPECT().Gets(ctx, gomock.Any(), gomock.Any(), tt.expectFilters).Return(tt.responseChatrooms, nil)
			chatroomID := uuid.Nil
			for _, cr := range tt.responseChatrooms {
				if cr.RoomOwnerID == tt.participantID {
					chatroomID = cr.ID
				}
				mockChatroom.EXPECT().RemoveParticipantID(ctx, cr.ID, tt.participantID).Return(&chatroom.Chatroom{}, nil)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(utilhandler.TimeGetCurTime())
			mockChatroom.EXPECT().Delete(ctx, chatroomID).Return(&chatroom.Chatroom{}, nil)

			res, err := h.RemoveParticipantID(ctx, tt.id, tt.participantID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.responseChat) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.responseChat, res)
			}
		})
	}
}
