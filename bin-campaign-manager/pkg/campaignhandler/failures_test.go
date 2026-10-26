package campaignhandler

import (
	"context"
	stderrors "errors"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-campaign-manager/models/campaign"
	"monorepo/bin-campaign-manager/pkg/campaigncallhandler"
	"monorepo/bin-campaign-manager/pkg/dbhandler"
	"monorepo/bin-campaign-manager/pkg/outplanhandler"
)

func Test_NewCampaignHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockCampaigncall := campaigncallhandler.NewMockCampaigncallHandler(mc)
	mockOutplan := outplanhandler.NewMockOutplanHandler(mc)

	res := NewCampaignHandler(mockDB, mockReq, mockNotify, mockCampaigncall, mockOutplan)

	h, ok := res.(*campaignHandler)
	if !ok {
		t.Fatalf("Wrong match. expect: *campaignHandler, got: %T", res)
	}
	if h.db != mockDB || h.reqHandler != mockReq || h.notifyHandler != mockNotify ||
		h.campaigncallHandler != mockCampaigncall || h.outplanHandler != mockOutplan {
		t.Errorf("Wrong match. the given dependencies must be kept. handler: %+v", h)
	}
	if h.util == nil {
		t.Errorf("Wrong match. expect: util handler is set, got: nil")
	}
}

func Test_List_dbErrors(t *testing.T) {
	customerID := uuid.FromStringOrNil("c1000000-0000-4000-8000-000000000001")
	errDB := stderrors.New("db unavailable")

	tests := []struct {
		name string
		fn   func(ctx context.Context, h *campaignHandler) ([]*campaign.Campaign, error)
		set  func(ctx context.Context, m *executeMocks)
	}{
		{
			name: "list",
			fn: func(ctx context.Context, h *campaignHandler) ([]*campaign.Campaign, error) {
				return h.List(ctx, "2026-01-01 00:00:00.000000", 10, map[campaign.Field]any{campaign.FieldCustomerID: customerID})
			},
			set: func(ctx context.Context, m *executeMocks) {
				m.db.EXPECT().CampaignList(ctx, "2026-01-01 00:00:00.000000", uint64(10), map[campaign.Field]any{campaign.FieldCustomerID: customerID}).Return(nil, errDB)
			},
		},
		{
			name: "list by customer id",
			fn: func(ctx context.Context, h *campaignHandler) ([]*campaign.Campaign, error) {
				return h.ListByCustomerID(ctx, customerID, "2026-01-01 00:00:00.000000", 10)
			},
			set: func(ctx context.Context, m *executeMocks) {
				m.db.EXPECT().CampaignListByCustomerID(ctx, customerID, "2026-01-01 00:00:00.000000", uint64(10)).Return(nil, errDB)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newExecuteTestHandler(mc)
			ctx := context.Background()
			tt.set(ctx, m)

			res, err := tt.fn(ctx, h)
			if res != nil {
				t.Errorf("Wrong match. expect: nil result, got: %v", res)
			}
			if !stderrors.Is(err, errDB) {
				t.Errorf("Wrong match. expect: %v, got: %v", errDB, err)
			}
		})
	}
}

// expectStoppableChain expects the isStoppable lookup for an idle campaign
// (execution already stopped, no ongoing campaigncall).
func expectStoppableChain(ctx context.Context, m *executeMocks, idle *campaign.Campaign) {
	m.db.EXPECT().CampaignGet(ctx, idle.ID).Return(idle, nil)
	m.util.EXPECT().TimeGetCurTime().Return("2026-01-01 00:00:00.000000")
	m.campaigncall.EXPECT().ListOngoingByCampaignID(ctx, idle.ID, "2026-01-01 00:00:00.000000", uint64(1)).Return(nil, nil)
}

func Test_updateExecuteStop_stopFails(t *testing.T) {
	id := uuid.FromStringOrNil("c2000000-0000-4000-8000-000000000001")
	errDB := stderrors.New("status update failed")
	idle := &campaign.Campaign{Identity: commonidentity.Identity{ID: id}, Status: campaign.StatusStopping, Execute: campaign.ExecuteStop}

	mc := gomock.NewController(t)
	defer mc.Finish()

	h, m := newExecuteTestHandler(mc)
	ctx := context.Background()

	m.db.EXPECT().CampaignUpdateExecute(ctx, id, campaign.ExecuteStop).Return(nil)
	expectStoppableChain(ctx, m, idle)
	// campaignStopNow fails at the status update, so no read back and no webhook follow
	m.db.EXPECT().CampaignGet(ctx, id).Return(idle, nil)
	m.db.EXPECT().CampaignUpdateStatus(ctx, id, campaign.StatusStop).Return(errDB)

	if err := h.updateExecuteStop(ctx, id); !stderrors.Is(err, errDB) {
		t.Errorf("Wrong match. expect: %v, got: %v", errDB, err)
	}
}

// Test_Execute_notRunStopFails checks that a campaign which is not in run status ends
// the execution even when the stop request itself fails, and that nothing else is requested.
func Test_Execute_notRunStopFails(t *testing.T) {
	id := uuid.FromStringOrNil("c3000000-0000-4000-8000-000000000001")
	stopped := &campaign.Campaign{Identity: commonidentity.Identity{ID: id}, Status: campaign.StatusStop}

	mc := gomock.NewController(t)
	defer mc.Finish()

	// the outplan handler and the campaigncall handler have no expectation, so any call to them fails the test
	h, m := newExecuteTestHandler(mc)
	ctx := context.Background()

	m.db.EXPECT().CampaignGet(ctx, id).Return(stopped, nil)
	expectStopAbort(ctx, m, id)

	h.Execute(ctx, id)
}

func Test_EventHandle_stopBranches(t *testing.T) {
	id := uuid.FromStringOrNil("c4000000-0000-4000-8000-000000000001")
	errDB := stderrors.New("status update failed")
	running := &campaign.Campaign{Identity: commonidentity.Identity{ID: id}, Status: campaign.StatusRun, Execute: campaign.ExecuteRun}
	idle := &campaign.Campaign{Identity: commonidentity.Identity{ID: id}, Status: campaign.StatusStopping, Execute: campaign.ExecuteStop}

	handlers := []struct {
		name string
		fn   func(ctx context.Context, h *campaignHandler, id uuid.UUID) error
	}{
		{"activeflow deleted", func(ctx context.Context, h *campaignHandler, id uuid.UUID) error {
			return h.EventHandleActiveflowDeleted(ctx, id)
		}},
		{"reference call hangup", func(ctx context.Context, h *campaignHandler, id uuid.UUID) error {
			return h.EventHandleReferenceCallHungup(ctx, id)
		}},
	}

	cases := []struct {
		name      string
		setup     func(ctx context.Context, m *executeMocks)
		expectErr error
	}{
		{
			name: "campaign still executing is left alone",
			setup: func(ctx context.Context, m *executeMocks) {
				m.db.EXPECT().CampaignGet(ctx, id).Return(running, nil)
			},
		},
		{
			name: "stop failure is returned",
			setup: func(ctx context.Context, m *executeMocks) {
				expectStoppableChain(ctx, m, idle)
				m.db.EXPECT().CampaignGet(ctx, id).Return(idle, nil)
				m.db.EXPECT().CampaignUpdateStatus(ctx, id, campaign.StatusStop).Return(errDB)
			},
			expectErr: errDB,
		},
	}

	for _, hd := range handlers {
		for _, tc := range cases {
			t.Run(hd.name+"/"+tc.name, func(t *testing.T) {
				mc := gomock.NewController(t)
				defer mc.Finish()

				h, m := newExecuteTestHandler(mc)
				ctx := context.Background()
				tc.setup(ctx, m)

				err := hd.fn(ctx, h, id)
				if !stderrors.Is(err, tc.expectErr) {
					t.Errorf("Wrong match. expect: %v, got: %v", tc.expectErr, err)
				}
			})
		}
	}
}

func Test_ReconcileOrphanedFlows_listFails(t *testing.T) {
	errDB := stderrors.New("db unavailable")
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

	mc := gomock.NewController(t)
	defer mc.Finish()

	h, m := newExecuteTestHandler(mc)

	// the flow requests have no expectation, so a flow lookup after the failed listing fails the test
	m.util.EXPECT().TimeNow().Return(&now)
	m.db.EXPECT().CampaignListDeletedSince(gomock.Any(), gomock.Any(), scanLimit).Return(nil, errDB)

	res, err := h.ReconcileOrphanedFlows(context.Background(), 3600)
	if !stderrors.Is(err, errDB) {
		t.Errorf("Wrong match. expect: %v, got: %v", errDB, err)
	}
	if res != (campaign.ReconcileResult{}) {
		t.Errorf("Wrong match. expect: empty result, got: %+v", res)
	}
}
