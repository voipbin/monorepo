package conversationhandler

import (
	"context"
	"fmt"
	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/models/message"

	"github.com/gofrs/uuid"
)

// setVariables sets the variables
func (h *conversationHandler) setVariables(ctx context.Context, activeflowID uuid.UUID, cv *conversation.Conversation, m *message.Message) error {

	variables := map[string]string{

		variableConversationSelfName:       cv.Self.Name,
		variableConversationSelfDetail:     cv.Self.Detail,
		variableConversationSelfTarget:     cv.Self.Target,
		variableConversationSelfTargetName: cv.Self.TargetName,
		variableConversationSelfType:       string(cv.Self.Type),

		variableConversationPeerName:       cv.Peer.Name,
		variableConversationPeerDetail:     cv.Peer.Detail,
		variableConversationPeerTarget:     cv.Peer.Target,
		variableConversationPeerTargetName: cv.Peer.TargetName,
		variableConversationPeerType:       string(cv.Peer.Type),

		variableConversationID:      cv.ID.String(),
		variableConversationOwnerID: cv.OwnerID.String(),

		variableConversationMessageID:        m.ID.String(),
		variableConversationMessageText:      string(m.Text),
		variableConversationMessageDirection: string(m.Direction),
	}

	if errSet := h.reqHandler.FlowV1VariableSetVariable(ctx, activeflowID, variables); errSet != nil {
		return fmt.Errorf("could not set the variable. variables: %s, err: %v", variables, errSet)
	}

	return nil
}
