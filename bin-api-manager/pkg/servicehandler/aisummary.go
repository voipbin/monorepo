package servicehandler

import (
	"context"
	"fmt"
	amagent "monorepo/bin-agent-manager/models/agent"
	amsummary "monorepo/bin-ai-manager/models/summary"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

// AIcallCreate is a service handler for aicall creation.
func (h *serviceHandler) AISummaryCreate(
	ctx context.Context,
	a *amagent.Agent,
	referenceType amsummary.ReferenceType,
	referenceID uuid.UUID,
	language string,

) (*amsummary.WebhookMessage, error) {

	tmpCustomerID := uuid.Nil
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
		referenceType,
		referenceID,
		language,
		50000,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create aicall")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
