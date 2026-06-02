package listenhandler

import (
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-registrar-manager/pkg/dbhandler"
	"monorepo/bin-registrar-manager/pkg/extensionhandler"
)

func Test_processV1ExtensionsIDDirectHashRegenerate_notFound(t *testing.T) {
	tests := []struct {
		name        string
		extensionID uuid.UUID
		request     *sock.Request
		expectRes   *sock.Response
	}{
		{
			name:        "not found returns 404",
			extensionID: uuid.FromStringOrNil("a1b2c3d4-0001-0001-0001-000000000001"),
			request: &sock.Request{
				URI:    "/v1/extensions/a1b2c3d4-0001-0001-0001-000000000001/direct-hash-regenerate",
				Method: sock.RequestMethodPost,
			},
			expectRes: &sock.Response{
				StatusCode: 404,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockExtension := extensionhandler.NewMockExtensionHandler(mc)

			h := &listenHandler{
				sockHandler:      mockSock,
				reqHandler:       mockReq,
				extensionHandler: mockExtension,
			}

			mockExtension.EXPECT().DirectHashRegenerate(gomock.Any(), tt.extensionID).Return(nil, dbhandler.ErrNotFound)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res.StatusCode != tt.expectRes.StatusCode {
				t.Errorf("Wrong status code. expect: %d, got: %d", tt.expectRes.StatusCode, res.StatusCode)
			}
		})
	}
}
