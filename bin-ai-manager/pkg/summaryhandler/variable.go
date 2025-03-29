package summaryhandler

import (
	"context"
	"monorepo/bin-ai-manager/models/summary"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

func (h *summaryHandler) variableSet(ctx context.Context, activeflowID uuid.UUID, s *summary.Summary) error {

	if activeflowID == uuid.Nil {
		return nil
	}

	variables := map[string]string{
		variableSummaryID:            s.ID.String(),
		variableSummaryReferenceType: string(s.ReferenceType),
		variableSummaryReferenceID:   s.ReferenceID.String(),
		variableSummaryLanguage:      s.Language,
		variableSummaryContent:       s.Content,
	}

	if errSet := h.reqHandler.FlowV1VariableSetVariable(ctx, activeflowID, variables); errSet != nil {
		return errors.Wrapf(errSet, "could not set the variable. summary_id: %s", s.ID)
	}

	return nil
}
