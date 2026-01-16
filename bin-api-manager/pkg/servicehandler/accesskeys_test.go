package servicehandler

import (
	"context"
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/pkg/dbhandler"
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	csaccesskey "monorepo/bin-customer-manager/models/accesskey"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_accesskeyGet(t *testing.T) {

	tests := []struct {
		name string

		agent       *amagent.Agent
		accesskeyID uuid.UUID

		responseAccesskey *csaccesskey.Accesskey
	}{
		{
			name: "normal",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("f1d53156-8dec-11ee-98a0-6ba69fe98bd2"),
					CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			accesskeyID: uuid.FromStringOrNil("9c1078ba-ab47-11ef-b8b7-27bf39014b86"),

			responseAccesskey: &csaccesskey.Accesskey{
				ID:         uuid.FromStringOrNil("9c1078ba-ab47-11ef-b8b7-27bf39014b86"),
				CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
				TMDelete:   defaultTimestamp,
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

			mockReq.EXPECT().CustomerV1AccesskeyGet(ctx, tt.accesskeyID).Return(tt.responseAccesskey, nil)

			res, err := h.accesskeyGet(ctx, tt.agent, tt.accesskeyID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseAccesskey) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.responseAccesskey, res)
			}
		})
	}
}

func Test_AccesskeyCreate(t *testing.T) {

	tests := []struct {
		name string

		agent         *amagent.Agent
		accesskeyName string
		detail        string
		expire        int32

		responseAccesskey *csaccesskey.Accesskey

		expectRes *csaccesskey.WebhookMessage
	}{
		{
			name: "normal",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			accesskeyName: "test name",
			detail:        "test detail",
			expire:        86400000,

			responseAccesskey: &csaccesskey.Accesskey{
				ID: uuid.FromStringOrNil("1fac8b00-ab48-11ef-a8de-87e41c6e4a91"),
			},

			expectRes: &csaccesskey.WebhookMessage{
				ID: uuid.FromStringOrNil("1fac8b00-ab48-11ef-a8de-87e41c6e4a91"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := serviceHandler{
				reqHandler:  mockReq,
				dbHandler:   mockDB,
				utilHandler: mockUtil,
			}
			ctx := context.Background()

			mockReq.EXPECT().CustomerV1AccesskeyCreate(ctx, tt.agent.CustomerID, tt.accesskeyName, tt.detail, tt.expire).Return(tt.responseAccesskey, nil)

			res, err := h.AccesskeyCreate(ctx, tt.agent, tt.accesskeyName, tt.detail, tt.expire)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_AccesskeyGet(t *testing.T) {

	tests := []struct {
		name string

		agent       *amagent.Agent
		accesskeyID uuid.UUID

		responseAccesskey *csaccesskey.Accesskey

		expectedRes *csaccesskey.WebhookMessage
	}{
		{
			name: "normal",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			accesskeyID: uuid.FromStringOrNil("589ebbc2-ab48-11ef-a7b6-0be2f7042cdf"),

			responseAccesskey: &csaccesskey.Accesskey{
				ID:         uuid.FromStringOrNil("589ebbc2-ab48-11ef-a7b6-0be2f7042cdf"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				TMDelete:   defaultTimestamp,
			},

			expectedRes: &csaccesskey.WebhookMessage{
				ID:         uuid.FromStringOrNil("589ebbc2-ab48-11ef-a7b6-0be2f7042cdf"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				TMDelete:   defaultTimestamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := serviceHandler{
				reqHandler:  mockReq,
				dbHandler:   mockDB,
				utilHandler: mockUtil,
			}
			ctx := context.Background()

			mockReq.EXPECT().CustomerV1AccesskeyGet(ctx, tt.accesskeyID).Return(tt.responseAccesskey, nil)

			res, err := h.AccesskeyGet(ctx, tt.agent, tt.accesskeyID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectedRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}

func Test_AccesskeyGets(t *testing.T) {

	tests := []struct {
		name string

		agent   *amagent.Agent
		size    uint64
		token   string
		filters map[csaccesskey.Field]any

		response  []csaccesskey.Accesskey
		expectRes []*csaccesskey.WebhookMessage
	}{
		{
			"normal",
			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("14003656-8e5e-11ee-b952-0ff7940c8c0e"),
					CustomerID: uuid.FromStringOrNil("51639bbe-8e5e-11ee-afc4-4fbef5d3d983"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			10,
			"2020-09-20 03:23:20.995000",
			map[csaccesskey.Field]any{
				csaccesskey.FieldCustomerID: uuid.FromStringOrNil("51639bbe-8e5e-11ee-afc4-4fbef5d3d983"),
				csaccesskey.FieldDeleted:    false,
			},

			[]csaccesskey.Accesskey{
				{
					ID: uuid.FromStringOrNil("a9f87db4-ab48-11ef-8226-cf280dfa7a21"),
				},
			},
			[]*csaccesskey.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("a9f87db4-ab48-11ef-8226-cf280dfa7a21"),
				},
			},
		},
		{
			"2 results",
			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("14003656-8e5e-11ee-b952-0ff7940c8c0e"),
					CustomerID: uuid.FromStringOrNil("51639bbe-8e5e-11ee-afc4-4fbef5d3d983"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			10,
			"2020-09-20 03:23:20.995000",
			map[csaccesskey.Field]any{
				csaccesskey.FieldCustomerID: uuid.FromStringOrNil("51639bbe-8e5e-11ee-afc4-4fbef5d3d983"),
				csaccesskey.FieldDeleted:    false,
			},

			[]csaccesskey.Accesskey{
				{
					ID: uuid.FromStringOrNil("aa6b2df0-ab48-11ef-a7a2-c7a1b7becbe2"),
				},
				{
					ID: uuid.FromStringOrNil("aaa34ca8-ab48-11ef-a801-ef5ec8c35742"),
				},
			},
			[]*csaccesskey.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("aa6b2df0-ab48-11ef-a7a2-c7a1b7becbe2"),
				},
				{
					ID: uuid.FromStringOrNil("aaa34ca8-ab48-11ef-a801-ef5ec8c35742"),
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

			mockReq.EXPECT().CustomerV1AccesskeyList(ctx, tt.token, tt.size, tt.filters).Return(tt.response, nil)

			res, err := h.AccesskeyGets(ctx, tt.agent, tt.size, tt.token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_AccesskeyDelete(t *testing.T) {

	tests := []struct {
		name string

		agent         *amagent.Agent
		accesskeyID   uuid.UUID
		accesskeyName string
		detail        string

		responseAccesskey *csaccesskey.Accesskey
	}{
		{
			name: "normal",
			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("14003656-8e5e-11ee-b952-0ff7940c8c0e"),
					CustomerID: uuid.FromStringOrNil("51639bbe-8e5e-11ee-afc4-4fbef5d3d983"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			accesskeyID: uuid.FromStringOrNil("bde73c60-ab49-11ef-810d-7b7404934537"),

			responseAccesskey: &csaccesskey.Accesskey{
				ID:         uuid.FromStringOrNil("bde73c60-ab49-11ef-810d-7b7404934537"),
				CustomerID: uuid.FromStringOrNil("51639bbe-8e5e-11ee-afc4-4fbef5d3d983"),
				TMDelete:   defaultTimestamp,
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

			mockReq.EXPECT().CustomerV1AccesskeyGet(ctx, tt.accesskeyID).Return(tt.responseAccesskey, nil)
			mockReq.EXPECT().CustomerV1AccesskeyDelete(ctx, tt.accesskeyID).Return(tt.responseAccesskey, nil)

			_, err := h.AccesskeyDelete(ctx, tt.agent, tt.accesskeyID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}

func Test_AccesskeyUpdate(t *testing.T) {

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
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("14003656-8e5e-11ee-b952-0ff7940c8c0e"),
					CustomerID: uuid.FromStringOrNil("51639bbe-8e5e-11ee-afc4-4fbef5d3d983"),
				},
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
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b3216dac-4fba-11ec-8551-5b4f1596d5f9"),
					CustomerID: uuid.FromStringOrNil("51639bbe-8e5e-11ee-afc4-4fbef5d3d983"),
				},
			},
			&amagent.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b3216dac-4fba-11ec-8551-5b4f1596d5f9"),
					CustomerID: uuid.FromStringOrNil("51639bbe-8e5e-11ee-afc4-4fbef5d3d983"),
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
