package summaryhandler

import (
	"context"
	stderrors "errors"
	"fmt"

	"monorepo/bin-ai-manager/models/summary"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

func (h *summaryHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,
	activeflowID uuid.UUID,
	onEndFlowID uuid.UUID,
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

		ActiveflowID: activeflowID,
		OnEndFlowID:  onEndFlowID,

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

	if errSet := h.variableSet(ctx, res.ActiveflowID, res); errSet != nil {
		return nil, errors.Wrapf(errSet, "could not set the variable")
	}

	if res.Status == summary.StatusDone {
		promSummaryDoneTotal.WithLabelValues(string(res.ReferenceType)).Inc()
	}

	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, summary.EventTypeCreated, res)
	return res, nil
}

func (h *summaryHandler) Get(ctx context.Context, id uuid.UUID) (*summary.Summary, error) {

	res, err := h.db.SummaryGet(ctx, id)
	if err != nil {
		if stderrors.Is(err, dbhandler.ErrNotFound) {
			return nil, cerrors.NotFound(
				commonoutline.ServiceNameAIManager,
				"SUMMARY_NOT_FOUND",
				"The summary was not found.",
			).Wrap(err)
		}
		return nil, errors.Wrapf(err, "could not get data")
	}

	return res, nil
}

func (h *summaryHandler) GetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*summary.Summary, error) {
	filters := map[summary.Field]any{
		summary.FieldDeleted:     false,
		summary.FieldReferenceID: referenceID,
	}

	tmps, err := h.db.SummaryList(ctx, 1, "", filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get data")
	}
	if len(tmps) == 0 {
		return nil, errors.Errorf("could not find the summary")
	}

	res := tmps[0]
	return res, nil
}

func (h *summaryHandler) List(ctx context.Context, size uint64, token string, filters map[summary.Field]any) ([]*summary.Summary, error) {
	res, err := h.db.SummaryList(ctx, size, token, filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get data")
	}

	return res, nil
}

func (h *summaryHandler) GetByCustomerIDAndReferenceIDAndLanguage(
	ctx context.Context,
	customerID uuid.UUID,
	referenceID uuid.UUID,
	language string,
) (*summary.Summary, error) {
	filters := map[summary.Field]any{
		summary.FieldDeleted:     false,
		summary.FieldCustomerID:  customerID,
		summary.FieldReferenceID: referenceID,
		summary.FieldLanguage:    language,
	}
	res, err := h.List(ctx, 1000, "", filters)
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
		return nil, errors.Wrapf(err, "could not get updated summary")
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, summary.EventTypeDeleted, res)

	return res, nil
}

// ErrSummaryAlreadyDone is returned by UpdateStatusDone when the conditional DB update
// affected zero rows: either the summary was already StatusDone at write time (VOIP-
// 1422: bin-conference-manager can publish conference_deleted twice for one
// conference, and neither ContentProcess nor its downstream, including
// startOnEndFlow, is safe to run twice), or -- rare, but the same WHERE clause can't
// tell the two apart -- the summary ID no longer exists at all. Both cases are a clean
// no-op from the caller's perspective, so this is intentionally one sentinel, not two.
// Callers must treat this as a clean no-op, not a failure -- do not wrap it in a
// user-facing error, do not retry, and do not run any of the on-success side effects
// (on-end-flow trigger, caller-side notifications). Matches the same
// conditional-update-plus-sentinel-error pattern aicallhandler.ErrAIcallNoLongerActive
// (pkg/aicallhandler/db.go) already uses for the equivalent problem on AIcalls.
var ErrSummaryAlreadyDone = stderrors.New("summary is already done")

// UpdateStatusDone updates the summary status to done. Returns ErrSummaryAlreadyDone
// (not a wrapped/generic error) if the summary was already done -- see that var's doc
// comment for why this must not be raced past.
func (h *summaryHandler) UpdateStatusDone(ctx context.Context, id uuid.UUID, content string) (*summary.Summary, error) {
	rowsAffected, err := h.db.SummaryUpdateStatusDoneIfNotDone(ctx, id, content)
	if err != nil {
		return nil, errors.Wrapf(err, "could not update the summary")
	}
	if rowsAffected == 0 {
		return nil, ErrSummaryAlreadyDone
	}

	res, err := h.db.SummaryGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get updated summary")
	}
	promSummaryDoneTotal.WithLabelValues(string(res.ReferenceType)).Inc()
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, summary.EventTypeUpdated, res)

	return res, nil
}
