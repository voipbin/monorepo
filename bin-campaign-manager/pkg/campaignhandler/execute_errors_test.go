package campaignhandler

import (
	"context"
	"errors"
	"testing"
	"time"

	amagent "monorepo/bin-agent-manager/models/agent"
	cmcall "monorepo/bin-call-manager/models/call"
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-flow-manager/models/activeflow"
	omoutdialtarget "monorepo/bin-outdial-manager/models/outdialtarget"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-campaign-manager/models/campaign"
	"monorepo/bin-campaign-manager/models/campaigncall"
	"monorepo/bin-campaign-manager/models/outplan"
	"monorepo/bin-campaign-manager/pkg/campaigncallhandler"
	"monorepo/bin-campaign-manager/pkg/dbhandler"
	"monorepo/bin-campaign-manager/pkg/outplanhandler"
)

type executeMocks struct {
	util         *utilhandler.MockUtilHandler
	db           *dbhandler.MockDBHandler
	notify       *notifyhandler.MockNotifyHandler
	req          *requesthandler.MockRequestHandler
	outplan      *outplanhandler.MockOutplanHandler
	campaigncall *campaigncallhandler.MockCampaigncallHandler
}

func newExecuteTestHandler(mc *gomock.Controller) (*campaignHandler, *executeMocks) {
	m := &executeMocks{
		util:         utilhandler.NewMockUtilHandler(mc),
		db:           dbhandler.NewMockDBHandler(mc),
		notify:       notifyhandler.NewMockNotifyHandler(mc),
		req:          requesthandler.NewMockRequestHandler(mc),
		outplan:      outplanhandler.NewMockOutplanHandler(mc),
		campaigncall: campaigncallhandler.NewMockCampaigncallHandler(mc),
	}
	h := &campaignHandler{
		util:                m.util,
		db:                  m.db,
		notifyHandler:       m.notify,
		reqHandler:          m.req,
		campaigncallHandler: m.campaigncall,
		outplanHandler:      m.outplan,
	}
	return h, m
}

// expectStopAbort expects the stop request of the campaign execution and makes
// the first step fail, so the stop chain ends right after it.
func expectStopAbort(ctx context.Context, m *executeMocks, id uuid.UUID) {
	m.db.EXPECT().CampaignUpdateExecute(ctx, id, campaign.ExecuteStop).Return(errors.New("abort"))
}

// expectStopFull expects the whole stop chain of the campaign execution that
// ends with the status stop.
func expectStopFull(ctx context.Context, m *executeMocks, stopped *campaign.Campaign) {
	m.db.EXPECT().CampaignUpdateExecute(ctx, stopped.ID, campaign.ExecuteStop).Return(nil)

	// isStoppable
	m.db.EXPECT().CampaignGet(ctx, stopped.ID).Return(stopped, nil)
	m.util.EXPECT().TimeGetCurTime().Return("2026-01-01 00:00:00.000000")
	m.campaigncall.EXPECT().ListOngoingByCampaignID(ctx, stopped.ID, "2026-01-01 00:00:00.000000", uint64(1)).Return([]*campaigncall.Campaigncall{}, nil)

	// campaignStopNow
	m.db.EXPECT().CampaignGet(ctx, stopped.ID).Return(stopped, nil)
	m.db.EXPECT().CampaignUpdateStatus(ctx, stopped.ID, campaign.StatusStop).Return(nil)
	m.db.EXPECT().CampaignGet(ctx, stopped.ID).Return(stopped, nil)
	m.notify.EXPECT().PublishWebhookEvent(ctx, stopped.CustomerID, campaign.EventTypeCampaignStatusStop, stopped)
}

func Test_Execute_branches(t *testing.T) {
	id := uuid.FromStringOrNil("a1000000-0000-4000-8000-000000000001")
	customerID := uuid.FromStringOrNil("a1000000-0000-4000-8000-000000000002")
	outplanID := uuid.FromStringOrNil("a1000000-0000-4000-8000-000000000003")
	outdialID := uuid.FromStringOrNil("a1000000-0000-4000-8000-000000000004")
	queueID := uuid.FromStringOrNil("a1000000-0000-4000-8000-000000000005")
	flowID := uuid.FromStringOrNil("a1000000-0000-4000-8000-000000000006")
	targetID := uuid.FromStringOrNil("a1000000-0000-4000-8000-000000000007")

	newCampaign := func(mod func(c *campaign.Campaign)) *campaign.Campaign {
		c := &campaign.Campaign{
			Identity:  commonidentity.Identity{ID: id, CustomerID: customerID},
			OutplanID: outplanID,
			OutdialID: outdialID,
			FlowID:    flowID,
			Status:    campaign.StatusRun,
			Type:      campaign.TypeFlow,
			EndHandle: campaign.EndHandleStop,
		}
		if mod != nil {
			mod(c)
		}
		return c
	}
	plan := &outplan.Outplan{
		Identity:     commonidentity.Identity{ID: outplanID},
		Source:       &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550000001"},
		TryInterval:  3600000,
		MaxTryCount0: 2,
	}
	dest := &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550000002"}
	freshTarget := omoutdialtarget.OutdialTarget{ID: targetID, Destination0: dest}
	available := []omoutdialtarget.OutdialTarget{freshTarget}

	expectPlanAndTarget := func(ctx context.Context, m *executeMocks, targets []omoutdialtarget.OutdialTarget, err error) {
		m.outplan.EXPECT().Get(ctx, outplanID).Return(plan, nil)
		m.req.EXPECT().OutdialV1OutdialtargetGetsAvailable(ctx, outdialID, 2, 0, 0, 0, 0, 1).Return(targets, err)
	}

	tests := []struct {
		name  string
		setup func(ctx context.Context, m *executeMocks)
	}{
		{
			name: "campaign get error stops the execution",
			setup: func(ctx context.Context, m *executeMocks) {
				m.db.EXPECT().CampaignGet(ctx, id).Return(nil, errors.New("db error"))
				expectStopAbort(ctx, m, id)
			},
		},
		{
			name: "campaign not in run status stops the execution through the full stop chain",
			setup: func(ctx context.Context, m *executeMocks) {
				stopping := newCampaign(func(c *campaign.Campaign) {
					c.Status = campaign.StatusStopping
					c.Execute = campaign.ExecuteStop
				})
				m.db.EXPECT().CampaignGet(ctx, id).Return(stopping, nil)
				expectStopFull(ctx, m, stopping)
			},
		},
		{
			name: "outplan get error stops the execution",
			setup: func(ctx context.Context, m *executeMocks) {
				m.db.EXPECT().CampaignGet(ctx, id).Return(newCampaign(nil), nil)
				m.outplan.EXPECT().Get(ctx, outplanID).Return(nil, errors.New("outplan error"))
				expectStopAbort(ctx, m, id)
			},
		},
		{
			name: "outdial target lookup error stops the execution",
			setup: func(ctx context.Context, m *executeMocks) {
				m.db.EXPECT().CampaignGet(ctx, id).Return(newCampaign(nil), nil)
				expectPlanAndTarget(ctx, m, nil, errors.New("outdial error"))
				expectStopAbort(ctx, m, id)
			},
		},
		{
			name: "no target and end handle stop stops the execution",
			setup: func(ctx context.Context, m *executeMocks) {
				m.db.EXPECT().CampaignGet(ctx, id).Return(newCampaign(nil), nil)
				expectPlanAndTarget(ctx, m, []omoutdialtarget.OutdialTarget{}, nil)
				expectStopAbort(ctx, m, id)
			},
		},
		{
			name: "no target and end handle continue reschedules with 5 seconds",
			setup: func(ctx context.Context, m *executeMocks) {
				m.db.EXPECT().CampaignGet(ctx, id).Return(newCampaign(func(c *campaign.Campaign) { c.EndHandle = campaign.EndHandleContinue }), nil)
				expectPlanAndTarget(ctx, m, []omoutdialtarget.OutdialTarget{}, nil)
				m.req.EXPECT().CampaignV1CampaignExecute(ctx, id, 5000).Return(nil)
			},
		},
		{
			name: "not dialable by capacity reschedules with 5 seconds",
			setup: func(ctx context.Context, m *executeMocks) {
				m.db.EXPECT().CampaignGet(ctx, id).Return(newCampaign(func(c *campaign.Campaign) {
					c.QueueID = queueID
					c.ServiceLevel = 100
				}), nil)
				expectPlanAndTarget(ctx, m, available, nil)
				m.req.EXPECT().QueueV1QueueGetAgents(ctx, queueID, map[amagent.Field]any{amagent.FieldStatus: amagent.StatusAvailable}).Return([]amagent.Agent{{}}, nil)
				m.campaigncall.EXPECT().ListByCampaignIDAndStatus(ctx, id, campaigncall.StatusDialing, "", uint64(100)).Return([]*campaigncall.Campaigncall{{}}, nil)
				m.req.EXPECT().CampaignV1CampaignExecute(ctx, id, 5000).Return(nil)
			},
		},
		{
			name: "not dialable and reschedule failure stops the execution",
			setup: func(ctx context.Context, m *executeMocks) {
				m.db.EXPECT().CampaignGet(ctx, id).Return(newCampaign(func(c *campaign.Campaign) {
					c.QueueID = queueID
					c.ServiceLevel = 100
				}), nil)
				expectPlanAndTarget(ctx, m, available, nil)
				m.req.EXPECT().QueueV1QueueGetAgents(ctx, queueID, gomock.Any()).Return([]amagent.Agent{{}}, nil)
				m.campaigncall.EXPECT().ListByCampaignIDAndStatus(ctx, id, campaigncall.StatusDialing, "", uint64(100)).Return([]*campaigncall.Campaigncall{{}}, nil)
				m.req.EXPECT().CampaignV1CampaignExecute(ctx, id, 5000).Return(errors.New("rpc error"))
				expectStopAbort(ctx, m, id)
			},
		},
		{
			name: "retry interval not elapsed reschedules with 5 seconds",
			setup: func(ctx context.Context, m *executeMocks) {
				created := time.Now().Add(-2 * time.Hour)
				updated := time.Now().Add(-1 * time.Minute)
				retried := omoutdialtarget.OutdialTarget{ID: targetID, Destination0: dest, TryCount0: 1, TMCreate: &created, TMUpdate: &updated}
				m.db.EXPECT().CampaignGet(ctx, id).Return(newCampaign(nil), nil)
				expectPlanAndTarget(ctx, m, []omoutdialtarget.OutdialTarget{retried}, nil)
				m.req.EXPECT().CampaignV1CampaignExecute(ctx, id, 5000).Return(nil)
			},
		},
		{
			name: "target without usable destination stops the execution",
			setup: func(ctx context.Context, m *executeMocks) {
				m.db.EXPECT().CampaignGet(ctx, id).Return(newCampaign(nil), nil)
				expectPlanAndTarget(ctx, m, []omoutdialtarget.OutdialTarget{{ID: targetID}}, nil)
				expectStopAbort(ctx, m, id)
			},
		},
		{
			name: "flow type with campaigncall create error stops the execution",
			setup: func(ctx context.Context, m *executeMocks) {
				m.db.EXPECT().CampaignGet(ctx, id).Return(newCampaign(nil), nil)
				expectPlanAndTarget(ctx, m, available, nil)
				m.util.EXPECT().UUIDCreate().Return(uuid.FromStringOrNil("a1000000-0000-4000-8000-0000000000a1"))
				m.campaigncall.EXPECT().Create(ctx, customerID, id, outplanID, outdialID, targetID, uuid.Nil,
					uuid.FromStringOrNil("a1000000-0000-4000-8000-0000000000a1"), flowID, campaigncall.ReferenceTypeFlow, uuid.Nil,
					plan.Source, dest, 0, 1).Return(nil, errors.New("create error"))
				expectStopAbort(ctx, m, id)
			},
		},
		{
			name: "flow type with activeflow create error marks the campaigncall as fail",
			setup: func(ctx context.Context, m *executeMocks) {
				ccID := uuid.FromStringOrNil("a1000000-0000-4000-8000-0000000000b1")
				created := &campaigncall.Campaigncall{Identity: commonidentity.Identity{ID: ccID}}
				progressed := &campaigncall.Campaigncall{
					Identity:     commonidentity.Identity{ID: ccID, CustomerID: customerID},
					ActiveflowID: uuid.FromStringOrNil("a1000000-0000-4000-8000-0000000000b2"),
					FlowID:       flowID,
				}
				m.db.EXPECT().CampaignGet(ctx, id).Return(newCampaign(nil), nil)
				expectPlanAndTarget(ctx, m, available, nil)
				m.util.EXPECT().UUIDCreate().Return(uuid.FromStringOrNil("a1000000-0000-4000-8000-0000000000a1"))
				m.campaigncall.EXPECT().Create(ctx, customerID, id, outplanID, outdialID, targetID, uuid.Nil,
					gomock.Any(), flowID, campaigncall.ReferenceTypeFlow, uuid.Nil, plan.Source, dest, 0, 1).Return(created, nil)
				m.campaigncall.EXPECT().Progressing(ctx, ccID).Return(progressed, nil)
				m.req.EXPECT().FlowV1ActiveflowCreate(ctx, progressed.ActiveflowID, customerID, flowID, activeflow.ReferenceTypeCampaign, ccID, uuid.Nil, nil, "", activeflow.WebhookMethodNone).
					Return(nil, errors.New("activeflow error"))
				m.campaigncall.EXPECT().Done(ctx, ccID, campaigncall.ResultFail).Return(nil, nil)
				expectStopAbort(ctx, m, id)
			},
		},
		{
			name: "flow type with activeflow execute error marks the campaigncall as fail",
			setup: func(ctx context.Context, m *executeMocks) {
				ccID := uuid.FromStringOrNil("a1000000-0000-4000-8000-0000000000c1")
				afID := uuid.FromStringOrNil("a1000000-0000-4000-8000-0000000000c2")
				created := &campaigncall.Campaigncall{Identity: commonidentity.Identity{ID: ccID}}
				progressed := &campaigncall.Campaigncall{
					Identity:     commonidentity.Identity{ID: ccID, CustomerID: customerID},
					ActiveflowID: afID,
					FlowID:       flowID,
				}
				m.db.EXPECT().CampaignGet(ctx, id).Return(newCampaign(nil), nil)
				expectPlanAndTarget(ctx, m, available, nil)
				m.util.EXPECT().UUIDCreate().Return(afID)
				m.campaigncall.EXPECT().Create(ctx, customerID, id, outplanID, outdialID, targetID, uuid.Nil,
					afID, flowID, campaigncall.ReferenceTypeFlow, uuid.Nil, plan.Source, dest, 0, 1).Return(created, nil)
				m.campaigncall.EXPECT().Progressing(ctx, ccID).Return(progressed, nil)
				m.req.EXPECT().FlowV1ActiveflowCreate(ctx, afID, customerID, flowID, activeflow.ReferenceTypeCampaign, ccID, uuid.Nil, nil, "", activeflow.WebhookMethodNone).
					Return(&activeflow.Activeflow{Identity: commonidentity.Identity{ID: afID}}, nil)
				m.req.EXPECT().FlowV1ActiveflowExecute(ctx, afID).Return(errors.New("execute error"))
				m.campaigncall.EXPECT().Done(ctx, ccID, campaigncall.ResultFail).Return(nil, nil)
				expectStopAbort(ctx, m, id)
			},
		},
		{
			name: "flow type follow-up request failure stops the execution",
			setup: func(ctx context.Context, m *executeMocks) {
				ccID := uuid.FromStringOrNil("a1000000-0000-4000-8000-0000000000d1")
				afID := uuid.FromStringOrNil("a1000000-0000-4000-8000-0000000000d2")
				cc := &campaigncall.Campaigncall{Identity: commonidentity.Identity{ID: ccID, CustomerID: customerID}, ActiveflowID: afID, FlowID: flowID}
				m.db.EXPECT().CampaignGet(ctx, id).Return(newCampaign(nil), nil)
				expectPlanAndTarget(ctx, m, available, nil)
				m.util.EXPECT().UUIDCreate().Return(afID)
				m.campaigncall.EXPECT().Create(ctx, customerID, id, outplanID, outdialID, targetID, uuid.Nil,
					afID, flowID, campaigncall.ReferenceTypeFlow, uuid.Nil, plan.Source, dest, 0, 1).Return(cc, nil)
				m.campaigncall.EXPECT().Progressing(ctx, ccID).Return(cc, nil)
				m.req.EXPECT().FlowV1ActiveflowCreate(ctx, afID, customerID, flowID, activeflow.ReferenceTypeCampaign, ccID, uuid.Nil, nil, "", activeflow.WebhookMethodNone).
					Return(&activeflow.Activeflow{Identity: commonidentity.Identity{ID: afID}}, nil)
				m.req.EXPECT().FlowV1ActiveflowExecute(ctx, afID).Return(nil)
				m.req.EXPECT().CampaignV1CampaignExecute(ctx, id, 500).Return(errors.New("rpc error"))
				expectStopAbort(ctx, m, id)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newExecuteTestHandler(mc)
			ctx := context.Background()

			tt.setup(ctx, m)

			h.Execute(ctx, id)
		})
	}
}

func Test_Execute_call_type(t *testing.T) {
	id := uuid.FromStringOrNil("b1000000-0000-4000-8000-000000000001")
	customerID := uuid.FromStringOrNil("b1000000-0000-4000-8000-000000000002")
	outplanID := uuid.FromStringOrNil("b1000000-0000-4000-8000-000000000003")
	outdialID := uuid.FromStringOrNil("b1000000-0000-4000-8000-000000000004")
	flowID := uuid.FromStringOrNil("b1000000-0000-4000-8000-000000000006")
	targetID := uuid.FromStringOrNil("b1000000-0000-4000-8000-000000000007")
	callID := uuid.FromStringOrNil("b1000000-0000-4000-8000-0000000000a1")
	activeflowID := uuid.FromStringOrNil("b1000000-0000-4000-8000-0000000000a2")
	ccID := uuid.FromStringOrNil("b1000000-0000-4000-8000-0000000000a3")

	c := &campaign.Campaign{
		Identity:  commonidentity.Identity{ID: id, CustomerID: customerID},
		OutplanID: outplanID,
		OutdialID: outdialID,
		FlowID:    flowID,
		Status:    campaign.StatusRun,
		Type:      campaign.TypeCall,
	}
	plan := &outplan.Outplan{
		Identity:     commonidentity.Identity{ID: outplanID},
		Source:       &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550000001"},
		MaxTryCount1: 3,
	}
	dest := &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550000003"}
	// the first destination is exhausted, the second one is used as the 2nd try.
	target := omoutdialtarget.OutdialTarget{
		ID:           targetID,
		Destination0: &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550000009"},
		TryCount0:    0,
		Destination1: dest,
		TryCount1:    1,
	}
	cc := &campaigncall.Campaigncall{Identity: commonidentity.Identity{ID: ccID, CustomerID: customerID}}

	tests := []struct {
		name  string
		setup func(ctx context.Context, m *executeMocks)
	}{
		{
			name: "call is created and the next execution is scheduled with 500 ms",
			setup: func(ctx context.Context, m *executeMocks) {
				m.campaigncall.EXPECT().Create(ctx, customerID, id, outplanID, outdialID, targetID, uuid.Nil,
					activeflowID, flowID, campaigncall.ReferenceTypeCall, callID, plan.Source, dest, 1, 2).Return(cc, nil)
				m.req.EXPECT().CallV1CallCreateWithID(ctx, callID, customerID, flowID, activeflowID, uuid.Nil, plan.Source, dest,
					uuid.Nil, false, false, "", nil, nil).Return(&cmcall.Call{Identity: commonidentity.Identity{ID: callID}}, nil)
				m.req.EXPECT().CampaignV1CampaignExecute(ctx, id, 500).Return(nil)
			},
		},
		{
			name: "campaigncall create error stops the execution without creating a call",
			setup: func(ctx context.Context, m *executeMocks) {
				m.campaigncall.EXPECT().Create(ctx, customerID, id, outplanID, outdialID, targetID, uuid.Nil,
					activeflowID, flowID, campaigncall.ReferenceTypeCall, callID, plan.Source, dest, 1, 2).Return(nil, errors.New("create error"))
				expectStopAbort(ctx, m, id)
			},
		},
		{
			name: "call create error marks the campaigncall as fail and stops the execution",
			setup: func(ctx context.Context, m *executeMocks) {
				m.campaigncall.EXPECT().Create(ctx, customerID, id, outplanID, outdialID, targetID, uuid.Nil,
					activeflowID, flowID, campaigncall.ReferenceTypeCall, callID, plan.Source, dest, 1, 2).Return(cc, nil)
				m.req.EXPECT().CallV1CallCreateWithID(ctx, callID, customerID, flowID, activeflowID, uuid.Nil, plan.Source, dest,
					uuid.Nil, false, false, "", nil, nil).Return(nil, errors.New("call error"))
				m.campaigncall.EXPECT().Done(ctx, ccID, campaigncall.ResultFail).Return(nil, nil)
				expectStopAbort(ctx, m, id)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newExecuteTestHandler(mc)
			ctx := context.Background()

			m.db.EXPECT().CampaignGet(ctx, id).Return(c, nil)
			m.outplan.EXPECT().Get(ctx, outplanID).Return(plan, nil)
			m.req.EXPECT().OutdialV1OutdialtargetGetsAvailable(ctx, outdialID, 0, 3, 0, 0, 0, 1).Return([]omoutdialtarget.OutdialTarget{target}, nil)

			// callID is generated first and the activeflow id second.
			gomock.InOrder(
				m.util.EXPECT().UUIDCreate().Return(callID),
				m.util.EXPECT().UUIDCreate().Return(activeflowID),
			)

			tt.setup(ctx, m)

			h.Execute(ctx, id)
		})
	}
}

func Test_isDialable_lookup_errors(t *testing.T) {
	campaignID := uuid.FromStringOrNil("c1000000-0000-4000-8000-000000000001")
	queueID := uuid.FromStringOrNil("c1000000-0000-4000-8000-000000000002")

	tests := []struct {
		name         string
		serviceLevel int
		agents       []amagent.Agent
		agentsErr    error
		dialings     []*campaigncall.Campaigncall
		dialingsErr  error
		expectRes    bool
	}{
		{
			name:      "agent lookup error is not dialable",
			agentsErr: errors.New("queue error"),
			expectRes: false,
		},
		{
			name:         "dialing lookup error is not dialable",
			serviceLevel: 100,
			agents:       []amagent.Agent{{}},
			dialingsErr:  errors.New("db error"),
			expectRes:    false,
		},
		{
			name:         "capacity equal to dialings is not dialable",
			serviceLevel: 100,
			agents:       []amagent.Agent{{}, {}},
			dialings:     []*campaigncall.Campaigncall{{}, {}},
			expectRes:    false,
		},
		{
			name:         "capacity above dialings is dialable",
			serviceLevel: 100,
			agents:       []amagent.Agent{{}, {}},
			dialings:     []*campaigncall.Campaigncall{{}},
			expectRes:    true,
		},
		{
			name:         "service level scales the capacity down",
			serviceLevel: 50,
			agents:       []amagent.Agent{{}, {}},
			dialings:     []*campaigncall.Campaigncall{{}},
			expectRes:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newExecuteTestHandler(mc)
			ctx := context.Background()

			m.req.EXPECT().QueueV1QueueGetAgents(ctx, queueID, gomock.Any()).Return(tt.agents, tt.agentsErr)
			if tt.agentsErr == nil {
				m.campaigncall.EXPECT().ListByCampaignIDAndStatus(ctx, campaignID, campaigncall.StatusDialing, "", uint64(100)).Return(tt.dialings, tt.dialingsErr)
			}

			if res := h.isDialable(ctx, campaignID, queueID, tt.serviceLevel); res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}
