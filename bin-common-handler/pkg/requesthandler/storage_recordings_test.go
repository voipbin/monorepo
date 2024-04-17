package requesthandler

import (
	"context"
	reflect "reflect"
	"testing"

	smbucketfile "monorepo/bin-storage-manager/models/bucketfile"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
)

func Test_StorageV1RecordingGet(t *testing.T) {

	tests := []struct {
		name string

		id             uuid.UUID
		requestTimeout int

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response

		expectResult *smbucketfile.BucketFile
	}{
		{
			"normal",

			uuid.FromStringOrNil("c7878bdc-93bd-11eb-ab3a-a7388c5862f4"),
			30000,

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
			&smbucketfile.BucketFile{
				DownloadURI: "https://example.com/c7878bdc-93bd-11eb-ab3a-a7388c5862f4",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.StorageV1RecordingGet(ctx, tt.id, tt.requestTimeout)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectResult, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectResult, res)
			}
		})
	}
}

func Test_StorageV1RecordingDelete(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response
	}{
		{
			"normal",

			uuid.FromStringOrNil("c0574ef6-8eb1-11ed-9ba3-2f05d999d9b3"),

			"bin-manager.storage-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/recordings/c0574ef6-8eb1-11ed-9ba3-2f05d999d9b3",
				Method:   rabbitmqhandler.RequestMethodDelete,
				DataType: ContentTypeJSON,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			if err := reqHandler.StorageV1RecordingDelete(ctx, tt.id); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
