package conversationhandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/models/message"
)

func Test_getExecuteMode(t *testing.T) {
	tests := []struct {
		name string
		cv   *conversation.Conversation
		want ExecuteMode
	}{
		{
			name: "unassigned conversation -> flow mode",
			cv: &conversation.Conversation{
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeNone,
					OwnerID:   uuid.Nil,
				},
			},
			want: ExecuteModeFlow,
		},
		{
			name: "agent owner with non-nil owner id -> agent mode",
			cv: &conversation.Conversation{
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
				},
			},
			want: ExecuteModeAgent,
		},
		{
			name: "agent owner with nil owner id -> flow mode (defensive against malformed state)",
			cv: &conversation.Conversation{
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   uuid.Nil,
				},
			},
			want: ExecuteModeFlow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &conversationHandler{}
			got := h.getExecuteMode(tt.cv)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func Test_runExecuteModeAgent_isNoop(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &conversationHandler{reqHandler: mockReq}

	cv := &conversation.Conversation{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
			CustomerID: uuid.FromStringOrNil("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
		},
	}
	m := &message.Message{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("cccccccc-cccc-cccc-cccc-cccccccccccc"),
		},
	}

	// mockReq.EXPECT() — no expectations: any RPC call will fail the test.

	err := h.runExecuteModeAgent(context.Background(), cv, m)
	if err != nil {
		t.Errorf("expected nil error, got: %v", err)
	}
}
