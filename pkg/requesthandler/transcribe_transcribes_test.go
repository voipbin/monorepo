package requesthandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	tmtranscribe "gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"
	tstranscribe "gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

func Test_TranscribeV1TranscribeGet(t *testing.T) {

	type test struct {
		name string

		transcribeID uuid.UUID

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response

		expectResult *tstranscribe.Transcribe
	}

	tests := []test{
		{
			"normal",

			uuid.FromStringOrNil("32b71878-8093-11ed-8578-775276ea57cf"),

			"bin-manager.transcribe-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/transcribes/32b71878-8093-11ed-8578-775276ea57cf",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: ContentTypeJSON,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"32b71878-8093-11ed-8578-775276ea57cf"}`),
			},
			&tstranscribe.Transcribe{
				ID: uuid.FromStringOrNil("32b71878-8093-11ed-8578-775276ea57cf"),
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

			res, err := reqHandler.TranscribeV1TranscribeGet(ctx, tt.transcribeID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectResult, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectResult, res)
			}
		})
	}
}

func Test_TranscribeV1TranscribeGets(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		pageToken  string
		pageSize   uint64

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response
		expectRes     []tmtranscribe.Transcribe
	}{
		{
			"1 item",

			uuid.FromStringOrNil("adddce70-8093-11ed-9a79-530f80f428d8"),
			"2020-09-20T03:23:20.995000",
			10,

			"bin-manager.transcribe-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/transcribes?page_token=2020-09-20T03%3A23%3A20.995000&page_size=10&customer_id=adddce70-8093-11ed-9a79-530f80f428d8",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"ae0a7cfe-8093-11ed-963d-abb334c8e6d8"}]`),
			},
			[]tmtranscribe.Transcribe{
				{
					ID: uuid.FromStringOrNil("ae0a7cfe-8093-11ed-963d-abb334c8e6d8"),
				},
			},
		},
		{
			"2 items",

			uuid.FromStringOrNil("bb3c9146-8093-11ed-a0df-6fbf1a76cbd3"),
			"2020-09-20T03:23:20.995000",
			10,

			"bin-manager.transcribe-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/transcribes?page_token=2020-09-20T03%3A23%3A20.995000&page_size=10&customer_id=bb3c9146-8093-11ed-a0df-6fbf1a76cbd3",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"bb6c13bc-8093-11ed-b647-5f3b613e1180"},{"id":"bb8fc46a-8093-11ed-9ea7-9304ab751b40"}]`),
			},
			[]tmtranscribe.Transcribe{
				{
					ID: uuid.FromStringOrNil("bb6c13bc-8093-11ed-b647-5f3b613e1180"),
				},
				{
					ID: uuid.FromStringOrNil("bb8fc46a-8093-11ed-9ea7-9304ab751b40"),
				},
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

			res, err := reqHandler.TranscribeV1TranscribeGets(ctx, tt.customerID, tt.pageToken, tt.pageSize)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
