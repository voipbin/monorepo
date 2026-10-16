package agenthandler

import (
	"context"
	stderrors "errors"
	"fmt"
	"testing"
	"time"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-agent-manager/pkg/dbhandler"
)

const testOnlyAdminPageSize = 1000

var (
	onlyAdminCustomerID = uuid.FromStringOrNil("c1a2b3d4-1607-4000-8000-000000000001")
	onlyAdminTargetID   = uuid.FromStringOrNil("c1a2b3d4-1607-4000-8000-0000000000aa")
	onlyAdminOtherID    = uuid.FromStringOrNil("c1a2b3d4-1607-4000-8000-0000000000bb")
	onlyAdminBaseTime   = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
)

// onlyAdminAgents returns count agents of perm, TMCreate descending by one second from start.
func onlyAdminAgents(count int, perm agent.Permission, start time.Time) []*agent.Agent {
	res := make([]*agent.Agent, 0, count)
	for i := 0; i < count; i++ {
		tm := start.Add(-time.Duration(i) * time.Second)
		res = append(res, &agent.Agent{
			Identity: commonidentity.Identity{
				ID:         uuid.Must(uuid.NewV4()),
				CustomerID: onlyAdminCustomerID,
			},
			Permission: perm,
			TMCreate:   &tm,
		})
	}
	return res
}

func onlyAdminFilters() map[agent.Field]any {
	return map[agent.Field]any{
		agent.FieldCustomerID: onlyAdminCustomerID,
		agent.FieldDeleted:    false,
	}
}

func onlyAdminTarget() *agent.Agent {
	tm := onlyAdminBaseTime.Add(-5000 * time.Second)
	return &agent.Agent{
		Identity: commonidentity.Identity{
			ID:         onlyAdminTargetID,
			CustomerID: onlyAdminCustomerID,
		},
		Permission: agent.PermissionCustomerAdmin,
		TMCreate:   &tm,
	}
}

func onlyAdminOtherAdmin() *agent.Agent {
	tm := onlyAdminBaseTime.Add(-6000 * time.Second)
	return &agent.Agent{
		Identity: commonidentity.Identity{
			ID:         onlyAdminOtherID,
			CustomerID: onlyAdminCustomerID,
		},
		Permission: agent.PermissionCustomerAdmin,
		TMCreate:   &tm,
	}
}

// onlyAdminSecondPageToken is page 1's last TMCreate: base - 999s, in ISO 8601 UTC with microseconds.
const onlyAdminSecondPageToken = "2025-12-31T23:43:21.000000Z"

func newOnlyAdminHandler(mc *gomock.Controller) (*agentHandler, *dbhandler.MockDBHandler, *requesthandler.MockRequestHandler, *notifyhandler.MockNotifyHandler) {
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	h := &agentHandler{
		reqHandler:    mockReq,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	return h, mockDB, mockReq, mockNotify
}

func requireLastAdmin(t *testing.T, err error) {
	t.Helper()

	var ve *cerrors.VoipbinError
	if !stderrors.As(err, &ve) {
		t.Fatalf("Wrong match. expect: VoipbinError, got: %v", err)
	}
	if ve.Status != cerrors.StatusFailedPrecondition || ve.Reason != "AGENT_LAST_ADMIN" {
		t.Errorf("Wrong match. expect: FailedPrecondition/AGENT_LAST_ADMIN, got: %v/%s", ve.Status, ve.Reason)
	}
}

func Test_Delete_otherAdminOnLaterPage(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockDB, _, mockNotify := newOnlyAdminHandler(mc)
	ctx := context.Background()

	target := onlyAdminTarget()
	page1 := onlyAdminAgents(testOnlyAdminPageSize, agent.PermissionCustomerAgent, onlyAdminBaseTime)
	page2 := []*agent.Agent{target, onlyAdminOtherAdmin()}

	mockDB.EXPECT().AgentGet(ctx, onlyAdminTargetID).Return(target, nil)
	gomock.InOrder(
		mockDB.EXPECT().AgentList(ctx, uint64(testOnlyAdminPageSize), "", onlyAdminFilters()).Return(page1, nil),
		mockDB.EXPECT().AgentList(ctx, uint64(testOnlyAdminPageSize), onlyAdminSecondPageToken, onlyAdminFilters()).Return(page2, nil),
	)

	// dbDelete
	mockDB.EXPECT().AgentGet(ctx, onlyAdminTargetID).Return(target, nil).Times(2)
	mockDB.EXPECT().AgentDelete(ctx, onlyAdminTargetID).Return(nil)
	mockNotify.EXPECT().PublishWebhookEvent(ctx, onlyAdminCustomerID, agent.EventTypeAgentDeleted, target)

	if _, err := h.Delete(ctx, onlyAdminTargetID); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}

func Test_UpdatePermission_otherAdminOnLaterPage(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockDB, _, mockNotify := newOnlyAdminHandler(mc)
	ctx := context.Background()

	target := onlyAdminTarget()
	page1 := onlyAdminAgents(testOnlyAdminPageSize, agent.PermissionCustomerAgent, onlyAdminBaseTime)
	page2 := []*agent.Agent{target, onlyAdminOtherAdmin()}

	mockDB.EXPECT().AgentGet(ctx, onlyAdminTargetID).Return(target, nil)
	gomock.InOrder(
		mockDB.EXPECT().AgentList(ctx, uint64(testOnlyAdminPageSize), "", onlyAdminFilters()).Return(page1, nil),
		mockDB.EXPECT().AgentList(ctx, uint64(testOnlyAdminPageSize), onlyAdminSecondPageToken, onlyAdminFilters()).Return(page2, nil),
	)

	// UpdatePermissionRaw
	mockDB.EXPECT().AgentSetPermission(ctx, onlyAdminTargetID, agent.PermissionCustomerManager).Return(nil)
	mockDB.EXPECT().AgentGet(ctx, onlyAdminTargetID).Return(target, nil)
	mockNotify.EXPECT().PublishWebhookEvent(ctx, onlyAdminCustomerID, agent.EventTypeAgentUpdated, target)

	if _, err := h.UpdatePermission(ctx, onlyAdminTargetID, agent.PermissionCustomerManager); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}

func Test_Delete_onlyAdminTyped(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockDB, _, _ := newOnlyAdminHandler(mc)
	ctx := context.Background()

	target := onlyAdminTarget()
	mockDB.EXPECT().AgentGet(ctx, onlyAdminTargetID).Return(target, nil)
	mockDB.EXPECT().AgentList(ctx, uint64(testOnlyAdminPageSize), "", onlyAdminFilters()).Return([]*agent.Agent{target}, nil)

	_, err := h.Delete(ctx, onlyAdminTargetID)
	requireLastAdmin(t, err)
}

func Test_UpdatePermission_onlyAdminTyped(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockDB, _, _ := newOnlyAdminHandler(mc)
	ctx := context.Background()

	target := onlyAdminTarget()
	mockDB.EXPECT().AgentGet(ctx, onlyAdminTargetID).Return(target, nil)
	mockDB.EXPECT().AgentList(ctx, uint64(testOnlyAdminPageSize), "", onlyAdminFilters()).Return([]*agent.Agent{target}, nil)

	_, err := h.UpdatePermission(ctx, onlyAdminTargetID, agent.PermissionNone)
	requireLastAdmin(t, err)
}

func Test_onlyAdminLookupErrors(t *testing.T) {
	errList := fmt.Errorf("db connection lost")

	tests := []struct {
		name string

		mockFunc func(mockDB *dbhandler.MockDBHandler, ctx context.Context)

		expectNotFound bool
		expectCause    error
	}{
		{
			name: "list error is denied and keeps the cause",

			mockFunc: func(mockDB *dbhandler.MockDBHandler, ctx context.Context) {
				mockDB.EXPECT().AgentGet(ctx, onlyAdminTargetID).Return(onlyAdminTarget(), nil)
				mockDB.EXPECT().AgentList(ctx, uint64(testOnlyAdminPageSize), "", onlyAdminFilters()).Return(nil, errList)
			},
			expectCause: errList,
		},
		{
			name: "agent not found keeps the typed 404",

			mockFunc: func(mockDB *dbhandler.MockDBHandler, ctx context.Context) {
				mockDB.EXPECT().AgentGet(ctx, onlyAdminTargetID).Return(nil, dbhandler.ErrNotFound)
			},
			expectNotFound: true,
		},
	}

	for _, tt := range tests {
		for _, op := range []string{"delete", "permission"} {
			t.Run(tt.name+" "+op, func(t *testing.T) {
				mc := gomock.NewController(t)
				defer mc.Finish()

				h, mockDB, _, _ := newOnlyAdminHandler(mc)
				ctx := context.Background()
				tt.mockFunc(mockDB, ctx)

				var err error
				if op == "delete" {
					_, err = h.Delete(ctx, onlyAdminTargetID)
				} else {
					_, err = h.UpdatePermission(ctx, onlyAdminTargetID, agent.PermissionNone)
				}
				if err == nil {
					t.Fatalf("Wrong match. expect: error, got: nil")
				}

				var ve *cerrors.VoipbinError
				isTyped := stderrors.As(err, &ve)
				if tt.expectNotFound {
					if !isTyped || ve.Status != cerrors.StatusNotFound || ve.Reason != "AGENT_NOT_FOUND" {
						t.Errorf("Wrong match. expect: NotFound/AGENT_NOT_FOUND, got: %v", err)
					}
					return
				}

				if isTyped && ve.Reason == "AGENT_LAST_ADMIN" {
					t.Errorf("Wrong match. a lookup failure must not be reported as the last admin: %v", err)
				}
				if !stderrors.Is(err, tt.expectCause) {
					t.Errorf("Wrong match. expect cause: %v, got: %v", tt.expectCause, err)
				}
			})
		}
	}
}

func Test_UpdatePermission_guestTyped(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, _, _, _ := newOnlyAdminHandler(mc)

	_, err := h.UpdatePermission(context.Background(), agent.GuestAgentID, agent.PermissionNone)

	var ve *cerrors.VoipbinError
	if !stderrors.As(err, &ve) || ve.Status != cerrors.StatusInvalidArgument || ve.Reason != "AGENT_GUEST_IMMUTABLE" {
		t.Errorf("Wrong match. expect: InvalidArgument/AGENT_GUEST_IMMUTABLE, got: %v", err)
	}
}

func Test_Delete_guestTyped(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, _, _, _ := newOnlyAdminHandler(mc)

	_, err := h.Delete(context.Background(), agent.GuestAgentID)

	var ve *cerrors.VoipbinError
	if !stderrors.As(err, &ve) || ve.Status != cerrors.StatusInvalidArgument || ve.Reason != "AGENT_GUEST_IMMUTABLE" {
		t.Errorf("Wrong match. expect: InvalidArgument/AGENT_GUEST_IMMUTABLE, got: %v", err)
	}
}

func Test_UpdatePermission_customerAdminSkipsOnlyAdminCheck(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockDB, _, mockNotify := newOnlyAdminHandler(mc)
	ctx := context.Background()

	target := onlyAdminTarget()

	// no AgentGet for the check and no AgentList: only UpdatePermissionRaw runs
	mockDB.EXPECT().AgentSetPermission(ctx, onlyAdminTargetID, agent.PermissionCustomerAdmin).Return(nil)
	mockDB.EXPECT().AgentGet(ctx, onlyAdminTargetID).Return(target, nil)
	mockNotify.EXPECT().PublishWebhookEvent(ctx, onlyAdminCustomerID, agent.EventTypeAgentUpdated, target)

	if _, err := h.UpdatePermission(ctx, onlyAdminTargetID, agent.PermissionCustomerAdmin); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}

func Test_Delete_onlyProjectSuperAdminTyped(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockDB, _, _ := newOnlyAdminHandler(mc)
	ctx := context.Background()

	target := onlyAdminTarget()
	target.Permission = agent.PermissionProjectSuperAdmin
	mockDB.EXPECT().AgentGet(ctx, onlyAdminTargetID).Return(target, nil)
	mockDB.EXPECT().AgentList(ctx, uint64(testOnlyAdminPageSize), "", onlyAdminFilters()).Return([]*agent.Agent{target}, nil)

	_, err := h.Delete(ctx, onlyAdminTargetID)
	requireLastAdmin(t, err)
}
