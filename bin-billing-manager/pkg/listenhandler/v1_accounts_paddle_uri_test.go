package listenhandler

import (
	"context"
	"testing"

	"monorepo/bin-common-handler/models/sock"
)

// Test_processV1AccountsIDPaddlePortalSessionPost_short_uri verifies that a URI without an account
// id segment is rejected as a bad request before any handler is called.
func Test_processV1AccountsIDPaddlePortalSessionPost_short_uri(t *testing.T) {
	// the listen handler has no account handler: a call to it would panic the test.
	h := &listenHandler{}

	res, err := h.processV1AccountsIDPaddlePortalSessionPost(context.Background(), &sock.Request{
		URI:    "/v1/accounts",
		Method: sock.RequestMethodPost,
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if res.StatusCode != 400 {
		t.Errorf("expected status 400, got: %d", res.StatusCode)
	}
}
