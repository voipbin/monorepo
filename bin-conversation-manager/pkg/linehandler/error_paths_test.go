package linehandler

import (
	"context"
	"errors"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"

	"monorepo/bin-conversation-manager/models/account"
	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/models/media"
)

func newTestLineAccount() *account.Account {
	return &account.Account{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("792c0222-e4a9-11ec-af5e-679fe5991907"),
		},
		Secret: "test-secret",
		Token:  "test-token",
	}
}

// Test_Send_nothingToPush verifies that a request without text never reaches the
// push call: media is unsupported and ignored, so no external call is made.
func Test_Send_nothingToPush(t *testing.T) {
	h := NewLineHandler(nil)

	cv := &conversation.Conversation{DialogID: "test-dialog"}
	medias := []media.Media{{Type: media.TypeFile}}

	if err := h.Send(context.Background(), cv, newTestLineAccount(), "", medias); err != nil {
		t.Errorf("expected nil, got: %v", err)
	}
}

// Test_SetupTeardown_canceledContext verifies that the webhook calls honor the
// caller's context: an already canceled context fails before a request is sent.
func Test_SetupTeardown_canceledContext(t *testing.T) {
	tests := []struct {
		name string
		call func(h LineHandler, ctx context.Context) error
	}{
		{"setup", func(h LineHandler, ctx context.Context) error { return h.Setup(ctx, newTestLineAccount()) }},
		{"teardown", func(h LineHandler, ctx context.Context) error { return h.Teardown(ctx, newTestLineAccount()) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewLineHandler(nil)

			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			err := tt.call(h, ctx)
			if !errors.Is(err, context.Canceled) {
				t.Errorf("expected context.Canceled, got: %v", err)
			}
		})
	}
}
