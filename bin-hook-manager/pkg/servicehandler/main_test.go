package servicehandler

import (
	"testing"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/pkg/requesthandler"
)

func TestNewServiceHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)

	tests := []struct {
		name       string
		reqHandler requesthandler.RequestHandler
	}{
		{
			name:       "create service handler",
			reqHandler: mockReq,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewServiceHandler(tt.reqHandler)
			if h == nil {
				t.Error("NewServiceHandler returned nil")
			}
		})
	}
}
