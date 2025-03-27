package summaryhandler

import (
	"context"
	"monorepo/bin-ai-manager/models/summary"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

func (h *summaryHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,
	referenceType summary.ReferenceType,
	referenceID uuid.UUID,
	language string,
	content string,
) (*summary.Summary, error) {

	id := h.utilHandler.UUIDCreate()

	m := &summary.Summary{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},

		ReferenceType: referenceType,
		ReferenceID:   referenceID,

		Language: language,
		Content:  content,
	}

	if errCreate := h.db.SummaryCreate(ctx, m); errCreate != nil {
		return nil, errors.Wrapf(errCreate, "could not create summary")
	}

	res, err := h.db.SummaryGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get created data")
	}

	if errSet := h.variableSet(ctx, res); errSet != nil {
		return nil, errors.Wrapf(errSet, "could not set the variable")
	}

	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, summary.EventTypeCreated, res)

	return res, nil
}
