package requesthandler

import (
	"context"
	"reflect"
	"testing"

	qmqueuecall "monorepo/bin-queue-manager/models/queuecall"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

func Test_QueueV1QueuecallGets(t *testing.T) {

	tests := []struct {
		name string

		pageToken string
		pageSize  uint64
		filters   map[string]string

		expectURL     string
		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response
		expectRes     []qmqueuecall.Queuecall
	}{
		{
			"normal",

			"2020-09-20T03:23:20.995000",
			10,
			map[string]string{
				"deleted": "false",
			},

			"/v1/queuecalls?page_token=2020-09-20T03%3A23%3A20.995000&page_size=10",
			"bin-manager.queue-manager.request",
			&sock.Request{
				URI:    "/v1/queuecalls?page_token=2020-09-20T03%3A23%3A20.995000&page_size=10&filter_deleted=false",
				Method: sock.RequestMethodGet,
			},
			&sock.Response{
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

			"2020-09-20T03:23:20.995000",
			10,
			map[string]string{
				"deleted": "false",
			},

			"/v1/queuecalls?page_token=2020-09-20T03%3A23%3A20.995000&page_size=10",
			"bin-manager.queue-manager.request",
			&sock.Request{
				URI:    "/v1/queuecalls?page_token=2020-09-20T03%3A23%3A20.995000&page_size=10&filter_deleted=false",
				Method: sock.RequestMethodGet,
			},
			&sock.Response{
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
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			h := requestHandler{
				sock:        mockSock,
				utilHandler: mockUtil,
			}
			ctx := context.Background()

			mockUtil.EXPECT().URLMergeFilters(tt.expectURL, tt.filters).Return(utilhandler.URLMergeFilters(tt.expectURL, tt.filters))
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := h.QueueV1QueuecallGets(ctx, tt.pageToken, tt.pageSize, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_QueueV1QueuecallGet(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response
		expectRes     *qmqueuecall.Queuecall
	}{
		{
			"normal",

			uuid.FromStringOrNil("a2764422-6159-11ec-8d87-975236f7d7b7"),

			"bin-manager.queue-manager.request",
			&sock.Request{
				URI:    "/v1/queuecalls/a2764422-6159-11ec-8d87-975236f7d7b7",
				Method: sock.RequestMethodGet,
			},
			&sock.Response{
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
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.QueueV1QueuecallGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_QueueV1QueuecallGetByReferenceID(t *testing.T) {

	tests := []struct {
		name string

		referenceID uuid.UUID

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response
		expectRes     *qmqueuecall.Queuecall
	}{
		{
			"normal",

			uuid.FromStringOrNil("f0d7b6e2-bcba-11ed-9715-db75795f979e"),

			"bin-manager.queue-manager.request",
			&sock.Request{
				URI:    "/v1/queuecalls/reference_id/f0d7b6e2-bcba-11ed-9715-db75795f979e",
				Method: sock.RequestMethodGet,
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"f102c5b2-bcba-11ed-8a78-07c8c15cd024"}`),
			},
			&qmqueuecall.Queuecall{
				ID: uuid.FromStringOrNil("f102c5b2-bcba-11ed-8a78-07c8c15cd024"),
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

			res, err := reqHandler.QueueV1QueuecallGetByReferenceID(ctx, tt.referenceID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_QMQueuecallDelete(t *testing.T) {

	tests := []struct {
		name string

		queuecallID uuid.UUID

		expectTarget  string
		expectRequest *sock.Request

		response  *sock.Response
		expectRes *qmqueuecall.Queuecall
	}{
		{
			"normal",

			uuid.FromStringOrNil("f4b44b28-4e79-11ec-be3c-73450ec23a51"),

			"bin-manager.queue-manager.request",
			&sock.Request{
				URI:    "/v1/queuecalls/f4b44b28-4e79-11ec-be3c-73450ec23a51",
				Method: sock.RequestMethodDelete,
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"f4b44b28-4e79-11ec-be3c-73450ec23a51"}`),
			},
			&qmqueuecall.Queuecall{
				ID: uuid.FromStringOrNil("f4b44b28-4e79-11ec-be3c-73450ec23a51"),
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

			res, err := reqHandler.QueueV1QueuecallDelete(ctx, tt.queuecallID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_QMQueuecallKick(t *testing.T) {

	tests := []struct {
		name string

		queuecallID uuid.UUID

		expectTarget  string
		expectRequest *sock.Request

		response  *sock.Response
		expectRes *qmqueuecall.Queuecall
	}{
		{
			"normal",

			uuid.FromStringOrNil("e96bfff6-bac8-11ed-a20f-9be3817d2737"),

			"bin-manager.queue-manager.request",
			&sock.Request{
				URI:    "/v1/queuecalls/e96bfff6-bac8-11ed-a20f-9be3817d2737/kick",
				Method: sock.RequestMethodPost,
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"e96bfff6-bac8-11ed-a20f-9be3817d2737"}`),
			},
			&qmqueuecall.Queuecall{
				ID: uuid.FromStringOrNil("e96bfff6-bac8-11ed-a20f-9be3817d2737"),
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

			res, err := reqHandler.QueueV1QueuecallKick(ctx, tt.queuecallID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_QMQueuecallKickByReferenceID(t *testing.T) {

	tests := []struct {
		name string

		queuecallID uuid.UUID

		expectTarget  string
		expectRequest *sock.Request

		response  *sock.Response
		expectRes *qmqueuecall.Queuecall
	}{
		{
			"normal",

			uuid.FromStringOrNil("e9d1f928-bac8-11ed-a65d-7fb580a1eb02"),

			"bin-manager.queue-manager.request",
			&sock.Request{
				URI:    "/v1/queuecalls/reference_id/e9d1f928-bac8-11ed-a65d-7fb580a1eb02/kick",
				Method: sock.RequestMethodPost,
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"e9d1f928-bac8-11ed-a65d-7fb580a1eb02"}`),
			},
			&qmqueuecall.Queuecall{
				ID: uuid.FromStringOrNil("e9d1f928-bac8-11ed-a65d-7fb580a1eb02"),
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

			res, err := reqHandler.QueueV1QueuecallKickByReferenceID(ctx, tt.queuecallID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_QueueV1QueuecallTimeoutWait(t *testing.T) {

	type test struct {
		name string

		queuecallID uuid.UUID
		delay       int

		expectTarget  string
		expectRequest *sock.Request
	}

	tests := []test{
		{
			"normal",

			uuid.FromStringOrNil("ff5c5fba-60b3-11ec-97c3-ff9e56e19a78"),
			1000,

			"bin-manager.queue-manager.request",
			&sock.Request{
				URI:    "/v1/queuecalls/ff5c5fba-60b3-11ec-97c3-ff9e56e19a78/timeout_wait",
				Method: sock.RequestMethodPost,
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
			mockSock.EXPECT().PublishExchangeDelayedRequest(gomock.Any(), tt.expectTarget, tt.expectRequest, tt.delay).Return(nil)

			if err := reqHandler.QueueV1QueuecallTimeoutWait(ctx, tt.queuecallID, tt.delay); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}

func Test_QueueV1QueuecallTimeoutService(t *testing.T) {

	type test struct {
		name string

		queuecallID uuid.UUID
		delay       int

		expectTarget  string
		expectRequest *sock.Request
	}

	tests := []test{
		{
			"normal",

			uuid.FromStringOrNil("ddf27cfa-60b4-11ec-b221-13486052ae97"),
			1000,

			"bin-manager.queue-manager.request",
			&sock.Request{
				URI:    "/v1/queuecalls/ddf27cfa-60b4-11ec-b221-13486052ae97/timeout_service",
				Method: sock.RequestMethodPost,
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
			mockSock.EXPECT().PublishExchangeDelayedRequest(gomock.Any(), tt.expectTarget, tt.expectRequest, tt.delay).Return(nil)

			if err := reqHandler.QueueV1QueuecallTimeoutService(ctx, tt.queuecallID, tt.delay); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}

func Test_QueueV1QueuecallUpdateStatusWaiting(t *testing.T) {
	tests := []struct {
		name string

		queuecallID uuid.UUID

		expectTarget  string
		expectRequest *sock.Request

		response  *sock.Response
		expectRes *qmqueuecall.Queuecall
	}{
		{
			"normal",

			uuid.FromStringOrNil("092c9606-d1c8-11ec-8a0e-3383eeba05b5"),

			"bin-manager.queue-manager.request",
			&sock.Request{
				URI:    "/v1/queuecalls/092c9606-d1c8-11ec-8a0e-3383eeba05b5/status_waiting",
				Method: sock.RequestMethodPost,
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"092c9606-d1c8-11ec-8a0e-3383eeba05b5"}`),
			},
			&qmqueuecall.Queuecall{
				ID: uuid.FromStringOrNil("092c9606-d1c8-11ec-8a0e-3383eeba05b5"),
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

			res, err := reqHandler.QueueV1QueuecallUpdateStatusWaiting(ctx, tt.queuecallID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_QueueV1QueuecallExecute(t *testing.T) {

	tests := []struct {
		name string

		queuecallID uuid.UUID
		agentID     uuid.UUID

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response
		expectRes     *qmqueuecall.Queuecall
	}{
		{
			"normal",

			uuid.FromStringOrNil("dc293afd-dd16-492e-a725-a690dd300658"),
			uuid.FromStringOrNil("5b3d4931-40d9-4e54-aaa7-df221c8624b5"),

			"bin-manager.queue-manager.request",
			&sock.Request{
				URI:      "/v1/queuecalls/dc293afd-dd16-492e-a725-a690dd300658/execute",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"agent_id":"5b3d4931-40d9-4e54-aaa7-df221c8624b5"}`),
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"dc293afd-dd16-492e-a725-a690dd300658"}`),
			},
			&qmqueuecall.Queuecall{
				ID: uuid.FromStringOrNil("dc293afd-dd16-492e-a725-a690dd300658"),
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

			res, err := reqHandler.QueueV1QueuecallExecute(ctx, tt.queuecallID, tt.agentID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_QueueV1QueuecallHealthCheck(t *testing.T) {

	tests := []struct {
		name string

		queuecallID uuid.UUID
		retryCount  int

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
	}{
		{
			"normal",

			uuid.FromStringOrNil("1a788e4e-d539-11ee-8f84-335e0b9857ba"),
			1,

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},

			"bin-manager.queue-manager.request",
			&sock.Request{
				URI:      "/v1/queuecalls/1a788e4e-d539-11ee-8f84-335e0b9857ba/health-check",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"retry_count":1}`),
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

			err := reqHandler.QueueV1QueuecallHealthCheck(ctx, tt.queuecallID, 0, tt.retryCount)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
