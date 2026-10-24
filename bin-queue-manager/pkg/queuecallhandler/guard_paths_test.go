package queuecallhandler

import (
	"context"
	"errors"
	"testing"
	"time"

	cmcall "monorepo/bin-call-manager/models/call"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-queue-manager/models/queuecall"
	"monorepo/bin-queue-manager/pkg/dbhandler"
	"monorepo/bin-queue-manager/pkg/queuehandler"
)

var (
	tQueuecallID = uuid.FromStringOrNil("5e3a7c10-9a33-11ee-b201-000000000001")
	tReferenceID = uuid.FromStringOrNil("5e3a7c10-9a33-11ee-b201-000000000002")
	tActiveflow  = uuid.FromStringOrNil("5e3a7c10-9a33-11ee-b201-000000000003")
	tConfbridge  = uuid.FromStringOrNil("5e3a7c10-9a33-11ee-b201-000000000004")
)

type qcMocks struct {
	util   *utilhandler.MockUtilHandler
	db     *dbhandler.MockDBHandler
	req    *requesthandler.MockRequestHandler
	notify *notifyhandler.MockNotifyHandler
	queue  *queuehandler.MockQueueHandler
}

func newQCTestHandler(mc *gomock.Controller) (*queuecallHandler, *qcMocks) {
	m := &qcMocks{
		util:   utilhandler.NewMockUtilHandler(mc),
		db:     dbhandler.NewMockDBHandler(mc),
		req:    requesthandler.NewMockRequestHandler(mc),
		notify: notifyhandler.NewMockNotifyHandler(mc),
		queue:  queuehandler.NewMockQueueHandler(mc),
	}
	return &queuecallHandler{
		utilHandler:   m.util,
		db:            m.db,
		reqHandler:    m.req,
		notifyhandler: m.notify,
		queueHandler:  m.queue,
	}, m
}

func newQC(status queuecall.Status, refType queuecall.ReferenceType) *queuecall.Queuecall {
	return &queuecall.Queuecall{
		Identity:              commonidentity.Identity{ID: tQueuecallID},
		ReferenceType:         refType,
		ReferenceID:           tReferenceID,
		ReferenceActiveflowID: tActiveflow,
		ConfbridgeID:          tConfbridge,
		Status:                status,
	}
}

func Test_HealthCheck(t *testing.T) {
	errGet := errors.New("get failed")
	errCall := errors.New("call lookup failed")

	tests := []struct {
		name string

		retryCount int
		setup      func(m *qcMocks, ctx context.Context)
	}{
		{
			name:       "queuecall lookup failure ends the check without rescheduling",
			retryCount: 0,
			setup: func(m *qcMocks, ctx context.Context) {
				m.db.EXPECT().QueuecallGet(ctx, tQueuecallID).Return(nil, errGet)
			},
		},
		{
			name:       "abandoned queuecall is not checked any more",
			retryCount: 0,
			setup: func(m *qcMocks, ctx context.Context) {
				m.db.EXPECT().QueuecallGet(ctx, tQueuecallID).Return(newQC(queuecall.StatusAbandoned, queuecall.ReferenceTypeCall), nil)
			},
		},
		{
			name:       "done queuecall is not checked any more",
			retryCount: 0,
			setup: func(m *qcMocks, ctx context.Context) {
				m.db.EXPECT().QueuecallGet(ctx, tQueuecallID).Return(newQC(queuecall.StatusDone, queuecall.ReferenceTypeCall), nil)
			},
		},
		{
			name:       "non-call reference is not checked",
			retryCount: 0,
			setup: func(m *qcMocks, ctx context.Context) {
				m.db.EXPECT().QueuecallGet(ctx, tQueuecallID).Return(newQC(queuecall.StatusWaiting, queuecall.ReferenceType("conference")), nil)
			},
		},
		{
			name:       "reference call lookup failure reschedules with an increased retry count",
			retryCount: 1,
			setup: func(m *qcMocks, ctx context.Context) {
				m.db.EXPECT().QueuecallGet(ctx, tQueuecallID).Return(newQC(queuecall.StatusWaiting, queuecall.ReferenceTypeCall), nil)
				m.req.EXPECT().CallV1CallGet(ctx, tReferenceID).Return(nil, errCall)
				m.req.EXPECT().QueueV1QueuecallHealthCheck(ctx, tQueuecallID, defaultHealthCheckDelay, 2).Return(nil)
			},
		},
		{
			name:       "hung up reference call reschedules with an increased retry count",
			retryCount: 0,
			setup: func(m *qcMocks, ctx context.Context) {
				m.db.EXPECT().QueuecallGet(ctx, tQueuecallID).Return(newQC(queuecall.StatusWaiting, queuecall.ReferenceTypeCall), nil)
				m.req.EXPECT().CallV1CallGet(ctx, tReferenceID).Return(&cmcall.Call{Status: cmcall.StatusHangup}, nil)
				m.req.EXPECT().QueueV1QueuecallHealthCheck(ctx, tQueuecallID, defaultHealthCheckDelay, 1).Return(nil)
			},
		},
		{
			name:       "retry count at the maximum still reschedules",
			retryCount: defaultHealthCheckMaxRetryCount,
			setup: func(m *qcMocks, ctx context.Context) {
				m.db.EXPECT().QueuecallGet(ctx, tQueuecallID).Return(newQC(queuecall.StatusWaiting, queuecall.ReferenceTypeCall), nil)
				m.req.EXPECT().CallV1CallGet(ctx, tReferenceID).Return(&cmcall.Call{Status: cmcall.StatusHangup}, nil)
				m.req.EXPECT().QueueV1QueuecallHealthCheck(ctx, tQueuecallID, defaultHealthCheckDelay, defaultHealthCheckMaxRetryCount+1).Return(nil)
			},
		},
		{
			name:       "healthy reference call resets the retry count",
			retryCount: 2,
			setup: func(m *qcMocks, ctx context.Context) {
				m.db.EXPECT().QueuecallGet(ctx, tQueuecallID).Return(newQC(queuecall.StatusWaiting, queuecall.ReferenceTypeCall), nil)
				m.req.EXPECT().CallV1CallGet(ctx, tReferenceID).Return(&cmcall.Call{Status: cmcall.StatusProgressing}, nil)
				m.req.EXPECT().QueueV1QueuecallHealthCheck(ctx, tQueuecallID, defaultHealthCheckDelay, 0).Return(nil)
			},
		},
		{
			name:       "retry count above the maximum force-kicks, which stops when the queuecall ended meanwhile",
			retryCount: defaultHealthCheckMaxRetryCount + 1,
			setup: func(m *qcMocks, ctx context.Context) {
				// first read in HealthCheck: still waiting; second read in kickForce: already done.
				gomock.InOrder(
					m.db.EXPECT().QueuecallGet(ctx, tQueuecallID).Return(newQC(queuecall.StatusWaiting, queuecall.ReferenceTypeCall), nil),
					m.db.EXPECT().QueuecallGet(ctx, tQueuecallID).Return(newQC(queuecall.StatusDone, queuecall.ReferenceTypeCall), nil),
				)
			},
		},
		{
			name:       "force-kick tolerates a failing flow stop and ends when the fresh read fails",
			retryCount: defaultHealthCheckMaxRetryCount + 1,
			setup: func(m *qcMocks, ctx context.Context) {
				gomock.InOrder(
					m.db.EXPECT().QueuecallGet(ctx, tQueuecallID).Return(newQC(queuecall.StatusWaiting, queuecall.ReferenceTypeCall), nil),
					m.db.EXPECT().QueuecallGet(ctx, tQueuecallID).Return(newQC(queuecall.StatusWaiting, queuecall.ReferenceTypeCall), nil),
					m.req.EXPECT().FlowV1ActiveflowServiceStop(ctx, tActiveflow, tQueuecallID, 0).Return(errors.New("stop failed")),
					m.db.EXPECT().QueuecallGet(ctx, tQueuecallID).Return(nil, errGet),
				)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newQCTestHandler(mc)
			ctx := context.Background()

			tt.setup(m, ctx)

			h.HealthCheck(ctx, tQueuecallID, tt.retryCount)
		})
	}
}

func Test_Get_errors(t *testing.T) {
	errDB := errors.New("db failure")

	t.Run("record-not-found becomes a typed NotFound error", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newQCTestHandler(mc)
		ctx := context.Background()

		m.db.EXPECT().QueuecallGet(ctx, tQueuecallID).Return(nil, dbhandler.ErrNotFound)

		res, err := h.Get(ctx, tQueuecallID)
		var ve *cerrors.VoipbinError
		if res != nil || !errors.As(err, &ve) || ve.Status != cerrors.StatusNotFound || ve.Reason != "QUEUECALL_NOT_FOUND" {
			t.Errorf("Wrong match. expect: (nil, NOT_FOUND/QUEUECALL_NOT_FOUND), got: (%v, %v)", res, err)
		}
		if !errors.Is(err, dbhandler.ErrNotFound) {
			t.Errorf("Wrong match. the cause must stay reachable. err: %v", err)
		}
	})

	t.Run("other db errors are wrapped and stay untyped", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newQCTestHandler(mc)
		ctx := context.Background()

		m.db.EXPECT().QueuecallGet(ctx, tQueuecallID).Return(nil, errDB)

		res, err := h.Get(ctx, tQueuecallID)
		var ve *cerrors.VoipbinError
		if res != nil || !errors.Is(err, errDB) || errors.As(err, &ve) {
			t.Errorf("Wrong match. expect: (nil, untyped wrapping %v), got: (%v, %v)", errDB, res, err)
		}
	})

	t.Run("GetByReferenceID wraps the db error", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newQCTestHandler(mc)
		ctx := context.Background()

		m.db.EXPECT().QueuecallGetByReferenceID(ctx, tReferenceID).Return(nil, errDB)

		res, err := h.GetByReferenceID(ctx, tReferenceID)
		if res != nil || !errors.Is(err, errDB) {
			t.Errorf("Wrong match. expect: (nil, %v), got: (%v, %v)", errDB, res, err)
		}
	})

	t.Run("List returns the db error unchanged", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newQCTestHandler(mc)
		ctx := context.Background()

		filters := map[queuecall.Field]any{queuecall.FieldDeleted: false}
		m.db.EXPECT().QueuecallList(ctx, uint64(10), "tok", filters).Return(nil, errDB)

		res, err := h.List(ctx, 10, "tok", filters)
		if res != nil || err != errDB {
			t.Errorf("Wrong match. expect: (nil, %v), got: (%v, %v)", errDB, res, err)
		}
	})
}

func Test_Kick_errors(t *testing.T) {
	errGet := errors.New("get failed")
	errStop := errors.New("stop failed")

	t.Run("lookup failure", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newQCTestHandler(mc)
		ctx := context.Background()

		m.db.EXPECT().QueuecallGet(ctx, tQueuecallID).Return(nil, errGet)

		res, err := h.Kick(ctx, tQueuecallID)
		if res != nil || !errors.Is(err, errGet) {
			t.Errorf("Wrong match. expect: (nil, %v), got: (%v, %v)", errGet, res, err)
		}
	})

	for _, status := range []queuecall.Status{queuecall.StatusDone, queuecall.StatusAbandoned} {
		t.Run("already "+string(status)+" is rejected without stopping the flow", func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newQCTestHandler(mc)
			ctx := context.Background()

			m.db.EXPECT().QueuecallGet(ctx, tQueuecallID).Return(newQC(status, queuecall.ReferenceTypeCall), nil)

			res, err := h.Kick(ctx, tQueuecallID)
			if res != nil || err == nil {
				t.Errorf("Wrong match. expect: (nil, error), got: (%v, %v)", res, err)
			}
		})
	}

	t.Run("flow stop failure aborts before any status change", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newQCTestHandler(mc)
		ctx := context.Background()

		m.db.EXPECT().QueuecallGet(ctx, tQueuecallID).Return(newQC(queuecall.StatusWaiting, queuecall.ReferenceTypeCall), nil)
		m.req.EXPECT().FlowV1ActiveflowServiceStop(ctx, tActiveflow, tQueuecallID, 0).Return(errStop)

		res, err := h.Kick(ctx, tQueuecallID)
		if res != nil || !errors.Is(err, errStop) {
			t.Errorf("Wrong match. expect: (nil, %v), got: (%v, %v)", errStop, res, err)
		}
	})

	t.Run("fresh read failure after the flow stop", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newQCTestHandler(mc)
		ctx := context.Background()

		gomock.InOrder(
			m.db.EXPECT().QueuecallGet(ctx, tQueuecallID).Return(newQC(queuecall.StatusWaiting, queuecall.ReferenceTypeCall), nil),
			m.req.EXPECT().FlowV1ActiveflowServiceStop(ctx, tActiveflow, tQueuecallID, 0).Return(nil),
			m.db.EXPECT().QueuecallGet(ctx, tQueuecallID).Return(nil, errGet),
		)

		res, err := h.Kick(ctx, tQueuecallID)
		if res != nil || !errors.Is(err, errGet) {
			t.Errorf("Wrong match. expect: (nil, %v), got: (%v, %v)", errGet, res, err)
		}
	})

	t.Run("a queuecall that reached service during the flow stop is returned untouched", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newQCTestHandler(mc)
		ctx := context.Background()

		serviced := newQC(queuecall.StatusService, queuecall.ReferenceTypeCall)
		gomock.InOrder(
			m.db.EXPECT().QueuecallGet(ctx, tQueuecallID).Return(newQC(queuecall.StatusWaiting, queuecall.ReferenceTypeCall), nil),
			m.req.EXPECT().FlowV1ActiveflowServiceStop(ctx, tActiveflow, tQueuecallID, 0).Return(nil),
			m.db.EXPECT().QueuecallGet(ctx, tQueuecallID).Return(serviced, nil),
		)

		res, err := h.Kick(ctx, tQueuecallID)
		if err != nil || res != serviced {
			t.Errorf("Wrong match. expect: (%v, nil), got: (%v, %v)", serviced, res, err)
		}
	})
}

func Test_KickByReferenceID_errors(t *testing.T) {
	errGet := errors.New("get failed")

	t.Run("lookup failure", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newQCTestHandler(mc)
		ctx := context.Background()

		m.db.EXPECT().QueuecallGetByReferenceID(ctx, tReferenceID).Return(nil, errGet)

		res, err := h.KickByReferenceID(ctx, tReferenceID)
		if res != nil || !errors.Is(err, errGet) {
			t.Errorf("Wrong match. expect: (nil, %v), got: (%v, %v)", errGet, res, err)
		}
	})

	t.Run("an already ended queuecall is returned without kicking", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newQCTestHandler(mc)
		ctx := context.Background()

		ended := newQC(queuecall.StatusDone, queuecall.ReferenceTypeCall)
		ended.TMEnd = timePtr(time.Date(2021, time.April, 18, 3, 22, 17, 0, time.UTC))
		m.db.EXPECT().QueuecallGetByReferenceID(ctx, tReferenceID).Return(ended, nil)

		res, err := h.KickByReferenceID(ctx, tReferenceID)
		if err != nil || res != ended {
			t.Errorf("Wrong match. expect: (%v, nil), got: (%v, %v)", ended, res, err)
		}
	})

	t.Run("kick failure of a running queuecall is returned", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newQCTestHandler(mc)
		ctx := context.Background()

		running := newQC(queuecall.StatusWaiting, queuecall.ReferenceTypeCall)
		gomock.InOrder(
			m.db.EXPECT().QueuecallGetByReferenceID(ctx, tReferenceID).Return(running, nil),
			m.db.EXPECT().QueuecallGet(ctx, tQueuecallID).Return(nil, errGet),
		)

		res, err := h.KickByReferenceID(ctx, tReferenceID)
		if res != nil || !errors.Is(err, errGet) {
			t.Errorf("Wrong match. expect: (nil, %v), got: (%v, %v)", errGet, res, err)
		}
	})
}

// Test_Timeout_ignored verifies that a timeout for a queuecall that is not in
// the matching status (or cannot be read) does not kick anything: the strict
// mocks have no flow-stop or status-update expectations.
func Test_Timeout_ignored(t *testing.T) {
	errGet := errors.New("get failed")

	tests := []struct {
		name string
		call func(h *queuecallHandler, ctx context.Context)
		resp func() (*queuecall.Queuecall, error)
	}{
		{"wait timeout, lookup failure", func(h *queuecallHandler, ctx context.Context) { h.TimeoutWait(ctx, tQueuecallID) }, func() (*queuecall.Queuecall, error) { return nil, errGet }},
		{"wait timeout, queuecall already in service", func(h *queuecallHandler, ctx context.Context) { h.TimeoutWait(ctx, tQueuecallID) }, func() (*queuecall.Queuecall, error) {
			return newQC(queuecall.StatusService, queuecall.ReferenceTypeCall), nil
		}},
		{"wait timeout, queuecall already done", func(h *queuecallHandler, ctx context.Context) { h.TimeoutWait(ctx, tQueuecallID) }, func() (*queuecall.Queuecall, error) {
			return newQC(queuecall.StatusDone, queuecall.ReferenceTypeCall), nil
		}},
		{"service timeout, lookup failure", func(h *queuecallHandler, ctx context.Context) { h.TimeoutService(ctx, tQueuecallID) }, func() (*queuecall.Queuecall, error) { return nil, errGet }},
		{"service timeout, queuecall still waiting", func(h *queuecallHandler, ctx context.Context) { h.TimeoutService(ctx, tQueuecallID) }, func() (*queuecall.Queuecall, error) {
			return newQC(queuecall.StatusWaiting, queuecall.ReferenceTypeCall), nil
		}},
		{"service timeout, queuecall already abandoned", func(h *queuecallHandler, ctx context.Context) { h.TimeoutService(ctx, tQueuecallID) }, func() (*queuecall.Queuecall, error) {
			return newQC(queuecall.StatusAbandoned, queuecall.ReferenceTypeCall), nil
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newQCTestHandler(mc)
			ctx := context.Background()

			qc, err := tt.resp()
			m.db.EXPECT().QueuecallGet(ctx, tQueuecallID).Return(qc, err)

			tt.call(h, ctx)
		})
	}
}

// Test_Event_ignored verifies the guard conditions of the call-manager event
// handlers: a missing, ended, serviced or foreign-confbridge queuecall must not
// be updated (the strict mocks have no update expectations).
func Test_Event_ignored(t *testing.T) {
	errGet := errors.New("get failed")
	otherConfbridge := uuid.FromStringOrNil("5e3a7c10-9a33-11ee-b201-0000000000ff")
	ended := timePtr(time.Date(2021, time.April, 18, 3, 22, 17, 0, time.UTC))

	withEnd := func(status queuecall.Status) *queuecall.Queuecall {
		qc := newQC(status, queuecall.ReferenceTypeCall)
		qc.TMEnd = ended
		return qc
	}

	tests := []struct {
		name string
		call func(h *queuecallHandler, ctx context.Context)
		resp func() (*queuecall.Queuecall, error)
	}{
		{"hangup, no queuecall", func(h *queuecallHandler, ctx context.Context) { h.EventCallCallHangup(ctx, tReferenceID) }, func() (*queuecall.Queuecall, error) { return nil, errGet }},
		{"hangup, already ended", func(h *queuecallHandler, ctx context.Context) { h.EventCallCallHangup(ctx, tReferenceID) }, func() (*queuecall.Queuecall, error) {
			return withEnd(queuecall.StatusAbandoned), nil
		}},
		{"hangup, in service is left to the confbridge handler", func(h *queuecallHandler, ctx context.Context) { h.EventCallCallHangup(ctx, tReferenceID) }, func() (*queuecall.Queuecall, error) {
			return newQC(queuecall.StatusService, queuecall.ReferenceTypeCall), nil
		}},
		{"confbridge joined, no queuecall", func(h *queuecallHandler, ctx context.Context) {
			h.EventCallConfbridgeJoined(ctx, tReferenceID, tConfbridge)
		}, func() (*queuecall.Queuecall, error) { return nil, errGet }},
		{"confbridge joined, already ended", func(h *queuecallHandler, ctx context.Context) {
			h.EventCallConfbridgeJoined(ctx, tReferenceID, tConfbridge)
		}, func() (*queuecall.Queuecall, error) { return withEnd(queuecall.StatusDone), nil }},
		{"confbridge joined, another confbridge", func(h *queuecallHandler, ctx context.Context) {
			h.EventCallConfbridgeJoined(ctx, tReferenceID, otherConfbridge)
		}, func() (*queuecall.Queuecall, error) {
			return newQC(queuecall.StatusConnecting, queuecall.ReferenceTypeCall), nil
		}},
		{"confbridge leaved, no queuecall", func(h *queuecallHandler, ctx context.Context) {
			h.EventCallConfbridgeLeaved(ctx, tReferenceID, tConfbridge)
		}, func() (*queuecall.Queuecall, error) { return nil, errGet }},
		{"confbridge leaved, already ended", func(h *queuecallHandler, ctx context.Context) {
			h.EventCallConfbridgeLeaved(ctx, tReferenceID, tConfbridge)
		}, func() (*queuecall.Queuecall, error) { return withEnd(queuecall.StatusDone), nil }},
		{"confbridge leaved, another confbridge", func(h *queuecallHandler, ctx context.Context) {
			h.EventCallConfbridgeLeaved(ctx, tReferenceID, otherConfbridge)
		}, func() (*queuecall.Queuecall, error) {
			return newQC(queuecall.StatusService, queuecall.ReferenceTypeCall), nil
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newQCTestHandler(mc)
			ctx := context.Background()

			qc, err := tt.resp()
			m.db.EXPECT().QueuecallGetByReferenceID(ctx, tReferenceID).Return(qc, err)

			tt.call(h, ctx)
		})
	}
}
