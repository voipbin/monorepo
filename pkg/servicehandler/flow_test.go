package servicehandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	fmflow "gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"

	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
)

func TestFlowCreate(t *testing.T) {

	tests := []struct {
		name     string
		customer *cscustomer.Customer

		flowName string
		detail   string
		actions  []fmaction.Action
		persist  bool

		response  *fmflow.Flow
		expectRes *fmflow.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},

			"test name",
			"test detail",
			[]fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
			},
			true,

			&fmflow.Flow{
				ID: uuid.FromStringOrNil("50daef5a-f2f6-11ea-9649-33c2eb34ec4c"),
			},
			&fmflow.WebhookMessage{
				ID: uuid.FromStringOrNil("50daef5a-f2f6-11ea-9649-33c2eb34ec4c"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().FlowV1FlowCreate(ctx, tt.customer.ID, fmflow.TypeFlow, tt.flowName, tt.detail, tt.actions, tt.persist).Return(tt.response, nil)
			res, err := h.FlowCreate(ctx, tt.customer, tt.flowName, tt.detail, tt.actions, tt.persist)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func TestFlowDelete(t *testing.T) {

	tests := []struct {
		name     string
		customer *cscustomer.Customer
		flowID   uuid.UUID

		response  *fmflow.Flow
		expectRes *fmflow.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			uuid.FromStringOrNil("00efc020-67cb-11eb-bd5e-b3c491185912"),

			&fmflow.Flow{
				ID:         uuid.FromStringOrNil("00efc020-67cb-11eb-bd5e-b3c491185912"),
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
				Name:       "test",
				Detail:     "test detail",
				Actions:    []fmaction.Action{},
			},
			&fmflow.WebhookMessage{
				ID:         uuid.FromStringOrNil("00efc020-67cb-11eb-bd5e-b3c491185912"),
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
				Name:       "test",
				Detail:     "test detail",
				Actions:    []fmaction.Action{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().FlowV1FlowGet(ctx, tt.flowID).Return(tt.response, nil)
			mockReq.EXPECT().FlowV1FlowDelete(ctx, tt.flowID).Return(tt.response, nil)

			res, err := h.FlowDelete(ctx, tt.customer, tt.flowID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func TestFlowGet(t *testing.T) {

	tests := []struct {
		name     string
		customer *cscustomer.Customer
		flowID   uuid.UUID

		response  *fmflow.Flow
		expectRes *fmflow.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			uuid.FromStringOrNil("1f80baf0-0c5c-11eb-9df4-1f217b30d87c"),

			&fmflow.Flow{
				ID:         uuid.FromStringOrNil("1f80baf0-0c5c-11eb-9df4-1f217b30d87c"),
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
				Name:       "test",
				Detail:     "test detail",
				Actions:    []fmaction.Action{},
			},
			&fmflow.WebhookMessage{
				ID:         uuid.FromStringOrNil("1f80baf0-0c5c-11eb-9df4-1f217b30d87c"),
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
				Name:       "test",
				Detail:     "test detail",
				Actions:    []fmaction.Action{},
			},
		},
		{
			"action answer",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			uuid.FromStringOrNil("5ce8210a-66af-11eb-a7f4-a36a8393fce1"),

			&fmflow.Flow{
				ID:         uuid.FromStringOrNil("5ce8210a-66af-11eb-a7f4-a36a8393fce1"),
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
				Name:       "test",
				Detail:     "test detail",
				Actions: []fmaction.Action{
					{
						ID:   uuid.FromStringOrNil("61f86f60-66af-11eb-917f-838fd6836e1f"),
						Type: fmaction.TypeAnswer,
					},
				},
			},
			&fmflow.WebhookMessage{
				ID:         uuid.FromStringOrNil("5ce8210a-66af-11eb-a7f4-a36a8393fce1"),
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
				Name:       "test",
				Detail:     "test detail",
				Actions: []fmaction.Action{
					{
						ID:   uuid.FromStringOrNil("61f86f60-66af-11eb-917f-838fd6836e1f"),
						Type: "answer",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().FlowV1FlowGet(ctx, tt.flowID).Return(tt.response, nil)

			res, err := h.FlowGet(ctx, tt.customer, tt.flowID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func TestFlowGets(t *testing.T) {

	tests := []struct {
		name      string
		customer  *cscustomer.Customer
		pageToken string
		pageSize  uint64

		response  []fmflow.Flow
		expectRes []*fmflow.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			"2020-10-20T01:00:00.995000",
			10,

			[]fmflow.Flow{
				{
					ID:         uuid.FromStringOrNil("ccda6eb2-0c5c-11eb-ae7e-a3ae4bcd3975"),
					CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
					Name:       "test1",
					Detail:     "test detail1",
					Actions:    []fmaction.Action{},
				},
				{
					ID:         uuid.FromStringOrNil("d950aef4-0c5c-11eb-82dd-3b31d4ba2ea4"),
					CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
					Name:       "test2",
					Detail:     "test detail2",
					Actions:    []fmaction.Action{},
				},
			},
			[]*fmflow.WebhookMessage{
				{
					ID:         uuid.FromStringOrNil("ccda6eb2-0c5c-11eb-ae7e-a3ae4bcd3975"),
					CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
					Name:       "test1",
					Detail:     "test detail1",
					Actions:    []fmaction.Action{},
				},
				{
					ID:         uuid.FromStringOrNil("d950aef4-0c5c-11eb-82dd-3b31d4ba2ea4"),
					CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
					Name:       "test2",
					Detail:     "test detail2",
					Actions:    []fmaction.Action{},
				},
			},
		},
		{
			"1 action",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			"2020-10-20T01:00:00.995000",
			10,

			[]fmflow.Flow{
				{
					ID:         uuid.FromStringOrNil("5a109d00-66ae-11eb-ad00-bbcf73569888"),
					CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
					Name:       "test1",
					Detail:     "test detail1",
					Actions: []fmaction.Action{
						{
							ID:   uuid.FromStringOrNil("775f5cde-66ae-11eb-9626-0f488d332e1e"),
							Type: fmaction.TypeAnswer,
						},
					},
				},
			},
			[]*fmflow.WebhookMessage{
				{
					ID:         uuid.FromStringOrNil("5a109d00-66ae-11eb-ad00-bbcf73569888"),
					CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
					Name:       "test1",
					Detail:     "test detail1",
					Actions: []fmaction.Action{
						{
							ID:   uuid.FromStringOrNil("775f5cde-66ae-11eb-9626-0f488d332e1e"),
							Type: "answer",
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().FlowV1FlowGets(ctx, tt.customer.ID, fmflow.TypeFlow, tt.pageToken, tt.pageSize).Return(tt.response, nil)

			res, err := h.FlowGets(ctx, tt.customer, tt.pageSize, tt.pageToken)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func TestFlowUpdate(t *testing.T) {

	tests := []struct {
		name     string
		customer *cscustomer.Customer

		flowID   uuid.UUID
		flowName string
		detail   string
		actions  []fmaction.Action

		response  *fmflow.Flow
		expectRes *fmflow.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},

			uuid.FromStringOrNil("a64ff8ce-1ab3-4564-9d34-e5f3147810e5"),
			"test name",
			"test detail",
			[]fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
			},

			&fmflow.Flow{
				ID:         uuid.FromStringOrNil("a64ff8ce-1ab3-4564-9d34-e5f3147810e5"),
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			&fmflow.WebhookMessage{
				ID:         uuid.FromStringOrNil("a64ff8ce-1ab3-4564-9d34-e5f3147810e5"),
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			f := &fmflow.Flow{
				ID:      tt.flowID,
				Name:    tt.flowName,
				Detail:  tt.detail,
				Actions: tt.actions,
			}

			mockReq.EXPECT().FlowV1FlowGet(ctx, tt.flowID).Return(tt.response, nil)
			mockReq.EXPECT().FlowV1FlowUpdate(ctx, f).Return(tt.response, nil)
			res, err := h.FlowUpdate(ctx, tt.customer, f)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_FlowUpdateActions(t *testing.T) {

	tests := []struct {
		name     string
		customer *cscustomer.Customer
		flowID   uuid.UUID
		actions  []fmaction.Action

		response  *fmflow.Flow
		expectRes *fmflow.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			uuid.FromStringOrNil("1058806a-45c1-4bc0-9605-1148e20008c1"),
			[]fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
			},

			&fmflow.Flow{
				ID:         uuid.FromStringOrNil("00498856-678d-11eb-89a6-37bc9314dc94"),
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},

			&fmflow.WebhookMessage{
				ID:         uuid.FromStringOrNil("00498856-678d-11eb-89a6-37bc9314dc94"),
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().FlowV1FlowGet(ctx, tt.flowID).Return(tt.response, nil)
			mockReq.EXPECT().FlowV1FlowUpdateActions(ctx, tt.flowID, tt.actions).Return(tt.response, nil)
			res, err := h.FlowUpdateActions(ctx, tt.customer, tt.flowID, tt.actions)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}
