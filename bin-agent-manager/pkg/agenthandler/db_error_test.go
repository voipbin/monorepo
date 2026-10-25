package agenthandler

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	dmdirect "monorepo/bin-direct-manager/models/direct"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-agent-manager/pkg/dbhandler"
)

// errMocks bundles the mocks of an agentHandler built for an error-path test.
// None of the mocks has a default expectation: every call must be declared.
type errMocks struct {
	req    *requesthandler.MockRequestHandler
	db     *dbhandler.MockDBHandler
	notify *notifyhandler.MockNotifyHandler
	util   *utilhandler.MockUtilHandler
}

func newErrTestHandler(mc *gomock.Controller) (*agentHandler, *errMocks) {
	m := &errMocks{
		req:    requesthandler.NewMockRequestHandler(mc),
		db:     dbhandler.NewMockDBHandler(mc),
		notify: notifyhandler.NewMockNotifyHandler(mc),
		util:   utilhandler.NewMockUtilHandler(mc),
	}
	h := &agentHandler{
		reqHandler:    m.req,
		db:            m.db,
		notifyHandler: m.notify,
		utilHandler:   m.util,
	}
	return h, m
}

func assertAddressAssignedError(t *testing.T, err error) {
	t.Helper()

	var ve *cerrors.VoipbinError
	if !errors.As(err, &ve) {
		t.Fatalf("Wrong error type. expect: VoipbinError, got: %v", err)
	}
	if ve.Reason != "ADDRESS_ALREADY_ASSIGNED" {
		t.Errorf("Wrong reason. expect: ADDRESS_ALREADY_ASSIGNED, got: %s", ve.Reason)
	}
}

func Test_dbCreate_errors(t *testing.T) {
	customerID := uuid.FromStringOrNil("5fd7f9b8-cb37-11ee-bd29-f30560a6ac86")
	agentID := uuid.FromStringOrNil("a1a1a1a1-2222-2222-2222-222222222222")
	directID := uuid.FromStringOrNil("d1d1d1d1-2222-2222-2222-222222222222")
	direct := &dmdirect.Direct{Identity: commonidentity.Identity{ID: directID}, Hash: "test-direct-hash"}

	errBoom := fmt.Errorf("boom")

	tests := []struct {
		name string

		ringMethod agent.RingMethod
		setup      func(ctx context.Context, m *errMocks)

		expectErr func(t *testing.T, err error)
	}{
		{
			name:       "unsupported ring method is rejected before any other call",
			ringMethod: agent.RingMethodLinear,
			setup:      func(ctx context.Context, m *errMocks) {},
			expectErr: func(t *testing.T, err error) {
				var ve *cerrors.VoipbinError
				if !errors.As(err, &ve) || ve.Reason != "INVALID_RING_METHOD" {
					t.Errorf("Wrong error. expect: INVALID_RING_METHOD, got: %v", err)
				}
			},
		},
		{
			name:       "hash generation failure is returned",
			ringMethod: agent.RingMethodRingAll,
			setup: func(ctx context.Context, m *errMocks) {
				m.util.EXPECT().HashGenerate("test-password", defaultPasswordHashCost).Return("", errBoom)
			},
			expectErr: func(t *testing.T, err error) {
				if !errors.Is(err, errBoom) {
					t.Errorf("Wrong error. expect: %v, got: %v", errBoom, err)
				}
			},
		},
		{
			name:       "direct creation failure is wrapped and nothing is written",
			ringMethod: agent.RingMethodRingAll,
			setup: func(ctx context.Context, m *errMocks) {
				m.util.EXPECT().HashGenerate("test-password", defaultPasswordHashCost).Return("test-hash", nil)
				m.util.EXPECT().UUIDCreate().Return(agentID)
				m.req.EXPECT().DirectV1DirectCreate(ctx, customerID, dmdirect.ResourceTypeAgent, agentID).Return(nil, errBoom)
			},
			expectErr: func(t *testing.T, err error) {
				if !errors.Is(err, errBoom) || !strings.Contains(err.Error(), "could not create direct hash") {
					t.Errorf("Wrong error. expect: wrapped direct create error, got: %v", err)
				}
			},
		},
		{
			name:       "agent creation failure deletes the orphaned direct and keeps the original error",
			ringMethod: agent.RingMethodRingAll,
			setup: func(ctx context.Context, m *errMocks) {
				m.util.EXPECT().HashGenerate("test-password", defaultPasswordHashCost).Return("test-hash", nil)
				m.util.EXPECT().UUIDCreate().Return(agentID)
				m.req.EXPECT().DirectV1DirectCreate(ctx, customerID, dmdirect.ResourceTypeAgent, agentID).Return(direct, nil)
				m.db.EXPECT().AgentCreate(ctx, gomock.Any()).Return(errBoom)
				m.req.EXPECT().DirectV1DirectDelete(ctx, directID).Return(nil, fmt.Errorf("cleanup failed"))
			},
			expectErr: func(t *testing.T, err error) {
				if !errors.Is(err, errBoom) {
					t.Errorf("Wrong error. expect: %v, got: %v", errBoom, err)
				}
			},
		},
		{
			name:       "duplicate address is a typed already-exists error and the direct is deleted",
			ringMethod: agent.RingMethodRingAll,
			setup: func(ctx context.Context, m *errMocks) {
				m.util.EXPECT().HashGenerate("test-password", defaultPasswordHashCost).Return("test-hash", nil)
				m.util.EXPECT().UUIDCreate().Return(agentID)
				m.req.EXPECT().DirectV1DirectCreate(ctx, customerID, dmdirect.ResourceTypeAgent, agentID).Return(direct, nil)
				m.db.EXPECT().AgentCreate(ctx, gomock.Any()).Return(fmt.Errorf("insert: %w", dbhandler.ErrAlreadyExists))
				m.req.EXPECT().DirectV1DirectDelete(ctx, directID).Return(direct, nil)
			},
			expectErr: assertAddressAssignedError,
		},
		{
			name:       "fetching the created agent fails",
			ringMethod: agent.RingMethodRingAll,
			setup: func(ctx context.Context, m *errMocks) {
				m.util.EXPECT().HashGenerate("test-password", defaultPasswordHashCost).Return("test-hash", nil)
				m.util.EXPECT().UUIDCreate().Return(agentID)
				m.req.EXPECT().DirectV1DirectCreate(ctx, customerID, dmdirect.ResourceTypeAgent, agentID).Return(direct, nil)
				m.db.EXPECT().AgentCreate(ctx, gomock.Any()).Return(nil)
				m.db.EXPECT().AgentGet(ctx, agentID).Return(nil, errBoom)
			},
			expectErr: func(t *testing.T, err error) {
				if !errors.Is(err, errBoom) {
					t.Errorf("Wrong error. expect: %v, got: %v", errBoom, err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newErrTestHandler(mc)
			ctx := context.Background()

			tt.setup(ctx, m)

			res, err := h.dbCreate(ctx, customerID, "test-user", "test-password", "test name", "test detail", tt.ringMethod, agent.PermissionNone, []uuid.UUID{}, []commonaddress.Address{})
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: ok")
			}
			if res != nil {
				t.Errorf("Wrong result. expect: nil, got: %v", res)
			}
			tt.expectErr(t, err)
		})
	}
}

func Test_dbDelete_errors(t *testing.T) {
	id := uuid.FromStringOrNil("a1a1a1a1-3333-3333-3333-333333333333")
	customerID := uuid.FromStringOrNil("c1c1c1c1-3333-3333-3333-333333333333")
	directID := uuid.FromStringOrNil("d1d1d1d1-3333-3333-3333-333333333333")

	errBoom := fmt.Errorf("boom")
	withDirect := &agent.Agent{Identity: commonidentity.Identity{ID: id, CustomerID: customerID}, DirectID: directID}
	withoutDirect := &agent.Agent{Identity: commonidentity.Identity{ID: id, CustomerID: customerID}}

	tests := []struct {
		name string

		setup func(ctx context.Context, m *errMocks)

		expectErr error
		// expectResult is the agent returned when the direct deletion fails but the deletion succeeds.
		expectResult *agent.Agent
	}{
		{
			name: "initial lookup failure stops before any write",
			setup: func(ctx context.Context, m *errMocks) {
				m.db.EXPECT().AgentGet(ctx, id).Return(nil, errBoom)
			},
			expectErr: errBoom,
		},
		{
			name: "agent deletion failure is returned",
			setup: func(ctx context.Context, m *errMocks) {
				m.db.EXPECT().AgentGet(ctx, id).Return(withoutDirect, nil)
				m.db.EXPECT().AgentDelete(ctx, id).Return(errBoom)
			},
			expectErr: errBoom,
		},
		{
			name: "lookup after deletion failure is returned without an event",
			setup: func(ctx context.Context, m *errMocks) {
				gomock.InOrder(
					m.db.EXPECT().AgentGet(ctx, id).Return(withoutDirect, nil),
					m.db.EXPECT().AgentDelete(ctx, id).Return(nil),
					m.db.EXPECT().AgentGet(ctx, id).Return(nil, errBoom),
				)
			},
			expectErr: errBoom,
		},
		{
			name: "direct deletion failure does not block the agent deletion",
			setup: func(ctx context.Context, m *errMocks) {
				gomock.InOrder(
					m.db.EXPECT().AgentGet(ctx, id).Return(withDirect, nil),
					m.req.EXPECT().DirectV1DirectDelete(ctx, directID).Return(nil, errBoom),
					m.db.EXPECT().AgentDelete(ctx, id).Return(nil),
					m.db.EXPECT().AgentGet(ctx, id).Return(withDirect, nil),
					m.notify.EXPECT().PublishWebhookEvent(ctx, customerID, agent.EventTypeAgentDeleted, withDirect),
				)
			},
			expectResult: withDirect,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newErrTestHandler(mc)
			ctx := context.Background()

			tt.setup(ctx, m)

			res, err := h.dbDelete(ctx, id)
			if tt.expectErr != nil {
				if !errors.Is(err, tt.expectErr) {
					t.Errorf("Wrong error. expect: %v, got: %v", tt.expectErr, err)
				}
				if res != nil {
					t.Errorf("Wrong result. expect: nil, got: %v", res)
				}
				return
			}

			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res != tt.expectResult {
				t.Errorf("Wrong result. expect: %v, got: %v", tt.expectResult, res)
			}
		})
	}
}

func Test_dbLogin_errors(t *testing.T) {
	tests := []struct {
		name string

		setup func(ctx context.Context, m *errMocks)

		expectMessage string
	}{
		{
			name: "unknown username",
			setup: func(ctx context.Context, m *errMocks) {
				m.db.EXPECT().AgentGetByUsername(ctx, "test-user").Return(nil, dbhandler.ErrNotFound)
			},
			expectMessage: "no agent info",
		},
		{
			name: "wrong password",
			setup: func(ctx context.Context, m *errMocks) {
				m.db.EXPECT().AgentGetByUsername(ctx, "test-user").Return(&agent.Agent{PasswordHash: "test-hash"}, nil)
				m.util.EXPECT().HashCheckPassword("test-password", "test-hash").Return(false)
			},
			expectMessage: "wrong password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newErrTestHandler(mc)
			ctx := context.Background()

			tt.setup(ctx, m)

			res, err := h.dbLogin(ctx, "test-user", "test-password")
			if err == nil || err.Error() != tt.expectMessage {
				t.Errorf("Wrong error. expect: %s, got: %v", tt.expectMessage, err)
			}
			if res != nil {
				t.Errorf("Wrong result. expect: nil, got: %v", res)
			}
		})
	}
}

func Test_dbUpdate_errors(t *testing.T) {
	id := uuid.FromStringOrNil("a1a1a1a1-4444-4444-4444-444444444444")
	errBoom := fmt.Errorf("boom")
	name := "test name"

	// each case drives one dbUpdateXxx through one failing dependency; the mock has
	// no other expectation, so a webhook or event publication would fail the test.
	tests := []struct {
		name string

		call  func(ctx context.Context, h *agentHandler) (*agent.Agent, error)
		setup func(ctx context.Context, m *errMocks)

		// expectIs is matched with errors.Is, expectAssigned expects the typed address conflict.
		expectIs       error
		expectAssigned bool
	}{
		{
			name: "info without changes: lookup failure",
			call: func(ctx context.Context, h *agentHandler) (*agent.Agent, error) {
				return h.dbUpdateInfo(ctx, id, nil, nil, nil)
			},
			setup: func(ctx context.Context, m *errMocks) {
				m.db.EXPECT().AgentGet(ctx, id).Return(nil, errBoom)
			},
			expectIs: errBoom,
		},
		{
			name: "info: update failure",
			call: func(ctx context.Context, h *agentHandler) (*agent.Agent, error) {
				return h.dbUpdateInfo(ctx, id, &name, nil, nil)
			},
			setup: func(ctx context.Context, m *errMocks) {
				m.db.EXPECT().AgentUpdate(ctx, id, map[agent.Field]any{agent.FieldName: name}).Return(errBoom)
			},
			expectIs: errBoom,
		},
		{
			name: "info: lookup after update failure",
			call: func(ctx context.Context, h *agentHandler) (*agent.Agent, error) {
				return h.dbUpdateInfo(ctx, id, &name, nil, nil)
			},
			setup: func(ctx context.Context, m *errMocks) {
				m.db.EXPECT().AgentUpdate(ctx, id, map[agent.Field]any{agent.FieldName: name}).Return(nil)
				m.db.EXPECT().AgentGet(ctx, id).Return(nil, errBoom)
			},
			expectIs: errBoom,
		},
		{
			name: "password: hash generation failure",
			call: func(ctx context.Context, h *agentHandler) (*agent.Agent, error) {
				return h.dbUpdatePassword(ctx, id, "test-password")
			},
			setup: func(ctx context.Context, m *errMocks) {
				m.util.EXPECT().HashGenerate("test-password", defaultPasswordHashCost).Return("", errBoom)
			},
			expectIs: errBoom,
		},
		{
			name: "password: store failure",
			call: func(ctx context.Context, h *agentHandler) (*agent.Agent, error) {
				return h.dbUpdatePassword(ctx, id, "test-password")
			},
			setup: func(ctx context.Context, m *errMocks) {
				m.util.EXPECT().HashGenerate("test-password", defaultPasswordHashCost).Return("test-hash", nil)
				m.db.EXPECT().AgentSetPasswordHash(ctx, id, "test-hash").Return(errBoom)
			},
			expectIs: errBoom,
		},
		{
			name: "password: lookup after store failure",
			call: func(ctx context.Context, h *agentHandler) (*agent.Agent, error) {
				return h.dbUpdatePassword(ctx, id, "test-password")
			},
			setup: func(ctx context.Context, m *errMocks) {
				m.util.EXPECT().HashGenerate("test-password", defaultPasswordHashCost).Return("test-hash", nil)
				m.db.EXPECT().AgentSetPasswordHash(ctx, id, "test-hash").Return(nil)
				m.db.EXPECT().AgentGet(ctx, id).Return(nil, errBoom)
			},
			expectIs: errBoom,
		},
		{
			name: "permission: store failure",
			call: func(ctx context.Context, h *agentHandler) (*agent.Agent, error) {
				return h.dbUpdatePermission(ctx, id, agent.PermissionCustomerAdmin)
			},
			setup: func(ctx context.Context, m *errMocks) {
				m.db.EXPECT().AgentSetPermission(ctx, id, agent.PermissionCustomerAdmin).Return(errBoom)
			},
			expectIs: errBoom,
		},
		{
			name: "permission: lookup after store failure",
			call: func(ctx context.Context, h *agentHandler) (*agent.Agent, error) {
				return h.dbUpdatePermission(ctx, id, agent.PermissionCustomerAdmin)
			},
			setup: func(ctx context.Context, m *errMocks) {
				m.db.EXPECT().AgentSetPermission(ctx, id, agent.PermissionCustomerAdmin).Return(nil)
				m.db.EXPECT().AgentGet(ctx, id).Return(nil, errBoom)
			},
			expectIs: errBoom,
		},
		{
			name: "tag ids: store failure",
			call: func(ctx context.Context, h *agentHandler) (*agent.Agent, error) {
				return h.dbUpdateTagIDs(ctx, id, []uuid.UUID{id})
			},
			setup: func(ctx context.Context, m *errMocks) {
				m.db.EXPECT().AgentSetTagIDs(ctx, id, []uuid.UUID{id}).Return(errBoom)
			},
			expectIs: errBoom,
		},
		{
			name: "tag ids: lookup after store failure",
			call: func(ctx context.Context, h *agentHandler) (*agent.Agent, error) {
				return h.dbUpdateTagIDs(ctx, id, []uuid.UUID{id})
			},
			setup: func(ctx context.Context, m *errMocks) {
				m.db.EXPECT().AgentSetTagIDs(ctx, id, []uuid.UUID{id}).Return(nil)
				m.db.EXPECT().AgentGet(ctx, id).Return(nil, errBoom)
			},
			expectIs: errBoom,
		},
		{
			name: "addresses: duplicate address is a typed conflict",
			call: func(ctx context.Context, h *agentHandler) (*agent.Agent, error) {
				return h.dbUpdateAddresses(ctx, id, []commonaddress.Address{})
			},
			setup: func(ctx context.Context, m *errMocks) {
				m.db.EXPECT().AgentSetAddresses(ctx, id, []commonaddress.Address{}).Return(fmt.Errorf("update: %w", dbhandler.ErrAlreadyExists))
			},
			expectAssigned: true,
		},
		{
			name: "addresses: other store failure is returned as is",
			call: func(ctx context.Context, h *agentHandler) (*agent.Agent, error) {
				return h.dbUpdateAddresses(ctx, id, []commonaddress.Address{})
			},
			setup: func(ctx context.Context, m *errMocks) {
				m.db.EXPECT().AgentSetAddresses(ctx, id, []commonaddress.Address{}).Return(errBoom)
			},
			expectIs: errBoom,
		},
		{
			name: "addresses: lookup after store failure",
			call: func(ctx context.Context, h *agentHandler) (*agent.Agent, error) {
				return h.dbUpdateAddresses(ctx, id, []commonaddress.Address{})
			},
			setup: func(ctx context.Context, m *errMocks) {
				m.db.EXPECT().AgentSetAddresses(ctx, id, []commonaddress.Address{}).Return(nil)
				m.db.EXPECT().AgentGet(ctx, id).Return(nil, errBoom)
			},
			expectIs: errBoom,
		},
		{
			name: "status: store failure",
			call: func(ctx context.Context, h *agentHandler) (*agent.Agent, error) {
				return h.dbUpdateStatus(ctx, id, agent.StatusBusy)
			},
			setup: func(ctx context.Context, m *errMocks) {
				m.db.EXPECT().AgentSetStatus(ctx, id, agent.StatusBusy).Return(errBoom)
			},
			expectIs: errBoom,
		},
		{
			name: "status: lookup after store failure",
			call: func(ctx context.Context, h *agentHandler) (*agent.Agent, error) {
				return h.dbUpdateStatus(ctx, id, agent.StatusBusy)
			},
			setup: func(ctx context.Context, m *errMocks) {
				m.db.EXPECT().AgentSetStatus(ctx, id, agent.StatusBusy).Return(nil)
				m.db.EXPECT().AgentGet(ctx, id).Return(nil, errBoom)
			},
			expectIs: errBoom,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newErrTestHandler(mc)
			ctx := context.Background()

			tt.setup(ctx, m)

			res, err := tt.call(ctx, h)
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: ok")
			}
			if res != nil {
				t.Errorf("Wrong result. expect: nil, got: %v", res)
			}
			if tt.expectAssigned {
				assertAddressAssignedError(t, err)
				return
			}
			if !errors.Is(err, tt.expectIs) {
				t.Errorf("Wrong error. expect: %v, got: %v", tt.expectIs, err)
			}
		})
	}
}
