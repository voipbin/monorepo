package servicehandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"

	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
)

func TestAgentCreate(t *testing.T) {

	tests := []struct {
		name string

		customer      *cscustomer.Customer
		username      string
		password      string
		agentName     string
		detail        string
		webhookMethod string
		webhookURI    string
		ringMethod    amagent.RingMethod
		permission    amagent.Permission
		tagIDs        []uuid.UUID
		addresses     []commonaddress.Address

		response  *amagent.Agent
		expectRes *amagent.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("852b9d5e-7ff9-11ec-9ca0-cf3c47e8c96b"),
			},
			"test1",
			"password1",
			"test1 name",
			"test1 detail",
			"",
			"",
			"ringall",
			0,
			[]uuid.UUID{},
			[]commonaddress.Address{},

			&amagent.Agent{
				ID: uuid.FromStringOrNil("b3216dac-4fba-11ec-8551-5b4f1596d5f9"),
			},
			&amagent.WebhookMessage{
				ID: uuid.FromStringOrNil("b3216dac-4fba-11ec-8551-5b4f1596d5f9"),
			},
		},
		{
			"have webhook",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("852b9d5e-7ff9-11ec-9ca0-cf3c47e8c96b"),
			},
			"test1",
			"password1",
			"test1 name",
			"test1 detail",
			"POST",
			"test.com",
			"ringall",
			0,
			[]uuid.UUID{},
			[]commonaddress.Address{},

			&amagent.Agent{
				ID: uuid.FromStringOrNil("3d39a6c2-79ae-11ec-8f44-6bc6091af769"),
			},
			&amagent.WebhookMessage{
				ID: uuid.FromStringOrNil("3d39a6c2-79ae-11ec-8f44-6bc6091af769"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().AgentV1AgentCreate(ctx, 30, tt.customer.ID, tt.username, tt.password, tt.agentName, tt.detail, tt.ringMethod, tt.permission, tt.tagIDs, tt.addresses).Return(tt.response, nil)

			res, err := h.AgentCreate(ctx, tt.customer, tt.username, tt.password, tt.agentName, tt.detail, tt.ringMethod, tt.permission, tt.tagIDs, tt.addresses)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(*res, *tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", *tt.expectRes, *res)
			}
		})
	}
}

func TestAgentGet(t *testing.T) {

	tests := []struct {
		name string

		customer *cscustomer.Customer
		agentID  uuid.UUID

		response  *amagent.Agent
		expectRes *amagent.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("852b9d5e-7ff9-11ec-9ca0-cf3c47e8c96b"),
			},
			uuid.FromStringOrNil("450c8f6a-5067-11ec-bda4-039a4b9a1158"),

			&amagent.Agent{
				ID:         uuid.FromStringOrNil("b3216dac-4fba-11ec-8551-5b4f1596d5f9"),
				CustomerID: uuid.FromStringOrNil("852b9d5e-7ff9-11ec-9ca0-cf3c47e8c96b"),
			},
			&amagent.WebhookMessage{
				ID:         uuid.FromStringOrNil("b3216dac-4fba-11ec-8551-5b4f1596d5f9"),
				CustomerID: uuid.FromStringOrNil("852b9d5e-7ff9-11ec-9ca0-cf3c47e8c96b"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().AgentV1AgentGet(ctx, tt.agentID).Return(tt.response, nil)

			res, err := h.AgentGet(ctx, tt.customer, tt.agentID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func TestAgentGets(t *testing.T) {

	tests := []struct {
		name string

		customer *cscustomer.Customer
		size     uint64
		token    string
		tagIDs   []uuid.UUID
		status   amagent.Status

		response  []amagent.Agent
		expectRes []*amagent.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("852b9d5e-7ff9-11ec-9ca0-cf3c47e8c96b"),
			},
			10,
			"2020-09-20 03:23:20.995000",
			[]uuid.UUID{},
			"",

			[]amagent.Agent{
				{
					ID: uuid.FromStringOrNil("b3216dac-4fba-11ec-8551-5b4f1596d5f9"),
				},
			},
			[]*amagent.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("b3216dac-4fba-11ec-8551-5b4f1596d5f9"),
				},
			},
		},
		{
			"2 agents",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("852b9d5e-7ff9-11ec-9ca0-cf3c47e8c96b"),
			},
			10,
			"2020-09-20 03:23:20.995000",
			[]uuid.UUID{},
			"",

			[]amagent.Agent{
				{
					ID: uuid.FromStringOrNil("b3216dac-4fba-11ec-8551-5b4f1596d5f9"),
				},
				{
					ID: uuid.FromStringOrNil("c0f620ee-4fbf-11ec-87b2-7372cbac1bb0"),
				},
			},
			[]*amagent.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("b3216dac-4fba-11ec-8551-5b4f1596d5f9"),
				},
				{
					ID: uuid.FromStringOrNil("c0f620ee-4fbf-11ec-87b2-7372cbac1bb0"),
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
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().AgentV1AgentGets(ctx, tt.customer.ID, tt.token, tt.size).Return(tt.response, nil)

			res, err := h.AgentGets(ctx, tt.customer, tt.size, tt.token, tt.tagIDs, tt.status)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func TestAgentGetsByTagIDs(t *testing.T) {

	tests := []struct {
		name string

		customer *cscustomer.Customer
		size     uint64
		token    string
		tagIDs   []uuid.UUID
		status   amagent.Status

		response  []amagent.Agent
		expectRes []*amagent.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("852b9d5e-7ff9-11ec-9ca0-cf3c47e8c96b"),
			},
			10,
			"2020-09-20 03:23:20.995000",
			[]uuid.UUID{
				uuid.FromStringOrNil("ed33fa28-4fbf-11ec-9aab-efb29082f61d"),
			},
			"",

			[]amagent.Agent{
				{
					ID: uuid.FromStringOrNil("b3216dac-4fba-11ec-8551-5b4f1596d5f9"),
				},
			},
			[]*amagent.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("b3216dac-4fba-11ec-8551-5b4f1596d5f9"),
				},
			},
		},
		{
			"2 tag ids",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("852b9d5e-7ff9-11ec-9ca0-cf3c47e8c96b"),
			},
			10,
			"2020-09-20 03:23:20.995000",
			[]uuid.UUID{
				uuid.FromStringOrNil("ed33fa28-4fbf-11ec-9aab-efb29082f61d"),
				uuid.FromStringOrNil("07eec5dc-4fc0-11ec-adfb-dbfffb9e6dc5"),
			},
			"",

			[]amagent.Agent{
				{
					ID: uuid.FromStringOrNil("b3216dac-4fba-11ec-8551-5b4f1596d5f9"),
				},
			},
			[]*amagent.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("b3216dac-4fba-11ec-8551-5b4f1596d5f9"),
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
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().AgentV1AgentGetsByTagIDs(ctx, tt.customer.ID, tt.tagIDs).Return(tt.response, nil)

			res, err := h.AgentGets(ctx, tt.customer, tt.size, tt.token, tt.tagIDs, tt.status)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func TestAgentGetsByTagIDsAndStatus(t *testing.T) {

	tests := []struct {
		name string

		customer *cscustomer.Customer
		size     uint64
		token    string
		tagIDs   []uuid.UUID
		status   amagent.Status

		response  []amagent.Agent
		expectRes []*amagent.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("852b9d5e-7ff9-11ec-9ca0-cf3c47e8c96b"),
			},
			10,
			"2020-09-20T03:23:20.995000",
			[]uuid.UUID{
				uuid.FromStringOrNil("ed33fa28-4fbf-11ec-9aab-efb29082f61d"),
			},
			amagent.StatusAvailable,

			[]amagent.Agent{
				{
					ID: uuid.FromStringOrNil("b3216dac-4fba-11ec-8551-5b4f1596d5f9"),
				},
			},
			[]*amagent.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("b3216dac-4fba-11ec-8551-5b4f1596d5f9"),
				},
			},
		},
		{
			"2 tag ids",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("852b9d5e-7ff9-11ec-9ca0-cf3c47e8c96b"),
			},
			10,
			"2020-09-20T03:23:20.995000",
			[]uuid.UUID{
				uuid.FromStringOrNil("ed33fa28-4fbf-11ec-9aab-efb29082f61d"),
				uuid.FromStringOrNil("07eec5dc-4fc0-11ec-adfb-dbfffb9e6dc5"),
			},
			amagent.StatusAvailable,

			[]amagent.Agent{
				{
					ID: uuid.FromStringOrNil("b3216dac-4fba-11ec-8551-5b4f1596d5f9"),
				},
			},
			[]*amagent.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("b3216dac-4fba-11ec-8551-5b4f1596d5f9"),
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
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().AgentV1AgentGetsByTagIDsAndStatus(ctx, tt.customer.ID, tt.tagIDs, amagent.Status(tt.status)).Return(tt.response, nil)

			res, err := h.AgentGets(ctx, tt.customer, tt.size, tt.token, tt.tagIDs, tt.status)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func TestAgentDelete(t *testing.T) {

	tests := []struct {
		name string

		customer *cscustomer.Customer
		agentID  uuid.UUID

		resAgentGet *amagent.Agent
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("852b9d5e-7ff9-11ec-9ca0-cf3c47e8c96b"),
			},
			uuid.FromStringOrNil("97508ea4-4fc0-11ec-b4fb-e7721649d9b8"),

			&amagent.Agent{
				ID:         uuid.FromStringOrNil("b3216dac-4fba-11ec-8551-5b4f1596d5f9"),
				CustomerID: uuid.FromStringOrNil("852b9d5e-7ff9-11ec-9ca0-cf3c47e8c96b"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().AgentV1AgentGet(ctx, tt.agentID).Return(tt.resAgentGet, nil)
			mockReq.EXPECT().AgentV1AgentDelete(ctx, tt.agentID).Return(&amagent.Agent{}, nil)

			_, err := h.AgentDelete(ctx, tt.customer, tt.agentID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}

func Test_AgentUpdate(t *testing.T) {

	tests := []struct {
		name string

		customer   *cscustomer.Customer
		agentID    uuid.UUID
		agentName  string
		detail     string
		ringMethod amagent.RingMethod

		resAgentGet *amagent.Agent
		resAgentPut *amagent.Agent
		expectRes   *amagent.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("852b9d5e-7ff9-11ec-9ca0-cf3c47e8c96b"),
			},
			uuid.FromStringOrNil("97508ea4-4fc0-11ec-b4fb-e7721649d9b8"),
			"test1",
			"detail",
			amagent.RingMethodRingAll,

			&amagent.Agent{
				ID:         uuid.FromStringOrNil("b3216dac-4fba-11ec-8551-5b4f1596d5f9"),
				CustomerID: uuid.FromStringOrNil("852b9d5e-7ff9-11ec-9ca0-cf3c47e8c96b"),
			},
			&amagent.Agent{
				ID:         uuid.FromStringOrNil("b3216dac-4fba-11ec-8551-5b4f1596d5f9"),
				CustomerID: uuid.FromStringOrNil("852b9d5e-7ff9-11ec-9ca0-cf3c47e8c96b"),
			},
			&amagent.WebhookMessage{
				ID:         uuid.FromStringOrNil("b3216dac-4fba-11ec-8551-5b4f1596d5f9"),
				CustomerID: uuid.FromStringOrNil("852b9d5e-7ff9-11ec-9ca0-cf3c47e8c96b"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().AgentV1AgentGet(ctx, tt.agentID).Return(tt.resAgentGet, nil)
			mockReq.EXPECT().AgentV1AgentUpdate(ctx, tt.agentID, tt.agentName, tt.detail, tt.ringMethod).Return(tt.resAgentPut, nil)

			res, err := h.AgentUpdate(ctx, tt.customer, tt.agentID, tt.agentName, tt.detail, tt.ringMethod)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func TestAgentUpdateAddresses(t *testing.T) {

	tests := []struct {
		name string

		customer  *cscustomer.Customer
		agentID   uuid.UUID
		addresses []commonaddress.Address

		resAgentGet *amagent.Agent
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("852b9d5e-7ff9-11ec-9ca0-cf3c47e8c96b"),
			},
			uuid.FromStringOrNil("97508ea4-4fc0-11ec-b4fb-e7721649d9b8"),
			[]commonaddress.Address{
				{
					Type:   commonaddress.TypeTel,
					Target: "+821021656521",
				},
			},

			&amagent.Agent{
				ID:         uuid.FromStringOrNil("b3216dac-4fba-11ec-8551-5b4f1596d5f9"),
				CustomerID: uuid.FromStringOrNil("852b9d5e-7ff9-11ec-9ca0-cf3c47e8c96b"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().AgentV1AgentGet(ctx, tt.agentID).Return(tt.resAgentGet, nil)
			mockReq.EXPECT().AgentV1AgentUpdateAddresses(ctx, tt.agentID, tt.addresses).Return(&amagent.Agent{}, nil)

			_, err := h.AgentUpdateAddresses(ctx, tt.customer, tt.agentID, tt.addresses)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestAgentUpdateTagIDs(t *testing.T) {

	tests := []struct {
		name string

		customer *cscustomer.Customer
		agentID  uuid.UUID
		tagIDs   []uuid.UUID

		resAgentGet *amagent.Agent
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("852b9d5e-7ff9-11ec-9ca0-cf3c47e8c96b"),
			},
			uuid.FromStringOrNil("97508ea4-4fc0-11ec-b4fb-e7721649d9b8"),
			[]uuid.UUID{
				uuid.FromStringOrNil("29d3e984-5065-11ec-ad4e-5765fa1c5b55"),
			},

			&amagent.Agent{
				ID:         uuid.FromStringOrNil("b3216dac-4fba-11ec-8551-5b4f1596d5f9"),
				CustomerID: uuid.FromStringOrNil("852b9d5e-7ff9-11ec-9ca0-cf3c47e8c96b"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().AgentV1AgentGet(ctx, tt.agentID).Return(tt.resAgentGet, nil)
			mockReq.EXPECT().AgentV1AgentUpdateTagIDs(ctx, tt.agentID, tt.tagIDs).Return(&amagent.Agent{}, nil)

			_, err := h.AgentUpdateTagIDs(ctx, tt.customer, tt.agentID, tt.tagIDs)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestAgentUpdateStatus(t *testing.T) {

	tests := []struct {
		name string

		customer *cscustomer.Customer
		agentID  uuid.UUID
		status   amagent.Status

		resAgentGet *amagent.Agent
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("852b9d5e-7ff9-11ec-9ca0-cf3c47e8c96b"),
			},
			uuid.FromStringOrNil("97508ea4-4fc0-11ec-b4fb-e7721649d9b8"),
			amagent.StatusAvailable,

			&amagent.Agent{
				ID:         uuid.FromStringOrNil("b3216dac-4fba-11ec-8551-5b4f1596d5f9"),
				CustomerID: uuid.FromStringOrNil("852b9d5e-7ff9-11ec-9ca0-cf3c47e8c96b"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().AgentV1AgentGet(ctx, tt.agentID).Return(tt.resAgentGet, nil)
			mockReq.EXPECT().AgentV1AgentUpdateStatus(ctx, tt.agentID, amagent.Status(tt.status)).Return(&amagent.Agent{}, nil)

			_, err := h.AgentUpdateStatus(ctx, tt.customer, tt.agentID, tt.status)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
