package listenhandler

import (
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/request-manager.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queue"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/queuehandler"
)

func TestProcessV1QueuesPost(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	mockQueue := queuehandler.NewMockQueueHandler(mc)

	h := &listenHandler{
		rabbitSock:   mockSock,
		db:           mockDB,
		reqHandler:   mockReq,
		queueHandler: mockQueue,
	}

	tests := []struct {
		name string

		request *rabbitmqhandler.Request

		userID         uint64
		queueName      string
		detail         string
		webhookURI     string
		webhookMethod  string
		routingMethod  queue.RoutingMethod
		tagIDs         []uuid.UUID
		waitActions    []fmaction.Action
		waitTimeout    int
		serviceTimeout int

		queue *queue.Queue

		expectRes *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/queues",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"user_id":1,"name":"name","detail":"detail","webhook_uri":"test.com","webhook_method":"POST","routing_method":"random","tag_ids":["a4d0c36c-5f35-11ec-bf02-3b945ceab651"],"wait_actions":[{"type":"answer"}],"wait_timeout":600000,"service_timeout":6000000}`),
			},

			1,
			"name",
			"detail",
			"test.com",
			"POST",
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
				ID:              uuid.FromStringOrNil("cba57fb6-59de-11ec-b230-5b6ab3380040"),
				UserID:          1,
				FlowID:          uuid.FromStringOrNil("538791ae-5c81-11ec-9cd9-4f0755b8aca6"),
				ConfbridgeID:    uuid.FromStringOrNil("9ec20c00-5f36-11ec-8ab6-1339e6ad8fb7"),
				ForwardActionID: uuid.FromStringOrNil("9ee0848c-5f36-11ec-b4f4-6fa8ca6f5406"),

				Name:          "name",
				Detail:        "detail",
				WebhookURI:    "test.com",
				WebhookMethod: "POST",

				RoutingMethod: queue.RoutingMethodRandom,
				TagIDs: []uuid.UUID{
					uuid.FromStringOrNil("a4d0c36c-5f35-11ec-bf02-3b945ceab651"),
				},

				WaitActions: []fmaction.Action{
					{
						ID:   uuid.FromStringOrNil("8299402a-5f36-11ec-bd2a-b75b037f00f2"),
						Type: fmaction.TypeAnswer,
					},
				},
				WaitQueueCallIDs:    []uuid.UUID{},
				WaitTimeout:         60000,
				ServiceQueueCallIDs: []uuid.UUID{},
				ServiceTimeout:      600000,

				TotalIncomingCount:  0,
				TotalServicedCount:  0,
				TotalAbandonedCount: 0,
				TotalWaitDuration:   0,
				TMCreate:            "2021-04-18 03:22:17.994000",
				TMUpdate:            dbhandler.DefaultTimeStamp,
				TMDelete:            dbhandler.DefaultTimeStamp,
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"cba57fb6-59de-11ec-b230-5b6ab3380040","user_id":1,"flow_id":"538791ae-5c81-11ec-9cd9-4f0755b8aca6","confbridge_id":"9ec20c00-5f36-11ec-8ab6-1339e6ad8fb7","forward_action_id":"9ee0848c-5f36-11ec-b4f4-6fa8ca6f5406","name":"name","detail":"detail","webhook_uri":"test.com","webhook_method":"POST","routing_method":"random","tag_ids":["a4d0c36c-5f35-11ec-bf02-3b945ceab651"],"wait_actions":[{"id":"8299402a-5f36-11ec-bd2a-b75b037f00f2","type":"answer"}],"wait_timeout":60000,"service_timeout":600000,"wait_queue_call_ids":[],"service_queue_call_ids":[],"total_incoming_count":0,"total_serviced_count":0,"total_abandoned_count":0,"total_waittime":0,"total_service_duration":0,"tm_create":"2021-04-18 03:22:17.994000","tm_update":"9999-01-01 00:00:00.000000","tm_delete":"9999-01-01 00:00:00.000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockQueue.EXPECT().Create(
				gomock.Any(),
				tt.userID,
				tt.queueName,
				tt.detail,
				tt.webhookURI,
				tt.webhookMethod,
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

func TestProcessV1QueuesGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockQueue := queuehandler.NewMockQueueHandler(mc)

	h := &listenHandler{
		rabbitSock: mockSock,
		db:         mockDB,
		reqHandler: mockReq,

		queueHandler: mockQueue,
	}

	tests := []struct {
		name string

		request *rabbitmqhandler.Request

		userID    uint64
		pageSize  uint64
		pageToken string

		queues []*queue.Queue

		expectRes *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:    "/v1/queues?page_size=10&page_token=2020-05-03%2021:35:02.809&user_id=1",
				Method: rabbitmqhandler.RequestMethodGet,
			},
			1,
			10,
			"2020-05-03 21:35:02.809",
			[]*queue.Queue{
				{
					ID:     uuid.FromStringOrNil("866ad964-620e-11eb-9f09-9fab48a7edd3"),
					UserID: 1,
				},
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"866ad964-620e-11eb-9f09-9fab48a7edd3","user_id":1,"flow_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","forward_action_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","webhook_uri":"","webhook_method":"","routing_method":"","tag_ids":null,"wait_actions":null,"wait_timeout":0,"service_timeout":0,"wait_queue_call_ids":null,"service_queue_call_ids":null,"total_incoming_count":0,"total_serviced_count":0,"total_abandoned_count":0,"total_waittime":0,"total_service_duration":0,"tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
		{
			"2 items",
			&rabbitmqhandler.Request{
				URI:    "/v1/queues?page_size=10&page_token=2020-05-03%2021:35:02.809&user_id=1",
				Method: rabbitmqhandler.RequestMethodGet,
			},
			1,
			10,
			"2020-05-03 21:35:02.809",
			[]*queue.Queue{
				{
					ID:     uuid.FromStringOrNil("866ad964-620e-11eb-9f09-9fab48a7edd3"),
					UserID: 1,
				},
				{
					ID:     uuid.FromStringOrNil("e218b154-5f6b-11ec-818d-633351f9e341"),
					UserID: 1,
				},
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"866ad964-620e-11eb-9f09-9fab48a7edd3","user_id":1,"flow_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","forward_action_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","webhook_uri":"","webhook_method":"","routing_method":"","tag_ids":null,"wait_actions":null,"wait_timeout":0,"service_timeout":0,"wait_queue_call_ids":null,"service_queue_call_ids":null,"total_incoming_count":0,"total_serviced_count":0,"total_abandoned_count":0,"total_waittime":0,"total_service_duration":0,"tm_create":"","tm_update":"","tm_delete":""},{"id":"e218b154-5f6b-11ec-818d-633351f9e341","user_id":1,"flow_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","forward_action_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","webhook_uri":"","webhook_method":"","routing_method":"","tag_ids":null,"wait_actions":null,"wait_timeout":0,"service_timeout":0,"wait_queue_call_ids":null,"service_queue_call_ids":null,"total_incoming_count":0,"total_serviced_count":0,"total_abandoned_count":0,"total_waittime":0,"total_service_duration":0,"tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockQueue.EXPECT().Gets(gomock.Any(), tt.userID, tt.pageSize, tt.pageToken).Return(tt.queues, nil)
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

func TestProcessV1QueuesIDPut(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	mockQueue := queuehandler.NewMockQueueHandler(mc)

	h := &listenHandler{
		rabbitSock:   mockSock,
		db:           mockDB,
		reqHandler:   mockReq,
		queueHandler: mockQueue,
	}

	tests := []struct {
		name string

		request *rabbitmqhandler.Request

		id            uuid.UUID
		queueName     string
		detail        string
		webhookURI    string
		webhookMethod string

		expectRes *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/queues/66f7d436-5f6c-11ec-9298-677df04a59c2",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"name":"name","detail":"detail","webhook_uri":"test.com","webhook_method":"POST"}`),
			},

			uuid.FromStringOrNil("66f7d436-5f6c-11ec-9298-677df04a59c2"),
			"name",
			"detail",
			"test.com",
			"POST",

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockQueue.EXPECT().UpdateBasicInfo(
				gomock.Any(),
				tt.id,
				tt.queueName,
				tt.detail,
				tt.webhookURI,
				tt.webhookMethod,
			).Return(nil)

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

func TestProcessV1QueuesIDQueuecallsPost(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	mockQueue := queuehandler.NewMockQueueHandler(mc)

	h := &listenHandler{
		rabbitSock:   mockSock,
		db:           mockDB,
		reqHandler:   mockReq,
		queueHandler: mockQueue,
	}

	tests := []struct {
		name string

		request *rabbitmqhandler.Request

		id            uuid.UUID
		referenceType queuecall.ReferenceType
		referenceID   uuid.UUID
		exitActionID  uuid.UUID

		queuecall *queuecall.Queuecall

		expectRes *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/queues/4c898be8-5f6d-11ec-b701-a7ba1509a629/queuecalls",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"reference_type":"call","reference_id":"4cb489b0-5f6d-11ec-b2cd-cb4148c6e166","exit_action_id":"4cd9ff4c-5f6d-11ec-b627-7331d2464ba7"}`),
			},

			uuid.FromStringOrNil("4c898be8-5f6d-11ec-b701-a7ba1509a629"),
			queuecall.ReferenceTypeCall,
			uuid.FromStringOrNil("4cb489b0-5f6d-11ec-b2cd-cb4148c6e166"),
			uuid.FromStringOrNil("4cd9ff4c-5f6d-11ec-b627-7331d2464ba7"),

			&queuecall.Queuecall{
				ID: uuid.FromStringOrNil("a7261c56-5f6d-11ec-8d91-ff8f64486712"),
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"a7261c56-5f6d-11ec-8d91-ff8f64486712","user_id":0,"queue_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","forward_action_id":"00000000-0000-0000-0000-000000000000","exit_action_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","webhook_uri":"","webhook_method":"","source":{"type":"","target":"","target_name":"","name":"","detail":""},"routing_method":"","tag_ids":null,"status":"","service_agent_id":"00000000-0000-0000-0000-000000000000","timeout_wait":0,"timeout_service":0,"tm_create":"","tm_service":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockQueue.EXPECT().Join(gomock.Any(), tt.id, tt.referenceType, tt.referenceID, tt.exitActionID).Return(tt.queuecall, nil)

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

func TestProcessV1QueuesIDTagIDsPut(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	mockQueue := queuehandler.NewMockQueueHandler(mc)

	h := &listenHandler{
		rabbitSock:   mockSock,
		db:           mockDB,
		reqHandler:   mockReq,
		queueHandler: mockQueue,
	}

	tests := []struct {
		name string

		request *rabbitmqhandler.Request

		id     uuid.UUID
		tagIDs []uuid.UUID

		expectRes *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/queues/4c898be8-5f6d-11ec-b701-a7ba1509a629/tag_ids",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"tag_ids":["7bd9f08a-6018-11ec-a177-6742e33b235a"]}`),
			},

			uuid.FromStringOrNil("4c898be8-5f6d-11ec-b701-a7ba1509a629"),
			[]uuid.UUID{
				uuid.FromStringOrNil("7bd9f08a-6018-11ec-a177-6742e33b235a"),
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},
		},
		{
			"2 tag ids",
			&rabbitmqhandler.Request{
				URI:      "/v1/queues/1c0938ae-6019-11ec-8a5d-ab6c7909948a/tag_ids",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"tag_ids":["14e7e750-6019-11ec-98ad-0753c69937ab", "153776e4-6019-11ec-b455-2b5b5ac589ec"]}`),
			},

			uuid.FromStringOrNil("1c0938ae-6019-11ec-8a5d-ab6c7909948a"),
			[]uuid.UUID{
				uuid.FromStringOrNil("14e7e750-6019-11ec-98ad-0753c69937ab"),
				uuid.FromStringOrNil("153776e4-6019-11ec-b455-2b5b5ac589ec"),
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockQueue.EXPECT().UpdateTagIDs(gomock.Any(), tt.id, tt.tagIDs).Return(nil)

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

func TestProcessV1QueuesIDRoutingMethodPut(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	mockQueue := queuehandler.NewMockQueueHandler(mc)

	h := &listenHandler{
		rabbitSock:   mockSock,
		db:           mockDB,
		reqHandler:   mockReq,
		queueHandler: mockQueue,
	}

	tests := []struct {
		name string

		request *rabbitmqhandler.Request

		id            uuid.UUID
		routingMethod queue.RoutingMethod

		expectRes *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/queues/89b402a8-6019-11ec-8f65-cb5c282f0024/routing_method",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"routing_method":"random"}`),
			},

			uuid.FromStringOrNil("89b402a8-6019-11ec-8f65-cb5c282f0024"),
			queue.RoutingMethodRandom,

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockQueue.EXPECT().UpdateRoutingMethod(gomock.Any(), tt.id, tt.routingMethod).Return(nil)

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

func TestProcessV1QueuesIDWaitActionsPut(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	mockQueue := queuehandler.NewMockQueueHandler(mc)

	h := &listenHandler{
		rabbitSock:   mockSock,
		db:           mockDB,
		reqHandler:   mockReq,
		queueHandler: mockQueue,
	}

	tests := []struct {
		name string

		request *rabbitmqhandler.Request

		id             uuid.UUID
		waitActions    []fmaction.Action
		waitTimeout    int
		serviceTimeout int

		expectRes *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/queues/e4d05ee8-6019-11ec-ac25-1bd30b213fe2/wait_actions",
				Method:   rabbitmqhandler.RequestMethodPut,
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

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockQueue.EXPECT().UpdateWaitActionsAndTimeouts(gomock.Any(), tt.id, tt.waitActions, tt.waitTimeout, tt.serviceTimeout).Return(nil)

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
