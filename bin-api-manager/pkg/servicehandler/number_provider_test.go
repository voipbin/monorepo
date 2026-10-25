package servicehandler

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	csaccesskey "monorepo/bin-customer-manager/models/accesskey"
	nmnumber "monorepo/bin-number-manager/models/number"
	nmprovider "monorepo/bin-number-manager/models/provider"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

const (
	testProviderActor  = "550e8400-e29b-41d4-a716-446655440000"
	testProviderSecret = "test-secret-value-9f3a"
)

func testProviderSuperAdmin() *auth.AuthIdentity {
	return auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil(testProviderActor),
			CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
		},
		Permission: amagent.PermissionProjectSuperAdmin,
	})
}

// testProviderDenied returns the identities that must be refused, with the error each must get.
func testProviderDenied() map[string]struct {
	a   *auth.AuthIdentity
	err error
} {
	customer := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	return map[string]struct {
		a   *auth.AuthIdentity
		err error
	}{
		"customer admin agent": {
			auth.NewAgentIdentity(&amagent.Agent{
				Identity:   commonidentity.Identity{ID: uuid.FromStringOrNil(testProviderActor), CustomerID: customer},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			serviceerrors.ErrPermissionDenied,
		},
		"accesskey": {
			auth.NewAccesskeyIdentity(&csaccesskey.Accesskey{ID: uuid.FromStringOrNil("b3e2a1c0-0000-0000-0000-0000000000aa"), CustomerID: customer}),
			serviceerrors.ErrPermissionDenied,
		},
		"delegate": {
			auth.NewDelegateIdentity(&auth.DelegateScope{CustomerID: customer}),
			serviceerrors.ErrPermissionDenied,
		},
		"direct": {
			auth.NewDirectIdentity(&auth.DirectScope{CustomerID: customer}),
			serviceerrors.ErrDirectAccessNotSupported,
		},
	}
}

func testProviderInfo() *nmprovider.Info {
	actor := uuid.FromStringOrNil(testProviderActor)
	return &nmprovider.Info{
		ProviderName:     "telnyx",
		Enabled:          true,
		Priority:         10,
		Config:           map[string]string{"connection_id": "test-connection"},
		HasSecret:        true,
		KeyVersion:       1,
		CredentialSource: nmprovider.CredentialSourceDB,
		Usable:           true,
		SecretUpdatedBy:  &actor,
	}
}

func Test_NumberProviderList(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &serviceHandler{reqHandler: mockReq}
	ctx := context.Background()

	mockReq.EXPECT().NumberV1NumberProviderList(ctx).Return([]*nmprovider.Info{testProviderInfo()}, nil)

	res, err := h.NumberProviderList(ctx, testProviderSuperAdmin())
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	expect := []*nmprovider.WebhookMessage{testProviderInfo().ConvertWebhookMessage()}
	if !reflect.DeepEqual(expect, res) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expect, res)
	}
}

func Test_NumberProviderList_rpcError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &serviceHandler{reqHandler: mockReq}
	ctx := context.Background()

	mockReq.EXPECT().NumberV1NumberProviderList(ctx).Return(nil, requesthandler.ErrInternal)

	if _, err := h.NumberProviderList(ctx, testProviderSuperAdmin()); !errors.Is(err, requesthandler.ErrInternal) {
		t.Errorf("Wrong match. expect: %v, got: %v", requesthandler.ErrInternal, err)
	}
}

func Test_NumberProviderGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &serviceHandler{reqHandler: mockReq}
	ctx := context.Background()

	mockReq.EXPECT().NumberV1NumberProviderGet(ctx, nmnumber.ProviderName("telnyx")).Return(testProviderInfo(), nil)

	res, err := h.NumberProviderGet(ctx, testProviderSuperAdmin(), "telnyx")
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	if !reflect.DeepEqual(testProviderInfo().ConvertWebhookMessage(), res) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", testProviderInfo().ConvertWebhookMessage(), res)
	}
}

func Test_NumberProviderGet_notFound(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &serviceHandler{reqHandler: mockReq}
	ctx := context.Background()

	mockReq.EXPECT().NumberV1NumberProviderGet(ctx, gomock.Any()).Return(nil, requesthandler.ErrNotFound)

	if _, err := h.NumberProviderGet(ctx, testProviderSuperAdmin(), "unknown"); !errors.Is(err, requesthandler.ErrNotFound) {
		t.Errorf("Wrong match. expect: %v, got: %v", requesthandler.ErrNotFound, err)
	}
}

func Test_NumberProviderUpdate(t *testing.T) {
	disabled := false
	zero := 0
	config := map[string]string{"connection_id": "test-connection", "profile_id": ""}

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &serviceHandler{reqHandler: mockReq}
	ctx := context.Background()

	// the agent id is passed as the actor, and false and 0 reach the rpc as they are
	mockReq.EXPECT().NumberV1NumberProviderUpdate(ctx, nmnumber.ProviderName("telnyx"), uuid.FromStringOrNil(testProviderActor), &disabled, &zero, config).Return(testProviderInfo(), nil)

	res, err := h.NumberProviderUpdate(ctx, testProviderSuperAdmin(), "telnyx", &disabled, &zero, config)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	if !reflect.DeepEqual(testProviderInfo().ConvertWebhookMessage(), res) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", testProviderInfo().ConvertWebhookMessage(), res)
	}
}

func Test_NumberProviderUpdateSecret(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &serviceHandler{reqHandler: mockReq}
	ctx := context.Background()

	mockReq.EXPECT().NumberV1NumberProviderUpdateSecret(ctx, nmnumber.ProviderName("telnyx"), uuid.FromStringOrNil(testProviderActor), testProviderSecret).Return(testProviderInfo(), nil)

	res, err := h.NumberProviderUpdateSecret(ctx, testProviderSuperAdmin(), "telnyx", testProviderSecret)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	if !reflect.DeepEqual(testProviderInfo().ConvertWebhookMessage(), res) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", testProviderInfo().ConvertWebhookMessage(), res)
	}
}

func Test_NumberProviderDeleteSecret(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &serviceHandler{reqHandler: mockReq}
	ctx := context.Background()

	mockReq.EXPECT().NumberV1NumberProviderDeleteSecret(ctx, nmnumber.ProviderName("telnyx"), uuid.FromStringOrNil(testProviderActor)).Return(testProviderInfo(), nil)

	res, err := h.NumberProviderDeleteSecret(ctx, testProviderSuperAdmin(), "telnyx")
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	if !reflect.DeepEqual(testProviderInfo().ConvertWebhookMessage(), res) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", testProviderInfo().ConvertWebhookMessage(), res)
	}
}

// every method refuses the identities that are not the project super admin before any rpc,
// and a bad provider name is refused after the permission check, also before any rpc
func Test_NumberProvider_authorization(t *testing.T) {
	calls := map[string]func(h *serviceHandler, a *auth.AuthIdentity, name string) error{
		"list": func(h *serviceHandler, a *auth.AuthIdentity, name string) error {
			_, err := h.NumberProviderList(context.Background(), a)
			return err
		},
		"get": func(h *serviceHandler, a *auth.AuthIdentity, name string) error {
			_, err := h.NumberProviderGet(context.Background(), a, name)
			return err
		},
		"update": func(h *serviceHandler, a *auth.AuthIdentity, name string) error {
			_, err := h.NumberProviderUpdate(context.Background(), a, name, nil, nil, nil)
			return err
		},
		"update secret": func(h *serviceHandler, a *auth.AuthIdentity, name string) error {
			_, err := h.NumberProviderUpdateSecret(context.Background(), a, name, testProviderSecret)
			return err
		},
		"delete secret": func(h *serviceHandler, a *auth.AuthIdentity, name string) error {
			_, err := h.NumberProviderDeleteSecret(context.Background(), a, name)
			return err
		},
	}

	for method, call := range calls {
		for who, tt := range testProviderDenied() {
			t.Run(method+" "+who, func(t *testing.T) {
				mc := gomock.NewController(t)
				defer mc.Finish()

				// no rpc is expected: the mock fails the test on any call
				h := &serviceHandler{reqHandler: requesthandler.NewMockRequestHandler(mc)}

				if err := call(h, tt.a, "telnyx"); !errors.Is(err, tt.err) {
					t.Errorf("Wrong match. expect: %v, got: %v", tt.err, err)
				}
			})
		}

		if method == "list" {
			continue
		}

		for _, name := range []string{"", "Telnyx", "a/b", "a..b", "telnyx/secret", "x y", "a123456789012345678901234567890123"} {
			t.Run(method+" bad name "+name, func(t *testing.T) {
				mc := gomock.NewController(t)
				defer mc.Finish()

				h := &serviceHandler{reqHandler: requesthandler.NewMockRequestHandler(mc)}

				if err := call(h, testProviderSuperAdmin(), name); !errors.Is(err, serviceerrors.ErrInvalidArgument) {
					t.Errorf("Wrong match. expect: %v, got: %v", serviceerrors.ErrInvalidArgument, err)
				}
			})
		}

		// the permission is checked first, so a bad name does not tell a non admin anything
		t.Run(method+" bad name and not allowed", func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h := &serviceHandler{reqHandler: requesthandler.NewMockRequestHandler(mc)}

			if err := call(h, testProviderDenied()["accesskey"].a, "a/b"); !errors.Is(err, serviceerrors.ErrPermissionDenied) {
				t.Errorf("Wrong match. expect: %v, got: %v", serviceerrors.ErrPermissionDenied, err)
			}
		})
	}

	// the name boundary: the longest valid name is accepted
	t.Run("name of 32 characters is accepted", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockReq := requesthandler.NewMockRequestHandler(mc)
		h := &serviceHandler{reqHandler: mockReq}
		mockReq.EXPECT().NumberV1NumberProviderGet(gomock.Any(), gomock.Any()).Return(testProviderInfo(), nil)

		if _, err := h.NumberProviderGet(context.Background(), testProviderSuperAdmin(), "a1234567890123456789012345678901"); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	})
}

// an agent without an id cannot pass the super admin check, so the actor is never nil
func Test_NumberProvider_agentWithoutIDIsRefused(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h := &serviceHandler{reqHandler: requesthandler.NewMockRequestHandler(mc)}
	a := &auth.AuthIdentity{Type: auth.TypeAgent}

	if _, err := h.NumberProviderUpdateSecret(context.Background(), a, "telnyx", testProviderSecret); !errors.Is(err, serviceerrors.ErrPermissionDenied) {
		t.Errorf("Wrong match. expect: %v, got: %v", serviceerrors.ErrPermissionDenied, err)
	}
}

// the secret appears in no log entry, on the success path and on the error path
func Test_NumberProviderUpdateSecret_logsHaveNoSecret(t *testing.T) {
	tests := []struct {
		name      string
		rpcErr    error
		expectMsg string
	}{
		{"success", nil, ""},
		{"rpc error", requesthandler.ErrServiceUnavailable, "Could not store the number provider secret"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			hook := newSecretLogHook(t)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &serviceHandler{reqHandler: mockReq}
			ctx := context.Background()

			var info *nmprovider.Info
			if tt.rpcErr == nil {
				info = testProviderInfo()
			}
			mockReq.EXPECT().NumberV1NumberProviderUpdateSecret(ctx, nmnumber.ProviderName("telnyx"), gomock.Any(), testProviderSecret).Return(info, tt.rpcErr)

			_, err := h.NumberProviderUpdateSecret(ctx, testProviderSuperAdmin(), "telnyx", testProviderSecret)
			if !errors.Is(err, tt.rpcErr) {
				t.Fatalf("Wrong match. expect: %v, got: %v", tt.rpcErr, err)
			}

			if tt.expectMsg != "" {
				assertNoSecretInLogs(t, hook, tt.expectMsg, testProviderSecret)
				return
			}
			for _, entry := range hook.AllEntries() {
				if entry.Message != "" && strings.Contains(entry.Message, testProviderSecret) {
					t.Errorf("secret leaked into the log message: %q", entry.Message)
				}
			}
		})
	}
}

// a refused request logs no secret either
func Test_NumberProviderUpdateSecret_refusedLogsHaveNoSecret(t *testing.T) {
	for who, tt := range testProviderDenied() {
		t.Run(who, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			hook := newSecretLogHook(t)
			h := &serviceHandler{reqHandler: requesthandler.NewMockRequestHandler(mc)}

			if _, err := h.NumberProviderUpdateSecret(context.Background(), tt.a, "telnyx", testProviderSecret); !errors.Is(err, tt.err) {
				t.Fatalf("Wrong match. expect: %v, got: %v", tt.err, err)
			}

			assertNoSecretInLogs(t, hook, "The request is not allowed.", testProviderSecret)
		})
	}
}
