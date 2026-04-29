package conversationhandler

import (
	"testing"

	"github.com/gofrs/uuid"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-conversation-manager/models/conversation"
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
