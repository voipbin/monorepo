package listenhandler

import (
	stderrors "errors"
	"fmt"
	"net/http"
	"testing"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-registrar-manager/pkg/dbhandler"

	pkgerrors "github.com/pkg/errors"
)

func Test_errorResponse_typed(t *testing.T) {
	resp := errorResponse(cerrors.NotFound(commonoutline.ServiceNameRegistrarManager, "TRUNK_NOT_FOUND", "x"))
	if resp.StatusCode != http.StatusNotFound || resp.DataType != cerrors.DataTypeVoipbinError {
		t.Errorf("typed: got %d/%s", resp.StatusCode, resp.DataType)
	}
}
func Test_errorResponse_typedThroughPkgErrorsWrap(t *testing.T) {
	if errorResponse(pkgerrors.Wrap(cerrors.NotFound(commonoutline.ServiceNameRegistrarManager, "x", "x"), "outer")).StatusCode != http.StatusNotFound {
		t.Errorf("wrapped failed")
	}
}
func Test_errorResponse_legacyNotFound(t *testing.T) {
	if errorResponse(pkgerrors.Wrap(dbhandler.ErrNotFound, "x")).StatusCode != http.StatusNotFound {
		t.Errorf("legacy failed")
	}
}
func Test_errorResponse_unclassifiedError(t *testing.T) {
	if errorResponse(fmt.Errorf("x")).StatusCode != http.StatusInternalServerError {
		t.Errorf("unclassified failed")
	}
}
func Test_errorResponse_nilError(t *testing.T) {
	if errorResponse(nil).StatusCode != http.StatusInternalServerError {
		t.Errorf("nil failed")
	}
}
func Test_errorResponse_typedNotFoundFromBusinessHandler(t *testing.T) {
	typed := cerrors.NotFound(commonoutline.ServiceNameRegistrarManager, "TRUNK_NOT_FOUND", "x").Wrap(dbhandler.ErrNotFound)
	if errorResponse(typed).StatusCode != http.StatusNotFound {
		t.Errorf("e2e failed")
	}
	if !stderrors.Is(typed, dbhandler.ErrNotFound) {
		t.Errorf("Is should walk")
	}
}
