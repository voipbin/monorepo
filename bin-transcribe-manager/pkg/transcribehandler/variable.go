package transcribehandler

import (
	"context"
	"monorepo/bin-transcribe-manager/models/transcribe"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

func (h *transcribeHandler) variableSet(ctx context.Context, activeflowID uuid.UUID, tr *transcribe.Transcribe) error {
	if activeflowID == uuid.Nil {
		return nil
	}

	variables := map[string]string{
		variableTranscribeID:        tr.ID.String(),
		variableTranscribeLanguage:  tr.Language,
		variableTranscribeDirection: string(tr.Direction),
	}

	if errSet := h.reqHandler.FlowV1VariableSetVariable(ctx, activeflowID, variables); errSet != nil {
		return errors.Wrapf(errSet, "could not set the variables.")
	}

	return nil
}
