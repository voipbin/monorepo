package listenhandler

import (
	stderrors "errors"
	"fmt"
	"net/http"
	"testing"

	"monorepo/bin-agent-manager/pkg/dbhandler"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	pkgerrors "github.com/pkg/errors"
)

func Test_errorResponse_typed(t *testing.T) {
	ve := cerrors.NotFound(commonoutline.ServiceNameAgentManager, "AGENT_NOT_FOUND", "The agent was not found.")
	resp := errorResponse(ve)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("StatusCode mismatch. expected: 404, got: %d", resp.StatusCode)
	}
	if resp.DataType != cerrors.DataTypeVoipbinError {
		t.Errorf("DataType mismatch. expected: %s, got: %s", cerrors.DataTypeVoipbinError, resp.DataType)
	}
}

func Test_errorResponse_typedThroughPkgErrorsWrap(t *testing.T) {
	ve := cerrors.NotFound(commonoutline.ServiceNameAgentManager, "AGENT_NOT_FOUND", "Not found.")
	wrapped := pkgerrors.Wrap(ve, "outer")
	resp := errorResponse(wrapped)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("StatusCode mismatch. expected: 404, got: %d", resp.StatusCode)
	}
}

func Test_errorResponse_legacyNotFound(t *testing.T) {
	wrapped := pkgerrors.Wrap(dbhandler.ErrNotFound, "x")
	resp := errorResponse(wrapped)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("StatusCode mismatch. expected: 404, got: %d", resp.StatusCode)
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
	typed := cerrors.NotFound(commonoutline.ServiceNameAgentManager, "AGENT_NOT_FOUND", "Not found.").Wrap(dbErr)
	resp := errorResponse(typed)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("StatusCode mismatch. expected: 404, got: %d", resp.StatusCode)
	}
	if !stderrors.Is(typed, dbhandler.ErrNotFound) {
		t.Errorf("errors.Is should walk VoipbinError.Unwrap chain to find ErrNotFound")
	}
}
