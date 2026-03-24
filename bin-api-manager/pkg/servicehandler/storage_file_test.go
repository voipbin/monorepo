package servicehandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

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

func Test_StorageFileDownloadRedirect(t *testing.T) {

	futureTime := time.Now().Add(1 * time.Hour)
	pastTime := time.Now().Add(-1 * time.Hour)

	tests := []struct {
		name string

		agent  *amagent.Agent
		fileID uuid.UUID

		responseStorageFile  *smfile.File
		responseStorageErr   error
		responseRefreshURI   string
		responseRefreshErr   error
		expectRefreshCalled  bool
		expectRes            string
		expectErr            bool
	}{
		{
			name: "valid URL not expired",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d1b2c3d4-1111-1111-1111-000000000001"),
					CustomerID: uuid.FromStringOrNil("d1b2c3d4-1111-1111-1111-000000000002"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			fileID: uuid.FromStringOrNil("d1b2c3d4-1111-1111-1111-000000000003"),

			responseStorageFile: &smfile.File{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d1b2c3d4-1111-1111-1111-000000000003"),
					CustomerID: uuid.FromStringOrNil("d1b2c3d4-1111-1111-1111-000000000002"),
				},
				URIDownload:      "https://storage.example.com/file.txt?token=valid",
				TMDownloadExpire: &futureTime,
			},
			expectRefreshCalled: false,
			expectRes:           "https://storage.example.com/file.txt?token=valid",
			expectErr:           false,
		},
		{
			name: "expired URL triggers refresh",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d1b2c3d4-2222-2222-2222-000000000001"),
					CustomerID: uuid.FromStringOrNil("d1b2c3d4-2222-2222-2222-000000000002"),
				},
				Permission: amagent.PermissionCustomerManager,
			},
			fileID: uuid.FromStringOrNil("d1b2c3d4-2222-2222-2222-000000000003"),

			responseStorageFile: &smfile.File{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d1b2c3d4-2222-2222-2222-000000000003"),
					CustomerID: uuid.FromStringOrNil("d1b2c3d4-2222-2222-2222-000000000002"),
				},
				URIDownload:      "https://storage.example.com/file.txt?token=expired",
				TMDownloadExpire: &pastTime,
			},
			responseRefreshURI:  "https://storage.example.com/file.txt?token=fresh",
			expectRefreshCalled: true,
			expectRes:           "https://storage.example.com/file.txt?token=fresh",
			expectErr:           false,
		},
		{
			name: "nil expiration triggers refresh",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d1b2c3d4-3333-3333-3333-000000000001"),
					CustomerID: uuid.FromStringOrNil("d1b2c3d4-3333-3333-3333-000000000002"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			fileID: uuid.FromStringOrNil("d1b2c3d4-3333-3333-3333-000000000003"),

			responseStorageFile: &smfile.File{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d1b2c3d4-3333-3333-3333-000000000003"),
					CustomerID: uuid.FromStringOrNil("d1b2c3d4-3333-3333-3333-000000000002"),
				},
				URIDownload:      "https://storage.example.com/file.txt?token=old",
				TMDownloadExpire: nil,
			},
			responseRefreshURI:  "https://storage.example.com/file.txt?token=refreshed",
			expectRefreshCalled: true,
			expectRes:           "https://storage.example.com/file.txt?token=refreshed",
			expectErr:           false,
		},
		{
			name: "empty URIDownload triggers refresh",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d1b2c3d4-4444-4444-4444-000000000001"),
					CustomerID: uuid.FromStringOrNil("d1b2c3d4-4444-4444-4444-000000000002"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			fileID: uuid.FromStringOrNil("d1b2c3d4-4444-4444-4444-000000000003"),

			responseStorageFile: &smfile.File{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d1b2c3d4-4444-4444-4444-000000000003"),
					CustomerID: uuid.FromStringOrNil("d1b2c3d4-4444-4444-4444-000000000002"),
				},
				URIDownload:      "",
				TMDownloadExpire: &futureTime,
			},
			responseRefreshURI:  "https://storage.example.com/file.txt?token=new",
			expectRefreshCalled: true,
			expectRes:           "https://storage.example.com/file.txt?token=new",
			expectErr:           false,
		},
		{
			name: "permission denied - different customer",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d1b2c3d4-5555-5555-5555-000000000001"),
					CustomerID: uuid.FromStringOrNil("d1b2c3d4-5555-5555-5555-000000000002"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			fileID: uuid.FromStringOrNil("d1b2c3d4-5555-5555-5555-000000000003"),

			responseStorageFile: &smfile.File{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d1b2c3d4-5555-5555-5555-000000000003"),
					CustomerID: uuid.FromStringOrNil("d1b2c3d4-5555-5555-5555-999999999999"), // different customer
				},
				URIDownload:      "https://storage.example.com/file.txt?token=valid",
				TMDownloadExpire: &futureTime,
			},
			expectRefreshCalled: false,
			expectErr:           true,
		},
		{
			name: "file not found",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d1b2c3d4-6666-6666-6666-000000000001"),
					CustomerID: uuid.FromStringOrNil("d1b2c3d4-6666-6666-6666-000000000002"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			fileID: uuid.FromStringOrNil("d1b2c3d4-6666-6666-6666-000000000003"),

			responseStorageFile: nil,
			responseStorageErr:  fmt.Errorf("file not found"),
			expectRefreshCalled: false,
			expectErr:           true,
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

			mockReq.EXPECT().StorageV1FileGet(ctx, tt.fileID).Return(tt.responseStorageFile, tt.responseStorageErr)

			if tt.expectRefreshCalled {
				mockReq.EXPECT().StorageV1FileDownloadURIRefresh(ctx, tt.fileID).Return(tt.responseRefreshURI, tt.responseRefreshErr)
			}

			res, err := h.StorageFileDownloadRedirect(ctx, tt.agent, tt.fileID)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: ok")
				}
				return
			}

			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res != tt.expectRes {
				t.Errorf("Wrong match.\nexpect: %s\ngot: %s", tt.expectRes, res)
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
