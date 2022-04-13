package storagehandler

import (
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	cmrecording "gitlab.com/voipbin/bin-manager/call-manager.git/models/recording"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/storage-manager.git/models/bucketrecording"
	"gitlab.com/voipbin/bin-manager/storage-manager.git/pkg/buckethandler"
	"gitlab.com/voipbin/bin-manager/storage-manager.git/pkg/requesthandler"
)

func TestCMRecordingGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockBucket := buckethandler.NewMockBucketHandler(mc)

	h := storageHandler{
		reqHandler:    mockReq,
		bucketHandler: mockBucket,
	}

	type test struct {
		name string

		recordingID uuid.UUID

		response  *rabbitmqhandler.Response
		recording *cmrecording.Recording

		filepath    string
		downloadURI string

		expectResult *bucketrecording.BucketRecording
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
				ID:          uuid.FromStringOrNil("5d946b94-9969-11eb-8bb3-07ff2b1cff3d"),
				CustomerID:  uuid.FromStringOrNil("e46238ef-c246-4024-9926-417246acdcba"),
				Type:        cmrecording.TypeCall,
				ReferenceID: uuid.FromStringOrNil("e2951d7c-ac2d-11ea-8d4b-aff0e70476d6"),
				Status:      cmrecording.StatusEnd,
				Filename:    "call_e2951d7c-ac2d-11ea-8d4b-aff0e70476d6_2020-05-03T21:35:02.809Z.wav",
			},
			"recording/call_e2951d7c-ac2d-11ea-8d4b-aff0e70476d6_2020-05-03T21:35:02.809Z.wav",
			"https://download.uri/recording/call_e2951d7c-ac2d-11ea-8d4b-aff0e70476d6_2020-05-03T21:35:02.809Z.wav",

			&bucketrecording.BucketRecording{
				RecordingID:      uuid.FromStringOrNil("5d946b94-9969-11eb-8bb3-07ff2b1cff3d"),
				BucketURI:        "gs://voipbin-production/recording/call_e2951d7c-ac2d-11ea-8d4b-aff0e70476d6_2020-05-03T21:35:02.809Z.wav",
				DownloadURI:      "https://download.uri/recording/call_e2951d7c-ac2d-11ea-8d4b-aff0e70476d6_2020-05-03T21:35:02.809Z.wav",
				TMDownloadExpire: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockReq.EXPECT().CMRecordingGet(tt.recordingID).Return(tt.recording, nil)
			mockBucket.EXPECT().FileGetDownloadURL(tt.filepath, gomock.Any()).Return(tt.downloadURI, nil)
			mockBucket.EXPECT().GetBucketName().Return("voipbin-production")
			res, err := h.GetRecording(tt.recordingID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res.TMDownloadExpire = ""
			if reflect.DeepEqual(*tt.expectResult, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectResult, *res)
			}
		})
	}
}
