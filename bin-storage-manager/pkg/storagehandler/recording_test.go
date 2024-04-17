package storagehandler

import (
	"context"
	"fmt"
	reflect "reflect"
	"testing"
	"time"

	cmrecording "monorepo/bin-call-manager/models/recording"

	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"monorepo/bin-storage-manager/models/bucketfile"
	"monorepo/bin-storage-manager/pkg/filehandler"
)

func Test_RecordingGet(t *testing.T) {

	type test struct {
		name string

		recordingID uuid.UUID

		responseRecording *cmrecording.Recording

		filepath            string
		responseBucketpath  string
		responseDownloadURI string

		expectTargets []string
		expectRes     *bucketfile.BucketFile
	}

	tests := []test{
		{
			"normal",

			uuid.FromStringOrNil("5d946b94-9969-11eb-8bb3-07ff2b1cff3d"),

			&cmrecording.Recording{
				ID:            uuid.FromStringOrNil("5d946b94-9969-11eb-8bb3-07ff2b1cff3d"),
				CustomerID:    uuid.FromStringOrNil("e46238ef-c246-4024-9926-417246acdcba"),
				ReferenceType: cmrecording.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("e2951d7c-ac2d-11ea-8d4b-aff0e70476d6"),
				Status:        cmrecording.StatusEnded,
				Filenames: []string{
					"call_e2951d7c-ac2d-11ea-8d4b-aff0e70476d6_2020-05-03T21:35:02.809Z_in.wav",
					"call_e2951d7c-ac2d-11ea-8d4b-aff0e70476d6_2020-05-03T21:35:02.809Z_out.wav",
				},
			},
			"recording/call_e2951d7c-ac2d-11ea-8d4b-aff0e70476d6_2020-05-03T21:35:02.809Z.wav",
			"gs://voipbin-production/tmp/bdd24974-8ce0-11ed-aca5-1b4a5f897d9f",
			"https://download.uri/recording/call_e2951d7c-ac2d-11ea-8d4b-aff0e70476d6_2020-05-03T21:35:02.809Z.wav",

			[]string{
				"recording/call_e2951d7c-ac2d-11ea-8d4b-aff0e70476d6_2020-05-03T21:35:02.809Z_in.wav",
				"recording/call_e2951d7c-ac2d-11ea-8d4b-aff0e70476d6_2020-05-03T21:35:02.809Z_out.wav",
			},
			&bucketfile.BucketFile{
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

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockBucket := filehandler.NewMockFileHandler(mc)

			h := storageHandler{
				reqHandler:  mockReq,
				fileHandler: mockBucket,

				bucketNameMedia: "media",
			}

			ctx := context.Background()

			mockReq.EXPECT().CallV1RecordingGet(ctx, tt.recordingID).Return(tt.responseRecording, nil)
			mockBucket.EXPECT().GetDownloadURI(ctx, h.bucketNameMedia, tt.expectTargets, time.Hour*24).Return(&tt.responseBucketpath, &tt.responseDownloadURI, nil)
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

		responseRecording *cmrecording.Recording
	}

	tests := []test{
		{
			"normal",

			uuid.FromStringOrNil("a18fbd98-8eaa-11ed-8d35-6b10d649e16f"),

			&cmrecording.Recording{
				ID:            uuid.FromStringOrNil("a18fbd98-8eaa-11ed-8d35-6b10d649e16f"),
				CustomerID:    uuid.FromStringOrNil("e46238ef-c246-4024-9926-417246acdcba"),
				ReferenceType: cmrecording.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("e2951d7c-ac2d-11ea-8d4b-aff0e70476d6"),
				Status:        cmrecording.StatusEnded,
				Filenames: []string{
					"call_e2951d7c-ac2d-11ea-8d4b-aff0e70476d6_2020-05-03T21:35:02.809Z_in.wav",
					"call_e2951d7c-ac2d-11ea-8d4b-aff0e70476d6_2020-05-03T21:35:02.809Z_out.wav",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockBucket := filehandler.NewMockFileHandler(mc)

			h := storageHandler{
				reqHandler:  mockReq,
				fileHandler: mockBucket,

				bucketNameMedia: "media",
			}

			ctx := context.Background()

			mockReq.EXPECT().CallV1RecordingGet(ctx, tt.recordingID).Return(tt.responseRecording, nil)

			for _, filename := range tt.responseRecording.Filenames {
				filepath := fmt.Sprintf("recording/%s", filename)
				mockBucket.EXPECT().IsExist(ctx, h.bucketNameMedia, filepath).Return(true)
				mockBucket.EXPECT().Delete(ctx, h.bucketNameMedia, filepath).Return(nil)
			}

			err := h.RecordingDelete(ctx, tt.recordingID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}
