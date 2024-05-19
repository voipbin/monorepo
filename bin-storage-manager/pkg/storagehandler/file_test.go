package storagehandler

import (
	"context"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-storage-manager/models/file"
	"monorepo/bin-storage-manager/pkg/filehandler"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
)

func Test_FileCreate(t *testing.T) {

	type test struct {
		name string

		customerID    uuid.UUID
		ownerID       uuid.UUID
		referenceType file.ReferenceType
		referenceID   uuid.UUID
		fileName      string
		detail        string
		bucketName    string
		filepath      string

		responseFile *file.File
	}

	tests := []test{
		{
			name: "normal",

			customerID:    uuid.FromStringOrNil("85e112c4-1534-11ef-8b98-bf28503ca953"),
			ownerID:       uuid.FromStringOrNil("86593c2c-1534-11ef-a02f-e32b18b3c9fe"),
			referenceType: file.ReferenceTypeNone,
			referenceID:   uuid.FromStringOrNil("43201fc0-1539-11ef-bd86-7f2a2e346825"),
			fileName:      "test name",
			detail:        "test detail",
			bucketName:    "test_bucket",
			filepath:      "/test/file/path",

			responseFile: &file.File{
				ID: uuid.FromStringOrNil("86940b22-1534-11ef-81e6-cf09fa8054e4"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockFile := filehandler.NewMockFileHandler(mc)

			h := storageHandler{
				utilHandler: mockUtil,
				reqHandler:  mockReq,
				fileHandler: mockFile,

				bucketNameMedia: "media",
			}
			ctx := context.Background()

			mockFile.EXPECT().Create(ctx, tt.customerID, tt.ownerID, tt.referenceType, tt.referenceID, tt.fileName, tt.detail, tt.bucketName, tt.filepath).Return(tt.responseFile, nil)

			res, err := h.FileCreate(ctx, tt.customerID, tt.ownerID, tt.referenceType, tt.referenceID, tt.fileName, tt.detail, tt.bucketName, tt.filepath)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseFile, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseFile, res)
			}
		})
	}
}

func Test_FileGet(t *testing.T) {

	type test struct {
		name string

		id uuid.UUID

		responseFile *file.File
	}

	tests := []test{
		{
			name: "normal",

			id: uuid.FromStringOrNil("5ddf17ac-1535-11ef-b4bc-ef75d13601ab"),

			responseFile: &file.File{
				ID: uuid.FromStringOrNil("5ddf17ac-1535-11ef-b4bc-ef75d13601ab"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockFile := filehandler.NewMockFileHandler(mc)

			h := storageHandler{
				utilHandler: mockUtil,
				reqHandler:  mockReq,
				fileHandler: mockFile,

				bucketNameMedia: "media",
			}
			ctx := context.Background()

			mockFile.EXPECT().Get(ctx, tt.id).Return(tt.responseFile, nil)

			res, err := h.FileGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseFile, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseFile, res)
			}
		})
	}
}

func Test_FileGets(t *testing.T) {

	type test struct {
		name string

		token   string
		size    uint64
		filters map[string]string

		responseFiles []*file.File
	}

	tests := []test{
		{
			name: "normal",

			token: "2024-05-16 03:22:17.995000",
			size:  10,
			filters: map[string]string{
				"customer_id": "c6f2b776-1535-11ef-a098-b38ca4e3bbb1",
			},

			responseFiles: []*file.File{
				{
					ID: uuid.FromStringOrNil("fe803ee8-1535-11ef-892e-a7b2c7cc99b0"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockFile := filehandler.NewMockFileHandler(mc)

			h := storageHandler{
				utilHandler: mockUtil,
				reqHandler:  mockReq,
				fileHandler: mockFile,

				bucketNameMedia: "media",
			}
			ctx := context.Background()

			mockFile.EXPECT().Gets(ctx, tt.token, tt.size, tt.filters).Return(tt.responseFiles, nil)

			res, err := h.FileGets(ctx, tt.token, tt.size, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseFiles, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseFiles, res)
			}
		})
	}
}

func Test_FileDelete(t *testing.T) {

	type test struct {
		name string

		id uuid.UUID

		responseFile *file.File
	}

	tests := []test{
		{
			name: "normal",

			id: uuid.FromStringOrNil("fd11a6ae-1536-11ef-86ad-9fab2a4d002f"),

			responseFile: &file.File{
				ID: uuid.FromStringOrNil("fd11a6ae-1536-11ef-86ad-9fab2a4d002f"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockFile := filehandler.NewMockFileHandler(mc)

			h := storageHandler{
				utilHandler: mockUtil,
				reqHandler:  mockReq,
				fileHandler: mockFile,

				bucketNameMedia: "media",
			}
			ctx := context.Background()

			mockFile.EXPECT().Delete(ctx, tt.id).Return(tt.responseFile, nil)

			res, err := h.FileDelete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseFile, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseFile, res)
			}
		})
	}
}
