package summaryhandler

import (
	"context"
	"fmt"
	"monorepo/bin-ai-manager/models/summary"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

func (h *summaryHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,
	activeflowID uuid.UUID,
	referenceType summary.ReferenceType,
	referenceID uuid.UUID,
	status summary.Status,
	language string,
	content string,
) (*summary.Summary, error) {

	id := h.utilHandler.UUIDCreate()

	m := &summary.Summary{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},

		ActiveflowID:  activeflowID,
		ReferenceType: referenceType,
		ReferenceID:   referenceID,

		Status:   status,
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

func (h *summaryHandler) Get(ctx context.Context, id uuid.UUID) (*summary.Summary, error) {

	res, err := h.db.SummaryGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get data")
	}

	return res, nil
}

func (h *summaryHandler) Gets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*summary.Summary, error) {
	res, err := h.db.SummaryGets(ctx, size, token, filters)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (h *summaryHandler) GetByCustomerIDAndReferenceIDAndLanguage(
	ctx context.Context,
	customerID uuid.UUID,
	referenceID uuid.UUID,
	language string,
) (*summary.Summary, error) {
	filters := map[string]string{
		"deleted":      "false",
		"customer_id":  customerID.String(),
		"reference_id": referenceID.String(),
		"language":     language,
	}

	res, err := h.Gets(ctx, 1000, "", filters)
	if err != nil {
		return nil, err
	}

	if len(res) == 0 {
		return nil, fmt.Errorf("could not find the summary")
	}

	return res[0], nil
}

// Delete deletes the summary.
func (h *summaryHandler) Delete(ctx context.Context, id uuid.UUID) (*summary.Summary, error) {
	if err := h.db.SummaryDelete(ctx, id); err != nil {
		return nil, errors.Wrapf(err, "could not delete the summary")
	}

	res, err := h.db.SummaryGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not updated summary")
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, summary.EventTypeDeleted, res)

	return res, nil
}
