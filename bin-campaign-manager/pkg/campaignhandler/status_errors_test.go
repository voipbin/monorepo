package campaignhandler

import (
	"context"
	stderrors "errors"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-campaign-manager/models/campaign"
)

func Test_UpdateStatus_dispatch(t *testing.T) {
	id := uuid.FromStringOrNil("11000000-0000-4000-8000-000000000001")
	customerID := uuid.FromStringOrNil("11000000-0000-4000-8000-000000000002")
	running := &campaign.Campaign{Identity: commonidentity.Identity{ID: id, CustomerID: customerID}, Status: campaign.StatusRun, Execute: campaign.ExecuteRun}
	stopping := &campaign.Campaign{Identity: commonidentity.Identity{ID: id, CustomerID: customerID}, Status: campaign.StatusStopping, Execute: campaign.ExecuteRun}

	tests := []struct {
		name      string
		status    campaign.Status
		setup     func(ctx context.Context, m *executeMocks)
		expectRes *campaign.Campaign
		expectErr string
	}{
		{
			name:      "stopping is not an accepted target status",
			status:    campaign.StatusStopping,
			setup:     func(ctx context.Context, m *executeMocks) {},
			expectErr: "unsupported status. status: stopping",
		},
		{
			name:      "empty status is rejected",
			status:    campaign.Status(""),
			setup:     func(ctx context.Context, m *executeMocks) {},
			expectErr: "unsupported status. status: ",
		},
		{
			name:   "run on a running campaign returns it unchanged",
			status: campaign.StatusRun,
			setup: func(ctx context.Context, m *executeMocks) {
				m.db.EXPECT().CampaignGet(ctx, id).Return(running, nil)
			},
			expectRes: running,
		},
		{
			name:   "stop on a campaign whose execution is still running moves it to stopping",
			status: campaign.StatusStop,
			setup: func(ctx context.Context, m *executeMocks) {
				// isStoppable: the execution is still running.
				m.db.EXPECT().CampaignGet(ctx, id).Return(running, nil)
				// campaignStopping
				m.db.EXPECT().CampaignGet(ctx, id).Return(running, nil)
				m.db.EXPECT().CampaignUpdateStatus(ctx, id, campaign.StatusStopping).Return(nil)
				m.db.EXPECT().CampaignGet(ctx, id).Return(stopping, nil)
				m.notify.EXPECT().PublishWebhookEvent(ctx, customerID, campaign.EventTypeCampaignStatusStopping, stopping)
			},
			expectRes: stopping,
		},
		{
			name:   "stop on a campaign that is already stopping returns it without a status update",
			status: campaign.StatusStop,
			setup: func(ctx context.Context, m *executeMocks) {
				m.db.EXPECT().CampaignGet(ctx, id).Return(stopping, nil)
				m.db.EXPECT().CampaignGet(ctx, id).Return(stopping, nil)
			},
			expectRes: stopping,
		},
		{
			name:   "stop status update error is returned",
			status: campaign.StatusStop,
			setup: func(ctx context.Context, m *executeMocks) {
				m.db.EXPECT().CampaignGet(ctx, id).Return(running, nil)
				m.db.EXPECT().CampaignGet(ctx, id).Return(running, nil)
				m.db.EXPECT().CampaignUpdateStatus(ctx, id, campaign.StatusStopping).Return(stderrors.New("update error"))
			},
			expectErr: "update error",
		},
		{
			name:   "stop read back error is returned",
			status: campaign.StatusStop,
			setup: func(ctx context.Context, m *executeMocks) {
				m.db.EXPECT().CampaignGet(ctx, id).Return(running, nil)
				m.db.EXPECT().CampaignGet(ctx, id).Return(running, nil)
				m.db.EXPECT().CampaignUpdateStatus(ctx, id, campaign.StatusStopping).Return(nil)
				m.db.EXPECT().CampaignGet(ctx, id).Return(nil, stderrors.New("read error"))
			},
			expectErr: "read error",
		},
		{
			name:   "stop get error is returned",
			status: campaign.StatusStop,
			setup: func(ctx context.Context, m *executeMocks) {
				m.db.EXPECT().CampaignGet(ctx, id).Return(nil, stderrors.New("get error")).Times(2)
			},
			expectErr: "get error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newExecuteTestHandler(mc)
			ctx := context.Background()

			tt.setup(ctx, m)

			res, err := h.UpdateStatus(ctx, id, tt.status)
			if tt.expectErr != "" {
				if err == nil || err.Error() != tt.expectErr || res != nil {
					t.Errorf("Wrong match. expect error: %s, got: %v, res: %v", tt.expectErr, err, res)
				}
				return
			}
			if err != nil || res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v, err: %v", tt.expectRes, res, err)
			}
		})
	}
}

func Test_campaignStopNow_errors(t *testing.T) {
	id := uuid.FromStringOrNil("12000000-0000-4000-8000-000000000001")
	customerID := uuid.FromStringOrNil("12000000-0000-4000-8000-000000000002")
	stopping := &campaign.Campaign{Identity: commonidentity.Identity{ID: id, CustomerID: customerID}, Status: campaign.StatusStopping}
	stopped := &campaign.Campaign{Identity: commonidentity.Identity{ID: id, CustomerID: customerID}, Status: campaign.StatusStop}

	tests := []struct {
		name      string
		setup     func(ctx context.Context, m *executeMocks)
		expectRes *campaign.Campaign
		expectErr string
	}{
		{
			name: "get error",
			setup: func(ctx context.Context, m *executeMocks) {
				m.db.EXPECT().CampaignGet(ctx, id).Return(nil, stderrors.New("get error"))
			},
			expectErr: "get error",
		},
		{
			name: "already stopped campaign is returned without an update",
			setup: func(ctx context.Context, m *executeMocks) {
				m.db.EXPECT().CampaignGet(ctx, id).Return(stopped, nil)
			},
			expectRes: stopped,
		},
		{
			name: "status update error",
			setup: func(ctx context.Context, m *executeMocks) {
				m.db.EXPECT().CampaignGet(ctx, id).Return(stopping, nil)
				m.db.EXPECT().CampaignUpdateStatus(ctx, id, campaign.StatusStop).Return(stderrors.New("update error"))
			},
			expectErr: "update error",
		},
		{
			name: "read back error",
			setup: func(ctx context.Context, m *executeMocks) {
				m.db.EXPECT().CampaignGet(ctx, id).Return(stopping, nil)
				m.db.EXPECT().CampaignUpdateStatus(ctx, id, campaign.StatusStop).Return(nil)
				m.db.EXPECT().CampaignGet(ctx, id).Return(nil, stderrors.New("read error"))
			},
			expectErr: "read error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newExecuteTestHandler(mc)
			ctx := context.Background()

			tt.setup(ctx, m)

			res, err := h.campaignStopNow(ctx, id)
			if tt.expectErr != "" {
				if err == nil || err.Error() != tt.expectErr || res != nil {
					t.Errorf("Wrong match. expect error: %s, got: %v, res: %v", tt.expectErr, err, res)
				}
				return
			}
			if err != nil || res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v, err: %v", tt.expectRes, res, err)
			}
		})
	}
}

func Test_isStoppable_branches(t *testing.T) {
	id := uuid.FromStringOrNil("13000000-0000-4000-8000-000000000001")
	idle := &campaign.Campaign{Identity: commonidentity.Identity{ID: id}, Execute: campaign.ExecuteStop}
	running := &campaign.Campaign{Identity: commonidentity.Identity{ID: id}, Execute: campaign.ExecuteRun}

	tests := []struct {
		name   string
		setup  func(ctx context.Context, m *executeMocks)
		expect bool
	}{
		{
			name: "campaign lookup error is not stoppable",
			setup: func(ctx context.Context, m *executeMocks) {
				m.db.EXPECT().CampaignGet(ctx, id).Return(nil, stderrors.New("get error"))
			},
			expect: false,
		},
		{
			name: "running execution is not stoppable and no calls are listed",
			setup: func(ctx context.Context, m *executeMocks) {
				m.db.EXPECT().CampaignGet(ctx, id).Return(running, nil)
			},
			expect: false,
		},
		{
			name: "call listing error is logged and the campaign stays stoppable",
			setup: func(ctx context.Context, m *executeMocks) {
				m.db.EXPECT().CampaignGet(ctx, id).Return(idle, nil)
				m.util.EXPECT().TimeGetCurTime().Return("2026-01-01 00:00:00.000000")
				m.campaigncall.EXPECT().ListOngoingByCampaignID(ctx, id, "2026-01-01 00:00:00.000000", uint64(1)).Return(nil, stderrors.New("list error"))
			},
			expect: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newExecuteTestHandler(mc)
			ctx := context.Background()

			tt.setup(ctx, m)

			if got := h.isStoppable(ctx, id); got != tt.expect {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expect, got)
			}
		})
	}
}

func Test_campaignRun_errors(t *testing.T) {
	id := uuid.FromStringOrNil("14000000-0000-4000-8000-000000000001")
	customerID := uuid.FromStringOrNil("14000000-0000-4000-8000-000000000002")
	outplanID := uuid.FromStringOrNil("14000000-0000-4000-8000-000000000003")
	stopped := &campaign.Campaign{Identity: commonidentity.Identity{ID: id, CustomerID: customerID}, Status: campaign.StatusStop, Execute: campaign.ExecuteStop}
	running := &campaign.Campaign{Identity: commonidentity.Identity{ID: id, CustomerID: customerID}, Status: campaign.StatusRun, Execute: campaign.ExecuteRun}

	tests := []struct {
		name      string
		setup     func(ctx context.Context, m *executeMocks)
		expectErr string
	}{
		{
			name: "get error",
			setup: func(ctx context.Context, m *executeMocks) {
				m.db.EXPECT().CampaignGet(ctx, id).Return(nil, stderrors.New("get error"))
			},
			expectErr: "get error",
		},
		{
			name: "invalid resource is rejected before the status update",
			setup: func(ctx context.Context, m *executeMocks) {
				invalid := &campaign.Campaign{Identity: commonidentity.Identity{ID: id, CustomerID: customerID}, Status: campaign.StatusStop, OutplanID: outplanID}
				m.db.EXPECT().CampaignGet(ctx, id).Return(invalid, nil)
				m.outplan.EXPECT().Get(ctx, outplanID).Return(nil, stderrors.New("db error"))
			},
			expectErr: "the campaign resource is not valid",
		},
		{
			name: "status update error",
			setup: func(ctx context.Context, m *executeMocks) {
				m.db.EXPECT().CampaignGet(ctx, id).Return(stopped, nil)
				m.db.EXPECT().CampaignUpdateStatusAndExecute(ctx, id, campaign.StatusRun, campaign.ExecuteRun).Return(stderrors.New("update error"))
			},
			expectErr: "update error",
		},
		{
			name: "read back error",
			setup: func(ctx context.Context, m *executeMocks) {
				m.db.EXPECT().CampaignGet(ctx, id).Return(stopped, nil)
				m.db.EXPECT().CampaignUpdateStatusAndExecute(ctx, id, campaign.StatusRun, campaign.ExecuteRun).Return(nil)
				m.db.EXPECT().CampaignGet(ctx, id).Return(nil, stderrors.New("read error"))
			},
			expectErr: "read error",
		},
		{
			name: "execution start failure is returned after the stop attempt",
			setup: func(ctx context.Context, m *executeMocks) {
				m.db.EXPECT().CampaignGet(ctx, id).Return(stopped, nil)
				m.db.EXPECT().CampaignUpdateStatusAndExecute(ctx, id, campaign.StatusRun, campaign.ExecuteRun).Return(nil)
				m.db.EXPECT().CampaignGet(ctx, id).Return(running, nil)
				m.notify.EXPECT().PublishWebhookEvent(ctx, customerID, campaign.EventTypeCampaignStatusRun, running)
				m.req.EXPECT().CampaignV1CampaignExecute(ctx, id, 1000).Return(stderrors.New("rpc error"))
				// campaignStopNow ends at its first lookup
				m.db.EXPECT().CampaignGet(ctx, id).Return(nil, stderrors.New("stop lookup error"))
			},
			expectErr: "rpc error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newExecuteTestHandler(mc)
			ctx := context.Background()

			tt.setup(ctx, m)

			res, err := h.campaignRun(ctx, id)
			if err == nil || err.Error() != tt.expectErr || res != nil {
				t.Errorf("Wrong match. expect error: %s, got: %v, res: %v", tt.expectErr, err, res)
			}
		})
	}
}
