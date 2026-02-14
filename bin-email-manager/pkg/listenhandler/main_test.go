package listenhandler

import (
	"testing"

	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-email-manager/pkg/emailhandler"

	"go.uber.org/mock/gomock"
)

func TestNewListenHandler(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "creates_new_listen_handler",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockEmail := emailhandler.NewMockEmailHandler(mc)

			h := NewListenHandler(mockSock, mockEmail)

			if h == nil {
				t.Errorf("Expected non-nil handler")
			}
		})
	}
}

func TestSimpleResponse(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{
			name:       "creates_response_with_200",
			statusCode: 200,
		},
		{
			name:       "creates_response_with_404",
			statusCode: 404,
		},
		{
			name:       "creates_response_with_500",
			statusCode: 500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := simpleResponse(tt.statusCode)

			if resp == nil {
				t.Errorf("Expected non-nil response")
			}

			if resp.StatusCode != tt.statusCode {
				t.Errorf("Wrong status code. expect: %d, got: %d", tt.statusCode, resp.StatusCode)
			}
		})
	}
}
