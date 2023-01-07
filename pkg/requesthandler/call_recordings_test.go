package requesthandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	cmrecording "gitlab.com/voipbin/bin-manager/call-manager.git/models/recording"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

func Test_CallV1RecordingGets(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		pageToken  string
		pageSize   uint64

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response
		expectRes     []cmrecording.Recording
	}{
		{
			"normal",

			uuid.FromStringOrNil("c9869b68-8ebf-11ed-8133-9380f47d55fe"),
			"2020-09-20T03:23:20.995000",
			10,

			"bin-manager.call-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/recordings?page_token=2020-09-20T03%3A23%3A20.995000&page_size=10&customer_id=c9869b68-8ebf-11ed-8133-9380f47d55fe",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"c9c63840-8ebf-11ed-8f4c-534a60a32848"}]`),
			},
			[]cmrecording.Recording{
				{
					ID: uuid.FromStringOrNil("c9c63840-8ebf-11ed-8f4c-534a60a32848"),
				},
			},
		},
		{
			"2 items",

			uuid.FromStringOrNil("c9efb170-8ebf-11ed-9710-4bad3233a5d5"),
			"2020-09-20T03:23:20.995000",
			10,

			"bin-manager.call-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/recordings?page_token=2020-09-20T03%3A23%3A20.995000&page_size=10&customer_id=c9efb170-8ebf-11ed-9710-4bad3233a5d5",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"ca1558b2-8ebf-11ed-9014-33c1de740f04"},{"id":"e445e45e-8ebf-11ed-89f3-8b24e2aee52e"}]`),
			},
			[]cmrecording.Recording{
				{
					ID: uuid.FromStringOrNil("ca1558b2-8ebf-11ed-9014-33c1de740f04"),
				},
				{
					ID: uuid.FromStringOrNil("e445e45e-8ebf-11ed-89f3-8b24e2aee52e"),
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

			res, err := reqHandler.CallV1RecordingGets(ctx, tt.customerID, tt.pageSize, tt.pageToken)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_CallV1RecordingGet(t *testing.T) {

	tests := []struct {
		name string

		callID uuid.UUID

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response
		expectRes     *cmrecording.Recording
	}{
		{
			"normal",

			uuid.FromStringOrNil("32154990-8ec0-11ed-98c2-7f6a7e0cc03e"),

			"bin-manager.call-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/recordings/32154990-8ec0-11ed-98c2-7f6a7e0cc03e",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"32154990-8ec0-11ed-98c2-7f6a7e0cc03e"}`),
			},
			&cmrecording.Recording{
				ID: uuid.FromStringOrNil("32154990-8ec0-11ed-98c2-7f6a7e0cc03e"),
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

			res, err := reqHandler.CallV1RecordingGet(ctx, tt.callID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_CallV1RecordingDelete(t *testing.T) {

	tests := []struct {
		name string

		callID uuid.UUID

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectResult  *cmrecording.Recording
	}{
		{
			"normal",

			uuid.FromStringOrNil("570ddfbe-8ec0-11ed-9dd8-1f8e11bf6de2"),

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"570ddfbe-8ec0-11ed-9dd8-1f8e11bf6de2"}`),
			},

			"bin-manager.call-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/recordings/570ddfbe-8ec0-11ed-9dd8-1f8e11bf6de2",
				Method:   rabbitmqhandler.RequestMethodDelete,
				DataType: "application/json",
			},
			&cmrecording.Recording{
				ID: uuid.FromStringOrNil("570ddfbe-8ec0-11ed-9dd8-1f8e11bf6de2"),
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

			res, err := reqHandler.CallV1RecordingDelete(ctx, tt.callID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectResult, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectResult, *res)
			}
		})
	}
}
