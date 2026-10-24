package queuehandler

import (
	"context"
	"errors"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	bmaccount "monorepo/bin-billing-manager/models/account"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	cucustomer "monorepo/bin-customer-manager/models/customer"
	dmdirect "monorepo/bin-direct-manager/models/direct"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-queue-manager/models/queue"
	"monorepo/bin-queue-manager/pkg/dbhandler"
)

var (
	testQueueID     = uuid.FromStringOrNil("a1b2c3d4-0001-11ee-8a01-000000000001")
	testQueuecallID = uuid.FromStringOrNil("a1b2c3d4-0002-11ee-8a01-000000000002")
	testCustomerID  = uuid.FromStringOrNil("a1b2c3d4-0003-11ee-8a01-000000000003")
	testDirectID    = uuid.FromStringOrNil("a1b2c3d4-0004-11ee-8a01-000000000004")
)

type testMocks struct {
	db     *dbhandler.MockDBHandler
	req    *requesthandler.MockRequestHandler
	notify *notifyhandler.MockNotifyHandler
	util   *utilhandler.MockUtilHandler
}

func newTestHandler(mc *gomock.Controller) (*queueHandler, *testMocks) {
	m := &testMocks{
		db:     dbhandler.NewMockDBHandler(mc),
		req:    requesthandler.NewMockRequestHandler(mc),
		notify: notifyhandler.NewMockNotifyHandler(mc),
		util:   utilhandler.NewMockUtilHandler(mc),
	}
	h := &queueHandler{
		db:            m.db,
		reqHandler:    m.req,
		notifyhandler: m.notify,
		utilHandler:   m.util,
	}
	return h, m
}

func Test_queuecallIDMutations(t *testing.T) {
	errWrite := errors.New("write failed")
	errGet := errors.New("get failed")

	// each entry wires the single db write that the method performs first.
	methods := []struct {
		name       string
		call       func(h *queueHandler, ctx context.Context) (*queue.Queue, error)
		expectDBOp func(m *testMocks, ctx context.Context, err error)
	}{
		{
			name: "AddWaitQueueCallID",
			call: func(h *queueHandler, ctx context.Context) (*queue.Queue, error) {
				return h.AddWaitQueueCallID(ctx, testQueueID, testQueuecallID)
			},
			expectDBOp: func(m *testMocks, ctx context.Context, err error) {
				m.db.EXPECT().QueueAddWaitQueueCallID(ctx, testQueueID, testQueuecallID).Return(err)
			},
		},
		{
			name: "RemoveServiceQueuecallID",
			call: func(h *queueHandler, ctx context.Context) (*queue.Queue, error) {
				return h.RemoveServiceQueuecallID(ctx, testQueueID, testQueuecallID)
			},
			expectDBOp: func(m *testMocks, ctx context.Context, err error) {
				m.db.EXPECT().QueueRemoveServiceQueueCall(ctx, testQueueID, testQueuecallID).Return(err)
			},
		},
		{
			name: "RemoveWaitQueuecallID",
			call: func(h *queueHandler, ctx context.Context) (*queue.Queue, error) {
				return h.RemoveWaitQueuecallID(ctx, testQueueID, testQueuecallID)
			},
			expectDBOp: func(m *testMocks, ctx context.Context, err error) {
				m.db.EXPECT().QueueRemoveWaitQueueCall(ctx, testQueueID, testQueuecallID).Return(err)
			},
		},
		{
			name: "AddServiceQueuecallID",
			call: func(h *queueHandler, ctx context.Context) (*queue.Queue, error) {
				return h.AddServiceQueuecallID(ctx, testQueueID, testQueuecallID)
			},
			expectDBOp: func(m *testMocks, ctx context.Context, err error) {
				m.db.EXPECT().QueueIncreaseTotalServicedCount(ctx, testQueueID, testQueuecallID).Return(err)
			},
		},
	}

	for _, mt := range methods {
		t.Run(mt.name+" write error is propagated without reading or notifying", func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newTestHandler(mc)
			ctx := context.Background()

			mt.expectDBOp(m, ctx, errWrite)

			res, err := mt.call(h, ctx)
			if res != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", res)
			}
			if !errors.Is(err, errWrite) {
				t.Errorf("Wrong match. expect: %v, got: %v", errWrite, err)
			}
		})

		t.Run(mt.name+" read-back error is propagated without notifying", func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newTestHandler(mc)
			ctx := context.Background()

			mt.expectDBOp(m, ctx, nil)
			m.db.EXPECT().QueueGet(ctx, testQueueID).Return(nil, errGet)

			res, err := mt.call(h, ctx)
			if res != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", res)
			}
			if !errors.Is(err, errGet) {
				t.Errorf("Wrong match. expect: %v, got: %v", errGet, err)
			}
		})

		t.Run(mt.name+" success publishes the updated queue", func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newTestHandler(mc)
			ctx := context.Background()

			updated := &queue.Queue{
				Identity:         commonidentity.Identity{ID: testQueueID},
				WaitQueuecallIDs: []uuid.UUID{testQueuecallID},
			}

			mt.expectDBOp(m, ctx, nil)
			m.db.EXPECT().QueueGet(ctx, testQueueID).Return(updated, nil)
			m.notify.EXPECT().PublishEvent(ctx, queue.EventTypeQueueUpdated, updated)

			res, err := mt.call(h, ctx)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res != updated {
				t.Errorf("Wrong match. expect: %p, got: %p", updated, res)
			}
		})
	}
}

func Test_RemoveQueuecallID_errors(t *testing.T) {
	errWait := errors.New("wait removal failed")
	errService := errors.New("service removal failed")
	errGet := errors.New("get failed")

	tests := []struct {
		name string

		setup     func(m *testMocks, ctx context.Context)
		expectErr error
	}{
		{
			name: "wait removal fails, service removal is not attempted",
			setup: func(m *testMocks, ctx context.Context) {
				m.db.EXPECT().QueueRemoveWaitQueueCall(ctx, testQueueID, testQueuecallID).Return(errWait)
			},
			expectErr: errWait,
		},
		{
			name: "service removal fails, no read-back",
			setup: func(m *testMocks, ctx context.Context) {
				m.db.EXPECT().QueueRemoveWaitQueueCall(ctx, testQueueID, testQueuecallID).Return(nil)
				m.db.EXPECT().QueueRemoveServiceQueueCall(ctx, testQueueID, testQueuecallID).Return(errService)
			},
			expectErr: errService,
		},
		{
			name: "read-back fails, no event",
			setup: func(m *testMocks, ctx context.Context) {
				m.db.EXPECT().QueueRemoveWaitQueueCall(ctx, testQueueID, testQueuecallID).Return(nil)
				m.db.EXPECT().QueueRemoveServiceQueueCall(ctx, testQueueID, testQueuecallID).Return(nil)
				m.db.EXPECT().QueueGet(ctx, testQueueID).Return(nil, errGet)
			},
			expectErr: errGet,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newTestHandler(mc)
			ctx := context.Background()

			tt.setup(m, ctx)

			res, err := h.RemoveQueuecallID(ctx, testQueueID, testQueuecallID)
			if res != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", res)
			}
			if !errors.Is(err, tt.expectErr) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectErr, err)
			}
		})
	}
}

func Test_ListGet_errors(t *testing.T) {
	errDB := errors.New("db failure")

	t.Run("List propagates the db error", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newTestHandler(mc)
		ctx := context.Background()

		m.db.EXPECT().QueueList(ctx, uint64(10), "token", gomock.Any()).Return(nil, errDB)

		res, err := h.List(ctx, 10, "token", map[queue.Field]any{queue.FieldDeleted: false})
		if res != nil || !errors.Is(err, errDB) {
			t.Errorf("Wrong match. expect: (nil, %v), got: (%v, %v)", errDB, res, err)
		}
	})

	t.Run("Get maps record-not-found to a typed NotFound error", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newTestHandler(mc)
		ctx := context.Background()

		m.db.EXPECT().QueueGet(ctx, testQueueID).Return(nil, dbhandler.ErrNotFound)

		res, err := h.Get(ctx, testQueueID)
		if res != nil {
			t.Errorf("Wrong match. expect: nil, got: %v", res)
		}

		var ve *cerrors.VoipbinError
		if !errors.As(err, &ve) {
			t.Fatalf("Wrong match. expect: VoipbinError, got: %v", err)
		}
		if ve.Status != cerrors.StatusNotFound || ve.Reason != "QUEUE_NOT_FOUND" {
			t.Errorf("Wrong match. expect: NOT_FOUND/QUEUE_NOT_FOUND, got: %v/%v", ve.Status, ve.Reason)
		}
		if !errors.Is(err, dbhandler.ErrNotFound) {
			t.Errorf("Wrong match. the cause must stay reachable. err: %v", err)
		}
	})

	t.Run("Get passes other db errors through untyped", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newTestHandler(mc)
		ctx := context.Background()

		m.db.EXPECT().QueueGet(ctx, testQueueID).Return(nil, errDB)

		res, err := h.Get(ctx, testQueueID)
		if res != nil || err != errDB {
			t.Errorf("Wrong match. expect: (nil, %v), got: (%v, %v)", errDB, res, err)
		}
	})
}

func Test_dbDelete_errors(t *testing.T) {
	errDel := errors.New("delete failed")
	errGet := errors.New("get failed")

	t.Run("delete error stops before read-back", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newTestHandler(mc)
		ctx := context.Background()

		m.db.EXPECT().QueueDelete(ctx, testQueueID).Return(errDel)

		res, err := h.dbDelete(ctx, testQueueID)
		if res != nil || !errors.Is(err, errDel) {
			t.Errorf("Wrong match. expect: (nil, %v), got: (%v, %v)", errDel, res, err)
		}
	})

	t.Run("read-back error does not publish the deleted event", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newTestHandler(mc)
		ctx := context.Background()

		m.db.EXPECT().QueueDelete(ctx, testQueueID).Return(nil)
		m.db.EXPECT().QueueGet(ctx, testQueueID).Return(nil, errGet)

		res, err := h.dbDelete(ctx, testQueueID)
		if res != nil || !errors.Is(err, errGet) {
			t.Errorf("Wrong match. expect: (nil, %v), got: (%v, %v)", errGet, res, err)
		}
	})
}

func Test_Update_errors(t *testing.T) {
	errUpdate := errors.New("update failed")
	errGet := errors.New("get failed")

	name := "renamed"

	tests := []struct {
		name string

		call      func(h *queueHandler, ctx context.Context) (*queue.Queue, error)
		setup     func(m *testMocks, ctx context.Context)
		expectErr error
	}{
		{
			name: "UpdateBasicInfo update error",
			call: func(h *queueHandler, ctx context.Context) (*queue.Queue, error) {
				return h.UpdateBasicInfo(ctx, testQueueID, &name, nil, nil, nil, nil, nil, nil)
			},
			setup: func(m *testMocks, ctx context.Context) {
				m.db.EXPECT().QueueUpdate(ctx, testQueueID, map[queue.Field]any{queue.FieldName: name}).Return(errUpdate)
			},
			expectErr: errUpdate,
		},
		{
			name: "UpdateBasicInfo read-back error",
			call: func(h *queueHandler, ctx context.Context) (*queue.Queue, error) {
				return h.UpdateBasicInfo(ctx, testQueueID, &name, nil, nil, nil, nil, nil, nil)
			},
			setup: func(m *testMocks, ctx context.Context) {
				m.db.EXPECT().QueueUpdate(ctx, testQueueID, map[queue.Field]any{queue.FieldName: name}).Return(nil)
				m.db.EXPECT().QueueGet(ctx, testQueueID).Return(nil, errGet)
			},
			expectErr: errGet,
		},
		{
			name: "UpdateBasicInfo with nothing to change only reads the queue",
			call: func(h *queueHandler, ctx context.Context) (*queue.Queue, error) {
				return h.UpdateBasicInfo(ctx, testQueueID, nil, nil, nil, nil, nil, nil, nil)
			},
			setup: func(m *testMocks, ctx context.Context) {
				m.db.EXPECT().QueueGet(ctx, testQueueID).Return(nil, errGet)
			},
			expectErr: errGet,
		},
		{
			name: "UpdateTagIDs update error",
			call: func(h *queueHandler, ctx context.Context) (*queue.Queue, error) {
				return h.UpdateTagIDs(ctx, testQueueID, []uuid.UUID{testCustomerID})
			},
			setup: func(m *testMocks, ctx context.Context) {
				m.db.EXPECT().QueueUpdate(ctx, testQueueID, map[queue.Field]any{queue.FieldTagIDs: []uuid.UUID{testCustomerID}}).Return(errUpdate)
			},
			expectErr: errUpdate,
		},
		{
			name: "UpdateTagIDs read-back error",
			call: func(h *queueHandler, ctx context.Context) (*queue.Queue, error) {
				return h.UpdateTagIDs(ctx, testQueueID, []uuid.UUID{testCustomerID})
			},
			setup: func(m *testMocks, ctx context.Context) {
				m.db.EXPECT().QueueUpdate(ctx, testQueueID, gomock.Any()).Return(nil)
				m.db.EXPECT().QueueGet(ctx, testQueueID).Return(nil, errGet)
			},
			expectErr: errGet,
		},
		{
			name: "UpdateRoutingMethod update error",
			call: func(h *queueHandler, ctx context.Context) (*queue.Queue, error) {
				return h.UpdateRoutingMethod(ctx, testQueueID, queue.RoutingMethodRandom)
			},
			setup: func(m *testMocks, ctx context.Context) {
				m.db.EXPECT().QueueUpdate(ctx, testQueueID, map[queue.Field]any{queue.FieldRoutingMethod: queue.RoutingMethodRandom}).Return(errUpdate)
			},
			expectErr: errUpdate,
		},
		{
			name: "UpdateRoutingMethod read-back error",
			call: func(h *queueHandler, ctx context.Context) (*queue.Queue, error) {
				return h.UpdateRoutingMethod(ctx, testQueueID, queue.RoutingMethodRandom)
			},
			setup: func(m *testMocks, ctx context.Context) {
				m.db.EXPECT().QueueUpdate(ctx, testQueueID, gomock.Any()).Return(nil)
				m.db.EXPECT().QueueGet(ctx, testQueueID).Return(nil, errGet)
			},
			expectErr: errGet,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newTestHandler(mc)
			ctx := context.Background()

			tt.setup(m, ctx)

			res, err := tt.call(h, ctx)
			if res != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", res)
			}
			if !errors.Is(err, tt.expectErr) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectErr, err)
			}
		})
	}
}

func Test_Create_errors(t *testing.T) {
	errBilling := errors.New("billing unavailable")
	errDirect := errors.New("direct create failed")
	errCleanup := errors.New("cleanup failed")
	errCreate := errors.New("queue create failed")
	errGet := errors.New("get failed")

	newID := uuid.FromStringOrNil("a1b2c3d4-0005-11ee-8a01-000000000005")
	direct := &dmdirect.Direct{
		Identity: commonidentity.Identity{ID: testDirectID},
		Hash:     "hash-value",
	}

	tests := []struct {
		name string

		routingMethod queue.RoutingMethod
		setup         func(m *testMocks, ctx context.Context)

		expectErrIs  error
		expectErrMsg string
		expectReason string
	}{
		{
			name:          "billing lookup fails, nothing is created",
			routingMethod: queue.RoutingMethodRandom,
			setup: func(m *testMocks, ctx context.Context) {
				m.req.EXPECT().BillingV1AccountIsValidResourceLimitByCustomerID(ctx, testCustomerID, bmaccount.ResourceTypeQueue).Return(false, errBilling)
			},
			expectErrIs: errBilling,
		},
		{
			name:          "resource limit exceeded, nothing is created",
			routingMethod: queue.RoutingMethodRandom,
			setup: func(m *testMocks, ctx context.Context) {
				m.req.EXPECT().BillingV1AccountIsValidResourceLimitByCustomerID(ctx, testCustomerID, bmaccount.ResourceTypeQueue).Return(false, nil)
			},
			expectErrMsg: "resource limit exceeded",
		},
		{
			name:          "direct creation fails, no queue row is written",
			routingMethod: queue.RoutingMethodRandom,
			setup: func(m *testMocks, ctx context.Context) {
				m.req.EXPECT().BillingV1AccountIsValidResourceLimitByCustomerID(ctx, testCustomerID, bmaccount.ResourceTypeQueue).Return(true, nil)
				m.util.EXPECT().UUIDCreate().Return(newID)
				m.req.EXPECT().DirectV1DirectCreate(ctx, testCustomerID, dmdirect.ResourceTypeQueue, newID).Return(nil, errDirect)
			},
			expectErrIs: errDirect,
		},
		{
			name:          "unsupported routing method deletes the orphaned direct even when cleanup fails",
			routingMethod: queue.RoutingMethod("unsupported"),
			setup: func(m *testMocks, ctx context.Context) {
				m.req.EXPECT().BillingV1AccountIsValidResourceLimitByCustomerID(ctx, testCustomerID, bmaccount.ResourceTypeQueue).Return(true, nil)
				m.util.EXPECT().UUIDCreate().Return(newID)
				m.req.EXPECT().DirectV1DirectCreate(ctx, testCustomerID, dmdirect.ResourceTypeQueue, newID).Return(direct, nil)
				m.req.EXPECT().DirectV1DirectDelete(ctx, testDirectID).Return(nil, errCleanup)
			},
			expectReason: "INVALID_ROUTING_METHOD",
		},
		{
			name:          "queue row insert fails, the orphaned direct is deleted",
			routingMethod: queue.RoutingMethodRandom,
			setup: func(m *testMocks, ctx context.Context) {
				m.req.EXPECT().BillingV1AccountIsValidResourceLimitByCustomerID(ctx, testCustomerID, bmaccount.ResourceTypeQueue).Return(true, nil)
				m.util.EXPECT().UUIDCreate().Return(newID)
				m.req.EXPECT().DirectV1DirectCreate(ctx, testCustomerID, dmdirect.ResourceTypeQueue, newID).Return(direct, nil)
				m.db.EXPECT().QueueCreate(ctx, gomock.Any()).Return(errCreate)
				m.req.EXPECT().DirectV1DirectDelete(ctx, testDirectID).Return(direct, nil)
			},
			expectErrIs: errCreate,
		},
		{
			name:          "queue row insert and direct cleanup both fail, the insert error wins",
			routingMethod: queue.RoutingMethodRandom,
			setup: func(m *testMocks, ctx context.Context) {
				m.req.EXPECT().BillingV1AccountIsValidResourceLimitByCustomerID(ctx, testCustomerID, bmaccount.ResourceTypeQueue).Return(true, nil)
				m.util.EXPECT().UUIDCreate().Return(newID)
				m.req.EXPECT().DirectV1DirectCreate(ctx, testCustomerID, dmdirect.ResourceTypeQueue, newID).Return(direct, nil)
				m.db.EXPECT().QueueCreate(ctx, gomock.Any()).Return(errCreate)
				m.req.EXPECT().DirectV1DirectDelete(ctx, testDirectID).Return(nil, errCleanup)
			},
			expectErrIs: errCreate,
		},
		{
			name:          "read-back of the created queue fails",
			routingMethod: queue.RoutingMethodRandom,
			setup: func(m *testMocks, ctx context.Context) {
				m.req.EXPECT().BillingV1AccountIsValidResourceLimitByCustomerID(ctx, testCustomerID, bmaccount.ResourceTypeQueue).Return(true, nil)
				m.util.EXPECT().UUIDCreate().Return(newID)
				m.req.EXPECT().DirectV1DirectCreate(ctx, testCustomerID, dmdirect.ResourceTypeQueue, newID).Return(direct, nil)
				m.db.EXPECT().QueueCreate(ctx, gomock.Any()).Return(nil)
				m.db.EXPECT().QueueGet(ctx, newID).Return(nil, errGet)
			},
			expectErrIs: errGet,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newTestHandler(mc)
			ctx := context.Background()

			tt.setup(m, ctx)

			res, err := h.Create(ctx, testCustomerID, "name", "detail", tt.routingMethod, nil, uuid.Nil, 1000, 2000)
			if res != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", res)
			}
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: nil")
			}
			if tt.expectErrIs != nil && !errors.Is(err, tt.expectErrIs) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectErrIs, err)
			}
			if tt.expectErrMsg != "" && err.Error() != tt.expectErrMsg {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectErrMsg, err)
			}
			if tt.expectReason != "" {
				var ve *cerrors.VoipbinError
				if !errors.As(err, &ve) || ve.Reason != tt.expectReason || ve.Status != cerrors.StatusInvalidArgument {
					t.Errorf("Wrong match. expect: INVALID_ARGUMENT/%v, got: %v", tt.expectReason, err)
				}
			}
		})
	}
}

func Test_DirectHashRegenerate_errors(t *testing.T) {
	errGet := errors.New("get failed")
	errRegen := errors.New("regenerate failed")
	errCreate := errors.New("direct create failed")
	errUpdate := errors.New("update failed")

	withDirect := &queue.Queue{
		Identity: commonidentity.Identity{ID: testQueueID, CustomerID: testCustomerID},
		DirectID: testDirectID,
	}
	withoutDirect := &queue.Queue{
		Identity: commonidentity.Identity{ID: testQueueID, CustomerID: testCustomerID},
	}
	newDirect := &dmdirect.Direct{
		Identity: commonidentity.Identity{ID: testDirectID},
		Hash:     "new-hash",
	}
	wantFields := map[queue.Field]any{
		queue.FieldDirectID:   testDirectID,
		queue.FieldDirectHash: "new-hash",
	}

	tests := []struct {
		name string

		setup     func(m *testMocks, ctx context.Context)
		expectErr error
		expectRes *queue.Queue
	}{
		{
			name: "initial read fails",
			setup: func(m *testMocks, ctx context.Context) {
				m.db.EXPECT().QueueGet(ctx, testQueueID).Return(nil, errGet)
			},
			expectErr: errGet,
		},
		{
			name: "regenerate of the existing direct fails",
			setup: func(m *testMocks, ctx context.Context) {
				m.db.EXPECT().QueueGet(ctx, testQueueID).Return(withDirect, nil)
				m.req.EXPECT().DirectV1DirectRegenerate(ctx, testDirectID).Return(nil, errRegen)
			},
			expectErr: errRegen,
		},
		{
			name: "queue without a direct creates a new one and fails",
			setup: func(m *testMocks, ctx context.Context) {
				m.db.EXPECT().QueueGet(ctx, testQueueID).Return(withoutDirect, nil)
				m.req.EXPECT().DirectV1DirectCreate(ctx, testCustomerID, dmdirect.ResourceTypeQueue, testQueueID).Return(nil, errCreate)
			},
			expectErr: errCreate,
		},
		{
			name: "queue update fails",
			setup: func(m *testMocks, ctx context.Context) {
				m.db.EXPECT().QueueGet(ctx, testQueueID).Return(withDirect, nil)
				m.req.EXPECT().DirectV1DirectRegenerate(ctx, testDirectID).Return(newDirect, nil)
				m.db.EXPECT().QueueUpdate(ctx, testQueueID, wantFields).Return(errUpdate)
			},
			expectErr: errUpdate,
		},
		{
			name: "final read fails",
			setup: func(m *testMocks, ctx context.Context) {
				m.db.EXPECT().QueueGet(ctx, testQueueID).Return(withDirect, nil)
				m.req.EXPECT().DirectV1DirectRegenerate(ctx, testDirectID).Return(newDirect, nil)
				m.db.EXPECT().QueueUpdate(ctx, testQueueID, wantFields).Return(nil)
				m.db.EXPECT().QueueGet(ctx, testQueueID).Return(nil, errGet)
			},
			expectErr: errGet,
		},
		{
			name: "queue without a direct gets a newly created one stored",
			setup: func(m *testMocks, ctx context.Context) {
				m.db.EXPECT().QueueGet(ctx, testQueueID).Return(withoutDirect, nil)
				m.req.EXPECT().DirectV1DirectCreate(ctx, testCustomerID, dmdirect.ResourceTypeQueue, testQueueID).Return(newDirect, nil)
				m.db.EXPECT().QueueUpdate(ctx, testQueueID, wantFields).Return(nil)
				m.db.EXPECT().QueueGet(ctx, testQueueID).Return(withDirect, nil)
			},
			expectRes: withDirect,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newTestHandler(mc)
			ctx := context.Background()

			tt.setup(m, ctx)

			res, err := h.DirectHashRegenerate(ctx, testQueueID)
			if !errors.Is(err, tt.expectErr) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectErr, err)
			}
			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_Delete_errors(t *testing.T) {
	errGet := errors.New("get failed")
	errDirect := errors.New("direct delete failed")
	errDel := errors.New("delete failed")

	withDirect := &queue.Queue{
		Identity: commonidentity.Identity{ID: testQueueID},
		DirectID: testDirectID,
	}

	t.Run("lookup failure stops the deletion", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newTestHandler(mc)
		ctx := context.Background()

		m.db.EXPECT().QueueGet(ctx, testQueueID).Return(nil, errGet)

		res, err := h.Delete(ctx, testQueueID)
		if res != nil || !errors.Is(err, errGet) {
			t.Errorf("Wrong match. expect: (nil, %v), got: (%v, %v)", errGet, res, err)
		}
	})

	t.Run("direct cleanup failure is best effort and the queue is still deleted", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newTestHandler(mc)
		ctx := context.Background()

		deleted := &queue.Queue{Identity: commonidentity.Identity{ID: testQueueID}}

		m.db.EXPECT().QueueGet(ctx, testQueueID).Return(withDirect, nil)
		m.req.EXPECT().DirectV1DirectDelete(ctx, testDirectID).Return(nil, errDirect)
		m.db.EXPECT().QueueDelete(ctx, testQueueID).Return(nil)
		m.db.EXPECT().QueueGet(ctx, testQueueID).Return(deleted, nil)
		m.notify.EXPECT().PublishEvent(ctx, queue.EventTypeQueueDeleted, deleted)

		res, err := h.Delete(ctx, testQueueID)
		if err != nil || res != deleted {
			t.Errorf("Wrong match. expect: (%v, nil), got: (%v, %v)", deleted, res, err)
		}
	})

	t.Run("db deletion failure is returned", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newTestHandler(mc)
		ctx := context.Background()

		m.db.EXPECT().QueueGet(ctx, testQueueID).Return(&queue.Queue{Identity: commonidentity.Identity{ID: testQueueID}}, nil)
		m.db.EXPECT().QueueDelete(ctx, testQueueID).Return(errDel)

		res, err := h.Delete(ctx, testQueueID)
		if res != nil || !errors.Is(err, errDel) {
			t.Errorf("Wrong match. expect: (nil, %v), got: (%v, %v)", errDel, res, err)
		}
	})
}

func Test_GetQueuesByAgent_errors(t *testing.T) {
	errList := errors.New("list failed")

	agent := amagent.Agent{
		Identity: commonidentity.Identity{ID: testQueueID, CustomerID: testCustomerID},
	}
	filters := map[queue.Field]any{
		queue.FieldDeleted:    false,
		queue.FieldCustomerID: testCustomerID.String(),
	}

	t.Run("list failure is returned", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newTestHandler(mc)
		ctx := context.Background()

		m.db.EXPECT().QueueList(ctx, uint64(queueListPageSize), "", filters).Return(nil, errList)

		res, err := h.GetQueuesByAgent(ctx, agent)
		if res != nil || !errors.Is(err, errList) {
			t.Errorf("Wrong match. expect: (nil, %v), got: (%v, %v)", errList, res, err)
		}
	})

	t.Run("a full page whose last row has no creation time stops the walk", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newTestHandler(mc)
		ctx := context.Background()

		page := make([]*queue.Queue, 0, queueListPageSize)
		for i := 0; i < queueListPageSize; i++ {
			page = append(page, &queue.Queue{Identity: commonidentity.Identity{ID: uuid.Must(uuid.NewV4())}})
		}

		// exactly one list call: with no tm_create there is no token to continue with.
		m.db.EXPECT().QueueList(ctx, uint64(queueListPageSize), "", filters).Return(page, nil).Times(1)

		res, err := h.GetQueuesByAgent(ctx, agent)
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if len(res) != queueListPageSize {
			t.Errorf("Wrong match. expect: %d untagged queues, got: %d", queueListPageSize, len(res))
		}
	})
}

func Test_GetAgents_errors(t *testing.T) {
	errGet := errors.New("get failed")
	errList := errors.New("agent list failed")

	q := &queue.Queue{
		Identity: commonidentity.Identity{ID: testQueueID, CustomerID: testCustomerID},
	}

	t.Run("queue lookup failure is returned", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newTestHandler(mc)
		ctx := context.Background()

		m.db.EXPECT().QueueGet(ctx, testQueueID).Return(nil, errGet)

		res, err := h.GetAgents(ctx, testQueueID, amagent.StatusAvailable)
		if res != nil || !errors.Is(err, errGet) {
			t.Errorf("Wrong match. expect: (nil, %v), got: (%v, %v)", errGet, res, err)
		}
	})

	t.Run("untagged queue omits the tag filter and agent list failure is returned", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newTestHandler(mc)
		ctx := context.Background()

		expectFilters := map[amagent.Field]any{
			amagent.FieldDeleted:    false,
			amagent.FieldCustomerID: testCustomerID.String(),
			amagent.FieldStatus:     string(amagent.StatusAvailable),
		}

		m.db.EXPECT().QueueGet(ctx, testQueueID).Return(q, nil)
		m.util.EXPECT().TimeGetCurTime().Return("2024-01-01T00:00:00.000Z")
		m.req.EXPECT().AgentV1AgentList(ctx, "2024-01-01T00:00:00.000Z", uint64(100), expectFilters).Return(nil, errList)

		res, err := h.GetAgents(ctx, testQueueID, amagent.StatusAvailable)
		if res != nil || !errors.Is(err, errList) {
			t.Errorf("Wrong match. expect: (nil, %v), got: (%v, %v)", errList, res, err)
		}
	})

	t.Run("status none leaves the status filter out", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newTestHandler(mc)
		ctx := context.Background()

		expectFilters := map[amagent.Field]any{
			amagent.FieldDeleted:    false,
			amagent.FieldCustomerID: testCustomerID.String(),
		}
		agents := []amagent.Agent{{Identity: commonidentity.Identity{ID: testQueuecallID}}}

		m.db.EXPECT().QueueGet(ctx, testQueueID).Return(q, nil)
		m.util.EXPECT().TimeGetCurTime().Return("2024-01-01T00:00:00.000Z")
		m.req.EXPECT().AgentV1AgentList(ctx, "2024-01-01T00:00:00.000Z", uint64(100), expectFilters).Return(agents, nil)

		res, err := h.GetAgents(ctx, testQueueID, amagent.StatusNone)
		if err != nil || len(res) != 1 || res[0].ID != testQueuecallID {
			t.Errorf("Wrong match. expect: (%v, nil), got: (%v, %v)", agents, res, err)
		}
	})
}

func Test_EventCUCustomerDeleted_errors(t *testing.T) {
	errList := errors.New("list failed")
	errGet := errors.New("get failed")

	cu := &cucustomer.Customer{ID: testCustomerID}
	filters := map[queue.Field]any{
		queue.FieldCustomerID: testCustomerID,
		queue.FieldDeleted:    false,
	}

	t.Run("list failure is returned", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newTestHandler(mc)
		ctx := context.Background()

		m.db.EXPECT().QueueList(ctx, uint64(1000), "", filters).Return(nil, errList)

		if err := h.EventCUCustomerDeleted(ctx, cu); !errors.Is(err, errList) {
			t.Errorf("Wrong match. expect: %v, got: %v", errList, err)
		}
	})

	t.Run("one failing deletion does not stop the remaining queues", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newTestHandler(mc)
		ctx := context.Background()

		first := &queue.Queue{Identity: commonidentity.Identity{ID: testQueueID, CustomerID: testCustomerID}}
		secondID := uuid.FromStringOrNil("a1b2c3d4-0006-11ee-8a01-000000000006")
		second := &queue.Queue{Identity: commonidentity.Identity{ID: secondID, CustomerID: testCustomerID}}

		m.db.EXPECT().QueueList(ctx, uint64(1000), "", filters).Return([]*queue.Queue{first, second}, nil)
		m.db.EXPECT().QueueGet(ctx, testQueueID).Return(nil, errGet)
		m.db.EXPECT().QueueGet(ctx, secondID).Return(second, nil)
		m.db.EXPECT().QueueDelete(ctx, secondID).Return(nil)
		m.db.EXPECT().QueueGet(ctx, secondID).Return(second, nil)
		m.notify.EXPECT().PublishEvent(ctx, queue.EventTypeQueueDeleted, second)

		if err := h.EventCUCustomerDeleted(ctx, cu); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	})
}

func Test_CountByCustomerID_error(t *testing.T) {
	errCount := errors.New("count failed")

	mc := gomock.NewController(t)
	defer mc.Finish()

	h, m := newTestHandler(mc)
	ctx := context.Background()

	m.db.EXPECT().QueueCountByCustomerID(ctx, testCustomerID).Return(0, errCount)

	count, err := h.CountByCustomerID(ctx, testCustomerID)
	if count != 0 || !errors.Is(err, errCount) {
		t.Errorf("Wrong match. expect: (0, %v), got: (%d, %v)", errCount, count, err)
	}
}
