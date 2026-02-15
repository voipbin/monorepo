package dbhandler

import (
	"context"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-transfer-manager/models/transfer"
	"monorepo/bin-transfer-manager/pkg/cachehandler"
)

func TestTransferCreate(t *testing.T) {
	tests := []struct {
		name        string
		transfer    *transfer.Transfer
		cacheError  error
		shouldError bool
	}{
		{
			name: "creates_transfer_successfully",
			transfer: &transfer.Transfer{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
					CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440001"),
				},
				Type:             transfer.TypeAttended,
				TransfererCallID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440002"),
				TransfereeAddresses: []commonaddress.Address{
					{Type: commonaddress.TypeTel, Target: "+821100000001"},
				},
				GroupcallID:  uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440004"),
				ConfbridgeID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440005"),
			},
			cacheError:  nil,
			shouldError: false,
		},
		{
			name: "fails_when_cache_set_fails",
			transfer: &transfer.Transfer{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("660e8400-e29b-41d4-a716-446655440000"),
					CustomerID: uuid.FromStringOrNil("660e8400-e29b-41d4-a716-446655440001"),
				},
				Type:             transfer.TypeBlind,
				TransfererCallID: uuid.FromStringOrNil("660e8400-e29b-41d4-a716-446655440002"),
				TransfereeAddresses: []commonaddress.Address{
					{Type: commonaddress.TypeSIP, Target: "sip:user@domain.com"},
				},
				GroupcallID:  uuid.FromStringOrNil("660e8400-e29b-41d4-a716-446655440004"),
				ConfbridgeID: uuid.FromStringOrNil("660e8400-e29b-41d4-a716-446655440005"),
			},
			cacheError:  ErrNotFound,
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := NewHandler(dbTest, mockCache)

			ctx := context.Background()

			mockCache.EXPECT().TransferSet(ctx, gomock.Any()).Return(tt.cacheError)

			err := h.TransferCreate(ctx, tt.transfer)

			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				// Verify timestamps were set
				if tt.transfer.TMCreate == nil {
					t.Error("TMCreate should be set")
				}
				if tt.transfer.TMUpdate != nil {
					t.Error("TMUpdate should be nil for new transfer")
				}
				if tt.transfer.TMDelete != nil {
					t.Error("TMDelete should be nil for new transfer")
				}
			}
		})
	}
}

func TestTransferGet(t *testing.T) {
	tests := []struct {
		name            string
		transferID      uuid.UUID
		cacheTransfer   *transfer.Transfer
		cacheError      error
		shouldError     bool
		expectedTransfer *transfer.Transfer
	}{
		{
			name:       "gets_transfer_successfully",
			transferID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
			cacheTransfer: &transfer.Transfer{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
					CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440001"),
				},
				Type:             transfer.TypeAttended,
				TransfererCallID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440002"),
			},
			cacheError:  nil,
			shouldError: false,
			expectedTransfer: &transfer.Transfer{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
					CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440001"),
				},
				Type:             transfer.TypeAttended,
				TransfererCallID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440002"),
			},
		},
		{
			name:          "fails_when_transfer_not_found",
			transferID:    uuid.FromStringOrNil("660e8400-e29b-41d4-a716-446655440000"),
			cacheTransfer: nil,
			cacheError:    ErrNotFound,
			shouldError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := NewHandler(dbTest, mockCache)

			ctx := context.Background()

			mockCache.EXPECT().TransferGet(ctx, tt.transferID).Return(tt.cacheTransfer, tt.cacheError)

			result, err := h.TransferGet(ctx, tt.transferID)

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
				if result.ID != tt.expectedTransfer.ID {
					t.Errorf("Wrong ID. expect: %s, got: %s", tt.expectedTransfer.ID, result.ID)
				}
			}
		})
	}
}

func TestTransferGetByTransfererCallID(t *testing.T) {
	tests := []struct {
		name               string
		transfererCallID   uuid.UUID
		cacheTransfer      *transfer.Transfer
		cacheError         error
		shouldError        bool
		expectedTransferID uuid.UUID
	}{
		{
			name:             "gets_transfer_by_transferer_call_id_successfully",
			transfererCallID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440002"),
			cacheTransfer: &transfer.Transfer{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
					CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440001"),
				},
				Type:             transfer.TypeAttended,
				TransfererCallID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440002"),
			},
			cacheError:         nil,
			shouldError:        false,
			expectedTransferID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
		},
		{
			name:             "fails_when_transfer_not_found",
			transfererCallID: uuid.FromStringOrNil("660e8400-e29b-41d4-a716-446655440002"),
			cacheTransfer:    nil,
			cacheError:       ErrNotFound,
			shouldError:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := NewHandler(dbTest, mockCache)

			ctx := context.Background()

			mockCache.EXPECT().TransferGetByTransfererCallID(ctx, tt.transfererCallID).Return(tt.cacheTransfer, tt.cacheError)

			result, err := h.TransferGetByTransfererCallID(ctx, tt.transfererCallID)

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
				if result.ID != tt.expectedTransferID {
					t.Errorf("Wrong ID. expect: %s, got: %s", tt.expectedTransferID, result.ID)
				}
			}
		})
	}
}

func TestTransferGetByGroupcallID(t *testing.T) {
	tests := []struct {
		name               string
		groupcallID        uuid.UUID
		cacheTransfer      *transfer.Transfer
		cacheError         error
		shouldError        bool
		expectedTransferID uuid.UUID
	}{
		{
			name:        "gets_transfer_by_groupcall_id_successfully",
			groupcallID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440004"),
			cacheTransfer: &transfer.Transfer{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
					CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440001"),
				},
				Type:        transfer.TypeBlind,
				GroupcallID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440004"),
			},
			cacheError:         nil,
			shouldError:        false,
			expectedTransferID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
		},
		{
			name:          "fails_when_transfer_not_found",
			groupcallID:   uuid.FromStringOrNil("660e8400-e29b-41d4-a716-446655440004"),
			cacheTransfer: nil,
			cacheError:    ErrNotFound,
			shouldError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := NewHandler(dbTest, mockCache)

			ctx := context.Background()

			mockCache.EXPECT().TransferGetByGroupcallID(ctx, tt.groupcallID).Return(tt.cacheTransfer, tt.cacheError)

			result, err := h.TransferGetByGroupcallID(ctx, tt.groupcallID)

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
				if result.ID != tt.expectedTransferID {
					t.Errorf("Wrong ID. expect: %s, got: %s", tt.expectedTransferID, result.ID)
				}
			}
		})
	}
}

func TestTransferUpdate(t *testing.T) {
	tests := []struct {
		name        string
		transfer    *transfer.Transfer
		cacheError  error
		shouldError bool
	}{
		{
			name: "updates_transfer_successfully",
			transfer: &transfer.Transfer{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
					CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440001"),
				},
				Type:             transfer.TypeAttended,
				TransfererCallID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440002"),
				TransfereeCallID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440003"),
			},
			cacheError:  nil,
			shouldError: false,
		},
		{
			name: "fails_when_cache_set_fails",
			transfer: &transfer.Transfer{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("660e8400-e29b-41d4-a716-446655440000"),
					CustomerID: uuid.FromStringOrNil("660e8400-e29b-41d4-a716-446655440001"),
				},
				Type:             transfer.TypeBlind,
				TransfererCallID: uuid.FromStringOrNil("660e8400-e29b-41d4-a716-446655440002"),
			},
			cacheError:  ErrNotFound,
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := NewHandler(dbTest, mockCache)

			ctx := context.Background()

			mockCache.EXPECT().TransferSet(ctx, gomock.Any()).Return(tt.cacheError)

			err := h.TransferUpdate(ctx, tt.transfer)

			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				// Verify TMUpdate was set
				if tt.transfer.TMUpdate == nil {
					t.Error("TMUpdate should be set")
				}
			}
		})
	}
}
