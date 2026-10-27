package linehandler

import (
	"context"
	"strings"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gofrs/uuid"
	"github.com/line/line-bot-sdk-go/v7/linebot"
	"go.uber.org/mock/gomock"

	"monorepo/bin-conversation-manager/models/account"
	"monorepo/bin-conversation-manager/models/conversation"
)

// newAccountWithoutCredentials returns an account whose credentials are empty, so the line client cannot be created and
// no request can leave the process.
func newAccountWithoutCredentials() *account.Account {
	return &account.Account{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("4f1d6c52-0b7e-4a3c-8f21-6a9d3e5b7c10"),
			CustomerID: uuid.FromStringOrNil("4f1d6c52-0b7e-4a3c-8f21-6a9d3e5b7c11"),
		},
	}
}

func Test_clientCreationFailure(t *testing.T) {
	tests := []struct {
		name string
		call func(h *lineHandler, ctx context.Context, ac *account.Account) error

		expectContext string
	}{
		{
			name: "send with a text",
			call: func(h *lineHandler, ctx context.Context, ac *account.Account) error {
				return h.Send(ctx, &conversation.Conversation{DialogID: "test-dialog"}, ac, "hello", nil)
			},
		},
		{
			name: "send without a text",
			call: func(h *lineHandler, ctx context.Context, ac *account.Account) error {
				return h.Send(ctx, &conversation.Conversation{DialogID: "test-dialog"}, ac, "", nil)
			},
		},
		{
			name: "setup",
			call: func(h *lineHandler, ctx context.Context, ac *account.Account) error {
				return h.Setup(ctx, ac)
			},
		},
		{
			name: "teardown",
			call: func(h *lineHandler, ctx context.Context, ac *account.Account) error {
				return h.Teardown(ctx, ac)
			},
		},
		{
			name: "get peer",
			call: func(h *lineHandler, ctx context.Context, ac *account.Account) error {
				_, err := h.GetPeer(ctx, ac, "test-user")
				return err
			},
			expectContext: "Could not get client",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// no request is declared: a failure to create the client must stop the call before any request
			h := &lineHandler{reqHandler: requesthandler.NewMockRequestHandler(mc)}

			err := tt.call(h, context.Background(), newAccountWithoutCredentials())
			if err == nil {
				t.Fatalf("error = nil, want the client creation failure")
			}
			if !strings.Contains(strings.ToLower(err.Error()), "channel") {
				t.Errorf("error = %q, want the cause from the line client", err.Error())
			}
			if tt.expectContext != "" && !strings.Contains(err.Error(), tt.expectContext) {
				t.Errorf("error = %q, want the context %q", err.Error(), tt.expectContext)
			}
		})
	}
}

func Test_hookEventHandle_nilEvent(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h := &lineHandler{reqHandler: requesthandler.NewMockRequestHandler(mc)}

	res, err := h.hookEventHandle(context.Background(), newAccountWithoutCredentials(), nil)
	if err == nil || err.Error() != "event is empty" {
		t.Errorf("error = %v, want %q", err, "event is empty")
	}
	if res != nil {
		t.Errorf("result = %v, want nil", res)
	}
}

func Test_hookEventTypeFollow_peerLookupFailure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	// no conversation is created when the peer cannot be resolved
	h := &lineHandler{reqHandler: requesthandler.NewMockRequestHandler(mc)}

	e := &linebot.Event{
		Type:   linebot.EventTypeFollow,
		Source: &linebot.EventSource{Type: linebot.EventSourceTypeUser, UserID: "test-user"},
	}

	err := h.hookEventTypeFollow(context.Background(), newAccountWithoutCredentials(), e)
	if err == nil {
		t.Fatalf("error = nil, want the peer lookup failure")
	}
	if !strings.Contains(err.Error(), "Could not get participant info") {
		t.Errorf("error = %q, want the participant context", err.Error())
	}
}
