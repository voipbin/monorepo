package servicehandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	qmqueue "gitlab.com/voipbin/bin-manager/queue-manager.git/models/queue"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
)

func TestQueueGets(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

	type test struct {
		name      string
		user      *user.User
		pageToken string
		pageSize  uint64

		response  []qmqueue.Queue
		expectRes []*qmqueue.WebhookMessage
	}

	tests := []test{
		{
			"normal",
			&user.User{
				ID: 1,
			},
			"2021-03-01 01:00:00.995000",
			10,

			[]qmqueue.Queue{
				{
					ID: uuid.FromStringOrNil("2130337e-7b1c-11eb-a431-b714a0a4b6fc"),
				},
			},
			[]*qmqueue.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("2130337e-7b1c-11eb-a431-b714a0a4b6fc"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReq.EXPECT().QMV1QueueGets(gomock.Any(), tt.user.ID, tt.pageToken, tt.pageSize).Return(tt.response, nil)

			res, err := h.QueueGets(tt.user, tt.pageSize, tt.pageToken)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			for _, num := range res {
				num.TMCreate = ""
				num.TMUpdate = ""
				num.TMDelete = ""
			}

			if !reflect.DeepEqual(res[0], tt.expectRes[0]) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes[0], res[0])
			}
		})
	}
}

func TestQueueGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

	type test struct {
		name string
		user *user.User
		id   uuid.UUID

		response  *qmqueue.Queue
		expectRes *qmqueue.WebhookMessage
	}

	tests := []test{
		{
			"normal",
			&user.User{
				ID: 1,
			},
			uuid.FromStringOrNil("17bd8d64-7be4-11eb-b887-8f1b24b98639"),

			&qmqueue.Queue{
				ID:     uuid.FromStringOrNil("17bd8d64-7be4-11eb-b887-8f1b24b98639"),
				UserID: 1,
			},
			&qmqueue.WebhookMessage{
				ID: uuid.FromStringOrNil("17bd8d64-7be4-11eb-b887-8f1b24b98639"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockReq.EXPECT().QMV1QueueGet(gomock.Any(), tt.id).Return(tt.response, nil)

			res, err := h.QueueGet(tt.user, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func TestQueueCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

	type test struct {
		name string

		user           *user.User
		queueName      string
		detail         string
		webhookURI     string
		webhookMethod  string
		routingMethod  qmqueue.RoutingMethod
		tagIDs         []uuid.UUID
		waitActions    []fmaction.Action
		timeoutWait    int
		timeoutService int

		response  *qmqueue.Queue
		expectRes *qmqueue.WebhookMessage
	}

	tests := []test{
		{
			"normal",

			&user.User{
				ID: 1,
			},
			"name",
			"detail",
			"test.com",
			"POST",
			qmqueue.RoutingMethodRandom,
			[]uuid.UUID{
				uuid.FromStringOrNil("2a743344-6316-11ec-b247-af52c2375309"),
			},
			[]fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
			},
			100000,
			1000000,

			&qmqueue.Queue{
				ID:     uuid.FromStringOrNil("eb2ee214-6316-11ec-88b2-db9da3dd0931"),
				UserID: 1,
			},
			&qmqueue.WebhookMessage{
				ID: uuid.FromStringOrNil("eb2ee214-6316-11ec-88b2-db9da3dd0931"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockReq.EXPECT().QMV1QueueCreate(
				gomock.Any(),
				tt.user.ID,
				tt.queueName,
				tt.detail,
				tt.webhookURI,
				tt.webhookMethod,
				tt.routingMethod,
				tt.tagIDs,
				tt.waitActions,
				tt.timeoutWait,
				tt.timeoutService,
			).Return(tt.response, nil)

			res, err := h.QueueCreate(
				tt.user,
				tt.queueName,
				tt.detail,
				tt.webhookURI,
				tt.webhookMethod,
				string(tt.routingMethod),
				tt.tagIDs,
				tt.waitActions,
				tt.timeoutWait,
				tt.timeoutService,
			)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func TestQueueDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

	type test struct {
		name string

		user    *user.User
		queueID uuid.UUID

		response *qmqueue.Queue
	}

	tests := []test{
		{
			"normal",

			&user.User{
				ID: 1,
			},
			uuid.FromStringOrNil("6aa878a2-6317-11ec-94b7-c7ba9436173f"),

			&qmqueue.Queue{
				ID:     uuid.FromStringOrNil("6aa878a2-6317-11ec-94b7-c7ba9436173f"),
				UserID: 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockReq.EXPECT().QMV1QueueGet(gomock.Any(), tt.queueID).Return(tt.response, nil)
			mockReq.EXPECT().QMV1QueueDelete(gomock.Any(), tt.queueID).Return(nil)

			if err := h.QueueDelete(tt.user, tt.queueID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestQueueUpdate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

	type test struct {
		name string

		user          *user.User
		queueID       uuid.UUID
		queueName     string
		detail        string
		webhookURI    string
		webhookMethod string

		response *qmqueue.Queue
	}

	tests := []test{
		{
			"normal",

			&user.User{
				ID: 1,
			},
			uuid.FromStringOrNil("116b515e-6391-11ec-a2ab-2b13d87ce328"),
			"name",
			"detail",
			"test.com",
			"POST",

			&qmqueue.Queue{
				ID:     uuid.FromStringOrNil("116b515e-6391-11ec-a2ab-2b13d87ce328"),
				UserID: 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockReq.EXPECT().QMV1QueueGet(gomock.Any(), tt.queueID).Return(tt.response, nil)
			mockReq.EXPECT().QMV1QueueUpdate(gomock.Any(), tt.queueID, tt.queueName, tt.detail, tt.webhookURI, tt.webhookMethod).Return(nil)

			if err := h.QueueUpdate(tt.user, tt.queueID, tt.queueName, tt.detail, tt.webhookURI, tt.webhookMethod); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestQueueUpdateTagIDs(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

	type test struct {
		name string

		user    *user.User
		queueID uuid.UUID
		tagIDs  []uuid.UUID

		response *qmqueue.Queue
	}

	tests := []test{
		{
			"normal",

			&user.User{
				ID: 1,
			},
			uuid.FromStringOrNil("4f10fcca-6391-11ec-b1a8-cf59a893226a"),
			[]uuid.UUID{
				uuid.FromStringOrNil("50c7c31e-6391-11ec-b1f6-cb24701d7df3"),
			},

			&qmqueue.Queue{
				ID:     uuid.FromStringOrNil("4f10fcca-6391-11ec-b1a8-cf59a893226a"),
				UserID: 1,
			},
		},
		{
			"2 items",

			&user.User{
				ID: 1,
			},
			uuid.FromStringOrNil("7472d542-6391-11ec-8e92-6f12cb507950"),
			[]uuid.UUID{
				uuid.FromStringOrNil("74963b9a-6391-11ec-84ae-337b926b8136"),
				uuid.FromStringOrNil("74b790d8-6391-11ec-be28-5fd8bcbf3b9c"),
			},

			&qmqueue.Queue{
				ID:     uuid.FromStringOrNil("7472d542-6391-11ec-8e92-6f12cb507950"),
				UserID: 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockReq.EXPECT().QMV1QueueGet(gomock.Any(), tt.queueID).Return(tt.response, nil)
			mockReq.EXPECT().QMV1QueueUpdateTagIDs(gomock.Any(), tt.queueID, tt.tagIDs).Return(nil)

			if err := h.QueueUpdateTagIDs(tt.user, tt.queueID, tt.tagIDs); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestQueueUpdateRoutingMethod(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

	type test struct {
		name string

		user          *user.User
		queueID       uuid.UUID
		routingMethod qmqueue.RoutingMethod

		response *qmqueue.Queue
	}

	tests := []test{
		{
			"routing method random",

			&user.User{
				ID: 1,
			},
			uuid.FromStringOrNil("af14400a-6391-11ec-baed-7fb98aebe61a"),
			qmqueue.RoutingMethodRandom,

			&qmqueue.Queue{
				ID:     uuid.FromStringOrNil("af14400a-6391-11ec-baed-7fb98aebe61a"),
				UserID: 1,
			},
		},
		{
			"routing method none",

			&user.User{
				ID: 1,
			},
			uuid.FromStringOrNil("af2efe86-6391-11ec-8100-c3e8d3057916"),
			qmqueue.RoutingMethodNone,

			&qmqueue.Queue{
				ID:     uuid.FromStringOrNil("af2efe86-6391-11ec-8100-c3e8d3057916"),
				UserID: 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockReq.EXPECT().QMV1QueueGet(gomock.Any(), tt.queueID).Return(tt.response, nil)
			mockReq.EXPECT().QMV1QueueUpdateRoutingMethod(gomock.Any(), tt.queueID, tt.routingMethod).Return(nil)

			if err := h.QueueUpdateRoutingMethod(tt.user, tt.queueID, tt.routingMethod); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestQueueUpdateActions(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

	type test struct {
		name string

		user           *user.User
		queueID        uuid.UUID
		waitActions    []fmaction.Action
		timeoutWait    int
		timeoutService int

		response *qmqueue.Queue
	}

	tests := []test{
		{
			"routing method random",

			&user.User{
				ID: 1,
			},
			uuid.FromStringOrNil("f4fc8e6a-6391-11ec-bd03-337ff376d96d"),
			[]fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
			},
			10000,
			100000,

			&qmqueue.Queue{
				ID:     uuid.FromStringOrNil("f4fc8e6a-6391-11ec-bd03-337ff376d96d"),
				UserID: 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockReq.EXPECT().QMV1QueueGet(gomock.Any(), tt.queueID).Return(tt.response, nil)
			mockReq.EXPECT().QMV1QueueUpdateActions(gomock.Any(), tt.queueID, tt.waitActions, tt.timeoutWait, tt.timeoutService).Return(nil)

			if err := h.QueueUpdateActions(tt.user, tt.queueID, tt.waitActions, tt.timeoutWait, tt.timeoutService); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
