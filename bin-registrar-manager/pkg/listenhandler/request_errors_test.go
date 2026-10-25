package listenhandler

import (
	"net/http"
	"testing"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"

	pkgerrors "github.com/pkg/errors"
	"go.uber.org/mock/gomock"

	"monorepo/bin-registrar-manager/pkg/contacthandler"
	"monorepo/bin-registrar-manager/pkg/dbhandler"
)

// Test_processRequest_handlerErrorMapping verifies how an error returned by a business
// handler is mapped to a response: a typed error keeps its own status and data type, a
// legacy not-found error becomes 404, and any other error becomes 400.
func Test_processRequest_handlerErrorMapping(t *testing.T) {
	tests := []struct {
		name string
		err  error

		expectStatus   int
		expectDataType string
	}{
		{
			name:           "typed error keeps its status and data type",
			err:            cerrors.PermissionDenied(commonoutline.ServiceNameRegistrarManager, "CASE_REASON", "case message"),
			expectStatus:   http.StatusForbidden,
			expectDataType: cerrors.DataTypeVoipbinError,
		},
		{
			name:         "wrapped legacy not found becomes 404",
			err:          pkgerrors.Wrap(dbhandler.ErrNotFound, "case"),
			expectStatus: http.StatusNotFound,
		},
		{
			name:         "other errors become 400",
			err:          pkgerrors.New("case failure"),
			expectStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockContact := contacthandler.NewMockContactHandler(mc)
			h := &listenHandler{contactHandler: mockContact}

			mockContact.EXPECT().ContactGetsByExtension(gomock.Any(), gomock.Any(), "1001").Return(nil, tt.err)

			res, err := h.processRequest(&sock.Request{
				URI:    "/v1/contacts",
				Method: sock.RequestMethodGet,
				Data:   []byte(`{"customer_id":"e0000000-0000-4000-8000-000000000001","extension":"1001"}`),
			})
			if err != nil {
				t.Fatalf("Wrong match. processRequest must not return the handler error, got: %v", err)
			}
			if res == nil || res.StatusCode != tt.expectStatus {
				t.Fatalf("Wrong match. expect status: %d, got: %+v", tt.expectStatus, res)
			}
			if tt.expectDataType != "" && res.DataType != tt.expectDataType {
				t.Errorf("Wrong match. expect data type: %s, got: %s", tt.expectDataType, res.DataType)
			}
		})
	}
}
