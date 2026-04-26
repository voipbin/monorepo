package listenhandler

import (
	stderrors "errors"
	"fmt"
	"net/http"
	"testing"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-flow-manager/pkg/dbhandler"

	pkgerrors "github.com/pkg/errors"
)

func Test_errorResponse_typed(t *testing.T) {
	ve := cerrors.NotFound(commonoutline.ServiceNameFlowManager, "FLOW_NOT_FOUND", "The flow was not found.")
	resp := errorResponse(ve)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("StatusCode mismatch. expected: 404, got: %d", resp.StatusCode)
	}
	if resp.DataType != cerrors.DataTypeVoipbinError {
		t.Errorf("DataType mismatch. expected: %s, got: %s", cerrors.DataTypeVoipbinError, resp.DataType)
	}
}

func Test_errorResponse_typedThroughPkgErrorsWrap(t *testing.T) {
	ve := cerrors.NotFound(commonoutline.ServiceNameFlowManager, "ACTIVEFLOW_NOT_FOUND", "The active flow was not found.")
	wrapped := pkgerrors.Wrap(ve, "outer context")
	resp := errorResponse(wrapped)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("StatusCode mismatch. expected: 404, got: %d", resp.StatusCode)
	}
	if resp.DataType != cerrors.DataTypeVoipbinError {
		t.Errorf("DataType mismatch. expected: %s, got: %s", cerrors.DataTypeVoipbinError, resp.DataType)
	}
}

func Test_errorResponse_legacyNotFound(t *testing.T) {
	wrapped := pkgerrors.Wrap(dbhandler.ErrNotFound, "could not get flow")
	resp := errorResponse(wrapped)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("StatusCode mismatch. expected: 404, got: %d", resp.StatusCode)
	}
	if resp.DataType != "" {
		t.Errorf("DataType should be empty for legacy 404, got: %s", resp.DataType)
	}
}

func Test_errorResponse_unclassifiedError(t *testing.T) {
	resp := errorResponse(fmt.Errorf("connection refused"))
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("StatusCode mismatch. expected: 500, got: %d", resp.StatusCode)
	}
}

func Test_errorResponse_nilError(t *testing.T) {
	resp := errorResponse(nil)
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("StatusCode mismatch. expected: 500, got: %d", resp.StatusCode)
	}
}

func Test_errorResponse_typedNotFoundFromBusinessHandler(t *testing.T) {
	dbErr := dbhandler.ErrNotFound
	typed := cerrors.NotFound(commonoutline.ServiceNameFlowManager, "FLOW_NOT_FOUND", "The flow was not found.").Wrap(dbErr)
	resp := errorResponse(typed)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("StatusCode mismatch. expected: 404, got: %d", resp.StatusCode)
	}
	if resp.DataType != cerrors.DataTypeVoipbinError {
		t.Errorf("DataType mismatch. expected: %s, got: %s", cerrors.DataTypeVoipbinError, resp.DataType)
	}
	if !stderrors.Is(typed, dbhandler.ErrNotFound) {
		t.Errorf("errors.Is should walk VoipbinError.Unwrap chain to find ErrNotFound")
	}
}
