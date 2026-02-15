package transferhandler

import (
	"context"
	"errors"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-transfer-manager/models/transfer"
	"monorepo/bin-transfer-manager/pkg/dbhandler"
)

func TestCreate(t *testing.T) {
	tests := []struct {
		name                string
		customerID          uuid.UUID
		transferType        transfer.Type
		transfererCallID    uuid.UUID
		transfereeAddresses []commonaddress.Address
		groupcallID         uuid.UUID
		confbridgeID        uuid.UUID
		dbError             error
		shouldError         bool
	}{
		{
			name:             "creates_attended_transfer_successfully",
			customerID:       uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440001"),
			transferType:     transfer.TypeAttended,
			transfererCallID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440002"),
			transfereeAddresses: []commonaddress.Address{
				{Type: commonaddress.TypeTel, Target: "+821100000001"},
			},
			groupcallID:  uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440004"),
			confbridgeID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440005"),
			dbError:      nil,
			shouldError:  false,
		},
		{
			name:             "creates_blind_transfer_successfully",
			customerID:       uuid.FromStringOrNil("660e8400-e29b-41d4-a716-446655440001"),
			transferType:     transfer.TypeBlind,
			transfererCallID: uuid.FromStringOrNil("660e8400-e29b-41d4-a716-446655440002"),
			transfereeAddresses: []commonaddress.Address{
				{Type: commonaddress.TypeSIP, Target: "sip:user@domain.com"},
			},
			groupcallID:  uuid.FromStringOrNil("660e8400-e29b-41d4-a716-446655440004"),
			confbridgeID: uuid.FromStringOrNil("660e8400-e29b-41d4-a716-446655440005"),
			dbError:      nil,
			shouldError:  false,
		},
		{
			name:             "fails_when_db_create_fails",
			customerID:       uuid.FromStringOrNil("770e8400-e29b-41d4-a716-446655440001"),
			transferType:     transfer.TypeAttended,
			transfererCallID: uuid.FromStringOrNil("770e8400-e29b-41d4-a716-446655440002"),
			transfereeAddresses: []commonaddress.Address{
				{Type: commonaddress.TypeTel, Target: "+821100000001"},
			},
			groupcallID:  uuid.FromStringOrNil("770e8400-e29b-41d4-a716-446655440004"),
			confbridgeID: uuid.FromStringOrNil("770e8400-e29b-41d4-a716-446655440005"),
			dbError:      errors.New("database error"),
			shouldError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &transferHandler{
				utilHandler:   utilhandler.NewUtilHandler(),
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().TransferCreate(ctx, gomock.Any()).Return(tt.dbError)

			result, err := h.Create(ctx, tt.customerID, tt.transferType, tt.transfererCallID, tt.transfereeAddresses, tt.groupcallID, tt.confbridgeID)

			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if result != nil {
					t.Error("Expected nil result but got transfer")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result == nil {
					t.Error("Expected transfer but got nil")
				}
				if result.CustomerID != tt.customerID {
					t.Errorf("Wrong CustomerID. expect: %s, got: %s", tt.customerID, result.CustomerID)
				}
				if result.Type != tt.transferType {
					t.Errorf("Wrong Type. expect: %s, got: %s", tt.transferType, result.Type)
				}
				if result.TransfererCallID != tt.transfererCallID {
					t.Errorf("Wrong TransfererCallID. expect: %s, got: %s", tt.transfererCallID, result.TransfererCallID)
				}
				if result.GroupcallID != tt.groupcallID {
					t.Errorf("Wrong GroupcallID. expect: %s, got: %s", tt.groupcallID, result.GroupcallID)
				}
				if result.ConfbridgeID != tt.confbridgeID {
					t.Errorf("Wrong ConfbridgeID. expect: %s, got: %s", tt.confbridgeID, result.ConfbridgeID)
				}
			}
		})
	}
}

func TestGet(t *testing.T) {
	tests := []struct {
		name        string
		transferID  uuid.UUID
		dbTransfer  *transfer.Transfer
		dbError     error
		shouldError bool
	}{
		{
			name:       "gets_transfer_successfully",
			transferID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
			dbTransfer: &transfer.Transfer{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
					CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440001"),
				},
				Type: transfer.TypeAttended,
			},
			dbError:     nil,
			shouldError: false,
		},
		{
			name:        "fails_when_transfer_not_found",
			transferID:  uuid.FromStringOrNil("660e8400-e29b-41d4-a716-446655440000"),
			dbTransfer:  nil,
			dbError:     errors.New("not found"),
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &transferHandler{
				utilHandler:   utilhandler.NewUtilHandler(),
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().TransferGet(ctx, tt.transferID).Return(tt.dbTransfer, tt.dbError)

			result, err := h.Get(ctx, tt.transferID)

			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result == nil {
					t.Error("Expected transfer but got nil")
				}
				if result.ID != tt.dbTransfer.ID {
					t.Errorf("Wrong ID. expect: %s, got: %s", tt.dbTransfer.ID, result.ID)
				}
			}
		})
	}
}

func TestGetByGroupcallID(t *testing.T) {
	tests := []struct {
		name        string
		groupcallID uuid.UUID
		dbTransfer  *transfer.Transfer
		dbError     error
		shouldError bool
	}{
		{
			name:        "gets_transfer_by_groupcall_id_successfully",
			groupcallID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440004"),
			dbTransfer: &transfer.Transfer{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
					CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440001"),
				},
				GroupcallID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440004"),
			},
			dbError:     nil,
			shouldError: false,
		},
		{
			name:        "fails_when_transfer_not_found",
			groupcallID: uuid.FromStringOrNil("660e8400-e29b-41d4-a716-446655440004"),
			dbTransfer:  nil,
			dbError:     errors.New("not found"),
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &transferHandler{
				utilHandler:   utilhandler.NewUtilHandler(),
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().TransferGetByGroupcallID(ctx, tt.groupcallID).Return(tt.dbTransfer, tt.dbError)

			result, err := h.GetByGroupcallID(ctx, tt.groupcallID)

			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result == nil {
					t.Error("Expected transfer but got nil")
				}
			}
		})
	}
}

func TestGetByTransfererCallID(t *testing.T) {
	tests := []struct {
		name             string
		transfererCallID uuid.UUID
		dbTransfer       *transfer.Transfer
		dbError          error
		shouldError      bool
	}{
		{
			name:             "gets_transfer_by_transferer_call_id_successfully",
			transfererCallID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440002"),
			dbTransfer: &transfer.Transfer{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
					CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440001"),
				},
				TransfererCallID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440002"),
			},
			dbError:     nil,
			shouldError: false,
		},
		{
			name:             "fails_when_transfer_not_found",
			transfererCallID: uuid.FromStringOrNil("660e8400-e29b-41d4-a716-446655440002"),
			dbTransfer:       nil,
			dbError:          errors.New("not found"),
			shouldError:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &transferHandler{
				utilHandler:   utilhandler.NewUtilHandler(),
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().TransferGetByTransfererCallID(ctx, tt.transfererCallID).Return(tt.dbTransfer, tt.dbError)

			result, err := h.GetByTransfererCallID(ctx, tt.transfererCallID)

			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result == nil {
					t.Error("Expected transfer but got nil")
				}
			}
		})
	}
}

func Test_updateTransfereeCallID(t *testing.T) {
	tests := []struct {
		name             string
		transferID       uuid.UUID
		transfereeCallID uuid.UUID
		getTransfer      *transfer.Transfer
		getError         error
		updateError      error
		shouldError      bool
	}{
		{
			name:             "updates_transferee_call_id_successfully",
			transferID:       uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
			transfereeCallID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440003"),
			getTransfer: &transfer.Transfer{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
					CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440001"),
				},
				Type:             transfer.TypeAttended,
				TransfereeCallID: uuid.Nil,
			},
			getError:    nil,
			updateError: nil,
			shouldError: false,
		},
		{
			name:             "fails_when_first_get_fails",
			transferID:       uuid.FromStringOrNil("660e8400-e29b-41d4-a716-446655440000"),
			transfereeCallID: uuid.FromStringOrNil("660e8400-e29b-41d4-a716-446655440003"),
			getTransfer:      nil,
			getError:         errors.New("not found"),
			updateError:      nil,
			shouldError:      true,
		},
		{
			name:             "fails_when_update_fails",
			transferID:       uuid.FromStringOrNil("770e8400-e29b-41d4-a716-446655440000"),
			transfereeCallID: uuid.FromStringOrNil("770e8400-e29b-41d4-a716-446655440003"),
			getTransfer: &transfer.Transfer{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("770e8400-e29b-41d4-a716-446655440000"),
					CustomerID: uuid.FromStringOrNil("770e8400-e29b-41d4-a716-446655440001"),
				},
				Type: transfer.TypeBlind,
			},
			getError:    nil,
			updateError: errors.New("update failed"),
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &transferHandler{
				utilHandler:   utilhandler.NewUtilHandler(),
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			// First Get call
			mockDB.EXPECT().TransferGet(ctx, tt.transferID).Return(tt.getTransfer, tt.getError)

			if tt.getError == nil {
				// Update call
				mockDB.EXPECT().TransferUpdate(ctx, gomock.Any()).Return(tt.updateError)

				if tt.updateError == nil {
					// Second Get call after update
					updatedTransfer := &transfer.Transfer{
						Identity: commonidentity.Identity{
							ID:         tt.transferID,
							CustomerID: tt.getTransfer.CustomerID,
						},
						Type:             tt.getTransfer.Type,
						TransfereeCallID: tt.transfereeCallID,
					}
					mockDB.EXPECT().TransferGet(ctx, tt.transferID).Return(updatedTransfer, nil)
				}
			}

			result, err := h.updateTransfereeCallID(ctx, tt.transferID, tt.transfereeCallID)

			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result == nil {
					t.Error("Expected transfer but got nil")
				}
				if result.TransfereeCallID != tt.transfereeCallID {
					t.Errorf("Wrong TransfereeCallID. expect: %s, got: %s", tt.transfereeCallID, result.TransfereeCallID)
				}
			}
		})
	}
}
