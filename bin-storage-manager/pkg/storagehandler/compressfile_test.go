package storagehandler

import (
	"context"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	compressfile "monorepo/bin-storage-manager/models/compressfile"
	"monorepo/bin-storage-manager/models/file"
	"monorepo/bin-storage-manager/pkg/filehandler"
	reflect "reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_CompressCreate(t *testing.T) {

	type test struct {
		name string

		referenceIDs []uuid.UUID
		fileIDs      []uuid.UUID

		responseFilesByFileIDs      []*file.File
		responseFilesByReferenceIDs [][]*file.File
		responseFilepath            string
		responseDownloadURI         string
		responseCurTimeAdd          string

		expectSourcefilePaths []string
		expectFiles           []*file.File
		expectRes             *compressfile.CompressFile
	}

	tests := []test{
		{
			name: "normal only file_ids has given",

			referenceIDs: []uuid.UUID{
				uuid.FromStringOrNil("ca2d9b96-1d6d-11ef-9cfc-87a44a8fe35c"),
			},
			fileIDs: []uuid.UUID{
				uuid.FromStringOrNil("4ada1a3e-1d6a-11ef-99c3-076e46349385"),
			},

			responseFilesByFileIDs: []*file.File{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("4ada1a3e-1d6a-11ef-99c3-076e46349385"),
					},
					Filepath: "test/file/path/4ada1a3e-1d6a-11ef-99c3-076e46349385",
				},
			},
			responseFilesByReferenceIDs: [][]*file.File{
				{
					{
						Identity: commonidentity.Identity{
							ID: uuid.FromStringOrNil("536b5c64-1d6d-11ef-bf65-1ffae2d556ab"),
						},
						Filepath: "test/file/path/536b5c64-1d6d-11ef-bf65-1ffae2d556ab",
					},
				},
			},
			responseFilepath:    "test/result/file/path/3bb54938-1d6b-11ef-996a-b79925fc2e03",
			responseDownloadURI: "http://localhost/download/uri",
			responseCurTimeAdd:  "2024-05-16 03:22:17.995000",

			expectSourcefilePaths: []string{
				"test/file/path/4ada1a3e-1d6a-11ef-99c3-076e46349385",
				"test/file/path/536b5c64-1d6d-11ef-bf65-1ffae2d556ab",
			},
			expectFiles: []*file.File{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("4ada1a3e-1d6a-11ef-99c3-076e46349385"),
					},
					Filepath: "test/file/path/4ada1a3e-1d6a-11ef-99c3-076e46349385",
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("536b5c64-1d6d-11ef-bf65-1ffae2d556ab"),
					},
					Filepath: "test/file/path/536b5c64-1d6d-11ef-bf65-1ffae2d556ab",
				},
			},
			expectRes: &compressfile.CompressFile{
				FileIDs: []uuid.UUID{
					uuid.FromStringOrNil("4ada1a3e-1d6a-11ef-99c3-076e46349385"),
					uuid.FromStringOrNil("536b5c64-1d6d-11ef-bf65-1ffae2d556ab"),
				},
				DownloadURI:      "http://localhost/download/uri",
				TMDownloadExpire: "2024-05-16 03:22:17.995000",
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

			for i, id := range tt.fileIDs {
				mockFile.EXPECT().Get(ctx, id).Return(tt.responseFilesByFileIDs[i], nil)
			}
			for i, id := range tt.referenceIDs {
				filters := map[file.Field]any{
					file.FieldDeleted:     false,
					file.FieldReferenceID: id,
				}
				mockFile.EXPECT().Gets(ctx, "", uint64(1000), filters).Return(tt.responseFilesByReferenceIDs[i], nil)
			}

			// mockFile.EXPECT().CompressCreateRaw(ctx, h.bucketNameMedia, tt.expectSourcefilePaths).Return(h.bucketNameMedia, tt.responseFilepath, nil)
			mockFile.EXPECT().CompressCreate(ctx, tt.expectFiles).Return(h.bucketNameMedia, tt.responseFilepath, nil)
			mockFile.EXPECT().DownloadURIGet(ctx, h.bucketNameMedia, tt.responseFilepath, time.Hour*24).Return("", tt.responseDownloadURI, nil)
			mockUtil.EXPECT().TimeGetCurTimeAdd(24 * time.Hour).Return(tt.responseCurTimeAdd)

			res, err := h.CompressfileCreate(ctx, tt.referenceIDs, tt.fileIDs)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_compressGetFilesByReferenceIDs(t *testing.T) {

	type test struct {
		name string

		referenceIDs []uuid.UUID

		responseFiles [][]*file.File
		expectRes     []*file.File
	}

	tests := []test{
		{
			name: "normal",

			referenceIDs: []uuid.UUID{
				uuid.FromStringOrNil("b3c52e46-1d68-11ef-8f18-ef993b964a77"),
				uuid.FromStringOrNil("b4027e40-1d68-11ef-84f1-9feb2b2bdc77"),
			},

			responseFiles: [][]*file.File{
				{
					{
						Identity: commonidentity.Identity{
							ID: uuid.FromStringOrNil("d61c666c-1d68-11ef-8744-17f98a45c5ef"),
						},
					},
					{
						Identity: commonidentity.Identity{
							ID: uuid.FromStringOrNil("d646bbec-1d68-11ef-a64e-b321b4fb881e"),
						},
					},
				},
				{
					{
						Identity: commonidentity.Identity{
							ID: uuid.FromStringOrNil("4d274c5e-1d69-11ef-b65d-076feda109fe"),
						},
					},
					{
						Identity: commonidentity.Identity{
							ID: uuid.FromStringOrNil("4d577500-1d69-11ef-bfa6-73b9e46b5a44"),
						},
					},
				},
			},
			expectRes: []*file.File{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("d61c666c-1d68-11ef-8744-17f98a45c5ef"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("d646bbec-1d68-11ef-a64e-b321b4fb881e"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("4d274c5e-1d69-11ef-b65d-076feda109fe"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("4d577500-1d69-11ef-bfa6-73b9e46b5a44"),
					},
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

			for i, id := range tt.referenceIDs {
				filters := map[file.Field]any{
					file.FieldDeleted:     false,
					file.FieldReferenceID: id,
				}
				mockFile.EXPECT().Gets(ctx, "", uint64(1000), filters).Return(tt.responseFiles[i], nil)
			}

			res, err := h.compressGetFilesByReferenceIDs(ctx, tt.referenceIDs)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_compressGetFilesByFileIDs(t *testing.T) {

	type test struct {
		name string

		fileIDs []uuid.UUID

		responseFiles []*file.File
		expectRes     []*file.File
	}

	tests := []test{
		{
			name: "normal",

			fileIDs: []uuid.UUID{
				uuid.FromStringOrNil("95c8ef3a-1d69-11ef-b139-9b94b9c069db"),
				uuid.FromStringOrNil("962e1d38-1d69-11ef-9014-e3361107ef8d"),
			},

			responseFiles: []*file.File{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("95c8ef3a-1d69-11ef-b139-9b94b9c069db"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("962e1d38-1d69-11ef-9014-e3361107ef8d"),
					},
				},
			},
			expectRes: []*file.File{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("95c8ef3a-1d69-11ef-b139-9b94b9c069db"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("962e1d38-1d69-11ef-9014-e3361107ef8d"),
					},
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

			for i, id := range tt.fileIDs {
				mockFile.EXPECT().Get(ctx, id).Return(tt.responseFiles[i], nil)
			}

			res, err := h.compressGetFilesByFileIDs(ctx, tt.fileIDs)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
