package servicehandler

import (
	"context"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	csaccesskey "monorepo/bin-customer-manager/models/accesskey"
	wcmessage "monorepo/bin-webchat-manager/models/message"
	wcsession "monorepo/bin-webchat-manager/models/session"
	wcwidget "monorepo/bin-webchat-manager/models/widget"

	"github.com/gofrs/uuid"
)

var (
	h3WebSessionID   = uuid.FromStringOrNil("aa847807-6cc4-4713-9dec-53a42840e74c")
	h3WebWidgetID    = uuid.FromStringOrNil("bb847807-6cc4-4713-9dec-53a42840e74c")
	h3WebAccesskeyID = uuid.FromStringOrNil("cc847807-6cc4-4713-9dec-53a42840e74c")
)

func h3WebMsg() *wcmessage.Message {
	return &wcmessage.Message{
		Identity:  commonidentity.Identity{ID: h3ResourceID, CustomerID: h3CustomerID},
		SessionID: h3WebSessionID,
		Direction: wcmessage.DirectionOutbound,
	}
}

func h3WebSession(status wcsession.Status, customerID, widgetID uuid.UUID) *wcsession.Session {
	return &wcsession.Session{
		Identity: commonidentity.Identity{ID: h3WebSessionID, CustomerID: customerID},
		WidgetID: widgetID,
		Status:   status,
	}
}

func h3WebDirect(allowedTypes []string, allowedID uuid.UUID) *auth.AuthIdentity {
	return auth.NewDirectIdentity(&auth.DirectScope{
		CustomerID:           h3CustomerID,
		ResourceType:         "webchat_widget",
		ResourceID:           h3WebWidgetID,
		AllowedResourceTypes: allowedTypes,
		AllowedResourceID:    allowedID,
	})
}

func h3WebSessionRPC(m *h3Mock, ctx context.Context, s *wcsession.Session, err error) {
	m.EXPECT().WebchatV1SessionGet(ctx, h3WebSessionID).Return(s, err)
}

func h3WebWidgetRPC(m *h3Mock, ctx context.Context, err error) {
	var w *wcwidget.Widget
	if err == nil {
		w = &wcwidget.Widget{Identity: commonidentity.Identity{ID: h3WebWidgetID, CustomerID: h3CustomerID}}
	}
	m.EXPECT().WebchatV1WidgetGet(ctx, h3WebWidgetID).Return(w, err)
}

func Test_WebchatMessageGet_guards_h3(t *testing.T) {
	lookup := func(m *h3Mock, ctx context.Context) {
		m.EXPECT().WebchatV1MessageGet(ctx, h3ResourceID).Return(h3WebMsg(), nil)
	}
	fail := func(m *h3Mock, ctx context.Context) {
		m.EXPECT().WebchatV1MessageGet(ctx, h3ResourceID).Return(nil, errH3)
	}
	o := h3Ops[*wcmessage.WebhookMessage]{
		call: func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*wcmessage.WebhookMessage, error) {
			return h.WebchatMessageGet(ctx, a, h3ResourceID)
		},
		a:      h3Admin(),
		denied: h3OwnedDenied(lookup),
		opFail: fail,
		opOK:   lookup,
		want:   h3WebMsg().ConvertWebhookMessage(),
	}
	rows := h3GuardRows(o)
	rows = append(rows, h3ResRow[*wcmessage.WebhookMessage]{
		name: "soft-deleted message is not found", a: h3Admin(), call: o.call, wantErr: serviceerrors.ErrNotFound,
		setup: func(m *h3Mock, ctx context.Context) {
			msg := h3WebMsg()
			msg.TMDelete = timePtr("2024-01-01T00:00:00.000000Z")
			m.EXPECT().WebchatV1MessageGet(ctx, h3ResourceID).Return(msg, nil)
		},
	})
	h3RunRes(t, rows)
}

func Test_WebchatMessageDelete_guards_h3(t *testing.T) {
	lookup := func(m *h3Mock, ctx context.Context) {
		m.EXPECT().WebchatV1MessageGet(ctx, h3ResourceID).Return(h3WebMsg(), nil)
	}
	h3RunRes(t, h3GuardRows(h3Ops[*wcmessage.WebhookMessage]{
		call: func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*wcmessage.WebhookMessage, error) {
			return h.WebchatMessageDelete(ctx, a, h3ResourceID)
		},
		a:      h3Admin(),
		denied: h3OwnedDenied(lookup),
		getFail: func(m *h3Mock, ctx context.Context) {
			m.EXPECT().WebchatV1MessageGet(ctx, h3ResourceID).Return(nil, errH3)
		},
		opPre: lookup,
		opFail: func(m *h3Mock, ctx context.Context) {
			m.EXPECT().WebchatV1MessageDelete(ctx, h3ResourceID).Return(nil, errH3)
		},
		opOK: func(m *h3Mock, ctx context.Context) {
			m.EXPECT().WebchatV1MessageDelete(ctx, h3ResourceID).Return(h3WebMsg(), nil)
		},
		want: h3WebMsg().ConvertWebhookMessage(),
	}))
}

func Test_WebchatMessageList_guards_h3(t *testing.T) {
	filtersAdmin := func(withSession bool) map[wcmessage.Field]any {
		f := map[wcmessage.Field]any{
			wcmessage.FieldDeleted:    false,
			wcmessage.FieldCustomerID: h3CustomerID,
		}
		if withSession {
			f[wcmessage.FieldSessionID] = h3WebSessionID
		}
		return f
	}
	listAdmin := func(sid uuid.UUID) func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) ([]*wcmessage.WebhookMessage, error) {
		return func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) ([]*wcmessage.WebhookMessage, error) {
			return h.WebchatMessageList(ctx, a, 10, "tok", sid)
		}
	}
	directFilters := map[wcmessage.Field]any{
		wcmessage.FieldDeleted:    false,
		wcmessage.FieldCustomerID: h3CustomerID,
		wcmessage.FieldSessionID:  h3WebSessionID,
	}
	direct := h3WebDirect([]string{"webchat_session"}, h3WebSessionID)

	h3RunRes(t, []h3ResRow[[]*wcmessage.WebhookMessage]{
		{name: "no permission", a: h3NoPerm(), call: listAdmin(uuid.Nil), wantErr: h3ErrDenied},
		{
			name: "session lookup error", a: h3Admin(), call: listAdmin(h3WebSessionID), wantErr: errH3,
			setup: func(m *h3Mock, ctx context.Context) { h3WebSessionRPC(m, ctx, nil, errH3) },
		},
		{
			name: "session from another customer", a: h3Admin(), call: listAdmin(h3WebSessionID), wantErr: h3ErrDenied,
			setup: func(m *h3Mock, ctx context.Context) {
				h3WebSessionRPC(m, ctx, h3WebSession(wcsession.StatusActive, h3OtherCustomerID, h3WebWidgetID), nil)
			},
		},
		{
			name: "list rpc error", a: h3Admin(), call: listAdmin(uuid.Nil), wantErr: errH3,
			setup: func(m *h3Mock, ctx context.Context) {
				m.EXPECT().WebchatV1MessageList(ctx, "tok", uint64(10), filtersAdmin(false)).Return(nil, errH3)
			},
		},
		{
			name: "success with session filter", a: h3Admin(), call: listAdmin(h3WebSessionID),
			want: []*wcmessage.WebhookMessage{h3WebMsg().ConvertWebhookMessage()},
			setup: func(m *h3Mock, ctx context.Context) {
				h3WebSessionRPC(m, ctx, h3WebSession(wcsession.StatusActive, h3CustomerID, h3WebWidgetID), nil)
				m.EXPECT().WebchatV1MessageList(ctx, "tok", uint64(10), filtersAdmin(true)).Return([]*wcmessage.Message{h3WebMsg()}, nil)
			},
		},
		{
			name: "direct token without the session resource type", a: h3WebDirect([]string{"other"}, h3WebSessionID),
			call: listAdmin(h3WebSessionID), wantErr: h3ErrDenied,
		},
		{
			name: "direct session lookup error", a: direct, call: listAdmin(uuid.Nil), wantErr: errH3,
			setup: func(m *h3Mock, ctx context.Context) { h3WebSessionRPC(m, ctx, nil, errH3) },
		},
		{
			name: "direct widget lookup error", a: direct, call: listAdmin(uuid.Nil), wantErr: errH3,
			setup: func(m *h3Mock, ctx context.Context) {
				h3WebSessionRPC(m, ctx, h3WebSession(wcsession.StatusActive, h3CustomerID, h3WebWidgetID), nil)
				h3WebWidgetRPC(m, ctx, errH3)
			},
		},
		{
			name: "direct list rpc error", a: direct, call: listAdmin(uuid.Nil), wantErr: errH3,
			setup: func(m *h3Mock, ctx context.Context) {
				h3WebSessionRPC(m, ctx, h3WebSession(wcsession.StatusActive, h3CustomerID, h3WebWidgetID), nil)
				h3WebWidgetRPC(m, ctx, nil)
				m.EXPECT().WebchatV1MessageList(ctx, "tok", uint64(10), directFilters).Return(nil, errH3)
			},
		},
	})
}

func Test_WebchatMessageList_emptyToken_h3(t *testing.T) {
	mc := h3NewController(t)
	h, mockReq, mockUtil := h3NewHandler(mc)
	ctx := context.Background()

	mockUtil.EXPECT().TimeGetCurTime().Return("2024-04-04 00:00:00.000000")
	mockReq.EXPECT().WebchatV1MessageList(ctx, "2024-04-04 00:00:00.000000", uint64(5), map[wcmessage.Field]any{
		wcmessage.FieldDeleted:    false,
		wcmessage.FieldCustomerID: h3CustomerID,
	}).Return([]*wcmessage.Message{}, nil)

	res, err := h.WebchatMessageList(ctx, h3Admin(), 5, "", uuid.Nil)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res == nil || len(res) != 0 {
		t.Errorf("Wrong match. expect: empty non-nil slice, got: %v", res)
	}
}

func Test_WebchatMessageCreate_h3(t *testing.T) {
	call := func(a *auth.AuthIdentity) func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*wcmessage.WebhookMessage, error) {
		return func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*wcmessage.WebhookMessage, error) {
			// the requested direction is never trusted: pass the opposite of what each path derives.
			return h.WebchatMessageCreate(ctx, a, h3WebSessionID, wcmessage.DirectionInbound, "hi")
		}
	}
	active := h3WebSession(wcsession.StatusActive, h3CustomerID, h3WebWidgetID)
	ended := h3WebSession(wcsession.StatusEnded, h3CustomerID, h3WebWidgetID)
	otherWidget := h3WebSession(wcsession.StatusActive, h3CustomerID, uuid.FromStringOrNil("dd847807-6cc4-4713-9dec-53a42840e74c"))
	created := h3WebMsg()
	want := created.ConvertWebhookMessage()
	accesskey := auth.NewAccesskeyIdentity(&csaccesskey.Accesskey{ID: h3WebAccesskeyID, CustomerID: h3CustomerID})
	direct := h3WebDirect([]string{"webchat_session"}, h3WebSessionID)
	create := func(m *h3Mock, ctx context.Context, dir wcmessage.Direction, sender uuid.UUID) *h3GomockCall {
		return m.EXPECT().WebchatV1MessageCreate(ctx, h3CustomerID, h3WebSessionID, dir, sender, "hi")
	}

	h3RunRes(t, []h3ResRow[*wcmessage.WebhookMessage]{
		{
			name: "agent: session lookup error", a: h3Admin(), call: call(nil), wantErr: errH3,
			setup: func(m *h3Mock, ctx context.Context) { h3WebSessionRPC(m, ctx, nil, errH3) },
		},
		{
			name: "agent: cross tenant", a: h3Agent(h3OtherCustomerID, amagent.PermissionCustomerAdmin), call: call(nil), wantErr: h3ErrDenied,
			setup: func(m *h3Mock, ctx context.Context) { h3WebSessionRPC(m, ctx, active, nil) },
		},
		{
			name: "agent: no permission", a: h3NoPerm(), call: call(nil), wantErr: h3ErrDenied,
			setup: func(m *h3Mock, ctx context.Context) { h3WebSessionRPC(m, ctx, active, nil) },
		},
		{
			name: "agent: ended session", a: h3Admin(), call: call(nil), wantErr: serviceerrors.ErrNotFound,
			setup: func(m *h3Mock, ctx context.Context) { h3WebSessionRPC(m, ctx, ended, nil) },
		},
		{
			name: "agent: create rpc error", a: h3Admin(), call: call(nil), wantErr: errH3,
			setup: func(m *h3Mock, ctx context.Context) {
				h3WebSessionRPC(m, ctx, active, nil)
				create(m, ctx, wcmessage.DirectionOutbound, h3AgentID).Return(nil, errH3)
			},
		},
		{
			name: "agent: outbound with the agent as sender", a: h3Admin(), call: call(nil), want: want,
			setup: func(m *h3Mock, ctx context.Context) {
				h3WebSessionRPC(m, ctx, active, nil)
				create(m, ctx, wcmessage.DirectionOutbound, h3AgentID).Return(created, nil)
			},
		},
		{
			name: "accesskey: outbound with the accesskey as sender", a: accesskey, call: call(nil), want: want,
			setup: func(m *h3Mock, ctx context.Context) {
				h3WebSessionRPC(m, ctx, active, nil)
				create(m, ctx, wcmessage.DirectionOutbound, h3WebAccesskeyID).Return(created, nil)
			},
		},
		{
			name: "direct: token without the session resource type", a: h3WebDirect([]string{"other"}, h3WebSessionID), call: call(nil), wantErr: h3ErrDenied,
		},
		{
			name: "direct: unbound token", a: h3WebDirect([]string{"webchat_session"}, uuid.Nil), call: call(nil), wantErr: h3ErrDenied,
		},
		{
			name: "direct: session lookup error", a: direct, call: call(nil), wantErr: errH3,
			setup: func(m *h3Mock, ctx context.Context) { h3WebSessionRPC(m, ctx, nil, errH3) },
		},
		{
			name: "direct: session of another widget", a: direct, call: call(nil), wantErr: h3ErrDenied,
			setup: func(m *h3Mock, ctx context.Context) { h3WebSessionRPC(m, ctx, otherWidget, nil) },
		},
		{
			name: "direct: ended session", a: direct, call: call(nil), wantErr: serviceerrors.ErrNotFound,
			setup: func(m *h3Mock, ctx context.Context) { h3WebSessionRPC(m, ctx, ended, nil) },
		},
		{
			name: "direct: widget lookup error", a: direct, call: call(nil), wantErr: errH3,
			setup: func(m *h3Mock, ctx context.Context) {
				h3WebSessionRPC(m, ctx, active, nil)
				h3WebWidgetRPC(m, ctx, errH3)
			},
		},
		{
			name: "direct: inbound without a sender", a: direct, call: call(nil), want: want,
			setup: func(m *h3Mock, ctx context.Context) {
				h3WebSessionRPC(m, ctx, active, nil)
				h3WebWidgetRPC(m, ctx, nil)
				create(m, ctx, wcmessage.DirectionInbound, uuid.Nil).Return(created, nil)
			},
		},
		{
			name: "delegate identity is denied", a: auth.NewDelegateIdentity(&auth.DelegateScope{CustomerID: h3CustomerID, IssuedBy: h3AgentID, JTI: "h3-jti"}),
			call: call(nil), wantErr: h3ErrDenied,
		},
	})
}
