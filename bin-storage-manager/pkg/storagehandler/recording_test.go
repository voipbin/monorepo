package storagehandler

import (
	"context"
	reflect "reflect"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-storage-manager/models/bucketfile"
	"monorepo/bin-storage-manager/models/file"
	"monorepo/bin-storage-manager/pkg/filehandler"
)

func Test_RecordingGet(t *testing.T) {

	type test struct {
		name string

		recordingID uuid.UUID

		responseFiles []*file.File

		filepath            string
		responseBucketName  string
		responseFilepath    string
		responseBucketURI   string
		responseDownloadURI string

		expectFilters map[string]string
		expectTargets []string
		expectRes     *bucketfile.BucketFile
	}

	tests := []test{
		{
			name: "normal",

			recordingID: uuid.FromStringOrNil("5d946b94-9969-11eb-8bb3-07ff2b1cff3d"),

			responseFiles: []*file.File{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("1dc8ab1e-05f4-11f0-bfba-0784b300b355"),
					},
					ReferenceType: file.ReferenceTypeRecording,
					Filepath:      "recording/call_e2951d7c-ac2d-11ea-8d4b-aff0e70476d6_2020-05-03T21:35:02.809Z_in.wav",
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("1dfb051e-05f4-11f0-bee7-9f67fb14fdf4"),
					},
					ReferenceType: file.ReferenceTypeRecording,
					Filepath:      "recording/call_e2951d7c-ac2d-11ea-8d4b-aff0e70476d6_2020-05-03T21:35:02.809Z_out.wav",
				},
			},

			filepath:            "recording/call_e2951d7c-ac2d-11ea-8d4b-aff0e70476d6_2020-05-03T21:35:02.809Z.wav",
			responseBucketName:  "bucketTmp",
			responseFilepath:    "tmp/bdd24974-8ce0-11ed-aca5-1b4a5f897d9f",
			responseBucketURI:   "gs://voipbin-production/tmp/bdd24974-8ce0-11ed-aca5-1b4a5f897d9f",
			responseDownloadURI: "https://download.uri/recording/call_e2951d7c-ac2d-11ea-8d4b-aff0e70476d6_2020-05-03T21:35:02.809Z.wav",

			expectFilters: map[string]string{
				"deleted":      "false",
				"reference_id": "5d946b94-9969-11eb-8bb3-07ff2b1cff3d",
			},
			expectTargets: []string{
				"recording/call_e2951d7c-ac2d-11ea-8d4b-aff0e70476d6_2020-05-03T21:35:02.809Z_in.wav",
				"recording/call_e2951d7c-ac2d-11ea-8d4b-aff0e70476d6_2020-05-03T21:35:02.809Z_out.wav",
			},
			expectRes: &bucketfile.BucketFile{
				ReferenceType:    bucketfile.ReferenceTypeRecording,
				ReferenceID:      uuid.FromStringOrNil("5d946b94-9969-11eb-8bb3-07ff2b1cff3d"),
				BucketURI:        "gs://voipbin-production/tmp/bdd24974-8ce0-11ed-aca5-1b4a5f897d9f",
				DownloadURI:      "https://download.uri/recording/call_e2951d7c-ac2d-11ea-8d4b-aff0e70476d6_2020-05-03T21:35:02.809Z.wav",
				TMDownloadExpire: "",
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

			mockFile.EXPECT().Gets(ctx, "", uint64(100), tt.expectFilters).Return(tt.responseFiles, nil)

			mockFile.EXPECT().CompressCreate(ctx, tt.responseFiles).Return(tt.responseBucketName, tt.responseFilepath, nil)
			mockFile.EXPECT().DownloadURIGet(ctx, tt.responseBucketName, tt.responseFilepath, time.Hour*24).Return(tt.responseBucketURI, tt.responseDownloadURI, nil)
			mockUtil.EXPECT().TimeGetCurTimeAdd(gomock.Any()).Return("")

			res, err := h.RecordingGet(ctx, tt.recordingID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			tt.expectRes.TMDownloadExpire = res.TMDownloadExpire
			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectRes, *res)
			}
		})
	}
}

func Test_RecordingDelete(t *testing.T) {

	type test struct {
		name string

		recordingID uuid.UUID

		responseFiles []*file.File

		expectFilters map[string]string
	}

	tests := []test{
		{
			name: "normal",

			recordingID: uuid.FromStringOrNil("a18fbd98-8eaa-11ed-8d35-6b10d649e16f"),

			responseFiles: []*file.File{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("1e1a782c-05f4-11f0-ac5e-cb8994e2e2dc"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("1e3a9b34-05f4-11f0-b628-37e63e1558d0"),
					},
				},
			},

			expectFilters: map[string]string{
				"deleted":      "false",
				"reference_id": "a18fbd98-8eaa-11ed-8d35-6b10d649e16f",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockFile := filehandler.NewMockFileHandler(mc)

			h := storageHandler{
				reqHandler:  mockReq,
				fileHandler: mockFile,

				bucketNameMedia: "media",
			}
			ctx := context.Background()

			mockFile.EXPECT().Gets(ctx, "", uint64(100), tt.expectFilters).Return(tt.responseFiles, nil)
			for _, f := range tt.responseFiles {
				mockFile.EXPECT().Delete(ctx, f.ID).Return(&file.File{}, nil)
			}

			if errDel := h.RecordingDelete(ctx, tt.recordingID); errDel != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errDel)
			}

		})
	}
}
