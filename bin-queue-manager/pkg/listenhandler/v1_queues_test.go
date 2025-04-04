package listenhandler

import (
	reflect "reflect"
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	commonidentity "monorepo/bin-common-handler/models/identity"
	fmaction "monorepo/bin-flow-manager/models/action"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-queue-manager/models/queue"
	"monorepo/bin-queue-manager/models/queuecall"
	"monorepo/bin-queue-manager/pkg/dbhandler"
	"monorepo/bin-queue-manager/pkg/queuecallhandler"
	"monorepo/bin-queue-manager/pkg/queuehandler"
)

func Test_processV1QueuesPost(t *testing.T) {

	tests := []struct {
		name string

		request *sock.Request

		customerID     uuid.UUID
		queueName      string
		detail         string
		routingMethod  queue.RoutingMethod
		tagIDs         []uuid.UUID
		waitActions    []fmaction.Action
		waitTimeout    int
		serviceTimeout int

		queue *queue.Queue

		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/queues",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"442f5d62-7f55-11ec-a2c0-0bcd3814d515","name":"name","detail":"detail","routing_method":"random","tag_ids":["a4d0c36c-5f35-11ec-bf02-3b945ceab651"],"wait_actions":[{"type":"answer"}],"wait_timeout":600000,"service_timeout":6000000}`),
			},

			uuid.FromStringOrNil("442f5d62-7f55-11ec-a2c0-0bcd3814d515"),
			"name",
			"detail",
			queue.RoutingMethodRandom,
			[]uuid.UUID{
				uuid.FromStringOrNil("a4d0c36c-5f35-11ec-bf02-3b945ceab651"),
			},
			[]fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
			},
			600000,
			6000000,

			&queue.Queue{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("cba57fb6-59de-11ec-b230-5b6ab3380040"),
					CustomerID: uuid.FromStringOrNil("442f5d62-7f55-11ec-a2c0-0bcd3814d515"),
				},

				Name:   "name",
				Detail: "detail",

				RoutingMethod: queue.RoutingMethodRandom,
				TagIDs: []uuid.UUID{
					uuid.FromStringOrNil("a4d0c36c-5f35-11ec-bf02-3b945ceab651"),
				},

				Execute: queue.ExecuteStop,

				WaitActions: []fmaction.Action{
					{
						ID:   uuid.FromStringOrNil("8299402a-5f36-11ec-bd2a-b75b037f00f2"),
						Type: fmaction.TypeAnswer,
					},
				},
				WaitQueuecallIDs:    []uuid.UUID{},
				WaitTimeout:         60000,
				ServiceQueuecallIDs: []uuid.UUID{},
				ServiceTimeout:      600000,

				TotalIncomingCount:  0,
				TotalServicedCount:  0,
				TotalAbandonedCount: 0,
				TMCreate:            "2021-04-18 03:22:17.994000",
				TMUpdate:            dbhandler.DefaultTimeStamp,
				TMDelete:            dbhandler.DefaultTimeStamp,
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"cba57fb6-59de-11ec-b230-5b6ab3380040","customer_id":"442f5d62-7f55-11ec-a2c0-0bcd3814d515","name":"name","detail":"detail","routing_method":"random","tag_ids":["a4d0c36c-5f35-11ec-bf02-3b945ceab651"],"execute":"stop","wait_actions":[{"id":"8299402a-5f36-11ec-bd2a-b75b037f00f2","next_id":"00000000-0000-0000-0000-000000000000","type":"answer"}],"wait_timeout":60000,"service_timeout":600000,"wait_queuecall_ids":[],"service_queuecall_ids":[],"total_incoming_count":0,"total_serviced_count":0,"total_abandoned_count":0,"tm_create":"2021-04-18 03:22:17.994000","tm_update":"9999-01-01 00:00:00.000000","tm_delete":"9999-01-01 00:00:00.000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockQueue := queuehandler.NewMockQueueHandler(mc)

			h := &listenHandler{
				sockHandler:  mockSock,
				queueHandler: mockQueue,
			}

			mockQueue.EXPECT().Create(
				gomock.Any(),
				tt.customerID,
				tt.queueName,
				tt.detail,
				tt.routingMethod,
				tt.tagIDs,
				tt.waitActions,
				tt.waitTimeout,
				tt.serviceTimeout,
			).Return(tt.queue, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1QueuesGet(t *testing.T) {

	tests := []struct {
		name string

		request *sock.Request

		pageSize        uint64
		pageToken       string
		responseFilters map[string]string

		queues []*queue.Queue

		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:    "/v1/queues?page_size=10&page_token=2020-05-03%2021:35:02.809&customer_id=570b5094-7f55-11ec-b5cd-1b925f9028af&filter_deleted=false",
				Method: sock.RequestMethodGet,
			},

			10,
			"2020-05-03 21:35:02.809",
			map[string]string{
				"customer_id": "570b5094-7f55-11ec-b5cd-1b925f9028af",
				"deleted":     "false",
			},

			[]*queue.Queue{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("866ad964-620e-11eb-9f09-9fab48a7edd3"),
						CustomerID: uuid.FromStringOrNil("570b5094-7f55-11ec-b5cd-1b925f9028af"),
					},
				},
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"866ad964-620e-11eb-9f09-9fab48a7edd3","customer_id":"570b5094-7f55-11ec-b5cd-1b925f9028af","name":"","detail":"","routing_method":"","tag_ids":null,"execute":"","wait_actions":null,"wait_timeout":0,"service_timeout":0,"wait_queuecall_ids":null,"service_queuecall_ids":null,"total_incoming_count":0,"total_serviced_count":0,"total_abandoned_count":0,"tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
		{
			"2 items",
			&sock.Request{
				URI:    "/v1/queues?page_size=10&page_token=2020-05-03%2021:35:02.809&customer_id=6a7ce2b4-7f55-11ec-a666-8b44aa06d0db&filter_deleted=false",
				Method: sock.RequestMethodGet,
			},

			10,
			"2020-05-03 21:35:02.809",
			map[string]string{
				"customer_id": "6a7ce2b4-7f55-11ec-a666-8b44aa06d0db",
				"deleted":     "false",
			},

			[]*queue.Queue{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("866ad964-620e-11eb-9f09-9fab48a7edd3"),
						CustomerID: uuid.FromStringOrNil("6a7ce2b4-7f55-11ec-a666-8b44aa06d0db"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("e218b154-5f6b-11ec-818d-633351f9e341"),
						CustomerID: uuid.FromStringOrNil("6a7ce2b4-7f55-11ec-a666-8b44aa06d0db"),
					},
				},
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"866ad964-620e-11eb-9f09-9fab48a7edd3","customer_id":"6a7ce2b4-7f55-11ec-a666-8b44aa06d0db","name":"","detail":"","routing_method":"","tag_ids":null,"execute":"","wait_actions":null,"wait_timeout":0,"service_timeout":0,"wait_queuecall_ids":null,"service_queuecall_ids":null,"total_incoming_count":0,"total_serviced_count":0,"total_abandoned_count":0,"tm_create":"","tm_update":"","tm_delete":""},{"id":"e218b154-5f6b-11ec-818d-633351f9e341","customer_id":"6a7ce2b4-7f55-11ec-a666-8b44aa06d0db","name":"","detail":"","routing_method":"","tag_ids":null,"execute":"","wait_actions":null,"wait_timeout":0,"service_timeout":0,"wait_queuecall_ids":null,"service_queuecall_ids":null,"total_incoming_count":0,"total_serviced_count":0,"total_abandoned_count":0,"tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockSock := sockhandler.NewMockSockHandler(mc)
			mockQueue := queuehandler.NewMockQueueHandler(mc)

			h := &listenHandler{
				utilHanlder: mockUtil,
				sockHandler: mockSock,

				queueHandler: mockQueue,
			}

			mockUtil.EXPECT().URLParseFilters(gomock.Any()).Return(tt.responseFilters)
			mockQueue.EXPECT().Gets(gomock.Any(), tt.pageSize, tt.pageToken, tt.responseFilters).Return(tt.queues, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func Test_processV1QueuesIDGet(t *testing.T) {

	tests := []struct {
		name string

		request *sock.Request

		id uuid.UUID

		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/queues/a8e8faba-6150-11ec-bde0-e75ae9f16df7",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},

			uuid.FromStringOrNil("a8e8faba-6150-11ec-bde0-e75ae9f16df7"),

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"00000000-0000-0000-0000-000000000000","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","routing_method":"","tag_ids":null,"execute":"","wait_actions":null,"wait_timeout":0,"service_timeout":0,"wait_queuecall_ids":null,"service_queuecall_ids":null,"total_incoming_count":0,"total_serviced_count":0,"total_abandoned_count":0,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockQueue := queuehandler.NewMockQueueHandler(mc)

			h := &listenHandler{
				sockHandler:  mockSock,
				queueHandler: mockQueue,
			}

			mockQueue.EXPECT().Get(gomock.Any(), tt.id).Return(&queue.Queue{}, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1QueuesIDDelete(t *testing.T) {

	tests := []struct {
		name string

		request *sock.Request

		id uuid.UUID

		responseQueue *queue.Queue
		expectRes     *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/queues/a8e8faba-6150-11ec-bde0-e75ae9f16df7",
				Method:   sock.RequestMethodDelete,
				DataType: "application/json",
			},

			uuid.FromStringOrNil("a8e8faba-6150-11ec-bde0-e75ae9f16df7"),

			&queue.Queue{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a8e8faba-6150-11ec-bde0-e75ae9f16df7"),
				},
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"a8e8faba-6150-11ec-bde0-e75ae9f16df7","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","routing_method":"","tag_ids":null,"execute":"","wait_actions":null,"wait_timeout":0,"service_timeout":0,"wait_queuecall_ids":null,"service_queuecall_ids":null,"total_incoming_count":0,"total_serviced_count":0,"total_abandoned_count":0,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockQueue := queuehandler.NewMockQueueHandler(mc)

			h := &listenHandler{
				sockHandler:  mockSock,
				queueHandler: mockQueue,
			}

			mockQueue.EXPECT().Delete(gomock.Any(), tt.id).Return(tt.responseQueue, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1QueuesIDPut(t *testing.T) {

	tests := []struct {
		name string

		request *sock.Request

		id             uuid.UUID
		queueName      string
		detail         string
		routingMethod  queue.RoutingMethod
		tagIDs         []uuid.UUID
		waitActions    []fmaction.Action
		waitTimeout    int
		serviceTimeout int

		responseQueue *queue.Queue
		expectRes     *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/queues/66f7d436-5f6c-11ec-9298-677df04a59c2",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"name":"name","detail":"detail","routing_method":"random","tag_ids":["1988fb8c-4a7d-11ee-8019-77954c15f154","19f6fcf4-4a7d-11ee-8632-b7cc10cd1d20"],"wait_actions":[{"type":"answer"}],"wait_timeout":60000,"service_timeout":6000000}`),
			},

			uuid.FromStringOrNil("66f7d436-5f6c-11ec-9298-677df04a59c2"),
			"name",
			"detail",
			queue.RoutingMethodRandom,
			[]uuid.UUID{
				uuid.FromStringOrNil("1988fb8c-4a7d-11ee-8019-77954c15f154"),
				uuid.FromStringOrNil("19f6fcf4-4a7d-11ee-8632-b7cc10cd1d20"),
			},
			[]fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
			},
			60000,
			6000000,

			&queue.Queue{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("66f7d436-5f6c-11ec-9298-677df04a59c2"),
				},
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"66f7d436-5f6c-11ec-9298-677df04a59c2","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","routing_method":"","tag_ids":null,"execute":"","wait_actions":null,"wait_timeout":0,"service_timeout":0,"wait_queuecall_ids":null,"service_queuecall_ids":null,"total_incoming_count":0,"total_serviced_count":0,"total_abandoned_count":0,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockQueue := queuehandler.NewMockQueueHandler(mc)

			h := &listenHandler{
				sockHandler:  mockSock,
				queueHandler: mockQueue,
			}

			mockQueue.EXPECT().UpdateBasicInfo(
				gomock.Any(),
				tt.id,
				tt.queueName,
				tt.detail,
				tt.routingMethod,
				tt.tagIDs,
				tt.waitActions,
				tt.waitTimeout,
				tt.serviceTimeout,
			).Return(tt.responseQueue, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1QueuesIDTagIDsPut(t *testing.T) {

	tests := []struct {
		name string

		request *sock.Request

		id     uuid.UUID
		tagIDs []uuid.UUID

		responseQueue *queue.Queue
		expectRes     *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/queues/4c898be8-5f6d-11ec-b701-a7ba1509a629/tag_ids",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"tag_ids":["7bd9f08a-6018-11ec-a177-6742e33b235a"]}`),
			},

			uuid.FromStringOrNil("4c898be8-5f6d-11ec-b701-a7ba1509a629"),
			[]uuid.UUID{
				uuid.FromStringOrNil("7bd9f08a-6018-11ec-a177-6742e33b235a"),
			},

			&queue.Queue{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("4c898be8-5f6d-11ec-b701-a7ba1509a629"),
				},
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"4c898be8-5f6d-11ec-b701-a7ba1509a629","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","routing_method":"","tag_ids":null,"execute":"","wait_actions":null,"wait_timeout":0,"service_timeout":0,"wait_queuecall_ids":null,"service_queuecall_ids":null,"total_incoming_count":0,"total_serviced_count":0,"total_abandoned_count":0,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
		{
			"2 tag ids",
			&sock.Request{
				URI:      "/v1/queues/1c0938ae-6019-11ec-8a5d-ab6c7909948a/tag_ids",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"tag_ids":["14e7e750-6019-11ec-98ad-0753c69937ab", "153776e4-6019-11ec-b455-2b5b5ac589ec"]}`),
			},

			uuid.FromStringOrNil("1c0938ae-6019-11ec-8a5d-ab6c7909948a"),
			[]uuid.UUID{
				uuid.FromStringOrNil("14e7e750-6019-11ec-98ad-0753c69937ab"),
				uuid.FromStringOrNil("153776e4-6019-11ec-b455-2b5b5ac589ec"),
			},

			&queue.Queue{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("1c0938ae-6019-11ec-8a5d-ab6c7909948a"),
				},
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"1c0938ae-6019-11ec-8a5d-ab6c7909948a","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","routing_method":"","tag_ids":null,"execute":"","wait_actions":null,"wait_timeout":0,"service_timeout":0,"wait_queuecall_ids":null,"service_queuecall_ids":null,"total_incoming_count":0,"total_serviced_count":0,"total_abandoned_count":0,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockQueue := queuehandler.NewMockQueueHandler(mc)

			h := &listenHandler{
				sockHandler:  mockSock,
				queueHandler: mockQueue,
			}

			mockQueue.EXPECT().UpdateTagIDs(gomock.Any(), tt.id, tt.tagIDs).Return(tt.responseQueue, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1QueuesIDRoutingMethodPut(t *testing.T) {

	tests := []struct {
		name string

		request *sock.Request

		id            uuid.UUID
		routingMethod queue.RoutingMethod

		responseQueue *queue.Queue
		expectRes     *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/queues/89b402a8-6019-11ec-8f65-cb5c282f0024/routing_method",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"routing_method":"random"}`),
			},

			uuid.FromStringOrNil("89b402a8-6019-11ec-8f65-cb5c282f0024"),
			queue.RoutingMethodRandom,

			&queue.Queue{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("89b402a8-6019-11ec-8f65-cb5c282f0024"),
				},
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"89b402a8-6019-11ec-8f65-cb5c282f0024","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","routing_method":"","tag_ids":null,"execute":"","wait_actions":null,"wait_timeout":0,"service_timeout":0,"wait_queuecall_ids":null,"service_queuecall_ids":null,"total_incoming_count":0,"total_serviced_count":0,"total_abandoned_count":0,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockQueue := queuehandler.NewMockQueueHandler(mc)

			h := &listenHandler{
				sockHandler:  mockSock,
				queueHandler: mockQueue,
			}

			mockQueue.EXPECT().UpdateRoutingMethod(gomock.Any(), tt.id, tt.routingMethod).Return(tt.responseQueue, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1QueuesIDWaitActionsPut(t *testing.T) {

	tests := []struct {
		name string

		request *sock.Request

		id             uuid.UUID
		waitActions    []fmaction.Action
		waitTimeout    int
		serviceTimeout int

		responseQueue *queue.Queue
		expectRes     *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/queues/e4d05ee8-6019-11ec-ac25-1bd30b213fe2/wait_actions",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"wait_actions":[{"type":"answer"}],"wait_timeout":10000,"service_timeout":100000}`),
			},

			uuid.FromStringOrNil("e4d05ee8-6019-11ec-ac25-1bd30b213fe2"),
			[]fmaction.Action{

				{
					Type: fmaction.TypeAnswer,
				},
			},
			10000,
			100000,

			&queue.Queue{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("e4d05ee8-6019-11ec-ac25-1bd30b213fe2"),
				},
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"e4d05ee8-6019-11ec-ac25-1bd30b213fe2","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","routing_method":"","tag_ids":null,"execute":"","wait_actions":null,"wait_timeout":0,"service_timeout":0,"wait_queuecall_ids":null,"service_queuecall_ids":null,"total_incoming_count":0,"total_serviced_count":0,"total_abandoned_count":0,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockQueue := queuehandler.NewMockQueueHandler(mc)

			h := &listenHandler{
				sockHandler:  mockSock,
				queueHandler: mockQueue,
			}

			mockQueue.EXPECT().UpdateWaitActionsAndTimeouts(gomock.Any(), tt.id, tt.waitActions, tt.waitTimeout, tt.serviceTimeout).Return(tt.responseQueue, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1QueuesIDAgentsGet(t *testing.T) {
	tests := []struct {
		name string

		request *sock.Request

		id     uuid.UUID
		status amagent.Status

		responseQueue []amagent.Agent
		expectRes     *sock.Response
	}{
		{
			"available",
			&sock.Request{
				URI:      "/v1/queues/2e2ca500-b49e-11ec-bde5-4f7293129cfd/agents?status=available",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},

			uuid.FromStringOrNil("2e2ca500-b49e-11ec-bde5-4f7293129cfd"),
			amagent.StatusAvailable,

			[]amagent.Agent{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("2e5b56a2-b49e-11ec-a643-5b72b632781f"),
					},
				},
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"2e5b56a2-b49e-11ec-a643-5b72b632781f","customer_id":"00000000-0000-0000-0000-000000000000","username":"","password_hash":"","name":"","detail":"","ring_method":"","status":"","permission":0,"tag_ids":null,"addresses":null,"tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockQueue := queuehandler.NewMockQueueHandler(mc)

			h := &listenHandler{
				sockHandler:  mockSock,
				queueHandler: mockQueue,
			}

			mockQueue.EXPECT().GetAgents(gomock.Any(), tt.id, tt.status).Return(tt.responseQueue, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1QueuesIDExecuteRunPost(t *testing.T) {
	tests := []struct {
		name string

		request *sock.Request

		id uuid.UUID

		expectRes *sock.Response
	}{
		{
			"available",
			&sock.Request{
				URI:      "/v1/queues/c58d96e0-d1a7-11ec-a088-07232a972294/execute_run",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
			},

			uuid.FromStringOrNil("c58d96e0-d1a7-11ec-a088-07232a972294"),

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockQueue := queuehandler.NewMockQueueHandler(mc)

			h := &listenHandler{
				sockHandler:  mockSock,
				queueHandler: mockQueue,
			}

			mockQueue.EXPECT().Execute(gomock.Any(), tt.id)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1QueuesIDExecutePut(t *testing.T) {
	tests := []struct {
		name string

		request *sock.Request

		queueID uuid.UUID
		execute queue.Execute

		responseQueue *queue.Queue

		expectRes *sock.Response
	}{
		{
			"available",
			&sock.Request{
				URI:      "/v1/queues/e5e9af02-d1d7-11ec-b5e1-0782d8999acb/execute",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"execute":"run"}`),
			},

			uuid.FromStringOrNil("e5e9af02-d1d7-11ec-b5e1-0782d8999acb"),
			queue.ExecuteRun,

			&queue.Queue{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("e5e9af02-d1d7-11ec-b5e1-0782d8999acb"),
				},
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"e5e9af02-d1d7-11ec-b5e1-0782d8999acb","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","routing_method":"","tag_ids":null,"execute":"","wait_actions":null,"wait_timeout":0,"service_timeout":0,"wait_queuecall_ids":null,"service_queuecall_ids":null,"total_incoming_count":0,"total_serviced_count":0,"total_abandoned_count":0,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockQueue := queuehandler.NewMockQueueHandler(mc)

			h := &listenHandler{
				sockHandler:  mockSock,
				queueHandler: mockQueue,
			}

			mockQueue.EXPECT().UpdateExecute(gomock.Any(), tt.queueID, tt.execute).Return(tt.responseQueue, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1QueuecallsIDStatusWaitingPost(t *testing.T) {

	tests := []struct {
		name string

		request *sock.Request

		queuecallID uuid.UUID

		responseQueuecall *queuecall.Queuecall
		expectRes         *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/queuecalls/7c9e9cae-d1ca-11ec-a81e-0baaef8ce608/status_waiting",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
			},

			queuecallID: uuid.FromStringOrNil("7c9e9cae-d1ca-11ec-a81e-0baaef8ce608"),

			responseQueuecall: &queuecall.Queuecall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("7c9e9cae-d1ca-11ec-a81e-0baaef8ce608"),
				},
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"7c9e9cae-d1ca-11ec-a81e-0baaef8ce608","customer_id":"00000000-0000-0000-0000-000000000000","queue_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","reference_activeflow_id":"00000000-0000-0000-0000-000000000000","forward_action_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","source":{},"routing_method":"","tag_ids":null,"status":"","service_agent_id":"00000000-0000-0000-0000-000000000000","timeout_wait":0,"timeout_service":0,"duration_waiting":0,"duration_service":0,"tm_create":"","tm_service":"","tm_update":"","tm_end":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockQueuecall := queuecallhandler.NewMockQueuecallHandler(mc)

			h := &listenHandler{
				sockHandler:      mockSock,
				queuecallHandler: mockQueuecall,
			}

			mockQueuecall.EXPECT().UpdateStatusWaiting(gomock.Any(), tt.queuecallID).Return(tt.responseQueuecall, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
