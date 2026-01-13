package servicehandler

import (
	"context"
	"fmt"
	amagent "monorepo/bin-agent-manager/models/agent"
	amsummary "monorepo/bin-ai-manager/models/summary"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

// AISummaryCreate is a service handler for ai summary creation.
func (h *serviceHandler) AISummaryCreate(
	ctx context.Context,
	a *amagent.Agent,
	onEndFlowID uuid.UUID,
	referenceType amsummary.ReferenceType,
	referenceID uuid.UUID,
	language string,
) (*amsummary.WebhookMessage, error) {

	var tmpCustomerID uuid.UUID
	// get reference's customer info
	switch referenceType {
	case amsummary.ReferenceTypeCall:
		tmp, err := h.callGet(ctx, referenceID)
		if err != nil {
			return nil, errors.Wrapf(err, "could not get call info")
		}
		tmpCustomerID = tmp.CustomerID

	case amsummary.ReferenceTypeTranscribe:
		tmp, err := h.transcribeGet(ctx, referenceID)
		if err != nil {
			return nil, errors.Wrapf(err, "could not get transcribe info")
		}
		tmpCustomerID = tmp.CustomerID

	case amsummary.ReferenceTypeRecording:
		tmp, err := h.recordingGet(ctx, referenceID)
		if err != nil {
			return nil, errors.Wrapf(err, "could not get recording info")
		}
		tmpCustomerID = tmp.CustomerID

	case amsummary.ReferenceTypeConference:
		tmp, err := h.conferenceGet(ctx, referenceID)
		if err != nil {
			return nil, errors.Wrapf(err, "could not get conference info")
		}
		tmpCustomerID = tmp.CustomerID

	default:
		return nil, fmt.Errorf("unsupported reference type")
	}

	if !h.hasPermission(ctx, a, tmpCustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	tmp, err := h.reqHandler.AIV1SummaryCreate(
		ctx,
		a.CustomerID,
		uuid.Nil,
		onEndFlowID,
		referenceType,
		referenceID,
		language,
		50000,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create ai summary")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// aisummaryGet returns the ai summary info.
func (h *serviceHandler) aisummaryGet(ctx context.Context, id uuid.UUID) (*amsummary.Summary, error) {
	// send request
	res, err := h.reqHandler.AIV1SummaryGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the resource info")
	}

	return res, nil
}

// AISummaryGetsByCustomerID gets the list of aisummaries of the given customer id.
// It returns list of aisummaries if it succeed.
func (h *serviceHandler) AISummaryGetsByCustomerID(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*amsummary.WebhookMessage, error) {

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	// filters
	filters := map[string]string{
		"deleted":     "false", // we don't need deleted items
		"customer_id": a.CustomerID.String(),
	}

	// Convert string filters to typed filters
	typedFilters, err := h.convertAISummaryFilters(filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not convert filters")
	}

	tmps, err := h.reqHandler.AIV1SummaryGets(ctx, token, size, typedFilters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get ai summaries info")
	}

	// create result
	res := []*amsummary.WebhookMessage{}
	for _, f := range tmps {
		tmp := f.ConvertWebhookMessage()
		res = append(res, tmp)
	}

	return res, nil
}

// convertAISummaryFilters converts map[string]string to map[amsummary.Field]any
func (h *serviceHandler) convertAISummaryFilters(filters map[string]string) (map[amsummary.Field]any, error) {
	// Convert to map[string]any first
	srcAny := make(map[string]any, len(filters))
	for k, v := range filters {
		srcAny[k] = v
	}

	// Use reflection-based converter
	typed, err := commondatabasehandler.ConvertMapToTypedMap(srcAny, amsummary.Summary{})
	if err != nil {
		return nil, err
	}

	// Convert string keys to Field type
	result := make(map[amsummary.Field]any, len(typed))
	for k, v := range typed {
		result[amsummary.Field(k)] = v
	}

	return result, nil
}

// AISummaryGet gets the ai summary of the given id.
// It returns ai summary if it succeed.
func (h *serviceHandler) AISummaryGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*amsummary.WebhookMessage, error) {
	tmp, err := h.aisummaryGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get ai summaries info")
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// AISummaryDelete deletes the ai summary.
func (h *serviceHandler) AISummaryDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*amsummary.WebhookMessage, error) {
	c, err := h.aisummaryGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get ai summary info")
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	tmp, err := h.reqHandler.AIV1SummaryDelete(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not delete the ai summaries")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
