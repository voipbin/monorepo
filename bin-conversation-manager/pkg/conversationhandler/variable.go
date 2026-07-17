package conversationhandler

import (
	"context"
	"fmt"
	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/models/message"

	"github.com/gofrs/uuid"
)

// setVariables sets both the conversation and message variables.
// Kept for the message-triggered path (executeActiveflow) where both
// a Conversation and a Message are always available.
func (h *conversationHandler) setVariables(ctx context.Context, activeflowID uuid.UUID, cv *conversation.Conversation, m *message.Message) error {
	if errSet := h.setVariablesConversation(ctx, activeflowID, cv); errSet != nil {
		return errSet
	}

	if errSet := h.setVariablesMessage(ctx, activeflowID, m); errSet != nil {
		return errSet
	}

	return nil
}

// setVariablesConversation sets the voipbin.conversation.* variables.
// Usable on its own for flow-trigger paths that have a Conversation but
// no Message yet (e.g. CreateAndExecuteFlow's webchat session-start
// trigger, which runs before any Message exists).
func (h *conversationHandler) setVariablesConversation(ctx context.Context, activeflowID uuid.UUID, cv *conversation.Conversation) error {

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
	}

	if errSet := h.reqHandler.FlowV1VariableSetVariable(ctx, activeflowID, variables); errSet != nil {
		return fmt.Errorf("could not set the conversation variable. variables: %s, err: %v", variables, errSet)
	}

	return nil
}

// setVariablesMessage sets the voipbin.conversation_message.* variables.
func (h *conversationHandler) setVariablesMessage(ctx context.Context, activeflowID uuid.UUID, m *message.Message) error {

	variables := map[string]string{
		variableConversationMessageID:        m.ID.String(),
		variableConversationMessageText:      string(m.Text),
		variableConversationMessageDirection: string(m.Direction),
	}

	if errSet := h.reqHandler.FlowV1VariableSetVariable(ctx, activeflowID, variables); errSet != nil {
		return fmt.Errorf("could not set the message variable. variables: %s, err: %v", variables, errSet)
	}

	return nil
}
