package messagehandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"

	"monorepo/bin-talk-manager/models/message"
	"monorepo/bin-talk-manager/models/participant"
	"monorepo/bin-talk-manager/models/talk"
	"monorepo/bin-talk-manager/pkg/dbhandler"
)

func Test_MessageCreate(t *testing.T) {
	customerID := uuid.FromStringOrNil("ba3ad8aa-cb0d-47fe-beef-f7c76c61a9f4")
	chatID := uuid.FromStringOrNil("e8427fa8-17b2-4e9e-8855-90e516bcf1d3")
	ownerID := uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc")
	parentID := uuid.FromStringOrNil("6a9a0ed0-1bcb-46de-a225-e638bbaf2fc1")
	softDeletedParentID := uuid.FromStringOrNil("04bc94c1-9cc1-4ce8-8559-39d6f1892109")

	tests := []struct {
		name string

		req MessageCreateRequest

		responseTalk         *talk.Talk
		responseParticipants []*participant.Participant
		responseParent       *message.Message

		expectError bool
	}{
		{
			name: "normal_root_message_without_parent",

			req: MessageCreateRequest{
				CustomerID: customerID,
				ChatID:     chatID,
				OwnerType:  "agent",
				OwnerID:    ownerID,
				Type:       message.TypeNormal,
				Text:       "Hello, this is a root message",
				Medias:     "",
			},

			responseTalk: &talk.Talk{
				Identity: commonidentity.Identity{
					ID:         chatID,
					CustomerID: customerID,
				},
				Type: talk.TypeNormal,
			},
			responseParticipants: []*participant.Participant{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("ac810dc4-298c-11ee-984c-ebb7811c4114"),
						CustomerID: customerID,
					},
					Owner: commonidentity.Owner{
						OwnerType: "agent",
						OwnerID:   ownerID,
					},
					ChatID: chatID,
				},
			},
			responseParent: nil,

			expectError: false,
		},
		{
			name: "normal_threaded_reply_with_parent",

			req: MessageCreateRequest{
				CustomerID: customerID,
				ChatID:     chatID,
				ParentID:   &parentID,
				OwnerType:  "agent",
				OwnerID:    ownerID,
				Type:       message.TypeNormal,
				Text:       "This is a reply to a message",
				Medias:     "",
			},

			responseTalk: &talk.Talk{
				Identity: commonidentity.Identity{
					ID:         chatID,
					CustomerID: customerID,
				},
				Type: talk.TypeNormal,
			},
			responseParticipants: []*participant.Participant{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("ac810dc4-298c-11ee-984c-ebb7811c4114"),
						CustomerID: customerID,
					},
					Owner: commonidentity.Owner{
						OwnerType: "agent",
						OwnerID:   ownerID,
					},
					ChatID: chatID,
				},
			},
			responseParent: &message.Message{
				Identity: commonidentity.Identity{
					ID:         parentID,
					CustomerID: customerID,
				},
				ChatID:   chatID,
				Type:     message.TypeNormal,
				Text:     "Original message",
				TMDelete: "", // Not deleted
			},

			expectError: false,
		},
		{
			name: "normal_reply_to_soft_deleted_parent_allowed",

			req: MessageCreateRequest{
				CustomerID: customerID,
				ChatID:     chatID,
				ParentID:   &softDeletedParentID,
				OwnerType:  "agent",
				OwnerID:    ownerID,
				Type:       message.TypeNormal,
				Text:       "Reply to deleted message",
				Medias:     "",
			},

			responseTalk: &talk.Talk{
				Identity: commonidentity.Identity{
					ID:         chatID,
					CustomerID: customerID,
				},
				Type: talk.TypeNormal,
			},
			responseParticipants: []*participant.Participant{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("31536998-da36-11ee-976a-b31b049d62c2"),
						CustomerID: customerID,
					},
					Owner: commonidentity.Owner{
						OwnerType: "agent",
						OwnerID:   ownerID,
					},
					ChatID: chatID,
				},
			},
			responseParent: &message.Message{
				Identity: commonidentity.Identity{
					ID:         softDeletedParentID,
					CustomerID: customerID,
				},
				ChatID:   chatID,
				Type:     message.TypeNormal,
				Text:     "Deleted parent message",
				TMDelete: "2024-01-17 00:00:00.000000", // Soft-deleted
			},

			expectError: false, // MUST NOT error - soft-deleted parents allowed
		},
		{
			name: "normal_system_message",

			req: MessageCreateRequest{
				CustomerID: customerID,
				ChatID:     chatID,
				OwnerType:  "system",
				OwnerID:    uuid.FromStringOrNil("62b0e2b7-0583-4f78-9406-45b00d17a9b4"),
				Type:       message.TypeSystem,
				Text:       "Agent joined the conversation",
				Medias:     "",
			},

			responseTalk: &talk.Talk{
				Identity: commonidentity.Identity{
					ID:         chatID,
					CustomerID: customerID,
				},
				Type: talk.TypeNormal,
			},
			responseParticipants: []*participant.Participant{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("62b0e2b7-0583-4f78-9406-45b00d17a9b4"),
						CustomerID: customerID,
					},
					Owner: commonidentity.Owner{
						OwnerType: "system",
						OwnerID:   uuid.FromStringOrNil("62b0e2b7-0583-4f78-9406-45b00d17a9b4"),
					},
					ChatID: chatID,
				},
			},
			responseParent: nil,

			expectError: false,
		},
		{
			name: "normal_message_with_medias",

			req: MessageCreateRequest{
				CustomerID: customerID,
				ChatID:     chatID,
				OwnerType:  "agent",
				OwnerID:    ownerID,
				Type:       message.TypeNormal,
				Text:       "Check this file",
				Medias:     `[{"type":"file"}]`,
			},

			responseTalk: &talk.Talk{
				Identity: commonidentity.Identity{
					ID:         chatID,
					CustomerID: customerID,
				},
				Type: talk.TypeNormal,
			},
			responseParticipants: []*participant.Participant{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("af243cbc-de04-4705-ad2b-78350d0a4fba"),
						CustomerID: customerID,
					},
					Owner: commonidentity.Owner{
						OwnerType: "agent",
						OwnerID:   ownerID,
					},
					ChatID: chatID,
				},
			},
			responseParent: nil,

			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &messageHandler{
				dbHandler:     mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			// Mock TalkGet
			mockDB.EXPECT().TalkGet(ctx, tt.req.ChatID).Return(tt.responseTalk, nil)

			// Mock ParticipantList
			mockDB.EXPECT().ParticipantList(ctx, gomock.Any()).Return(tt.responseParticipants, nil)

			// Mock MessageGet for parent validation (if parent_id provided)
			if tt.req.ParentID != nil {
				mockDB.EXPECT().MessageGet(ctx, *tt.req.ParentID).Return(tt.responseParent, nil)
			}

			// Mock MessageCreate
			mockDB.EXPECT().MessageCreate(ctx, gomock.Any()).Return(nil)

			// Mock webhook event publishing
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.req.CustomerID, message.EventTypeMessageCreated, gomock.Any())

			res, err := h.MessageCreate(ctx, tt.req)
			if tt.expectError && err == nil {
				t.Errorf("Wrong match. expect: error, got: ok")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !tt.expectError {
				if res.CustomerID != tt.req.CustomerID {
					t.Errorf("Wrong customer_id.\nexpect: %v\ngot: %v", tt.req.CustomerID, res.CustomerID)
				}

				if res.ChatID != tt.req.ChatID {
					t.Errorf("Wrong chat_id.\nexpect: %v\ngot: %v", tt.req.ChatID, res.ChatID)
				}

				if res.Type != message.Type(tt.req.Type) {
					t.Errorf("Wrong type.\nexpect: %v\ngot: %v", tt.req.Type, res.Type)
				}

				if res.Text != tt.req.Text {
					t.Errorf("Wrong text.\nexpect: %v\ngot: %v", tt.req.Text, res.Text)
				}

				if res.ID == uuid.Nil {
					t.Errorf("Wrong match. ID should not be nil")
				}

				if res.Metadata == "" {
					t.Errorf("Wrong match. Metadata should be initialized")
				}

				// Verify ParentID matches request
				if tt.req.ParentID != nil {
					if res.ParentID == nil {
						t.Errorf("Wrong match. ParentID should not be nil")
					} else if *res.ParentID != *tt.req.ParentID {
						t.Errorf("Wrong parent_id.\nexpect: %v\ngot: %v", *tt.req.ParentID, *res.ParentID)
					}
				} else {
					if res.ParentID != nil {
						t.Errorf("Wrong match. ParentID should be nil")
					}
				}
			}
		})
	}
}

func Test_MessageCreate_error(t *testing.T) {
	customerID := uuid.FromStringOrNil("ba3ad8aa-cb0d-47fe-beef-f7c76c61a9f4")
	chatID := uuid.FromStringOrNil("e8427fa8-17b2-4e9e-8855-90e516bcf1d3")
	chatID2 := uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b")
	ownerID := uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc")
	parentID := uuid.FromStringOrNil("6a9a0ed0-1bcb-46de-a225-e638bbaf2fc1")

	tests := []struct {
		name string

		req MessageCreateRequest

		responseTalk         *talk.Talk
		getTalkError         error
		responseParticipants []*participant.Participant
		getParticipantsError error
		responseParent       *message.Message
		getParentError       error
		createError          error

		expectError bool
	}{
		{
			name: "error_nil_customer_id",

			req: MessageCreateRequest{
				CustomerID: uuid.Nil,
				ChatID:     chatID,
				OwnerType:  "agent",
				OwnerID:    ownerID,
				Type:       message.TypeNormal,
				Text:       "Test message",
			},

			expectError: true,
		},
		{
			name: "error_nil_chat_id",

			req: MessageCreateRequest{
				CustomerID: customerID,
				ChatID:     uuid.Nil,
				OwnerType:  "agent",
				OwnerID:    ownerID,
				Type:       message.TypeNormal,
				Text:       "Test message",
			},

			expectError: true,
		},
		{
			name: "error_empty_owner_type",

			req: MessageCreateRequest{
				CustomerID: customerID,
				ChatID:     chatID,
				OwnerType:  "",
				OwnerID:    ownerID,
				Type:       message.TypeNormal,
				Text:       "Test message",
			},

			expectError: true,
		},
		{
			name: "error_nil_owner_id",

			req: MessageCreateRequest{
				CustomerID: customerID,
				ChatID:     chatID,
				OwnerType:  "agent",
				OwnerID:    uuid.Nil,
				Type:       message.TypeNormal,
				Text:       "Test message",
			},

			expectError: true,
		},
		{
			name: "error_empty_type",

			req: MessageCreateRequest{
				CustomerID: customerID,
				ChatID:     chatID,
				OwnerType:  "agent",
				OwnerID:    ownerID,
				Type:       "",
				Text:       "Test message",
			},

			expectError: true,
		},
		{
			name: "error_invalid_type_value",

			req: MessageCreateRequest{
				CustomerID: customerID,
				ChatID:     chatID,
				OwnerType:  "agent",
				OwnerID:    ownerID,
				Type:       "invalid_type",
				Text:       "Test message",
			},

			expectError: true,
		},
		{
			name: "error_invalid_medias_json_format",

			req: MessageCreateRequest{
				CustomerID: customerID,
				ChatID:     chatID,
				OwnerType:  "agent",
				OwnerID:    ownerID,
				Type:       message.TypeNormal,
				Text:       "Test message",
				Medias:     `{invalid json}`,
			},

			expectError: true,
		},
		{
			name: "error_talk_not_found",

			req: MessageCreateRequest{
				CustomerID: customerID,
				ChatID:     chatID,
				OwnerType:  "agent",
				OwnerID:    ownerID,
				Type:       message.TypeNormal,
				Text:       "Test message",
			},

			responseTalk: nil,
			getTalkError: fmt.Errorf("not found"),

			expectError: true,
		},
		{
			name: "error_talk_get_returns_nil",

			req: MessageCreateRequest{
				CustomerID: customerID,
				ChatID:     chatID,
				OwnerType:  "agent",
				OwnerID:    ownerID,
				Type:       message.TypeNormal,
				Text:       "Test message",
			},

			responseTalk: nil,
			getTalkError: nil,

			expectError: true,
		},
		{
			name: "error_sender_not_participant",

			req: MessageCreateRequest{
				CustomerID: customerID,
				ChatID:     chatID,
				OwnerType:  "agent",
				OwnerID:    ownerID,
				Type:       message.TypeNormal,
				Text:       "Test message",
			},

			responseTalk: &talk.Talk{
				Identity: commonidentity.Identity{
					ID:         chatID,
					CustomerID: customerID,
				},
				Type: talk.TypeNormal,
			},
			getTalkError:         nil,
			responseParticipants: []*participant.Participant{}, // Empty - sender not participant
			getParticipantsError: nil,

			expectError: true,
		},
		{
			name: "error_participant_check_failed",

			req: MessageCreateRequest{
				CustomerID: customerID,
				ChatID:     chatID,
				OwnerType:  "agent",
				OwnerID:    ownerID,
				Type:       message.TypeNormal,
				Text:       "Test message",
			},

			responseTalk: &talk.Talk{
				Identity: commonidentity.Identity{
					ID:         chatID,
					CustomerID: customerID,
				},
				Type: talk.TypeNormal,
			},
			getTalkError:         nil,
			responseParticipants: nil,
			getParticipantsError: fmt.Errorf("database error"),

			expectError: true,
		},
		{
			name: "error_parent_not_found",

			req: MessageCreateRequest{
				CustomerID: customerID,
				ChatID:     chatID,
				ParentID:   &parentID,
				OwnerType:  "agent",
				OwnerID:    ownerID,
				Type:       message.TypeNormal,
				Text:       "Reply message",
			},

			responseTalk: &talk.Talk{
				Identity: commonidentity.Identity{
					ID:         chatID,
					CustomerID: customerID,
				},
				Type: talk.TypeNormal,
			},
			getTalkError: nil,
			responseParticipants: []*participant.Participant{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("ac810dc4-298c-11ee-984c-ebb7811c4114"),
						CustomerID: customerID,
					},
					Owner: commonidentity.Owner{
						OwnerType: "agent",
						OwnerID:   ownerID,
					},
					ChatID: chatID,
				},
			},
			getParticipantsError: nil,
			responseParent:       nil,
			getParentError:       fmt.Errorf("not found"),

			expectError: true,
		},
		{
			name: "error_parent_get_returns_nil",

			req: MessageCreateRequest{
				CustomerID: customerID,
				ChatID:     chatID,
				ParentID:   &parentID,
				OwnerType:  "agent",
				OwnerID:    ownerID,
				Type:       message.TypeNormal,
				Text:       "Reply message",
			},

			responseTalk: &talk.Talk{
				Identity: commonidentity.Identity{
					ID:         chatID,
					CustomerID: customerID,
				},
				Type: talk.TypeNormal,
			},
			getTalkError: nil,
			responseParticipants: []*participant.Participant{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("ac810dc4-298c-11ee-984c-ebb7811c4114"),
						CustomerID: customerID,
					},
					Owner: commonidentity.Owner{
						OwnerType: "agent",
						OwnerID:   ownerID,
					},
					ChatID: chatID,
				},
			},
			getParticipantsError: nil,
			responseParent:       nil,
			getParentError:       nil,

			expectError: true,
		},
		{
			name: "error_parent_in_different_talk",

			req: MessageCreateRequest{
				CustomerID: customerID,
				ChatID:     chatID,
				ParentID:   &parentID,
				OwnerType:  "agent",
				OwnerID:    ownerID,
				Type:       message.TypeNormal,
				Text:       "Cross-talk threading attack",
			},

			responseTalk: &talk.Talk{
				Identity: commonidentity.Identity{
					ID:         chatID,
					CustomerID: customerID,
				},
				Type: talk.TypeNormal,
			},
			getTalkError: nil,
			responseParticipants: []*participant.Participant{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("31536998-da36-11ee-976a-b31b049d62c2"),
						CustomerID: customerID,
					},
					Owner: commonidentity.Owner{
						OwnerType: "agent",
						OwnerID:   ownerID,
					},
					ChatID: chatID,
				},
			},
			getParticipantsError: nil,
			responseParent: &message.Message{
				Identity: commonidentity.Identity{
					ID:         parentID,
					CustomerID: customerID,
				},
				ChatID:   chatID2, // Different talk!
				Type:     message.TypeNormal,
				Text:     "Message from another talk",
				TMDelete: "",
			},
			getParentError: nil,

			expectError: true, // MUST error - cross-talk threading blocked
		},
		{
			name: "error_database_create_failed",

			req: MessageCreateRequest{
				CustomerID: customerID,
				ChatID:     chatID,
				OwnerType:  "agent",
				OwnerID:    ownerID,
				Type:       message.TypeNormal,
				Text:       "Test message",
			},

			responseTalk: &talk.Talk{
				Identity: commonidentity.Identity{
					ID:         chatID,
					CustomerID: customerID,
				},
				Type: talk.TypeNormal,
			},
			getTalkError: nil,
			responseParticipants: []*participant.Participant{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("62b0e2b7-0583-4f78-9406-45b00d17a9b4"),
						CustomerID: customerID,
					},
					Owner: commonidentity.Owner{
						OwnerType: "agent",
						OwnerID:   ownerID,
					},
					ChatID: chatID,
				},
			},
			getParticipantsError: nil,
			createError:          fmt.Errorf("database error"),

			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &messageHandler{
				dbHandler:     mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			// Only mock dependencies if validation passes initial checks
			if tt.req.CustomerID != uuid.Nil &&
				tt.req.ChatID != uuid.Nil &&
				tt.req.OwnerType != "" &&
				tt.req.OwnerID != uuid.Nil &&
				tt.req.Type != "" &&
				tt.req.Type == message.TypeNormal || tt.req.Type == message.TypeSystem {

				// Check medias JSON format
				mediasValid := true
				if tt.req.Medias != "" {
					if tt.req.Medias == `{invalid json}` {
						mediasValid = false
					}
				}

				if mediasValid {
					// Mock TalkGet (always needed if basic validation passes)
					mockDB.EXPECT().TalkGet(ctx, tt.req.ChatID).Return(tt.responseTalk, tt.getTalkError)

					// Mock ParticipantList if talk exists
					if tt.getTalkError == nil && tt.responseTalk != nil {
						mockDB.EXPECT().ParticipantList(ctx, gomock.Any()).Return(tt.responseParticipants, tt.getParticipantsError)
					}

					// Mock MessageGet if parent_id provided and participant check passed
					if tt.req.ParentID != nil &&
						tt.getTalkError == nil &&
						tt.responseTalk != nil &&
						tt.getParticipantsError == nil &&
						len(tt.responseParticipants) > 0 {
						mockDB.EXPECT().MessageGet(ctx, *tt.req.ParentID).Return(tt.responseParent, tt.getParentError)
					}

					// Mock MessageCreate only if all validations pass
					if tt.getTalkError == nil &&
						tt.responseTalk != nil &&
						tt.getParticipantsError == nil &&
						len(tt.responseParticipants) > 0 &&
						(tt.req.ParentID == nil ||
							(tt.getParentError == nil &&
								tt.responseParent != nil &&
								tt.responseParent.ChatID == tt.req.ChatID)) {
						mockDB.EXPECT().MessageCreate(ctx, gomock.Any()).Return(tt.createError)
					}
				}
			}

			res, err := h.MessageCreate(ctx, tt.req)
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: ok")
			}

			if res != nil {
				t.Errorf("Wrong match. expect: nil result, got: %v", res)
			}
		})
	}
}

func Test_MessageGet(t *testing.T) {
	tests := []struct {
		name string

		id uuid.UUID

		responseMessage *message.Message
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("e8427fa8-17b2-4e9e-8855-90e516bcf1d3"),

			responseMessage: &message.Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("e8427fa8-17b2-4e9e-8855-90e516bcf1d3"),
					CustomerID: uuid.FromStringOrNil("809656e2-305e-43cd-8d7b-ccb44373dddb"),
				},
				Owner: commonidentity.Owner{
					OwnerType: "agent",
					OwnerID:   uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
				},
				ChatID:   uuid.FromStringOrNil("ba3ad8aa-cb0d-47fe-beef-f7c76c61a9f4"),
				Type:     message.TypeNormal,
				Text:     "Hello, world!",
				Metadata: `{"reactions":[]}`,
				TMCreate: "2024-01-15 10:30:00.000000",
			},
		},
		{
			name: "normal_with_parent",

			id: uuid.FromStringOrNil("6a9a0ed0-1bcb-46de-a225-e638bbaf2fc1"),

			responseMessage: &message.Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("6a9a0ed0-1bcb-46de-a225-e638bbaf2fc1"),
					CustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
				},
				Owner: commonidentity.Owner{
					OwnerType: "agent",
					OwnerID:   uuid.FromStringOrNil("31536998-da36-11ee-976a-b31b049d62c2"),
				},
				ChatID:   uuid.FromStringOrNil("ac810dc4-298c-11ee-984c-ebb7811c4114"),
				ParentID: func() *uuid.UUID { id := uuid.FromStringOrNil("04bc94c1-9cc1-4ce8-8559-39d6f1892109"); return &id }(),
				Type:     message.TypeNormal,
				Text:     "This is a reply",
				Metadata: `{"reactions":[]}`,
				TMCreate: "2024-01-15 11:00:00.000000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &messageHandler{
				dbHandler:     mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().MessageGet(ctx, tt.id).Return(tt.responseMessage, nil)

			res, err := h.MessageGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseMessage) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseMessage, res)
			}
		})
	}
}

func Test_MessageGet_error(t *testing.T) {
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

			h := &messageHandler{
				dbHandler:     mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().MessageGet(ctx, tt.id).Return(nil, tt.dbError)

			res, err := h.MessageGet(ctx, tt.id)
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: ok")
			}

			if res != nil {
				t.Errorf("Wrong match. expect: nil result, got: %v", res)
			}
		})
	}
}

func Test_MessageList(t *testing.T) {
	tests := []struct {
		name string

		filters map[message.Field]any
		token   string
		size    uint64

		responseMessages []*message.Message
	}{
		{
			name: "normal",

			filters: map[message.Field]any{
				message.FieldCustomerID: uuid.FromStringOrNil("809656e2-305e-43cd-8d7b-ccb44373dddb"),
				message.FieldChatID:     uuid.FromStringOrNil("ba3ad8aa-cb0d-47fe-beef-f7c76c61a9f4"),
				message.FieldDeleted:    false,
			},
			token: "2024-01-15 10:30:00.000000",
			size:  10,

			responseMessages: []*message.Message{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("e8427fa8-17b2-4e9e-8855-90e516bcf1d3"),
						CustomerID: uuid.FromStringOrNil("809656e2-305e-43cd-8d7b-ccb44373dddb"),
					},
					Owner: commonidentity.Owner{
						OwnerType: "agent",
						OwnerID:   uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
					},
					ChatID:   uuid.FromStringOrNil("ba3ad8aa-cb0d-47fe-beef-f7c76c61a9f4"),
					Type:     message.TypeNormal,
					Text:     "First message",
					Metadata: `{"reactions":[]}`,
					TMCreate: "2024-01-15 10:30:00.000000",
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("04bc94c1-9cc1-4ce8-8559-39d6f1892109"),
						CustomerID: uuid.FromStringOrNil("809656e2-305e-43cd-8d7b-ccb44373dddb"),
					},
					Owner: commonidentity.Owner{
						OwnerType: "agent",
						OwnerID:   uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
					},
					ChatID:   uuid.FromStringOrNil("ba3ad8aa-cb0d-47fe-beef-f7c76c61a9f4"),
					Type:     message.TypeNormal,
					Text:     "Second message",
					Metadata: `{"reactions":[]}`,
					TMCreate: "2024-01-15 11:00:00.000000",
				},
			},
		},
		{
			name: "normal_with_owner_filter",

			filters: map[message.Field]any{
				message.FieldCustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
				message.FieldChatID:     uuid.FromStringOrNil("ac810dc4-298c-11ee-984c-ebb7811c4114"),
				message.FieldOwnerID:    uuid.FromStringOrNil("31536998-da36-11ee-976a-b31b049d62c2"),
				message.FieldDeleted:    false,
			},
			token: "",
			size:  50,

			responseMessages: []*message.Message{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("6a9a0ed0-1bcb-46de-a225-e638bbaf2fc1"),
						CustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
					},
					Owner: commonidentity.Owner{
						OwnerType: "agent",
						OwnerID:   uuid.FromStringOrNil("31536998-da36-11ee-976a-b31b049d62c2"),
					},
					ChatID:   uuid.FromStringOrNil("ac810dc4-298c-11ee-984c-ebb7811c4114"),
					Type:     message.TypeNormal,
					Text:     "Agent message",
					Metadata: `{"reactions":[]}`,
					TMCreate: "2024-01-15 12:00:00.000000",
				},
			},
		},
		{
			name: "normal_empty_result",

			filters: map[message.Field]any{
				message.FieldCustomerID: uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
				message.FieldChatID:     uuid.FromStringOrNil("62b0e2b7-0583-4f78-9406-45b00d17a9b4"),
				message.FieldDeleted:    false,
			},
			token: "",
			size:  10,

			responseMessages: []*message.Message{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &messageHandler{
				dbHandler:     mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().MessageList(ctx, tt.filters, tt.token, tt.size).Return(tt.responseMessages, nil)

			res, err := h.MessageList(ctx, tt.filters, tt.token, tt.size)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseMessages) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseMessages, res)
			}
		})
	}
}

func Test_MessageList_error(t *testing.T) {
	tests := []struct {
		name string

		filters map[message.Field]any
		token   string
		size    uint64

		dbError error
	}{
		{
			name: "error_database_failure",

			filters: map[message.Field]any{
				message.FieldCustomerID: uuid.FromStringOrNil("809656e2-305e-43cd-8d7b-ccb44373dddb"),
				message.FieldChatID:     uuid.FromStringOrNil("ba3ad8aa-cb0d-47fe-beef-f7c76c61a9f4"),
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

			h := &messageHandler{
				dbHandler:     mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().MessageList(ctx, tt.filters, tt.token, tt.size).Return(nil, tt.dbError)

			res, err := h.MessageList(ctx, tt.filters, tt.token, tt.size)
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: ok")
			}

			if res != nil {
				t.Errorf("Wrong match. expect: nil result, got: %v", res)
			}
		})
	}
}

func Test_MessageDelete(t *testing.T) {
	tests := []struct {
		name string

		id uuid.UUID

		responseMessage        *message.Message
		responseUpdatedMessage *message.Message
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("af243cbc-de04-4705-ad2b-78350d0a4fba"),

			responseMessage: &message.Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("af243cbc-de04-4705-ad2b-78350d0a4fba"),
					CustomerID: uuid.FromStringOrNil("809656e2-305e-43cd-8d7b-ccb44373dddb"),
				},
				Owner: commonidentity.Owner{
					OwnerType: "agent",
					OwnerID:   uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
				},
				ChatID:   uuid.FromStringOrNil("ba3ad8aa-cb0d-47fe-beef-f7c76c61a9f4"),
				Type:     message.TypeNormal,
				Text:     "Message to delete",
				Metadata: `{"reactions":[]}`,
				TMCreate: "2024-01-15 10:30:00.000000",
				TMDelete: "",
			},
			responseUpdatedMessage: &message.Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("af243cbc-de04-4705-ad2b-78350d0a4fba"),
					CustomerID: uuid.FromStringOrNil("809656e2-305e-43cd-8d7b-ccb44373dddb"),
				},
				Owner: commonidentity.Owner{
					OwnerType: "agent",
					OwnerID:   uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
				},
				ChatID:   uuid.FromStringOrNil("ba3ad8aa-cb0d-47fe-beef-f7c76c61a9f4"),
				Type:     message.TypeNormal,
				Text:     "Message to delete",
				Metadata: `{"reactions":[]}`,
				TMCreate: "2024-01-15 10:30:00.000000",
				TMDelete: "2024-01-17 15:00:00.000000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &messageHandler{
				dbHandler:     mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			// First get to verify message exists and is not deleted
			mockDB.EXPECT().MessageGet(ctx, tt.id).Return(tt.responseMessage, nil)

			// Delete the message
			mockDB.EXPECT().MessageDelete(ctx, tt.id).Return(nil)

			// Get updated message with tm_delete set
			mockDB.EXPECT().MessageGet(ctx, tt.id).Return(tt.responseUpdatedMessage, nil)

			// Publish webhook event
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseUpdatedMessage.CustomerID, message.EventTypeMessageDeleted, tt.responseUpdatedMessage)

			res, err := h.MessageDelete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseUpdatedMessage) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseUpdatedMessage, res)
			}

			// Verify TMDelete is set
			if res.TMDelete == "" {
				t.Errorf("Wrong match. TMDelete should be set after deletion")
			}
		})
	}
}

func Test_MessageDelete_error(t *testing.T) {
	tests := []struct {
		name string

		id uuid.UUID

		responseMessage *message.Message
		getError        error
		deleteError     error
		getAfterError   error
	}{
		{
			name: "error_get_before_delete_failed",

			id: uuid.FromStringOrNil("af243cbc-de04-4705-ad2b-78350d0a4fba"),

			responseMessage: nil,
			getError:        fmt.Errorf("not found"),
		},
		{
			name: "error_message_not_found",

			id: uuid.FromStringOrNil("62b0e2b7-0583-4f78-9406-45b00d17a9b4"),

			responseMessage: nil,
			getError:        nil,
		},
		{
			name: "error_already_deleted",

			id: uuid.FromStringOrNil("e8427fa8-17b2-4e9e-8855-90e516bcf1d3"),

			responseMessage: &message.Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("e8427fa8-17b2-4e9e-8855-90e516bcf1d3"),
					CustomerID: uuid.FromStringOrNil("809656e2-305e-43cd-8d7b-ccb44373dddb"),
				},
				ChatID:   uuid.FromStringOrNil("ba3ad8aa-cb0d-47fe-beef-f7c76c61a9f4"),
				TMDelete: "2024-01-17 10:00:00.000000", // Already deleted
			},
			getError: nil,
		},
		{
			name: "error_delete_failed",

			id: uuid.FromStringOrNil("6a9a0ed0-1bcb-46de-a225-e638bbaf2fc1"),

			responseMessage: &message.Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("6a9a0ed0-1bcb-46de-a225-e638bbaf2fc1"),
					CustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
				},
				ChatID:   uuid.FromStringOrNil("ac810dc4-298c-11ee-984c-ebb7811c4114"),
				TMDelete: "",
			},
			getError:    nil,
			deleteError: fmt.Errorf("database error"),
		},
		{
			name: "error_get_after_delete_failed",

			id: uuid.FromStringOrNil("04bc94c1-9cc1-4ce8-8559-39d6f1892109"),

			responseMessage: &message.Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("04bc94c1-9cc1-4ce8-8559-39d6f1892109"),
					CustomerID: uuid.FromStringOrNil("ba3ad8aa-cb0d-47fe-beef-f7c76c61a9f4"),
				},
				ChatID:   uuid.FromStringOrNil("31536998-da36-11ee-976a-b31b049d62c2"),
				TMDelete: "",
			},
			getError:      nil,
			deleteError:   nil,
			getAfterError: fmt.Errorf("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &messageHandler{
				dbHandler:     mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			// Always mock first get
			mockDB.EXPECT().MessageGet(ctx, tt.id).Return(tt.responseMessage, tt.getError)

			// Only mock delete if first get succeeded and message not already deleted
			if tt.getError == nil &&
				tt.responseMessage != nil &&
				tt.responseMessage.TMDelete == "" {
				mockDB.EXPECT().MessageDelete(ctx, tt.id).Return(tt.deleteError)

				// Only mock second get if delete succeeded
				if tt.deleteError == nil {
					mockDB.EXPECT().MessageGet(ctx, tt.id).Return(nil, tt.getAfterError)
				}
			}

			res, err := h.MessageDelete(ctx, tt.id)
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: ok")
			}

			if res != nil {
				t.Errorf("Wrong match. expect: nil result, got: %v", res)
			}
		})
	}
}
