package servicehandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"monorepo/bin-api-manager/pkg/serviceerrors"
	qmqueue "monorepo/bin-queue-manager/models/queue"
)

func h1Queue(customerID string) *qmqueue.Queue {
	return &qmqueue.Queue{
		Identity: commonidentity.Identity{
			ID:         h1UUID(h1ResourceID),
			CustomerID: h1UUID(customerID),
		},
		Name: "q",
	}
}

func h1QueueWebhook(customerID string) *qmqueue.WebhookMessage {
	return &qmqueue.WebhookMessage{
		Identity: commonidentity.Identity{
			ID:         h1UUID(h1ResourceID),
			CustomerID: h1UUID(customerID),
		},
		Name: "q",
	}
}

func Test_QueueGet_Guards(t *testing.T) {
	id := h1UUID(h1ResourceID)
	get := func(m *h1Mocks, c string) {
		m.req.EXPECT().QueueV1QueueGet(h1Ctx(), id).Return(h1Queue(c), nil)
	}

	h1Run(t, []h1Case{
		{
			name:    "direct access",
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return h.QueueGet(ctx, h1Direct(), id) },
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name:    "rpc error",
			setup:   func(m *h1Mocks) { m.req.EXPECT().QueueV1QueueGet(h1Ctx(), id).Return(nil, errH1RPC) },
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return h.QueueGet(ctx, h1Admin(), id) },
			wantErr: errH1RPC,
		},
		{
			name:    "no permission",
			setup:   func(m *h1Mocks) { get(m, h1CustomerID) },
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return h.QueueGet(ctx, h1NoPerm(), id) },
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name:    "cross tenant",
			setup:   func(m *h1Mocks) { get(m, h1CustomerID) },
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return h.QueueGet(ctx, h1Foreign(), id) },
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name:    "manager success",
			setup:   func(m *h1Mocks) { get(m, h1CustomerID) },
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return h.QueueGet(ctx, h1Manager(), id) },
			wantRes: h1QueueWebhook(h1CustomerID),
		},
	})
}

func Test_QueueList_Guards(t *testing.T) {
	filters := map[qmqueue.Field]any{
		qmqueue.FieldCustomerID: h1UUID(h1CustomerID),
		qmqueue.FieldDeleted:    false,
	}

	h1Run(t, []h1Case{
		{
			name: "direct access",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.QueueList(ctx, h1Direct(), 10, "t")
			},
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name: "no permission",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.QueueList(ctx, h1NoPerm(), 10, "t")
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "empty token and rpc error",
			setup: func(m *h1Mocks) {
				m.util.EXPECT().TimeGetCurTime().Return("2026-01-01T00:00:00.000000Z")
				m.req.EXPECT().QueueV1QueueList(h1Ctx(), "2026-01-01T00:00:00.000000Z", uint64(7), filters).Return(nil, errH1RPC)
			},
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return h.QueueList(ctx, h1Admin(), 7, "") },
			wantErr: errH1RPC,
		},
	})
}

func Test_QueueCreate_Guards(t *testing.T) {
	cid := h1UUID(h1CustomerID)
	tags := []uuid.UUID{h1UUID(h1AgentID)}
	wait := h1UUID(h1OtherID)

	create := func(ctx context.Context, h *serviceHandler) (any, error) {
		return h.QueueCreate(ctx, h1Admin(), "q", "d", qmqueue.RoutingMethodRandom, tags, wait, 100, 200)
	}

	h1Run(t, []h1Case{
		{
			name: "direct access",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.QueueCreate(ctx, h1Direct(), "q", "d", qmqueue.RoutingMethodRandom, tags, wait, 100, 200)
			},
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name: "no permission",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.QueueCreate(ctx, h1NoPerm(), "q", "d", qmqueue.RoutingMethodRandom, tags, wait, 100, 200)
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "rpc error",
			setup: func(m *h1Mocks) {
				m.req.EXPECT().QueueV1QueueCreate(h1Ctx(), cid, "q", "d", qmqueue.RoutingMethodRandom, tags, wait, 100, 200).Return(nil, errH1RPC)
			},
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return create(ctx, h) },
			wantErr: errH1RPC,
		},
		{
			name: "success",
			setup: func(m *h1Mocks) {
				m.req.EXPECT().QueueV1QueueCreate(h1Ctx(), cid, "q", "d", qmqueue.RoutingMethodRandom, tags, wait, 100, 200).Return(h1Queue(h1CustomerID), nil)
			},
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return create(ctx, h) },
			wantRes: h1QueueWebhook(h1CustomerID),
		},
	})
}

// h1QueueActCases builds the shared guard/propagation table for queue
// operations of the form: get queue -> permission check -> RPC.
func h1QueueActCases(
	call func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error),
	expectAct func(m *h1Mocks, ret *qmqueue.Queue, err error),
) []h1Case {
	id := h1UUID(h1ResourceID)
	get := func(m *h1Mocks, c string) {
		m.req.EXPECT().QueueV1QueueGet(h1Ctx(), id).Return(h1Queue(c), nil)
	}

	return []h1Case{
		{
			name:    "direct access",
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return call(ctx, h, h1Direct()) },
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name:    "get error",
			setup:   func(m *h1Mocks) { m.req.EXPECT().QueueV1QueueGet(h1Ctx(), id).Return(nil, errH1RPC) },
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return call(ctx, h, h1Admin()) },
			wantErr: errH1RPC,
		},
		{
			name:    "no permission",
			setup:   func(m *h1Mocks) { get(m, h1CustomerID) },
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return call(ctx, h, h1NoPerm()) },
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name:    "cross tenant",
			setup:   func(m *h1Mocks) { get(m, h1CustomerID) },
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return call(ctx, h, h1Foreign()) },
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "action rpc error",
			setup: func(m *h1Mocks) {
				get(m, h1CustomerID)
				expectAct(m, nil, errH1RPC)
			},
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return call(ctx, h, h1Admin()) },
			wantErr: errH1RPC,
		},
		{
			name: "success",
			setup: func(m *h1Mocks) {
				get(m, h1CustomerID)
				expectAct(m, h1Queue(h1CustomerID), nil)
			},
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return call(ctx, h, h1Manager()) },
			wantRes: h1QueueWebhook(h1CustomerID),
		},
	}
}

func Test_QueueDelete_Matrix(t *testing.T) {
	id := h1UUID(h1ResourceID)
	h1Run(t, h1QueueActCases(
		func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error) {
			return h.QueueDelete(ctx, a, id)
		},
		func(m *h1Mocks, ret *qmqueue.Queue, err error) {
			m.req.EXPECT().QueueV1QueueDelete(h1Ctx(), id).Return(ret, err)
		},
	))
}

func Test_QueueUpdate_Matrix(t *testing.T) {
	id := h1UUID(h1ResourceID)
	name := "new"
	detail := "new-detail"
	method := qmqueue.RoutingMethodRandom
	tags := []uuid.UUID{h1UUID(h1AgentID)}
	wait := h1UUID(h1OtherID)
	timeoutWait := 11
	timeoutSvc := 22

	h1Run(t, h1QueueActCases(
		func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error) {
			return h.QueueUpdate(ctx, a, id, &name, &detail, &method, &tags, &wait, &timeoutWait, &timeoutSvc)
		},
		func(m *h1Mocks, ret *qmqueue.Queue, err error) {
			m.req.EXPECT().QueueV1QueueUpdate(h1Ctx(), id, &name, &detail, &method, &tags, &wait, &timeoutWait, &timeoutSvc).Return(ret, err)
		},
	))
}

func Test_QueueUpdateTagIDs_Matrix(t *testing.T) {
	id := h1UUID(h1ResourceID)
	tags := []uuid.UUID{h1UUID(h1AgentID), h1UUID(h1OtherID)}

	h1Run(t, h1QueueActCases(
		func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error) {
			return h.QueueUpdateTagIDs(ctx, a, id, tags)
		},
		func(m *h1Mocks, ret *qmqueue.Queue, err error) {
			m.req.EXPECT().QueueV1QueueUpdateTagIDs(h1Ctx(), id, tags).Return(ret, err)
		},
	))
}

func Test_QueueUpdateRoutingMethod_Matrix(t *testing.T) {
	id := h1UUID(h1ResourceID)

	h1Run(t, h1QueueActCases(
		func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error) {
			return h.QueueUpdateRoutingMethod(ctx, a, id, qmqueue.RoutingMethodRandom)
		},
		func(m *h1Mocks, ret *qmqueue.Queue, err error) {
			m.req.EXPECT().QueueV1QueueUpdateRoutingMethod(h1Ctx(), id, qmqueue.RoutingMethodRandom).Return(ret, err)
		},
	))
}

func Test_QueueDirectHashRegenerate_Matrix(t *testing.T) {
	id := h1UUID(h1ResourceID)

	h1Run(t, h1QueueActCases(
		func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error) {
			return h.QueueDirectHashRegenerate(ctx, a, id)
		},
		func(m *h1Mocks, ret *qmqueue.Queue, err error) {
			m.req.EXPECT().QueueV1QueueDirectHashRegenerate(h1Ctx(), id).Return(ret, err)
		},
	))
}

func Test_convertQueueFilters_Invalid(t *testing.T) {
	h := &serviceHandler{}

	res, err := h.convertQueueFilters(map[string]string{"customer_id": "not-a-uuid"})
	if err == nil {
		t.Fatalf("Wrong match. expect: error, got: nil (res: %v)", res)
	}
	if res != nil {
		t.Errorf("Wrong result. expect: nil, got: %v", res)
	}
}
