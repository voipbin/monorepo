package servicehandler

import (
	"context"
	"strings"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	fmflow "monorepo/bin-flow-manager/models/flow"
	wcsession "monorepo/bin-webchat-manager/models/session"
	wcwidget "monorepo/bin-webchat-manager/models/widget"
)

const h1NilUUIDString = "00000000-0000-0000-0000-000000000000"

func h1Widget(customerID string) *wcwidget.Widget {
	return &wcwidget.Widget{
		Identity: commonidentity.Identity{
			ID:         h1UUID(h1ResourceID),
			CustomerID: h1UUID(customerID),
		},
		Name: "w",
	}
}

func h1WidgetWebhook(customerID string) *wcwidget.WebhookMessage {
	return &wcwidget.WebhookMessage{
		Identity: commonidentity.Identity{
			ID:         h1UUID(h1ResourceID),
			CustomerID: h1UUID(customerID),
		},
		Name:          "w",
		SessionFlowID: h1NilUUIDString,
		MessageFlowID: h1NilUUIDString,
	}
}

func h1Flow(id, customerID string) *fmflow.Flow {
	return &fmflow.Flow{
		Identity: commonidentity.Identity{
			ID:         h1UUID(id),
			CustomerID: h1UUID(customerID),
		},
	}
}

// h1WidgetActCases builds the table for widget operations of the form:
// direct check -> widget get -> permission -> RPC.
func h1WidgetActCases(
	call func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error),
	expectAct func(m *h1Mocks, ret *wcwidget.Widget, err error),
) []h1Case {
	id := h1UUID(h1ResourceID)
	now := h1Now()

	return []h1Case{
		{
			name:    "direct access",
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return call(ctx, h, h1Direct()) },
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name:    "get error",
			setup:   func(m *h1Mocks) { m.req.EXPECT().WebchatV1WidgetGet(h1Ctx(), id).Return(nil, errH1RPC) },
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return call(ctx, h, h1Admin()) },
			wantErr: errH1RPC,
		},
		{
			name: "deleted widget is not found",
			setup: func(m *h1Mocks) {
				w := h1Widget(h1CustomerID)
				w.TMDelete = &now
				m.req.EXPECT().WebchatV1WidgetGet(h1Ctx(), id).Return(w, nil)
			},
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return call(ctx, h, h1Admin()) },
			wantErr: serviceerrors.ErrNotFound,
		},
		{
			name:    "no permission",
			setup:   func(m *h1Mocks) { m.req.EXPECT().WebchatV1WidgetGet(h1Ctx(), id).Return(h1Widget(h1CustomerID), nil) },
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return call(ctx, h, h1NoPerm()) },
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name:    "cross tenant",
			setup:   func(m *h1Mocks) { m.req.EXPECT().WebchatV1WidgetGet(h1Ctx(), id).Return(h1Widget(h1CustomerID), nil) },
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return call(ctx, h, h1Foreign()) },
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "action rpc error",
			setup: func(m *h1Mocks) {
				m.req.EXPECT().WebchatV1WidgetGet(h1Ctx(), id).Return(h1Widget(h1CustomerID), nil)
				expectAct(m, nil, errH1RPC)
			},
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return call(ctx, h, h1Manager()) },
			wantErr: errH1RPC,
		},
		{
			name: "success",
			setup: func(m *h1Mocks) {
				m.req.EXPECT().WebchatV1WidgetGet(h1Ctx(), id).Return(h1Widget(h1CustomerID), nil)
				expectAct(m, h1Widget(h1CustomerID), nil)
			},
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return call(ctx, h, h1Manager()) },
			wantRes: h1WidgetWebhook(h1CustomerID),
		},
	}
}

func Test_WebchatWidgetGet_Matrix(t *testing.T) {
	id := h1UUID(h1ResourceID)
	h1Run(t, []h1Case{
		{
			name: "direct access",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.WebchatWidgetGet(ctx, h1Direct(), id)
			},
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name:  "rpc error",
			setup: func(m *h1Mocks) { m.req.EXPECT().WebchatV1WidgetGet(h1Ctx(), id).Return(nil, errH1RPC) },
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.WebchatWidgetGet(ctx, h1Admin(), id)
			},
			wantErr: errH1RPC,
		},
		{
			name:  "cross tenant",
			setup: func(m *h1Mocks) { m.req.EXPECT().WebchatV1WidgetGet(h1Ctx(), id).Return(h1Widget(h1CustomerID), nil) },
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.WebchatWidgetGet(ctx, h1Foreign(), id)
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name:  "success",
			setup: func(m *h1Mocks) { m.req.EXPECT().WebchatV1WidgetGet(h1Ctx(), id).Return(h1Widget(h1CustomerID), nil) },
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.WebchatWidgetGet(ctx, h1Manager(), id)
			},
			wantRes: h1WidgetWebhook(h1CustomerID),
		},
	})
}

func Test_WebchatWidgetList_Matrix(t *testing.T) {
	filters := map[wcwidget.Field]any{
		wcwidget.FieldCustomerID: h1UUID(h1CustomerID),
		wcwidget.FieldDeleted:    false,
	}

	h1Run(t, []h1Case{
		{
			name: "direct access",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.WebchatWidgetList(ctx, h1Direct(), 10, "t")
			},
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name: "no permission",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.WebchatWidgetList(ctx, h1NoPerm(), 10, "t")
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "empty token and rpc error",
			setup: func(m *h1Mocks) {
				m.util.EXPECT().TimeGetCurTime().Return("2026-01-01T00:00:00.000000Z")
				m.req.EXPECT().WebchatV1WidgetList(h1Ctx(), "2026-01-01T00:00:00.000000Z", uint64(3), filters).Return(nil, errH1RPC)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.WebchatWidgetList(ctx, h1Admin(), 3, "")
			},
			wantErr: errH1RPC,
		},
		{
			name: "success",
			setup: func(m *h1Mocks) {
				m.req.EXPECT().WebchatV1WidgetList(h1Ctx(), "tok", uint64(3), filters).Return([]*wcwidget.Widget{h1Widget(h1CustomerID)}, nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.WebchatWidgetList(ctx, h1Manager(), 3, "tok")
			},
			wantRes: []*wcwidget.WebhookMessage{h1WidgetWebhook(h1CustomerID)},
		},
	})
}

func Test_WebchatWidgetCreate_Matrix(t *testing.T) {
	cid := h1UUID(h1CustomerID)
	sf := h1UUID("6a1f0b10-0000-4000-8000-0000000000f1")
	mf := h1UUID("6a1f0b10-0000-4000-8000-0000000000f2")
	theme := &wcwidget.ThemeConfig{}

	create := func(ctx context.Context, h *serviceHandler, a *h1Identity, s, m uuid.UUID) (any, error) {
		return h.WebchatWidgetCreate(ctx, a, "w", s, m, 60, theme)
	}

	h1Run(t, []h1Case{
		{
			name:    "direct access",
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return create(ctx, h, h1Direct(), sf, mf) },
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name:    "no permission",
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return create(ctx, h, h1NoPerm(), sf, mf) },
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name:    "session flow lookup error",
			setup:   func(m *h1Mocks) { m.req.EXPECT().FlowV1FlowGet(h1Ctx(), sf).Return(nil, errH1RPC) },
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return create(ctx, h, h1Admin(), sf, mf) },
			wantErr: errH1RPC,
		},
		{
			name: "session flow of another customer",
			setup: func(m *h1Mocks) {
				m.req.EXPECT().FlowV1FlowGet(h1Ctx(), sf).Return(h1Flow(h1ResourceID, h1OtherID), nil)
			},
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return create(ctx, h, h1Admin(), sf, mf) },
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "message flow lookup error",
			setup: func(m *h1Mocks) {
				m.req.EXPECT().FlowV1FlowGet(h1Ctx(), sf).Return(h1Flow(h1ResourceID, h1CustomerID), nil)
				m.req.EXPECT().FlowV1FlowGet(h1Ctx(), mf).Return(nil, errH1RPC)
			},
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return create(ctx, h, h1Admin(), sf, mf) },
			wantErr: errH1RPC,
		},
		{
			name: "message flow of another customer",
			setup: func(m *h1Mocks) {
				m.req.EXPECT().FlowV1FlowGet(h1Ctx(), sf).Return(h1Flow(h1ResourceID, h1CustomerID), nil)
				m.req.EXPECT().FlowV1FlowGet(h1Ctx(), mf).Return(h1Flow(h1AgentID, h1OtherID), nil)
			},
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return create(ctx, h, h1Admin(), sf, mf) },
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "rpc error without flows",
			setup: func(m *h1Mocks) {
				m.req.EXPECT().WebchatV1WidgetCreate(h1Ctx(), cid, "w", uuid.Nil, uuid.Nil, 60, theme).Return(nil, errH1RPC)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return create(ctx, h, h1Admin(), uuid.Nil, uuid.Nil)
			},
			wantErr: errH1RPC,
		},
		{
			name: "success with both flows",
			setup: func(m *h1Mocks) {
				m.req.EXPECT().FlowV1FlowGet(h1Ctx(), sf).Return(h1Flow(h1ResourceID, h1CustomerID), nil)
				m.req.EXPECT().FlowV1FlowGet(h1Ctx(), mf).Return(h1Flow(h1AgentID, h1CustomerID), nil)
				w := h1Widget(h1CustomerID)
				w.SessionFlowID = sf
				w.MessageFlowID = mf
				m.req.EXPECT().WebchatV1WidgetCreate(h1Ctx(), cid, "w", sf, mf, 60, theme).Return(w, nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) { return create(ctx, h, h1Manager(), sf, mf) },
			wantRes: func() *wcwidget.WebhookMessage {
				w := h1WidgetWebhook(h1CustomerID)
				w.SessionFlowID = sf.String()
				w.MessageFlowID = mf.String()
				return w
			}(),
		},
	})
}

func Test_WebchatWidgetUpdate_Matrix(t *testing.T) {
	id := h1UUID(h1ResourceID)
	sf := h1UUID("6a1f0b10-0000-4000-8000-0000000000f1")
	mf := h1UUID("6a1f0b10-0000-4000-8000-0000000000f2")
	theme := &wcwidget.ThemeConfig{}
	getWidget := func(m *h1Mocks) {
		m.req.EXPECT().WebchatV1WidgetGet(h1Ctx(), id).Return(h1Widget(h1CustomerID), nil)
	}
	update := func(ctx context.Context, h *serviceHandler, a *h1Identity, s, m uuid.UUID) (any, error) {
		return h.WebchatWidgetUpdate(ctx, a, id, "w", s, m, 60, theme)
	}

	// shared guards (direct, get error, deleted, permission, rpc error, success)
	tests := h1WidgetActCases(
		func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error) {
			return update(ctx, h, a, uuid.Nil, uuid.Nil)
		},
		func(m *h1Mocks, ret *wcwidget.Widget, err error) {
			m.req.EXPECT().WebchatV1WidgetUpdate(h1Ctx(), id, "w", uuid.Nil, uuid.Nil, 60, theme).Return(ret, err)
		},
	)

	tests = append(tests,
		h1Case{
			name:    "session flow lookup error",
			setup:   func(m *h1Mocks) { getWidget(m); m.req.EXPECT().FlowV1FlowGet(h1Ctx(), sf).Return(nil, errH1RPC) },
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return update(ctx, h, h1Admin(), sf, mf) },
			wantErr: errH1RPC,
		},
		h1Case{
			name: "session flow of another customer",
			setup: func(m *h1Mocks) {
				getWidget(m)
				m.req.EXPECT().FlowV1FlowGet(h1Ctx(), sf).Return(h1Flow(h1ResourceID, h1OtherID), nil)
			},
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return update(ctx, h, h1Admin(), sf, mf) },
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		h1Case{
			name: "message flow lookup error",
			setup: func(m *h1Mocks) {
				getWidget(m)
				m.req.EXPECT().FlowV1FlowGet(h1Ctx(), sf).Return(h1Flow(h1ResourceID, h1CustomerID), nil)
				m.req.EXPECT().FlowV1FlowGet(h1Ctx(), mf).Return(nil, errH1RPC)
			},
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return update(ctx, h, h1Admin(), sf, mf) },
			wantErr: errH1RPC,
		},
		h1Case{
			name: "message flow of another customer",
			setup: func(m *h1Mocks) {
				getWidget(m)
				m.req.EXPECT().FlowV1FlowGet(h1Ctx(), sf).Return(h1Flow(h1ResourceID, h1CustomerID), nil)
				m.req.EXPECT().FlowV1FlowGet(h1Ctx(), mf).Return(h1Flow(h1AgentID, h1OtherID), nil)
			},
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return update(ctx, h, h1Admin(), sf, mf) },
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		h1Case{
			name: "success with both flows",
			setup: func(m *h1Mocks) {
				getWidget(m)
				m.req.EXPECT().FlowV1FlowGet(h1Ctx(), sf).Return(h1Flow(h1ResourceID, h1CustomerID), nil)
				m.req.EXPECT().FlowV1FlowGet(h1Ctx(), mf).Return(h1Flow(h1AgentID, h1CustomerID), nil)
				m.req.EXPECT().WebchatV1WidgetUpdate(h1Ctx(), id, "w", sf, mf, 60, theme).Return(h1Widget(h1CustomerID), nil)
			},
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return update(ctx, h, h1Admin(), sf, mf) },
			wantRes: h1WidgetWebhook(h1CustomerID),
		},
	)

	h1Run(t, tests)
}

func Test_WebchatWidgetDelete_Matrix(t *testing.T) {
	id := h1UUID(h1ResourceID)
	h1Run(t, h1WidgetActCases(
		func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error) {
			return h.WebchatWidgetDelete(ctx, a, id)
		},
		func(m *h1Mocks, ret *wcwidget.Widget, err error) {
			m.req.EXPECT().WebchatV1WidgetDelete(h1Ctx(), id).Return(ret, err)
		},
	))
}

func Test_WebchatWidgetDirectHashRegenerate_Matrix(t *testing.T) {
	id := h1UUID(h1ResourceID)
	h1Run(t, h1WidgetActCases(
		func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error) {
			return h.WebchatWidgetDirectHashRegenerate(ctx, a, id)
		},
		func(m *h1Mocks, ret *wcwidget.Widget, err error) {
			m.req.EXPECT().WebchatV1WidgetDirectHashRegenerate(h1Ctx(), id).Return(ret, err)
		},
	))
}

// ---- sessions ----

func h1Session(customerID string) *wcsession.Session {
	return &wcsession.Session{
		Identity: commonidentity.Identity{
			ID:         h1UUID(h1AgentID),
			CustomerID: h1UUID(customerID),
		},
		WidgetID: h1UUID(h1ResourceID),
	}
}

func h1SessionWebhook(customerID string) *wcsession.WebhookMessage {
	return &wcsession.WebhookMessage{
		Identity: commonidentity.Identity{
			ID:         h1UUID(h1AgentID),
			CustomerID: h1UUID(customerID),
		},
		WidgetID: h1UUID(h1ResourceID),
	}
}

// h1DirectSessionScope returns a direct identity bound to the widget.
func h1DirectSessionScope(types []string, resourceID, allowedID string) *h1Identity {
	return auth.NewDirectIdentity(&auth.DirectScope{
		CustomerID:           h1UUID(h1CustomerID),
		ResourceType:         "webchat_widget",
		ResourceID:           h1UUID(resourceID),
		AllowedResourceTypes: types,
		AllowedResourceID:    h1UUID(allowedID),
	})
}

func Test_sessionGet_Paths(t *testing.T) {
	id := h1UUID(h1AgentID)
	now := h1Now()

	h1Run(t, []h1Case{
		{
			name:    "rpc error",
			setup:   func(m *h1Mocks) { m.req.EXPECT().WebchatV1SessionGet(h1Ctx(), id).Return(nil, errH1RPC) },
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return h.sessionGet(ctx, id) },
			wantErr: errH1RPC,
		},
		{
			name: "deleted session is not found",
			setup: func(m *h1Mocks) {
				s := h1Session(h1CustomerID)
				s.TMDelete = &now
				m.req.EXPECT().WebchatV1SessionGet(h1Ctx(), id).Return(s, nil)
			},
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return h.sessionGet(ctx, id) },
			wantErr: serviceerrors.ErrNotFound,
		},
		{
			name:    "success",
			setup:   func(m *h1Mocks) { m.req.EXPECT().WebchatV1SessionGet(h1Ctx(), id).Return(h1Session(h1CustomerID), nil) },
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return h.sessionGet(ctx, id) },
			wantRes: h1Session(h1CustomerID),
		},
	})
}

func Test_WebchatSessionGet_Matrix(t *testing.T) {
	id := h1UUID(h1AgentID)
	get := func(m *h1Mocks) {
		m.req.EXPECT().WebchatV1SessionGet(h1Ctx(), id).Return(h1Session(h1CustomerID), nil)
	}

	h1Run(t, []h1Case{
		{
			name: "direct access",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.WebchatSessionGet(ctx, h1Direct(), id)
			},
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name:  "rpc error",
			setup: func(m *h1Mocks) { m.req.EXPECT().WebchatV1SessionGet(h1Ctx(), id).Return(nil, errH1RPC) },
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.WebchatSessionGet(ctx, h1Admin(), id)
			},
			wantErr: errH1RPC,
		},
		{
			name:  "no permission",
			setup: get,
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.WebchatSessionGet(ctx, h1NoPerm(), id)
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name:  "cross tenant",
			setup: get,
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.WebchatSessionGet(ctx, h1Foreign(), id)
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name:  "success",
			setup: get,
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.WebchatSessionGet(ctx, h1Manager(), id)
			},
			wantRes: h1SessionWebhook(h1CustomerID),
		},
	})
}

func Test_WebchatSessionList_Matrix(t *testing.T) {
	wid := h1UUID(h1ResourceID)
	base := map[wcsession.Field]any{
		wcsession.FieldCustomerID: h1UUID(h1CustomerID),
		wcsession.FieldDeleted:    false,
	}
	withWidget := map[wcsession.Field]any{
		wcsession.FieldCustomerID: h1UUID(h1CustomerID),
		wcsession.FieldDeleted:    false,
		wcsession.FieldWidgetID:   wid,
	}

	h1Run(t, []h1Case{
		{
			name: "direct access",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.WebchatSessionList(ctx, h1Direct(), 10, "t", uuid.Nil)
			},
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name: "no permission",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.WebchatSessionList(ctx, h1NoPerm(), 10, "t", uuid.Nil)
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name:  "widget lookup error",
			setup: func(m *h1Mocks) { m.req.EXPECT().WebchatV1WidgetGet(h1Ctx(), wid).Return(nil, errH1RPC) },
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.WebchatSessionList(ctx, h1Admin(), 10, "t", wid)
			},
			wantErr: errH1RPC,
		},
		{
			name:  "widget of another customer",
			setup: func(m *h1Mocks) { m.req.EXPECT().WebchatV1WidgetGet(h1Ctx(), wid).Return(h1Widget(h1OtherID), nil) },
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.WebchatSessionList(ctx, h1Admin(), 10, "t", wid)
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "empty token and rpc error",
			setup: func(m *h1Mocks) {
				m.util.EXPECT().TimeGetCurTime().Return("2026-01-01T00:00:00.000000Z")
				m.req.EXPECT().WebchatV1SessionList(h1Ctx(), "2026-01-01T00:00:00.000000Z", uint64(4), base).Return(nil, errH1RPC)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.WebchatSessionList(ctx, h1Admin(), 4, "", uuid.Nil)
			},
			wantErr: errH1RPC,
		},
		{
			name: "success filtered by widget",
			setup: func(m *h1Mocks) {
				m.req.EXPECT().WebchatV1WidgetGet(h1Ctx(), wid).Return(h1Widget(h1CustomerID), nil)
				m.req.EXPECT().WebchatV1SessionList(h1Ctx(), "tok", uint64(4), withWidget).Return([]*wcsession.Session{h1Session(h1CustomerID)}, nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.WebchatSessionList(ctx, h1Manager(), 4, "tok", wid)
			},
			wantRes: []*wcsession.WebhookMessage{h1SessionWebhook(h1CustomerID)},
		},
	})
}

func Test_WebchatSessionCreate_Matrix(t *testing.T) {
	wid := h1UUID(h1ResourceID)
	cid := h1UUID(h1CustomerID)
	sid := h1UUID(h1AgentID)
	longURL := "https://example.com/" + strings.Repeat("a", 2048)
	getWidget := func(m *h1Mocks) {
		m.req.EXPECT().WebchatV1WidgetGet(h1Ctx(), wid).Return(h1Widget(h1CustomerID), nil)
	}
	create := func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error) {
		return h.WebchatSessionCreate(ctx, a, wid, "https://example.com/page", "https://example.com/ref")
	}
	types := []string{"webchat_session"}

	h1Run(t, []h1Case{
		{
			name: "page url too long",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.WebchatSessionCreate(ctx, h1Admin(), wid, longURL, "")
			},
			wantErr: serviceerrors.ErrInvalidArgument,
		},
		{
			name: "page url scheme",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.WebchatSessionCreate(ctx, h1Admin(), wid, "ftp://example.com/page", "")
			},
			wantErr: serviceerrors.ErrInvalidArgument,
		},
		{
			name: "referrer too long",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.WebchatSessionCreate(ctx, h1Admin(), wid, "", longURL)
			},
			wantErr: serviceerrors.ErrInvalidArgument,
		},
		{
			name: "referrer scheme",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.WebchatSessionCreate(ctx, h1Admin(), wid, "", "javascript:alert(1)")
			},
			wantErr: serviceerrors.ErrInvalidArgument,
		},
		{
			name:    "agent widget lookup error",
			setup:   func(m *h1Mocks) { m.req.EXPECT().WebchatV1WidgetGet(h1Ctx(), wid).Return(nil, errH1RPC) },
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return create(ctx, h, h1Admin()) },
			wantErr: errH1RPC,
		},
		{
			name:    "agent without permission",
			setup:   getWidget,
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return create(ctx, h, h1NoPerm()) },
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name:    "agent of another customer",
			setup:   getWidget,
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return create(ctx, h, h1Foreign()) },
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "agent success creates with a nil session id",
			setup: func(m *h1Mocks) {
				getWidget(m)
				m.req.EXPECT().WebchatV1SessionCreate(h1Ctx(), uuid.Nil, cid, wid, "https://example.com/page", "https://example.com/ref").Return(h1Session(h1CustomerID), nil)
			},
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return create(ctx, h, h1Admin()) },
			wantRes: h1SessionWebhook(h1CustomerID),
		},
		{
			name: "agent rpc error",
			setup: func(m *h1Mocks) {
				getWidget(m)
				m.req.EXPECT().WebchatV1SessionCreate(h1Ctx(), uuid.Nil, cid, wid, "https://example.com/page", "https://example.com/ref").Return(nil, errH1RPC)
			},
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return create(ctx, h, h1Admin()) },
			wantErr: errH1RPC,
		},
		{
			name: "direct without the webchat_session resource type",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return create(ctx, h, h1DirectSessionScope([]string{"other"}, h1ResourceID, h1AgentID))
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "direct bound to another widget",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return create(ctx, h, h1DirectSessionScope(types, h1OtherID, h1AgentID))
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name:  "direct widget lookup error",
			setup: func(m *h1Mocks) { m.req.EXPECT().WebchatV1WidgetGet(h1Ctx(), wid).Return(nil, errH1RPC) },
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return create(ctx, h, h1DirectSessionScope(types, h1ResourceID, h1AgentID))
			},
			wantErr: errH1RPC,
		},
		{
			name:  "direct without an allowed resource id",
			setup: getWidget,
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return create(ctx, h, h1DirectSessionScope(types, h1ResourceID, h1NilUUIDString))
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "direct success forces the allowed session id",
			setup: func(m *h1Mocks) {
				getWidget(m)
				m.req.EXPECT().WebchatV1SessionCreate(h1Ctx(), sid, cid, wid, "https://example.com/page", "https://example.com/ref").Return(h1Session(h1CustomerID), nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return create(ctx, h, h1DirectSessionScope(types, h1ResourceID, h1AgentID))
			},
			wantRes: h1SessionWebhook(h1CustomerID),
		},
		{
			name: "delegate identity is refused",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return create(ctx, h, auth.NewDelegateIdentity(&auth.DelegateScope{CustomerID: cid}))
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
	})
}

func Test_validatePageURL_Referrer_Boundary(t *testing.T) {
	exact := "https://" + strings.Repeat("a", 2048-len("https://"))
	over := exact + "a"

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"empty is allowed", "", false},
		{"http is allowed", "http://example.com", false},
		{"exactly 2048 bytes is allowed", exact, false},
		{"2049 bytes is rejected", over, true},
		{"other scheme is rejected", "ftp://example.com", true},
		{"no scheme is rejected", "example.com", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for fn, f := range map[string]func(string) error{"validatePageURL": validatePageURL, "validateReferrer": validateReferrer} {
				err := f(tt.value)
				if tt.wantErr {
					if err == nil || !strings.Contains(err.Error(), serviceerrors.ErrInvalidArgument.Error()) {
						t.Errorf("%s: expect ErrInvalidArgument, got: %v", fn, err)
					}
					continue
				}
				if err != nil {
					t.Errorf("%s: expect nil, got: %v", fn, err)
				}
			}
		})
	}
}

func Test_WebchatSessionDelete_Matrix(t *testing.T) {
	id := h1UUID(h1AgentID)
	get := func(m *h1Mocks) {
		m.req.EXPECT().WebchatV1SessionGet(h1Ctx(), id).Return(h1Session(h1CustomerID), nil)
	}

	h1Run(t, []h1Case{
		{
			name: "direct access",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.WebchatSessionDelete(ctx, h1Direct(), id)
			},
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name:  "get error",
			setup: func(m *h1Mocks) { m.req.EXPECT().WebchatV1SessionGet(h1Ctx(), id).Return(nil, errH1RPC) },
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.WebchatSessionDelete(ctx, h1Admin(), id)
			},
			wantErr: errH1RPC,
		},
		{
			name:  "cross tenant",
			setup: get,
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.WebchatSessionDelete(ctx, h1Foreign(), id)
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "rpc error",
			setup: func(m *h1Mocks) {
				get(m)
				m.req.EXPECT().WebchatV1SessionDelete(h1Ctx(), id).Return(nil, errH1RPC)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.WebchatSessionDelete(ctx, h1Admin(), id)
			},
			wantErr: errH1RPC,
		},
		{
			name: "success",
			setup: func(m *h1Mocks) {
				get(m)
				m.req.EXPECT().WebchatV1SessionDelete(h1Ctx(), id).Return(h1Session(h1CustomerID), nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.WebchatSessionDelete(ctx, h1Manager(), id)
			},
			wantRes: h1SessionWebhook(h1CustomerID),
		},
	})
}

func Test_WebchatSessionEnd_Matrix(t *testing.T) {
	id := h1UUID(h1AgentID)
	wid := h1UUID(h1ResourceID)
	types := []string{"webchat_session"}
	getSession := func(m *h1Mocks) {
		m.req.EXPECT().WebchatV1SessionGet(h1Ctx(), id).Return(h1Session(h1CustomerID), nil)
	}
	expectEnd := func(m *h1Mocks) {
		m.req.EXPECT().WebchatV1SessionEnd(h1Ctx(), id).Return(h1Session(h1CustomerID), nil)
	}
	end := func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error) {
		return h.WebchatSessionEnd(ctx, a, id)
	}

	h1Run(t, []h1Case{
		{
			name:    "agent session lookup error",
			setup:   func(m *h1Mocks) { m.req.EXPECT().WebchatV1SessionGet(h1Ctx(), id).Return(nil, errH1RPC) },
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return end(ctx, h, h1Admin()) },
			wantErr: errH1RPC,
		},
		{
			name:    "agent without permission",
			setup:   getSession,
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return end(ctx, h, h1NoPerm()) },
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name:    "agent of another customer",
			setup:   getSession,
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return end(ctx, h, h1Foreign()) },
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name:    "agent success",
			setup:   func(m *h1Mocks) { getSession(m); expectEnd(m) },
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return end(ctx, h, h1Manager()) },
			wantRes: h1SessionWebhook(h1CustomerID),
		},
		{
			name: "direct without the webchat_session resource type",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return end(ctx, h, h1DirectSessionScope([]string{"other"}, h1ResourceID, h1AgentID))
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name:  "direct session lookup error",
			setup: func(m *h1Mocks) { m.req.EXPECT().WebchatV1SessionGet(h1Ctx(), id).Return(nil, errH1RPC) },
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return end(ctx, h, h1DirectSessionScope(types, h1ResourceID, h1AgentID))
			},
			wantErr: errH1RPC,
		},
		{
			name:  "direct session of another widget",
			setup: getSession,
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return end(ctx, h, h1DirectSessionScope(types, h1OtherID, h1AgentID))
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "direct widget lookup error",
			setup: func(m *h1Mocks) {
				getSession(m)
				m.req.EXPECT().WebchatV1WidgetGet(h1Ctx(), wid).Return(nil, errH1RPC)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return end(ctx, h, h1DirectSessionScope(types, h1ResourceID, h1AgentID))
			},
			wantErr: errH1RPC,
		},
		{
			name: "direct success",
			setup: func(m *h1Mocks) {
				getSession(m)
				m.req.EXPECT().WebchatV1WidgetGet(h1Ctx(), wid).Return(h1Widget(h1CustomerID), nil)
				expectEnd(m)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return end(ctx, h, h1DirectSessionScope(types, h1ResourceID, h1AgentID))
			},
			wantRes: h1SessionWebhook(h1CustomerID),
		},
		{
			name: "delegate identity is refused",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return end(ctx, h, auth.NewDelegateIdentity(&auth.DelegateScope{CustomerID: h1UUID(h1CustomerID)}))
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "rpc error",
			setup: func(m *h1Mocks) {
				getSession(m)
				m.req.EXPECT().WebchatV1SessionEnd(h1Ctx(), id).Return(nil, errH1RPC)
			},
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return end(ctx, h, h1Admin()) },
			wantErr: errH1RPC,
		},
	})
}
