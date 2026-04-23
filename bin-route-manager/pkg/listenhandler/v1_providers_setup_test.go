package listenhandler

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-route-manager/models/provider"
	"monorepo/bin-route-manager/pkg/providerhandler"
	"monorepo/bin-route-manager/pkg/telnyxclient"

	"github.com/gofrs/uuid"
)

func Test_v1ProvidersSetupPost(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		setupMock    func(m *providerhandler.MockProviderHandler)
		expectStatus int
		expectErr    bool
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/providers/setup",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"carrier":"telnyx","name":"My Telnyx","detail":"desc","credentials":{"api_key":"KEY_xxx"}}`),
			},
			setupMock: func(m *providerhandler.MockProviderHandler) {
				m.EXPECT().Setup(gomock.Any(), "telnyx", "My Telnyx", "desc", "KEY_xxx").
					Return(&provider.Provider{ID: uuid.FromStringOrNil("997a7752-4872-11ed-be7a-5783111a9092")}, nil)
			},
			expectStatus: 200,
		},
		{
			name: "invalid_api_key_returns_422",
			request: &sock.Request{
				URI:      "/v1/providers/setup",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"carrier":"telnyx","name":"My Telnyx","detail":"desc","credentials":{"api_key":"BAD_KEY"}}`),
			},
			setupMock: func(m *providerhandler.MockProviderHandler) {
				m.EXPECT().Setup(gomock.Any(), "telnyx", "My Telnyx", "desc", "BAD_KEY").
					Return(nil, telnyxclient.ErrInvalidKey)
			},
			expectStatus: 422,
		},
		{
			name: "generic_error_propagates",
			request: &sock.Request{
				URI:      "/v1/providers/setup",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"carrier":"telnyx","name":"My Telnyx","detail":"desc","credentials":{"api_key":"KEY_xxx"}}`),
			},
			setupMock: func(m *providerhandler.MockProviderHandler) {
				m.EXPECT().Setup(gomock.Any(), "telnyx", "My Telnyx", "desc", "KEY_xxx").
					Return(nil, errors.New("internal error"))
			},
			expectErr: true,
		},
		{
			name: "missing_carrier_returns_400",
			request: &sock.Request{
				URI:      "/v1/providers/setup",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"carrier":"","name":"My Telnyx","detail":"desc","credentials":{"api_key":"KEY_xxx"}}`),
			},
			setupMock:    func(m *providerhandler.MockProviderHandler) {},
			expectStatus: 400,
		},
		{
			name: "missing_api_key_returns_400",
			request: &sock.Request{
				URI:      "/v1/providers/setup",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"carrier":"telnyx","name":"My Telnyx","detail":"desc","credentials":{"api_key":""}}`),
			},
			setupMock:    func(m *providerhandler.MockProviderHandler) {},
			expectStatus: 400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockProvider := providerhandler.NewMockProviderHandler(ctrl)
			tt.setupMock(mockProvider)

			h := &listenHandler{providerHandler: mockProvider}
			res, err := h.v1ProvidersSetupPost(context.Background(), tt.request)

			if tt.expectErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.StatusCode != tt.expectStatus {
				t.Fatalf("expected status %d, got %d", tt.expectStatus, res.StatusCode)
			}
		})
	}
}
