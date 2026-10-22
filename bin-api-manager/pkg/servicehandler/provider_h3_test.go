package servicehandler

import (
	"context"
	"errors"
	"strings"
	"testing"

	"monorepo/bin-api-manager/models/auth"
	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/pkg/requesthandler"
	rmprovider "monorepo/bin-route-manager/models/provider"
)

func h3Provider() *rmprovider.Provider {
	return &rmprovider.Provider{
		ID:       h3ResourceID,
		Type:     rmprovider.TypeSIP,
		Hostname: "sip.example.com",
		Name:     "h3 provider",
	}
}

func h3ProviderLookup(m *requesthandler.MockRequestHandler, ctx context.Context) {
	m.EXPECT().RouteV1ProviderGet(ctx, h3ResourceID).Return(h3Provider(), nil)
}

func h3ProviderLookupFail(m *requesthandler.MockRequestHandler, ctx context.Context) {
	m.EXPECT().RouteV1ProviderGet(ctx, h3ResourceID).Return(nil, errH3)
}

func Test_ProviderGet_guards_h3(t *testing.T) {
	// the lookup happens before the permission check, so denied rows reach the lookup.
	h3RunRes(t, h3GuardRows(h3Ops[*rmprovider.WebhookMessage]{
		call: func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*rmprovider.WebhookMessage, error) {
			return h.ProviderGet(ctx, a, h3ResourceID)
		},
		a:      h3Super(),
		denied: h3SuperDenied(h3ProviderLookup),
		opFail: h3ProviderLookupFail,
		opOK:   h3ProviderLookup,
		want:   h3Provider().ConvertWebhookMessage(),
	}))
}

func Test_ProviderList_guards_h3(t *testing.T) {
	h3RunRes(t, h3GuardRows(h3Ops[[]*rmprovider.WebhookMessage]{
		call: func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) ([]*rmprovider.WebhookMessage, error) {
			return h.ProviderList(ctx, a, 10, "tok")
		},
		a:      h3Super(),
		denied: h3SuperDenied(nil),
		opFail: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
			m.EXPECT().RouteV1ProviderList(ctx, "tok", uint64(10)).Return(nil, errH3)
		},
		opOK: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
			m.EXPECT().RouteV1ProviderList(ctx, "tok", uint64(10)).Return([]rmprovider.Provider{*h3Provider()}, nil)
		},
		want: []*rmprovider.WebhookMessage{h3Provider().ConvertWebhookMessage()},
	}))
}

func Test_ProviderCreate_guards_h3(t *testing.T) {
	headers := map[string]string{"X-H3": "1"}
	create := func(m *requesthandler.MockRequestHandler, ctx context.Context) *h3GomockCall {
		return m.EXPECT().RouteV1ProviderCreate(ctx, rmprovider.Type(rmprovider.TypeSIP), "sip.example.com", "pre", "post", headers, "n", "d", "ulaw")
	}
	h3RunRes(t, h3GuardRows(h3Ops[*rmprovider.WebhookMessage]{
		call: func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*rmprovider.WebhookMessage, error) {
			return h.ProviderCreate(ctx, a, rmprovider.Type(rmprovider.TypeSIP), "sip.example.com", "pre", "post", headers, "n", "d", "ulaw")
		},
		a:      h3Super(),
		denied: h3SuperDenied(nil),
		opFail: func(m *requesthandler.MockRequestHandler, ctx context.Context) { create(m, ctx).Return(nil, errH3) },
		opOK: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
			create(m, ctx).Return(h3Provider(), nil)
		},
		want: h3Provider().ConvertWebhookMessage(),
	}))
}

func Test_ProviderDelete_guards_h3(t *testing.T) {
	// permission is checked before any lookup: denied rows must not issue rpcs.
	h3RunRes(t, h3GuardRows(h3Ops[*rmprovider.WebhookMessage]{
		call: func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*rmprovider.WebhookMessage, error) {
			return h.ProviderDelete(ctx, a, h3ResourceID)
		},
		a:       h3Super(),
		denied:  h3SuperDenied(nil),
		getFail: h3ProviderLookupFail,
		opPre:   h3ProviderLookup,
		opFail: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
			m.EXPECT().RouteV1ProviderDelete(ctx, h3ResourceID).Return(nil, errH3)
		},
		opOK: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
			m.EXPECT().RouteV1ProviderDelete(ctx, h3ResourceID).Return(h3Provider(), nil)
		},
		want: h3Provider().ConvertWebhookMessage(),
	}))
}

func Test_ProviderUpdate_guards_h3(t *testing.T) {
	name := "renamed"
	h3RunRes(t, h3GuardRows(h3Ops[*rmprovider.WebhookMessage]{
		call: func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*rmprovider.WebhookMessage, error) {
			return h.ProviderUpdate(ctx, a, h3ResourceID, nil, nil, nil, nil, nil, &name, nil, nil)
		},
		a:       h3Super(),
		denied:  h3SuperDenied(nil),
		getFail: h3ProviderLookupFail,
		opPre:   h3ProviderLookup,
		opFail: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
			m.EXPECT().RouteV1ProviderUpdate(ctx, h3ResourceID, nil, nil, nil, nil, nil, &name, nil, nil).Return(nil, errH3)
		},
		opOK: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
			m.EXPECT().RouteV1ProviderUpdate(ctx, h3ResourceID, nil, nil, nil, nil, nil, &name, nil, nil).Return(h3Provider(), nil)
		},
		want: h3Provider().ConvertWebhookMessage(),
	}))
}

func Test_ProviderSetup_h3(t *testing.T) {
	call := func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*rmprovider.WebhookMessage, error) {
		return h.ProviderSetup(ctx, a, "carrier-h3", "n", "d", "key-h3")
	}
	setup := func(m *requesthandler.MockRequestHandler, ctx context.Context) *h3GomockCall {
		return m.EXPECT().RouteV1ProviderSetup(ctx, "carrier-h3", "n", "d", "key-h3")
	}
	ve := &cerrors.VoipbinError{Status: cerrors.StatusInvalidArgument, Reason: "h3", Message: "bad carrier"}

	tests := []struct {
		name    string
		a       *auth.AuthIdentity
		setup   func(m *requesthandler.MockRequestHandler, ctx context.Context)
		wantErr error
		wantRes *rmprovider.WebhookMessage
		prefix  string
	}{
		{name: "direct", a: h3Direct(), wantErr: h3ErrDirect},
		{name: "customer admin is denied", a: h3Admin(), wantErr: h3ErrDenied},
		{
			name: "voipbin error passes through unwrapped", a: h3Super(), wantErr: ve,
			setup: func(m *requesthandler.MockRequestHandler, ctx context.Context) { setup(m, ctx).Return(nil, ve) },
		},
		{
			name: "unprocessable entity passes through", a: h3Super(), wantErr: requesthandler.ErrUnprocessableEntity,
			setup: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
				setup(m, ctx).Return(nil, requesthandler.ErrUnprocessableEntity)
			},
		},
		{
			name: "other error is wrapped", a: h3Super(), wantErr: errH3, prefix: "provider setup failed: ",
			setup: func(m *requesthandler.MockRequestHandler, ctx context.Context) { setup(m, ctx).Return(nil, errH3) },
		},
		{
			name: "success", a: h3Super(), wantRes: h3Provider().ConvertWebhookMessage(),
			setup: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
				setup(m, ctx).Return(h3Provider(), nil)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := h3NewController(t)
			h, mockReq, _ := h3NewHandler(mc)
			ctx := context.Background()
			if tt.setup != nil {
				tt.setup(mockReq, ctx)
			}
			res, err := call(h, ctx, tt.a)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Wrong match. expect: %v, got: %v", tt.wantErr, err)
				}
				if tt.prefix != "" && !strings.HasPrefix(err.Error(), tt.prefix) {
					t.Errorf("Wrong match. expect prefix %q, got: %v", tt.prefix, err)
				}
				if tt.prefix == "" && tt.wantErr != h3ErrDirect && tt.wantErr != h3ErrDenied && err != tt.wantErr {
					t.Errorf("Wrong match. expect the error returned as-is: %v, got: %v", tt.wantErr, err)
				}
				if res != nil {
					t.Errorf("Wrong match. expect nil result, got: %v", res)
				}
				return
			}
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if !h3DeepEqual(res, tt.wantRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.wantRes, res)
			}
		})
	}
}
