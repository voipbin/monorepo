package servicehandler

import (
	"context"
	"testing"

	"monorepo/bin-api-manager/models/auth"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	qmqueuecall "monorepo/bin-queue-manager/models/queuecall"

	"github.com/gofrs/uuid"
)

func h3Queuecall() *qmqueuecall.Queuecall {
	return &qmqueuecall.Queuecall{
		Identity: commonidentity.Identity{ID: h3ResourceID, CustomerID: h3CustomerID},
	}
}

func h3QueuecallLookup(m *requesthandler.MockRequestHandler, ctx context.Context) {
	m.EXPECT().QueueV1QueuecallGet(ctx, h3ResourceID).Return(h3Queuecall(), nil)
}

func h3QueuecallLookupFail(m *requesthandler.MockRequestHandler, ctx context.Context) {
	m.EXPECT().QueueV1QueuecallGet(ctx, h3ResourceID).Return(nil, errH3)
}

func Test_QueuecallGet_guards_h3(t *testing.T) {
	h3RunRes(t, h3GuardRows(h3Ops[*qmqueuecall.WebhookMessage]{
		call: func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*qmqueuecall.WebhookMessage, error) {
			return h.QueuecallGet(ctx, a, h3ResourceID)
		},
		a:      h3Admin(),
		denied: h3OwnedDenied(h3QueuecallLookup),
		opFail: h3QueuecallLookupFail,
		opOK:   h3QueuecallLookup,
		want:   h3Queuecall().ConvertWebhookMessage(),
	}))
}

func Test_QueuecallDeleteKick_h3(t *testing.T) {
	cases := []struct {
		name   string
		call   func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*qmqueuecall.WebhookMessage, error)
		opFail func(m *requesthandler.MockRequestHandler, ctx context.Context)
		opOK   func(m *requesthandler.MockRequestHandler, ctx context.Context)
	}{
		{
			"QueuecallDelete",
			func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*qmqueuecall.WebhookMessage, error) {
				return h.QueuecallDelete(ctx, a, h3ResourceID)
			},
			func(m *requesthandler.MockRequestHandler, ctx context.Context) {
				m.EXPECT().QueueV1QueuecallDelete(ctx, h3ResourceID).Return(nil, errH3)
			},
			func(m *requesthandler.MockRequestHandler, ctx context.Context) {
				m.EXPECT().QueueV1QueuecallDelete(ctx, h3ResourceID).Return(h3Queuecall(), nil)
			},
		},
		{
			"QueuecallKick",
			func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*qmqueuecall.WebhookMessage, error) {
				return h.QueuecallKick(ctx, a, h3ResourceID)
			},
			func(m *requesthandler.MockRequestHandler, ctx context.Context) {
				m.EXPECT().QueueV1QueuecallKick(ctx, h3ResourceID).Return(nil, errH3)
			},
			func(m *requesthandler.MockRequestHandler, ctx context.Context) {
				m.EXPECT().QueueV1QueuecallKick(ctx, h3ResourceID).Return(h3Queuecall(), nil)
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h3RunRes(t, h3GuardRows(h3Ops[*qmqueuecall.WebhookMessage]{
				call:    c.call,
				a:       h3Admin(),
				denied:  h3OwnedDenied(h3QueuecallLookup),
				getFail: h3QueuecallLookupFail,
				opPre:   h3QueuecallLookup,
				opFail:  c.opFail,
				opOK:    c.opOK,
				want:    h3Queuecall().ConvertWebhookMessage(),
			}))
		})
	}
}

func Test_QueuecallKickByReferenceID_h3(t *testing.T) {
	refID := uuid.FromStringOrNil("33333333-0000-0000-0000-000000000003")
	lookup := func(m *requesthandler.MockRequestHandler, ctx context.Context) {
		m.EXPECT().QueueV1QueuecallGetByReferenceID(ctx, refID).Return(h3Queuecall(), nil)
	}
	h3RunRes(t, h3GuardRows(h3Ops[*qmqueuecall.WebhookMessage]{
		call: func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*qmqueuecall.WebhookMessage, error) {
			return h.QueuecallKickByReferenceID(ctx, a, refID)
		},
		a: h3Admin(),
		// the by-reference lookup checks the permission first, so a no-permission caller never reaches the rpc.
		denied: h3NoPermOnly(),
		getFail: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
			m.EXPECT().QueueV1QueuecallGetByReferenceID(ctx, refID).Return(nil, errH3)
		},
		opPre: lookup,
		opFail: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
			m.EXPECT().QueueV1QueuecallKick(ctx, h3ResourceID).Return(nil, errH3)
		},
		opOK: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
			m.EXPECT().QueueV1QueuecallKick(ctx, h3ResourceID).Return(h3Queuecall(), nil)
		},
		want: h3Queuecall().ConvertWebhookMessage(),
	}))
}

func Test_QueuecallKickByReferenceID_otherTenantQueuecall_h3(t *testing.T) {
	refID := uuid.FromStringOrNil("33333333-0000-0000-0000-000000000003")
	other := h3Queuecall()
	other.CustomerID = h3OtherCustomerID
	h3RunRes(t, []h3ResRow[*qmqueuecall.WebhookMessage]{
		{
			name: "queuecall of another customer is denied and not kicked", a: h3Admin(), wantErr: h3ErrDenied,
			call: func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*qmqueuecall.WebhookMessage, error) {
				return h.QueuecallKickByReferenceID(ctx, a, refID)
			},
			setup: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
				m.EXPECT().QueueV1QueuecallGetByReferenceID(ctx, refID).Return(other, nil)
			},
		},
	})
}

func Test_QueuecallList_h3(t *testing.T) {
	filters := map[qmqueuecall.Field]any{
		qmqueuecall.FieldCustomerID: h3CustomerID,
		qmqueuecall.FieldDeleted:    false,
	}
	h3RunRes(t, h3GuardRows(h3Ops[[]*qmqueuecall.WebhookMessage]{
		call: func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) ([]*qmqueuecall.WebhookMessage, error) {
			return h.QueuecallList(ctx, a, 10, "tok")
		},
		a:      h3Admin(),
		denied: h3NoPermOnly(),
		opFail: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
			m.EXPECT().QueueV1QueuecallList(ctx, "tok", uint64(10), filters).Return(nil, errH3)
		},
		opOK: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
			m.EXPECT().QueueV1QueuecallList(ctx, "tok", uint64(10), filters).Return([]qmqueuecall.Queuecall{*h3Queuecall()}, nil)
		},
		want: []*qmqueuecall.WebhookMessage{h3Queuecall().ConvertWebhookMessage()},
	}))
}
