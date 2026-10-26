package listenhandler

import (
	"net/http"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
)

// A typed error whose details cannot be encoded still ends in a plain 500 response, and the
// failure is logged. The standard logger is global, so this test does not run in parallel.
func Test_errorResponse_unencodableDetails(t *testing.T) {
	logger := logrus.StandardLogger()
	previous := logger.ReplaceHooks(make(logrus.LevelHooks))
	hook := logrustest.NewGlobal()
	t.Cleanup(func() { logger.ReplaceHooks(previous) })

	typed := cerrors.NotFound(commonoutline.ServiceNameTalkManager, "CHAT_NOT_FOUND", "x")
	typed.Details = []map[string]any{{"unencodable": make(chan int)}}

	resp := errorResponse(typed)
	if resp.StatusCode != http.StatusInternalServerError || resp.DataType == cerrors.DataTypeVoipbinError {
		t.Errorf("errorResponse() = %d/%s, want a plain 500 response", resp.StatusCode, resp.DataType)
	}

	var failures int
	for _, entry := range hook.AllEntries() {
		if entry.Level == logrus.ErrorLevel && strings.Contains(entry.Message, "ToResponse failed") {
			failures++
		}
	}
	if failures != 1 {
		t.Errorf("errorResponse() logged the encoding failure %d times, want 1", failures)
	}
}
