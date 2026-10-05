package servicehandler

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	rmextension "monorepo/bin-registrar-manager/models/extension"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	csaccesskey "monorepo/bin-customer-manager/models/accesskey"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/pkg/dbhandler"
)

var (
	testSAExtCustomerID      = uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	testSAExtOtherCustomerID = uuid.FromStringOrNil("6a1b3c4d-8e5f-11ee-97b2-cfe7337b701c")
	testSAExtAgentID         = uuid.FromStringOrNil("2ddf1c90-bbc1-11ef-b991-1bd6ee52cbc5")
	testSAExtOwnID           = uuid.FromStringOrNil("2e3b3c3c-bbc1-11ef-93c0-17537443cb56")
	testSAExtOtherID         = uuid.FromStringOrNil("2e67d562-bbc1-11ef-b531-a3248f6d1477")
	testSAExtForeignID       = uuid.FromStringOrNil("7a7a7a7a-bbc1-11ef-b531-a3248f6d1477")
	testSAExtMissingID       = uuid.FromStringOrNil("8b8b8b8b-bbc1-11ef-b531-a3248f6d1477")
)

// newTestSAExtAgent returns an agent identity whose JWT-time addresses are jwtAddresses.
func newTestSAExtAgent(permission amagent.Permission, jwtAddresses []commonaddress.Address) *auth.AuthIdentity {
	return auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         testSAExtAgentID,
			CustomerID: testSAExtCustomerID,
		},
		Permission: permission,
		Addresses:  jwtAddresses,
	})
}

func newTestSAExtAddress(extensionID uuid.UUID) commonaddress.Address {
	return commonaddress.Address{Type: commonaddress.TypeExtension, Target: extensionID.String()}
}

func newTestSAExtension(id uuid.UUID, customerID uuid.UUID, name string) rmextension.Extension {
	return rmextension.Extension{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
		Name:       name,
		Extension:  "1001",
		DomainName: "ab12.reg.voipbin.net",
		Realm:      "ab12.reg.voipbin.net",
		Username:   "1001",
		Password:   "s3cretPass-" + name,
		DirectHash: "direct.hash-" + name,
	}
}

// newTestSAExtMasked returns the expected webhook message of ext when it is not owned by the caller.
func newTestSAExtMasked(ext rmextension.Extension) *rmextension.WebhookMessage {
	m := ext.ConvertWebhookMessage()
	m.Password = ""
	m.DirectHash = ""
	return m
}

func newTestSAExtNonAgentIdentities() map[string]*auth.AuthIdentity {
	return map[string]*auth.AuthIdentity{
		"direct":    auth.NewDirectIdentity(&auth.DirectScope{CustomerID: testSAExtCustomerID}),
		"accesskey": auth.NewAccesskeyIdentity(&csaccesskey.Accesskey{CustomerID: testSAExtCustomerID}),
		"delegate":  auth.NewDelegateIdentity(&auth.DelegateScope{CustomerID: testSAExtCustomerID}),
	}
}

func Test_ServiceAgentExtensionList(t *testing.T) {
	const pageToken = "2020-10-20T01:00:00.995000Z"
	const pageSize = uint64(10)
	expectFilters := map[rmextension.Field]any{
		rmextension.FieldCustomerID: testSAExtCustomerID,
		rmextension.FieldDeleted:    false,
	}

	ownExt := newTestSAExtension(testSAExtOwnID, testSAExtCustomerID, "own")
	otherExt := newTestSAExtension(testSAExtOtherID, testSAExtCustomerID, "other")
	foreignExt := newTestSAExtension(testSAExtForeignID, testSAExtOtherCustomerID, "foreign")

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		// responseAgent is what agentGet (the up-to-date record) returns. It can differ from the JWT-time addresses.
		responseAgent      *amagent.Agent
		responseExtensions []rmextension.Extension

		expectRes []*rmextension.WebhookMessage
	}{
		{
			name:  "own is plain, other is masked",
			agent: newTestSAExtAgent(amagent.PermissionCustomerAgent, []commonaddress.Address{newTestSAExtAddress(testSAExtOwnID)}),
			responseAgent: &amagent.Agent{
				Addresses: []commonaddress.Address{newTestSAExtAddress(testSAExtOwnID)},
			},
			responseExtensions: []rmextension.Extension{ownExt, otherExt},
			expectRes:          []*rmextension.WebhookMessage{ownExt.ConvertWebhookMessage(), newTestSAExtMasked(otherExt)},
		},
		{
			name:  "ownership follows the fresh agent record, not the JWT addresses (assigned after login)",
			agent: newTestSAExtAgent(amagent.PermissionCustomerAgent, nil),
			responseAgent: &amagent.Agent{
				Addresses: []commonaddress.Address{newTestSAExtAddress(testSAExtOwnID)},
			},
			responseExtensions: []rmextension.Extension{ownExt, otherExt},
			expectRes:          []*rmextension.WebhookMessage{ownExt.ConvertWebhookMessage(), newTestSAExtMasked(otherExt)},
		},
		{
			name:  "ownership revoked after login is masked even though the JWT still lists it",
			agent: newTestSAExtAgent(amagent.PermissionCustomerAgent, []commonaddress.Address{newTestSAExtAddress(testSAExtOwnID)}),
			responseAgent: &amagent.Agent{
				Addresses: []commonaddress.Address{},
			},
			responseExtensions: []rmextension.Extension{ownExt},
			expectRes:          []*rmextension.WebhookMessage{newTestSAExtMasked(ownExt)},
		},
		{
			name:  "agent without any extension sees everything masked (a deleted agent has empty addresses)",
			agent: newTestSAExtAgent(amagent.PermissionCustomerAgent, nil),
			responseAgent: &amagent.Agent{
				Addresses: []commonaddress.Address{},
			},
			responseExtensions: []rmextension.Extension{otherExt},
			expectRes:          []*rmextension.WebhookMessage{newTestSAExtMasked(otherExt)},
		},
		{
			name:  "non-extension addresses do not grant ownership",
			agent: newTestSAExtAgent(amagent.PermissionCustomerAgent, nil),
			responseAgent: &amagent.Agent{
				Addresses: []commonaddress.Address{
					{Type: commonaddress.TypeTel, Target: testSAExtOtherID.String()},
				},
			},
			responseExtensions: []rmextension.Extension{otherExt},
			expectRes:          []*rmextension.WebhookMessage{newTestSAExtMasked(otherExt)},
		},
		{
			name:  "an item of another customer is dropped defensively",
			agent: newTestSAExtAgent(amagent.PermissionCustomerAgent, nil),
			responseAgent: &amagent.Agent{
				Addresses: []commonaddress.Address{newTestSAExtAddress(testSAExtOwnID)},
			},
			responseExtensions: []rmextension.Extension{ownExt, foreignExt, otherExt},
			expectRes:          []*rmextension.WebhookMessage{ownExt.ConvertWebhookMessage(), newTestSAExtMasked(otherExt)},
		},
		{
			name:  "empty list",
			agent: newTestSAExtAgent(amagent.PermissionCustomerAgent, nil),
			responseAgent: &amagent.Agent{
				Addresses: []commonaddress.Address{},
			},
			responseExtensions: []rmextension.Extension{},
			expectRes:          []*rmextension.WebhookMessage{},
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

			// the ownership lookup must happen exactly once per request.
			mockReq.EXPECT().AgentV1AgentGet(ctx, testSAExtAgentID).Return(tt.responseAgent, nil).Times(1)
			mockReq.EXPECT().RegistrarV1ExtensionList(ctx, pageToken, pageSize, expectFilters).Return(tt.responseExtensions, nil).Times(1)

			res, err := h.ServiceAgentExtensionList(ctx, tt.agent, pageSize, pageToken)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ServiceAgentExtensionList_does_not_mutate_the_source_records(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &serviceHandler{reqHandler: mockReq}
	ctx := context.Background()

	otherExt := newTestSAExtension(testSAExtOtherID, testSAExtCustomerID, "other")
	source := []rmextension.Extension{otherExt}

	mockReq.EXPECT().AgentV1AgentGet(ctx, testSAExtAgentID).Return(&amagent.Agent{}, nil)
	mockReq.EXPECT().RegistrarV1ExtensionList(ctx, "token", uint64(5), gomock.Any()).Return(source, nil)

	a := newTestSAExtAgent(amagent.PermissionCustomerAgent, nil)
	if _, err := h.ServiceAgentExtensionList(ctx, a, 5, "token"); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	if source[0].Password != otherExt.Password || source[0].DirectHash != otherExt.DirectHash {
		t.Errorf("Masking must act on the copy, but the source record was changed. got password: %q, direct_hash: %q", source[0].Password, source[0].DirectHash)
	}
}

func Test_ServiceAgentExtensionList_defaults_the_page_token(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	h := &serviceHandler{reqHandler: mockReq, utilHandler: mockUtil}
	ctx := context.Background()

	mockUtil.EXPECT().TimeGetCurTime().Return("2020-10-20T01:00:00.995000Z")
	mockReq.EXPECT().AgentV1AgentGet(ctx, testSAExtAgentID).Return(&amagent.Agent{}, nil)
	mockReq.EXPECT().RegistrarV1ExtensionList(ctx, "2020-10-20T01:00:00.995000Z", uint64(100), gomock.Any()).Return([]rmextension.Extension{}, nil)

	a := newTestSAExtAgent(amagent.PermissionCustomerAgent, nil)
	if _, err := h.ServiceAgentExtensionList(ctx, a, 100, ""); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
}

func Test_ServiceAgentExtensionList_errors(t *testing.T) {
	ctx := context.Background()

	t.Run("non-agent identities are rejected before any RPC", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h := &serviceHandler{reqHandler: requesthandler.NewMockRequestHandler(mc)}

		for name, a := range newTestSAExtNonAgentIdentities() {
			if _, err := h.ServiceAgentExtensionList(ctx, a, 10, "t"); !errors.Is(err, serviceerrors.ErrAuthenticationRequired) {
				t.Errorf("%s: Wrong match. expect: ErrAuthenticationRequired, got: %v", name, err)
			}
		}
	})

	t.Run("ownership lookup failure fails closed", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		mockReq := requesthandler.NewMockRequestHandler(mc)
		h := &serviceHandler{reqHandler: mockReq}

		boom := errors.New("agent-manager down")
		mockReq.EXPECT().AgentV1AgentGet(ctx, testSAExtAgentID).Return(nil, boom)

		a := newTestSAExtAgent(amagent.PermissionCustomerAgent, nil)
		if res, err := h.ServiceAgentExtensionList(ctx, a, 10, "t"); !errors.Is(err, boom) || res != nil {
			t.Errorf("Wrong match. expect: the lookup error and no result, got: res=%v err=%v", res, err)
		}
	})

	t.Run("registrar list failure is an error", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		mockReq := requesthandler.NewMockRequestHandler(mc)
		h := &serviceHandler{reqHandler: mockReq}

		boom := errors.New("registrar down")
		mockReq.EXPECT().AgentV1AgentGet(ctx, testSAExtAgentID).Return(&amagent.Agent{}, nil)
		mockReq.EXPECT().RegistrarV1ExtensionList(ctx, "t", uint64(10), gomock.Any()).Return(nil, boom)

		a := newTestSAExtAgent(amagent.PermissionCustomerAgent, nil)
		if res, err := h.ServiceAgentExtensionList(ctx, a, 10, "t"); !errors.Is(err, boom) || res != nil {
			t.Errorf("Wrong match. expect: the list error and no result, got: res=%v err=%v", res, err)
		}
	})
}

func Test_ServiceAgentExtensionGet(t *testing.T) {
	deletedAt := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)

	ownExt := newTestSAExtension(testSAExtOwnID, testSAExtCustomerID, "own")
	otherExt := newTestSAExtension(testSAExtOtherID, testSAExtCustomerID, "other")
	foreignExt := newTestSAExtension(testSAExtForeignID, testSAExtOtherCustomerID, "foreign")
	deletedOwnExt := ownExt
	deletedOwnExt.TMDelete = &deletedAt

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		extensionID       uuid.UUID
		responseExtension *rmextension.Extension
		responseAgent     *amagent.Agent // nil: the owner must not be looked up

		expectRes *rmextension.WebhookMessage // nil: the canonical "extension not found" error is expected
	}{
		{
			name:              "own extension is returned with credentials",
			agent:             newTestSAExtAgent(amagent.PermissionCustomerAgent, nil),
			extensionID:       testSAExtOwnID,
			responseExtension: &ownExt,
			responseAgent:     &amagent.Agent{Addresses: []commonaddress.Address{newTestSAExtAddress(testSAExtOwnID)}},
			expectRes:         ownExt.ConvertWebhookMessage(),
		},
		{
			name:              "another agent's extension of the same customer is masked",
			agent:             newTestSAExtAgent(amagent.PermissionCustomerAgent, []commonaddress.Address{newTestSAExtAddress(testSAExtOwnID)}),
			extensionID:       testSAExtOtherID,
			responseExtension: &otherExt,
			responseAgent:     &amagent.Agent{Addresses: []commonaddress.Address{newTestSAExtAddress(testSAExtOwnID)}},
			expectRes:         newTestSAExtMasked(otherExt),
		},
		{
			name:              "ownership follows the fresh agent record, not the JWT addresses",
			agent:             newTestSAExtAgent(amagent.PermissionCustomerAgent, []commonaddress.Address{newTestSAExtAddress(testSAExtOwnID)}),
			extensionID:       testSAExtOwnID,
			responseExtension: &ownExt,
			responseAgent:     &amagent.Agent{Addresses: []commonaddress.Address{}},
			expectRes:         newTestSAExtMasked(ownExt),
		},
		{
			name:              "an extension of another customer is not found and the owner is never looked up",
			agent:             newTestSAExtAgent(amagent.PermissionCustomerAgent, nil),
			extensionID:       testSAExtForeignID,
			responseExtension: &foreignExt,
		},
		{
			name:              "a project super admin still cannot read another customer's extension",
			agent:             newTestSAExtAgent(amagent.PermissionProjectSuperAdmin, nil),
			extensionID:       testSAExtForeignID,
			responseExtension: &foreignExt,
		},
		{
			name:              "a deleted extension is not found even when it was the agent's own",
			agent:             newTestSAExtAgent(amagent.PermissionCustomerAgent, []commonaddress.Address{newTestSAExtAddress(testSAExtOwnID)}),
			extensionID:       testSAExtOwnID,
			responseExtension: &deletedOwnExt,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &serviceHandler{reqHandler: mockReq}
			ctx := context.Background()

			mockReq.EXPECT().RegistrarV1ExtensionGet(ctx, tt.extensionID).Return(tt.responseExtension, nil)
			if tt.responseAgent != nil {
				mockReq.EXPECT().AgentV1AgentGet(ctx, testSAExtAgentID).Return(tt.responseAgent, nil).Times(1)
			}

			res, err := h.ServiceAgentExtensionGet(ctx, tt.agent, tt.extensionID)
			if tt.expectRes == nil {
				assertExtensionNotFound(t, err)
				if res != nil {
					t.Errorf("Wrong match. expect: no result, got: %v", res)
				}
				return
			}

			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

// assertExtensionNotFound asserts that err is the canonical "extension not found" error.
func assertExtensionNotFound(t *testing.T, err error) {
	t.Helper()

	var ve *cerrors.VoipbinError
	if !errors.As(err, &ve) {
		t.Fatalf("Wrong match. expect: a typed VoipbinError, got: %v", err)
	}
	if ve.Status != cerrors.StatusNotFound || ve.Reason != "EXTENSION_NOT_FOUND" || ve.Domain != string(commonoutline.ServiceNameRegistrarManager) || ve.Message != "The extension was not found." {
		t.Errorf("Wrong match. expect: NOT_FOUND/EXTENSION_NOT_FOUND from registrar-manager, got: %+v", ve)
	}
}

// The three "cannot see this extension" cases must be indistinguishable to the caller,
// otherwise the response reason leaks whether an id exists in another customer.
func Test_ServiceAgentExtensionGet_not_found_responses_are_identical(t *testing.T) {
	ctx := context.Background()
	deletedAt := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)

	foreign := newTestSAExtension(testSAExtForeignID, testSAExtOtherCustomerID, "foreign")
	deleted := newTestSAExtension(testSAExtOwnID, testSAExtCustomerID, "deleted")
	deleted.TMDelete = &deletedAt

	// what the registrar really returns for an unknown id (extensionhandler.Get)
	registrarNotFound := cerrors.NotFound(commonoutline.ServiceNameRegistrarManager, "EXTENSION_NOT_FOUND", "The extension was not found.").Wrap(errors.New("db: not found"))

	collect := func(id uuid.UUID, ext *rmextension.Extension, rpcErr error) *cerrors.VoipbinError {
		mc := gomock.NewController(t)
		defer mc.Finish()
		mockReq := requesthandler.NewMockRequestHandler(mc)
		h := &serviceHandler{reqHandler: mockReq}
		mockReq.EXPECT().RegistrarV1ExtensionGet(ctx, id).Return(ext, rpcErr)

		a := newTestSAExtAgent(amagent.PermissionCustomerAgent, nil)
		_, err := h.ServiceAgentExtensionGet(ctx, a, id)
		var ve *cerrors.VoipbinError
		if !errors.As(err, &ve) {
			t.Fatalf("Wrong match. expect: a typed VoipbinError, got: %v", err)
		}
		return ve
	}

	missing := collect(testSAExtMissingID, nil, registrarNotFound)
	other := collect(testSAExtForeignID, &foreign, nil)
	gone := collect(testSAExtOwnID, &deleted, nil)

	for name, ve := range map[string]*cerrors.VoipbinError{"other customer": other, "deleted": gone} {
		if ve.Status != missing.Status || ve.Reason != missing.Reason || ve.Domain != missing.Domain || ve.Message != missing.Message {
			t.Errorf("Wrong match. the %s response differs from the missing-id response.\nmissing: %+v\n%s: %+v", name, missing, name, ve)
		}
	}
}

func Test_ServiceAgentExtensionGet_errors(t *testing.T) {
	ctx := context.Background()

	t.Run("non-agent identities are rejected before any RPC", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h := &serviceHandler{reqHandler: requesthandler.NewMockRequestHandler(mc)}

		for name, a := range newTestSAExtNonAgentIdentities() {
			if _, err := h.ServiceAgentExtensionGet(ctx, a, testSAExtOwnID); !errors.Is(err, serviceerrors.ErrAuthenticationRequired) {
				t.Errorf("%s: Wrong match. expect: ErrAuthenticationRequired, got: %v", name, err)
			}
		}
	})

	t.Run("ownership lookup failure fails closed", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		mockReq := requesthandler.NewMockRequestHandler(mc)
		h := &serviceHandler{reqHandler: mockReq}

		ext := newTestSAExtension(testSAExtOwnID, testSAExtCustomerID, "own")
		boom := errors.New("agent-manager down")
		mockReq.EXPECT().RegistrarV1ExtensionGet(ctx, testSAExtOwnID).Return(&ext, nil)
		mockReq.EXPECT().AgentV1AgentGet(ctx, testSAExtAgentID).Return(nil, boom)

		a := newTestSAExtAgent(amagent.PermissionCustomerAgent, nil)
		if res, err := h.ServiceAgentExtensionGet(ctx, a, testSAExtOwnID); !errors.Is(err, boom) || res != nil {
			t.Errorf("Wrong match. expect: the lookup error and no result, got: res=%v err=%v", res, err)
		}
	})

	t.Run("registrar failure is returned as is", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		mockReq := requesthandler.NewMockRequestHandler(mc)
		h := &serviceHandler{reqHandler: mockReq}

		boom := errors.New("registrar down")
		mockReq.EXPECT().RegistrarV1ExtensionGet(ctx, testSAExtOwnID).Return(nil, boom)

		a := newTestSAExtAgent(amagent.PermissionCustomerAgent, nil)
		if _, err := h.ServiceAgentExtensionGet(ctx, a, testSAExtOwnID); !errors.Is(err, boom) {
			t.Errorf("Wrong match. expect: %v, got: %v", boom, err)
		}
	})
}
