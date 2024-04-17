package messagehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/message-manager.git/models/message"
)

// setVariables sets the variables
func (h *messageHandler) setVariables(ctx context.Context, activeflowID uuid.UUID, m *message.Message) error {

	variables := map[string]string{

		// source
		variableMessageSourceName:       m.Source.Name,
		variableMessageSourceDetail:     m.Source.Detail,
		variableMessageSourceTarget:     m.Source.Target,
		variableMessageSourceTargetName: m.Source.Target,
		variableMessageSourceType:       string(m.Source.Type),

		// destination
		variableMessageTargetDestinationName:       m.Targets[0].Destination.Name,
		variableMessageTargetDestinationDetail:     m.Targets[0].Destination.Detail,
		variableMessageTargetDestinationTarget:     m.Targets[0].Destination.Target,
		variableMessageTargetDestinationTargetName: m.Targets[0].Destination.TargetName,
		variableMessageTargetDestinationType:       string(m.Targets[0].Destination.Type),

		// others
		variableMessageID:        m.ID.String(),
		variableMessageText:      m.Text,
		variableMessageDirection: string(m.Direction),
	}

	if errSet := h.reqHandler.FlowV1VariableSetVariable(ctx, activeflowID, variables); errSet != nil {
		return fmt.Errorf("could not set the variable. variables: %s, err: %v", variables, errSet)
	}

	return nil
}
