package servicehandler

import (
	"context"
	"net/http"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"

	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/websockhandler"
)

type mockResponseWriter struct{}

func (h *mockResponseWriter) Header() http.Header        { return http.Header{} }
func (h *mockResponseWriter) Write([]byte) (int, error)  { return 0, nil }
func (h *mockResponseWriter) WriteHeader(statusCode int) {}

func Test_WebsockCreate(t *testing.T) {

	tests := []struct {
		name     string
		customer *cscustomer.Customer

		writer  http.ResponseWriter
		request *http.Request
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
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

			mockWebsock.EXPECT().Run(ctx, gomock.Any(), gomock.Any(), tt.customer.ID).Return(nil)

			if err := h.WebsockCreate(ctx, tt.customer, tt.writer, tt.request); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}
