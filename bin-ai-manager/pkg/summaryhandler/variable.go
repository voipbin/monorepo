package summaryhandler

import (
	"context"
	"monorepo/bin-ai-manager/models/summary"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

func (h *summaryHandler) variableSet(ctx context.Context, s *summary.Summary) error {

	if s.ActiveflowID == uuid.Nil {
		return nil
	}

	variables := map[string]string{
		variableSummaryID:            s.ID.String(),
		variableSummaryReferenceType: string(s.ReferenceType),
		variableSummaryReferenceID:   s.ReferenceID.String(),
		variableSummaryLanguage:      s.Language,
		variableSummaryContent:       s.Content,
	}

	if errSet := h.reqestHandler.FlowV1VariableSetVariable(ctx, s.ActiveflowID, variables); errSet != nil {
		return errors.Wrapf(errSet, "could not set the variable. summary_id: %s", s.ID)
	}

	return nil
}
