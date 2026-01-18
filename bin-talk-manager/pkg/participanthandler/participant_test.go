package participanthandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	commonutil "monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-talk-manager/models/participant"
	"monorepo/bin-talk-manager/pkg/dbhandler"
)

func Test_ParticipantAdd(t *testing.T) {
	tests := []struct {
		name string

		customerID uuid.UUID
		chatID     uuid.UUID
		ownerID    uuid.UUID
		ownerType  string

		expectError bool
	}{
		{
			name: "normal_first_join",

			customerID: uuid.FromStringOrNil("ba3ad8aa-cb0d-47fe-beef-f7c76c61a9f4"),
			chatID:     uuid.FromStringOrNil("e8427fa8-17b2-4e9e-8855-90e516bcf1d3"),
			ownerID:    uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
			ownerType:  "agent",

			expectError: false,
		},
		{
			name: "normal_agent_participant",

			customerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
			chatID:     uuid.FromStringOrNil("ac810dc4-298c-11ee-984c-ebb7811c4114"),
			ownerID:    uuid.FromStringOrNil("31536998-da36-11ee-976a-b31b049d62c2"),
			ownerType:  "agent",

			expectError: false,
		},
		{
			name: "normal_system_participant",

			customerID: uuid.FromStringOrNil("809656e2-305e-43cd-8d7b-ccb44373dddb"),
			chatID:     uuid.FromStringOrNil("6a9a0ed0-1bcb-46de-a225-e638bbaf2fc1"),
			ownerID:    uuid.FromStringOrNil("04bc94c1-9cc1-4ce8-8559-39d6f1892109"),
			ownerType:  "system",

			expectError: false,
		},
		{
			name: "rejoin_after_leaving_upsert",

			customerID: uuid.FromStringOrNil("ba3ad8aa-cb0d-47fe-beef-f7c76c61a9f4"),
			chatID:     uuid.FromStringOrNil("e8427fa8-17b2-4e9e-8855-90e516bcf1d3"),
			ownerID:    uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
			ownerType:  "agent",

			expectError: false, // MUST NOT error - UPSERT behavior
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockUtil := commonutil.NewMockUtilHandler(mc)

			h := &participantHandler{
				dbHandler:     mockDB,
				notifyHandler: mockNotify,
				utilHandler:   mockUtil,
			}

			ctx := context.Background()

			// Mock UUID generation
			mockUtil.EXPECT().UUIDCreate().Return(uuid.FromStringOrNil("93d48228-3ed7-11ef-a9ca-070e7ba46a55")).AnyTimes()

			// Mock database create (UPSERT behavior)
			mockDB.EXPECT().ParticipantCreate(ctx, gomock.Any()).Return(nil)

			// Mock webhook event publishing
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.customerID, participant.EventParticipantAdded, gomock.Any())

			res, err := h.ParticipantAdd(ctx, tt.customerID, tt.chatID, tt.ownerID, tt.ownerType)
			if tt.expectError && err == nil {
				t.Errorf("Wrong match. expect: error, got: ok")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !tt.expectError {
				if res.CustomerID != tt.customerID {
					t.Errorf("Wrong customer_id.\nexpect: %v\ngot: %v", tt.customerID, res.CustomerID)
				}

				if res.ChatID != tt.chatID {
					t.Errorf("Wrong chat_id.\nexpect: %v\ngot: %v", tt.chatID, res.ChatID)
				}

				if res.OwnerID != tt.ownerID {
					t.Errorf("Wrong owner_id.\nexpect: %v\ngot: %v", tt.ownerID, res.OwnerID)
				}

				if string(res.OwnerType) != tt.ownerType {
					t.Errorf("Wrong owner_type.\nexpect: %v\ngot: %v", tt.ownerType, res.OwnerType)
				}

				if res.ID == uuid.Nil {
					t.Errorf("Wrong match. ID should not be nil")
				}
			}
		})
	}
}

func Test_ParticipantAdd_error(t *testing.T) {
	tests := []struct {
		name string

		customerID uuid.UUID
		chatID     uuid.UUID
		ownerID    uuid.UUID
		ownerType  string

		createError error
		expectError bool
	}{
		{
			name: "error_nil_customer_id",

			customerID: uuid.Nil,
			chatID:     uuid.FromStringOrNil("e8427fa8-17b2-4e9e-8855-90e516bcf1d3"),
			ownerID:    uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
			ownerType:  "agent",

			expectError: true,
		},
		{
			name: "error_nil_chat_id",

			customerID: uuid.FromStringOrNil("ba3ad8aa-cb0d-47fe-beef-f7c76c61a9f4"),
			chatID:     uuid.Nil,
			ownerID:    uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
			ownerType:  "agent",

			expectError: true,
		},
		{
			name: "error_nil_owner_id",

			customerID: uuid.FromStringOrNil("ba3ad8aa-cb0d-47fe-beef-f7c76c61a9f4"),
			chatID:     uuid.FromStringOrNil("e8427fa8-17b2-4e9e-8855-90e516bcf1d3"),
			ownerID:    uuid.Nil,
			ownerType:  "agent",

			expectError: true,
		},
		{
			name: "error_empty_owner_type",

			customerID: uuid.FromStringOrNil("ba3ad8aa-cb0d-47fe-beef-f7c76c61a9f4"),
			chatID:     uuid.FromStringOrNil("e8427fa8-17b2-4e9e-8855-90e516bcf1d3"),
			ownerID:    uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
			ownerType:  "",

			expectError: true,
		},
		{
			name: "error_database_create_failure",

			customerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
			chatID:     uuid.FromStringOrNil("ac810dc4-298c-11ee-984c-ebb7811c4114"),
			ownerID:    uuid.FromStringOrNil("31536998-da36-11ee-976a-b31b049d62c2"),
			ownerType:  "agent",

			createError: fmt.Errorf("database error"),
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

			h := &participantHandler{
				dbHandler:     mockDB,
				notifyHandler: mockNotify,
				utilHandler:   mockUtil,
			}

			ctx := context.Background()

			// Only mock database call if validation passes
			if tt.customerID != uuid.Nil &&
				tt.chatID != uuid.Nil &&
				tt.ownerID != uuid.Nil &&
				tt.ownerType != "" {
				// Mock UUID generation for this case
				mockUtil.EXPECT().UUIDCreate().Return(uuid.FromStringOrNil("93d48228-3ed7-11ef-a9ca-070e7ba46a55"))
				mockDB.EXPECT().ParticipantCreate(ctx, gomock.Any()).Return(tt.createError)
			}

			res, err := h.ParticipantAdd(ctx, tt.customerID, tt.chatID, tt.ownerID, tt.ownerType)
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: ok")
			}

			if res != nil {
				t.Errorf("Wrong match. expect: nil result, got: %v", res)
			}
		})
	}
}

func Test_ParticipantList(t *testing.T) {
	tests := []struct {
		name string

		customerID uuid.UUID
		chatID     uuid.UUID

		responseParticipants []*participant.Participant
	}{
		{
			name: "normal_single_participant",

			customerID: uuid.FromStringOrNil("809656e2-305e-43cd-8d7b-ccb44373dddb"),
			chatID:     uuid.FromStringOrNil("ba3ad8aa-cb0d-47fe-beef-f7c76c61a9f4"),

			responseParticipants: []*participant.Participant{
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
					TMJoined: "2024-01-15 10:30:00.000000",
				},
			},
		},
		{
			name: "normal_multiple_participants",

			customerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
			chatID:     uuid.FromStringOrNil("ac810dc4-298c-11ee-984c-ebb7811c4114"),

			responseParticipants: []*participant.Participant{
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
					TMJoined: "2024-01-15 10:30:00.000000",
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("04bc94c1-9cc1-4ce8-8559-39d6f1892109"),
						CustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
					},
					Owner: commonidentity.Owner{
						OwnerType: "agent",
						OwnerID:   uuid.FromStringOrNil("62b0e2b7-0583-4f78-9406-45b00d17a9b4"),
					},
					ChatID:   uuid.FromStringOrNil("ac810dc4-298c-11ee-984c-ebb7811c4114"),
					TMJoined: "2024-01-15 11:00:00.000000",
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("af243cbc-de04-4705-ad2b-78350d0a4fba"),
						CustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
					},
					Owner: commonidentity.Owner{
						OwnerType: "system",
						OwnerID:   uuid.FromStringOrNil("809656e2-305e-43cd-8d7b-ccb44373dddb"),
					},
					ChatID:   uuid.FromStringOrNil("ac810dc4-298c-11ee-984c-ebb7811c4114"),
					TMJoined: "2024-01-15 12:00:00.000000",
				},
			},
		},
		{
			name: "normal_empty_result",

			customerID: uuid.FromStringOrNil("ba3ad8aa-cb0d-47fe-beef-f7c76c61a9f4"),
			chatID:     uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),

			responseParticipants: []*participant.Participant{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockUtil := commonutil.NewMockUtilHandler(mc)

			h := &participantHandler{
				dbHandler:     mockDB,
				notifyHandler: mockNotify,
				utilHandler:   mockUtil,
			}

			ctx := context.Background()

			// Mock UUID generation
			mockUtil.EXPECT().UUIDCreate().Return(uuid.FromStringOrNil("93d48228-3ed7-11ef-a9ca-070e7ba46a55")).AnyTimes()

			// Mock database list call
			mockDB.EXPECT().ParticipantList(ctx, gomock.Any()).Return(tt.responseParticipants, nil)

			res, err := h.ParticipantList(ctx, tt.customerID, tt.chatID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseParticipants) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseParticipants, res)
			}
		})
	}
}

func Test_ParticipantList_error(t *testing.T) {
	tests := []struct {
		name string

		customerID uuid.UUID
		chatID     uuid.UUID

		dbError error
	}{
		{
			name: "error_nil_customer_id",

			customerID: uuid.Nil,
			chatID:     uuid.FromStringOrNil("ba3ad8aa-cb0d-47fe-beef-f7c76c61a9f4"),

			dbError: nil,
		},
		{
			name: "error_nil_chat_id",

			customerID: uuid.FromStringOrNil("809656e2-305e-43cd-8d7b-ccb44373dddb"),
			chatID:     uuid.Nil,

			dbError: nil,
		},
		{
			name: "error_database_failure",

			customerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
			chatID:     uuid.FromStringOrNil("ac810dc4-298c-11ee-984c-ebb7811c4114"),

			dbError: fmt.Errorf("database connection error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockUtil := commonutil.NewMockUtilHandler(mc)

			h := &participantHandler{
				dbHandler:     mockDB,
				notifyHandler: mockNotify,
				utilHandler:   mockUtil,
			}

			ctx := context.Background()

			// Only mock database call if validation passes
			if tt.customerID != uuid.Nil && tt.chatID != uuid.Nil {
				mockDB.EXPECT().ParticipantList(ctx, gomock.Any()).Return(nil, tt.dbError)
			}

			res, err := h.ParticipantList(ctx, tt.customerID, tt.chatID)
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: ok")
			}

			if res != nil {
				t.Errorf("Wrong match. expect: nil result, got: %v", res)
			}
		})
	}
}

func Test_ParticipantRemove(t *testing.T) {
	tests := []struct {
		name string

		customerID    uuid.UUID
		participantID uuid.UUID

		responseParticipant *participant.Participant
	}{
		{
			name: "normal_hard_delete",

			customerID:    uuid.FromStringOrNil("809656e2-305e-43cd-8d7b-ccb44373dddb"),
			participantID: uuid.FromStringOrNil("af243cbc-de04-4705-ad2b-78350d0a4fba"),

			responseParticipant: &participant.Participant{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("af243cbc-de04-4705-ad2b-78350d0a4fba"),
					CustomerID: uuid.FromStringOrNil("809656e2-305e-43cd-8d7b-ccb44373dddb"),
				},
				Owner: commonidentity.Owner{
					OwnerType: "agent",
					OwnerID:   uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
				},
				ChatID:   uuid.FromStringOrNil("ba3ad8aa-cb0d-47fe-beef-f7c76c61a9f4"),
				TMJoined: "2024-01-15 10:30:00.000000",
			},
		},
		{
			name: "normal_agent_leaving_chat",

			customerID:    uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
			participantID: uuid.FromStringOrNil("6a9a0ed0-1bcb-46de-a225-e638bbaf2fc1"),

			responseParticipant: &participant.Participant{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("6a9a0ed0-1bcb-46de-a225-e638bbaf2fc1"),
					CustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
				},
				Owner: commonidentity.Owner{
					OwnerType: "agent",
					OwnerID:   uuid.FromStringOrNil("31536998-da36-11ee-976a-b31b049d62c2"),
				},
				ChatID:   uuid.FromStringOrNil("ac810dc4-298c-11ee-984c-ebb7811c4114"),
				TMJoined: "2024-01-15 11:00:00.000000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockUtil := commonutil.NewMockUtilHandler(mc)

			h := &participantHandler{
				dbHandler:     mockDB,
				notifyHandler: mockNotify,
				utilHandler:   mockUtil,
			}

			ctx := context.Background()

			// First get participant before deletion (for webhook payload)
			mockDB.EXPECT().ParticipantGet(ctx, tt.participantID).Return(tt.responseParticipant, nil)

			// Hard delete from database
			mockDB.EXPECT().ParticipantDelete(ctx, tt.participantID).Return(nil)

			// Publish webhook event
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.customerID, participant.EventParticipantRemoved, tt.responseParticipant)

			err := h.ParticipantRemove(ctx, tt.customerID, tt.participantID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_ParticipantRemove_error(t *testing.T) {
	tests := []struct {
		name string

		customerID    uuid.UUID
		participantID uuid.UUID

		responseParticipant *participant.Participant
		getError            error
		deleteError         error
	}{
		{
			name: "error_nil_customer_id",

			customerID:    uuid.Nil,
			participantID: uuid.FromStringOrNil("af243cbc-de04-4705-ad2b-78350d0a4fba"),

			responseParticipant: nil,
			getError:            nil,
			deleteError:         nil,
		},
		{
			name: "error_nil_participant_id",

			customerID:    uuid.FromStringOrNil("809656e2-305e-43cd-8d7b-ccb44373dddb"),
			participantID: uuid.Nil,

			responseParticipant: nil,
			getError:            nil,
			deleteError:         nil,
		},
		{
			name: "error_participant_not_found",

			customerID:    uuid.FromStringOrNil("ba3ad8aa-cb0d-47fe-beef-f7c76c61a9f4"),
			participantID: uuid.FromStringOrNil("62b0e2b7-0583-4f78-9406-45b00d17a9b4"),

			responseParticipant: nil,
			getError:            fmt.Errorf("not found"),
			deleteError:         nil,
		},
		{
			name: "error_get_before_delete_failed",

			customerID:    uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
			participantID: uuid.FromStringOrNil("04bc94c1-9cc1-4ce8-8559-39d6f1892109"),

			responseParticipant: nil,
			getError:            fmt.Errorf("database error"),
			deleteError:         nil,
		},
		{
			name: "error_delete_failed",

			customerID:    uuid.FromStringOrNil("809656e2-305e-43cd-8d7b-ccb44373dddb"),
			participantID: uuid.FromStringOrNil("e8427fa8-17b2-4e9e-8855-90e516bcf1d3"),

			responseParticipant: &participant.Participant{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("e8427fa8-17b2-4e9e-8855-90e516bcf1d3"),
					CustomerID: uuid.FromStringOrNil("809656e2-305e-43cd-8d7b-ccb44373dddb"),
				},
				Owner: commonidentity.Owner{
					OwnerType: "agent",
					OwnerID:   uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
				},
				ChatID: uuid.FromStringOrNil("ba3ad8aa-cb0d-47fe-beef-f7c76c61a9f4"),
			},
			getError:    nil,
			deleteError: fmt.Errorf("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockUtil := commonutil.NewMockUtilHandler(mc)

			h := &participantHandler{
				dbHandler:     mockDB,
				notifyHandler: mockNotify,
				utilHandler:   mockUtil,
			}

			ctx := context.Background()

			// Only mock dependencies if validation passes
			if tt.customerID != uuid.Nil && tt.participantID != uuid.Nil {
				// Mock get call
				mockDB.EXPECT().ParticipantGet(ctx, tt.participantID).Return(tt.responseParticipant, tt.getError)

				// Only mock delete if get succeeds
				if tt.getError == nil && tt.responseParticipant != nil {
					mockDB.EXPECT().ParticipantDelete(ctx, tt.participantID).Return(tt.deleteError)
				}
			}

			err := h.ParticipantRemove(ctx, tt.customerID, tt.participantID)
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: ok")
			}
		})
	}
}
