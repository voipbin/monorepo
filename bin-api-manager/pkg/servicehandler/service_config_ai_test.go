package servicehandler

import (
	"context"
	"errors"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/pkg/serviceerrors"
	"monorepo/bin-common-handler/pkg/confighandler/configrpc"
	"monorepo/bin-common-handler/pkg/requesthandler"
)

// The services of the config API: all of them are allowed, for all four operations, and the service name
// reaches the request handler as it is.
func Test_ServiceConfig_services(t *testing.T) {
	tests := []struct {
		service string
		key     string
	}{
		{"email", "mailgun.api_key"},
		{"ai", "llm.gemini.api_key"},
		{"message", "provider.telnyx.auth_token"},
	}

	for _, tt := range tests {
		t.Run(tt.service, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &serviceHandler{reqHandler: mockReq}
			ctx := context.Background()
			a := testProviderSuperAdmin()

			key := testServiceConfigKeyModel()
			key.Key = tt.key

			mockReq.EXPECT().ConfigV1ServiceGet(ctx, tt.service).Return(&configrpc.Service{Service: tt.service}, nil).Times(2)
			mockReq.EXPECT().ConfigV1KeySecretSet(ctx, tt.service, tt.key, uuid.FromStringOrNil(testProviderActor), testServiceConfigSecret, uint32(4)).Return(key, nil)
			mockReq.EXPECT().ConfigV1KeyClear(ctx, tt.service, tt.key, uuid.FromStringOrNil(testProviderActor), uint32(5)).Return(key, nil)
			mockReq.EXPECT().ConfigV1KeyHistoryList(ctx, tt.service, tt.key, "token", 5).Return(&configrpc.HistoryList{}, nil)

			if _, err := h.ServiceConfigGet(ctx, a, tt.service); err != nil {
				t.Errorf("Wrong match. get expect: ok, got: %v", err)
			}
			if _, err := h.ServiceConfigKeySecretSet(ctx, a, tt.service, tt.key, testServiceConfigSecret, 4); err != nil {
				t.Errorf("Wrong match. secret set expect: ok, got: %v", err)
			}
			if _, err := h.ServiceConfigKeyClear(ctx, a, tt.service, tt.key, 5); err != nil {
				t.Errorf("Wrong match. clear expect: ok, got: %v", err)
			}
			if _, err := h.ServiceConfigKeyHistoryList(ctx, a, tt.service, tt.key, "token", 5); err != nil {
				t.Errorf("Wrong match. history expect: ok, got: %v", err)
			}
		})
	}
}

// The allow list of this handler and the queue map of the request handler are two lists. When a
// service is in this one only, the request handler refuses it before anything is sent, and the
// secret is not sent either: the probe of the secret write gets the error first.
func Test_ServiceConfig_serviceUnknownToTheRequestHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &serviceHandler{reqHandler: mockReq}
	ctx := context.Background()
	a := testProviderSuperAdmin()

	// ConfigV1KeySecretSet is not expected: gomock fails the test if the secret is sent
	mockReq.EXPECT().ConfigV1ServiceGet(ctx, "ai").Return(nil, requesthandler.ErrConfigServiceUnknown).Times(2)

	if _, err := h.ServiceConfigGet(ctx, a, "ai"); !errors.Is(err, requesthandler.ErrConfigServiceUnknown) {
		t.Errorf("Wrong match. expect: %v, got: %v", requesthandler.ErrConfigServiceUnknown, err)
	}
	if _, err := h.ServiceConfigKeySecretSet(ctx, a, "ai", "llm.gemini.api_key", testServiceConfigSecret, 0); !errors.Is(err, requesthandler.ErrConfigServiceUnknown) {
		t.Errorf("Wrong match. expect: %v, got: %v", requesthandler.ErrConfigServiceUnknown, err)
	}
}

// A manager that is not updated yet answers the config requests with a bare 404: a read or a
// delete is a not found, and the secret write is refused as unavailable before the secret is sent.
func Test_ServiceConfig_managerNotUpdated(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &serviceHandler{reqHandler: mockReq}
	ctx := context.Background()
	a := testProviderSuperAdmin()

	mockReq.EXPECT().ConfigV1ServiceGet(ctx, "ai").Return(nil, requesthandler.ErrNotFound).Times(2)
	mockReq.EXPECT().ConfigV1KeyClear(ctx, "ai", "llm.gemini.api_key", gomock.Any(), uint32(0)).Return(nil, requesthandler.ErrNotFound)

	if _, err := h.ServiceConfigGet(ctx, a, "ai"); !errors.Is(err, requesthandler.ErrNotFound) {
		t.Errorf("Wrong match. expect: %v, got: %v", requesthandler.ErrNotFound, err)
	}
	if _, err := h.ServiceConfigKeySecretSet(ctx, a, "ai", "llm.gemini.api_key", testServiceConfigSecret, 0); !errors.Is(err, serviceerrors.ErrServiceUnavailable) {
		t.Errorf("Wrong match. expect: %v, got: %v", serviceerrors.ErrServiceUnavailable, err)
	}
	if _, err := h.ServiceConfigKeyClear(ctx, a, "ai", "llm.gemini.api_key", 0); !errors.Is(err, requesthandler.ErrNotFound) {
		t.Errorf("Wrong match. expect: %v, got: %v", requesthandler.ErrNotFound, err)
	}
}
