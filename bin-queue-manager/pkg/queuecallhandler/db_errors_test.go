package queuecallhandler

import (
	"context"
	"errors"
	"testing"
	"time"

	cmconfbridge "monorepo/bin-call-manager/models/confbridge"
	cmgroupcall "monorepo/bin-call-manager/models/groupcall"
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-queue-manager/models/queue"
	"monorepo/bin-queue-manager/models/queuecall"
)

var (
	tDBCustomerID = uuid.FromStringOrNil("7a1c0e10-9a33-11ee-b201-000000000011")
	tDBQueueID    = uuid.FromStringOrNil("7a1c0e10-9a33-11ee-b201-000000000012")
	tDBAgentID    = uuid.FromStringOrNil("7a1c0e10-9a33-11ee-b201-000000000013")
	tDBGroupcall  = uuid.FromStringOrNil("7a1c0e10-9a33-11ee-b201-000000000014")
)

// newDBQC returns a queuecall with every field the update handlers branch on.
func newDBQC() *queuecall.Queuecall {
	qc := newQC(queuecall.StatusService, queuecall.ReferenceTypeCall)
	qc.CustomerID = tDBCustomerID
	qc.QueueID = tDBQueueID
	qc.ServiceAgentID = tDBAgentID
	qc.GroupcallID = tDBGroupcall
	qc.TimeoutService = 30000
	return qc
}

func Test_Create_errors(t *testing.T) {
	errDB := errors.New("db failure")
	q := &queue.Queue{
		Identity:       commonidentity.Identity{ID: tDBQueueID, CustomerID: tDBCustomerID},
		WaitTimeout:    10000,
		ServiceTimeout: 20000,
	}
	src := commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550100001"}

	t.Run("create failure stops before the re-read", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newQCTestHandler(mc)
		ctx := context.Background()

		m.db.EXPECT().QueuecallCreate(ctx, gomock.Any()).Return(errDB)

		res, err := h.Create(ctx, q, tQueuecallID, queuecall.ReferenceTypeCall, tReferenceID, tActiveflow, uuid.Nil, tConfbridge, src)
		if res != nil || !errors.Is(err, errDB) {
			t.Errorf("Wrong match. expect: (nil, %v), got: (%v, %v)", errDB, res, err)
		}
	})

	t.Run("re-read failure means no webhook and no timers", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newQCTestHandler(mc)
		ctx := context.Background()

		m.db.EXPECT().QueuecallCreate(ctx, gomock.Any()).Return(nil)
		m.db.EXPECT().QueuecallGet(ctx, tQueuecallID).Return(nil, errDB)

		res, err := h.Create(ctx, q, tQueuecallID, queuecall.ReferenceTypeCall, tReferenceID, tActiveflow, uuid.Nil, tConfbridge, src)
		if res != nil || !errors.Is(err, errDB) {
			t.Errorf("Wrong match. expect: (nil, %v), got: (%v, %v)", errDB, res, err)
		}
	})

	t.Run("a missing id is generated and follow-up failures do not fail the create", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newQCTestHandler(mc)
		ctx := context.Background()

		created := newQC(queuecall.StatusInitiating, queuecall.ReferenceTypeCall)
		created.CustomerID = tDBCustomerID
		created.TimeoutWait = q.WaitTimeout

		m.util.EXPECT().UUIDCreate().Return(tQueuecallID)
		m.db.EXPECT().QueuecallCreate(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, qc *queuecall.Queuecall) error {
			if qc.ID != tQueuecallID || qc.Status != queuecall.StatusInitiating || qc.TimeoutWait != q.WaitTimeout || qc.TimeoutService != q.ServiceTimeout {
				t.Errorf("Wrong queuecall passed to the db. got: %+v", qc)
			}
			return nil
		})
		m.db.EXPECT().QueuecallGet(ctx, tQueuecallID).Return(created, nil)
		m.notify.EXPECT().PublishWebhookEvent(ctx, tDBCustomerID, queuecall.EventTypeQueuecallCreated, created)
		m.req.EXPECT().FlowV1VariableSetVariable(ctx, tActiveflow, gomock.Any()).Return(errDB)
		m.req.EXPECT().QueueV1QueuecallTimeoutWait(ctx, tQueuecallID, q.WaitTimeout).Return(errDB)
		m.req.EXPECT().QueueV1QueuecallHealthCheck(ctx, tQueuecallID, defaultHealthCheckDelay, 0).Return(errDB)

		res, err := h.Create(ctx, q, uuid.Nil, queuecall.ReferenceTypeCall, tReferenceID, tActiveflow, uuid.Nil, tConfbridge, src)
		if err != nil || res != created {
			t.Errorf("Wrong match. expect: (%v, nil), got: (%v, %v)", created, res, err)
		}
	})

	t.Run("zero wait timeout schedules no timeout-wait", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newQCTestHandler(mc)
		ctx := context.Background()

		noWait := &queue.Queue{Identity: commonidentity.Identity{ID: tDBQueueID, CustomerID: tDBCustomerID}}
		created := newQC(queuecall.StatusInitiating, queuecall.ReferenceTypeCall)
		created.CustomerID = tDBCustomerID

		m.db.EXPECT().QueuecallCreate(ctx, gomock.Any()).Return(nil)
		m.db.EXPECT().QueuecallGet(ctx, tQueuecallID).Return(created, nil)
		m.notify.EXPECT().PublishWebhookEvent(ctx, tDBCustomerID, queuecall.EventTypeQueuecallCreated, created)
		m.req.EXPECT().FlowV1VariableSetVariable(ctx, tActiveflow, gomock.Any()).Return(nil)
		m.req.EXPECT().QueueV1QueuecallHealthCheck(ctx, tQueuecallID, defaultHealthCheckDelay, 0).Return(nil)

		if _, err := h.Create(ctx, noWait, tQueuecallID, queuecall.ReferenceTypeCall, tReferenceID, tActiveflow, uuid.Nil, tConfbridge, src); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	})
}

func Test_Delete_errors(t *testing.T) {
	errDB := errors.New("db failure")

	t.Run("delete failure is wrapped and nothing is published", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newQCTestHandler(mc)
		ctx := context.Background()

		m.db.EXPECT().QueuecallDelete(ctx, tQueuecallID).Return(errDB)

		res, err := h.Delete(ctx, tQueuecallID)
		if res != nil || !errors.Is(err, errDB) {
			t.Errorf("Wrong match. expect: (nil, %v), got: (%v, %v)", errDB, res, err)
		}
	})

	t.Run("re-read failure after delete is wrapped and nothing is published", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newQCTestHandler(mc)
		ctx := context.Background()

		m.db.EXPECT().QueuecallDelete(ctx, tQueuecallID).Return(nil)
		m.db.EXPECT().QueuecallGet(ctx, tQueuecallID).Return(nil, errDB)

		res, err := h.Delete(ctx, tQueuecallID)
		if res != nil || !errors.Is(err, errDB) {
			t.Errorf("Wrong match. expect: (nil, %v), got: (%v, %v)", errDB, res, err)
		}
	})

	t.Run("success publishes the deleted event", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newQCTestHandler(mc)
		ctx := context.Background()

		deleted := newDBQC()
		m.db.EXPECT().QueuecallDelete(ctx, tQueuecallID).Return(nil)
		m.db.EXPECT().QueuecallGet(ctx, tQueuecallID).Return(deleted, nil)
		m.notify.EXPECT().PublishWebhookEvent(ctx, tDBCustomerID, queuecall.EventTypeQueuecallDeleted, deleted)

		res, err := h.Delete(ctx, tQueuecallID)
		if err != nil || res != deleted {
			t.Errorf("Wrong match. expect: (%v, nil), got: (%v, %v)", deleted, res, err)
		}
	})
}

func Test_UpdateStatusConnecting_errors(t *testing.T) {
	errDB := errors.New("db failure")

	t.Run("setter failure", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newQCTestHandler(mc)
		ctx := context.Background()

		m.db.EXPECT().QueuecallSetStatusConnecting(ctx, tQueuecallID, tDBAgentID, tDBGroupcall).Return(int64(0), errDB)

		res, won, err := h.UpdateStatusConnecting(ctx, tQueuecallID, tDBAgentID, tDBGroupcall)
		if res != nil || won || !errors.Is(err, errDB) {
			t.Errorf("Wrong match. expect: (nil, false, %v), got: (%v, %v, %v)", errDB, res, won, err)
		}
	})

	t.Run("lost CAS returns the current row without publishing", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newQCTestHandler(mc)
		ctx := context.Background()

		cur := newDBQC()
		m.db.EXPECT().QueuecallSetStatusConnecting(ctx, tQueuecallID, tDBAgentID, tDBGroupcall).Return(int64(0), nil)
		m.db.EXPECT().QueuecallGet(ctx, tQueuecallID).Return(cur, nil)

		res, won, err := h.UpdateStatusConnecting(ctx, tQueuecallID, tDBAgentID, tDBGroupcall)
		if res != cur || won || err != nil {
			t.Errorf("Wrong match. expect: (%v, false, nil), got: (%v, %v, %v)", cur, res, won, err)
		}
	})

	t.Run("lost CAS and re-read failure", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newQCTestHandler(mc)
		ctx := context.Background()

		m.db.EXPECT().QueuecallSetStatusConnecting(ctx, tQueuecallID, tDBAgentID, tDBGroupcall).Return(int64(0), nil)
		m.db.EXPECT().QueuecallGet(ctx, tQueuecallID).Return(nil, errDB)

		res, won, err := h.UpdateStatusConnecting(ctx, tQueuecallID, tDBAgentID, tDBGroupcall)
		if res != nil || won || !errors.Is(err, errDB) {
			t.Errorf("Wrong match. expect: (nil, false, %v), got: (%v, %v, %v)", errDB, res, won, err)
		}
	})

	t.Run("won CAS and re-read failure publishes nothing", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newQCTestHandler(mc)
		ctx := context.Background()

		m.db.EXPECT().QueuecallSetStatusConnecting(ctx, tQueuecallID, tDBAgentID, tDBGroupcall).Return(int64(1), nil)
		m.db.EXPECT().QueuecallGet(ctx, tQueuecallID).Return(nil, errDB)

		res, won, err := h.UpdateStatusConnecting(ctx, tQueuecallID, tDBAgentID, tDBGroupcall)
		if res != nil || won || !errors.Is(err, errDB) {
			t.Errorf("Wrong match. expect: (nil, false, %v), got: (%v, %v, %v)", errDB, res, won, err)
		}
	})
}

// updateStatusCase drives the common CAS shape of the service, abandoned and
// done transitions: setter error, lost CAS, and re-read failures.
type updateStatusCase struct {
	name   string
	setter func(m *qcMocks, ctx context.Context, id uuid.UUID, dur int, at *time.Time) *gomock.Call
	call   func(h *queuecallHandler, ctx context.Context, qc *queuecall.Queuecall) (*queuecall.Queuecall, error)
}

func Test_UpdateStatus_CASAndReadErrors(t *testing.T) {
	errDB := errors.New("db failure")
	createdAt := time.Date(2023, time.February, 16, 3, 21, 17, 0, time.UTC)
	now := time.Date(2023, time.February, 16, 3, 22, 17, 0, time.UTC)
	const wantDuration = 60000

	cases := []updateStatusCase{
		{
			name: "service",
			setter: func(m *qcMocks, ctx context.Context, id uuid.UUID, dur int, at *time.Time) *gomock.Call {
				return m.db.EXPECT().QueuecallSetStatusService(ctx, id, dur, at)
			},
			call: func(h *queuecallHandler, ctx context.Context, qc *queuecall.Queuecall) (*queuecall.Queuecall, error) {
				return h.UpdateStatusService(ctx, qc)
			},
		},
		{
			name: "abandoned",
			setter: func(m *qcMocks, ctx context.Context, id uuid.UUID, dur int, at *time.Time) *gomock.Call {
				return m.db.EXPECT().QueuecallSetStatusAbandoned(ctx, id, dur, at)
			},
			call: func(h *queuecallHandler, ctx context.Context, qc *queuecall.Queuecall) (*queuecall.Queuecall, error) {
				return h.UpdateStatusAbandoned(ctx, qc)
			},
		},
		{
			name: "done",
			setter: func(m *qcMocks, ctx context.Context, id uuid.UUID, dur int, at *time.Time) *gomock.Call {
				return m.db.EXPECT().QueuecallSetStatusDone(ctx, id, dur, at)
			},
			call: func(h *queuecallHandler, ctx context.Context, qc *queuecall.Queuecall) (*queuecall.Queuecall, error) {
				return h.UpdateStatusDone(ctx, qc)
			},
		},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name+" setter failure", func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			h, m := newQCTestHandler(mc)
			ctx := context.Background()
			qc := newDBQC()
			qc.TMCreate = &createdAt

			m.util.EXPECT().TimeNow().Return(&now)
			c.setter(m, ctx, qc.ID, wantDuration, &now).Return(int64(0), errDB)

			res, err := c.call(h, ctx, qc)
			if res != nil || !errors.Is(err, errDB) {
				t.Errorf("Wrong match. expect: (nil, %v), got: (%v, %v)", errDB, res, err)
			}
		})

		t.Run(c.name+" lost CAS returns the current row with no side effects", func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			h, m := newQCTestHandler(mc)
			ctx := context.Background()
			qc := newDBQC()
			qc.TMCreate = &createdAt
			cur := newDBQC()

			m.util.EXPECT().TimeNow().Return(&now)
			c.setter(m, ctx, qc.ID, wantDuration, &now).Return(int64(0), nil)
			m.db.EXPECT().QueuecallGet(ctx, qc.ID).Return(cur, nil)

			res, err := c.call(h, ctx, qc)
			if res != cur || err != nil {
				t.Errorf("Wrong match. expect: (%v, nil), got: (%v, %v)", cur, res, err)
			}
		})

		t.Run(c.name+" lost CAS and re-read failure", func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			h, m := newQCTestHandler(mc)
			ctx := context.Background()
			qc := newDBQC()
			qc.TMCreate = &createdAt

			m.util.EXPECT().TimeNow().Return(&now)
			c.setter(m, ctx, qc.ID, wantDuration, &now).Return(int64(0), nil)
			m.db.EXPECT().QueuecallGet(ctx, qc.ID).Return(nil, errDB)

			res, err := c.call(h, ctx, qc)
			if res != nil || !errors.Is(err, errDB) {
				t.Errorf("Wrong match. expect: (nil, %v), got: (%v, %v)", errDB, res, err)
			}
		})

		t.Run(c.name+" won CAS and re-read failure publishes nothing", func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			h, m := newQCTestHandler(mc)
			ctx := context.Background()
			qc := newDBQC()
			qc.TMCreate = &createdAt

			m.util.EXPECT().TimeNow().Return(&now)
			c.setter(m, ctx, qc.ID, wantDuration, &now).Return(int64(1), nil)
			m.db.EXPECT().QueuecallGet(ctx, qc.ID).Return(nil, errDB)

			res, err := c.call(h, ctx, qc)
			if res != nil || !errors.Is(err, errDB) {
				t.Errorf("Wrong match. expect: (nil, %v), got: (%v, %v)", errDB, res, err)
			}
		})
	}
}

func Test_UpdateStatusService_sideEffects_errors(t *testing.T) {
	errRPC := errors.New("rpc failure")
	createdAt := time.Date(2023, time.February, 16, 3, 21, 17, 0, time.UTC)
	now := time.Date(2023, time.February, 16, 3, 22, 17, 0, time.UTC)

	mc := gomock.NewController(t)
	defer mc.Finish()
	h, m := newQCTestHandler(mc)
	ctx := context.Background()

	qc := newDBQC()
	qc.TMCreate = &createdAt
	res0 := newDBQC()

	m.util.EXPECT().TimeNow().Return(&now)
	m.db.EXPECT().QueuecallSetStatusService(ctx, qc.ID, 60000, &now).Return(int64(1), nil)
	m.db.EXPECT().QueuecallGet(ctx, qc.ID).Return(res0, nil)
	m.notify.EXPECT().PublishWebhookEvent(ctx, tDBCustomerID, queuecall.EventTypeQueuecallServiced, res0)
	m.queue.EXPECT().AddServiceQueuecallID(ctx, tDBQueueID, qc.ID).Return(nil, errRPC)
	m.req.EXPECT().AgentV1AgentReserveRelease(ctx, tDBAgentID, qc.ID).Return(errRPC)
	m.req.EXPECT().QueueV1QueuecallTimeoutService(ctx, qc.ID, res0.TimeoutService).Return(errRPC)

	res, err := h.UpdateStatusService(ctx, qc)
	if err != nil || res != res0 {
		t.Errorf("Wrong match. expect: (%v, nil), got: (%v, %v)", res0, res, err)
	}
}

func Test_UpdateStatusService_noAgentNoTimeout_errors(t *testing.T) {
	createdAt := time.Date(2023, time.February, 16, 3, 21, 17, 0, time.UTC)
	now := time.Date(2023, time.February, 16, 3, 22, 17, 0, time.UTC)

	mc := gomock.NewController(t)
	defer mc.Finish()
	h, m := newQCTestHandler(mc)
	ctx := context.Background()

	qc := newDBQC()
	qc.TMCreate = &createdAt
	qc.TimeoutService = 0
	res0 := newDBQC()
	res0.ServiceAgentID = uuid.Nil

	// no AgentV1AgentReserveRelease and no QueueV1QueuecallTimeoutService expectation:
	// both must be skipped.
	m.util.EXPECT().TimeNow().Return(&now)
	m.db.EXPECT().QueuecallSetStatusService(ctx, qc.ID, 60000, &now).Return(int64(1), nil)
	m.db.EXPECT().QueuecallGet(ctx, qc.ID).Return(res0, nil)
	m.notify.EXPECT().PublishWebhookEvent(ctx, tDBCustomerID, queuecall.EventTypeQueuecallServiced, res0)
	m.queue.EXPECT().AddServiceQueuecallID(ctx, tDBQueueID, qc.ID).Return(&queue.Queue{}, nil)

	res, err := h.UpdateStatusService(ctx, qc)
	if err != nil || res != res0 {
		t.Errorf("Wrong match. expect: (%v, nil), got: (%v, %v)", res0, res, err)
	}
}

func Test_UpdateStatusAbandonedDone_removeFailure_errors(t *testing.T) {
	errDB := errors.New("db failure")
	createdAt := time.Date(2023, time.February, 16, 3, 21, 17, 0, time.UTC)
	now := time.Date(2023, time.February, 16, 3, 22, 17, 0, time.UTC)

	t.Run("abandoned: queue removal failure stops the teardown", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newQCTestHandler(mc)
		ctx := context.Background()
		qc := newDBQC()
		qc.TMCreate = &createdAt

		m.util.EXPECT().TimeNow().Return(&now)
		m.db.EXPECT().QueuecallSetStatusAbandoned(ctx, qc.ID, 60000, &now).Return(int64(1), nil)
		m.db.EXPECT().QueuecallGet(ctx, qc.ID).Return(qc, nil)
		m.notify.EXPECT().PublishWebhookEvent(ctx, tDBCustomerID, queuecall.EventTypeQueuecallAbandoned, qc)
		m.queue.EXPECT().RemoveQueuecallID(ctx, tDBQueueID, qc.ID).Return(nil, errDB)

		res, err := h.UpdateStatusAbandoned(ctx, qc)
		if res != nil || !errors.Is(err, errDB) {
			t.Errorf("Wrong match. expect: (nil, %v), got: (%v, %v)", errDB, res, err)
		}
	})

	t.Run("done: queue removal failure stops the teardown", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newQCTestHandler(mc)
		ctx := context.Background()
		qc := newDBQC()
		qc.TMCreate = &createdAt

		m.util.EXPECT().TimeNow().Return(&now)
		m.db.EXPECT().QueuecallSetStatusDone(ctx, qc.ID, 60000, &now).Return(int64(1), nil)
		m.db.EXPECT().QueuecallGet(ctx, qc.ID).Return(qc, nil)
		m.notify.EXPECT().PublishWebhookEvent(ctx, tDBCustomerID, queuecall.EventTypeQueuecallDone, qc)
		m.queue.EXPECT().RemoveQueuecallID(ctx, tDBQueueID, qc.ID).Return(nil, errDB)

		res, err := h.UpdateStatusDone(ctx, qc)
		if res != nil || !errors.Is(err, errDB) {
			t.Errorf("Wrong match. expect: (nil, %v), got: (%v, %v)", errDB, res, err)
		}
	})
}

func Test_UpdateStatusAbandonedDone_cleanup_errors(t *testing.T) {
	errRPC := errors.New("rpc failure")
	createdAt := time.Date(2023, time.February, 16, 3, 21, 17, 0, time.UTC)
	now := time.Date(2023, time.February, 16, 3, 22, 17, 0, time.UTC)

	t.Run("abandoned", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newQCTestHandler(mc)
		ctx := context.Background()
		qc := newDBQC()
		qc.TMCreate = &createdAt

		m.util.EXPECT().TimeNow().Return(&now)
		m.db.EXPECT().QueuecallSetStatusAbandoned(ctx, qc.ID, 60000, &now).Return(int64(1), nil)
		m.db.EXPECT().QueuecallGet(ctx, qc.ID).Return(qc, nil)
		m.notify.EXPECT().PublishWebhookEvent(ctx, tDBCustomerID, queuecall.EventTypeQueuecallAbandoned, qc)
		m.queue.EXPECT().RemoveQueuecallID(ctx, tDBQueueID, qc.ID).Return(&queue.Queue{}, nil)
		m.req.EXPECT().CallV1ConfbridgeDelete(ctx, tConfbridge).Return((*cmconfbridge.Confbridge)(nil), errRPC)
		m.req.EXPECT().FlowV1VariableDeleteVariable(ctx, tActiveflow, gomock.Any()).Return(errRPC)
		m.req.EXPECT().CallV1GroupcallHangup(ctx, tDBGroupcall).Return((*cmgroupcall.Groupcall)(nil), errRPC)
		m.req.EXPECT().AgentV1AgentReserveRelease(ctx, tDBAgentID, qc.ID).Return(errRPC)

		res, err := h.UpdateStatusAbandoned(ctx, qc)
		if err != nil || res != qc {
			t.Errorf("Wrong match. expect: (%v, nil), got: (%v, %v)", qc, res, err)
		}
	})

	t.Run("abandoned without groupcall or agent skips those RPCs", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newQCTestHandler(mc)
		ctx := context.Background()
		qc := newDBQC()
		qc.TMCreate = &createdAt
		qc.GroupcallID = uuid.Nil
		qc.ServiceAgentID = uuid.Nil

		m.util.EXPECT().TimeNow().Return(&now)
		m.db.EXPECT().QueuecallSetStatusAbandoned(ctx, qc.ID, 60000, &now).Return(int64(1), nil)
		m.db.EXPECT().QueuecallGet(ctx, qc.ID).Return(qc, nil)
		m.notify.EXPECT().PublishWebhookEvent(ctx, tDBCustomerID, queuecall.EventTypeQueuecallAbandoned, qc)
		m.queue.EXPECT().RemoveQueuecallID(ctx, tDBQueueID, qc.ID).Return(&queue.Queue{}, nil)
		m.req.EXPECT().CallV1ConfbridgeDelete(ctx, tConfbridge).Return(&cmconfbridge.Confbridge{}, nil)
		m.req.EXPECT().FlowV1VariableDeleteVariable(ctx, tActiveflow, gomock.Any()).Return(nil).Times(6)

		if _, err := h.UpdateStatusAbandoned(ctx, qc); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	})

	t.Run("done", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newQCTestHandler(mc)
		ctx := context.Background()
		qc := newDBQC()
		qc.TMCreate = &createdAt

		m.util.EXPECT().TimeNow().Return(&now)
		m.db.EXPECT().QueuecallSetStatusDone(ctx, qc.ID, 60000, &now).Return(int64(1), nil)
		m.db.EXPECT().QueuecallGet(ctx, qc.ID).Return(qc, nil)
		m.notify.EXPECT().PublishWebhookEvent(ctx, tDBCustomerID, queuecall.EventTypeQueuecallDone, qc)
		m.queue.EXPECT().RemoveQueuecallID(ctx, tDBQueueID, qc.ID).Return(&queue.Queue{}, nil)
		m.req.EXPECT().CallV1ConfbridgeDelete(ctx, tConfbridge).Return((*cmconfbridge.Confbridge)(nil), errRPC)
		m.req.EXPECT().FlowV1VariableDeleteVariable(ctx, tActiveflow, gomock.Any()).Return(errRPC)

		res, err := h.UpdateStatusDone(ctx, qc)
		if err != nil || res != qc {
			t.Errorf("Wrong match. expect: (%v, nil), got: (%v, %v)", qc, res, err)
		}
	})
}

func Test_UpdateStatusWaiting_errors(t *testing.T) {
	errDB := errors.New("db failure")

	t.Run("setter failure", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newQCTestHandler(mc)
		ctx := context.Background()

		m.db.EXPECT().QueuecallSetStatusWaitingIfInitiating(ctx, tQueuecallID).Return(int64(0), errDB)

		res, err := h.UpdateStatusWaiting(ctx, tQueuecallID)
		if res != nil || !errors.Is(err, errDB) {
			t.Errorf("Wrong match. expect: (nil, %v), got: (%v, %v)", errDB, res, err)
		}
	})

	t.Run("lost CAS returns the current row and starts no matching", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newQCTestHandler(mc)
		ctx := context.Background()

		cur := newDBQC()
		m.db.EXPECT().QueuecallSetStatusWaitingIfInitiating(ctx, tQueuecallID).Return(int64(0), nil)
		m.db.EXPECT().QueuecallGet(ctx, tQueuecallID).Return(cur, nil)

		res, err := h.UpdateStatusWaiting(ctx, tQueuecallID)
		if res != cur || err != nil {
			t.Errorf("Wrong match. expect: (%v, nil), got: (%v, %v)", cur, res, err)
		}
	})

	t.Run("lost CAS and re-read failure", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newQCTestHandler(mc)
		ctx := context.Background()

		m.db.EXPECT().QueuecallSetStatusWaitingIfInitiating(ctx, tQueuecallID).Return(int64(0), nil)
		m.db.EXPECT().QueuecallGet(ctx, tQueuecallID).Return(nil, errDB)

		res, err := h.UpdateStatusWaiting(ctx, tQueuecallID)
		if res != nil || !errors.Is(err, errDB) {
			t.Errorf("Wrong match. expect: (nil, %v), got: (%v, %v)", errDB, res, err)
		}
	})

	t.Run("won CAS and re-read failure publishes nothing", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newQCTestHandler(mc)
		ctx := context.Background()

		m.db.EXPECT().QueuecallSetStatusWaitingIfInitiating(ctx, tQueuecallID).Return(int64(1), nil)
		m.db.EXPECT().QueuecallGet(ctx, tQueuecallID).Return(nil, errDB)

		res, err := h.UpdateStatusWaiting(ctx, tQueuecallID)
		if res != nil || !errors.Is(err, errDB) {
			t.Errorf("Wrong match. expect: (nil, %v), got: (%v, %v)", errDB, res, err)
		}
	})

	t.Run("won CAS but queue registration failure ends the async matching", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newQCTestHandler(mc)
		ctx := context.Background()

		cur := newDBQC()
		done := make(chan struct{})
		m.db.EXPECT().QueuecallSetStatusWaitingIfInitiating(ctx, tQueuecallID).Return(int64(1), nil)
		m.db.EXPECT().QueuecallGet(ctx, tQueuecallID).Return(cur, nil)
		m.notify.EXPECT().PublishWebhookEvent(ctx, tDBCustomerID, queuecall.EventTypeQueuecallWaiting, cur)
		m.queue.EXPECT().AddWaitQueueCallID(ctx, tDBQueueID, tQueuecallID).DoAndReturn(
			func(_ context.Context, _, _ uuid.UUID) (*queue.Queue, error) {
				defer close(done)
				return nil, errDB
			})

		res, err := h.UpdateStatusWaiting(ctx, tQueuecallID)
		if res != cur || err != nil {
			t.Errorf("Wrong match. expect: (%v, nil), got: (%v, %v)", cur, res, err)
		}
		waitDone(t, done)
	})
}

func Test_UpdateStatusWaitingRollback_errors(t *testing.T) {
	errDB := errors.New("db failure")

	t.Run("setter failure", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newQCTestHandler(mc)
		ctx := context.Background()
		qc := newDBQC()

		m.db.EXPECT().QueuecallSetStatusWaitingIfConnecting(ctx, qc.ID).Return(int64(0), errDB)

		res, err := h.UpdateStatusWaitingRollback(ctx, qc)
		if res != nil || !errors.Is(err, errDB) {
			t.Errorf("Wrong match. expect: (nil, %v), got: (%v, %v)", errDB, res, err)
		}
	})

	t.Run("lost CAS releases and hangs up nothing", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newQCTestHandler(mc)
		ctx := context.Background()
		qc := newDBQC()
		cur := newDBQC()

		m.db.EXPECT().QueuecallSetStatusWaitingIfConnecting(ctx, qc.ID).Return(int64(0), nil)
		m.db.EXPECT().QueuecallGet(ctx, qc.ID).Return(cur, nil)

		res, err := h.UpdateStatusWaitingRollback(ctx, qc)
		if res != cur || err != nil {
			t.Errorf("Wrong match. expect: (%v, nil), got: (%v, %v)", cur, res, err)
		}
	})

	t.Run("lost CAS and re-read failure", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newQCTestHandler(mc)
		ctx := context.Background()
		qc := newDBQC()

		m.db.EXPECT().QueuecallSetStatusWaitingIfConnecting(ctx, qc.ID).Return(int64(0), nil)
		m.db.EXPECT().QueuecallGet(ctx, qc.ID).Return(nil, errDB)

		res, err := h.UpdateStatusWaitingRollback(ctx, qc)
		if res != nil || !errors.Is(err, errDB) {
			t.Errorf("Wrong match. expect: (nil, %v), got: (%v, %v)", errDB, res, err)
		}
	})

	t.Run("won CAS tolerates release and hangup failures and re-read failure is fatal", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newQCTestHandler(mc)
		ctx := context.Background()
		qc := newDBQC()

		m.db.EXPECT().QueuecallSetStatusWaitingIfConnecting(ctx, qc.ID).Return(int64(1), nil)
		m.req.EXPECT().AgentV1AgentReserveRelease(ctx, tDBAgentID, qc.ID).Return(errDB)
		m.req.EXPECT().CallV1GroupcallHangup(ctx, tDBGroupcall).Return((*cmgroupcall.Groupcall)(nil), errDB)
		m.db.EXPECT().QueuecallGet(ctx, qc.ID).Return(nil, errDB)

		res, err := h.UpdateStatusWaitingRollback(ctx, qc)
		if res != nil || !errors.Is(err, errDB) {
			t.Errorf("Wrong match. expect: (nil, %v), got: (%v, %v)", errDB, res, err)
		}
	})

	t.Run("won CAS without agent or groupcall re-queues the queuecall", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newQCTestHandler(mc)
		ctx := context.Background()
		qc := newDBQC()
		qc.ServiceAgentID = uuid.Nil
		qc.GroupcallID = uuid.Nil
		cur := newDBQC()
		done := make(chan struct{})

		m.db.EXPECT().QueuecallSetStatusWaitingIfConnecting(ctx, qc.ID).Return(int64(1), nil)
		m.db.EXPECT().QueuecallGet(ctx, qc.ID).Return(cur, nil)
		m.notify.EXPECT().PublishWebhookEvent(ctx, tDBCustomerID, queuecall.EventTypeQueuecallWaiting, cur)
		m.queue.EXPECT().AddWaitQueueCallID(ctx, tDBQueueID, qc.ID).DoAndReturn(
			func(_ context.Context, _, _ uuid.UUID) (*queue.Queue, error) {
				defer close(done)
				return nil, errDB
			})

		res, err := h.UpdateStatusWaitingRollback(ctx, qc)
		if res != cur || err != nil {
			t.Errorf("Wrong match. expect: (%v, nil), got: (%v, %v)", cur, res, err)
		}
		waitDone(t, done)
	})
}

func waitDone(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for the asynchronous call")
	}
}
