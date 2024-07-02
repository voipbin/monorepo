package servicehandler

import (
	"context"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	smfile "monorepo/bin-storage-manager/models/file"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"monorepo/bin-api-manager/pkg/dbhandler"
)

func Test_storageFileGet(t *testing.T) {

	tests := []struct {
		name string

		agent  *amagent.Agent
		fileID uuid.UUID

		responseStorageFile *smfile.File
	}{
		{
			"normal",
			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b7ef6ea6-1bd7-11ef-88a6-ff71fa05d8bd"),
					CustomerID: uuid.FromStringOrNil("b83e3c98-1bd7-11ef-8f14-9f07e5f6c56b"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("ee294376-1c03-11ef-b40d-372468bd9437"),
			&smfile.File{
				ID:         uuid.FromStringOrNil("ee294376-1c03-11ef-b40d-372468bd9437"),
				CustomerID: uuid.FromStringOrNil("b83e3c98-1bd7-11ef-8f14-9f07e5f6c56b"),
				TMDelete:   defaultTimestamp,
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

			res, err := h.storageFileGet(ctx, tt.fileID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseStorageFile) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.responseStorageFile, res)
			}
		})
	}
}

func Test_StorageFileDelete(t *testing.T) {

	tests := []struct {
		name string

		agent         *amagent.Agent
		storageFileID uuid.UUID

		responseStorageFile *smfile.File
		expectRes           *smfile.WebhookMessage
	}{
		{
			name: "normal",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("1a49c8f8-1bd8-11ef-b861-bf0a568022b9"),
					CustomerID: uuid.FromStringOrNil("1a73a632-1bd8-11ef-8c46-4fdca968dac2"),
				},
				Permission: amagent.PermissionProjectSuperAdmin,
			},
			storageFileID: uuid.FromStringOrNil("1aa43522-1bd8-11ef-870e-4f7d5cfff4f5"),

			responseStorageFile: &smfile.File{
				ID:         uuid.FromStringOrNil("1aa43522-1bd8-11ef-870e-4f7d5cfff4f5"),
				CustomerID: uuid.FromStringOrNil("1a73a632-1bd8-11ef-8c46-4fdca968dac2"),
				TMDelete:   defaultTimestamp,
			},
			expectRes: &smfile.WebhookMessage{
				ID:         uuid.FromStringOrNil("1aa43522-1bd8-11ef-870e-4f7d5cfff4f5"),
				CustomerID: uuid.FromStringOrNil("1a73a632-1bd8-11ef-8c46-4fdca968dac2"),
				TMDelete:   defaultTimestamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().StorageV1FileGet(ctx, tt.storageFileID).Return(tt.responseStorageFile, nil)
			mockReq.EXPECT().StorageV1FileDelete(ctx, tt.storageFileID, 60000).Return(tt.responseStorageFile, nil)

			res, err := h.StorageFileDelete(ctx, tt.agent, tt.storageFileID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_StorageFileGets(t *testing.T) {

	tests := []struct {
		name string

		agent *amagent.Agent
		size  uint64
		token string

		responseStorageFiles []smfile.File
		expectFilters        map[string]string
		expectRes            []*smfile.WebhookMessage
	}{
		{
			name: "normal",
			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("6998ca62-1bd8-11ef-bfe1-f3c47f813931"),
					CustomerID: uuid.FromStringOrNil("69dc78e8-1bd8-11ef-9710-ffa2bc5ebf93"),
				},
				Permission: amagent.PermissionProjectSuperAdmin,
			},
			size:  10,
			token: "2020-09-20 03:23:20.995000",

			responseStorageFiles: []smfile.File{
				{
					ID: uuid.FromStringOrNil("6a1a3db8-1bd8-11ef-bffb-8bab4b517f52"),
				},
				{
					ID: uuid.FromStringOrNil("6a5476cc-1bd8-11ef-9863-3b26eb47b0e0"),
				},
			},
			expectFilters: map[string]string{
				"customer_id": "69dc78e8-1bd8-11ef-9710-ffa2bc5ebf93",
				"deleted":     "false",
				"owner_id":    "6998ca62-1bd8-11ef-bfe1-f3c47f813931",
			},
			expectRes: []*smfile.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("6a1a3db8-1bd8-11ef-bffb-8bab4b517f52"),
				},
				{
					ID: uuid.FromStringOrNil("6a5476cc-1bd8-11ef-9863-3b26eb47b0e0"),
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
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().StorageV1FileGets(ctx, tt.token, tt.size, tt.expectFilters).Return(tt.responseStorageFiles, nil)

			res, err := h.StorageFileGetsByOnwerID(ctx, tt.agent, tt.size, tt.token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

// func Test_StorageFileCreate(t *testing.T) {

// 	tests := []struct {
// 		name string

// 		agent      *amagent.Agent
// 		customerID uuid.UUID
// 		filePart   multipart.File
// 		fileName   string
// 		detail     string
// 		filename   string

// 		responseStorageFile *smfile.File
// 		expectRes           *smfile.WebhookMessage
// 	}{
// 		{
// 			name: "normal",

// 			agent: &amagent.Agent{
// 				ID:         uuid.FromStringOrNil("f2295586-1bd8-11ef-8610-73602171ce63"),
// 				CustomerID: uuid.FromStringOrNil("f24de752-1bd8-11ef-b438-4361eeff2690"),
// 				Permission: amagent.PermissionProjectSuperAdmin,
// 			},
// 			customerID: uuid.FromStringOrNil("e9942acc-1bd8-11ef-9c19-33ff4d2cf1ae"),

// 			fileName: "test name",
// 			detail:   "test detail",
// 			filename: "test.txt",

// 			responseStorageFile: &smfile.File{
// 				ID: uuid.FromStringOrNil("f27afcba-1bd8-11ef-a4b8-6f4d6a5ab550"),
// 			},
// 			expectRes: &smfile.WebhookMessage{
// 				ID: uuid.FromStringOrNil("f27afcba-1bd8-11ef-a4b8-6f4d6a5ab550"),
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {

// 			mc := gomock.NewController(t)
// 			defer mc.Finish()

// 			mockUtil := utilhandler.NewMockUtilHandler(mc)
// 			mockReq := requesthandler.NewMockRequestHandler(mc)
// 			mockDB := dbhandler.NewMockDBHandler(mc)
// 			h := serviceHandler{
// 				utilHandler: mockUtil,
// 				reqHandler:  mockReq,
// 				dbHandler:   mockDB,
// 			}
// 			ctx := context.Background()

// 			mockUtil.EXPECT().UUIDCreate().Return(utilhandler.UUIDCreate())
// 			mockReq.EXPECT().StorageV1FileCreate(ctx, tt.agent.CustomerID, tt.agent.ID, smfile.ReferenceTypeNone, uuid.Nil, tt.fileName, tt.detail, tt.filename, gomock.Any(), gomock.Any(), 60000).Return(tt.responseStorageFile, nil)

// 			res, err := h.StorageFileCreate(ctx, tt.agent, nil, tt.fileName, tt.detail, tt.filename)
// 			if err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}

// 			if !reflect.DeepEqual(tt.expectRes, res) {
// 				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
// 			}
// 		})
// 	}
// }
