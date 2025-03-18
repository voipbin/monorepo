package servicehandler

import (
	"context"
	"fmt"
	amagent "monorepo/bin-agent-manager/models/agent"
	ammessage "monorepo/bin-ai-manager/models/message"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

// aimessageGet returns the aimessage info.
func (h *serviceHandler) aimessageGet(ctx context.Context, id uuid.UUID) (*ammessage.Message, error) {
	res, err := h.reqHandler.AIV1MessageGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the message info. id: %v", id)
	}
	return res, nil
}

// AImessageCreate sends a message to the aicall.
func (h *serviceHandler) AImessageCreate(
	ctx context.Context,
	a *amagent.Agent,
	aicallID uuid.UUID,
	role ammessage.Role,
	content string,
) (*ammessage.WebhookMessage, error) {
	_, err := h.AIcallGet(ctx, a, aicallID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the aicall info. aicall_id: %v", aicallID)
	}

	tmp, err := h.reqHandler.AIV1MessageSend(ctx, aicallID, role, content, 30000)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create a new ai message. err: %v", err)
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// AImessageGetsByAIcallID gets the list of ai messages of the given aicall id.
// It returns list of ai messages if it succeed.
func (h *serviceHandler) AImessageGetsByAIcallID(ctx context.Context, a *amagent.Agent, aicallID uuid.UUID, size uint64, token string) ([]*ammessage.WebhookMessage, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	if size == 0 {
		size = 100
	}

	_, err := h.AIcallGet(ctx, a, aicallID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the aicall info. id: %v", aicallID)
	}

	filters := map[string]string{
		"deleted":   "false",
		"aicall_id": aicallID.String(),
	}
	tmps, err := h.reqHandler.AIV1MessageGetsByAIcallID(ctx, aicallID, token, size, filters)
	if err != nil {
		return nil, fmt.Errorf("could not find ai messages info. err: %v", err)
	}

	res := []*ammessage.WebhookMessage{}
	for _, f := range tmps {
		tmp := f.ConvertWebhookMessage()
		res = append(res, tmp)
	}

	return res, nil
}

// AImessageGet gets the ai message of the given id.
// It returns ai message if it succeed.
func (h *serviceHandler) AImessageGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*ammessage.WebhookMessage, error) {
	tmp, err := h.aimessageGet(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("could not find ai message info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("agent has no permission")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// AImessageDelete deletes the aimessage.
func (h *serviceHandler) AImessageDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*ammessage.WebhookMessage, error) {
	_, err := h.AImessageGet(ctx, a, id)
	if err != nil {
		return nil, fmt.Errorf("could not find aimessage info. err: %v", err)
	}

	tmp, err := h.reqHandler.AIV1MessageDelete(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not delete the aimessage. err: %v", err)
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
