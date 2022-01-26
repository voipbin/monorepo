package requesthandler

import (
	"context"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	smbucketrecording "gitlab.com/voipbin/bin-manager/storage-manager.git/models/bucketrecording"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

func TestSMRecordingGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock: mockSock,
	}

	type test struct {
		name string

		id uuid.UUID

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response

		expectResult *smbucketrecording.BucketRecording
	}

	tests := []test{
		{
			"normal",
			uuid.FromStringOrNil("c7878bdc-93bd-11eb-ab3a-a7388c5862f4"),

			"bin-manager.storage-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/recordings/c7878bdc-93bd-11eb-ab3a-a7388c5862f4",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: ContentTypeJSON,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"download_uri":"https://example.com/c7878bdc-93bd-11eb-ab3a-a7388c5862f4"}`),
			},
			&smbucketrecording.BucketRecording{
				DownloadURI: "https://example.com/c7878bdc-93bd-11eb-ab3a-a7388c5862f4",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.SMV1RecordingGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectResult, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectResult, res)
			}
		})
	}
}
