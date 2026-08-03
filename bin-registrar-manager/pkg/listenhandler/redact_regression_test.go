package listenhandler

import (
	"fmt"
	"strings"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	"go.uber.org/mock/gomock"

	"monorepo/bin-registrar-manager/models/extension"
	"monorepo/bin-registrar-manager/pkg/extensionhandler"
)

// assertNoLeak fails the test if any captured log entry (message or any
// field, formatted the same way logrus would emit it) contains needle. See
// pkg/extensionhandler/redact_regression_test.go for the rationale; this
// exercises the transport (listenhandler) layer specifically, since the
// direct_hash leak found in VOIP-1287 review only reproduced through the
// response-body logging path in processRequest, not through the business
// layer directly.
func assertNoLeak(t *testing.T, entries []*logrus.Entry, needle string) {
	t.Helper()

	for _, entry := range entries {
		if strings.Contains(entry.Message, needle) {
			t.Errorf("Secret leaked into log message: %q", entry.Message)
		}
		for k, v := range entry.Data {
			if strings.Contains(fmt.Sprintf("%v", v), needle) {
				t.Errorf("Secret leaked into log field %q: %v", k, v)
			}
		}
	}
}

func Test_processV1ExtensionsIDDirectHashRegenerate_neverLogsPlaintextHash(t *testing.T) {
	hook := logrustest.NewLocal(logrus.StandardLogger())
	defer func() { logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks)) }()
	logrus.SetLevel(logrus.DebugLevel)
	defer logrus.SetLevel(logrus.InfoLevel)

	const plaintextHash = "direct-hash-must-not-leak-9f8e7d"

	extensionID := uuid.FromStringOrNil("a1b2c3d4-0001-0001-0001-000000000001")

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

	responseExtension := &extension.Extension{
		Identity:   commonidentity.Identity{ID: extensionID},
		DirectHash: plaintextHash,
	}
	mockExtension.EXPECT().DirectHashRegenerate(gomock.Any(), extensionID).Return(responseExtension, nil)

	req := &sock.Request{
		URI:    "/v1/extensions/a1b2c3d4-0001-0001-0001-000000000001/direct-hash-regenerate",
		Method: sock.RequestMethodPost,
	}

	res, err := h.processRequest(req)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.StatusCode != 200 {
		t.Fatalf("Wrong status code. expect: 200, got: %d", res.StatusCode)
	}
	if !strings.Contains(string(res.Data), plaintextHash) {
		t.Fatalf("Test setup invalid: response body does not contain the hash, redaction couldn't have been exercised")
	}

	assertNoLeak(t, hook.AllEntries(), plaintextHash)
}
