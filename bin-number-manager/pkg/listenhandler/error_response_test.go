package listenhandler

import (
	stderrors "errors"
	"fmt"
	"net/http"
	"testing"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-number-manager/pkg/dbhandler"

	pkgerrors "github.com/pkg/errors"
)

func Test_errorResponse_typed(t *testing.T) {
	ve := cerrors.NotFound(commonoutline.ServiceNameNumberManager, "NUMBER_NOT_FOUND", "x")
	resp := errorResponse(ve)
	if resp.StatusCode != http.StatusNotFound || resp.DataType != cerrors.DataTypeVoipbinError {
		t.Errorf("typed: got %d/%s", resp.StatusCode, resp.DataType)
	}
}

func Test_errorResponse_typedThroughPkgErrorsWrap(t *testing.T) {
	ve := cerrors.NotFound(commonoutline.ServiceNameNumberManager, "NUMBER_NOT_FOUND", "x")
	resp := errorResponse(pkgerrors.Wrap(ve, "outer"))
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("wrapped: got %d", resp.StatusCode)
	}
}

func Test_errorResponse_legacyNotFound(t *testing.T) {
	resp := errorResponse(pkgerrors.Wrap(dbhandler.ErrNotFound, "x"))
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("legacy: got %d", resp.StatusCode)
	}
}

func Test_errorResponse_unclassifiedError(t *testing.T) {
	if errorResponse(fmt.Errorf("x")).StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500")
	}
}

func Test_errorResponse_nilError(t *testing.T) {
	if errorResponse(nil).StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500")
	}
}

func Test_errorResponse_typedNotFoundFromBusinessHandler(t *testing.T) {
	typed := cerrors.NotFound(commonoutline.ServiceNameNumberManager, "NUMBER_NOT_FOUND", "x").Wrap(dbhandler.ErrNotFound)
	resp := errorResponse(typed)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("got %d", resp.StatusCode)
	}
	if !stderrors.Is(typed, dbhandler.ErrNotFound) {
		t.Errorf("Is should walk")
	}
}
