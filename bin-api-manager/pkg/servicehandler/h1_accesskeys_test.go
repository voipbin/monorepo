package servicehandler

import (
	"context"
	"testing"

	"monorepo/bin-api-manager/pkg/serviceerrors"
	csaccesskey "monorepo/bin-customer-manager/models/accesskey"
)

func h1AK(customerID string) *csaccesskey.Accesskey {
	return &csaccesskey.Accesskey{
		ID:         h1UUID(h1ResourceID),
		CustomerID: h1UUID(customerID),
		Name:       "ak",
	}
}

func h1AKWebhook(customerID string) *csaccesskey.WebhookMessage {
	return &csaccesskey.WebhookMessage{
		ID:         h1UUID(h1ResourceID),
		CustomerID: h1UUID(customerID),
		Name:       "ak",
	}
}

func Test_accesskeyGet_Guards(t *testing.T) {
	id := h1UUID(h1ResourceID)
	now := h1Now()

	h1Run(t, []h1Case{
		{
			name:    "rpc error",
			setup:   func(m *h1Mocks) { m.req.EXPECT().CustomerV1AccesskeyGet(h1Ctx(), id).Return(nil, errH1RPC) },
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return h.accesskeyGet(ctx, h1Admin(), id) },
			wantErr: errH1RPC,
		},
		{
			name: "cross tenant is hidden as not found",
			setup: func(m *h1Mocks) {
				m.req.EXPECT().CustomerV1AccesskeyGet(h1Ctx(), id).Return(h1AK(h1OtherID), nil)
			},
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return h.accesskeyGet(ctx, h1Admin(), id) },
			wantErr: serviceerrors.ErrNotFound,
		},
		{
			name: "deleted is not found",
			setup: func(m *h1Mocks) {
				ak := h1AK(h1CustomerID)
				ak.TMDelete = &now
				m.req.EXPECT().CustomerV1AccesskeyGet(h1Ctx(), id).Return(ak, nil)
			},
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return h.accesskeyGet(ctx, h1Admin(), id) },
			wantErr: serviceerrors.ErrNotFound,
		},
		{
			name: "success",
			setup: func(m *h1Mocks) {
				m.req.EXPECT().CustomerV1AccesskeyGet(h1Ctx(), id).Return(h1AK(h1CustomerID), nil)
			},
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return h.accesskeyGet(ctx, h1Admin(), id) },
			wantRes: h1AK(h1CustomerID),
		},
	})
}

func Test_AccesskeyCreate_Guards(t *testing.T) {
	cid := h1UUID(h1CustomerID)

	h1Run(t, []h1Case{
		{
			name: "direct access",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.AccesskeyCreate(ctx, h1Direct(), "n", "d", 86400)
			},
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name: "no permission",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.AccesskeyCreate(ctx, h1Manager(), "n", "d", 86400)
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "expire below one day",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.AccesskeyCreate(ctx, h1Admin(), "n", "d", 86399)
			},
			wantErr: serviceerrors.ErrInvalidArgument,
		},
		{
			name: "rpc error",
			setup: func(m *h1Mocks) {
				m.req.EXPECT().CustomerV1AccesskeyCreate(h1Ctx(), cid, "n", "d", int32(86400)).Return(nil, errH1RPC)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.AccesskeyCreate(ctx, h1Admin(), "n", "d", 86400)
			},
			wantErr: errH1RPC,
		},
	})
}

func Test_AccesskeyGet_Guards(t *testing.T) {
	id := h1UUID(h1ResourceID)

	h1Run(t, []h1Case{
		{
			name:    "direct access",
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return h.AccesskeyGet(ctx, h1Direct(), id) },
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name:    "rpc error",
			setup:   func(m *h1Mocks) { m.req.EXPECT().CustomerV1AccesskeyGet(h1Ctx(), id).Return(nil, errH1RPC) },
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return h.AccesskeyGet(ctx, h1Admin(), id) },
			wantErr: errH1RPC,
		},
		{
			name: "no permission",
			setup: func(m *h1Mocks) {
				m.req.EXPECT().CustomerV1AccesskeyGet(h1Ctx(), id).Return(h1AK(h1CustomerID), nil)
			},
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return h.AccesskeyGet(ctx, h1NoPerm(), id) },
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "cross tenant",
			setup: func(m *h1Mocks) {
				m.req.EXPECT().CustomerV1AccesskeyGet(h1Ctx(), id).Return(h1AK(h1CustomerID), nil)
			},
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return h.AccesskeyGet(ctx, h1Foreign(), id) },
			wantErr: serviceerrors.ErrNotFound,
		},
	})
}

func Test_AccesskeyRawGetByToken(t *testing.T) {
	filters := map[csaccesskey.Field]any{
		csaccesskey.FieldTokenHash: "hash-value",
		csaccesskey.FieldDeleted:   false,
	}
	one := csaccesskey.Accesskey{ID: h1UUID(h1ResourceID), CustomerID: h1UUID(h1CustomerID)}
	other := csaccesskey.Accesskey{ID: h1UUID(h1AgentID), CustomerID: h1UUID(h1CustomerID)}

	h1Run(t, []h1Case{
		{
			name: "rpc error",
			setup: func(m *h1Mocks) {
				m.util.EXPECT().HashSHA256Hex("tok").Return("hash-value")
				m.req.EXPECT().CustomerV1AccesskeyList(h1Ctx(), "", uint64(10), filters).Return(nil, errH1RPC)
			},
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return h.AccesskeyRawGetByToken(ctx, "tok") },
			wantErr: errH1RPC,
		},
		{
			name: "no match",
			setup: func(m *h1Mocks) {
				m.util.EXPECT().HashSHA256Hex("tok").Return("hash-value")
				m.req.EXPECT().CustomerV1AccesskeyList(h1Ctx(), "", uint64(10), filters).Return([]csaccesskey.Accesskey{}, nil)
			},
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return h.AccesskeyRawGetByToken(ctx, "tok") },
			wantErr: serviceerrors.ErrNotFound,
		},
		{
			name: "ambiguous match",
			setup: func(m *h1Mocks) {
				m.util.EXPECT().HashSHA256Hex("tok").Return("hash-value")
				m.req.EXPECT().CustomerV1AccesskeyList(h1Ctx(), "", uint64(10), filters).Return([]csaccesskey.Accesskey{one, other}, nil)
			},
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return h.AccesskeyRawGetByToken(ctx, "tok") },
			wantErr: serviceerrors.ErrInternal,
		},
		{
			name: "success",
			setup: func(m *h1Mocks) {
				m.util.EXPECT().HashSHA256Hex("tok").Return("hash-value")
				m.req.EXPECT().CustomerV1AccesskeyList(h1Ctx(), "", uint64(10), filters).Return([]csaccesskey.Accesskey{one}, nil)
			},
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return h.AccesskeyRawGetByToken(ctx, "tok") },
			wantRes: &one,
		},
	})
}

func Test_AccesskeyList_Guards(t *testing.T) {
	filters := map[csaccesskey.Field]any{
		csaccesskey.FieldCustomerID: h1UUID(h1CustomerID),
		csaccesskey.FieldDeleted:    false,
	}

	h1Run(t, []h1Case{
		{
			name: "direct access",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.AccesskeyList(ctx, h1Direct(), 10, "t")
			},
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name: "no permission",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.AccesskeyList(ctx, h1Manager(), 10, "t")
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "empty token uses current time, rpc error",
			setup: func(m *h1Mocks) {
				m.util.EXPECT().TimeGetCurTime().Return("2026-01-01T00:00:00.000000Z")
				m.req.EXPECT().CustomerV1AccesskeyList(h1Ctx(), "2026-01-01T00:00:00.000000Z", uint64(5), filters).Return(nil, errH1RPC)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.AccesskeyList(ctx, h1Admin(), 5, "")
			},
			wantErr: errH1RPC,
		},
	})
}

func Test_AccesskeyDelete_Guards(t *testing.T) {
	id := h1UUID(h1ResourceID)

	h1Run(t, []h1Case{
		{
			name: "direct access",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.AccesskeyDelete(ctx, h1Direct(), id)
			},
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name:  "get error",
			setup: func(m *h1Mocks) { m.req.EXPECT().CustomerV1AccesskeyGet(h1Ctx(), id).Return(nil, errH1RPC) },
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.AccesskeyDelete(ctx, h1Admin(), id)
			},
			wantErr: errH1RPC,
		},
		{
			name: "no permission",
			setup: func(m *h1Mocks) {
				m.req.EXPECT().CustomerV1AccesskeyGet(h1Ctx(), id).Return(h1AK(h1CustomerID), nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.AccesskeyDelete(ctx, h1NoPerm(), id)
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "rpc error",
			setup: func(m *h1Mocks) {
				m.req.EXPECT().CustomerV1AccesskeyGet(h1Ctx(), id).Return(h1AK(h1CustomerID), nil)
				m.req.EXPECT().CustomerV1AccesskeyDelete(h1Ctx(), id).Return(nil, errH1RPC)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.AccesskeyDelete(ctx, h1Admin(), id)
			},
			wantErr: errH1RPC,
		},
	})
}

func Test_AccesskeyUpdate_Guards(t *testing.T) {
	id := h1UUID(h1ResourceID)

	h1Run(t, []h1Case{
		{
			name: "direct access",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.AccesskeyUpdate(ctx, h1Direct(), id, "n", "d")
			},
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name:  "get error",
			setup: func(m *h1Mocks) { m.req.EXPECT().CustomerV1AccesskeyGet(h1Ctx(), id).Return(nil, errH1RPC) },
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.AccesskeyUpdate(ctx, h1Admin(), id, "n", "d")
			},
			wantErr: errH1RPC,
		},
		{
			name: "no permission",
			setup: func(m *h1Mocks) {
				m.req.EXPECT().CustomerV1AccesskeyGet(h1Ctx(), id).Return(h1AK(h1CustomerID), nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.AccesskeyUpdate(ctx, h1NoPerm(), id, "n", "d")
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "rpc error",
			setup: func(m *h1Mocks) {
				m.req.EXPECT().CustomerV1AccesskeyGet(h1Ctx(), id).Return(h1AK(h1CustomerID), nil)
				m.req.EXPECT().CustomerV1AccesskeyUpdate(h1Ctx(), id, "n", "d").Return(nil, errH1RPC)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.AccesskeyUpdate(ctx, h1Admin(), id, "n", "d")
			},
			wantErr: errH1RPC,
		},
		{
			name: "success",
			setup: func(m *h1Mocks) {
				m.req.EXPECT().CustomerV1AccesskeyGet(h1Ctx(), id).Return(h1AK(h1CustomerID), nil)
				m.req.EXPECT().CustomerV1AccesskeyUpdate(h1Ctx(), id, "n", "d").Return(h1AK(h1CustomerID), nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.AccesskeyUpdate(ctx, h1Admin(), id, "n", "d")
			},
			wantRes: h1AKWebhook(h1CustomerID),
		},
	})
}
