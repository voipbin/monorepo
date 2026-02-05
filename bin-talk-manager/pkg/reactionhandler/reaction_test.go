package reactionhandler

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	commonutil "monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-talk-manager/models/message"
	"monorepo/bin-talk-manager/pkg/dbhandler"
)

func timePtr(t time.Time) *time.Time {
	return &t
}

func Test_ReactionAdd(t *testing.T) {
	messageID := uuid.FromStringOrNil("e8427fa8-17b2-4e9e-8855-90e516bcf1d3")
	customerID := uuid.FromStringOrNil("809656e2-305e-43cd-8d7b-ccb44373dddb")
	chatID := uuid.FromStringOrNil("ba3ad8aa-cb0d-47fe-beef-f7c76c61a9f4")
	ownerID := uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc")

	tests := []struct {
		name string

		messageID uuid.UUID
		emoji     string
		ownerType string
		ownerID   uuid.UUID

		responseMessage        *message.Message
		responseUpdatedMessage *message.Message

		expectError bool
	}{
		{
			name: "normal_add_new_reaction",

			messageID: messageID,
			emoji:     "👍",
			ownerType: "agent",
			ownerID:   ownerID,

			responseMessage: &message.Message{
				Identity: commonidentity.Identity{
					ID:         messageID,
					CustomerID: customerID,
				},
				ChatID:   chatID,
				Type:     message.TypeNormal,
				Text:     "Hello, world!",
				Metadata: message.Metadata{Reactions: []message.Reaction{}},
				TMCreate: timePtr(time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)),
			},
			responseUpdatedMessage: &message.Message{
				Identity: commonidentity.Identity{
					ID:         messageID,
					CustomerID: customerID,
				},
				ChatID:   chatID,
				Type:     message.TypeNormal,
				Text:     "Hello, world!",
				Metadata: message.Metadata{Reactions: []message.Reaction{{Emoji: "👍", OwnerType: "agent", OwnerID: uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"), TMCreate: timePtr(time.Date(2024, 1, 17, 10, 30, 0, 0, time.UTC))}}},
				TMCreate: timePtr(time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)),
			},

			expectError: false,
		},
		{
			name: "normal_add_second_reaction_different_emoji",

			messageID: messageID,
			emoji:     "❤️",
			ownerType: "agent",
			ownerID:   ownerID,

			responseMessage: &message.Message{
				Identity: commonidentity.Identity{
					ID:         messageID,
					CustomerID: customerID,
				},
				ChatID:   chatID,
				Type:     message.TypeNormal,
				Text:     "Message with existing reaction",
				Metadata: message.Metadata{Reactions: []message.Reaction{{Emoji: "👍", OwnerType: "agent", OwnerID: uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"), TMCreate: timePtr(time.Date(2024, 1, 17, 10, 30, 0, 0, time.UTC))}}},
				TMCreate: timePtr(time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)),
			},
			responseUpdatedMessage: &message.Message{
				Identity: commonidentity.Identity{
					ID:         messageID,
					CustomerID: customerID,
				},
				ChatID:   chatID,
				Type:     message.TypeNormal,
				Text:     "Message with existing reaction",
				Metadata: message.Metadata{Reactions: []message.Reaction{{Emoji: "👍", OwnerType: "agent", OwnerID: uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"), TMCreate: timePtr(time.Date(2024, 1, 17, 10, 30, 0, 0, time.UTC))}, {Emoji: "❤️", OwnerType: "agent", OwnerID: uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"), TMCreate: timePtr(time.Date(2024, 1, 17, 10, 35, 0, 0, time.UTC))}}},
				TMCreate: timePtr(time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)),
			},

			expectError: false,
		},
		{
			name: "normal_add_duplicate_reaction_idempotent",

			messageID: messageID,
			emoji:     "👍",
			ownerType: "agent",
			ownerID:   ownerID,

			responseMessage: &message.Message{
				Identity: commonidentity.Identity{
					ID:         messageID,
					CustomerID: customerID,
				},
				ChatID:   chatID,
				Type:     message.TypeNormal,
				Text:     "Message with reaction",
				Metadata: message.Metadata{Reactions: []message.Reaction{{Emoji: "👍", OwnerType: "agent", OwnerID: uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"), TMCreate: timePtr(time.Date(2024, 1, 17, 10, 30, 0, 0, time.UTC))}}},
				TMCreate: timePtr(time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)),
			},
			responseUpdatedMessage: nil, // Not used - returns same message

			expectError: false, // MUST NOT error - idempotent behavior
		},
		{
			name: "normal_different_owner_same_emoji",

			messageID: messageID,
			emoji:     "👍",
			ownerType: "agent",
			ownerID:   uuid.FromStringOrNil("31536998-da36-11ee-976a-b31b049d62c2"),

			responseMessage: &message.Message{
				Identity: commonidentity.Identity{
					ID:         messageID,
					CustomerID: customerID,
				},
				ChatID:   chatID,
				Type:     message.TypeNormal,
				Text:     "Message with reaction from another user",
				Metadata: message.Metadata{Reactions: []message.Reaction{{Emoji: "👍", OwnerType: "agent", OwnerID: uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"), TMCreate: timePtr(time.Date(2024, 1, 17, 10, 30, 0, 0, time.UTC))}}},
				TMCreate: timePtr(time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)),
			},
			responseUpdatedMessage: &message.Message{
				Identity: commonidentity.Identity{
					ID:         messageID,
					CustomerID: customerID,
				},
				ChatID:   chatID,
				Type:     message.TypeNormal,
				Text:     "Message with reaction from another user",
				Metadata: message.Metadata{Reactions: []message.Reaction{{Emoji: "👍", OwnerType: "agent", OwnerID: uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"), TMCreate: timePtr(time.Date(2024, 1, 17, 10, 30, 0, 0, time.UTC))}, {Emoji: "👍", OwnerType: "agent", OwnerID: uuid.FromStringOrNil("31536998-da36-11ee-976a-b31b049d62c2"), TMCreate: timePtr(time.Date(2024, 1, 17, 10, 35, 0, 0, time.UTC))}}},
				TMCreate: timePtr(time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)),
			},

			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockUtil := commonutil.NewMockUtilHandler(mc)

			h := &reactionHandler{
				dbHandler:     mockDB,
				notifyHandler: mockNotify,
				utilHandler:   mockUtil,
			}

			ctx := context.Background()

			// Mock initial MessageGet to check for duplicates
			mockDB.EXPECT().MessageGet(ctx, tt.messageID).Return(tt.responseMessage, nil)

			// If not idempotent case, mock atomic add and final get
			if tt.responseUpdatedMessage != nil {
				// Mock timestamp generation
				mockUtil.EXPECT().TimeNow().Return(timePtr(time.Date(2024, 1, 17, 10, 30, 0, 0, time.UTC)))
				mockDB.EXPECT().MessageAddReactionAtomic(ctx, tt.messageID, gomock.Any()).Return(nil)
				mockDB.EXPECT().MessageGet(ctx, tt.messageID).Return(tt.responseUpdatedMessage, nil)
			}

			// Always mock webhook publish
			mockNotify.EXPECT().PublishWebhookEvent(ctx, customerID, message.EventTypeMessageReactionUpdated, gomock.Any())

			res, err := h.ReactionAdd(ctx, tt.messageID, tt.emoji, tt.ownerType, tt.ownerID)
			if tt.expectError && err == nil {
				t.Errorf("Wrong match. expect: error, got: ok")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !tt.expectError {
				if res == nil {
					t.Errorf("Wrong match. expect: message, got: nil")
					return
				}

				if res.ID != tt.messageID {
					t.Errorf("Wrong message_id.\nexpect: %v\ngot: %v", tt.messageID, res.ID)
				}

				if res.CustomerID != customerID {
					t.Errorf("Wrong customer_id.\nexpect: %v\ngot: %v", customerID, res.CustomerID)
				}
			}
		})
	}
}

func Test_ReactionAdd_error(t *testing.T) {
	messageID := uuid.FromStringOrNil("e8427fa8-17b2-4e9e-8855-90e516bcf1d3")
	customerID := uuid.FromStringOrNil("809656e2-305e-43cd-8d7b-ccb44373dddb")
	chatID := uuid.FromStringOrNil("ba3ad8aa-cb0d-47fe-beef-f7c76c61a9f4")
	ownerID := uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc")

	tests := []struct {
		name string

		messageID uuid.UUID
		emoji     string
		ownerType string
		ownerID   uuid.UUID

		responseMessage *message.Message
		getError        error
		atomicAddError  error
		getAfterError   error

		expectError bool
	}{
		{
			name: "error_nil_message_id",

			messageID: uuid.Nil,
			emoji:     "👍",
			ownerType: "agent",
			ownerID:   ownerID,

			expectError: true,
		},
		{
			name: "error_empty_emoji",

			messageID: messageID,
			emoji:     "",
			ownerType: "agent",
			ownerID:   ownerID,

			expectError: true,
		},
		{
			name: "error_empty_owner_type",

			messageID: messageID,
			emoji:     "👍",
			ownerType: "",
			ownerID:   ownerID,

			expectError: true,
		},
		{
			name: "error_nil_owner_id",

			messageID: messageID,
			emoji:     "👍",
			ownerType: "agent",
			ownerID:   uuid.Nil,

			expectError: true,
		},
		{
			name: "error_message_not_found",

			messageID: messageID,
			emoji:     "👍",
			ownerType: "agent",
			ownerID:   ownerID,

			responseMessage: nil,
			getError:        fmt.Errorf("not found"),

			expectError: true,
		},
		{
			name: "error_message_get_returns_nil",

			messageID: messageID,
			emoji:     "👍",
			ownerType: "agent",
			ownerID:   ownerID,

			responseMessage: nil,
			getError:        nil,

			expectError: true,
		},
		{
			name: "error_atomic_add_failed",

			messageID: messageID,
			emoji:     "👍",
			ownerType: "agent",
			ownerID:   ownerID,

			responseMessage: &message.Message{
				Identity: commonidentity.Identity{
					ID:         messageID,
					CustomerID: customerID,
				},
				ChatID:   chatID,
				Type:     message.TypeNormal,
				Text:     "Test message",
				Metadata: message.Metadata{Reactions: []message.Reaction{}},
			},
			getError:       nil,
			atomicAddError: fmt.Errorf("database error"),

			expectError: true,
		},
		{
			name: "error_get_after_add_failed",

			messageID: messageID,
			emoji:     "👍",
			ownerType: "agent",
			ownerID:   ownerID,

			responseMessage: &message.Message{
				Identity: commonidentity.Identity{
					ID:         messageID,
					CustomerID: customerID,
				},
				ChatID:   chatID,
				Type:     message.TypeNormal,
				Text:     "Test message",
				Metadata: message.Metadata{Reactions: []message.Reaction{}},
			},
			getError:       nil,
			atomicAddError: nil,
			getAfterError:  fmt.Errorf("database error"),

			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockUtil := commonutil.NewMockUtilHandler(mc)

			h := &reactionHandler{
				dbHandler:     mockDB,
				notifyHandler: mockNotify,
				utilHandler:   mockUtil,
			}

			ctx := context.Background()

			// Only mock dependencies if validation passes
			if tt.messageID != uuid.Nil &&
				tt.emoji != "" &&
				tt.ownerType != "" &&
				tt.ownerID != uuid.Nil {

				// Mock initial MessageGet
				mockDB.EXPECT().MessageGet(ctx, tt.messageID).Return(tt.responseMessage, tt.getError)

				// Only mock atomic add if get succeeded and returned non-nil message
				if tt.getError == nil && tt.responseMessage != nil {
					// Mock UUID generation for timestamp
					mockUtil.EXPECT().TimeNow().Return(timePtr(time.Date(2024, 1, 17, 10, 30, 0, 0, time.UTC)))
					mockDB.EXPECT().MessageAddReactionAtomic(ctx, tt.messageID, gomock.Any()).Return(tt.atomicAddError)

					// Only mock second get if atomic add succeeded
					if tt.atomicAddError == nil {
						mockDB.EXPECT().MessageGet(ctx, tt.messageID).Return(nil, tt.getAfterError)
					}
				}
			}

			res, err := h.ReactionAdd(ctx, tt.messageID, tt.emoji, tt.ownerType, tt.ownerID)
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: ok")
			}

			if res != nil {
				t.Errorf("Wrong match. expect: nil result, got: %v", res)
			}
		})
	}
}

func Test_ReactionRemove(t *testing.T) {
	messageID := uuid.FromStringOrNil("e8427fa8-17b2-4e9e-8855-90e516bcf1d3")
	customerID := uuid.FromStringOrNil("809656e2-305e-43cd-8d7b-ccb44373dddb")
	chatID := uuid.FromStringOrNil("ba3ad8aa-cb0d-47fe-beef-f7c76c61a9f4")
	ownerID := uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc")

	tests := []struct {
		name string

		messageID uuid.UUID
		emoji     string
		ownerType string
		ownerID   uuid.UUID

		responseUpdatedMessage *message.Message

		expectError bool
	}{
		{
			name: "normal_remove_reaction",

			messageID: messageID,
			emoji:     "👍",
			ownerType: "agent",
			ownerID:   ownerID,

			responseUpdatedMessage: &message.Message{
				Identity: commonidentity.Identity{
					ID:         messageID,
					CustomerID: customerID,
				},
				ChatID:   chatID,
				Type:     message.TypeNormal,
				Text:     "Message after reaction removed",
				Metadata: message.Metadata{Reactions: []message.Reaction{}},
				TMCreate: timePtr(time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)),
			},

			expectError: false,
		},
		{
			name: "normal_remove_one_of_multiple_reactions",

			messageID: messageID,
			emoji:     "👍",
			ownerType: "agent",
			ownerID:   ownerID,

			responseUpdatedMessage: &message.Message{
				Identity: commonidentity.Identity{
					ID:         messageID,
					CustomerID: customerID,
				},
				ChatID:   chatID,
				Type:     message.TypeNormal,
				Text:     "Message with remaining reactions",
				Metadata: message.Metadata{Reactions: []message.Reaction{{Emoji: "❤️", OwnerType: "agent", OwnerID: uuid.FromStringOrNil("31536998-da36-11ee-976a-b31b049d62c2"), TMCreate: timePtr(time.Date(2024, 1, 17, 10, 35, 0, 0, time.UTC))}}},
				TMCreate: timePtr(time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)),
			},

			expectError: false,
		},
		{
			name: "normal_remove_from_different_owner",

			messageID: messageID,
			emoji:     "👍",
			ownerType: "agent",
			ownerID:   uuid.FromStringOrNil("31536998-da36-11ee-976a-b31b049d62c2"),

			responseUpdatedMessage: &message.Message{
				Identity: commonidentity.Identity{
					ID:         messageID,
					CustomerID: customerID,
				},
				ChatID:   chatID,
				Type:     message.TypeNormal,
				Text:     "Message after specific user reaction removed",
				Metadata: message.Metadata{Reactions: []message.Reaction{{Emoji: "👍", OwnerType: "agent", OwnerID: uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"), TMCreate: timePtr(time.Date(2024, 1, 17, 10, 30, 0, 0, time.UTC))}}},
				TMCreate: timePtr(time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)),
			},

			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockUtil := commonutil.NewMockUtilHandler(mc)

			h := &reactionHandler{
				dbHandler:     mockDB,
				notifyHandler: mockNotify,
				utilHandler:   mockUtil,
			}

			ctx := context.Background()

			// Mock atomic remove
			mockDB.EXPECT().MessageRemoveReactionAtomic(ctx, tt.messageID, tt.emoji, tt.ownerType, tt.ownerID).Return(nil)

			// Mock get after remove
			mockDB.EXPECT().MessageGet(ctx, tt.messageID).Return(tt.responseUpdatedMessage, nil)

			// Mock webhook publish (message is converted to WebhookMessage before publishing)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, customerID, message.EventTypeMessageReactionUpdated, gomock.Any())

			res, err := h.ReactionRemove(ctx, tt.messageID, tt.emoji, tt.ownerType, tt.ownerID)
			if tt.expectError && err == nil {
				t.Errorf("Wrong match. expect: error, got: ok")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !tt.expectError {
				if res == nil {
					t.Errorf("Wrong match. expect: message, got: nil")
					return
				}

				if res.ID != tt.messageID {
					t.Errorf("Wrong message_id.\nexpect: %v\ngot: %v", tt.messageID, res.ID)
				}

				if res.CustomerID != customerID {
					t.Errorf("Wrong customer_id.\nexpect: %v\ngot: %v", customerID, res.CustomerID)
				}
			}
		})
	}
}

func Test_ReactionRemove_error(t *testing.T) {
	messageID := uuid.FromStringOrNil("e8427fa8-17b2-4e9e-8855-90e516bcf1d3")
	ownerID := uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc")

	tests := []struct {
		name string

		messageID uuid.UUID
		emoji     string
		ownerType string
		ownerID   uuid.UUID

		atomicRemoveError error
		getAfterError     error

		expectError bool
	}{
		{
			name: "error_nil_message_id",

			messageID: uuid.Nil,
			emoji:     "👍",
			ownerType: "agent",
			ownerID:   ownerID,

			expectError: true,
		},
		{
			name: "error_empty_emoji",

			messageID: messageID,
			emoji:     "",
			ownerType: "agent",
			ownerID:   ownerID,

			expectError: true,
		},
		{
			name: "error_empty_owner_type",

			messageID: messageID,
			emoji:     "👍",
			ownerType: "",
			ownerID:   ownerID,

			expectError: true,
		},
		{
			name: "error_nil_owner_id",

			messageID: messageID,
			emoji:     "👍",
			ownerType: "agent",
			ownerID:   uuid.Nil,

			expectError: true,
		},
		{
			name: "error_atomic_remove_failed",

			messageID: messageID,
			emoji:     "👍",
			ownerType: "agent",
			ownerID:   ownerID,

			atomicRemoveError: fmt.Errorf("database error"),

			expectError: true,
		},
		{
			name: "error_get_after_remove_failed",

			messageID: messageID,
			emoji:     "👍",
			ownerType: "agent",
			ownerID:   ownerID,

			atomicRemoveError: nil,
			getAfterError:     fmt.Errorf("database error"),

			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockUtil := commonutil.NewMockUtilHandler(mc)

			h := &reactionHandler{
				dbHandler:     mockDB,
				notifyHandler: mockNotify,
				utilHandler:   mockUtil,
			}

			ctx := context.Background()

			// Only mock dependencies if validation passes
			if tt.messageID != uuid.Nil &&
				tt.emoji != "" &&
				tt.ownerType != "" &&
				tt.ownerID != uuid.Nil {

				// Mock atomic remove
				mockDB.EXPECT().MessageRemoveReactionAtomic(ctx, tt.messageID, tt.emoji, tt.ownerType, tt.ownerID).Return(tt.atomicRemoveError)

				// Only mock get if atomic remove succeeded
				if tt.atomicRemoveError == nil {
					mockDB.EXPECT().MessageGet(ctx, tt.messageID).Return(nil, tt.getAfterError)
				}
			}

			res, err := h.ReactionRemove(ctx, tt.messageID, tt.emoji, tt.ownerType, tt.ownerID)
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: ok")
			}

			if res != nil {
				t.Errorf("Wrong match. expect: nil result, got: %v", res)
			}
		})
	}
}
