package chathandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-talk-manager/models/chat"
	"monorepo/bin-talk-manager/models/participant"
	"monorepo/bin-talk-manager/pkg/dbhandler"
	"monorepo/bin-talk-manager/pkg/participanthandler"
)

func Test_ChatCreate(t *testing.T) {
	tests := []struct {
		name string

		customerID   uuid.UUID
		chatType     chat.Type
		participants []participant.ParticipantInput

		expectChat *chat.Chat
		expectRes  *chat.Chat
	}{
		{
			name: "normal_type_talk",

			customerID:   uuid.FromStringOrNil("ba3ad8aa-cb0d-47fe-beef-f7c76c61a9f4"),
			chatType:     chat.TypeTalk,
			participants: nil, // Talk type doesn't require participants

			expectChat: &chat.Chat{
				Identity: commonidentity.Identity{
					CustomerID: uuid.FromStringOrNil("ba3ad8aa-cb0d-47fe-beef-f7c76c61a9f4"),
				},
				Type: chat.TypeTalk,
			},
			expectRes: &chat.Chat{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("31536998-da36-11ee-976a-b31b049d62c2"),
					CustomerID: uuid.FromStringOrNil("ba3ad8aa-cb0d-47fe-beef-f7c76c61a9f4"),
				},
				Type: chat.TypeTalk,
			},
		},
		{
			name: "normal_type_direct",

			customerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
			chatType:   chat.TypeDirect,
			participants: []participant.ParticipantInput{
				{OwnerType: "agent", OwnerID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")},
			},

			expectChat: &chat.Chat{
				Identity: commonidentity.Identity{
					CustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
				},
				Type: chat.TypeDirect,
			},
			expectRes: &chat.Chat{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("ac810dc4-298c-11ee-984c-ebb7811c4114"),
					CustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
				},
				Type: chat.TypeDirect,
			},
		},
		{
			name: "normal_type_group_no_participants",

			customerID:   uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
			chatType:     chat.TypeGroup,
			participants: nil, // Group can start with just the creator

			expectChat: &chat.Chat{
				Identity: commonidentity.Identity{
					CustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
				},
				Type: chat.TypeGroup,
			},
			expectRes: &chat.Chat{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("ac810dc4-298c-11ee-984c-ebb7811c4114"),
					CustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
				},
				Type: chat.TypeGroup,
			},
		},
		{
			name: "normal_type_group_with_participants",

			customerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
			chatType:   chat.TypeGroup,
			participants: []participant.ParticipantInput{
				{OwnerType: "agent", OwnerID: uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")},
			},

			expectChat: &chat.Chat{
				Identity: commonidentity.Identity{
					CustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
				},
				Type: chat.TypeGroup,
			},
			expectRes: &chat.Chat{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("ac810dc4-298c-11ee-984c-ebb7811c4114"),
					CustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
				},
				Type: chat.TypeGroup,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockParticipant := participanthandler.NewMockParticipantHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := &chatHandler{
				dbHandler:          mockDB,
				participantHandler: mockParticipant,
				notifyHandler:      mockNotify,
				utilHandler:        mockUtil,
			}

			ctx := context.Background()

			mockUtil.EXPECT().UUIDCreate().Return(tt.expectRes.ID)
			mockDB.EXPECT().ChatCreate(ctx, gomock.Any()).Return(nil)
			mockDB.EXPECT().ChatGet(ctx, tt.expectRes.ID).Return(tt.expectRes, nil)
			mockDB.EXPECT().ParticipantListByChatIDs(ctx, gomock.Any()).Return([]*participant.Participant{}, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.customerID, chat.EventTypeChatCreated, gomock.Any())

			// Mock participant additions for each provided participant
			for range tt.participants {
				mockParticipant.EXPECT().ParticipantAdd(ctx, tt.customerID, tt.expectRes.ID, gomock.Any(), gomock.Any()).Return(&participant.Participant{}, nil)
			}

			res, err := h.ChatCreate(ctx, tt.customerID, tt.chatType, "", "", "", uuid.Nil, tt.participants)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res.CustomerID != tt.expectRes.CustomerID {
				t.Errorf("Wrong customer_id.\nexpect: %v\ngot: %v", tt.expectRes.CustomerID, res.CustomerID)
			}

			if res.Type != tt.expectRes.Type {
				t.Errorf("Wrong type.\nexpect: %v\ngot: %v", tt.expectRes.Type, res.Type)
			}

			if res.ID == uuid.Nil {
				t.Errorf("Wrong match. ID should not be nil")
			}
		})
	}
}

func Test_ChatCreate_error(t *testing.T) {
	tests := []struct {
		name string

		customerID   uuid.UUID
		chatType     chat.Type
		participants []participant.ParticipantInput

		expectError string
	}{
		{
			name: "error_nil_customer_id",

			customerID:   uuid.Nil,
			chatType:     chat.TypeTalk,
			participants: nil,

			expectError: "customer ID cannot be nil",
		},
		{
			name: "error_invalid_type",

			customerID:   uuid.FromStringOrNil("ba3ad8aa-cb0d-47fe-beef-f7c76c61a9f4"),
			chatType:     "invalid_type",
			participants: nil,

			expectError: "invalid chat type",
		},
		{
			name: "error_direct_no_participant",

			customerID:   uuid.FromStringOrNil("ba3ad8aa-cb0d-47fe-beef-f7c76c61a9f4"),
			chatType:     chat.TypeDirect,
			participants: nil,

			expectError: "direct chat requires exactly 1 other participant",
		},
		{
			name: "error_direct_too_many_participants",

			customerID: uuid.FromStringOrNil("ba3ad8aa-cb0d-47fe-beef-f7c76c61a9f4"),
			chatType:   chat.TypeDirect,
			participants: []participant.ParticipantInput{
				{OwnerType: "agent", OwnerID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")},
				{OwnerType: "agent", OwnerID: uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")},
			},

			expectError: "direct chat requires exactly 1 other participant",
		},
		{
			name: "error_database_failure",

			customerID:   uuid.FromStringOrNil("ba3ad8aa-cb0d-47fe-beef-f7c76c61a9f4"),
			chatType:     chat.TypeTalk,
			participants: nil,

			expectError: "failed to create chat in database",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockParticipant := participanthandler.NewMockParticipantHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := &chatHandler{
				dbHandler:          mockDB,
				participantHandler: mockParticipant,
				notifyHandler:      mockNotify,
				utilHandler:        mockUtil,
			}

			ctx := context.Background()

			// Mock UUID creation for valid paths
			mockUtil.EXPECT().UUIDCreate().Return(uuid.FromStringOrNil("ac810dc4-298c-11ee-984c-ebb7811c4114")).AnyTimes()

			// Only mock database call for database failure test
			if tt.name == "error_database_failure" {
				mockDB.EXPECT().ChatCreate(ctx, gomock.Any()).Return(fmt.Errorf("database error"))
			}

			res, err := h.ChatCreate(ctx, tt.customerID, tt.chatType, "", "", "", uuid.Nil, tt.participants)
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: ok")
			}

			if res != nil {
				t.Errorf("Wrong match. expect: nil result, got: %v", res)
			}
		})
	}
}

func Test_ChatGet(t *testing.T) {
	tests := []struct {
		name string

		id uuid.UUID

		responseChat *chat.Chat
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("e8427fa8-17b2-4e9e-8855-90e516bcf1d3"),

			responseChat: &chat.Chat{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("e8427fa8-17b2-4e9e-8855-90e516bcf1d3"),
					CustomerID: uuid.FromStringOrNil("809656e2-305e-43cd-8d7b-ccb44373dddb"),
				},
				Type:     chat.TypeDirect,
				TMCreate: "2024-01-15 10:30:00.000000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &chatHandler{
				dbHandler:     mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().ChatGet(ctx, tt.id).Return(tt.responseChat, nil)
			mockDB.EXPECT().ParticipantListByChatIDs(ctx, gomock.Any()).Return([]*participant.Participant{}, nil)

			res, err := h.ChatGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseChat) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseChat, res)
			}
		})
	}
}

func Test_ChatGet_error(t *testing.T) {
	tests := []struct {
		name string

		id uuid.UUID

		dbError error
	}{
		{
			name: "error_not_found",

			id: uuid.FromStringOrNil("e8427fa8-17b2-4e9e-8855-90e516bcf1d3"),

			dbError: fmt.Errorf("not found"),
		},
		{
			name: "error_database_failure",

			id: uuid.FromStringOrNil("62b0e2b7-0583-4f78-9406-45b00d17a9b4"),

			dbError: fmt.Errorf("database connection error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &chatHandler{
				dbHandler:     mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().ChatGet(ctx, tt.id).Return(nil, tt.dbError)

			res, err := h.ChatGet(ctx, tt.id)
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: ok")
			}

			if res != nil {
				t.Errorf("Wrong match. expect: nil result, got: %v", res)
			}
		})
	}
}

func Test_ChatList(t *testing.T) {
	tests := []struct {
		name string

		filters map[chat.Field]any
		token   string
		size    uint64

		responseTalks []*chat.Chat
	}{
		{
			name: "normal",

			filters: map[chat.Field]any{
				chat.FieldCustomerID: uuid.FromStringOrNil("809656e2-305e-43cd-8d7b-ccb44373dddb"),
				chat.FieldDeleted:    false,
			},
			token: "2024-01-15 10:30:00.000000",
			size:  10,

			responseTalks: []*chat.Chat{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("e8427fa8-17b2-4e9e-8855-90e516bcf1d3"),
						CustomerID: uuid.FromStringOrNil("809656e2-305e-43cd-8d7b-ccb44373dddb"),
					},
					Type:     chat.TypeDirect,
					TMCreate: "2024-01-15 10:30:00.000000",
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("04bc94c1-9cc1-4ce8-8559-39d6f1892109"),
						CustomerID: uuid.FromStringOrNil("809656e2-305e-43cd-8d7b-ccb44373dddb"),
					},
					Type:     chat.TypeGroup,
					TMCreate: "2024-01-15 11:00:00.000000",
				},
			},
		},
		{
			name: "normal_with_type_filter",

			filters: map[chat.Field]any{
				chat.FieldCustomerID: uuid.FromStringOrNil("ba3ad8aa-cb0d-47fe-beef-f7c76c61a9f4"),
				chat.FieldType:       chat.TypeGroup,
				chat.FieldDeleted:    false,
			},
			token: "",
			size:  50,

			responseTalks: []*chat.Chat{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("6a9a0ed0-1bcb-46de-a225-e638bbaf2fc1"),
						CustomerID: uuid.FromStringOrNil("ba3ad8aa-cb0d-47fe-beef-f7c76c61a9f4"),
					},
					Type:     chat.TypeGroup,
					TMCreate: "2024-01-15 12:00:00.000000",
				},
			},
		},
		{
			name: "normal_empty_result",

			filters: map[chat.Field]any{
				chat.FieldCustomerID: uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
				chat.FieldDeleted:    false,
			},
			token: "",
			size:  10,

			responseTalks: []*chat.Chat{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &chatHandler{
				dbHandler:     mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().ChatList(ctx, tt.filters, tt.token, tt.size).Return(tt.responseTalks, nil)

			res, err := h.ChatList(ctx, tt.filters, tt.token, tt.size)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseTalks) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseTalks, res)
			}
		})
	}
}

func Test_ChatList_error(t *testing.T) {
	tests := []struct {
		name string

		filters map[chat.Field]any
		token   string
		size    uint64

		dbError error
	}{
		{
			name: "error_database_failure",

			filters: map[chat.Field]any{
				chat.FieldCustomerID: uuid.FromStringOrNil("809656e2-305e-43cd-8d7b-ccb44373dddb"),
			},
			token: "",
			size:  10,

			dbError: fmt.Errorf("database connection error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &chatHandler{
				dbHandler:     mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().ChatList(ctx, tt.filters, tt.token, tt.size).Return(nil, tt.dbError)

			res, err := h.ChatList(ctx, tt.filters, tt.token, tt.size)
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: ok")
			}

			if res != nil {
				t.Errorf("Wrong match. expect: nil result, got: %v", res)
			}
		})
	}
}

func Test_ChatDelete(t *testing.T) {
	tests := []struct {
		name string

		id uuid.UUID

		responseChat *chat.Chat
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("af243cbc-de04-4705-ad2b-78350d0a4fba"),

			responseChat: &chat.Chat{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("af243cbc-de04-4705-ad2b-78350d0a4fba"),
					CustomerID: uuid.FromStringOrNil("809656e2-305e-43cd-8d7b-ccb44373dddb"),
				},
				Type:     chat.TypeDirect,
				TMCreate: "2024-01-15 10:30:00.000000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &chatHandler{
				dbHandler:     mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().ChatGet(ctx, tt.id).Return(tt.responseChat, nil)
			mockDB.EXPECT().ChatDelete(ctx, tt.id).Return(nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseChat.CustomerID, chat.EventTypeChatDeleted, tt.responseChat)

			res, err := h.ChatDelete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseChat) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseChat, res)
			}
		})
	}
}

func Test_ChatDelete_error(t *testing.T) {
	tests := []struct {
		name string

		id uuid.UUID

		getTalkError    error
		deleteTalkError error
	}{
		{
			name: "error_get_before_delete_failed",

			id: uuid.FromStringOrNil("af243cbc-de04-4705-ad2b-78350d0a4fba"),

			getTalkError:    fmt.Errorf("not found"),
			deleteTalkError: nil,
		},
		{
			name: "error_delete_failed",

			id: uuid.FromStringOrNil("62b0e2b7-0583-4f78-9406-45b00d17a9b4"),

			getTalkError:    nil,
			deleteTalkError: fmt.Errorf("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &chatHandler{
				dbHandler:     mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			if tt.getTalkError != nil {
				// Get before delete fails
				mockDB.EXPECT().ChatGet(ctx, tt.id).Return(nil, tt.getTalkError)
			} else {
				// Get succeeds but delete fails
				responseChat := &chat.Chat{
					Identity: commonidentity.Identity{
						ID:         tt.id,
						CustomerID: uuid.FromStringOrNil("809656e2-305e-43cd-8d7b-ccb44373dddb"),
					},
					Type: chat.TypeDirect,
				}
				mockDB.EXPECT().ChatGet(ctx, tt.id).Return(responseChat, nil)
				mockDB.EXPECT().ChatDelete(ctx, tt.id).Return(tt.deleteTalkError)
			}

			res, err := h.ChatDelete(ctx, tt.id)
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: ok")
			}

			if res != nil {
				t.Errorf("Wrong match. expect: nil result, got: %v", res)
			}
		})
	}
}
