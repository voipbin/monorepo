package queuehandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	cmconfbridge "gitlab.com/voipbin/bin-manager/call-manager.git/models/confbridge"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	fmflow "gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queue"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/dbhandler"
)

func TestCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	h := &queueHandler{
		db:            mockDB,
		reqHandler:    mockReq,
		notifyhandler: mockNotify,
	}

	tests := []struct {
		name string

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

		responseConfbridge *cmconfbridge.Confbridge
		responseFlow       *fmflow.Flow

		expectRes *queue.Queue
	}{
		{
			"normal",

			1,
			"name",
			"detail",
			"test.com",
			"POST",
			queue.RoutingMethodRandom,
			[]uuid.UUID{
				uuid.FromStringOrNil("074b6e1e-60e6-11ec-9dc5-4bc92b81a572"),
			},
			[]fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
			},
			100000,
			1000000,

			&cmconfbridge.Confbridge{
				ID: uuid.FromStringOrNil("ad4c17a0-60e6-11ec-9eeb-e76c2c4c7fd4"),
			},
			&fmflow.Flow{
				Actions: []fmaction.Action{
					{
						ID:   uuid.FromStringOrNil("1cf6612c-60e8-11ec-810d-a79b29cef25c"),
						Type: fmaction.TypeConfbridgeJoin,
					},
				},
			},

			&queue.Queue{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// mockReq.EXPECT().CMV1ConfbridgeCreate(gomock.Any()).Return(tt.responseConfbridge, nil)
			// mockReq.EXPECT().FMV1FlowCreate(gomock.Any(), tt.userID, fmflow.TypeQueue, gomock.Any(), gomock.Any(), "", gomock.Any(), true).Return(&fmflow.Flow{}, nil)
			// mockReq.EXPECT().FMV1FlowGet(gomock.Any(), gomock.Any()).Return(tt.responseFlow, nil)

			mockDB.EXPECT().QueueCreate(gomock.Any(), gomock.Any()).Return(nil)
			mockDB.EXPECT().QueueGet(gomock.Any(), gomock.Any()).Return(&queue.Queue{}, nil)

			res, err := h.Create(
				ctx,
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
			)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func TestCreateQueueFlow(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	h := &queueHandler{
		db:            mockDB,
		reqHandler:    mockReq,
		notifyhandler: mockNotify,
	}

	tests := []struct {
		name string

		userID       uint64
		queueID      uuid.UUID
		confbridgeID uuid.UUID
		waitActions  []fmaction.Action

		flowName     string
		flowActions  []fmaction.Action
		responseFlow *fmflow.Flow

		expectRes *fmflow.Flow
	}{
		{
			"normal",

			1,
			uuid.FromStringOrNil("61a4651c-60e3-11ec-86ff-efca21ef8707"),
			uuid.FromStringOrNil("61d32f5a-60e3-11ec-943d-db1b16329a1c"),
			[]fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
			},

			"queue-61a4651c-60e3-11ec-86ff-efca21ef8707",
			[]fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
				{
					Type:   fmaction.TypeGoto,
					Option: []byte(`{"target_index":0,"target_id":"00000000-0000-0000-0000-000000000000","loop":false,"loop_count":0}`),
				},
				{
					Type:   fmaction.TypeConfbridgeJoin,
					Option: []byte(`{"confbridge_id":"61d32f5a-60e3-11ec-943d-db1b16329a1c"}`),
				},
			},
			&fmflow.Flow{
				ID: uuid.FromStringOrNil("ebcbfe7a-60e4-11ec-aaf3-738bb5a45bb0"),
			},

			&fmflow.Flow{
				ID: uuid.FromStringOrNil("ebcbfe7a-60e4-11ec-aaf3-738bb5a45bb0"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockReq.EXPECT().FMV1FlowCreate(gomock.Any(), tt.userID, fmflow.TypeQueue, tt.flowName, "generated for queue by queue-manager.", "", tt.flowActions, true).Return(tt.responseFlow, nil)

			res, err := h.createQueueFlow(ctx, tt.userID, tt.queueID, tt.confbridgeID, tt.waitActions)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func TestCreateQueueFlowActions(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	h := &queueHandler{
		db:            mockDB,
		reqHandler:    mockReq,
		notifyhandler: mockNotify,
	}

	tests := []struct {
		name string

		waitActions  []fmaction.Action
		confbridgeID uuid.UUID

		expectRes []fmaction.Action
	}{
		{
			"normal",

			[]fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
			},
			uuid.FromStringOrNil("f1b786fa-60e0-11ec-82a4-a3997b361548"),

			[]fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
				{
					Type:   fmaction.TypeGoto,
					Option: []byte(`{"target_index":0,"target_id":"00000000-0000-0000-0000-000000000000","loop":false,"loop_count":0}`),
				},
				{
					Type:   fmaction.TypeConfbridgeJoin,
					Option: []byte(`{"confbridge_id":"f1b786fa-60e0-11ec-82a4-a3997b361548"}`),
				},
			},
		},
		{
			"2 wait actions",

			[]fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
				{
					Type:   fmaction.TypeTalk,
					Option: []byte(`{"text":"hello"}`),
				},
			},
			uuid.FromStringOrNil("d9ad87d8-60e2-11ec-8fe6-7bb5167cee96"),

			[]fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
				{
					Type:   fmaction.TypeTalk,
					Option: []byte(`{"text":"hello"}`),
				},
				{
					Type:   fmaction.TypeGoto,
					Option: []byte(`{"target_index":0,"target_id":"00000000-0000-0000-0000-000000000000","loop":false,"loop_count":0}`),
				},
				{
					Type:   fmaction.TypeConfbridgeJoin,
					Option: []byte(`{"confbridge_id":"d9ad87d8-60e2-11ec-8fe6-7bb5167cee96"}`),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res, err := h.createQueueFlowActions(tt.waitActions, tt.confbridgeID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func TestGetForwardActionID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	h := &queueHandler{
		db:            mockDB,
		reqHandler:    mockReq,
		notifyhandler: mockNotify,
	}

	tests := []struct {
		name string

		// flowID       uuid.UUID
		flow *fmflow.Flow

		expectRes uuid.UUID
	}{
		{
			"normal",

			// uuid.FromStringOrNil("f21fdbc6-60de-11ec-ad49-5fa7a8a1a0fc"),
			&fmflow.Flow{
				Actions: []fmaction.Action{
					{
						ID:   uuid.FromStringOrNil("550a33ee-60df-11ec-9fbd-8f75958d1453"),
						Type: fmaction.TypeAnswer,
					},
					{
						ID:   uuid.FromStringOrNil("78534d7c-60df-11ec-805c-7786412e957a"),
						Type: fmaction.TypeConfbridgeJoin,
					},
				},
			},

			uuid.FromStringOrNil("78534d7c-60df-11ec-805c-7786412e957a"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// mockReq.EXPECT().FMV1FlowGet(gomock.Any(), tt.flowID).Return(tt.responseFlow, nil)

			res, err := h.getForwardActionID(ctx, tt.flow)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}
