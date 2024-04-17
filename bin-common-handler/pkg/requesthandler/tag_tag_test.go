package requesthandler

import (
	"context"
	"reflect"
	"testing"

	tmtag "monorepo/bin-tag-manager/models/tag"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
)

func Test_TagV1TagCreate(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		tagName    string
		detail     string

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response

		expectRes *tmtag.Tag
	}{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("55ecfc4e-2c74-11ee-98fb-0762519529f3"),
			tagName:    "test name",
			detail:     "test detail",

			expectTarget: "bin-manager.tag-manager.request",
			expectRequest: &rabbitmqhandler.Request{
				URI:      "/v1/tags",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"customer_id":"55ecfc4e-2c74-11ee-98fb-0762519529f3","name":"test name","detail":"test detail"}`),
			},
			response: &rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"5623e25e-2c74-11ee-87a6-bfa8ae34077f"}`),
			},
			expectRes: &tmtag.Tag{
				ID: uuid.FromStringOrNil("5623e25e-2c74-11ee-87a6-bfa8ae34077f"),
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

			res, err := reqHandler.TagV1TagCreate(ctx, tt.customerID, tt.tagName, tt.detail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectRes, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectRes, *res)
			}
		})
	}
}

func Test_TagV1TagUpdate(t *testing.T) {

	tests := []struct {
		name string

		id      uuid.UUID
		tagName string
		detail  string

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response

		expectRes *tmtag.Tag
	}{
		{
			name: "normal",

			id:      uuid.FromStringOrNil("8f8b1638-2c75-11ee-89e0-23baf30fef23"),
			tagName: "test name",
			detail:  "test detail",

			expectTarget: "bin-manager.tag-manager.request",
			expectRequest: &rabbitmqhandler.Request{
				URI:      "/v1/tags/8f8b1638-2c75-11ee-89e0-23baf30fef23",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"name":"test name","detail":"test detail"}`),
			},
			response: &rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"8f8b1638-2c75-11ee-89e0-23baf30fef23"}`),
			},
			expectRes: &tmtag.Tag{
				ID: uuid.FromStringOrNil("8f8b1638-2c75-11ee-89e0-23baf30fef23"),
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

			res, err := reqHandler.TagV1TagUpdate(ctx, tt.id, tt.tagName, tt.detail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectRes, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectRes, *res)
			}
		})
	}
}

func Test_TagV1TagDelete(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response

		expectRes *tmtag.Tag
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("8fb8f1fc-2c75-11ee-aa2c-cb07baf2171a"),

			expectTarget: "bin-manager.tag-manager.request",
			expectRequest: &rabbitmqhandler.Request{
				URI:      "/v1/tags/8fb8f1fc-2c75-11ee-aa2c-cb07baf2171a",
				Method:   rabbitmqhandler.RequestMethodDelete,
				DataType: ContentTypeNone,
			},
			response: &rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"8fb8f1fc-2c75-11ee-aa2c-cb07baf2171a"}`),
			},
			expectRes: &tmtag.Tag{
				ID: uuid.FromStringOrNil("8fb8f1fc-2c75-11ee-aa2c-cb07baf2171a"),
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

			res, err := reqHandler.TagV1TagDelete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectRes, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectRes, *res)
			}
		})
	}
}

func Test_TagV1TagGet(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response

		expectRes *tmtag.Tag
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("8fe6c136-2c75-11ee-a3a4-37400837e12e"),

			expectTarget: "bin-manager.tag-manager.request",
			expectRequest: &rabbitmqhandler.Request{
				URI:      "/v1/tags/8fe6c136-2c75-11ee-a3a4-37400837e12e",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: ContentTypeNone,
			},
			response: &rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"8fe6c136-2c75-11ee-a3a4-37400837e12e"}`),
			},
			expectRes: &tmtag.Tag{
				ID: uuid.FromStringOrNil("8fe6c136-2c75-11ee-a3a4-37400837e12e"),
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

			res, err := reqHandler.TagV1TagGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectRes, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectRes, *res)
			}
		})
	}
}

func Test_TagV1TagGets(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		token      string
		pageSize   uint64

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response

		expectRes []tmtag.Tag
	}{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("8fe6c136-2c75-11ee-a3a4-37400837e12e"),
			token:      "2020-09-20 03:23:20.995000",
			pageSize:   10,

			expectTarget: "bin-manager.tag-manager.request",
			expectRequest: &rabbitmqhandler.Request{
				URI:      "/v1/tags?page_token=2020-09-20+03%3A23%3A20.995000&page_size=10&customer_id=8fe6c136-2c75-11ee-a3a4-37400837e12e",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: ContentTypeNone,
			},
			response: &rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"8fe6c136-2c75-11ee-a3a4-37400837e12e"}]`),
			},
			expectRes: []tmtag.Tag{
				{
					ID: uuid.FromStringOrNil("8fe6c136-2c75-11ee-a3a4-37400837e12e"),
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

			res, err := reqHandler.TagV1TagGets(ctx, tt.customerID, tt.token, tt.pageSize)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes[0], res[0])
			}
		})
	}
}
