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

	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
)

func Test_AgentCreate(t *testing.T) {

	tests := []struct {
		name string

		agent         *amagent.Agent
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
			&amagent.Agent{
				ID:         uuid.FromStringOrNil("14003656-8e5e-11ee-b952-0ff7940c8c0e"),
				CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
				Permission: amagent.PermissionCustomerAdmin,
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
			&amagent.Agent{
				ID:         uuid.FromStringOrNil("14003656-8e5e-11ee-b952-0ff7940c8c0e"),
				CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
				Permission: amagent.PermissionCustomerAdmin,
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

			mockReq.EXPECT().AgentV1AgentCreate(ctx, 30, tt.agent.CustomerID, tt.username, tt.password, tt.agentName, tt.detail, tt.ringMethod, tt.permission, tt.tagIDs, tt.addresses).Return(tt.response, nil)

			res, err := h.AgentCreate(ctx, tt.agent, tt.username, tt.password, tt.agentName, tt.detail, tt.ringMethod, tt.permission, tt.tagIDs, tt.addresses)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(*res, *tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", *tt.expectRes, *res)
			}
		})
	}
}

func Test_AgentGet(t *testing.T) {

	tests := []struct {
		name string

		agent   *amagent.Agent
		agentID uuid.UUID

		response  *amagent.Agent
		expectRes *amagent.WebhookMessage
	}{
		{
			"normal",
			&amagent.Agent{
				ID:         uuid.FromStringOrNil("14003656-8e5e-11ee-b952-0ff7940c8c0e"),
				CustomerID: uuid.FromStringOrNil("51639bbe-8e5e-11ee-afc4-4fbef5d3d983"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("450c8f6a-5067-11ec-bda4-039a4b9a1158"),

			&amagent.Agent{
				ID:         uuid.FromStringOrNil("b3216dac-4fba-11ec-8551-5b4f1596d5f9"),
				CustomerID: uuid.FromStringOrNil("51639bbe-8e5e-11ee-afc4-4fbef5d3d983"),
			},
			&amagent.WebhookMessage{
				ID:         uuid.FromStringOrNil("b3216dac-4fba-11ec-8551-5b4f1596d5f9"),
				CustomerID: uuid.FromStringOrNil("51639bbe-8e5e-11ee-afc4-4fbef5d3d983"),
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

			res, err := h.AgentGet(ctx, tt.agent, tt.agentID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_AgentGets(t *testing.T) {

	tests := []struct {
		name string

		agent  *amagent.Agent
		size   uint64
		token  string
		tagIDs []uuid.UUID
		status amagent.Status

		response  []amagent.Agent
		expectRes []*amagent.WebhookMessage
	}{
		{
			"normal",
			&amagent.Agent{
				ID:         uuid.FromStringOrNil("14003656-8e5e-11ee-b952-0ff7940c8c0e"),
				CustomerID: uuid.FromStringOrNil("51639bbe-8e5e-11ee-afc4-4fbef5d3d983"),
				Permission: amagent.PermissionCustomerAdmin,
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
			&amagent.Agent{
				ID:         uuid.FromStringOrNil("14003656-8e5e-11ee-b952-0ff7940c8c0e"),
				CustomerID: uuid.FromStringOrNil("51639bbe-8e5e-11ee-afc4-4fbef5d3d983"),
				Permission: amagent.PermissionCustomerAdmin,
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

			mockReq.EXPECT().AgentV1AgentGets(ctx, tt.agent.CustomerID, tt.token, tt.size).Return(tt.response, nil)

			res, err := h.AgentGets(ctx, tt.agent, tt.size, tt.token, tt.tagIDs, tt.status)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_AgentGetsByTagIDs(t *testing.T) {

	tests := []struct {
		name string

		agent  *amagent.Agent
		size   uint64
		token  string
		tagIDs []uuid.UUID
		status amagent.Status

		response  []amagent.Agent
		expectRes []*amagent.WebhookMessage
	}{
		{
			"normal",
			&amagent.Agent{
				ID:         uuid.FromStringOrNil("14003656-8e5e-11ee-b952-0ff7940c8c0e"),
				CustomerID: uuid.FromStringOrNil("51639bbe-8e5e-11ee-afc4-4fbef5d3d983"),
				Permission: amagent.PermissionCustomerAdmin,
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
			&amagent.Agent{
				ID:         uuid.FromStringOrNil("14003656-8e5e-11ee-b952-0ff7940c8c0e"),
				CustomerID: uuid.FromStringOrNil("51639bbe-8e5e-11ee-afc4-4fbef5d3d983"),
				Permission: amagent.PermissionCustomerAdmin,
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

			mockReq.EXPECT().AgentV1AgentGetsByTagIDs(ctx, tt.agent.CustomerID, tt.tagIDs).Return(tt.response, nil)

			res, err := h.AgentGets(ctx, tt.agent, tt.size, tt.token, tt.tagIDs, tt.status)
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

		agent  *amagent.Agent
		size   uint64
		token  string
		tagIDs []uuid.UUID
		status amagent.Status

		response  []amagent.Agent
		expectRes []*amagent.WebhookMessage
	}{
		{
			"normal",
			&amagent.Agent{
				ID:         uuid.FromStringOrNil("14003656-8e5e-11ee-b952-0ff7940c8c0e"),
				CustomerID: uuid.FromStringOrNil("51639bbe-8e5e-11ee-afc4-4fbef5d3d983"),
				Permission: amagent.PermissionCustomerAdmin,
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
			&amagent.Agent{
				ID:         uuid.FromStringOrNil("14003656-8e5e-11ee-b952-0ff7940c8c0e"),
				CustomerID: uuid.FromStringOrNil("51639bbe-8e5e-11ee-afc4-4fbef5d3d983"),
				Permission: amagent.PermissionCustomerAdmin,
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

			mockReq.EXPECT().AgentV1AgentGetsByTagIDsAndStatus(ctx, tt.agent.CustomerID, tt.tagIDs, amagent.Status(tt.status)).Return(tt.response, nil)

			res, err := h.AgentGets(ctx, tt.agent, tt.size, tt.token, tt.tagIDs, tt.status)
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

		agent   *amagent.Agent
		agentID uuid.UUID

		resAgentGet *amagent.Agent
	}{
		{
			"normal",
			&amagent.Agent{
				ID:         uuid.FromStringOrNil("14003656-8e5e-11ee-b952-0ff7940c8c0e"),
				CustomerID: uuid.FromStringOrNil("51639bbe-8e5e-11ee-afc4-4fbef5d3d983"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("97508ea4-4fc0-11ec-b4fb-e7721649d9b8"),

			&amagent.Agent{
				ID:         uuid.FromStringOrNil("b3216dac-4fba-11ec-8551-5b4f1596d5f9"),
				CustomerID: uuid.FromStringOrNil("51639bbe-8e5e-11ee-afc4-4fbef5d3d983"),
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

			_, err := h.AgentDelete(ctx, tt.agent, tt.agentID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}

func Test_AgentUpdate(t *testing.T) {

	tests := []struct {
		name string

		agent      *amagent.Agent
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
			&amagent.Agent{
				ID:         uuid.FromStringOrNil("14003656-8e5e-11ee-b952-0ff7940c8c0e"),
				CustomerID: uuid.FromStringOrNil("51639bbe-8e5e-11ee-afc4-4fbef5d3d983"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("97508ea4-4fc0-11ec-b4fb-e7721649d9b8"),
			"test1",
			"detail",
			amagent.RingMethodRingAll,

			&amagent.Agent{
				ID:         uuid.FromStringOrNil("b3216dac-4fba-11ec-8551-5b4f1596d5f9"),
				CustomerID: uuid.FromStringOrNil("51639bbe-8e5e-11ee-afc4-4fbef5d3d983"),
			},
			&amagent.Agent{
				ID:         uuid.FromStringOrNil("b3216dac-4fba-11ec-8551-5b4f1596d5f9"),
				CustomerID: uuid.FromStringOrNil("51639bbe-8e5e-11ee-afc4-4fbef5d3d983"),
			},
			&amagent.WebhookMessage{
				ID:         uuid.FromStringOrNil("b3216dac-4fba-11ec-8551-5b4f1596d5f9"),
				CustomerID: uuid.FromStringOrNil("51639bbe-8e5e-11ee-afc4-4fbef5d3d983"),
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

			res, err := h.AgentUpdate(ctx, tt.agent, tt.agentID, tt.agentName, tt.detail, tt.ringMethod)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func Test_AgentUpdateAddresses(t *testing.T) {

	tests := []struct {
		name string

		agent     *amagent.Agent
		agentID   uuid.UUID
		addresses []commonaddress.Address

		responseAgent *amagent.Agent
		expectRes     *amagent.WebhookMessage
	}{
		{
			"normal",
			&amagent.Agent{
				ID:         uuid.FromStringOrNil("14003656-8e5e-11ee-b952-0ff7940c8c0e"),
				CustomerID: uuid.FromStringOrNil("51639bbe-8e5e-11ee-afc4-4fbef5d3d983"),
				Permission: amagent.PermissionCustomerAdmin,
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
				CustomerID: uuid.FromStringOrNil("51639bbe-8e5e-11ee-afc4-4fbef5d3d983"),
			},
			&amagent.WebhookMessage{
				ID:         uuid.FromStringOrNil("b3216dac-4fba-11ec-8551-5b4f1596d5f9"),
				CustomerID: uuid.FromStringOrNil("51639bbe-8e5e-11ee-afc4-4fbef5d3d983"),
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

			mockReq.EXPECT().AgentV1AgentGet(ctx, tt.agentID).Return(tt.responseAgent, nil)
			mockReq.EXPECT().AgentV1AgentUpdateAddresses(ctx, tt.agentID, tt.addresses).Return(tt.responseAgent, nil)

			res, err := h.AgentUpdateAddresses(ctx, tt.agent, tt.agentID, tt.addresses)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_AgentUpdateTagIDs(t *testing.T) {

	tests := []struct {
		name string

		agent   *amagent.Agent
		agentID uuid.UUID
		tagIDs  []uuid.UUID

		responseAgent *amagent.Agent
		expectRes     *amagent.WebhookMessage
	}{
		{
			"normal",
			&amagent.Agent{
				ID:         uuid.FromStringOrNil("14003656-8e5e-11ee-b952-0ff7940c8c0e"),
				CustomerID: uuid.FromStringOrNil("51639bbe-8e5e-11ee-afc4-4fbef5d3d983"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("97508ea4-4fc0-11ec-b4fb-e7721649d9b8"),
			[]uuid.UUID{
				uuid.FromStringOrNil("29d3e984-5065-11ec-ad4e-5765fa1c5b55"),
			},

			&amagent.Agent{
				ID:         uuid.FromStringOrNil("b3216dac-4fba-11ec-8551-5b4f1596d5f9"),
				CustomerID: uuid.FromStringOrNil("51639bbe-8e5e-11ee-afc4-4fbef5d3d983"),
			},
			&amagent.WebhookMessage{
				ID:         uuid.FromStringOrNil("b3216dac-4fba-11ec-8551-5b4f1596d5f9"),
				CustomerID: uuid.FromStringOrNil("51639bbe-8e5e-11ee-afc4-4fbef5d3d983"),
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

			mockReq.EXPECT().AgentV1AgentGet(ctx, tt.agentID).Return(tt.responseAgent, nil)
			mockReq.EXPECT().AgentV1AgentUpdateTagIDs(ctx, tt.agentID, tt.tagIDs).Return(tt.responseAgent, nil)

			res, err := h.AgentUpdateTagIDs(ctx, tt.agent, tt.agentID, tt.tagIDs)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_AgentUpdateStatus(t *testing.T) {

	tests := []struct {
		name string

		agent   *amagent.Agent
		agentID uuid.UUID
		status  amagent.Status

		responseAgent *amagent.Agent
		expectRes     *amagent.WebhookMessage
	}{
		{
			"normal",
			&amagent.Agent{
				ID:         uuid.FromStringOrNil("14003656-8e5e-11ee-b952-0ff7940c8c0e"),
				CustomerID: uuid.FromStringOrNil("51639bbe-8e5e-11ee-afc4-4fbef5d3d983"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("97508ea4-4fc0-11ec-b4fb-e7721649d9b8"),
			amagent.StatusAvailable,

			&amagent.Agent{
				ID:         uuid.FromStringOrNil("b3216dac-4fba-11ec-8551-5b4f1596d5f9"),
				CustomerID: uuid.FromStringOrNil("51639bbe-8e5e-11ee-afc4-4fbef5d3d983"),
			},
			&amagent.WebhookMessage{
				ID:         uuid.FromStringOrNil("b3216dac-4fba-11ec-8551-5b4f1596d5f9"),
				CustomerID: uuid.FromStringOrNil("51639bbe-8e5e-11ee-afc4-4fbef5d3d983"),
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

			mockReq.EXPECT().AgentV1AgentGet(ctx, tt.agentID).Return(tt.responseAgent, nil)
			mockReq.EXPECT().AgentV1AgentUpdateStatus(ctx, tt.agentID, amagent.Status(tt.status)).Return(tt.responseAgent, nil)

			res, err := h.AgentUpdateStatus(ctx, tt.agent, tt.agentID, tt.status)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
