package queuecallhandler

import (
	"context"
	"errors"
	"testing"

	cmcall "monorepo/bin-call-manager/models/call"
	cmgroupcall "monorepo/bin-call-manager/models/groupcall"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	fmflow "monorepo/bin-flow-manager/models/flow"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-queue-manager/models/queuecall"
	"monorepo/bin-queue-manager/pkg/dbhandler"
)

var errExecTest = errors.New("simulated dependency failure")

// Test_Execute_errors checks every failure branch of Execute: the error is
// returned to the caller and the unwind calls (hangup, reservation release)
// are made exactly where the match had progressed far enough to need them.
// A call that is not declared in a case is not expected to happen.
func Test_Execute_errors(t *testing.T) {
	id := uuid.FromStringOrNil("5c1d2e3f-2001-11f1-9b00-000000000001")
	agentID := uuid.FromStringOrNil("5c1d2e3f-2002-11f1-9b00-000000000002")
	customerID := uuid.FromStringOrNil("5c1d2e3f-2003-11f1-9b00-000000000003")
	groupcallID := uuid.FromStringOrNil("5c1d2e3f-2004-11f1-9b00-000000000004")
	flowID := uuid.FromStringOrNil("5c1d2e3f-2005-11f1-9b00-000000000005")
	activeflowID := uuid.FromStringOrNil("5c1d2e3f-2006-11f1-9b00-000000000006")
	actionID := uuid.FromStringOrNil("5c1d2e3f-2007-11f1-9b00-000000000007")

	qc := &queuecall.Queuecall{
		Identity:              commonidentity.Identity{ID: id, CustomerID: customerID},
		ReferenceActiveflowID: activeflowID,
		ForwardActionID:       actionID,
		Status:                queuecall.StatusWaiting,
	}
	flow := &fmflow.Flow{Identity: commonidentity.Identity{ID: flowID}}

	tests := []struct {
		name string

		setup func(ctx context.Context, mockDB *dbhandler.MockDBHandler, mockReq *requesthandler.MockRequestHandler, mockNotify *notifyhandler.MockNotifyHandler)
	}{
		{
			name: "queuecall lookup fails",
			setup: func(ctx context.Context, mockDB *dbhandler.MockDBHandler, mockReq *requesthandler.MockRequestHandler, mockNotify *notifyhandler.MockNotifyHandler) {
				mockDB.EXPECT().QueuecallGet(ctx, id).Return(nil, errExecTest)
			},
		},
		{
			name: "flow creation fails",
			setup: func(ctx context.Context, mockDB *dbhandler.MockDBHandler, mockReq *requesthandler.MockRequestHandler, mockNotify *notifyhandler.MockNotifyHandler) {
				mockDB.EXPECT().QueuecallGet(ctx, id).Return(qc, nil)
				mockReq.EXPECT().FlowV1FlowCreate(ctx, customerID, fmflow.TypeFlow, gomock.Any(), gomock.Any(), gomock.Any(), uuid.Nil, false).Return(nil, errExecTest)
			},
		},
		{
			name: "no groupcall is produced, release fails too",
			setup: func(ctx context.Context, mockDB *dbhandler.MockDBHandler, mockReq *requesthandler.MockRequestHandler, mockNotify *notifyhandler.MockNotifyHandler) {
				mockDB.EXPECT().QueuecallGet(ctx, id).Return(qc, nil)
				mockReq.EXPECT().FlowV1FlowCreate(ctx, customerID, fmflow.TypeFlow, gomock.Any(), gomock.Any(), gomock.Any(), uuid.Nil, false).Return(flow, nil)
				mockReq.EXPECT().AgentV1AgentReserve(ctx, agentID, "queuecall", id).Return(true, nil)
				mockReq.EXPECT().CallV1CallsCreate(ctx, customerID, flowID, qc.ReferenceID, &qc.Source, gomock.Any(), false, false, "", nil, nil).Return([]*cmcall.Call{}, []*cmgroupcall.Groupcall{}, nil)
				mockReq.EXPECT().AgentV1AgentReserveRelease(ctx, agentID, id).Return(errors.New("release failed"))
			},
		},
		{
			name: "dial fails, release fails too",
			setup: func(ctx context.Context, mockDB *dbhandler.MockDBHandler, mockReq *requesthandler.MockRequestHandler, mockNotify *notifyhandler.MockNotifyHandler) {
				mockDB.EXPECT().QueuecallGet(ctx, id).Return(qc, nil)
				mockReq.EXPECT().FlowV1FlowCreate(ctx, customerID, fmflow.TypeFlow, gomock.Any(), gomock.Any(), gomock.Any(), uuid.Nil, false).Return(flow, nil)
				mockReq.EXPECT().AgentV1AgentReserve(ctx, agentID, "queuecall", id).Return(true, nil)
				mockReq.EXPECT().CallV1CallsCreate(ctx, customerID, flowID, qc.ReferenceID, &qc.Source, gomock.Any(), false, false, "", nil, nil).Return(nil, nil, errExecTest)
				mockReq.EXPECT().AgentV1AgentReserveRelease(ctx, agentID, id).Return(errors.New("release failed"))
			},
		},
		{
			name: "connecting update fails, reservation is released",
			setup: func(ctx context.Context, mockDB *dbhandler.MockDBHandler, mockReq *requesthandler.MockRequestHandler, mockNotify *notifyhandler.MockNotifyHandler) {
				mockDB.EXPECT().QueuecallGet(ctx, id).Return(qc, nil)
				mockReq.EXPECT().FlowV1FlowCreate(ctx, customerID, fmflow.TypeFlow, gomock.Any(), gomock.Any(), gomock.Any(), uuid.Nil, false).Return(flow, nil)
				mockReq.EXPECT().AgentV1AgentReserve(ctx, agentID, "queuecall", id).Return(true, nil)
				mockReq.EXPECT().CallV1CallsCreate(ctx, customerID, flowID, qc.ReferenceID, &qc.Source, gomock.Any(), false, false, "", nil, nil).Return([]*cmcall.Call{}, []*cmgroupcall.Groupcall{{Identity: commonidentity.Identity{ID: groupcallID}}}, nil)
				mockDB.EXPECT().QueuecallSetStatusConnecting(ctx, id, agentID, groupcallID).Return(int64(0), errExecTest)
				mockReq.EXPECT().AgentV1AgentReserveRelease(ctx, agentID, id).Return(errors.New("release failed"))
			},
		},
		{
			name: "flow forward fails after the match is won",
			setup: func(ctx context.Context, mockDB *dbhandler.MockDBHandler, mockReq *requesthandler.MockRequestHandler, mockNotify *notifyhandler.MockNotifyHandler) {
				mockDB.EXPECT().QueuecallGet(ctx, id).Return(qc, nil).Times(2)
				mockReq.EXPECT().FlowV1FlowCreate(ctx, customerID, fmflow.TypeFlow, gomock.Any(), gomock.Any(), gomock.Any(), uuid.Nil, false).Return(flow, nil)
				mockReq.EXPECT().AgentV1AgentReserve(ctx, agentID, "queuecall", id).Return(true, nil)
				mockReq.EXPECT().CallV1CallsCreate(ctx, customerID, flowID, qc.ReferenceID, &qc.Source, gomock.Any(), false, false, "", nil, nil).Return([]*cmcall.Call{}, []*cmgroupcall.Groupcall{{Identity: commonidentity.Identity{ID: groupcallID}}}, nil)
				mockDB.EXPECT().QueuecallSetStatusConnecting(ctx, id, agentID, groupcallID).Return(int64(1), nil)
				mockNotify.EXPECT().PublishWebhookEvent(ctx, customerID, queuecall.EventTypeQueuecallConnecting, qc)
				mockReq.EXPECT().FlowV1ActiveflowUpdateForwardActionID(ctx, activeflowID, actionID, true).Return(errExecTest)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &queuecallHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyhandler: mockNotify,
			}
			ctx := context.Background()

			tt.setup(ctx, mockDB, mockReq, mockNotify)

			res, err := h.Execute(ctx, id, agentID)
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: nil (res: %v)", res)
			}
			if res != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", res)
			}
		})
	}
}

// Test_Execute_entryCASLost_unwindFails verifies that the current queuecall is
// still returned when the unwind calls after a lost entry CAS fail.
func Test_Execute_entryCASLost_unwindFails(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	id := uuid.FromStringOrNil("5c1d2e3f-2011-11f1-9b00-000000000011")
	agentID := uuid.FromStringOrNil("5c1d2e3f-2012-11f1-9b00-000000000012")
	customerID := uuid.FromStringOrNil("5c1d2e3f-2013-11f1-9b00-000000000013")
	groupcallID := uuid.FromStringOrNil("5c1d2e3f-2014-11f1-9b00-000000000014")
	flowID := uuid.FromStringOrNil("5c1d2e3f-2015-11f1-9b00-000000000015")

	qc := &queuecall.Queuecall{
		Identity: commonidentity.Identity{ID: id, CustomerID: customerID},
		Status:   queuecall.StatusConnecting,
	}

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &queuecallHandler{db: mockDB, reqHandler: mockReq, notifyhandler: mockNotify}
	ctx := context.Background()

	mockDB.EXPECT().QueuecallGet(ctx, id).Return(qc, nil).Times(2)
	mockReq.EXPECT().FlowV1FlowCreate(ctx, customerID, fmflow.TypeFlow, gomock.Any(), gomock.Any(), gomock.Any(), uuid.Nil, false).Return(&fmflow.Flow{Identity: commonidentity.Identity{ID: flowID}}, nil)
	mockReq.EXPECT().AgentV1AgentReserve(ctx, agentID, "queuecall", id).Return(true, nil)
	mockReq.EXPECT().CallV1CallsCreate(ctx, customerID, flowID, qc.ReferenceID, &qc.Source, gomock.Any(), false, false, "", nil, nil).Return([]*cmcall.Call{}, []*cmgroupcall.Groupcall{{Identity: commonidentity.Identity{ID: groupcallID}}}, nil)
	mockDB.EXPECT().QueuecallSetStatusConnecting(ctx, id, agentID, groupcallID).Return(int64(0), nil)
	mockReq.EXPECT().CallV1GroupcallHangup(ctx, groupcallID).Return(nil, errExecTest)
	mockReq.EXPECT().AgentV1AgentReserveRelease(ctx, agentID, id).Return(errExecTest)

	res, err := h.Execute(ctx, id, agentID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res == nil || res.ID != id || res.Status != queuecall.StatusConnecting {
		t.Errorf("Wrong match. got: %v", res)
	}
}
