package requesthandler

import (
	"context"
	"encoding/json"
	"testing"

	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

func Test_KamailioV1ProviderHealthCheck(t *testing.T) {
	tests := []struct {
		name     string
		hostname string

		expectTarget  string
		expectRequest *sock.Request

		response     *sock.Response
		expectResult *KamailioProviderHealthResult
		expectErrNil bool
	}{
		{
			name:     "healthy provider",
			hostname: "sip.example.com",

			expectTarget: "kamailio.request",
			expectRequest: &sock.Request{
				URI:      "/v1/providers/health",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"hostname":"sip.example.com"}`),
			},

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"status":"healthy","result_code":"200"}`),
			},
			expectResult: &KamailioProviderHealthResult{
				Status:     "healthy",
				ResultCode: "200",
			},
			expectErrNil: true,
		},
		{
			name:     "unhealthy provider - timeout",
			hostname: "sip.unreachable.com",

			expectTarget: "kamailio.request",
			expectRequest: &sock.Request{
				URI:      "/v1/providers/health",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"hostname":"sip.unreachable.com"}`),
			},

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"status":"unhealthy","result_code":"timeout"}`),
			},
			expectResult: &KamailioProviderHealthResult{
				Status:     "unhealthy",
				ResultCode: "timeout",
			},
			expectErrNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.KamailioV1ProviderHealthCheck(context.Background(), tt.hostname)
			if tt.expectErrNil && err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if res == nil {
				t.Fatal("Expected non-nil result")
			}
			resJSON, _ := json.Marshal(res)
			expectJSON, _ := json.Marshal(tt.expectResult)
			if string(resJSON) != string(expectJSON) {
				t.Errorf("Wrong match.\n  expect: %s\n  got:    %s", expectJSON, resJSON)
			}
		})
	}
}
