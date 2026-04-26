package listenhandler

import (
	stderrors "errors"
	"fmt"
	"net/http"
	"testing"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-billing-manager/pkg/dbhandler"

	pkgerrors "github.com/pkg/errors"
)

// Test_errorResponse_typed verifies that a *cerrors.VoipbinError is encoded
// via cerrors.ToResponse — the api-manager edge will recover the typed error
// from the resulting sock.Response via FromResponse.
func Test_errorResponse_typed(t *testing.T) {
	ve := cerrors.InvalidArgument(
		commonoutline.ServiceNameBillingManager,
		"UNSUPPORTED_BILLING_TYPE",
		"The billing type is not supported.",
	)

	resp := errorResponse(ve)
	if resp == nil {
		t.Fatalf("expected response, got nil")
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("StatusCode mismatch. expected: %d, got: %d", http.StatusBadRequest, resp.StatusCode)
	}
	if resp.DataType != cerrors.DataTypeVoipbinError {
		t.Errorf("DataType mismatch. expected: %s, got: %s", cerrors.DataTypeVoipbinError, resp.DataType)
	}
	if len(resp.Data) == 0 {
		t.Errorf("expected non-empty Data containing JSON-encoded VoipbinError")
	}
}

// Test_errorResponse_typedThroughPkgErrorsWrap verifies that errors.As walks
// the pkg/errors.Wrapf chain — the standard wrapping idiom in business
// handlers — and recovers the typed error.
func Test_errorResponse_typedThroughPkgErrorsWrap(t *testing.T) {
	ve := cerrors.InvalidArgument(
		commonoutline.ServiceNameBillingManager,
		"INVALID_ACCOUNT_STATUS",
		"The account status is not valid.",
	)
	wrapped := pkgerrors.Wrap(ve, "outer context")

	resp := errorResponse(wrapped)
	if resp == nil {
		t.Fatalf("expected response, got nil")
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("StatusCode mismatch. expected: %d, got: %d", http.StatusBadRequest, resp.StatusCode)
	}
	if resp.DataType != cerrors.DataTypeVoipbinError {
		t.Errorf("DataType mismatch. expected: %s, got: %s", cerrors.DataTypeVoipbinError, resp.DataType)
	}
}

// Test_errorResponse_legacyNotFound verifies that dbhandler.ErrNotFound
// (wrapped via pkg/errors) preserves the legacy 404 mapping for code paths
// that have not yet been migrated to typed errors.
func Test_errorResponse_legacyNotFound(t *testing.T) {
	wrapped := pkgerrors.Wrap(dbhandler.ErrNotFound, "could not get account info")

	resp := errorResponse(wrapped)
	if resp == nil {
		t.Fatalf("expected response, got nil")
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("StatusCode mismatch. expected: 404, got: %d", resp.StatusCode)
	}
	if resp.DataType != "" {
		t.Errorf("DataType should be empty for legacy 404, got: %s", resp.DataType)
	}
	// stderrors.Is should still walk through pkg/errors wrap
	if !stderrors.Is(wrapped, dbhandler.ErrNotFound) {
		t.Errorf("stderrors.Is should walk pkg/errors chain to find ErrNotFound")
	}
}

// Test_errorResponse_unclassifiedError verifies that any unknown error
// (e.g., a DB connection failure) falls through to 500 — becomes INTERNAL
// at the api-manager translator default branch.
func Test_errorResponse_unclassifiedError(t *testing.T) {
	plain := fmt.Errorf("connection refused")

	resp := errorResponse(plain)
	if resp == nil {
		t.Fatalf("expected response, got nil")
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("StatusCode mismatch. expected: 500, got: %d", resp.StatusCode)
	}
}

// Test_errorResponse_nilError defensive test — calling with nil should not
// crash and should return 500.
func Test_errorResponse_nilError(t *testing.T) {
	resp := errorResponse(nil)
	if resp == nil {
		t.Fatalf("expected response, got nil")
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("StatusCode mismatch. expected: 500, got: %d", resp.StatusCode)
	}
}
