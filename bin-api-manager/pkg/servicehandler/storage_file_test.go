package servicehandler

import (
	"context"
	"reflect"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	smfile "monorepo/bin-storage-manager/models/file"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/pkg/dbhandler"
)

func Test_StorageFileGet(t *testing.T) {

	tests := []struct {
		name string

		agent  *amagent.Agent
		fileID uuid.UUID

		responseStorageFile *smfile.File
		expectRes           *smfile.WebhookMessage
	}{
		{
			name: "normal",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a1b2c3d4-1111-1111-1111-000000000001"),
					CustomerID: uuid.FromStringOrNil("a1b2c3d4-1111-1111-1111-000000000002"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			fileID: uuid.FromStringOrNil("a1b2c3d4-1111-1111-1111-000000000003"),

			responseStorageFile: &smfile.File{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a1b2c3d4-1111-1111-1111-000000000003"),
					CustomerID: uuid.FromStringOrNil("a1b2c3d4-1111-1111-1111-000000000002"),
				},
				Owner: commonidentity.Owner{
					OwnerID: uuid.FromStringOrNil("a1b2c3d4-1111-1111-1111-000000000001"),
				},
				TMDelete: nil,
			},
			expectRes: &smfile.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a1b2c3d4-1111-1111-1111-000000000003"),
					CustomerID: uuid.FromStringOrNil("a1b2c3d4-1111-1111-1111-000000000002"),
				},
				Owner: commonidentity.Owner{
					OwnerID: uuid.FromStringOrNil("a1b2c3d4-1111-1111-1111-000000000001"),
				},
				TMDelete: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().StorageV1FileGet(ctx, tt.fileID).Return(tt.responseStorageFile, nil)

			res, err := h.StorageFileGet(ctx, tt.agent, tt.fileID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_StorageFileDelete(t *testing.T) {

	tests := []struct {
		name string

		agent  *amagent.Agent
		fileID uuid.UUID

		responseStorageFile *smfile.File
		expectRes           *smfile.WebhookMessage
	}{
		{
			name: "normal",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b1b2c3d4-2222-2222-2222-000000000001"),
					CustomerID: uuid.FromStringOrNil("b1b2c3d4-2222-2222-2222-000000000002"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			fileID: uuid.FromStringOrNil("b1b2c3d4-2222-2222-2222-000000000003"),

			responseStorageFile: &smfile.File{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b1b2c3d4-2222-2222-2222-000000000003"),
					CustomerID: uuid.FromStringOrNil("b1b2c3d4-2222-2222-2222-000000000002"),
				},
				Owner: commonidentity.Owner{
					OwnerID: uuid.FromStringOrNil("b1b2c3d4-2222-2222-2222-000000000001"),
				},
				TMDelete: nil,
			},
			expectRes: &smfile.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b1b2c3d4-2222-2222-2222-000000000003"),
					CustomerID: uuid.FromStringOrNil("b1b2c3d4-2222-2222-2222-000000000002"),
				},
				Owner: commonidentity.Owner{
					OwnerID: uuid.FromStringOrNil("b1b2c3d4-2222-2222-2222-000000000001"),
				},
				TMDelete: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().StorageV1FileGet(ctx, tt.fileID).Return(tt.responseStorageFile, nil)
			mockReq.EXPECT().StorageV1FileDelete(ctx, tt.fileID, 60000).Return(tt.responseStorageFile, nil)

			res, err := h.StorageFileDelete(ctx, tt.agent, tt.fileID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_StorageFileList_StorageFile(t *testing.T) {

	tests := []struct {
		name string

		agent *amagent.Agent
		size  uint64
		token string

		responseStorageFiles []smfile.File
		expectFilters        map[smfile.Field]any
		expectRes            []*smfile.WebhookMessage
	}{
		{
			name: "normal",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c1b2c3d4-3333-3333-3333-000000000001"),
					CustomerID: uuid.FromStringOrNil("c1b2c3d4-3333-3333-3333-000000000002"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			size:  10,
			token: "2020-09-20T03:23:20.995000Z",

			responseStorageFiles: []smfile.File{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("c1b2c3d4-3333-3333-3333-000000000003"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("c1b2c3d4-3333-3333-3333-000000000004"),
					},
				},
			},
			expectFilters: map[smfile.Field]any{
				smfile.FieldCustomerID: uuid.FromStringOrNil("c1b2c3d4-3333-3333-3333-000000000002"),
				smfile.FieldDeleted:    false,
			},
			expectRes: []*smfile.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("c1b2c3d4-3333-3333-3333-000000000003"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("c1b2c3d4-3333-3333-3333-000000000004"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().StorageV1FileList(ctx, tt.token, tt.size, tt.expectFilters).Return(tt.responseStorageFiles, nil)

			res, err := h.StorageFileList(ctx, tt.agent, tt.size, tt.token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
