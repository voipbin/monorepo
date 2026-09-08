package servicehandler

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	amagent "monorepo/bin-agent-manager/models/agent"
	cscustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/pkg/dbhandler"
	"monorepo/bin-api-manager/pkg/serviceerrors"
)

func Test_AuthLogin(t *testing.T) {

	tests := []struct {
		name string

		username string
		password string

		responseAgent    *amagent.Agent
		responseCustomer *cscustomer.Customer
		responseCurTime  string
	}{
		{
			name: "normal",

			username: "test@test.com",
			password: "testpassword",

			responseAgent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("6bc342d0-8aed-11ee-a07d-7bc7fee5a336"),
					CustomerID: uuid.FromStringOrNil("6c0ff198-8aed-11ee-8a04-474584947e03"),
				},
			},
			responseCustomer: &cscustomer.Customer{
				ID:     uuid.FromStringOrNil("6c0ff198-8aed-11ee-8a04-474584947e03"),
				Status: cscustomer.StatusActive,
			},
			responseCurTime: "2023-11-19 09:29:11.763331118",
		},
		{
			// Regression guard for design 4-1-1. A user inside the 72h
			// post-signup verification window (VOIP-1490) is status='initial'
			// and MUST still be able to log in. If this row starts failing,
			// someone replaced the deny-list with AuthBoot's
			// "status != active" allow-list and broke onboarding.
			name: "initial customer is allowed to log in",

			username: "initial@test.com",
			password: "testpassword",

			responseAgent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("3f0d1a26-8d55-11f0-9c4a-3b6c5e2f7a11"),
					CustomerID: uuid.FromStringOrNil("3f0d1e04-8d55-11f0-8f2b-df1a7c9b4e22"),
				},
			},
			responseCustomer: &cscustomer.Customer{
				ID:     uuid.FromStringOrNil("3f0d1e04-8d55-11f0-8f2b-df1a7c9b4e22"),
				Status: cscustomer.StatusInitial,
			},
			responseCurTime: "2023-11-19 09:29:11.763331118",
		},
		{
			// Regression guard for design 4-1-1. The shipped frozen
			// self-recovery UX calls DELETE /auth/unregister, which sits behind
			// authentication -- so a frozen user who cannot log in cannot
			// recover at all. Blocking this row would delete a shipped
			// recovery path, which is worse than the bug being fixed.
			name: "frozen customer is allowed to log in",

			username: "frozen@test.com",
			password: "testpassword",

			responseAgent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("3f0d1f9e-8d55-11f0-a1d3-9f4e2b7c6a33"),
					CustomerID: uuid.FromStringOrNil("3f0d20fc-8d55-11f0-b7e5-4c8a1d3f9b44"),
				},
			},
			responseCustomer: &cscustomer.Customer{
				ID:     uuid.FromStringOrNil("3f0d20fc-8d55-11f0-b7e5-4c8a1d3f9b44"),
				Status: cscustomer.StatusFrozen,
			},
			responseCurTime: "2023-11-19 09:29:11.763331118",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := serviceHandler{
				reqHandler:  mockReq,
				dbHandler:   mockDB,
				utilHandler: mockUtil,
				jwtKey:      []byte("testkey"),
			}
			ctx := context.Background()

			mockReq.EXPECT().AgentV1Login(ctx, gomock.Any(), tt.username, tt.password).Return(tt.responseAgent, nil)
			mockReq.EXPECT().CustomerV1CustomerGet(ctx, tt.responseAgent.CustomerID).Return(tt.responseCustomer, nil)
			mockUtil.EXPECT().TimeGetCurTimeAdd(TokenExpiration).Return(tt.responseCurTime)

			res, err := h.AuthLogin(ctx, tt.username, tt.password)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res == "" {
				t.Errorf("Expected non-empty token, got empty string")
			}

			// Parse the token and verify claims
			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			claims, err := h.AuthJWTParse(ctx, res)
			if err != nil {
				t.Errorf("Could not parse token. err: %v", err)
			}

			// Verify "type" claim is "agent"
			tokenType, ok := claims["type"]
			if !ok {
				t.Errorf("Expected 'type' claim in token, but not found")
			}
			if tokenType != "agent" {
				t.Errorf("Wrong type claim. expected: agent, got: %v", tokenType)
			}

			// Verify "agent" claim exists
			if _, ok := claims["agent"]; !ok {
				t.Errorf("Expected 'agent' claim in token, but not found")
			}
		})
	}
}

// Test_AuthLogin_blockedCustomerStatus covers design 4-1b: AuthLogin refuses to
// mint a JWT for an account the v1 gate would reject anyway, so an
// expired-account holder cannot stockpile 7-day tokens and wait for the gate's
// fail-open branch.
func Test_AuthLogin_blockedCustomerStatus(t *testing.T) {

	tests := []struct {
		name string

		username string
		password string

		responseAgent    *amagent.Agent
		responseCustomer *cscustomer.Customer

		expectErr error
	}{
		{
			name: "expired customer is rejected",

			username: "expired@test.com",
			password: "testpassword",

			responseAgent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("8a1c4b10-8d55-11f0-9e6d-1f2a3b4c5d66"),
					CustomerID: uuid.FromStringOrNil("8a1c4d5e-8d55-11f0-8b7c-2e3f4a5b6c77"),
				},
			},
			responseCustomer: &cscustomer.Customer{
				ID:     uuid.FromStringOrNil("8a1c4d5e-8d55-11f0-8b7c-2e3f4a5b6c77"),
				Status: cscustomer.StatusExpired,
			},

			expectErr: serviceerrors.ErrAccountExpired,
		},
		{
			// The customer_deleted cascade is known to miss customers
			// (VOIP-1395), leaving a live agent that AgentGetByUsername's
			// deleted=false filter happily returns. Close it here too.
			name: "deleted customer is rejected",

			username: "deleted@test.com",
			password: "testpassword",

			responseAgent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("8a1c4eb2-8d55-11f0-a4e8-3c5d6e7f8a88"),
					CustomerID: uuid.FromStringOrNil("8a1c5006-8d55-11f0-92f1-4d6e7f8a9b99"),
				},
			},
			responseCustomer: &cscustomer.Customer{
				ID:     uuid.FromStringOrNil("8a1c5006-8d55-11f0-92f1-4d6e7f8a9b99"),
				Status: cscustomer.StatusDeleted,
			},

			expectErr: serviceerrors.ErrAccountDeleted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := serviceHandler{
				reqHandler:  mockReq,
				dbHandler:   mockDB,
				utilHandler: mockUtil,
				jwtKey:      []byte("testkey"),
			}
			ctx := context.Background()

			mockReq.EXPECT().AgentV1Login(ctx, gomock.Any(), tt.username, tt.password).Return(tt.responseAgent, nil)
			mockReq.EXPECT().CustomerV1CustomerGet(ctx, tt.responseAgent.CustomerID).Return(tt.responseCustomer, nil)
			// No TimeGetCurTimeAdd expectation: gomock fails the test if the
			// token is minted anyway, which is the property under test.

			res, err := h.AuthLogin(ctx, tt.username, tt.password)
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: ok")
			}
			if !errors.Is(err, tt.expectErr) {
				t.Errorf("Wrong error. expect: %v, got: %v", tt.expectErr, err)
			}
			if res != "" {
				t.Errorf("Expected empty token on rejection, got: %v", res)
			}
		})
	}
}

// Test_AuthLogin_customerGetFailsClosed covers design 4-1-1: the customer
// lookup fails CLOSED. If it failed open, an outage window would double as a
// credential-issuing window and restore exactly the amplification 4-1b removes.
//
// The returned error must NOT satisfy the account-status sentinels -- it is not
// an expiry determination, and PostLogin relies on that to keep returning the
// opaque 400 rather than a 403 ACCOUNT_EXPIRED envelope.
func Test_AuthLogin_customerGetFailsClosed(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	h := serviceHandler{
		reqHandler:  mockReq,
		dbHandler:   mockDB,
		utilHandler: mockUtil,
		jwtKey:      []byte("testkey"),
	}
	ctx := context.Background()

	a := &amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("c31f7a44-8d55-11f0-8c2e-5e7f8a9b0c11"),
			CustomerID: uuid.FromStringOrNil("c31f7c92-8d55-11f0-97a3-6f8a9b0c1d22"),
		},
	}
	rpcErr := fmt.Errorf("customer-manager unavailable")

	mockReq.EXPECT().AgentV1Login(ctx, gomock.Any(), "someone@test.com", "testpassword").Return(a, nil)
	mockReq.EXPECT().CustomerV1CustomerGet(ctx, a.CustomerID).Return(nil, rpcErr)
	// No TimeGetCurTimeAdd expectation: minting a token here would fail the test.

	res, err := h.AuthLogin(ctx, "someone@test.com", "testpassword")
	if err == nil {
		t.Fatalf("Wrong match. expect: error (fail closed), got: ok")
	}
	if res != "" {
		t.Errorf("Expected empty token on lookup failure, got: %v", res)
	}
	if !errors.Is(err, rpcErr) {
		t.Errorf("Expected the underlying RPC error to be wrapped. got: %v", err)
	}
	if errors.Is(err, serviceerrors.ErrAccountExpired) || errors.Is(err, serviceerrors.ErrAccountDeleted) {
		t.Errorf("An RPC failure MUST NOT be reported as an account-status refusal. got: %v", err)
	}
}

func Test_AuthJWTGenerate(t *testing.T) {

	tests := []struct {
		name string

		data map[string]interface{}

		responseCurTime string

		expectRes map[string]interface{}
	}{
		{
			name: "normal",

			data: map[string]interface{}{
				"key1": "val1",
				"key2": "val2",
			},

			responseCurTime: "2023-11-19 09:29:11.763331118",
			expectRes: map[string]interface{}{
				"key1":   "val1",
				"key2":   "val2",
				"expire": "2023-11-19 09:29:11.763331118",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &serviceHandler{
				reqHandler:  mockReq,
				dbHandler:   mockDB,
				utilHandler: mockUtil,
				jwtKey:      []byte("testkey"),
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTimeAdd(TokenExpiration).Return(tt.responseCurTime)
			token, err := h.AuthJWTGenerate(tt.data)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			res, err := h.AuthJWTParse(ctx, token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}
