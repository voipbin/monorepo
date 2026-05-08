package servicehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	bmaccount "monorepo/bin-billing-manager/models/account"
	cmoutboundconfig "monorepo/bin-call-manager/models/outboundconfig"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	cscustomer "monorepo/bin-customer-manager/models/customer"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/pkg/dbhandler"
)

func Test_CustomerCreate(t *testing.T) {

	type test struct {
		name string

		agent         *auth.AuthIdentity
		customerName  string
		detail        string
		email         string
		phoneNumber   string
		address       string
		webhookMethod cscustomer.WebhookMethod
		webhookURI    string

		responseCustomer       *cscustomer.Customer
		responseAgent          *amagent.Agent
		responseBillingAccount *bmaccount.Account

		expectFilters map[cscustomer.Field]any
		expectRes     *cscustomer.Customer
	}

	tests := []test{
		{
			name: "normal",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionProjectSuperAdmin,
			}),
			customerName:  "test",
			detail:        "test detail",
			email:         "test@test.com",
			phoneNumber:   "+821100000001",
			address:       "somewhere",
			webhookMethod: cscustomer.WebhookMethodPost,
			webhookURI:    "test.com",

			responseCustomer: &cscustomer.Customer{
				ID: uuid.FromStringOrNil("ade4707c-837d-11ec-a600-f30a3ccf56ae"),
			},
			responseAgent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("0dae6fb4-cbaf-11ee-a779-4b088feeb56e"),
				},
			},
			responseBillingAccount: &bmaccount.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("504e0afa-cbaf-11ee-8184-cfce50394145"),
				},
			},

			expectFilters: map[cscustomer.Field]any{
				"deleted":  "false",
				"username": "test@test.com",
			},
			expectRes: &cscustomer.Customer{
				ID: uuid.FromStringOrNil("ade4707c-837d-11ec-a600-f30a3ccf56ae"),
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

			mockReq.EXPECT().CustomerV1CustomerCreate(ctx, 30000, tt.customerName, tt.detail, tt.email, tt.phoneNumber, tt.address, tt.webhookMethod, tt.webhookURI).Return(tt.responseCustomer, nil)
			mockReq.EXPECT().CallV1OutboundConfigCreate(ctx, tt.responseCustomer.ID, &cmoutboundconfig.UpdateRequest{}).Return(&cmoutboundconfig.OutboundConfig{}, nil)

			res, err := h.CustomerCreate(ctx, tt.agent, tt.customerName, tt.detail, tt.email, tt.phoneNumber, tt.address, tt.webhookMethod, tt.webhookURI)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

// Test_CustomerCreate_AutoOutboundConfigFailureRollsBack verifies that when
// OutboundConfig auto-create fails permanently after the bounded retry budget,
// CustomerCreate rolls back the just-created customer (CustomerV1CustomerDelete)
// and returns an error. This guarantees we never leave a customer that cannot
// make outgoing PSTN calls because OutboundConfig creation never succeeded.
func Test_CustomerCreate_AutoOutboundConfigFailureRollsBack(t *testing.T) {
	customerID := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	createdCustomer := &cscustomer.Customer{
		ID: customerID,
	}

	tests := []struct {
		name string
	}{
		{name: "OutboundConfig create fails persistently → customer rolled back, error returned"},
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

			agent := auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionProjectSuperAdmin,
			})

			mockReq.EXPECT().CustomerV1CustomerCreate(
				ctx, 30000, "n", "d", "e@x.y", "+12025550100", "addr", cscustomer.WebhookMethod("POST"), "https://x.y",
			).Return(createdCustomer, nil)
			// Three OutboundConfig attempts, all fail with the same error.
			mockReq.EXPECT().CallV1OutboundConfigCreate(ctx, customerID, &cmoutboundconfig.UpdateRequest{}).Return(nil, fmt.Errorf("persistent")).Times(3)
			// After each failure, the helper checks for an existing OutboundConfig
			// (idempotency guard for "INSERT succeeded but response lost"). All three
			// list calls return empty, confirming no record exists.
			mockReq.EXPECT().CallV1OutboundConfigList(ctx, customerID, uint64(1), "").Return([]cmoutboundconfig.OutboundConfig{}, nil).Times(3)
			// Rollback delete is called after all retries are exhausted.
			mockReq.EXPECT().CustomerV1CustomerDelete(ctx, customerID).Return(createdCustomer, nil)

			res, err := h.CustomerCreate(ctx, agent, "n", "d", "e@x.y", "+12025550100", "addr", cscustomer.WebhookMethod("POST"), "https://x.y")
			if err == nil || !strings.Contains(err.Error(), "could not create OutboundConfig") {
				t.Errorf("expected OutboundConfig error, got %v", err)
			}
			if res != nil {
				t.Errorf("Wrong match. expect: nil result on rollback, got: %v", res)
			}
		})
	}
}

// Test_CustomerSignup_AutoOutboundConfigFailureRollsBack verifies that when
// OutboundConfig auto-create fails permanently after the bounded retry budget,
// CustomerSignup rolls back the just-created customer (CustomerV1CustomerDelete)
// and returns an error. This guarantees that even self-service signup does not
// leave a customer that cannot make outgoing PSTN calls because OutboundConfig
// creation never succeeded.
func Test_CustomerSignup_AutoOutboundConfigFailureRollsBack(t *testing.T) {
	customerID := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")
	signupResult := &cscustomer.SignupResult{
		Customer: &cscustomer.Customer{
			ID: customerID,
		},
	}

	tests := []struct {
		name string
	}{
		{name: "OutboundConfig create fails persistently → customer rolled back, error returned"},
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

			mockReq.EXPECT().CustomerV1CustomerSignup(
				ctx, "n", "d", "e@x.y", "+12025550100", "addr", cscustomer.WebhookMethod("POST"), "https://x.y", "192.168.1.1",
			).Return(signupResult, nil)
			// Three OutboundConfig attempts, all fail with the same error.
			mockReq.EXPECT().CallV1OutboundConfigCreate(ctx, customerID, &cmoutboundconfig.UpdateRequest{}).Return(nil, fmt.Errorf("persistent")).Times(3)
			// After each failure, the helper checks for an existing OutboundConfig
			// (idempotency guard for "INSERT succeeded but response lost"). All three
			// list calls return empty, confirming no record exists.
			mockReq.EXPECT().CallV1OutboundConfigList(ctx, customerID, uint64(1), "").Return([]cmoutboundconfig.OutboundConfig{}, nil).Times(3)
			// Rollback delete is called after all retries are exhausted.
			mockReq.EXPECT().CustomerV1CustomerDelete(ctx, customerID).Return(signupResult.Customer, nil)

			res, err := h.CustomerSignup(ctx, "n", "d", "e@x.y", "+12025550100", "addr", cscustomer.WebhookMethod("POST"), "https://x.y", "192.168.1.1")
			if err == nil || !strings.Contains(err.Error(), "could not create OutboundConfig") {
				t.Errorf("expected OutboundConfig error, got %v", err)
			}
			if res != nil {
				t.Errorf("Wrong match. expect: nil result on rollback, got: %v", res)
			}
		})
	}
}

// Test_CustomerCreate_AutoOutboundConfigIdempotentRecovery covers the scenario
// where the first OutboundConfig INSERT actually succeeded on the server, but
// the response was lost (network blip, timeout). On the retry, the UNIQUE KEY
// uq_customer_id violation surfaces as an error from CallV1OutboundConfigCreate.
//
// Without the idempotency guard, the retry budget would exhaust and the
// healthy customer would be incorrectly rolled back via CustomerV1CustomerDelete.
//
// With the guard, after the failed CREATE we LIST the customer's OutboundConfigs
// and find the one created by the lost-ACK attempt. The retry helper returns
// nil (success) and the customer is preserved.
func Test_CustomerCreate_AutoOutboundConfigIdempotentRecovery(t *testing.T) {
	customerID := uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333")
	createdCustomer := &cscustomer.Customer{
		ID: customerID,
	}

	tests := []struct {
		name string
	}{
		{name: "CREATE returns error but List shows existing config → treat as success, no rollback"},
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

			agent := auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionProjectSuperAdmin,
			})

			mockReq.EXPECT().CustomerV1CustomerCreate(
				ctx, 30000, "n", "d", "e@x.y", "+12025550100", "addr", cscustomer.WebhookMethod("POST"), "https://x.y",
			).Return(createdCustomer, nil)
			// Attempt 1: CREATE returns an error (e.g., uq_customer_id violation
			// because the prior attempt's INSERT actually succeeded).
			mockReq.EXPECT().CallV1OutboundConfigCreate(ctx, customerID, &cmoutboundconfig.UpdateRequest{}).Return(nil, fmt.Errorf("duplicate key")).Times(1)
			// Idempotency check: existing config is found, helper returns success.
			mockReq.EXPECT().CallV1OutboundConfigList(ctx, customerID, uint64(1), "").Return([]cmoutboundconfig.OutboundConfig{
				{ID: uuid.FromStringOrNil("44444444-4444-4444-4444-444444444444"), CustomerID: customerID},
			}, nil).Times(1)
			// Crucially: CustomerV1CustomerDelete must NOT be called — the customer is healthy.

			res, err := h.CustomerCreate(ctx, agent, "n", "d", "e@x.y", "+12025550100", "addr", cscustomer.WebhookMethod("POST"), "https://x.y")
			if err != nil {
				t.Errorf("Wrong match. expect: ok (idempotent success), got: %v", err)
			}
			if res == nil || res.ID != customerID {
				t.Errorf("Wrong match. expect: customer preserved, got: %v", res)
			}
		})
	}
}

func TestCustomerGet(t *testing.T) {

	type test struct {
		name string

		agent *auth.AuthIdentity
		id    uuid.UUID

		responseCustomer *cscustomer.Customer
		expectRes        *cscustomer.Customer
	}

	tests := []test{
		{
			"normal",

			auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				},
				Permission: amagent.PermissionProjectSuperAdmin,
			}),
			uuid.FromStringOrNil("a0f4b592-837e-11ec-9f5f-2f2051d4adac"),

			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("a0f4b592-837e-11ec-9f5f-2f2051d4adac"),
			},
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("a0f4b592-837e-11ec-9f5f-2f2051d4adac"),
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

			mockReq.EXPECT().CustomerV1CustomerGet(ctx, tt.id).Return(tt.responseCustomer, nil)

			res, err := h.CustomerGet(ctx, tt.agent, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_CustomerSelfGet(t *testing.T) {
	tests := []struct {
		name string

		agent *auth.AuthIdentity

		responseCustomer *cscustomer.Customer
		expectRes        *cscustomer.WebhookMessage
	}{
		{
			name: "normal",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("a0f4b592-837e-11ec-9f5f-2f2051d4adac"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),

			responseCustomer: &cscustomer.Customer{
				ID: uuid.FromStringOrNil("a0f4b592-837e-11ec-9f5f-2f2051d4adac"),
			},
			expectRes: &cscustomer.WebhookMessage{
				ID: uuid.FromStringOrNil("a0f4b592-837e-11ec-9f5f-2f2051d4adac"),
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

			mockReq.EXPECT().CustomerV1CustomerGet(ctx, tt.agent.CustomerID).Return(tt.responseCustomer, nil)

			res, err := h.CustomerSelfGet(ctx, tt.agent)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_CustomerList(t *testing.T) {

	type test struct {
		name string

		agent         *auth.AuthIdentity
		size          uint64
		token         string
		filters       map[string]string
		expectFilters map[cscustomer.Field]any

		responseCustomers []cscustomer.Customer
		expectRes         []*cscustomer.Customer
	}

	tests := []test{
		{
			"normal",

			auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("3e6fe9c8-837e-11ec-84ef-b762e8a7a8fc"),
				},
				Permission: amagent.PermissionProjectSuperAdmin,
			}),

			10,
			"2020-09-20T03:23:20.995000Z",
			map[string]string{
				"deleted": "false",
			},
			map[cscustomer.Field]any{
				cscustomer.FieldDeleted: false,
			},

			[]cscustomer.Customer{
				{
					ID: uuid.FromStringOrNil("8ffa19a2-837f-11ec-b57e-9f3906006c0a"),
				},
			},
			[]*cscustomer.Customer{
				{
					ID: uuid.FromStringOrNil("8ffa19a2-837f-11ec-b57e-9f3906006c0a"),
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

			mockReq.EXPECT().CustomerV1CustomerList(ctx, tt.token, tt.size, tt.expectFilters).Return(tt.responseCustomers, nil)

			res, err := h.CustomerList(ctx, tt.agent, tt.size, tt.token, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_CustomerUpdate(t *testing.T) {

	type test struct {
		name  string
		agent *auth.AuthIdentity

		id            uuid.UUID
		customerName  string
		detail        string
		email         string
		phoneNumber   string
		address       string
		webhookMethod cscustomer.WebhookMethod
		webhookURI    string

		responseCustomers *cscustomer.Customer
		expectRes         *cscustomer.Customer
	}

	tests := []test{
		{
			"normal",

			auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				},
				Permission: amagent.PermissionProjectSuperAdmin,
			}),
			uuid.FromStringOrNil("8ffa19a2-837f-11ec-b57e-9f3906006c0a"),
			"name new",
			"detail new",
			"test@test.com",
			"+821100000001",
			"somewhere",
			cscustomer.WebhookMethodPost,
			"test.com",

			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("8ffa19a2-837f-11ec-b57e-9f3906006c0a"),
			},
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("8ffa19a2-837f-11ec-b57e-9f3906006c0a"),
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

			mockReq.EXPECT().CustomerV1CustomerGet(ctx, tt.id).Return(tt.responseCustomers, nil)
			mockReq.EXPECT().CustomerV1CustomerUpdate(ctx, tt.id, tt.customerName, tt.detail, tt.email, tt.phoneNumber, tt.address, tt.webhookMethod, tt.webhookURI).Return(tt.responseCustomers, nil)

			res, err := h.CustomerUpdate(ctx, tt.agent, tt.id, tt.customerName, tt.detail, tt.email, tt.phoneNumber, tt.address, tt.webhookMethod, tt.webhookURI)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_CustomerSelfUpdate(t *testing.T) {
	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		customerName  string
		detail        string
		email         string
		phoneNumber   string
		address       string
		webhookMethod cscustomer.WebhookMethod
		webhookURI    string

		responseCustomer *cscustomer.Customer
		expectRes        *cscustomer.WebhookMessage
	}{
		{
			name: "normal",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("8ffa19a2-837f-11ec-b57e-9f3906006c0a"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			customerName:  "name new",
			detail:        "detail new",
			email:         "test@test.com",
			phoneNumber:   "+821100000001",
			address:       "somewhere",
			webhookMethod: cscustomer.WebhookMethodPost,
			webhookURI:    "test.com",

			responseCustomer: &cscustomer.Customer{
				ID: uuid.FromStringOrNil("8ffa19a2-837f-11ec-b57e-9f3906006c0a"),
			},
			expectRes: &cscustomer.WebhookMessage{
				ID: uuid.FromStringOrNil("8ffa19a2-837f-11ec-b57e-9f3906006c0a"),
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

			mockReq.EXPECT().CustomerV1CustomerUpdate(ctx, tt.agent.CustomerID, tt.customerName, tt.detail, tt.email, tt.phoneNumber, tt.address, tt.webhookMethod, tt.webhookURI).Return(tt.responseCustomer, nil)

			res, err := h.CustomerSelfUpdate(ctx, tt.agent, tt.customerName, tt.detail, tt.email, tt.phoneNumber, tt.address, tt.webhookMethod, tt.webhookURI)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_CustomerDelete(t *testing.T) {

	type test struct {
		name string

		agent *auth.AuthIdentity
		id    uuid.UUID

		responseCustomers *cscustomer.Customer
		expectRes         *cscustomer.Customer
	}

	tests := []test{
		{
			"normal",

			auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("3e6fe9c8-837e-11ec-84ef-b762e8a7a8fc"),
				},
				Permission: amagent.PermissionProjectSuperAdmin,
			}),
			uuid.FromStringOrNil("d83b9e02-837f-11ec-af3d-b75e44476e6b"),

			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("8ffa19a2-837f-11ec-b57e-9f3906006c0a"),
			},
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("8ffa19a2-837f-11ec-b57e-9f3906006c0a"),
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

			mockReq.EXPECT().CustomerV1CustomerGet(ctx, tt.id).Return(tt.responseCustomers, nil)
			mockReq.EXPECT().CustomerV1CustomerDelete(ctx, tt.id).Return(tt.responseCustomers, nil)

			res, err := h.CustomerDelete(ctx, tt.agent, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_CustomerUpdateBillingAccountID(t *testing.T) {

	type test struct {
		name string

		agent *auth.AuthIdentity

		customerID       uuid.UUID
		billingAccountID uuid.UUID

		responseCustomer       *cscustomer.Customer
		responseBillingAccount *bmaccount.Account
		expectRes              *cscustomer.Customer
	}

	tests := []test{
		{
			name: "normal",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				},
				Permission: amagent.PermissionProjectSuperAdmin,
			}),

			customerID:       uuid.FromStringOrNil("965f317e-1771-11ee-ac07-77247b121f85"),
			billingAccountID: uuid.FromStringOrNil("96a2ce84-1771-11ee-a155-83bf9a14ae55"),

			responseCustomer: &cscustomer.Customer{
				ID: uuid.FromStringOrNil("965f317e-1771-11ee-ac07-77247b121f85"),
			},
			responseBillingAccount: &bmaccount.Account{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("96a2ce84-1771-11ee-a155-83bf9a14ae55"),
					CustomerID: uuid.FromStringOrNil("965f317e-1771-11ee-ac07-77247b121f85"),
				},
			},
			expectRes: &cscustomer.Customer{
				ID: uuid.FromStringOrNil("965f317e-1771-11ee-ac07-77247b121f85"),
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

			mockReq.EXPECT().CustomerV1CustomerGet(ctx, tt.customerID).Return(tt.responseCustomer, nil)
			mockReq.EXPECT().BillingV1AccountGet(ctx, tt.billingAccountID).Return(tt.responseBillingAccount, nil)
			mockReq.EXPECT().CustomerV1CustomerUpdateBillingAccountID(ctx, tt.customerID, tt.billingAccountID).Return(tt.responseCustomer, nil)

			res, err := h.CustomerUpdateBillingAccountID(ctx, tt.agent, tt.customerID, tt.billingAccountID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_CustomerSelfUpdateBillingAccountID(t *testing.T) {

	type test struct {
		name string

		agent *auth.AuthIdentity

		billingAccountID uuid.UUID

		responseBillingAccount *bmaccount.Account
		responseCustomer       *cscustomer.Customer
		expectRes              *cscustomer.WebhookMessage
	}

	tests := []test{
		{
			name: "normal",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("965f317e-1771-11ee-ac07-77247b121f85"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),

			billingAccountID: uuid.FromStringOrNil("96a2ce84-1771-11ee-a155-83bf9a14ae55"),

			responseBillingAccount: &bmaccount.Account{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("96a2ce84-1771-11ee-a155-83bf9a14ae55"),
					CustomerID: uuid.FromStringOrNil("965f317e-1771-11ee-ac07-77247b121f85"),
				},
				TMDelete: nil,
			},
			responseCustomer: &cscustomer.Customer{
				ID: uuid.FromStringOrNil("965f317e-1771-11ee-ac07-77247b121f85"),
			},
			expectRes: &cscustomer.WebhookMessage{
				ID: uuid.FromStringOrNil("965f317e-1771-11ee-ac07-77247b121f85"),
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

			mockReq.EXPECT().BillingV1AccountGet(ctx, tt.billingAccountID).Return(tt.responseBillingAccount, nil)
			mockReq.EXPECT().CustomerV1CustomerUpdateBillingAccountID(ctx, tt.agent.CustomerID, tt.billingAccountID).Return(tt.responseCustomer, nil)

			res, err := h.CustomerSelfUpdateBillingAccountID(ctx, tt.agent, tt.billingAccountID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_CustomerSignup(t *testing.T) {
	tests := []struct {
		name string

		customerName  string
		detail        string
		email         string
		phoneNumber   string
		address       string
		webhookMethod cscustomer.WebhookMethod
		webhookURI    string
		clientIP      string

		responseSignup *cscustomer.SignupResult
		expectRes      *cscustomer.SignupResultWebhookMessage
	}{
		{
			name:          "normal",
			customerName:  "Test Corp",
			detail:        "test detail",
			email:         "test@example.com",
			phoneNumber:   "+1234567890",
			address:       "123 Test St",
			webhookMethod: cscustomer.WebhookMethodPost,
			webhookURI:    "https://example.com/webhook",
			clientIP:      "192.168.1.1",

			responseSignup: &cscustomer.SignupResult{
				Customer: &cscustomer.Customer{
					ID:                 uuid.FromStringOrNil("81133fc8-4a01-11ee-8dbf-4bbf6dd46254"),
					Name:               "Test Corp",
					Email:              "test@example.com",
					TermsAgreedVersion: "2026-02-22T00:00:00Z",
					TermsAgreedIP:      "192.168.1.1",
				},
			},
			expectRes: &cscustomer.SignupResultWebhookMessage{
				Customer: &cscustomer.WebhookMessage{
					ID:    uuid.FromStringOrNil("81133fc8-4a01-11ee-8dbf-4bbf6dd46254"),
					Name:  "Test Corp",
					Email: "test@example.com",
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

			mockReq.EXPECT().CustomerV1CustomerSignup(
				ctx, tt.customerName, tt.detail, tt.email, tt.phoneNumber, tt.address, tt.webhookMethod, tt.webhookURI, tt.clientIP,
			).Return(tt.responseSignup, nil)
			if tt.responseSignup != nil && tt.responseSignup.Customer != nil {
				mockReq.EXPECT().CallV1OutboundConfigCreate(ctx, tt.responseSignup.Customer.ID, &cmoutboundconfig.UpdateRequest{}).Return(&cmoutboundconfig.OutboundConfig{}, nil)
			}

			res, err := h.CustomerSignup(ctx, tt.customerName, tt.detail, tt.email, tt.phoneNumber, tt.address, tt.webhookMethod, tt.webhookURI, tt.clientIP)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

			// Verify terms fields are excluded from JSON output
			b, err := json.Marshal(res)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			var raw map[string]any
			if err := json.Unmarshal(b, &raw); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			customerRaw, ok := raw["customer"].(map[string]any)
			if !ok {
				t.Fatal("Expected customer field in JSON output")
			}

			if _, exists := customerRaw["terms_agreed_version"]; exists {
				t.Error("terms_agreed_version should not leak in API response")
			}
			if _, exists := customerRaw["terms_agreed_ip"]; exists {
				t.Error("terms_agreed_ip should not leak in API response")
			}
		})
	}
}

func Test_CustomerSelfFreezeAndDelete(t *testing.T) {
	tests := []struct {
		name string

		agent *auth.AuthIdentity

		responseCustomer *cscustomer.Customer
		expectRes        *cscustomer.WebhookMessage
	}{
		{
			name: "normal",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("a0f4b592-837e-11ec-9f5f-2f2051d4adac"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),

			responseCustomer: &cscustomer.Customer{
				ID:     uuid.FromStringOrNil("a0f4b592-837e-11ec-9f5f-2f2051d4adac"),
				Status: cscustomer.StatusDeleted,
			},
			expectRes: &cscustomer.WebhookMessage{
				ID:     uuid.FromStringOrNil("a0f4b592-837e-11ec-9f5f-2f2051d4adac"),
				Status: cscustomer.StatusDeleted,
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

			mockReq.EXPECT().CustomerV1CustomerGet(ctx, tt.agent.CustomerID).Return(tt.responseCustomer, nil)
			mockReq.EXPECT().CustomerV1CustomerFreezeAndDelete(ctx, tt.agent.CustomerID).Return(tt.responseCustomer, nil)

			res, err := h.CustomerSelfFreezeAndDelete(ctx, tt.agent)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_CustomerSelfFreezeAndDelete_PermissionDenied(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

	ctx := context.Background()

	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: uuid.FromStringOrNil("a0f4b592-837e-11ec-9f5f-2f2051d4adac"),
		},
		Permission: amagent.PermissionCustomerAgent, // not admin
	})

	_, err := h.CustomerSelfFreezeAndDelete(ctx, agent)
	if err == nil {
		t.Errorf("Expected permission denied error, got nil")
	}
}
