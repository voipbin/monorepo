package servicehandler

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	cmcontact "monorepo/bin-contact-manager/models/contact"
	cmcontactstar "monorepo/bin-contact-manager/models/contactstar"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

var (
	starTestCustomerID = uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	starTestAgentID    = uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979")
	starTestContactID  = uuid.FromStringOrNil("b2c3d4e5-f6a7-8901-bcde-f12345678901")
)

func starTestAgent(perm amagent.Permission) *auth.AuthIdentity {
	return auth.NewAgentIdentity(&amagent.Agent{
		Identity:   commonidentity.Identity{ID: starTestAgentID, CustomerID: starTestCustomerID},
		Permission: perm,
	})
}

func starTestDirect() *auth.AuthIdentity {
	return auth.NewDirectIdentity(&auth.DirectScope{CustomerID: starTestCustomerID})
}

func Test_ServiceAgentContactStarSet(t *testing.T) {
	created := time.Date(2026, 11, 2, 9, 15, 30, 123456000, time.UTC)

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		expectCall bool
		callRes    *cmcontactstar.ContactStar
		callErr    error

		expectRes *cmcontactstar.WebhookMessage
		expectErr error
	}{
		{
			name:       "plain agent permission, agent id comes from the identity",
			agent:      starTestAgent(amagent.PermissionCustomerAgent),
			expectCall: true,
			callRes:    &cmcontactstar.ContactStar{ContactID: starTestContactID, AgentID: starTestAgentID, TMCreate: &created},
			expectRes:  &cmcontactstar.WebhookMessage{ContactID: starTestContactID, Starred: true, TMCreate: &created},
		},
		{
			name:      "direct identity rejected before any call",
			agent:     starTestDirect(),
			expectErr: serviceerrors.ErrAuthenticationRequired,
		},
		{
			name:       "downstream not found is returned",
			agent:      starTestAgent(amagent.PermissionCustomerAgent),
			expectCall: true,
			callErr:    serviceerrors.ErrNotFound,
			expectErr:  serviceerrors.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &serviceHandler{reqHandler: mockReq}
			ctx := context.Background()

			if tt.expectCall {
				mockReq.EXPECT().ContactV1ContactStar(ctx, starTestCustomerID, starTestAgentID, starTestContactID).Return(tt.callRes, tt.callErr)
			}

			res, err := h.ServiceAgentContactStarSet(ctx, tt.agent, starTestContactID)
			if tt.expectErr != nil {
				if !errors.Is(err, tt.expectErr) || res != nil {
					t.Fatalf("Wrong error. expect: %v, got: %v, %v", tt.expectErr, res, err)
				}
				return
			}
			if err != nil || !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v, err: %v", tt.expectRes, res, err)
			}
		})
	}
}

func Test_ServiceAgentContactStarDelete(t *testing.T) {
	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		expectCall bool
		callErr    error

		expectErr error
	}{
		{name: "plain agent permission", agent: starTestAgent(amagent.PermissionCustomerAgent), expectCall: true},
		{name: "direct identity rejected before any call", agent: starTestDirect(), expectErr: serviceerrors.ErrAuthenticationRequired},
		{name: "downstream not found is returned", agent: starTestAgent(amagent.PermissionCustomerAgent), expectCall: true, callErr: serviceerrors.ErrNotFound, expectErr: serviceerrors.ErrNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &serviceHandler{reqHandler: mockReq}
			ctx := context.Background()

			if tt.expectCall {
				mockReq.EXPECT().ContactV1ContactUnstar(ctx, starTestCustomerID, starTestAgentID, starTestContactID).Return(tt.callErr)
			}

			res, err := h.ServiceAgentContactStarDelete(ctx, tt.agent, starTestContactID)
			if tt.expectErr != nil {
				if !errors.Is(err, tt.expectErr) || res != nil {
					t.Fatalf("Wrong error. expect: %v, got: %v, %v", tt.expectErr, res, err)
				}
				return
			}
			expect := &cmcontactstar.WebhookMessage{ContactID: starTestContactID, Starred: false}
			if err != nil || !reflect.DeepEqual(res, expect) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v, err: %v", expect, res, err)
			}
		})
	}
}

func Test_ServiceAgentContactStarList(t *testing.T) {
	created := time.Date(2026, 11, 2, 9, 15, 30, 123456000, time.UTC)

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		expectCall bool
		callRes    []*cmcontactstar.ContactStar
		callToken  string
		callErr    error

		expectRes   []*cmcontactstar.WebhookMessage
		expectToken string
		expectErr   error
	}{
		{
			name:        "plain agent permission",
			agent:       starTestAgent(amagent.PermissionCustomerAgent),
			expectCall:  true,
			callRes:     []*cmcontactstar.ContactStar{{ContactID: starTestContactID, AgentID: starTestAgentID, TMCreate: &created}},
			callToken:   "2026-11-02T09:15:30.123456Z",
			expectRes:   []*cmcontactstar.WebhookMessage{{ContactID: starTestContactID, Starred: true, TMCreate: &created}},
			expectToken: "2026-11-02T09:15:30.123456Z",
		},
		{
			name:        "empty result is an empty list, not nil",
			agent:       starTestAgent(amagent.PermissionCustomerAgent),
			expectCall:  true,
			expectRes:   []*cmcontactstar.WebhookMessage{},
			expectToken: "",
		},
		{name: "direct identity rejected before any call", agent: starTestDirect(), expectErr: serviceerrors.ErrAuthenticationRequired},
		{name: "downstream error is returned", agent: starTestAgent(amagent.PermissionCustomerAgent), expectCall: true, callErr: serviceerrors.ErrNotFound, expectErr: serviceerrors.ErrNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &serviceHandler{reqHandler: mockReq}
			ctx := context.Background()

			if tt.expectCall {
				mockReq.EXPECT().ContactV1ContactStarList(ctx, starTestCustomerID, starTestAgentID, uint64(50), "tok").Return(tt.callRes, tt.callToken, tt.callErr)
			}

			res, token, err := h.ServiceAgentContactStarList(ctx, tt.agent, 50, "tok")
			if tt.expectErr != nil {
				if !errors.Is(err, tt.expectErr) {
					t.Fatalf("Wrong error. expect: %v, got: %v", tt.expectErr, err)
				}
				return
			}
			if err != nil || !reflect.DeepEqual(res, tt.expectRes) || token != tt.expectToken {
				t.Errorf("Wrong match.\nexpect: %v %q\ngot: %v %q, err: %v", tt.expectRes, tt.expectToken, res, token, err)
			}
		})
	}
}

func Test_ServiceAgentContactListStarred(t *testing.T) {
	created := time.Date(2026, 11, 2, 9, 15, 30, 123456000, time.UTC)
	starred := cmcontact.Contact{
		Identity:  commonidentity.Identity{ID: starTestContactID, CustomerID: starTestCustomerID},
		FirstName: "John",
		TMCreate:  &created,
	}

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		expectCall bool
		callRes    []cmcontact.Contact
		callToken  string
		callErr    error

		expectRes   []*cmcontact.WebhookMessage
		expectToken string
		expectErr   error
	}{
		{
			name:        "plain agent permission",
			agent:       starTestAgent(amagent.PermissionCustomerAgent),
			expectCall:  true,
			callRes:     []cmcontact.Contact{starred},
			callToken:   "2026-11-02T09:15:30.123456Z",
			expectRes:   []*cmcontact.WebhookMessage{starred.ConvertWebhookMessage()},
			expectToken: "2026-11-02T09:15:30.123456Z",
		},
		{name: "empty result is an empty list, not nil", agent: starTestAgent(amagent.PermissionCustomerAgent), expectCall: true, expectRes: []*cmcontact.WebhookMessage{}},
		{name: "direct identity rejected before any call", agent: starTestDirect(), expectErr: serviceerrors.ErrAuthenticationRequired},
		{name: "downstream error is returned", agent: starTestAgent(amagent.PermissionCustomerAgent), expectCall: true, callErr: serviceerrors.ErrNotFound, expectErr: serviceerrors.ErrNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &serviceHandler{reqHandler: mockReq}
			ctx := context.Background()

			if tt.expectCall {
				mockReq.EXPECT().ContactV1ContactStarredList(ctx, starTestCustomerID, starTestAgentID, uint64(50), "tok").Return(tt.callRes, tt.callToken, tt.callErr)
			}

			res, token, err := h.ServiceAgentContactListStarred(ctx, tt.agent, 50, "tok")
			if tt.expectErr != nil {
				if !errors.Is(err, tt.expectErr) {
					t.Fatalf("Wrong error. expect: %v, got: %v", tt.expectErr, err)
				}
				return
			}
			if err != nil || !reflect.DeepEqual(res, tt.expectRes) || token != tt.expectToken {
				t.Errorf("Wrong match.\nexpect: %v %q\ngot: %v %q, err: %v", tt.expectRes, tt.expectToken, res, token, err)
			}
		})
	}
}
