package extensiondirecthandler

import (
	"context"
	"errors"
	"testing"

	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-registrar-manager/models/extensiondirect"
	"monorepo/bin-registrar-manager/pkg/dbhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func TestCreate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	handler := NewExtensionDirectHandler(mockDB).(*extensionDirectHandler)

	customerID := uuid.Must(uuid.NewV4())
	extensionID := uuid.Must(uuid.NewV4())
	ctx := context.Background()

	tests := []struct {
		name    string
		setup   func()
		wantErr bool
	}{
		{
			name: "create_new_extension_direct_success",
			setup: func() {
				// No existing record
				mockDB.EXPECT().
					ExtensionDirectGetByExtensionID(ctx, extensionID).
					Return(nil, errors.New("not found"))

				mockDB.EXPECT().
					ExtensionDirectCreate(ctx, gomock.Any()).
					Return(nil)

				mockDB.EXPECT().
					ExtensionDirectGet(ctx, gomock.Any()).
					Return(&extensiondirect.ExtensionDirect{
						Identity: identity.Identity{
							ID:         uuid.Must(uuid.NewV4()),
							CustomerID: customerID,
						},
						ExtensionID: extensionID,
						Hash:        "abc123def456",
					}, nil)
			},
			wantErr: false,
		},
		{
			name: "extension_direct_already_exists",
			setup: func() {
				existing := &extensiondirect.ExtensionDirect{
					Identity: identity.Identity{
						ID:         uuid.Must(uuid.NewV4()),
						CustomerID: customerID,
					},
					ExtensionID: extensionID,
					Hash:        "existing123",
				}
				mockDB.EXPECT().
					ExtensionDirectGetByExtensionID(ctx, extensionID).
					Return(existing, nil)
			},
			wantErr: false,
		},
		{
			name: "create_fails_then_succeeds_on_retry",
			setup: func() {
				mockDB.EXPECT().
					ExtensionDirectGetByExtensionID(ctx, extensionID).
					Return(nil, errors.New("not found"))

				// First attempt fails
				mockDB.EXPECT().
					ExtensionDirectCreate(ctx, gomock.Any()).
					Return(errors.New("duplicate hash"))

				// Second attempt succeeds
				mockDB.EXPECT().
					ExtensionDirectCreate(ctx, gomock.Any()).
					Return(nil)

				mockDB.EXPECT().
					ExtensionDirectGet(ctx, gomock.Any()).
					Return(&extensiondirect.ExtensionDirect{
						Identity: identity.Identity{
							ID:         uuid.Must(uuid.NewV4()),
							CustomerID: customerID,
						},
						ExtensionID: extensionID,
						Hash:        "newHash123",
					}, nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			result, err := handler.Create(ctx, customerID, extensionID)
			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result == nil {
				t.Error("Create() returned nil result")
			}
		})
	}
}

func TestDelete(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	handler := NewExtensionDirectHandler(mockDB).(*extensionDirectHandler)

	id := uuid.Must(uuid.NewV4())
	ctx := context.Background()

	tests := []struct {
		name    string
		setup   func()
		wantErr bool
	}{
		{
			name: "delete_success",
			setup: func() {
				mockDB.EXPECT().
					ExtensionDirectDelete(ctx, id).
					Return(nil)

				mockDB.EXPECT().
					ExtensionDirectGet(ctx, id).
					Return(&extensiondirect.ExtensionDirect{
						Identity: identity.Identity{
							ID: id,
						},
					}, nil)
			},
			wantErr: false,
		},
		{
			name: "delete_db_error",
			setup: func() {
				mockDB.EXPECT().
					ExtensionDirectDelete(ctx, id).
					Return(errors.New("db error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			result, err := handler.Delete(ctx, id)
			if (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result == nil {
				t.Error("Delete() returned nil result")
			}
		})
	}
}

func TestGet(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	handler := NewExtensionDirectHandler(mockDB).(*extensionDirectHandler)

	id := uuid.Must(uuid.NewV4())
	ctx := context.Background()

	tests := []struct {
		name    string
		setup   func()
		wantErr bool
	}{
		{
			name: "get_success",
			setup: func() {
				mockDB.EXPECT().
					ExtensionDirectGet(ctx, id).
					Return(&extensiondirect.ExtensionDirect{
						Identity: identity.Identity{
							ID: id,
						},
					}, nil)
			},
			wantErr: false,
		},
		{
			name: "get_not_found",
			setup: func() {
				mockDB.EXPECT().
					ExtensionDirectGet(ctx, id).
					Return(nil, errors.New("not found"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			result, err := handler.Get(ctx, id)
			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result == nil {
				t.Error("Get() returned nil result")
			}
		})
	}
}

func TestGetByExtensionID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	handler := NewExtensionDirectHandler(mockDB).(*extensionDirectHandler)

	extensionID := uuid.Must(uuid.NewV4())
	ctx := context.Background()

	tests := []struct {
		name    string
		setup   func()
		wantErr bool
	}{
		{
			name: "get_by_extension_id_success",
			setup: func() {
				mockDB.EXPECT().
					ExtensionDirectGetByExtensionID(ctx, extensionID).
					Return(&extensiondirect.ExtensionDirect{
						ExtensionID: extensionID,
					}, nil)
			},
			wantErr: false,
		},
		{
			name: "get_by_extension_id_not_found",
			setup: func() {
				mockDB.EXPECT().
					ExtensionDirectGetByExtensionID(ctx, extensionID).
					Return(nil, errors.New("not found"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			result, err := handler.GetByExtensionID(ctx, extensionID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetByExtensionID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result == nil {
				t.Error("GetByExtensionID() returned nil result")
			}
		})
	}
}

func TestGetByExtensionIDs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	handler := NewExtensionDirectHandler(mockDB).(*extensionDirectHandler)

	extensionID1 := uuid.Must(uuid.NewV4())
	extensionID2 := uuid.Must(uuid.NewV4())
	extensionIDs := []uuid.UUID{extensionID1, extensionID2}
	ctx := context.Background()

	tests := []struct {
		name    string
		setup   func()
		wantErr bool
	}{
		{
			name: "get_by_extension_ids_success",
			setup: func() {
				mockDB.EXPECT().
					ExtensionDirectGetByExtensionIDs(ctx, extensionIDs).
					Return([]*extensiondirect.ExtensionDirect{
						{ExtensionID: extensionID1},
						{ExtensionID: extensionID2},
					}, nil)
			},
			wantErr: false,
		},
		{
			name: "get_by_extension_ids_error",
			setup: func() {
				mockDB.EXPECT().
					ExtensionDirectGetByExtensionIDs(ctx, extensionIDs).
					Return(nil, errors.New("db error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			result, err := handler.GetByExtensionIDs(ctx, extensionIDs)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetByExtensionIDs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result == nil {
				t.Error("GetByExtensionIDs() returned nil result")
			}
		})
	}
}

func TestGetByHash(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	handler := NewExtensionDirectHandler(mockDB).(*extensionDirectHandler)

	hash := "abc123def456"
	ctx := context.Background()

	tests := []struct {
		name    string
		setup   func()
		wantErr bool
	}{
		{
			name: "get_by_hash_success",
			setup: func() {
				mockDB.EXPECT().
					ExtensionDirectGetByHash(ctx, hash).
					Return(&extensiondirect.ExtensionDirect{
						Hash: hash,
					}, nil)
			},
			wantErr: false,
		},
		{
			name: "get_by_hash_not_found",
			setup: func() {
				mockDB.EXPECT().
					ExtensionDirectGetByHash(ctx, hash).
					Return(nil, errors.New("not found"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			result, err := handler.GetByHash(ctx, hash)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetByHash() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result == nil {
				t.Error("GetByHash() returned nil result")
			}
		})
	}
}

func TestRegenerate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	handler := NewExtensionDirectHandler(mockDB).(*extensionDirectHandler)

	id := uuid.Must(uuid.NewV4())
	ctx := context.Background()

	tests := []struct {
		name    string
		setup   func()
		wantErr bool
	}{
		{
			name: "regenerate_success",
			setup: func() {
				mockDB.EXPECT().
					ExtensionDirectUpdate(ctx, id, gomock.Any()).
					Return(nil)

				mockDB.EXPECT().
					ExtensionDirectGet(ctx, id).
					Return(&extensiondirect.ExtensionDirect{
						Identity: identity.Identity{
							ID: id,
						},
						Hash: "newHash123",
					}, nil)
			},
			wantErr: false,
		},
		{
			name: "regenerate_retry_then_success",
			setup: func() {
				// First attempt fails
				mockDB.EXPECT().
					ExtensionDirectUpdate(ctx, id, gomock.Any()).
					Return(errors.New("duplicate hash"))

				// Second attempt succeeds
				mockDB.EXPECT().
					ExtensionDirectUpdate(ctx, id, gomock.Any()).
					Return(nil)

				mockDB.EXPECT().
					ExtensionDirectGet(ctx, id).
					Return(&extensiondirect.ExtensionDirect{
						Identity: identity.Identity{
							ID: id,
						},
						Hash: "anotherHash",
					}, nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			result, err := handler.Regenerate(ctx, id)
			if (err != nil) != tt.wantErr {
				t.Errorf("Regenerate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result == nil {
				t.Error("Regenerate() returned nil result")
			}
		})
	}
}

func TestNewExtensionDirectHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	handler := NewExtensionDirectHandler(mockDB)

	if handler == nil {
		t.Error("NewExtensionDirectHandler() returned nil")
	}

	// Verify it implements the interface
	var _ ExtensionDirectHandler = handler
}

func TestGenerateHash(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"generate_hash_1"},
		{"generate_hash_2"},
		{"generate_hash_3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := generateHash()
			if err != nil {
				t.Errorf("generateHash() error = %v", err)
				return
			}
			if len(hash) != hashLength*2 {
				t.Errorf("generateHash() returned hash of length %d, expected %d", len(hash), hashLength*2)
			}
		})
	}
}
