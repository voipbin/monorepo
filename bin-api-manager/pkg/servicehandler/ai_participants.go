package servicehandler

import (
	"context"
	"fmt"

	amparticipant "monorepo/bin-ai-manager/models/participant"
	"monorepo/bin-api-manager/models/auth"

	"github.com/gofrs/uuid"
)

// AIcallParticipantGets gets the list of participants of the given aicall id.
// It returns a list of participant webhook messages if it succeeds.
func (h *serviceHandler) AIcallParticipantGets(ctx context.Context, a *auth.AuthIdentity, aicallID uuid.UUID, pageToken string, pageSize uint64) ([]*amparticipant.WebhookMessage, error) {
	if pageToken == "" {
		pageToken = h.utilHandler.TimeGetCurTime()
	}

	if pageSize == 0 {
		pageSize = 100
	}

	// verify the caller has access to the aicall
	_, err := h.AIcallGet(ctx, a, aicallID)
	if err != nil {
		return nil, fmt.Errorf("%w: could not get aicall info", err)
	}

	res, err := h.reqHandler.AIV1AIcallParticipantList(ctx, aicallID, pageToken, pageSize)
	if err != nil {
		return nil, fmt.Errorf("%w: could not get aicall participants", err)
	}

	return res, nil
}

// AIParticipantGets gets the list of participants of the given ai id.
// It returns a list of participant webhook messages if it succeeds.
func (h *serviceHandler) AIParticipantGets(ctx context.Context, a *auth.AuthIdentity, aiID uuid.UUID, pageToken string, pageSize uint64) ([]*amparticipant.WebhookMessage, error) {
	if pageToken == "" {
		pageToken = h.utilHandler.TimeGetCurTime()
	}

	if pageSize == 0 {
		pageSize = 100
	}

	// verify the caller has access to the ai
	_, err := h.AIGet(ctx, a, aiID)
	if err != nil {
		return nil, fmt.Errorf("%w: could not get ai info", err)
	}

	res, err := h.reqHandler.AIV1AIParticipantList(ctx, aiID, pageToken, pageSize)
	if err != nil {
		return nil, fmt.Errorf("%w: could not get ai participants", err)
	}

	return res, nil
}
