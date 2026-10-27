package listenhandler

import (
	"net/http"
	"testing"

	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/models/sock"

	"go.uber.org/mock/gomock"

	"monorepo/bin-registrar-manager/pkg/extensionhandler"
)

// Test_processRequest_extensionNumberMalformedEscape verifies that a request URI with an
// invalid percent escape is answered with 400 and never reaches the extension handler
// (the mock has no expectation).
func Test_processRequest_extensionNumberMalformedEscape(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockExt := extensionhandler.NewMockExtensionHandler(mc)
	h := &listenHandler{extensionHandler: mockExt}

	res, err := h.processRequest(&sock.Request{
		URI:    "/v1/extensions/extension/a%zz",
		Method: sock.RequestMethodGet,
		Data:   []byte(`{"customer_id":"e0000000-0000-4000-8000-000000000001"}`),
	})
	if err != nil {
		t.Fatalf("Wrong match. processRequest must not return the error, got: %v", err)
	}
	if res == nil || res.StatusCode != http.StatusBadRequest {
		t.Errorf("Wrong match. expect status: %d, got: %+v", http.StatusBadRequest, res)
	}
}

// Test_errorResponse_typedNil verifies that a typed nil *VoipbinError, which matches the
// typed-error branch but cannot be converted, is answered with 500.
func Test_errorResponse_typedNil(t *testing.T) {
	var typedNil *cerrors.VoipbinError

	res := errorResponse(typedNil)
	if res == nil || res.StatusCode != http.StatusInternalServerError {
		t.Errorf("Wrong match. expect status: %d, got: %+v", http.StatusInternalServerError, res)
	}
	if res != nil && res.DataType != "" {
		t.Errorf("Wrong match. expect no typed error body, got data type: %s", res.DataType)
	}
}
