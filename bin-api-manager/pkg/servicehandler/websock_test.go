package servicehandler

import (
	"context"
	"net/http"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/pkg/dbhandler"
	"monorepo/bin-api-manager/pkg/websockhandler"
)

type mockResponseWriter struct{}

func (h *mockResponseWriter) Header() http.Header        { return http.Header{} }
func (h *mockResponseWriter) Write([]byte) (int, error)  { return 0, nil }
func (h *mockResponseWriter) WriteHeader(statusCode int) {}

func Test_WebsockCreate(t *testing.T) {

	tests := []struct {
		name  string
		agent *amagent.Agent

		writer  http.ResponseWriter
		request *http.Request
	}{
		{
			"normal",
			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},

			&mockResponseWriter{},
			&http.Request{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			mockWebsock := websockhandler.NewMockWebsockHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,

				websockHandler: mockWebsock,
			}

			ctx := context.Background()

			mockWebsock.EXPECT().RunSubscription(ctx, gomock.Any(), gomock.Any(), tt.agent).Return(nil)

			if err := h.WebsockCreate(ctx, tt.agent, tt.writer, tt.request); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}
