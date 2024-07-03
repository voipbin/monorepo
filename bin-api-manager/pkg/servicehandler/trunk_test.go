package servicehandler

import (
	"context"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"

	rmsipauth "monorepo/bin-registrar-manager/models/sipauth"
	rmtrunk "monorepo/bin-registrar-manager/models/trunk"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"monorepo/bin-api-manager/pkg/dbhandler"
)

func Test_TrunkCreate(t *testing.T) {

	type test struct {
		name  string
		agent *amagent.Agent

		trunkName  string
		detail     string
		domainName string
		authTypes  []rmsipauth.AuthType
		username   string
		password   string
		allowedIPs []string

		response  *rmtrunk.Trunk
		expectRes *rmtrunk.WebhookMessage
	}

	tests := []test{
		{
			name: "normal",
			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},

			trunkName:  "test name",
			detail:     "test detail",
			domainName: "test-domain",
			authTypes:  []rmsipauth.AuthType{rmsipauth.AuthTypeBasic, rmsipauth.AuthTypeIP},
			username:   "testusername",
			password:   "testpassword",
			allowedIPs: []string{"1.2.3.4"},

			response: &rmtrunk.Trunk{
				ID: uuid.FromStringOrNil("bc669058-54a5-11ee-999b-a3c9289707a5"),
			},

			expectRes: &rmtrunk.WebhookMessage{
				ID: uuid.FromStringOrNil("bc669058-54a5-11ee-999b-a3c9289707a5"),
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

			mockReq.EXPECT().RegistrarV1TrunkCreate(ctx, tt.agent.CustomerID, tt.trunkName, tt.detail, tt.domainName, tt.authTypes, tt.username, tt.password, tt.allowedIPs).Return(tt.response, nil)

			res, err := h.TrunkCreate(ctx, tt.agent, tt.trunkName, tt.detail, tt.domainName, tt.authTypes, tt.username, tt.password, tt.allowedIPs)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_TrunkDelete(t *testing.T) {

	type test struct {
		name    string
		agent   *amagent.Agent
		trunkID uuid.UUID

		response  *rmtrunk.Trunk
		expectRes *rmtrunk.WebhookMessage
	}

	tests := []test{
		{
			"normal",
			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("02369970-54a6-11ee-8436-576a6c6712f2"),

			&rmtrunk.Trunk{
				ID:         uuid.FromStringOrNil("02369970-54a6-11ee-8436-576a6c6712f2"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
			},
			&rmtrunk.WebhookMessage{
				ID:         uuid.FromStringOrNil("02369970-54a6-11ee-8436-576a6c6712f2"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
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

			mockReq.EXPECT().RegistrarV1TrunkGet(ctx, tt.trunkID).Return(tt.response, nil)
			mockReq.EXPECT().RegistrarV1TrunkDelete(ctx, tt.trunkID).Return(tt.response, nil)

			res, err := h.TrunkDelete(ctx, tt.agent, tt.trunkID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_TrunkGet(t *testing.T) {

	type test struct {
		name    string
		agent   *amagent.Agent
		trunkID uuid.UUID

		response  *rmtrunk.Trunk
		expectRes *rmtrunk.WebhookMessage
	}

	tests := []test{
		{
			"normal",
			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("32309590-54a6-11ee-8466-9b257305288b"),

			&rmtrunk.Trunk{
				ID:         uuid.FromStringOrNil("32309590-54a6-11ee-8466-9b257305288b"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
			},
			&rmtrunk.WebhookMessage{
				ID:         uuid.FromStringOrNil("32309590-54a6-11ee-8466-9b257305288b"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
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

			mockReq.EXPECT().RegistrarV1TrunkGet(ctx, tt.trunkID).Return(tt.response, nil)

			res, err := h.TrunkGet(ctx, tt.agent, tt.trunkID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_TrunkGets(t *testing.T) {

	type test struct {
		name      string
		agent     *amagent.Agent
		pageToken string
		pageSize  uint64

		response []rmtrunk.Trunk

		expectFilters map[string]string
		expectRes     []*rmtrunk.WebhookMessage
	}

	tests := []test{
		{
			"normal",
			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			"2020-10-20T01:00:00.995000",
			10,

			[]rmtrunk.Trunk{
				{
					ID: uuid.FromStringOrNil("6ac312f2-54a6-11ee-9e12-3bb0c25cd1e2"),
				},
				{
					ID: uuid.FromStringOrNil("6af5fe56-54a6-11ee-8db0-b750622f6cc0"),
				},
			},

			map[string]string{
				"customer_id": "5f621078-8e5f-11ee-97b2-cfe7337b701c",
				"deleted":     "false",
			},
			[]*rmtrunk.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("6ac312f2-54a6-11ee-9e12-3bb0c25cd1e2"),
				},
				{
					ID: uuid.FromStringOrNil("6af5fe56-54a6-11ee-8db0-b750622f6cc0"),
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

			mockReq.EXPECT().RegistrarV1TrunkGets(ctx, tt.pageToken, tt.pageSize, tt.expectFilters).Return(tt.response, nil)

			res, err := h.TrunkGets(ctx, tt.agent, tt.pageSize, tt.pageToken)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_TrunkUpdateBasicInfo(t *testing.T) {

	type test struct {
		name  string
		agent *amagent.Agent

		id         uuid.UUID
		trunkName  string
		detail     string
		authTypes  []rmsipauth.AuthType
		username   string
		password   string
		allowedIPs []string

		responseTrunk *rmtrunk.Trunk
		expectRes     *rmtrunk.WebhookMessage
	}

	tests := []test{
		{
			"normal",
			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},

			uuid.FromStringOrNil("bdf14732-54a6-11ee-b262-1f9dcbae10b2"),
			"update name",
			"update detail",
			[]rmsipauth.AuthType{rmsipauth.AuthTypeBasic},
			"updateusername",
			"updatepassword",
			[]string{"1.2.3.4"},

			&rmtrunk.Trunk{
				ID:         uuid.FromStringOrNil("bdf14732-54a6-11ee-b262-1f9dcbae10b2"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
			},
			&rmtrunk.WebhookMessage{
				ID:         uuid.FromStringOrNil("bdf14732-54a6-11ee-b262-1f9dcbae10b2"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
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

			mockReq.EXPECT().RegistrarV1TrunkGet(ctx, tt.id).Return(tt.responseTrunk, nil)
			mockReq.EXPECT().RegistrarV1TrunkUpdateBasicInfo(ctx, tt.id, tt.trunkName, tt.detail, tt.authTypes, tt.username, tt.password, tt.allowedIPs).Return(tt.responseTrunk, nil)
			res, err := h.TrunkUpdateBasicInfo(ctx, tt.agent, tt.id, tt.trunkName, tt.detail, tt.authTypes, tt.username, tt.password, tt.allowedIPs)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}
