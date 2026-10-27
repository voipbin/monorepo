package queuecallhandler

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	amagent "monorepo/bin-agent-manager/models/agent"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	cucustomer "monorepo/bin-customer-manager/models/customer"
	fmflow "monorepo/bin-flow-manager/models/flow"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-queue-manager/models/queue"
	"monorepo/bin-queue-manager/models/queuecall"
	"monorepo/bin-queue-manager/pkg/cachehandler"
	"monorepo/bin-queue-manager/pkg/dbhandler"
	"monorepo/bin-queue-manager/pkg/queuehandler"
)

var errBranchTest = errors.New("simulated dependency failure")

// logSpy collects the messages written to the standard logger and restores the
// previous hooks when the test ends.
type logSpy struct {
	messages []string
}

func (s *logSpy) Levels() []logrus.Level { return logrus.AllLevels }

func (s *logSpy) Fire(e *logrus.Entry) error {
	s.messages = append(s.messages, e.Message)
	return nil
}

func (s *logSpy) has(text string) bool {
	for _, m := range s.messages {
		if strings.Contains(m, text) {
			return true
		}
	}
	return false
}

func spyLogs(t *testing.T) *logSpy {
	t.Helper()

	prev := make(logrus.LevelHooks)
	for level, hooks := range logrus.StandardLogger().Hooks {
		prev[level] = append([]logrus.Hook{}, hooks...)
	}

	prevLevel := logrus.GetLevel()
	logrus.SetLevel(logrus.DebugLevel)

	spy := &logSpy{}
	logrus.AddHook(spy)
	t.Cleanup(func() {
		logrus.StandardLogger().ReplaceHooks(prev)
		logrus.SetLevel(prevLevel)
	})

	return spy
}

func Test_EventCUCustomerDeleted_branches(t *testing.T) {
	customerID := uuid.FromStringOrNil("6a1b2c3d-5001-11f1-8c00-000000000001")
	filters := map[queuecall.Field]any{
		queuecall.FieldCustomerID: customerID,
		queuecall.FieldDeleted:    false,
	}
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	t.Run("list failure is returned and nothing is touched", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newQCTestHandler(mc)
		ctx := context.Background()

		m.db.EXPECT().QueuecallList(ctx, uint64(1000), "", filters).Return(nil, errBranchTest)

		err := h.EventCUCustomerDeleted(ctx, &cucustomer.Customer{ID: customerID})
		if !errors.Is(err, errBranchTest) {
			t.Errorf("Wrong match. expect: %v, got: %v", errBranchTest, err)
		}
	})

	t.Run("a queuecall that is kicked but cannot be deleted does not stop the sweep", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newQCTestHandler(mc)
		ctx := context.Background()
		spy := spyLogs(t)

		qc := newQC(queuecall.StatusWaiting, queuecall.ReferenceTypeCall)

		m.db.EXPECT().QueuecallList(ctx, uint64(1000), "", filters).Return([]*queuecall.Queuecall{qc}, nil)

		// kickForce: read, stop the flow, read again, abandon (the CAS is lost, so the current row is read once more)
		m.db.EXPECT().QueuecallGet(ctx, tQueuecallID).Return(qc, nil).Times(3)
		m.req.EXPECT().FlowV1ActiveflowServiceStop(ctx, tActiveflow, tQueuecallID, 0).Return(nil)
		m.util.EXPECT().TimeNow().Return(&now)
		m.db.EXPECT().QueuecallSetStatusAbandoned(ctx, tQueuecallID, gomock.Any(), &now).Return(int64(0), nil)

		m.db.EXPECT().QueuecallDelete(ctx, tQueuecallID).Return(errBranchTest)

		if err := h.EventCUCustomerDeleted(ctx, &cucustomer.Customer{ID: customerID}); err != nil {
			t.Errorf("Wrong match. expect: nil, got: %v", err)
		}
		if !spy.has("Could not delete queuecall info") {
			t.Errorf("Wrong match. the delete failure was not reported: %v", spy.messages)
		}
		if spy.has("Could not kick out the queuecall") {
			t.Errorf("Wrong match. the kick must have succeeded: %v", spy.messages)
		}
	})
}

func Test_HealthCheck_retryExceededKicksTheQueuecall(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, m := newQCTestHandler(mc)
	ctx := context.Background()
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	qc := newQC(queuecall.StatusWaiting, queuecall.ReferenceTypeCall)

	// the health check read, then kickForce: read, stop the flow, read again, abandon, read the current row
	m.db.EXPECT().QueuecallGet(ctx, tQueuecallID).Return(qc, nil).Times(4)
	m.req.EXPECT().FlowV1ActiveflowServiceStop(ctx, tActiveflow, tQueuecallID, 0).Return(nil)
	m.util.EXPECT().TimeNow().Return(&now)
	m.db.EXPECT().QueuecallSetStatusAbandoned(ctx, tQueuecallID, gomock.Any(), &now).Return(int64(0), nil)
	// the health check is not rescheduled once the queuecall is kicked

	h.HealthCheck(ctx, tQueuecallID, defaultHealthCheckMaxRetryCount+1)
}

func Test_Kick_statusUpdateFails(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, m := newQCTestHandler(mc)
	ctx := context.Background()
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	qc := newQC(queuecall.StatusWaiting, queuecall.ReferenceTypeCall)

	m.db.EXPECT().QueuecallGet(ctx, tQueuecallID).Return(qc, nil).Times(2)
	m.req.EXPECT().FlowV1ActiveflowServiceStop(ctx, tActiveflow, tQueuecallID, 0).Return(nil)
	m.util.EXPECT().TimeNow().Return(&now)
	m.db.EXPECT().QueuecallSetStatusAbandoned(ctx, tQueuecallID, gomock.Any(), &now).Return(int64(0), errBranchTest)

	res, err := h.Kick(ctx, tQueuecallID)
	if res != nil || !errors.Is(err, errBranchTest) {
		t.Errorf("Wrong match. expect: (nil, %v), got: (%v, %v)", errBranchTest, res, err)
	}
}

func Test_kickForce_errors(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	tests := []struct {
		name string

		setup func(ctx context.Context, m *qcMocks)
	}{
		{
			name: "lookup fails",
			setup: func(ctx context.Context, m *qcMocks) {
				m.db.EXPECT().QueuecallGet(ctx, tQueuecallID).Return(nil, errBranchTest)
			},
		},
		{
			name: "abandon fails for a waiting queuecall",
			setup: func(ctx context.Context, m *qcMocks) {
				m.db.EXPECT().QueuecallGet(ctx, tQueuecallID).Return(newQC(queuecall.StatusWaiting, queuecall.ReferenceTypeCall), nil).Times(2)
				m.req.EXPECT().FlowV1ActiveflowServiceStop(ctx, tActiveflow, tQueuecallID, 0).Return(nil)
				m.util.EXPECT().TimeNow().Return(&now)
				m.db.EXPECT().QueuecallSetStatusAbandoned(ctx, tQueuecallID, gomock.Any(), &now).Return(int64(0), errBranchTest)
			},
		},
		{
			name: "finish fails for a queuecall in service",
			setup: func(ctx context.Context, m *qcMocks) {
				m.db.EXPECT().QueuecallGet(ctx, tQueuecallID).Return(newQC(queuecall.StatusService, queuecall.ReferenceTypeCall), nil).Times(2)
				m.req.EXPECT().FlowV1ActiveflowServiceStop(ctx, tActiveflow, tQueuecallID, 0).Return(errBranchTest)
				m.util.EXPECT().TimeNow().Return(&now)
				m.db.EXPECT().QueuecallSetStatusDone(ctx, tQueuecallID, gomock.Any(), &now).Return(int64(0), errBranchTest)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newQCTestHandler(mc)
			ctx := context.Background()
			tt.setup(ctx, m)

			res, err := h.kickForce(ctx, tQueuecallID)
			if res != nil || !errors.Is(err, errBranchTest) {
				t.Errorf("Wrong match. expect: (nil, %v), got: (%v, %v)", errBranchTest, res, err)
			}
		})
	}
}

func Test_Timeout_kicksTheQueuecall(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	tests := []struct {
		name string

		status queuecall.Status
		run    func(ctx context.Context, h *queuecallHandler)

		expectGets int
		setup      func(ctx context.Context, m *qcMocks)
	}{
		{
			name:   "wait timeout abandons a waiting queuecall",
			status: queuecall.StatusWaiting,
			run: func(ctx context.Context, h *queuecallHandler) {
				h.TimeoutWait(ctx, tQueuecallID)
			},
			// timeout read, kick read, fresh read, and the current row after the lost CAS
			expectGets: 4,
			setup: func(ctx context.Context, m *qcMocks) {
				m.util.EXPECT().TimeNow().Return(&now)
				m.db.EXPECT().QueuecallSetStatusAbandoned(ctx, tQueuecallID, gomock.Any(), &now).Return(int64(0), nil)
			},
		},
		{
			name:   "service timeout leaves the teardown to the confbridge event",
			status: queuecall.StatusService,
			run: func(ctx context.Context, h *queuecallHandler) {
				h.TimeoutService(ctx, tQueuecallID)
			},
			// timeout read, kick read, fresh read
			expectGets: 3,
			setup:      func(ctx context.Context, m *qcMocks) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newQCTestHandler(mc)
			ctx := context.Background()
			spy := spyLogs(t)

			qc := newQC(tt.status, queuecall.ReferenceTypeCall)
			m.db.EXPECT().QueuecallGet(ctx, tQueuecallID).Return(qc, nil).Times(tt.expectGets)
			m.req.EXPECT().FlowV1ActiveflowServiceStop(ctx, tActiveflow, tQueuecallID, 0).Return(nil)
			tt.setup(ctx, m)

			tt.run(ctx, h)

			if !spy.has("Kicked out the queuecall timeout") {
				t.Errorf("Wrong match. the kick was not completed: %v", spy.messages)
			}
		})
	}
}

func Test_Execute_reserveFails(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, m := newQCTestHandler(mc)
	ctx := context.Background()

	agentID := uuid.FromStringOrNil("6a1b2c3d-5002-11f1-8c00-000000000002")
	flowID := uuid.FromStringOrNil("6a1b2c3d-5003-11f1-8c00-000000000003")
	qc := newQC(queuecall.StatusWaiting, queuecall.ReferenceTypeCall)
	flow := &fmflow.Flow{Identity: commonidentity.Identity{ID: flowID}}

	m.db.EXPECT().QueuecallGet(ctx, tQueuecallID).Return(qc, nil)
	m.req.EXPECT().FlowV1FlowCreate(ctx, qc.CustomerID, fmflow.TypeFlow, gomock.Any(), gomock.Any(), gomock.Any(), uuid.Nil, false).Return(flow, nil)
	// no dial and no release follow a reservation that was never made
	m.req.EXPECT().AgentV1AgentReserve(ctx, agentID, "queuecall", tQueuecallID).Return(false, errBranchTest)

	res, err := h.Execute(ctx, tQueuecallID, agentID)
	if res != nil || !errors.Is(err, errBranchTest) {
		t.Errorf("Wrong match. expect: (nil, %v), got: (%v, %v)", errBranchTest, res, err)
	}
}

func Test_matchWaitingQueuecall_picksAnAgent(t *testing.T) {
	agent := &amagent.Agent{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("6a1b2c3d-5004-11f1-8c00-000000000004")}}
	queueID := uuid.FromStringOrNil("6a1b2c3d-5005-11f1-8c00-000000000005")

	tests := []struct {
		name string

		queueResult *queue.Queue
		queueErr    error
	}{
		{
			// with a single agent the pick is deterministic; this case drives the routing method check, not the randomness
			name:        "random routing method with a single agent",
			queueResult: &queue.Queue{Identity: commonidentity.Identity{ID: queueID}, RoutingMethod: queue.RoutingMethodRandom},
		},
		{
			name:     "the first agent is used when the queue cannot be read",
			queueErr: errBranchTest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newQCTestHandler(mc)
			ctx := context.Background()
			spy := spyLogs(t)

			qc := newQC(queuecall.StatusWaiting, queuecall.ReferenceTypeCall)
			qc.QueueID = queueID

			m.queue.EXPECT().GetAgents(ctx, queueID, amagent.StatusAvailable).Return([]amagent.Agent{*agent}, nil)
			m.queue.EXPECT().Get(ctx, queueID).Return(tt.queueResult, tt.queueErr)
			// the single available agent is the target either way, and the execution stops at the missing queuecall
			m.db.EXPECT().QueuecallGet(ctx, tQueuecallID).Return(nil, errBranchTest)

			h.matchWaitingQueuecall(ctx, qc)

			if !spy.has("Could not execute the match") {
				t.Errorf("Wrong match. the failed execution was not reported: %v", spy.messages)
			}
		})
	}
}

func Test_Reconcile_branches(t *testing.T) {
	cutoff := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	tests := []struct {
		name string

		setup func(ctx context.Context, m *qcMocks, cache *cachehandler.MockCacheHandler)

		expectLog string
	}{
		{
			name: "the lease release failure is only logged",
			setup: func(ctx context.Context, m *qcMocks, cache *cachehandler.MockCacheHandler) {
				cache.EXPECT().ReconcileLockAcquire(ctx, gomock.Any(), reconcileLockTTL).Return(true, nil)
				m.util.EXPECT().TimeNowAdd(-connectingStaleAfter).Return(&cutoff)
				m.db.EXPECT().QueuecallListConnectingStale(ctx, cutoff, uint64(reconcileScanLimit)).Return(nil, nil)
				m.db.EXPECT().QueuecallListWaitingOldest(ctx, uint64(reconcileScanLimit)).Return(nil, nil)
				cache.EXPECT().ReconcileLockRelease(ctx).Return(errBranchTest)
			},
			expectLog: "Could not release the reconcile lease",
		},
		{
			name: "the stale scan failure skips the rollback and the waiting scan still runs",
			setup: func(ctx context.Context, m *qcMocks, cache *cachehandler.MockCacheHandler) {
				cache.EXPECT().ReconcileLockAcquire(ctx, gomock.Any(), reconcileLockTTL).Return(true, nil)
				m.util.EXPECT().TimeNowAdd(-connectingStaleAfter).Return(&cutoff)
				m.db.EXPECT().QueuecallListConnectingStale(ctx, cutoff, uint64(reconcileScanLimit)).Return(nil, errBranchTest)
				m.db.EXPECT().QueuecallListWaitingOldest(ctx, uint64(reconcileScanLimit)).Return(nil, nil)
				cache.EXPECT().ReconcileLockRelease(ctx).Return(nil)
			},
			expectLog: "Could not list connecting-stale queuecalls",
		},
		{
			name: "a failing rollback does not stop the pass",
			setup: func(ctx context.Context, m *qcMocks, cache *cachehandler.MockCacheHandler) {
				cache.EXPECT().ReconcileLockAcquire(ctx, gomock.Any(), reconcileLockTTL).Return(true, nil)
				m.util.EXPECT().TimeNowAdd(-connectingStaleAfter).Return(&cutoff)
				m.db.EXPECT().QueuecallListConnectingStale(ctx, cutoff, uint64(reconcileScanLimit)).Return([]*queuecall.Queuecall{newQC(queuecall.StatusConnecting, queuecall.ReferenceTypeCall)}, nil)
				m.db.EXPECT().QueuecallSetStatusWaitingIfConnecting(ctx, tQueuecallID).Return(int64(0), errBranchTest)
				m.db.EXPECT().QueuecallListWaitingOldest(ctx, uint64(reconcileScanLimit)).Return(nil, nil)
				cache.EXPECT().ReconcileLockRelease(ctx).Return(nil)
			},
			expectLog: "Could not roll back the stale connecting queuecall",
		},
		{
			name: "the waiting scan failure is only logged",
			setup: func(ctx context.Context, m *qcMocks, cache *cachehandler.MockCacheHandler) {
				cache.EXPECT().ReconcileLockAcquire(ctx, gomock.Any(), reconcileLockTTL).Return(true, nil)
				m.util.EXPECT().TimeNowAdd(-connectingStaleAfter).Return(&cutoff)
				m.db.EXPECT().QueuecallListConnectingStale(ctx, cutoff, uint64(reconcileScanLimit)).Return(nil, nil)
				m.db.EXPECT().QueuecallListWaitingOldest(ctx, uint64(reconcileScanLimit)).Return(nil, errBranchTest)
				cache.EXPECT().ReconcileLockRelease(ctx).Return(nil)
			},
			expectLog: "Could not list waiting queuecalls",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newQCTestHandler(mc)
			cache := cachehandler.NewMockCacheHandler(mc)
			h.cache = cache
			ctx := context.Background()
			spy := spyLogs(t)

			tt.setup(ctx, m, cache)

			h.Reconcile(ctx)

			if !spy.has(tt.expectLog) {
				t.Errorf("Wrong match. expect the log %q, got: %v", tt.expectLog, spy.messages)
			}
		})
	}
}

func Test_constructors_wireTheDependencies(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockQueue := queuehandler.NewMockQueueHandler(mc)

	h, ok := NewQueuecallHandler(mockReq, mockDB, mockCache, mockNotify, mockQueue).(*queuecallHandler)
	if !ok {
		t.Fatalf("Wrong match. the constructor must return the queuecall handler")
	}

	if h.reqHandler != mockReq || h.db != mockDB || h.cache != mockCache || h.notifyhandler != mockNotify || h.queueHandler != mockQueue {
		t.Errorf("Wrong match. a dependency was not wired: %+v", h)
	}
	if _, isReal := h.utilHandler.(*utilhandler.MockUtilHandler); isReal || h.utilHandler == nil {
		t.Errorf("Wrong match. the real util handler must be created, got: %T", h.utilHandler)
	}
}
