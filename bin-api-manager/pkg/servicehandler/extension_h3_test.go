package servicehandler

import (
	"context"
	amagent "monorepo/bin-agent-manager/models/agent"
	"regexp"
	"testing"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/cachehandler"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	rmextension "monorepo/bin-registrar-manager/models/extension"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func h3Extension() *rmextension.Extension {
	return &rmextension.Extension{
		Identity:   commonidentity.Identity{ID: h3ResourceID, CustomerID: h3CustomerID},
		Extension:  "1001",
		DomainName: "h3.registrar.example.com",
	}
}

func h3ExtLookup(m *requesthandler.MockRequestHandler, ctx context.Context) {
	m.EXPECT().RegistrarV1ExtensionGet(ctx, h3ResourceID).Return(h3Extension(), nil)
}

func h3ExtLookupFail(m *requesthandler.MockRequestHandler, ctx context.Context) {
	m.EXPECT().RegistrarV1ExtensionGet(ctx, h3ResourceID).Return(nil, errH3)
}

func Test_ExtensionOps_guards_h3(t *testing.T) {
	pw := "secret-h3"
	cases := []struct {
		name   string
		call   func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*rmextension.WebhookMessage, error)
		opFail func(m *requesthandler.MockRequestHandler, ctx context.Context)
		opOK   func(m *requesthandler.MockRequestHandler, ctx context.Context)
	}{
		{
			"ExtensionDelete",
			func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*rmextension.WebhookMessage, error) {
				return h.ExtensionDelete(ctx, a, h3ResourceID)
			},
			func(m *requesthandler.MockRequestHandler, ctx context.Context) {
				m.EXPECT().RegistrarV1ExtensionDelete(ctx, h3ResourceID).Return(nil, errH3)
			},
			func(m *requesthandler.MockRequestHandler, ctx context.Context) {
				m.EXPECT().RegistrarV1ExtensionDelete(ctx, h3ResourceID).Return(h3Extension(), nil)
			},
		},
		{
			"ExtensionUpdate",
			func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*rmextension.WebhookMessage, error) {
				return h.ExtensionUpdate(ctx, a, h3ResourceID, nil, nil, &pw)
			},
			func(m *requesthandler.MockRequestHandler, ctx context.Context) {
				m.EXPECT().RegistrarV1ExtensionUpdate(ctx, h3ResourceID, nil, nil, &pw).Return(nil, errH3)
			},
			func(m *requesthandler.MockRequestHandler, ctx context.Context) {
				m.EXPECT().RegistrarV1ExtensionUpdate(ctx, h3ResourceID, nil, nil, &pw).Return(h3Extension(), nil)
			},
		},
		{
			"ExtensionDirectHashRegenerate",
			func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*rmextension.WebhookMessage, error) {
				return h.ExtensionDirectHashRegenerate(ctx, a, h3ResourceID)
			},
			func(m *requesthandler.MockRequestHandler, ctx context.Context) {
				m.EXPECT().RegistrarV1ExtensionDirectHashRegenerate(ctx, h3ResourceID).Return(nil, errH3)
			},
			func(m *requesthandler.MockRequestHandler, ctx context.Context) {
				m.EXPECT().RegistrarV1ExtensionDirectHashRegenerate(ctx, h3ResourceID).Return(h3Extension(), nil)
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h3RunRes(t, h3GuardRows(h3Ops[*rmextension.WebhookMessage]{
				call:    c.call,
				a:       h3Admin(),
				denied:  h3OwnedDenied(h3ExtLookup),
				getFail: h3ExtLookupFail,
				opPre:   h3ExtLookup,
				opFail:  c.opFail,
				opOK:    c.opOK,
				want:    h3Extension().ConvertWebhookMessage(),
			}))
		})
	}
}

func Test_ExtensionGet_guards_h3(t *testing.T) {
	h3RunRes(t, h3GuardRows(h3Ops[*rmextension.WebhookMessage]{
		call: func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*rmextension.WebhookMessage, error) {
			return h.ExtensionGet(ctx, a, h3ResourceID)
		},
		a:      h3Admin(),
		denied: h3OwnedDenied(h3ExtLookup),
		opFail: h3ExtLookupFail,
		opOK:   h3ExtLookup,
		want:   h3Extension().ConvertWebhookMessage(),
	}))
}

func Test_ExtensionCreate_guards_h3(t *testing.T) {
	h3RunRes(t, h3GuardRows(h3Ops[*rmextension.WebhookMessage]{
		call: func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*rmextension.WebhookMessage, error) {
			return h.ExtensionCreate(ctx, a, "1001", "pw-h3", "name", "detail")
		},
		a:      h3Admin(),
		denied: h3NoPermOnly(),
		opFail: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
			m.EXPECT().RegistrarV1ExtensionCreate(ctx, h3CustomerID, "1001", "pw-h3", "name", "detail").Return(nil, errH3)
		},
		opOK: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
			m.EXPECT().RegistrarV1ExtensionCreate(ctx, h3CustomerID, "1001", "pw-h3", "name", "detail").Return(h3Extension(), nil)
		},
		want: h3Extension().ConvertWebhookMessage(),
	}))
}

func Test_ExtensionList_guards_h3(t *testing.T) {
	filters := map[rmextension.Field]any{
		rmextension.FieldCustomerID: h3CustomerID,
		rmextension.FieldDeleted:    false,
	}
	h3RunRes(t, h3GuardRows(h3Ops[[]*rmextension.WebhookMessage]{
		call: func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) ([]*rmextension.WebhookMessage, error) {
			return h.ExtensionList(ctx, a, 10, "tok")
		},
		a:      h3Admin(),
		denied: h3NoPermOnly(),
		opFail: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
			m.EXPECT().RegistrarV1ExtensionList(ctx, "tok", uint64(10), filters).Return(nil, errH3)
		},
		opOK: func(m *requesthandler.MockRequestHandler, ctx context.Context) {
			m.EXPECT().RegistrarV1ExtensionList(ctx, "tok", uint64(10), filters).Return([]rmextension.Extension{*h3Extension()}, nil)
		},
		want: []*rmextension.WebhookMessage{h3Extension().ConvertWebhookMessage()},
	}))
}

func Test_ExtensionProvisioningTokenCreate_guards_h3(t *testing.T) {
	call := func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*ExtensionProvisioningToken, error) {
		return h.ExtensionProvisioningTokenCreate(ctx, a, h3ResourceID)
	}
	h3RunRes(t, []h3ResRow[*ExtensionProvisioningToken]{
		{name: "direct", a: h3Direct(), call: call, wantErr: h3ErrDirect},
		{name: "lookup error", a: h3Admin(), call: call, wantErr: errH3, setup: h3ExtLookupFail},
		{name: "cross tenant", a: h3Agent(h3OtherCustomerID, amagent.PermissionCustomerAdmin), call: call, wantErr: h3ErrDenied, setup: h3ExtLookup},
		{name: "no permission", a: h3NoPerm(), call: call, wantErr: h3ErrDenied, setup: h3ExtLookup},
	})
}

func Test_ExtensionProvisioningTokenCreate_cacheError_h3(t *testing.T) {
	mc := h3NewController(t)
	h, mockReq, _ := h3NewHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h.cacheHandler = mockCache
	ctx := context.Background()

	h3ExtLookup(mockReq, ctx)
	mockCache.EXPECT().ProvisioningTokenSet(ctx, gomock.Cond(h3IsHexToken), h3ResourceID, ProvisioningTokenTTL).Return(errH3)

	res, err := h.ExtensionProvisioningTokenCreate(ctx, h3Admin(), h3ResourceID)
	if !h3ErrorsIs(err, errH3) {
		t.Errorf("Wrong match. expect: %v, got: %v", errH3, err)
	}
	if res != nil {
		t.Errorf("Wrong match. expect nil result, got: %v", res)
	}
}

func Test_ExtensionProvisioningXMLGet_errors_h3(t *testing.T) {
	deleted := timePtr("2024-01-01T00:00:00.000000Z")
	tests := []struct {
		name    string
		setup   func(m *requesthandler.MockRequestHandler, c *cachehandler.MockCacheHandler, ctx context.Context)
		wantErr error
	}{
		{
			name: "token cannot be resolved",
			setup: func(m *requesthandler.MockRequestHandler, c *cachehandler.MockCacheHandler, ctx context.Context) {
				c.EXPECT().ProvisioningTokenGet(ctx, "tok").Return(uuid.Nil, errH3)
			},
			wantErr: errH3,
		},
		{
			name: "extension lookup fails",
			setup: func(m *requesthandler.MockRequestHandler, c *cachehandler.MockCacheHandler, ctx context.Context) {
				c.EXPECT().ProvisioningTokenGet(ctx, "tok").Return(h3ResourceID, nil)
				h3ExtLookupFail(m, ctx)
			},
			wantErr: errH3,
		},
		{
			name: "deleted extension is not found",
			setup: func(m *requesthandler.MockRequestHandler, c *cachehandler.MockCacheHandler, ctx context.Context) {
				c.EXPECT().ProvisioningTokenGet(ctx, "tok").Return(h3ResourceID, nil)
				ext := h3Extension()
				ext.TMDelete = deleted
				m.EXPECT().RegistrarV1ExtensionGet(ctx, h3ResourceID).Return(ext, nil)
			},
			wantErr: serviceerrors.ErrNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := h3NewController(t)
			h, mockReq, _ := h3NewHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h.cacheHandler = mockCache
			ctx := context.Background()
			tt.setup(mockReq, mockCache, ctx)

			res, err := h.ExtensionProvisioningXMLGet(ctx, "tok")
			if !h3ErrorsIs(err, tt.wantErr) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.wantErr, err)
			}
			if res != nil {
				t.Errorf("Wrong match. expect nil body, got: %s", res)
			}
		})
	}
}

func Test_convertExtensionFilters_h3(t *testing.T) {
	mc := h3NewController(t)
	h, _, _ := h3NewHandler(mc)

	if _, err := h.convertExtensionFilters(map[string]string{"customer_id": "not-a-uuid"}); err == nil {
		t.Errorf("Wrong match. expect: error for invalid customer_id, got: nil")
	}
	res, err := h.convertExtensionFilters(map[string]string{"customer_id": h3CustomerID.String()})
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	want := map[rmextension.Field]any{rmextension.FieldCustomerID: h3CustomerID}
	if !h3DeepEqual(res, want) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", want, res)
	}
}

// h3HexToken matches the 32 random bytes of a provisioning token rendered as hex.
var h3HexToken = regexp.MustCompile(`^[0-9a-f]{64}$`)

func h3IsHexToken(x any) bool {
	token, ok := x.(string)
	return ok && h3HexToken.MatchString(token)
}
