package servicehandler

import (
	"context"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/pkg/serviceerrors"
	"monorepo/bin-common-handler/pkg/confighandler/configrpc"
	"monorepo/bin-common-handler/pkg/requesthandler"
)

const (
	testServiceConfigSecret = "test-secret-not-real"
	testServiceConfigKey    = "mailgun.api_key"
)

func testServiceConfigKeyModel() *configrpc.Key {
	return &configrpc.Key{Key: testServiceConfigKey, Kind: "secret", Type: "string", Description: "The key.", Source: "db", HasSecret: true, KeyVersion: 1, Revision: 4}
}

func Test_ServiceConfigGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &serviceHandler{reqHandler: mockReq}
	ctx := context.Background()

	expect := &configrpc.Service{Service: "email", SecretWritable: true, Keys: []configrpc.Key{*testServiceConfigKeyModel()}}
	mockReq.EXPECT().ConfigV1ServiceGet(ctx, "email").Return(expect, nil)

	res, err := h.ServiceConfigGet(ctx, testProviderSuperAdmin(), "email")
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(expect, res) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expect, res)
	}
}

func Test_ServiceConfigKeySecretSet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &serviceHandler{reqHandler: mockReq}
	ctx := context.Background()

	// the manager is asked first whether it serves the config requests, then the secret is sent, with
	// the agent as the actor
	gomock.InOrder(
		mockReq.EXPECT().ConfigV1ServiceGet(ctx, "email").Return(&configrpc.Service{}, nil),
		mockReq.EXPECT().ConfigV1KeySecretSet(ctx, "email", testServiceConfigKey, uuid.FromStringOrNil(testProviderActor), testServiceConfigSecret, uint32(4)).Return(testServiceConfigKeyModel(), nil),
	)

	res, err := h.ServiceConfigKeySecretSet(ctx, testProviderSuperAdmin(), "email", testServiceConfigKey, testServiceConfigSecret, 4)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(testServiceConfigKeyModel(), res) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", testServiceConfigKeyModel(), res)
	}
}

// A manager that does not serve the config requests yet (an older version, during a deployment)
// would log the body of the secret request as an unknown request. The secret is not sent then.
func Test_ServiceConfigKeySecretSet_probe(t *testing.T) {
	tests := []struct {
		name     string
		probeErr error
		expect   error
	}{
		{"manager without the route", requesthandler.ErrNotFound, serviceerrors.ErrServiceUnavailable},
		{"manager not reachable", requesthandler.ErrServiceUnavailable, requesthandler.ErrServiceUnavailable},
		{"internal error", requesthandler.ErrInternal, requesthandler.ErrInternal},
		{"timeout", context.DeadlineExceeded, context.DeadlineExceeded},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &serviceHandler{reqHandler: mockReq}
			ctx := context.Background()

			// no ConfigV1KeySecretSet is expected: gomock fails the test if the secret is sent
			mockReq.EXPECT().ConfigV1ServiceGet(ctx, "email").Return(nil, tt.probeErr)

			if _, err := h.ServiceConfigKeySecretSet(ctx, testProviderSuperAdmin(), "email", testServiceConfigKey, testServiceConfigSecret, 0); !errors.Is(err, tt.expect) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expect, err)
			}
		})
	}
}

func Test_ServiceConfigKeySecretSet_errorsPass(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &serviceHandler{reqHandler: mockReq}
	ctx := context.Background()

	mockReq.EXPECT().ConfigV1ServiceGet(ctx, "email").Return(&configrpc.Service{}, nil)
	mockReq.EXPECT().ConfigV1KeySecretSet(ctx, "email", testServiceConfigKey, gomock.Any(), testServiceConfigSecret, uint32(1)).Return(nil, requesthandler.ErrConflict)

	if _, err := h.ServiceConfigKeySecretSet(ctx, testProviderSuperAdmin(), "email", testServiceConfigKey, testServiceConfigSecret, 1); !errors.Is(err, requesthandler.ErrConflict) {
		t.Errorf("Wrong match. expect: %v, got: %v", requesthandler.ErrConflict, err)
	}
}

func Test_ServiceConfigKeyClear(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &serviceHandler{reqHandler: mockReq}
	ctx := context.Background()

	mockReq.EXPECT().ConfigV1KeyClear(ctx, "email", testServiceConfigKey, uuid.FromStringOrNil(testProviderActor), uint32(0)).Return(testServiceConfigKeyModel(), nil)

	res, err := h.ServiceConfigKeyClear(ctx, testProviderSuperAdmin(), "email", testServiceConfigKey, 0)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(testServiceConfigKeyModel(), res) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", testServiceConfigKeyModel(), res)
	}
}

func Test_ServiceConfigKeyHistoryList(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &serviceHandler{reqHandler: mockReq}
	ctx := context.Background()

	expect := &configrpc.HistoryList{Result: []configrpc.History{{Key: testServiceConfigKey, Action: "set", IsSecret: true}}, NextPageToken: "next"}
	mockReq.EXPECT().ConfigV1KeyHistoryList(ctx, "email", testServiceConfigKey, "token", 5).Return(expect, nil)

	res, err := h.ServiceConfigKeyHistoryList(ctx, testProviderSuperAdmin(), "email", testServiceConfigKey, "token", 5)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(expect, res) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expect, res)
	}
}

// Every operation refuses the identities that are not the project super admin, before it looks at
// the arguments and without any rpc: gomock fails the test on an unexpected call.
func Test_ServiceConfig_denied(t *testing.T) {
	for name, tt := range testProviderDenied() {
		t.Run(name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &serviceHandler{reqHandler: mockReq}
			ctx := context.Background()

			calls := map[string]func() error{
				"get": func() error {
					_, err := h.ServiceConfigGet(ctx, tt.a, "nothing")
					return err
				},
				"secret set": func() error {
					_, err := h.ServiceConfigKeySecretSet(ctx, tt.a, "email", "BAD KEY", testServiceConfigSecret, -1)
					return err
				},
				"clear": func() error {
					_, err := h.ServiceConfigKeyClear(ctx, tt.a, "email", "BAD KEY", -1)
					return err
				},
				"history": func() error {
					_, err := h.ServiceConfigKeyHistoryList(ctx, tt.a, "email", "BAD KEY", "", 0)
					return err
				},
			}
			for op, call := range calls {
				if err := call(); !errors.Is(err, tt.err) {
					t.Errorf("Wrong match. %s expect: %v, got: %v", op, tt.err, err)
				}
			}
		})
	}
}

// The service is one of the known names and the key has the shape of a key: a request value can
// not reach the rpc as anything else.
func Test_ServiceConfig_validation(t *testing.T) {
	tests := []struct {
		name string

		service  string
		key      string
		revision int64

		expect error
	}{
		{"unknown service", "nothing", testServiceConfigKey, 0, serviceerrors.ErrNotFound},
		{"empty service", "", testServiceConfigKey, 0, serviceerrors.ErrNotFound},
		{"upper case service", "EMAIL", testServiceConfigKey, 0, serviceerrors.ErrNotFound},
		{"empty key", "email", "", 0, serviceerrors.ErrInvalidArgument},
		{"key with a slash", "email", "a/secret", 0, serviceerrors.ErrInvalidArgument},
		{"key with a query", "email", "a?x=1", 0, serviceerrors.ErrInvalidArgument},
		{"key with an upper case letter", "email", "Mailgun.api_key", 0, serviceerrors.ErrInvalidArgument},
		{"key too long", "email", strings.Repeat("a", 129), 0, serviceerrors.ErrInvalidArgument},
		{"negative revision", "email", testServiceConfigKey, -1, serviceerrors.ErrInvalidArgument},
		{"revision above the range", "email", testServiceConfigKey, math.MaxUint32 + 1, serviceerrors.ErrInvalidArgument},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// no rpc at all is expected
			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &serviceHandler{reqHandler: mockReq}
			ctx := context.Background()
			a := testProviderSuperAdmin()

			if _, err := h.ServiceConfigKeySecretSet(ctx, a, tt.service, tt.key, testServiceConfigSecret, tt.revision); !errors.Is(err, tt.expect) {
				t.Errorf("Wrong match. secret set expect: %v, got: %v", tt.expect, err)
			}
			if _, err := h.ServiceConfigKeyClear(ctx, a, tt.service, tt.key, tt.revision); !errors.Is(err, tt.expect) {
				t.Errorf("Wrong match. clear expect: %v, got: %v", tt.expect, err)
			}
			if tt.revision == 0 {
				if _, err := h.ServiceConfigKeyHistoryList(ctx, a, tt.service, tt.key, "", 0); !errors.Is(err, tt.expect) {
					t.Errorf("Wrong match. history expect: %v, got: %v", tt.expect, err)
				}
				if tt.key == testServiceConfigKey {
					if _, err := h.ServiceConfigGet(ctx, a, tt.service); !errors.Is(err, tt.expect) {
						t.Errorf("Wrong match. get expect: %v, got: %v", tt.expect, err)
					}
				}
			}
		})
	}
}

// The largest revision of the range is accepted.
func Test_ServiceConfig_revisionRange(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &serviceHandler{reqHandler: mockReq}
	ctx := context.Background()

	mockReq.EXPECT().ConfigV1KeyClear(ctx, "email", testServiceConfigKey, gomock.Any(), uint32(math.MaxUint32)).Return(testServiceConfigKeyModel(), nil)

	if _, err := h.ServiceConfigKeyClear(ctx, testProviderSuperAdmin(), "email", testServiceConfigKey, math.MaxUint32); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}

// Nothing the handler does writes the secret to a log or an error, on the success path and on the
// failing ones.
func Test_ServiceConfigKeySecretSet_secretIsNotLogged(t *testing.T) {
	hook := logtest.NewGlobal()
	defer hook.Reset()
	prev := logrus.GetLevel()
	logrus.SetLevel(logrus.TraceLevel)
	defer logrus.SetLevel(prev)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &serviceHandler{reqHandler: mockReq}
	ctx := context.Background()
	a := testProviderSuperAdmin()

	mockReq.EXPECT().ConfigV1ServiceGet(ctx, "email").Return(&configrpc.Service{}, nil).Times(3)
	mockReq.EXPECT().ConfigV1KeySecretSet(ctx, "email", testServiceConfigKey, gomock.Any(), testServiceConfigSecret, gomock.Any()).Return(testServiceConfigKeyModel(), nil)
	mockReq.EXPECT().ConfigV1KeySecretSet(ctx, "email", testServiceConfigKey, gomock.Any(), testServiceConfigSecret, gomock.Any()).Return(nil, requesthandler.ErrConflict)
	mockReq.EXPECT().ConfigV1KeySecretSet(ctx, "email", testServiceConfigKey, gomock.Any(), testServiceConfigSecret, gomock.Any()).Return(nil, fmt.Errorf("could not send the request"))

	var errs []error
	for i := 0; i < 3; i++ {
		_, err := h.ServiceConfigKeySecretSet(ctx, a, "email", testServiceConfigKey, testServiceConfigSecret, 1)
		errs = append(errs, err)
	}
	_, err := h.ServiceConfigKeySecretSet(ctx, a, "email", "BAD KEY", testServiceConfigSecret, 1)
	errs = append(errs, err)
	_, err = h.ServiceConfigKeySecretSet(ctx, testProviderDenied()["customer admin agent"].a, "email", testServiceConfigKey, testServiceConfigSecret, 1)
	errs = append(errs, err)

	for _, e := range hook.AllEntries() {
		if strings.Contains(e.Message+fmt.Sprint(e.Data), testServiceConfigSecret) {
			t.Errorf("Wrong match. a log line holds the secret: %s", e.Message)
		}
	}
	for _, e := range errs {
		if e != nil && strings.Contains(e.Error(), testServiceConfigSecret) {
			t.Errorf("Wrong match. an error holds the secret: %v", e)
		}
	}
}
