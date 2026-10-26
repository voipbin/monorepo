package agenthandler

import (
	"context"
	stderrors "errors"
	"strings"
	"testing"

	bmaccount "monorepo/bin-billing-manager/models/account"
	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	rmextension "monorepo/bin-registrar-manager/models/extension"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-agent-manager/pkg/cachehandler"
	"monorepo/bin-agent-manager/pkg/dbhandler"
)

// wrapMocks bundles the mocks of an agentHandler built for the error wrapping tests.
type wrapMocks struct {
	req    *requesthandler.MockRequestHandler
	db     *dbhandler.MockDBHandler
	notify *notifyhandler.MockNotifyHandler
	util   *utilhandler.MockUtilHandler
	cache  *cachehandler.MockCacheHandler
}

func newWrapTestHandler(mc *gomock.Controller) (*agentHandler, *wrapMocks) {
	m := &wrapMocks{
		req:    requesthandler.NewMockRequestHandler(mc),
		db:     dbhandler.NewMockDBHandler(mc),
		notify: notifyhandler.NewMockNotifyHandler(mc),
		util:   utilhandler.NewMockUtilHandler(mc),
		cache:  cachehandler.NewMockCacheHandler(mc),
	}

	h := &agentHandler{
		reqHandler:    m.req,
		db:            m.db,
		notifyHandler: m.notify,
		utilHandler:   m.util,
		cache:         m.cache,
	}

	return h, m
}

// reasonOf returns the Reason of the typed error found in the chain of the given error.
func reasonOf(t *testing.T, err error) string {
	t.Helper()

	var ve *cerrors.VoipbinError
	if !stderrors.As(err, &ve) {
		t.Fatalf("Wrong match. expect: VoipbinError, got: %v", err)
	}

	return ve.Reason
}

func Test_Create_stepErrors(t *testing.T) {
	customerID := uuid.FromStringOrNil("5fd7f9b8-cb37-11ee-bd29-f30560a6ac86")
	username := "create-step@example.com"
	errBoom := stderrors.New("boom")

	tests := []struct {
		name string

		ringMethod agent.RingMethod
		mockFunc   func(ctx context.Context, m *wrapMocks)

		expectReason string
		expectPrefix string
	}{
		{
			name: "billing check rpc failure",

			ringMethod: agent.RingMethodRingAll,
			mockFunc: func(ctx context.Context, m *wrapMocks) {
				m.req.EXPECT().BillingV1AccountIsValidResourceLimitByCustomerID(ctx, customerID, bmaccount.ResourceTypeAgent).Return(false, errBoom)
			},

			expectReason: "BILLING_CHECK_FAILED",
		},
		{
			name: "resource limit exceeded",

			ringMethod: agent.RingMethodRingAll,
			mockFunc: func(ctx context.Context, m *wrapMocks) {
				m.req.EXPECT().BillingV1AccountIsValidResourceLimitByCustomerID(ctx, customerID, bmaccount.ResourceTypeAgent).Return(false, nil)
			},

			expectReason: "RESOURCE_LIMIT_EXCEEDED",
		},
		{
			name: "username is not an email address",

			ringMethod: agent.RingMethodRingAll,
			mockFunc: func(ctx context.Context, m *wrapMocks) {
				m.req.EXPECT().BillingV1AccountIsValidResourceLimitByCustomerID(ctx, customerID, bmaccount.ResourceTypeAgent).Return(true, nil)
				m.util.EXPECT().EmailIsValid(username).Return(false)
			},

			expectReason: "INVALID_USERNAME_FORMAT",
		},
		{
			name: "username already exists",

			ringMethod: agent.RingMethodRingAll,
			mockFunc: func(ctx context.Context, m *wrapMocks) {
				m.req.EXPECT().BillingV1AccountIsValidResourceLimitByCustomerID(ctx, customerID, bmaccount.ResourceTypeAgent).Return(true, nil)
				m.util.EXPECT().EmailIsValid(username).Return(true)
				m.db.EXPECT().AgentGetByUsername(ctx, username).Return(&agent.Agent{Username: username}, nil)
			},

			expectReason: "AGENT_ALREADY_EXISTS",
		},
		{
			name: "existence check fails with a non not-found error",

			ringMethod: agent.RingMethodRingAll,
			mockFunc: func(ctx context.Context, m *wrapMocks) {
				m.req.EXPECT().BillingV1AccountIsValidResourceLimitByCustomerID(ctx, customerID, bmaccount.ResourceTypeAgent).Return(true, nil)
				m.util.EXPECT().EmailIsValid(username).Return(true)
				m.db.EXPECT().AgentGetByUsername(ctx, username).Return(nil, errBoom)
			},

			expectPrefix: "could not check agent existence: boom",
		},
		{
			name: "creation fails after the existence check",

			ringMethod: agent.RingMethod("linear"),
			mockFunc: func(ctx context.Context, m *wrapMocks) {
				m.req.EXPECT().BillingV1AccountIsValidResourceLimitByCustomerID(ctx, customerID, bmaccount.ResourceTypeAgent).Return(true, nil)
				m.util.EXPECT().EmailIsValid(username).Return(true)
				m.db.EXPECT().AgentGetByUsername(ctx, username).Return(nil, dbhandler.ErrNotFound)
			},

			expectReason: "INVALID_RING_METHOD",
			expectPrefix: "could not create an agent: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newWrapTestHandler(mc)
			ctx := context.Background()

			tt.mockFunc(ctx, m)

			res, err := h.Create(ctx, customerID, username, "password-for-test", "name", "detail", tt.ringMethod, agent.PermissionNone, nil, nil)
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: ok")
			}
			if res != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", res)
			}

			if tt.expectReason != "" {
				if got := reasonOf(t, err); got != tt.expectReason {
					t.Errorf("Wrong reason. expect: %s, got: %s", tt.expectReason, got)
				}
			}
			if tt.expectPrefix != "" && !strings.HasPrefix(err.Error(), tt.expectPrefix) {
				t.Errorf("Wrong error. expect prefix: %q, got: %q", tt.expectPrefix, err.Error())
			}
		})
	}
}

func Test_Delete_dbDeleteFailure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, m := newWrapTestHandler(mc)
	ctx := context.Background()

	id := uuid.FromStringOrNil("69434cfa-79a4-11ec-a7b1-6ba5b7016d83")
	a := &agent.Agent{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
		},
		Permission: agent.PermissionNone,
	}

	// the first lookup is the only-admin check, the second one is the delete itself.
	m.db.EXPECT().AgentGet(ctx, id).Return(a, nil).Times(2)
	m.db.EXPECT().AgentDelete(ctx, id).Return(stderrors.New("boom"))

	res, err := h.Delete(ctx, id)
	if err == nil {
		t.Fatalf("Wrong match. expect: error, got: ok")
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", res)
	}

	expectErr := "could not delete the agent: could not delete the agent: boom"
	if err.Error() != expectErr {
		t.Errorf("Wrong error.\nexpect: %s\ngot: %s", expectErr, err.Error())
	}
}

func Test_Login_cases(t *testing.T) {
	id := uuid.FromStringOrNil("ac810dc4-298c-11ee-984c-ebb7811c4114")
	stored := &agent.Agent{
		Identity:     commonidentity.Identity{ID: id},
		Username:     "login@example.com",
		PasswordHash: "stored-hash",
	}

	tests := []struct {
		name string

		mockFunc func(ctx context.Context, m *wrapMocks)

		expectRes *agent.Agent
		expectErr string
	}{
		{
			name: "success",

			mockFunc: func(ctx context.Context, m *wrapMocks) {
				m.db.EXPECT().AgentGetByUsername(ctx, stored.Username).Return(stored, nil)
				m.util.EXPECT().HashCheckPassword("password-for-test", "stored-hash").Return(true)
			},

			expectRes: stored,
		},
		{
			name: "unknown username",

			mockFunc: func(ctx context.Context, m *wrapMocks) {
				m.db.EXPECT().AgentGetByUsername(ctx, stored.Username).Return(nil, dbhandler.ErrNotFound)
			},

			expectErr: "could not logged in: no agent info",
		},
		{
			name: "wrong password",

			mockFunc: func(ctx context.Context, m *wrapMocks) {
				m.db.EXPECT().AgentGetByUsername(ctx, stored.Username).Return(stored, nil)
				m.util.EXPECT().HashCheckPassword("password-for-test", "stored-hash").Return(false)
			},

			expectErr: "could not logged in: wrong password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newWrapTestHandler(mc)
			ctx := context.Background()

			tt.mockFunc(ctx, m)

			res, err := h.Login(ctx, stored.Username, "password-for-test")
			if tt.expectErr != "" {
				if err == nil || err.Error() != tt.expectErr {
					t.Errorf("Wrong error.\nexpect: %s\ngot: %v", tt.expectErr, err)
				}
				if res != nil {
					t.Errorf("Wrong match. expect: nil, got: %v", res)
				}
				return
			}

			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res != tt.expectRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_Update_dbFailures(t *testing.T) {
	id := uuid.FromStringOrNil("69434cfa-79a4-11ec-a7b1-6ba5b7016d83")
	customerID := uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc")
	errBoom := stderrors.New("boom")
	nonAdmin := &agent.Agent{
		Identity:   commonidentity.Identity{ID: id, CustomerID: customerID},
		Permission: agent.PermissionNone,
	}

	tests := []struct {
		name string

		mockFunc func(ctx context.Context, m *wrapMocks)
		call     func(ctx context.Context, h *agentHandler) error

		expectErr string
	}{
		{
			name: "UpdateBasicInfo",

			mockFunc: func(ctx context.Context, m *wrapMocks) {
				m.db.EXPECT().AgentGet(ctx, id).Return(nil, errBoom)
			},
			call: func(ctx context.Context, h *agentHandler) error {
				_, err := h.UpdateBasicInfo(ctx, id, nil, nil, nil)
				return err
			},

			expectErr: "could not update the agent info: boom",
		},
		{
			name: "UpdatePassword",

			mockFunc: func(ctx context.Context, m *wrapMocks) {
				m.util.EXPECT().HashGenerate("password-for-test", defaultPasswordHashCost).Return("", errBoom)
			},
			call: func(ctx context.Context, h *agentHandler) error {
				_, err := h.UpdatePassword(ctx, id, "password-for-test")
				return err
			},

			expectErr: "could not update the agent's password: boom",
		},
		{
			name: "UpdatePermission wraps the raw failure again",

			mockFunc: func(ctx context.Context, m *wrapMocks) {
				m.db.EXPECT().AgentGet(ctx, id).Return(nonAdmin, nil)
				m.db.EXPECT().AgentSetPermission(ctx, id, agent.PermissionCustomerManager).Return(errBoom)
			},
			call: func(ctx context.Context, h *agentHandler) error {
				_, err := h.UpdatePermission(ctx, id, agent.PermissionCustomerManager)
				return err
			},

			expectErr: "could not update the agent permission: could not update the agent permission: boom",
		},
		{
			name: "UpdatePermissionRaw wraps once",

			mockFunc: func(ctx context.Context, m *wrapMocks) {
				m.db.EXPECT().AgentSetPermission(ctx, id, agent.PermissionCustomerManager).Return(errBoom)
			},
			call: func(ctx context.Context, h *agentHandler) error {
				_, err := h.UpdatePermissionRaw(ctx, id, agent.PermissionCustomerManager)
				return err
			},

			expectErr: "could not update the agent permission: boom",
		},
		{
			name: "UpdateTagIDs",

			mockFunc: func(ctx context.Context, m *wrapMocks) {
				m.db.EXPECT().AgentSetTagIDs(ctx, id, []uuid.UUID{}).Return(errBoom)
			},
			call: func(ctx context.Context, h *agentHandler) error {
				_, err := h.UpdateTagIDs(ctx, id, []uuid.UUID{})
				return err
			},

			expectErr: "could not update the tag ids: boom",
		},
		{
			name: "UpdateAddresses lookup of the agent",

			mockFunc: func(ctx context.Context, m *wrapMocks) {
				m.db.EXPECT().AgentGet(ctx, id).Return(nil, errBoom)
			},
			call: func(ctx context.Context, h *agentHandler) error {
				_, err := h.UpdateAddresses(ctx, id, nil)
				return err
			},

			expectErr: "could not get the agent: boom",
		},
		{
			name: "UpdateAddresses final update",

			mockFunc: func(ctx context.Context, m *wrapMocks) {
				m.db.EXPECT().AgentGet(ctx, id).Return(nonAdmin, nil)
				m.db.EXPECT().AgentSetAddresses(ctx, id, []commonaddress.Address{}).Return(errBoom)
			},
			call: func(ctx context.Context, h *agentHandler) error {
				_, err := h.UpdateAddresses(ctx, id, []commonaddress.Address{})
				return err
			},

			expectErr: "could not update the addresses: boom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newWrapTestHandler(mc)
			ctx := context.Background()

			tt.mockFunc(ctx, m)

			err := tt.call(ctx, h)
			if err == nil || err.Error() != tt.expectErr {
				t.Errorf("Wrong error.\nexpect: %s\ngot: %v", tt.expectErr, err)
			}
		})
	}
}

func Test_UpdateAddresses_validationErrors(t *testing.T) {
	id := uuid.FromStringOrNil("69434cfa-79a4-11ec-a7b1-6ba5b7016d83")
	customerID := uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc")
	otherCustomerID := uuid.FromStringOrNil("b1e1a2d4-7fe2-11ec-848d-97c8e986acfc")
	extensionID := uuid.FromStringOrNil("d6f3a0c0-7fe2-11ec-9a0b-5f3b0a2f1c11")
	errBoom := stderrors.New("boom")
	current := &agent.Agent{Identity: commonidentity.Identity{ID: id, CustomerID: customerID}}

	tests := []struct {
		name string

		addresses []commonaddress.Address
		mockFunc  func(ctx context.Context, m *wrapMocks)

		expectErr string
	}{
		{
			name: "extension lookup rpc failure",

			addresses: []commonaddress.Address{{Type: commonaddress.TypeExtension, Target: extensionID.String()}},
			mockFunc: func(ctx context.Context, m *wrapMocks) {
				m.req.EXPECT().RegistrarV1ExtensionGet(ctx, extensionID).Return(nil, errBoom)
			},

			expectErr: "could not get extension info: boom",
		},
		{
			name: "extension belongs to another customer",

			addresses: []commonaddress.Address{{Type: commonaddress.TypeExtension, Target: extensionID.String()}},
			mockFunc: func(ctx context.Context, m *wrapMocks) {
				m.req.EXPECT().RegistrarV1ExtensionGet(ctx, extensionID).Return(&rmextension.Extension{
					Identity: commonidentity.Identity{ID: extensionID, CustomerID: otherCustomerID},
				}, nil)
			},

			expectErr: "wrong customer info",
		},
		{
			name: "empty sip target",

			addresses: []commonaddress.Address{{Type: commonaddress.TypeSIP, Target: ""}},
			mockFunc:  func(ctx context.Context, m *wrapMocks) {},

			expectErr: "invalid target",
		},
		{
			name: "address lookup fails with a non not-found error",

			addresses: []commonaddress.Address{{Type: commonaddress.TypeSIP, Target: "agent-a@sip.example.com"}},
			mockFunc: func(ctx context.Context, m *wrapMocks) {
				m.db.EXPECT().AgentGetByCustomerIDAndAddress(ctx, customerID, gomock.Any()).Return(nil, errBoom)
			},

			expectErr: "could not get agent info of the address: boom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newWrapTestHandler(mc)
			ctx := context.Background()

			m.db.EXPECT().AgentGet(ctx, id).Return(current, nil)
			tt.mockFunc(ctx, m)

			res, err := h.UpdateAddresses(ctx, id, tt.addresses)
			if err == nil || err.Error() != tt.expectErr {
				t.Errorf("Wrong error.\nexpect: %s\ngot: %v", tt.expectErr, err)
			}
			if res != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", res)
			}
		})
	}
}

func Test_PasswordReset_postUpdateFailures(t *testing.T) {
	token := "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"
	password := "newpassword123"
	agentID := uuid.FromStringOrNil("ac810dc4-298c-11ee-984c-ebb7811c4114")
	updated := &agent.Agent{
		Identity: commonidentity.Identity{
			ID:         agentID,
			CustomerID: uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
		},
		Username: "reset@example.com",
	}

	tests := []struct {
		name string

		mockFunc func(ctx context.Context, m *wrapMocks)
	}{
		{
			name: "token deletion fails but the agent is still published",

			mockFunc: func(ctx context.Context, m *wrapMocks) {
				m.cache.EXPECT().PasswordResetTokenDelete(ctx, token).Return(stderrors.New("boom"))
				m.db.EXPECT().AgentGet(ctx, agentID).Return(updated, nil)
				m.notify.EXPECT().PublishEvent(ctx, agent.EventTypeAgentUpdated, updated)
			},
		},
		{
			name: "reloading the agent fails and nothing is published",

			mockFunc: func(ctx context.Context, m *wrapMocks) {
				m.cache.EXPECT().PasswordResetTokenDelete(ctx, token).Return(nil)
				m.db.EXPECT().AgentGet(ctx, agentID).Return(nil, stderrors.New("boom"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newWrapTestHandler(mc)
			ctx := context.Background()

			m.cache.EXPECT().PasswordResetTokenGet(ctx, token).Return(agentID, nil)
			m.util.EXPECT().HashGenerate(password, defaultPasswordHashCost).Return("hashed_password", nil)
			m.db.EXPECT().AgentSetPasswordHash(ctx, agentID, "hashed_password").Return(nil)
			tt.mockFunc(ctx, m)

			if err := h.PasswordReset(ctx, token, password); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
