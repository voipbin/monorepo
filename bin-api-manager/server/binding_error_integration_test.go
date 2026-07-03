package server

// End-to-end regression test for ETC-3: proves the full wiring in
// cmd/api-manager/main.go (RegisterHandlersWithOptions with
// ErrorHandler: BindingErrorHandler) actually reaches an HTTP client
// through the generated wrapper chain, not just BindingErrorHandler in
// isolation (see error_test.go for unit-level coverage). See
// docs/plans/2026-07-04-standardize-binding-error-envelope.md.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/lib/middleware"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cerrors "monorepo/bin-common-handler/models/errors"

	"github.com/gin-gonic/gin"
	"go.uber.org/mock/gomock"
)

// TestBindingError_EndToEnd_MalformedPathUUID_ReturnsStandardEnvelope
// mirrors cmd/api-manager/main.go's actual RegisterHandlersWithOptions
// call (not the bare RegisterHandlers used by every other *_test.go in
// this package for handler-level testing) and fires a malformed UUID
// path parameter against a real generated route whose "id" parameter
// is typed openapi_types.UUID at the wrapper level (DELETE
// /aiaudits/{id}), so binding fails before the handler — and therefore
// before any auth check — ever runs. Before this fix, this request
// would have returned oapi-codegen's default
// {"msg": "Invalid format for parameter id: ..."} fallback because
// main.go registered no custom ErrorHandler.
func TestBindingError_EndToEnd_MalformedPathUUID_ReturnsStandardEnvelope(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)
	h := &server{
		serviceHandler: mockSvc,
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware.RequestID())

	openapi_server.RegisterHandlersWithOptions(r, h, openapi_server.GinServerOptions{
		ErrorHandler: BindingErrorHandler,
	})

	req := httptest.NewRequest(http.MethodDelete, "/aiaudits/not-a-uuid", nil)
	r.ServeHTTP(w, req)

	// mockSvc's handler methods must never be called — binding fails
	// before the handler runs. gomock's default (no .EXPECT() call
	// configured) already fails the test if any method is invoked.

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d want %d; body=%s", w.Code, http.StatusBadRequest, w.Body.String())
	}

	var body struct {
		Error struct {
			Status    string `json:"status"`
			Reason    string `json:"reason"`
			Message   string `json:"message"`
			RequestID string `json:"request_id"`
		} `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("response body is not the standard envelope: %v; body=%s", err, w.Body.String())
	}
	if want := string(cerrors.StatusInvalidArgument); body.Error.Status != want {
		t.Errorf("status field = %q want %q", body.Error.Status, want)
	}
	if body.Error.Reason != "INVALID_REQUEST_PARAMETER" {
		t.Errorf("reason = %q want INVALID_REQUEST_PARAMETER", body.Error.Reason)
	}
	if body.Error.Message == "" {
		t.Error("message is empty")
	}
	if body.Error.RequestID == "" {
		t.Error("request_id missing from response body")
	}

	// Structural check: the old fallback shape ({"msg": ...}) must be
	// entirely gone — no "msg" key anywhere in the top-level body.
	var full map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &full); err != nil {
		t.Fatalf("unmarshal full body: %v; body=%s", err, w.Body.String())
	}
	if _, hasMsg := full["msg"]; hasMsg {
		t.Errorf("legacy {\"msg\": ...} fallback shape still present; body=%s", w.Body.String())
	}
}
