package campaignhandler

import (
	"context"
	stderrors "errors"
	"testing"
	"time"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"

	fmaction "monorepo/bin-flow-manager/models/action"
	fmflow "monorepo/bin-flow-manager/models/flow"
	omoutdial "monorepo/bin-outdial-manager/models/outdial"
	qmqueue "monorepo/bin-queue-manager/models/queue"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-campaign-manager/models/campaign"
	"monorepo/bin-campaign-manager/models/outplan"
	"monorepo/bin-campaign-manager/pkg/dbhandler"
)

func counterValue(c prometheus.Counter) float64 {
	return testutil.ToFloat64(c)
}

func Test_Get_errors(t *testing.T) {
	id := uuid.FromStringOrNil("d1000000-0000-4000-8000-000000000001")

	tests := []struct {
		name         string
		dbErr        error
		expectTyped  bool
		expectReason string
	}{
		{
			name:         "not found is converted to the typed error with the cause kept",
			dbErr:        dbhandler.ErrNotFound,
			expectTyped:  true,
			expectReason: "CAMPAIGN_NOT_FOUND",
		},
		{
			name:  "other errors are returned as they are",
			dbErr: stderrors.New("db unavailable"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newExecuteTestHandler(mc)
			ctx := context.Background()

			m.db.EXPECT().CampaignGet(ctx, id).Return(nil, tt.dbErr)

			res, err := h.Get(ctx, id)
			if res != nil {
				t.Errorf("Wrong match. expect: nil result, got: %v", res)
			}
			if !stderrors.Is(err, tt.dbErr) {
				t.Errorf("Wrong match. the cause must be kept. err: %v", err)
			}

			var ve *cerrors.VoipbinError
			isTyped := stderrors.As(err, &ve)
			if isTyped != tt.expectTyped {
				t.Fatalf("Wrong match. expect typed: %v, got: %v", tt.expectTyped, isTyped)
			}
			if tt.expectTyped && (ve.Status != cerrors.StatusNotFound || ve.Reason != tt.expectReason) {
				t.Errorf("Wrong match. expect: %s/%s, got: %s/%s", cerrors.StatusNotFound, tt.expectReason, ve.Status, ve.Reason)
			}
		})
	}
}

func Test_isValid_rejections(t *testing.T) {
	id := uuid.FromStringOrNil("d2000000-0000-4000-8000-000000000001")
	customerID := uuid.FromStringOrNil("d2000000-0000-4000-8000-000000000002")
	otherCustomerID := uuid.FromStringOrNil("d2000000-0000-4000-8000-000000000003")
	otherCampaignID := uuid.FromStringOrNil("d2000000-0000-4000-8000-000000000004")
	targetID := uuid.FromStringOrNil("d2000000-0000-4000-8000-000000000005")
	deletedAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name  string
		setup func(ctx context.Context, m *executeMocks)
		call  func(ctx context.Context, h *campaignHandler) bool
		want  bool
	}{
		// outdial
		{
			name: "outdial nil id is valid without lookup",
			setup: func(ctx context.Context, m *executeMocks) {
			},
			call: func(ctx context.Context, h *campaignHandler) bool {
				return h.isValidOutdialID(ctx, uuid.Nil, id, customerID)
			},
			want: true,
		},
		{
			name: "outdial lookup error is invalid",
			setup: func(ctx context.Context, m *executeMocks) {
				m.req.EXPECT().OutdialV1OutdialGet(ctx, targetID).Return(nil, stderrors.New("rpc error"))
			},
			call: func(ctx context.Context, h *campaignHandler) bool {
				return h.isValidOutdialID(ctx, targetID, id, customerID)
			},
			want: false,
		},
		{
			name: "outdial of another customer is invalid",
			setup: func(ctx context.Context, m *executeMocks) {
				m.req.EXPECT().OutdialV1OutdialGet(ctx, targetID).Return(&omoutdial.Outdial{
					Identity: commonidentity.Identity{ID: targetID, CustomerID: otherCustomerID},
				}, nil)
			},
			call: func(ctx context.Context, h *campaignHandler) bool {
				return h.isValidOutdialID(ctx, targetID, id, customerID)
			},
			want: false,
		},
		{
			name: "outdial used by another campaign is invalid",
			setup: func(ctx context.Context, m *executeMocks) {
				m.req.EXPECT().OutdialV1OutdialGet(ctx, targetID).Return(&omoutdial.Outdial{
					Identity:   commonidentity.Identity{ID: targetID, CustomerID: customerID},
					CampaignID: otherCampaignID,
				}, nil)
			},
			call: func(ctx context.Context, h *campaignHandler) bool {
				return h.isValidOutdialID(ctx, targetID, id, customerID)
			},
			want: false,
		},
		{
			name: "deleted outdial is invalid",
			setup: func(ctx context.Context, m *executeMocks) {
				m.req.EXPECT().OutdialV1OutdialGet(ctx, targetID).Return(&omoutdial.Outdial{
					Identity: commonidentity.Identity{ID: targetID, CustomerID: customerID},
					TMDelete: &deletedAt,
				}, nil)
			},
			call: func(ctx context.Context, h *campaignHandler) bool {
				return h.isValidOutdialID(ctx, targetID, id, customerID)
			},
			want: false,
		},
		// outplan
		{
			name: "outplan nil id is valid without lookup",
			setup: func(ctx context.Context, m *executeMocks) {
			},
			call: func(ctx context.Context, h *campaignHandler) bool {
				return h.isValidOutplanID(ctx, uuid.Nil, customerID)
			},
			want: true,
		},
		{
			name: "outplan lookup error is invalid",
			setup: func(ctx context.Context, m *executeMocks) {
				m.outplan.EXPECT().Get(ctx, targetID).Return(nil, stderrors.New("db error"))
			},
			call: func(ctx context.Context, h *campaignHandler) bool {
				return h.isValidOutplanID(ctx, targetID, customerID)
			},
			want: false,
		},
		{
			name: "outplan of another customer is invalid",
			setup: func(ctx context.Context, m *executeMocks) {
				m.outplan.EXPECT().Get(ctx, targetID).Return(&outplan.Outplan{
					Identity: commonidentity.Identity{ID: targetID, CustomerID: otherCustomerID},
				}, nil)
			},
			call: func(ctx context.Context, h *campaignHandler) bool {
				return h.isValidOutplanID(ctx, targetID, customerID)
			},
			want: false,
		},
		{
			name: "deleted outplan is invalid",
			setup: func(ctx context.Context, m *executeMocks) {
				m.outplan.EXPECT().Get(ctx, targetID).Return(&outplan.Outplan{
					Identity: commonidentity.Identity{ID: targetID, CustomerID: customerID},
					TMDelete: &deletedAt,
				}, nil)
			},
			call: func(ctx context.Context, h *campaignHandler) bool {
				return h.isValidOutplanID(ctx, targetID, customerID)
			},
			want: false,
		},
		// queue
		{
			name: "queue nil id is valid without lookup",
			setup: func(ctx context.Context, m *executeMocks) {
			},
			call: func(ctx context.Context, h *campaignHandler) bool {
				return h.isValidQueueID(ctx, uuid.Nil, customerID)
			},
			want: true,
		},
		{
			name: "queue lookup error is invalid",
			setup: func(ctx context.Context, m *executeMocks) {
				m.req.EXPECT().QueueV1QueueGet(ctx, targetID).Return(nil, stderrors.New("rpc error"))
			},
			call: func(ctx context.Context, h *campaignHandler) bool {
				return h.isValidQueueID(ctx, targetID, customerID)
			},
			want: false,
		},
		{
			name: "queue of another customer is invalid",
			setup: func(ctx context.Context, m *executeMocks) {
				m.req.EXPECT().QueueV1QueueGet(ctx, targetID).Return(&qmqueue.Queue{
					Identity: commonidentity.Identity{ID: targetID, CustomerID: otherCustomerID},
				}, nil)
			},
			call: func(ctx context.Context, h *campaignHandler) bool {
				return h.isValidQueueID(ctx, targetID, customerID)
			},
			want: false,
		},
		{
			name: "deleted queue is invalid",
			setup: func(ctx context.Context, m *executeMocks) {
				m.req.EXPECT().QueueV1QueueGet(ctx, targetID).Return(&qmqueue.Queue{
					Identity: commonidentity.Identity{ID: targetID, CustomerID: customerID},
					TMDelete: &deletedAt,
				}, nil)
			},
			call: func(ctx context.Context, h *campaignHandler) bool {
				return h.isValidQueueID(ctx, targetID, customerID)
			},
			want: false,
		},
		{
			name: "queue of the same customer is valid",
			setup: func(ctx context.Context, m *executeMocks) {
				m.req.EXPECT().QueueV1QueueGet(ctx, targetID).Return(&qmqueue.Queue{
					Identity: commonidentity.Identity{ID: targetID, CustomerID: customerID},
				}, nil)
			},
			call: func(ctx context.Context, h *campaignHandler) bool {
				return h.isValidQueueID(ctx, targetID, customerID)
			},
			want: true,
		},
		// next campaign
		{
			name: "next campaign nil id is valid without lookup",
			setup: func(ctx context.Context, m *executeMocks) {
			},
			call: func(ctx context.Context, h *campaignHandler) bool {
				return h.isValidNextCampaignID(ctx, uuid.Nil, customerID)
			},
			want: true,
		},
		{
			name: "next campaign lookup error is invalid",
			setup: func(ctx context.Context, m *executeMocks) {
				m.db.EXPECT().CampaignGet(ctx, targetID).Return(nil, stderrors.New("db error"))
			},
			call: func(ctx context.Context, h *campaignHandler) bool {
				return h.isValidNextCampaignID(ctx, targetID, customerID)
			},
			want: false,
		},
		{
			name: "next campaign of another customer is invalid",
			setup: func(ctx context.Context, m *executeMocks) {
				m.db.EXPECT().CampaignGet(ctx, targetID).Return(&campaign.Campaign{
					Identity: commonidentity.Identity{ID: targetID, CustomerID: otherCustomerID},
				}, nil)
			},
			call: func(ctx context.Context, h *campaignHandler) bool {
				return h.isValidNextCampaignID(ctx, targetID, customerID)
			},
			want: false,
		},
		{
			name: "deleted next campaign is invalid",
			setup: func(ctx context.Context, m *executeMocks) {
				m.db.EXPECT().CampaignGet(ctx, targetID).Return(&campaign.Campaign{
					Identity: commonidentity.Identity{ID: targetID, CustomerID: customerID},
					TMDelete: &deletedAt,
				}, nil)
			},
			call: func(ctx context.Context, h *campaignHandler) bool {
				return h.isValidNextCampaignID(ctx, targetID, customerID)
			},
			want: false,
		},
		{
			name: "next campaign of the same customer is valid",
			setup: func(ctx context.Context, m *executeMocks) {
				m.db.EXPECT().CampaignGet(ctx, targetID).Return(&campaign.Campaign{
					Identity: commonidentity.Identity{ID: targetID, CustomerID: customerID},
				}, nil)
			},
			call: func(ctx context.Context, h *campaignHandler) bool {
				return h.isValidNextCampaignID(ctx, targetID, customerID)
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newExecuteTestHandler(mc)
			ctx := context.Background()

			tt.setup(ctx, m)

			if got := tt.call(ctx, h); got != tt.want {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.want, got)
			}
		})
	}
}

func Test_validateResources_stops_at_first_invalid(t *testing.T) {
	id := uuid.FromStringOrNil("d3000000-0000-4000-8000-000000000001")
	customerID := uuid.FromStringOrNil("d3000000-0000-4000-8000-000000000002")
	outplanID := uuid.FromStringOrNil("d3000000-0000-4000-8000-000000000003")
	outdialID := uuid.FromStringOrNil("d3000000-0000-4000-8000-000000000004")
	queueID := uuid.FromStringOrNil("d3000000-0000-4000-8000-000000000005")
	nextID := uuid.FromStringOrNil("d3000000-0000-4000-8000-000000000006")
	otherCustomerID := uuid.FromStringOrNil("d3000000-0000-4000-8000-000000000007")

	okPlan := &outplan.Outplan{Identity: commonidentity.Identity{ID: outplanID, CustomerID: customerID}}
	okOutdial := &omoutdial.Outdial{Identity: commonidentity.Identity{ID: outdialID, CustomerID: customerID}}
	okQueue := &qmqueue.Queue{Identity: commonidentity.Identity{ID: queueID, CustomerID: customerID}}
	okNext := &campaign.Campaign{Identity: commonidentity.Identity{ID: nextID, CustomerID: customerID}}

	tests := []struct {
		name  string
		setup func(ctx context.Context, m *executeMocks)
		want  bool
	}{
		{
			name: "invalid outplan stops before the outdial lookup",
			setup: func(ctx context.Context, m *executeMocks) {
				m.outplan.EXPECT().Get(ctx, outplanID).Return(&outplan.Outplan{Identity: commonidentity.Identity{CustomerID: otherCustomerID}}, nil)
			},
			want: false,
		},
		{
			name: "invalid outdial stops before the queue lookup",
			setup: func(ctx context.Context, m *executeMocks) {
				m.outplan.EXPECT().Get(ctx, outplanID).Return(okPlan, nil)
				m.req.EXPECT().OutdialV1OutdialGet(ctx, outdialID).Return(&omoutdial.Outdial{Identity: commonidentity.Identity{CustomerID: otherCustomerID}}, nil)
			},
			want: false,
		},
		{
			name: "invalid queue stops before the next campaign lookup",
			setup: func(ctx context.Context, m *executeMocks) {
				m.outplan.EXPECT().Get(ctx, outplanID).Return(okPlan, nil)
				m.req.EXPECT().OutdialV1OutdialGet(ctx, outdialID).Return(okOutdial, nil)
				m.req.EXPECT().QueueV1QueueGet(ctx, queueID).Return(&qmqueue.Queue{Identity: commonidentity.Identity{CustomerID: otherCustomerID}}, nil)
			},
			want: false,
		},
		{
			name: "invalid next campaign is rejected last",
			setup: func(ctx context.Context, m *executeMocks) {
				m.outplan.EXPECT().Get(ctx, outplanID).Return(okPlan, nil)
				m.req.EXPECT().OutdialV1OutdialGet(ctx, outdialID).Return(okOutdial, nil)
				m.req.EXPECT().QueueV1QueueGet(ctx, queueID).Return(okQueue, nil)
				m.db.EXPECT().CampaignGet(ctx, nextID).Return(&campaign.Campaign{Identity: commonidentity.Identity{CustomerID: otherCustomerID}}, nil)
			},
			want: false,
		},
		{
			name: "every resource valid",
			setup: func(ctx context.Context, m *executeMocks) {
				m.outplan.EXPECT().Get(ctx, outplanID).Return(okPlan, nil)
				m.req.EXPECT().OutdialV1OutdialGet(ctx, outdialID).Return(okOutdial, nil)
				m.req.EXPECT().QueueV1QueueGet(ctx, queueID).Return(okQueue, nil)
				m.db.EXPECT().CampaignGet(ctx, nextID).Return(okNext, nil)
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newExecuteTestHandler(mc)
			ctx := context.Background()

			tt.setup(ctx, m)

			if got := h.validateResources(ctx, id, customerID, outplanID, outdialID, queueID, nextID); got != tt.want {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.want, got)
			}
		})
	}
}

func Test_campaign_update_functions_errors(t *testing.T) {
	id := uuid.FromStringOrNil("d4000000-0000-4000-8000-000000000001")
	customerID := uuid.FromStringOrNil("d4000000-0000-4000-8000-000000000002")
	nextID := uuid.FromStringOrNil("d4000000-0000-4000-8000-000000000003")
	flowID := uuid.FromStringOrNil("d4000000-0000-4000-8000-000000000004")
	queueID := uuid.FromStringOrNil("d4000000-0000-4000-8000-000000000005")
	name := "n"
	level := 40
	actions := []fmaction.Action{{Type: fmaction.TypeAnswer}}
	cp := &campaign.Campaign{
		Identity: commonidentity.Identity{ID: id, CustomerID: customerID},
		FlowID:   flowID,
		QueueID:  queueID,
		Actions:  actions,
	}
	updatedFields := map[campaign.Field]any{campaign.FieldName: name, campaign.FieldServiceLevel: level}

	tests := []struct {
		name      string
		setup     func(ctx context.Context, m *executeMocks)
		call      func(ctx context.Context, h *campaignHandler) (*campaign.Campaign, error)
		expectErr string
	}{
		{
			name: "basic info update error",
			setup: func(ctx context.Context, m *executeMocks) {
				m.db.EXPECT().CampaignUpdate(ctx, id, updatedFields).Return(stderrors.New("update error"))
			},
			call: func(ctx context.Context, h *campaignHandler) (*campaign.Campaign, error) {
				return h.UpdateBasicInfo(ctx, id, &name, nil, nil, &level, nil)
			},
			expectErr: "update error",
		},
		{
			name: "basic info read back error",
			setup: func(ctx context.Context, m *executeMocks) {
				m.db.EXPECT().CampaignUpdate(ctx, id, updatedFields).Return(nil)
				m.db.EXPECT().CampaignGet(ctx, id).Return(nil, stderrors.New("get error"))
			},
			call: func(ctx context.Context, h *campaignHandler) (*campaign.Campaign, error) {
				return h.UpdateBasicInfo(ctx, id, &name, nil, nil, &level, nil)
			},
			expectErr: "get error",
		},
		{
			name: "basic info without fields only reads the campaign",
			setup: func(ctx context.Context, m *executeMocks) {
				m.db.EXPECT().CampaignGet(ctx, id).Return(nil, stderrors.New("get error"))
			},
			call: func(ctx context.Context, h *campaignHandler) (*campaign.Campaign, error) {
				return h.UpdateBasicInfo(ctx, id, nil, nil, nil, nil, nil)
			},
			expectErr: "get error",
		},
		{
			name: "next campaign id update error",
			setup: func(ctx context.Context, m *executeMocks) {
				m.db.EXPECT().CampaignUpdateNextCampaignID(ctx, id, nextID).Return(stderrors.New("update error"))
			},
			call: func(ctx context.Context, h *campaignHandler) (*campaign.Campaign, error) {
				return h.UpdateNextCampaignID(ctx, id, nextID)
			},
			expectErr: "update error",
		},
		{
			name: "next campaign id read back error",
			setup: func(ctx context.Context, m *executeMocks) {
				m.db.EXPECT().CampaignUpdateNextCampaignID(ctx, id, nextID).Return(nil)
				m.db.EXPECT().CampaignGet(ctx, id).Return(nil, stderrors.New("get error"))
			},
			call: func(ctx context.Context, h *campaignHandler) (*campaign.Campaign, error) {
				return h.UpdateNextCampaignID(ctx, id, nextID)
			},
			expectErr: "get error",
		},
		{
			name: "service level update error",
			setup: func(ctx context.Context, m *executeMocks) {
				m.db.EXPECT().CampaignUpdateServiceLevel(ctx, id, level).Return(stderrors.New("update error"))
			},
			call: func(ctx context.Context, h *campaignHandler) (*campaign.Campaign, error) {
				return h.UpdateServiceLevel(ctx, id, level)
			},
			expectErr: "update error",
		},
		{
			name: "service level read back error",
			setup: func(ctx context.Context, m *executeMocks) {
				m.db.EXPECT().CampaignUpdateServiceLevel(ctx, id, level).Return(nil)
				m.db.EXPECT().CampaignGet(ctx, id).Return(nil, stderrors.New("get error"))
			},
			call: func(ctx context.Context, h *campaignHandler) (*campaign.Campaign, error) {
				return h.UpdateServiceLevel(ctx, id, level)
			},
			expectErr: "get error",
		},
		{
			name: "actions update stops when the campaign is not found",
			setup: func(ctx context.Context, m *executeMocks) {
				m.db.EXPECT().CampaignGet(ctx, id).Return(nil, stderrors.New("get error"))
			},
			call: func(ctx context.Context, h *campaignHandler) (*campaign.Campaign, error) {
				return h.UpdateActions(ctx, id, actions)
			},
			expectErr: "get error",
		},
		{
			name: "actions update stops when the flow update fails before the campaign is changed",
			setup: func(ctx context.Context, m *executeMocks) {
				m.db.EXPECT().CampaignGet(ctx, id).Return(cp, nil)
				m.req.EXPECT().FlowV1FlowUpdateActions(ctx, flowID, gomock.Any()).Return(nil, stderrors.New("flow error"))
			},
			call: func(ctx context.Context, h *campaignHandler) (*campaign.Campaign, error) {
				return h.UpdateActions(ctx, id, actions)
			},
			expectErr: "flow error",
		},
		{
			name: "actions update db error",
			setup: func(ctx context.Context, m *executeMocks) {
				m.db.EXPECT().CampaignGet(ctx, id).Return(cp, nil)
				m.req.EXPECT().FlowV1FlowUpdateActions(ctx, flowID, gomock.Any()).Return(&fmflow.Flow{Identity: commonidentity.Identity{ID: flowID}}, nil)
				m.db.EXPECT().CampaignUpdateActions(ctx, id, actions).Return(stderrors.New("update error"))
			},
			call: func(ctx context.Context, h *campaignHandler) (*campaign.Campaign, error) {
				return h.UpdateActions(ctx, id, actions)
			},
			expectErr: "update error",
		},
		{
			name: "actions update read back error",
			setup: func(ctx context.Context, m *executeMocks) {
				m.db.EXPECT().CampaignGet(ctx, id).Return(cp, nil)
				m.req.EXPECT().FlowV1FlowUpdateActions(ctx, flowID, gomock.Any()).Return(&fmflow.Flow{Identity: commonidentity.Identity{ID: flowID}}, nil)
				m.db.EXPECT().CampaignUpdateActions(ctx, id, actions).Return(nil)
				m.db.EXPECT().CampaignGet(ctx, id).Return(nil, stderrors.New("get error"))
			},
			call: func(ctx context.Context, h *campaignHandler) (*campaign.Campaign, error) {
				return h.UpdateActions(ctx, id, actions)
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

			res, err := tt.call(ctx, h)
			if err == nil || err.Error() != tt.expectErr || res != nil {
				t.Errorf("Wrong match. expect error: %s, got: %v, res: %v", tt.expectErr, err, res)
			}
		})
	}
}

func Test_UpdateActions_appends_queue_join(t *testing.T) {
	id := uuid.FromStringOrNil("d5000000-0000-4000-8000-000000000001")
	flowID := uuid.FromStringOrNil("d5000000-0000-4000-8000-000000000004")
	queueID := uuid.FromStringOrNil("d5000000-0000-4000-8000-000000000005")
	actions := []fmaction.Action{{Type: fmaction.TypeAnswer}}
	cp := &campaign.Campaign{Identity: commonidentity.Identity{ID: id}, FlowID: flowID, QueueID: queueID}

	mc := gomock.NewController(t)
	defer mc.Finish()

	h, m := newExecuteTestHandler(mc)
	ctx := context.Background()

	m.db.EXPECT().CampaignGet(ctx, id).Return(cp, nil)
	m.req.EXPECT().FlowV1FlowUpdateActions(ctx, flowID, gomock.Any()).DoAndReturn(
		func(_ context.Context, _ uuid.UUID, got []fmaction.Action) (*fmflow.Flow, error) {
			if len(got) != 2 || got[0].Type != fmaction.TypeAnswer || got[1].Type != fmaction.TypeQueueJoin {
				t.Errorf("Wrong match. expect: the given action followed by queue join, got: %v", got)
			}
			return nil, stderrors.New("stop here")
		})

	if _, err := h.UpdateActions(ctx, id, actions); err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}

func Test_UpdateResourceInfo_errors(t *testing.T) {
	id := uuid.FromStringOrNil("d6000000-0000-4000-8000-000000000001")
	customerID := uuid.FromStringOrNil("d6000000-0000-4000-8000-000000000002")
	outplanID := uuid.FromStringOrNil("d6000000-0000-4000-8000-000000000003")
	outdialID := uuid.FromStringOrNil("d6000000-0000-4000-8000-000000000004")
	flowID := uuid.FromStringOrNil("d6000000-0000-4000-8000-000000000005")
	cp := &campaign.Campaign{Identity: commonidentity.Identity{ID: id, CustomerID: customerID}, FlowID: flowID}
	validPlan := &outplan.Outplan{Identity: commonidentity.Identity{ID: outplanID, CustomerID: customerID}}
	validOutdial := &omoutdial.Outdial{Identity: commonidentity.Identity{ID: outdialID, CustomerID: customerID}}

	call := func(ctx context.Context, h *campaignHandler) (*campaign.Campaign, error) {
		return h.UpdateResourceInfo(ctx, id, outplanID, outdialID, uuid.Nil, uuid.Nil)
	}
	expectValid := func(ctx context.Context, m *executeMocks) {
		m.db.EXPECT().CampaignGet(ctx, id).Return(cp, nil)
		m.outplan.EXPECT().Get(ctx, outplanID).Return(validPlan, nil)
		m.req.EXPECT().OutdialV1OutdialGet(ctx, outdialID).Return(validOutdial, nil)
	}

	tests := []struct {
		name      string
		setup     func(ctx context.Context, m *executeMocks)
		expectErr string
	}{
		{
			name: "campaign get error",
			setup: func(ctx context.Context, m *executeMocks) {
				m.db.EXPECT().CampaignGet(ctx, id).Return(nil, stderrors.New("get error"))
			},
			expectErr: "get error",
		},
		{
			name: "resource validation failure does not update the campaign",
			setup: func(ctx context.Context, m *executeMocks) {
				m.db.EXPECT().CampaignGet(ctx, id).Return(cp, nil)
				m.outplan.EXPECT().Get(ctx, outplanID).Return(nil, stderrors.New("db error"))
			},
			expectErr: "could not pass the resource validation",
		},
		{
			name: "db update error",
			setup: func(ctx context.Context, m *executeMocks) {
				expectValid(ctx, m)
				m.db.EXPECT().CampaignUpdateResourceInfo(ctx, id, outplanID, outdialID, uuid.Nil, uuid.Nil).Return(stderrors.New("update error"))
			},
			expectErr: "update error",
		},
		{
			name: "flow update error",
			setup: func(ctx context.Context, m *executeMocks) {
				expectValid(ctx, m)
				m.db.EXPECT().CampaignUpdateResourceInfo(ctx, id, outplanID, outdialID, uuid.Nil, uuid.Nil).Return(nil)
				m.req.EXPECT().FlowV1FlowUpdateActions(ctx, flowID, gomock.Any()).Return(nil, stderrors.New("flow error"))
			},
			expectErr: "flow error",
		},
		{
			name: "read back error",
			setup: func(ctx context.Context, m *executeMocks) {
				expectValid(ctx, m)
				m.db.EXPECT().CampaignUpdateResourceInfo(ctx, id, outplanID, outdialID, uuid.Nil, uuid.Nil).Return(nil)
				m.req.EXPECT().FlowV1FlowUpdateActions(ctx, flowID, gomock.Any()).Return(&fmflow.Flow{Identity: commonidentity.Identity{ID: flowID}}, nil)
				m.db.EXPECT().CampaignGet(ctx, id).Return(nil, stderrors.New("read error"))
			},
			expectErr: "read error",
		},
		{
			name: "referenced resource update failure is reported after the webhook",
			setup: func(ctx context.Context, m *executeMocks) {
				updated := &campaign.Campaign{Identity: commonidentity.Identity{ID: id, CustomerID: customerID}, OutdialID: outdialID}
				expectValid(ctx, m)
				m.db.EXPECT().CampaignUpdateResourceInfo(ctx, id, outplanID, outdialID, uuid.Nil, uuid.Nil).Return(nil)
				m.req.EXPECT().FlowV1FlowUpdateActions(ctx, flowID, gomock.Any()).Return(&fmflow.Flow{Identity: commonidentity.Identity{ID: flowID}}, nil)
				m.db.EXPECT().CampaignGet(ctx, id).Return(updated, nil)
				m.notify.EXPECT().PublishWebhookEvent(ctx, customerID, campaign.EventTypeCampaignUpdated, updated)
				m.req.EXPECT().OutdialV1OutdialUpdateCampaignID(ctx, outdialID, id).Return(nil, stderrors.New("outdial error"))
			},
			expectErr: "could not update the resources",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newExecuteTestHandler(mc)
			ctx := context.Background()

			tt.setup(ctx, m)

			res, err := call(ctx, h)
			if err == nil || err.Error() != tt.expectErr || res != nil {
				t.Errorf("Wrong match. expect error: %s, got: %v, res: %v", tt.expectErr, err, res)
			}
		})
	}
}

func Test_Create_errors(t *testing.T) {
	id := uuid.FromStringOrNil("d7000000-0000-4000-8000-000000000001")
	customerID := uuid.FromStringOrNil("d7000000-0000-4000-8000-000000000002")
	flowID := uuid.FromStringOrNil("d7000000-0000-4000-8000-000000000003")
	outdialID := uuid.FromStringOrNil("d7000000-0000-4000-8000-000000000004")

	call := func(ctx context.Context, h *campaignHandler, createID uuid.UUID, outdial uuid.UUID) (*campaign.Campaign, error) {
		return h.Create(ctx, createID, customerID, campaign.TypeCall, "n", "d", nil, 0, campaign.EndHandleStop, uuid.Nil, outdial, uuid.Nil, uuid.Nil)
	}
	flow := &fmflow.Flow{Identity: commonidentity.Identity{ID: flowID}}

	tests := []struct {
		name      string
		createID  uuid.UUID
		outdial   uuid.UUID
		setup     func(ctx context.Context, m *executeMocks)
		expectErr string
	}{
		{
			name:     "invalid resource is rejected before the flow is created",
			createID: id,
			outdial:  outdialID,
			setup: func(ctx context.Context, m *executeMocks) {
				m.req.EXPECT().OutdialV1OutdialGet(ctx, outdialID).Return(nil, stderrors.New("rpc error"))
			},
			expectErr: "could not pass the resource validation",
		},
		{
			name:     "flow create error",
			createID: id,
			setup: func(ctx context.Context, m *executeMocks) {
				m.req.EXPECT().FlowV1FlowCreate(ctx, customerID, fmflow.TypeCampaign, "", "", gomock.Any(), uuid.Nil, true).Return(nil, stderrors.New("flow error"))
			},
			expectErr: "flow error",
		},
		{
			name:     "db create error",
			createID: id,
			setup: func(ctx context.Context, m *executeMocks) {
				m.req.EXPECT().FlowV1FlowCreate(ctx, customerID, fmflow.TypeCampaign, "", "", gomock.Any(), uuid.Nil, true).Return(flow, nil)
				m.db.EXPECT().CampaignCreate(ctx, gomock.Any()).Return(stderrors.New("create error"))
			},
			expectErr: "create error",
		},
		{
			name:     "nil id is generated and the read back error is returned",
			createID: uuid.Nil,
			setup: func(ctx context.Context, m *executeMocks) {
				m.util.EXPECT().UUIDCreate().Return(id)
				m.req.EXPECT().FlowV1FlowCreate(ctx, customerID, fmflow.TypeCampaign, "", "", gomock.Any(), uuid.Nil, true).Return(flow, nil)
				m.db.EXPECT().CampaignCreate(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, c *campaign.Campaign) error {
					if c.ID != id || c.FlowID != flowID || c.Status != campaign.StatusStop || c.Execute != campaign.ExecuteStop {
						t.Errorf("Wrong match. expect: generated id and stopped campaign with the flow, got: %v", c)
					}
					return nil
				})
				m.db.EXPECT().CampaignGet(ctx, id).Return(nil, stderrors.New("get error"))
			},
			expectErr: "get error",
		},
		{
			name:     "referenced resource update failure is reported after the webhook",
			createID: id,
			outdial:  outdialID,
			setup: func(ctx context.Context, m *executeMocks) {
				created := &campaign.Campaign{Identity: commonidentity.Identity{ID: id, CustomerID: customerID}, OutdialID: outdialID}
				m.req.EXPECT().OutdialV1OutdialGet(ctx, outdialID).Return(&omoutdial.Outdial{Identity: commonidentity.Identity{ID: outdialID, CustomerID: customerID}}, nil)
				m.req.EXPECT().FlowV1FlowCreate(ctx, customerID, fmflow.TypeCampaign, "", "", gomock.Any(), uuid.Nil, true).Return(flow, nil)
				m.db.EXPECT().CampaignCreate(ctx, gomock.Any()).Return(nil)
				m.db.EXPECT().CampaignGet(ctx, id).Return(created, nil)
				m.notify.EXPECT().PublishWebhookEvent(ctx, customerID, campaign.EventTypeCampaignCreated, created)
				m.req.EXPECT().OutdialV1OutdialUpdateCampaignID(ctx, outdialID, id).Return(nil, stderrors.New("outdial error"))
			},
			expectErr: "could not update the resources info",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newExecuteTestHandler(mc)
			ctx := context.Background()

			tt.setup(ctx, m)

			res, err := call(ctx, h, tt.createID, tt.outdial)
			if err == nil || err.Error() != tt.expectErr || res != nil {
				t.Errorf("Wrong match. expect error: %s, got: %v, res: %v", tt.expectErr, err, res)
			}
		})
	}
}

func Test_Delete_errors(t *testing.T) {
	id := uuid.FromStringOrNil("d8000000-0000-4000-8000-000000000001")
	customerID := uuid.FromStringOrNil("d8000000-0000-4000-8000-000000000002")
	stopped := &campaign.Campaign{Identity: commonidentity.Identity{ID: id, CustomerID: customerID}, Status: campaign.StatusStop}

	tests := []struct {
		name      string
		setup     func(ctx context.Context, m *executeMocks)
		expectErr string
	}{
		{
			name: "campaign get error",
			setup: func(ctx context.Context, m *executeMocks) {
				m.db.EXPECT().CampaignGet(ctx, id).Return(nil, stderrors.New("get error"))
			},
			expectErr: "get error",
		},
		{
			name: "delete error",
			setup: func(ctx context.Context, m *executeMocks) {
				m.db.EXPECT().CampaignGet(ctx, id).Return(stopped, nil)
				m.db.EXPECT().CampaignDelete(ctx, id).Return(stderrors.New("delete error"))
			},
			expectErr: "delete error",
		},
		{
			name: "read back error after the delete",
			setup: func(ctx context.Context, m *executeMocks) {
				m.db.EXPECT().CampaignGet(ctx, id).Return(stopped, nil)
				m.db.EXPECT().CampaignDelete(ctx, id).Return(nil)
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

			res, err := h.Delete(ctx, id)
			if err == nil || err.Error() != tt.expectErr || res != nil {
				t.Errorf("Wrong match. expect error: %s, got: %v, res: %v", tt.expectErr, err, res)
			}
		})
	}
}

func Test_Delete_flow_delete_reason_label(t *testing.T) {
	// the failure of the flow delete is only reported through the metric,
	// and the deleted campaign is still returned.
	id := uuid.FromStringOrNil("d9000000-0000-4000-8000-000000000001")
	customerID := uuid.FromStringOrNil("d9000000-0000-4000-8000-000000000002")
	flowID := uuid.FromStringOrNil("d9000000-0000-4000-8000-000000000003")
	stopped := &campaign.Campaign{Identity: commonidentity.Identity{ID: id, CustomerID: customerID}, Status: campaign.StatusStop, FlowID: flowID}

	tests := []struct {
		name      string
		flowErr   error
		reason    string
		otherName string
	}{
		{"not found flow is counted as not_found", requesthandler.ErrNotFound, "not_found", "error"},
		{"other flow error is counted as error", stderrors.New("flow error"), "error", "not_found"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newExecuteTestHandler(mc)
			ctx := context.Background()

			m.db.EXPECT().CampaignGet(ctx, id).Return(stopped, nil)
			m.db.EXPECT().CampaignDelete(ctx, id).Return(nil)
			m.db.EXPECT().CampaignGet(ctx, id).Return(stopped, nil)
			m.notify.EXPECT().PublishWebhookEvent(ctx, customerID, campaign.EventTypeCampaignDeleted, stopped)
			m.req.EXPECT().FlowV1FlowDelete(ctx, flowID).Return(nil, tt.flowErr)

			before := counterValue(promCampaignFlowDeleteFailedTotal.WithLabelValues(tt.reason))
			beforeOther := counterValue(promCampaignFlowDeleteFailedTotal.WithLabelValues(tt.otherName))

			res, err := h.Delete(ctx, id)
			if err != nil || res != stopped {
				t.Fatalf("Wrong match. expect: the deleted campaign, got: %v, err: %v", res, err)
			}

			if got := counterValue(promCampaignFlowDeleteFailedTotal.WithLabelValues(tt.reason)); got != before+1 {
				t.Errorf("Wrong match. expect: %s counter incremented by 1, got: %v -> %v", tt.reason, before, got)
			}
			if got := counterValue(promCampaignFlowDeleteFailedTotal.WithLabelValues(tt.otherName)); got != beforeOther {
				t.Errorf("Wrong match. expect: %s counter unchanged, got: %v -> %v", tt.otherName, beforeOther, got)
			}
		})
	}
}
