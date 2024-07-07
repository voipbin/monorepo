package chatroomhandler

import (
	"context"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"monorepo/bin-chat-manager/models/chatroom"
	"monorepo/bin-chat-manager/pkg/dbhandler"
)

func Test_Get(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseChatroom *chatroom.Chatroom
	}{
		{
			"normal",

			uuid.FromStringOrNil("d1a339b0-900a-49d0-8c93-88bc29774123"),

			&chatroom.Chatroom{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("d1a339b0-900a-49d0-8c93-88bc29774123"),
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

			h := &chatroomHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().ChatroomGet(ctx, tt.id).Return(tt.responseChatroom, nil)

			res, err := h.Get(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.responseChatroom) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.responseChatroom, res)
			}
		})
	}
}

func Test_Create(t *testing.T) {

	tests := []struct {
		name string

		customerID     uuid.UUID
		agentID        uuid.UUID
		chatroomType   chatroom.Type
		chatID         uuid.UUID
		ownerID        uuid.UUID
		participantIDs []uuid.UUID
		chatName       string
		detail         string

		responseChatroom *chatroom.Chatroom
	}{
		{
			"normal",

			uuid.FromStringOrNil("8eb9ef06-8c04-441f-b985-96196ed2b437"),
			uuid.FromStringOrNil("3446f894-da32-11ee-9bf8-d775a4655080"),
			chatroom.TypeNormal,
			uuid.FromStringOrNil("4f699231-9e6c-4006-a6d4-ebc13d576bef"),
			uuid.FromStringOrNil("cfbd68c6-0a7a-40e0-83a3-0c82ca60479c"),
			[]uuid.UUID{
				uuid.FromStringOrNil("d705a658-a475-461b-a953-dc7074d8bb4d"),
				uuid.FromStringOrNil("cfbd68c6-0a7a-40e0-83a3-0c82ca60479c"),
			},
			"test name",
			"test detail",

			&chatroom.Chatroom{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("352b4a77-49e4-4f8f-814a-7d7c3e14679c"),
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

			h := &chatroomHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(utilhandler.TimeGetCurTime())
			mockDB.EXPECT().ChatroomCreate(ctx, gomock.Any()).Return(nil)
			mockDB.EXPECT().ChatroomGet(ctx, gomock.Any()).Return(tt.responseChatroom, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseChatroom.CustomerID, chatroom.EventTypeChatroomCreated, tt.responseChatroom)

			res, err := h.Create(ctx, tt.customerID, tt.agentID, tt.chatroomType, tt.chatID, tt.ownerID, tt.participantIDs, tt.chatName, tt.detail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.responseChatroom) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.responseChatroom, res)
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

		responseChatroom *chatroom.Chatroom
	}{
		{
			"normal",

			uuid.FromStringOrNil("2a6aa7be-7576-4351-93c1-cd125713334d"),
			"update name",
			"update detail",

			&chatroom.Chatroom{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a6aa7be-7576-4351-93c1-cd125713334d"),
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

			h := &chatroomHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().ChatroomUpdateBasicInfo(ctx, tt.id, tt.chatName, tt.detail).Return(nil)
			mockDB.EXPECT().ChatroomGet(ctx, tt.responseChatroom.ID).Return(tt.responseChatroom, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseChatroom.CustomerID, chatroom.EventTypeChatroomUpdated, tt.responseChatroom)

			res, err := h.UpdateBasicInfo(ctx, tt.id, tt.chatName, tt.detail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.responseChatroom) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.responseChatroom, res)
			}
		})
	}
}

func Test_AddParticipantID(t *testing.T) {

	tests := []struct {
		name string

		id            uuid.UUID
		participantID uuid.UUID

		responseChatroom *chatroom.Chatroom
	}{
		{
			"normal",

			uuid.FromStringOrNil("76ecc5c6-0fda-4a37-9626-9baa11804ed1"),
			uuid.FromStringOrNil("0a85ad34-4f3c-4bbe-bad7-b6239219719c"),

			&chatroom.Chatroom{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("76ecc5c6-0fda-4a37-9626-9baa11804ed1"),
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

			h := &chatroomHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().ChatroomAddParticipantID(ctx, tt.id, tt.participantID).Return(nil)
			mockDB.EXPECT().ChatroomGet(ctx, tt.responseChatroom.ID).Return(tt.responseChatroom, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseChatroom.CustomerID, chatroom.EventTypeChatroomUpdated, tt.responseChatroom)

			res, err := h.AddParticipantID(ctx, tt.id, tt.participantID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.responseChatroom) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.responseChatroom, res)
			}
		})
	}
}

func Test_RemoveParticipantID(t *testing.T) {

	tests := []struct {
		name string

		id            uuid.UUID
		participantID uuid.UUID

		responseChatroom *chatroom.Chatroom
	}{
		{
			"normal",

			uuid.FromStringOrNil("82c31fde-2b70-4f50-a336-fdb2422fa202"),
			uuid.FromStringOrNil("d7220b09-8759-489a-97ed-f5875e92fb0c"),

			&chatroom.Chatroom{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("82c31fde-2b70-4f50-a336-fdb2422fa202"),
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

			h := &chatroomHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().ChatroomRemoveParticipantID(ctx, tt.id, tt.participantID).Return(nil)
			mockDB.EXPECT().ChatroomGet(ctx, tt.responseChatroom.ID).Return(tt.responseChatroom, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseChatroom.CustomerID, chatroom.EventTypeChatroomUpdated, tt.responseChatroom)

			res, err := h.RemoveParticipantID(ctx, tt.id, tt.participantID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.responseChatroom) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.responseChatroom, res)
			}
		})
	}
}

func Test_Delete(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseChatroom *chatroom.Chatroom
	}{
		{
			"normal",

			uuid.FromStringOrNil("da0836f4-e25d-4340-b4b4-59399b26ad11"),

			&chatroom.Chatroom{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("da0836f4-e25d-4340-b4b4-59399b26ad11"),
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

			h := &chatroomHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().ChatroomDelete(ctx, tt.id).Return(nil)
			mockDB.EXPECT().ChatroomGet(ctx, tt.responseChatroom.ID).Return(tt.responseChatroom, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseChatroom.CustomerID, chatroom.EventTypeChatroomDeleted, tt.responseChatroom)

			res, err := h.Delete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.responseChatroom) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.responseChatroom, res)
			}
		})
	}
}
