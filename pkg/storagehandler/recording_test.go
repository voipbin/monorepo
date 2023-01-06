package storagehandler

import (
	"context"
	reflect "reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	cmrecording "gitlab.com/voipbin/bin-manager/call-manager.git/models/recording"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/storage-manager.git/models/bucketfile"
	"gitlab.com/voipbin/bin-manager/storage-manager.git/pkg/buckethandler"
)

func Test_GetRecording(t *testing.T) {

	type test struct {
		name string

		recordingID uuid.UUID

		response          *rabbitmqhandler.Response
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

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"5d946b94-9969-11eb-8bb3-07ff2b1cff3d","user_id":0,"type":"call","reference_id":"e2951d7c-ac2d-11ea-8d4b-aff0e70476d6","status":"ended","format":"","filename":"call_e2951d7c-ac2d-11ea-8d4b-aff0e70476d6_2020-05-03T21:35:02.809Z.wav","webhook_uri":"","asterisk_id":"","channel_id":"","tm_start":"","tm_end":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
			&cmrecording.Recording{
				ID:            uuid.FromStringOrNil("5d946b94-9969-11eb-8bb3-07ff2b1cff3d"),
				CustomerID:    uuid.FromStringOrNil("e46238ef-c246-4024-9926-417246acdcba"),
				ReferenceType: cmrecording.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("e2951d7c-ac2d-11ea-8d4b-aff0e70476d6"),
				Status:        cmrecording.StatusEnd,
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
			mockBucket := buckethandler.NewMockBucketHandler(mc)

			h := storageHandler{
				reqHandler:    mockReq,
				bucketHandler: mockBucket,
			}

			ctx := context.Background()

			mockReq.EXPECT().CallV1RecordingGet(ctx, tt.recordingID).Return(tt.responseRecording, nil)
			mockBucket.EXPECT().GetDownloadURI(ctx, tt.expectTargets, time.Hour*24).Return(&tt.responseBucketpath, &tt.responseDownloadURI, nil)
			res, err := h.GetRecording(ctx, tt.recordingID)
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
