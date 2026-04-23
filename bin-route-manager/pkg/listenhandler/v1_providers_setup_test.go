package listenhandler

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-route-manager/models/provider"
	"monorepo/bin-route-manager/pkg/providerhandler"

	"github.com/gofrs/uuid"
)

func Test_v1ProvidersSetupPost(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		carrier          string
		pname            string
		detail           string
		apiKey           string
		responseProvider *provider.Provider
		expectStatus     int
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/providers/setup",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"carrier":"telnyx","name":"My Telnyx","detail":"desc","credentials":{"api_key":"KEY_xxx"}}`),
			},
			carrier:          "telnyx",
			pname:            "My Telnyx",
			detail:           "desc",
			apiKey:           "KEY_xxx",
			responseProvider: &provider.Provider{ID: uuid.FromStringOrNil("997a7752-4872-11ed-be7a-5783111a9092")},
			expectStatus:     200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockProvider := providerhandler.NewMockProviderHandler(ctrl)
			mockProvider.EXPECT().Setup(gomock.Any(), tt.carrier, tt.pname, tt.detail, tt.apiKey).
				Return(tt.responseProvider, nil)

			h := &listenHandler{providerHandler: mockProvider}
			res, err := h.v1ProvidersSetupPost(context.Background(), tt.request)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.StatusCode != tt.expectStatus {
				t.Fatalf("expected status %d, got %d", tt.expectStatus, res.StatusCode)
			}
		})
	}
}
