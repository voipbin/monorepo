package requesthandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	qmqueuecall "gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
)

func TestQMV1QueuecallGets(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock:       mockSock,
		queueQueue: queueQueue,
	}

	tests := []struct {
		name string

		userID    uint64
		pageToken string
		pageSize  uint64

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response
		expectRes     []qmqueuecall.Queuecall
	}{
		{
			"normal",

			1,
			"2020-09-20T03:23:20.995000",
			10,

			"bin-manager.queue-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/queuecalls?page_token=2020-09-20T03%3A23%3A20.995000&page_size=10&user_id=1",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"07e42460-6159-11ec-8191-3b89ed95cdb5"}]`),
			},
			[]qmqueuecall.Queuecall{
				{
					ID: uuid.FromStringOrNil("07e42460-6159-11ec-8191-3b89ed95cdb5"),
				},
			},
		},
		{
			"2 results",

			1,
			"2020-09-20T03:23:20.995000",
			10,

			"bin-manager.queue-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/queuecalls?page_token=2020-09-20T03%3A23%3A20.995000&page_size=10&user_id=1",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"08a7c974-6159-11ec-9b3d-0f52d15f98f7"},{"id":"08c9ef2c-6159-11ec-9540-8b38d1cb2283"}]`),
			},
			[]qmqueuecall.Queuecall{
				{
					ID: uuid.FromStringOrNil("08a7c974-6159-11ec-9b3d-0f52d15f98f7"),
				},
				{
					ID: uuid.FromStringOrNil("08c9ef2c-6159-11ec-9540-8b38d1cb2283"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.QMV1QueuecallGets(ctx, tt.userID, tt.pageToken, tt.pageSize)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func TestAMV1QueuecallGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock:       mockSock,
		queueQueue: queueQueue,
	}

	tests := []struct {
		name string

		id uuid.UUID

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response
		expectRes     *qmqueuecall.Queuecall
	}{
		{
			"normal",

			uuid.FromStringOrNil("a2764422-6159-11ec-8d87-975236f7d7b7"),

			"bin-manager.queue-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/queuecalls/a2764422-6159-11ec-8d87-975236f7d7b7",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"a2764422-6159-11ec-8d87-975236f7d7b7"}`),
			},
			&qmqueuecall.Queuecall{
				ID: uuid.FromStringOrNil("a2764422-6159-11ec-8d87-975236f7d7b7"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.QMV1QueuecallGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func TestQMQueuecallDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock:       mockSock,
		queueQueue: queueQueue,
	}

	tests := []struct {
		name string

		queuecallID uuid.UUID

		expectTarget  string
		expectRequest *rabbitmqhandler.Request

		response *rabbitmqhandler.Response
	}{
		{
			"normal",

			uuid.FromStringOrNil("f4b44b28-4e79-11ec-be3c-73450ec23a51"),

			"bin-manager.queue-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/queuecalls/f4b44b28-4e79-11ec-be3c-73450ec23a51",
				Method:   rabbitmqhandler.RequestMethodDelete,
				DataType: "application/json",
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			if err := reqHandler.QMV1QueuecallDelete(ctx, tt.queuecallID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestQMQueuecallDeleteByReferenceID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock:       mockSock,
		queueQueue: queueQueue,
	}

	tests := []struct {
		name string

		referenceID uuid.UUID

		expectTarget  string
		expectRequest *rabbitmqhandler.Request

		response *rabbitmqhandler.Response
	}{
		{
			"normal",

			uuid.FromStringOrNil("f4b44b28-4e79-11ec-be3c-73450ec23a51"),

			"bin-manager.queue-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/queuecallreferences/f4b44b28-4e79-11ec-be3c-73450ec23a51",
				Method:   rabbitmqhandler.RequestMethodDelete,
				DataType: "application/json",
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			if err := reqHandler.QMV1QueuecallDeleteByReferenceID(ctx, tt.referenceID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestQMV1QueuecallExecute(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock:          mockSock,
		exchangeDelay: "bin-manager.delay",
		queueQueue:    queueQueue,
	}

	type test struct {
		name string

		queuecallID uuid.UUID
		delay       int

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
	}

	tests := []test{
		{
			"normal",

			uuid.FromStringOrNil("1e0d5a8c-5dcf-11ec-bc65-377573e53b24"),
			1000,

			"bin-manager.queue-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/queuecalls/1e0d5a8c-5dcf-11ec-bc65-377573e53b24/execute",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockSock.EXPECT().PublishExchangeDelayedRequest(gomock.Any(), tt.expectTarget, tt.expectRequest, tt.delay).Return(nil)

			if err := reqHandler.QMV1QueuecallExecute(ctx, tt.queuecallID, tt.delay); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}

func TestQMV1QueuecallTimeoutWait(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock:          mockSock,
		exchangeDelay: "bin-manager.delay",
		queueQueue:    queueQueue,
	}

	type test struct {
		name string

		queuecallID uuid.UUID
		delay       int

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
	}

	tests := []test{
		{
			"normal",

			uuid.FromStringOrNil("ff5c5fba-60b3-11ec-97c3-ff9e56e19a78"),
			1000,

			"bin-manager.queue-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/queuecalls/ff5c5fba-60b3-11ec-97c3-ff9e56e19a78/timeout_wait",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockSock.EXPECT().PublishExchangeDelayedRequest(gomock.Any(), tt.expectTarget, tt.expectRequest, tt.delay).Return(nil)

			if err := reqHandler.QMV1QueuecallTiemoutWait(ctx, tt.queuecallID, tt.delay); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}

func TestQMV1QueuecallTimeoutService(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock:          mockSock,
		exchangeDelay: "bin-manager.delay",
		queueQueue:    queueQueue,
	}

	type test struct {
		name string

		queuecallID uuid.UUID
		delay       int

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
	}

	tests := []test{
		{
			"normal",

			uuid.FromStringOrNil("ddf27cfa-60b4-11ec-b221-13486052ae97"),
			1000,

			"bin-manager.queue-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/queuecalls/ddf27cfa-60b4-11ec-b221-13486052ae97/timeout_service",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockSock.EXPECT().PublishExchangeDelayedRequest(gomock.Any(), tt.expectTarget, tt.expectRequest, tt.delay).Return(nil)

			if err := reqHandler.QMV1QueuecallTiemoutService(ctx, tt.queuecallID, tt.delay); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}
