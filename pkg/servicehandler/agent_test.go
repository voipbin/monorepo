package servicehandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"

	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
)

func TestAgentCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	h := serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

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
		permission    uint64
		tagIDs        []uuid.UUID
		addresses     []cmaddress.Address

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
			[]cmaddress.Address{},

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
			[]cmaddress.Address{},

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

			mockReq.EXPECT().AMV1AgentCreate(gomock.Any(), 30, tt.customer.ID, tt.username, tt.password, tt.agentName, tt.detail, tt.webhookMethod, tt.webhookURI, amagent.RingMethod(tt.ringMethod), amagent.Permission(tt.permission), tt.tagIDs, tt.addresses).Return(tt.response, nil)

			res, err := h.AgentCreate(tt.customer, tt.username, tt.password, tt.agentName, tt.detail, tt.webhookMethod, tt.webhookURI, tt.ringMethod, tt.permission, tt.tagIDs, tt.addresses)
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
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	h := serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

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
				ID: uuid.FromStringOrNil("b3216dac-4fba-11ec-8551-5b4f1596d5f9"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockReq.EXPECT().AMV1AgentGet(gomock.Any(), tt.agentID).Return(tt.response, nil)

			res, err := h.AgentGet(tt.customer, tt.agentID)
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
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	h := serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

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

			mockReq.EXPECT().AMV1AgentGets(gomock.Any(), tt.customer.ID, tt.token, tt.size).Return(tt.response, nil)

			res, err := h.AgentGets(tt.customer, tt.size, tt.token, tt.tagIDs, tt.status)
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
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	h := serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

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

			mockReq.EXPECT().AMV1AgentGetsByTagIDs(gomock.Any(), tt.customer.ID, tt.tagIDs).Return(tt.response, nil)

			res, err := h.AgentGets(tt.customer, tt.size, tt.token, tt.tagIDs, tt.status)
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
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	h := serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

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

			mockReq.EXPECT().AMV1AgentGetsByTagIDsAndStatus(gomock.Any(), tt.customer.ID, tt.tagIDs, amagent.Status(tt.status)).Return(tt.response, nil)

			res, err := h.AgentGets(tt.customer, tt.size, tt.token, tt.tagIDs, tt.status)
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
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	h := serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

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

			mockReq.EXPECT().AMV1AgentGet(gomock.Any(), tt.agentID).Return(tt.resAgentGet, nil)
			mockReq.EXPECT().AMV1AgentDelete(gomock.Any(), tt.agentID).Return(nil)

			if err := h.AgentDelete(tt.customer, tt.agentID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}

func TestAgentLogin(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	h := serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

	tests := []struct {
		name string

		customer *cscustomer.Customer
		username string
		password string

		response *amagent.Agent
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("852b9d5e-7ff9-11ec-9ca0-cf3c47e8c96b"),
			},
			"test1",
			"password1",

			&amagent.Agent{
				ID:         uuid.FromStringOrNil("b3216dac-4fba-11ec-8551-5b4f1596d5f9"),
				CustomerID: uuid.FromStringOrNil("852b9d5e-7ff9-11ec-9ca0-cf3c47e8c96b"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockReq.EXPECT().AMV1AgentLogin(gomock.Any(), 30000, tt.customer.ID, tt.username, tt.password).Return(tt.response, nil)

			_, err := h.AgentLogin(tt.customer.ID, tt.username, tt.password)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestAgentUpdate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	h := serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

	tests := []struct {
		name string

		customer   *cscustomer.Customer
		agentID    uuid.UUID
		agentName  string
		detail     string
		ringMethod amagent.RingMethod

		resAgentGet *amagent.Agent
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
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockReq.EXPECT().AMV1AgentGet(gomock.Any(), tt.agentID).Return(tt.resAgentGet, nil)
			mockReq.EXPECT().AMV1AgentUpdate(gomock.Any(), tt.agentID, tt.agentName, tt.detail, string(tt.ringMethod)).Return(nil)

			if err := h.AgentUpdate(tt.customer, tt.agentID, tt.agentName, tt.detail, tt.ringMethod); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestAgentUpdateAddresses(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	h := serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

	tests := []struct {
		name string

		customer  *cscustomer.Customer
		agentID   uuid.UUID
		addresses []cmaddress.Address

		resAgentGet *amagent.Agent
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("852b9d5e-7ff9-11ec-9ca0-cf3c47e8c96b"),
			},
			uuid.FromStringOrNil("97508ea4-4fc0-11ec-b4fb-e7721649d9b8"),
			[]cmaddress.Address{
				{
					Type:   cmaddress.TypeTel,
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

			mockReq.EXPECT().AMV1AgentGet(gomock.Any(), tt.agentID).Return(tt.resAgentGet, nil)
			mockReq.EXPECT().AMV1AgentUpdateAddresses(gomock.Any(), tt.agentID, tt.addresses).Return(nil)

			if err := h.AgentUpdateAddresses(tt.customer, tt.agentID, tt.addresses); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestAgentUpdateTagIDs(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	h := serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

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
			mockReq.EXPECT().AMV1AgentGet(gomock.Any(), tt.agentID).Return(tt.resAgentGet, nil)
			mockReq.EXPECT().AMV1AgentUpdateTagIDs(gomock.Any(), tt.agentID, tt.tagIDs).Return(nil)

			if err := h.AgentUpdateTagIDs(tt.customer, tt.agentID, tt.tagIDs); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestAgentUpdateStatus(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	h := serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

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

			mockReq.EXPECT().AMV1AgentGet(gomock.Any(), tt.agentID).Return(tt.resAgentGet, nil)
			mockReq.EXPECT().AMV1AgentUpdateStatus(gomock.Any(), tt.agentID, amagent.Status(tt.status)).Return(nil)

			if err := h.AgentUpdateStatus(tt.customer, tt.agentID, tt.status); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
