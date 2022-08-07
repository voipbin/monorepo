package queuehandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	cfconference "gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	fmflow "gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queue"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/dbhandler"
)

func Test_Create(t *testing.T) {

	tests := []struct {
		name string

		customerID     uuid.UUID
		queueName      string
		detail         string
		routingMethod  queue.RoutingMethod
		tagIDs         []uuid.UUID
		waitActions    []fmaction.Action
		waitTimeout    int
		serviceTimeout int

		responseConference *cfconference.Conference
		responseFlow       *fmflow.Flow

		expectRes *queue.Queue
	}{
		{
			"normal",

			uuid.FromStringOrNil("1ed812a6-7f56-11ec-82c1-8bb47b0f9d98"),
			"name",
			"detail",
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

			&cfconference.Conference{
				ID: uuid.FromStringOrNil("ad4c17a0-60e6-11ec-9eeb-e76c2c4c7fd4"),
			},
			&fmflow.Flow{
				Actions: []fmaction.Action{
					{
						ID:   uuid.FromStringOrNil("1cf6612c-60e8-11ec-810d-a79b29cef25c"),
						Type: fmaction.TypeConferenceJoin,
					},
				},
			},

			&queue.Queue{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			ctx := context.Background()

			mockDB.EXPECT().QueueCreate(ctx, gomock.Any()).Return(nil)
			mockDB.EXPECT().QueueGet(ctx, gomock.Any()).Return(&queue.Queue{}, nil)

			res, err := h.Create(
				ctx,
				tt.customerID,
				tt.queueName,
				tt.detail,
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

func Test_createQueueFlow(t *testing.T) {

	tests := []struct {
		name string

		customerID   uuid.UUID
		queueID      uuid.UUID
		conferenceID uuid.UUID
		waitActions  []fmaction.Action

		flowName     string
		flowActions  []fmaction.Action
		responseFlow *fmflow.Flow

		expectRes *fmflow.Flow
	}{
		{
			"normal",

			uuid.FromStringOrNil("66f84114-7f56-11ec-8e9a-ff1aed535827"),
			uuid.FromStringOrNil("61a4651c-60e3-11ec-86ff-efca21ef8707"),
			uuid.FromStringOrNil("61d32f5a-60e3-11ec-943d-db1b16329a1c"),
			[]fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
					ID:   uuid.FromStringOrNil("290f9c8a-adf5-11ec-93c7-4f5277bca38c"),
				},
			},

			"queue-61a4651c-60e3-11ec-86ff-efca21ef8707",
			[]fmaction.Action{
				{
					Type:   fmaction.TypeAnswer,
					ID:     uuid.FromStringOrNil("290f9c8a-adf5-11ec-93c7-4f5277bca38c"),
					NextID: uuid.FromStringOrNil("290f9c8a-adf5-11ec-93c7-4f5277bca38c"),
				},
				{
					Type:   fmaction.TypeConferenceJoin,
					Option: []byte(`{"conference_id":"61d32f5a-60e3-11ec-943d-db1b16329a1c"}`),
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

			ctx := context.Background()

			mockReq.EXPECT().FMV1FlowCreate(ctx, tt.customerID, fmflow.TypeQueue, tt.flowName, "generated for queue by queue-manager.", tt.flowActions, false).Return(tt.responseFlow, nil)

			res, err := h.createQueueFlow(ctx, tt.customerID, tt.queueID, tt.conferenceID, tt.waitActions)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_createQueueFlowActions(t *testing.T) {

	tests := []struct {
		name string

		waitActions  []fmaction.Action
		conferenceID uuid.UUID

		expectRes []fmaction.Action
	}{
		{
			"normal",

			[]fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
					ID:   uuid.FromStringOrNil("522072fc-adf5-11ec-83ee-13ad3cde9282"),
				},
			},
			uuid.FromStringOrNil("f1b786fa-60e0-11ec-82a4-a3997b361548"),

			[]fmaction.Action{
				{
					Type:   fmaction.TypeAnswer,
					ID:     uuid.FromStringOrNil("522072fc-adf5-11ec-83ee-13ad3cde9282"),
					NextID: uuid.FromStringOrNil("522072fc-adf5-11ec-83ee-13ad3cde9282"),
				},
				{
					Type:   fmaction.TypeConferenceJoin,
					Option: []byte(`{"conference_id":"f1b786fa-60e0-11ec-82a4-a3997b361548"}`),
				},
			},
		},
		{
			"2 wait actions",

			[]fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
					ID:   uuid.FromStringOrNil("604da3cc-adf5-11ec-9f76-cb5d6f62ee83"),
				},
				{
					Type:   fmaction.TypeTalk,
					ID:     uuid.FromStringOrNil("6f39e616-adf5-11ec-b0dd-9f59cb5454d9"),
					Option: []byte(`{"text":"hello"}`),
				},
			},
			uuid.FromStringOrNil("d9ad87d8-60e2-11ec-8fe6-7bb5167cee96"),

			[]fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
					ID:   uuid.FromStringOrNil("604da3cc-adf5-11ec-9f76-cb5d6f62ee83"),
				},
				{
					Type:   fmaction.TypeTalk,
					ID:     uuid.FromStringOrNil("6f39e616-adf5-11ec-b0dd-9f59cb5454d9"),
					NextID: uuid.FromStringOrNil("604da3cc-adf5-11ec-9f76-cb5d6f62ee83"),
					Option: []byte(`{"text":"hello"}`),
				},
				{
					Type:   fmaction.TypeConferenceJoin,
					Option: []byte(`{"conference_id":"d9ad87d8-60e2-11ec-8fe6-7bb5167cee96"}`),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			res, err := h.createQueueFlowActions(tt.waitActions, tt.conferenceID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_getForwardActionID(t *testing.T) {

	tests := []struct {
		name string

		flow *fmflow.Flow

		expectRes uuid.UUID
	}{
		{
			"normal",

			&fmflow.Flow{
				Actions: []fmaction.Action{
					{
						ID:   uuid.FromStringOrNil("550a33ee-60df-11ec-9fbd-8f75958d1453"),
						Type: fmaction.TypeAnswer,
					},
					{
						ID:   uuid.FromStringOrNil("78534d7c-60df-11ec-805c-7786412e957a"),
						Type: fmaction.TypeConferenceJoin,
					},
				},
			},

			uuid.FromStringOrNil("78534d7c-60df-11ec-805c-7786412e957a"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			ctx := context.Background()

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
