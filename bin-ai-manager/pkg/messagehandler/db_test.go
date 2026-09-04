package messagehandler

import (
	"context"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_Create(t *testing.T) {

	tests := []struct {
		name string

		customerID   uuid.UUID
		aicallID     uuid.UUID
		activeflowID uuid.UUID
		activeAIID   uuid.UUID
		direction    message.Direction
		role       message.Role
		content    string
		toolCalls  []message.ToolCall
		toolCallID string

		responseUUID uuid.UUID

		expectMessage *message.Message
	}{
		{
			name: "have all",

			customerID:   uuid.FromStringOrNil("f227397c-f260-11ef-b217-4f6ff6930cf2"),
			aicallID:     uuid.FromStringOrNil("f26fd614-f260-11ef-ae2f-ab1a2508e20d"),
			activeflowID: uuid.FromStringOrNil("a1b2c3d4-e5f6-7890-abcd-ef1234567890"),
			activeAIID:   uuid.FromStringOrNil("aabbccdd-1234-5678-abcd-ef1234567890"),
			direction:    message.DirectionIncoming,
			role:       message.RoleUser,
			content:    "Hello, world!",
			toolCalls: []message.ToolCall{
				{
					ID:   "62bfd2da-943b-11f0-9375-c711ec2159d9",
					Type: message.ToolTypeFunction,
					Function: message.FunctionCall{
						Name:      "get_current_weather",
						Arguments: `{"location": "Boston, MA", "unit": "celsius"}`,
					},
				},
			},
			toolCallID: "62ed2280-943b-11f0-b762-4f0b5a0bd115",

			responseUUID: uuid.FromStringOrNil("751956c2-8482-11f0-846a-c71f69f8c722"),

			expectMessage: &message.Message{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("751956c2-8482-11f0-846a-c71f69f8c722"),
					CustomerID: uuid.FromStringOrNil("f227397c-f260-11ef-b217-4f6ff6930cf2"),
				},
				AIcallID:     uuid.FromStringOrNil("f26fd614-f260-11ef-ae2f-ab1a2508e20d"),
				ActiveflowID: uuid.FromStringOrNil("a1b2c3d4-e5f6-7890-abcd-ef1234567890"),
				ActiveAIID:   uuid.FromStringOrNil("aabbccdd-1234-5678-abcd-ef1234567890"),

				Direction: message.DirectionIncoming,
				Role:      message.RoleUser,
				Content:   "Hello, world!",
				ToolCalls: []message.ToolCall{
					{
						ID:   "62bfd2da-943b-11f0-9375-c711ec2159d9",
						Type: message.ToolTypeFunction,
						Function: message.FunctionCall{
							Name:      "get_current_weather",
							Arguments: `{"location": "Boston, MA", "unit": "celsius"}`,
						},
					},
				},
				ToolCallID: "62ed2280-943b-11f0-b762-4f0b5a0bd115",

				DeliveryStatus: message.DeliveryStatusDelivered,
			},
		},
		{
			name: "empty",

			responseUUID: uuid.FromStringOrNil("0812955a-f262-11ef-a3a2-1bee273dee65"),

			expectMessage: &message.Message{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("0812955a-f262-11ef-a3a2-1bee273dee65"),
				},
				ToolCalls:      []message.ToolCall{},
				DeliveryStatus: message.DeliveryStatusDelivered,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := messageHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDB.EXPECT().MessageCreate(ctx, tt.expectMessage).Return(nil)
			mockDB.EXPECT().MessageGet(ctx, tt.responseUUID).Return(tt.expectMessage, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.expectMessage.CustomerID, message.EventTypeMessageCreated, tt.expectMessage)

			opts := []CreateOption{}
			if tt.activeAIID != uuid.Nil {
				opts = append(opts, WithActiveAIID(tt.activeAIID))
			}
			res, err := h.Create(ctx, uuid.Nil, tt.customerID, tt.aicallID, tt.activeflowID, tt.direction, tt.role, tt.content, tt.toolCalls, tt.toolCallID, opts...)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectMessage) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectMessage, res)
			}
		})
	}
}

func Test_List(t *testing.T) {

	tests := []struct {
		name string

		size    uint64
		token   string
		filters map[message.Field]any

		responseMessages []*message.Message
	}{
		{
			name: "normal",

			size:  10,
			token: "2023-01-03T21:35:02.809Z",
			filters: map[message.Field]any{
				message.FieldAIcallID: uuid.FromStringOrNil("5774f2dc-f262-11ef-b704-bb967f775316"),
				message.FieldDeleted:  false,
			},

			responseMessages: []*message.Message{
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("57bb3986-f262-11ef-b6db-57288b2a39c3"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &messageHandler{
				utilHandler:   mockUtil,
				notifyHandler: mockNotify,
				db:            mockDB,
			}
			ctx := context.Background()

			mockDB.EXPECT().MessageList(ctx, tt.size, tt.token, gomock.Any()).Return(tt.responseMessages, nil)

			res, err := h.List(ctx, tt.size, tt.token, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseMessages) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseMessages, res)
			}
		})
	}
}

func Test_Get(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseMessage *message.Message
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("2f6f4928-f2c0-11ef-b7ce-fbb7241790f5"),

			responseMessage: &message.Message{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("2f6f4928-f2c0-11ef-b7ce-fbb7241790f5"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &messageHandler{
				utilHandler:   mockUtil,
				notifyHandler: mockNotify,
				db:            mockDB,
			}
			ctx := context.Background()

			mockDB.EXPECT().MessageGet(ctx, tt.id).Return(tt.responseMessage, nil)

			res, err := h.Get(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseMessage) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseMessage, res)
			}
		})
	}
}

// Test_Create_WithOrigin pins that WithOrigin lands on the persisted row, and
// that omitting it still yields OriginNone. Origin drives both the frontend
// badge (proactive) and the LLM-replay exclusion (listen_internal), so a
// silently-dropped option is a silent correctness failure in two places.
func Test_Create_WithOrigin(t *testing.T) {
	tests := []struct {
		name string

		opts []CreateOption

		expectOrigin message.Origin
	}{
		{
			name:         "no option yields OriginNone",
			opts:         nil,
			expectOrigin: message.OriginNone,
		},
		{
			name:         "WithOrigin proactive",
			opts:         []CreateOption{WithOrigin(message.OriginProactive)},
			expectOrigin: message.OriginProactive,
		},
		{
			name:         "WithOrigin listen_internal",
			opts:         []CreateOption{WithOrigin(message.OriginListenInternal)},
			expectOrigin: message.OriginListenInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &messageHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			id := uuid.FromStringOrNil("6b1e8a40-2c7d-4f19-9a03-5e8c1d2f4b60")
			mockUtil.EXPECT().UUIDCreate().Return(id)

			mockDB.EXPECT().MessageCreate(ctx, gomock.Any()).DoAndReturn(
				func(_ context.Context, m *message.Message) error {
					if m.Origin != tt.expectOrigin {
						t.Errorf("Origin mismatch. expected: %q, got: %q", tt.expectOrigin, m.Origin)
					}
					return nil
				})
			mockDB.EXPECT().MessageGet(ctx, id).Return(&message.Message{Origin: tt.expectOrigin}, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, gomock.Any(), message.EventTypeMessageCreated, gomock.Any())

			res, err := h.Create(ctx, uuid.Nil, uuid.Nil, uuid.Nil, uuid.Nil,
				message.DirectionIncoming, message.RoleAssistant, "hello", nil, "", tt.opts...)
			if err != nil {
				t.Fatalf("Create returned an unexpected error. err: %v", err)
			}
			if res.Origin != tt.expectOrigin {
				t.Errorf("returned Origin mismatch. expected: %q, got: %q", tt.expectOrigin, res.Origin)
			}
		})
	}
}
