package servicehandler

import (
	"context"
	"fmt"
	amagent "monorepo/bin-agent-manager/models/agent"
	commonaddress "monorepo/bin-common-handler/models/address"
	ememail "monorepo/bin-email-manager/models/email"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

// emailGet gets the email by id.
func (h *serviceHandler) emailGet(ctx context.Context, emailID uuid.UUID) (*ememail.Email, error) {

	res, err := h.reqHandler.EmailV1EmailGet(ctx, emailID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the email")
	}

	if res.TMDelete < defaultTimestamp {
		return nil, fmt.Errorf("not found")
	}

	return res, nil
}

// EmailSend sends an email.
func (h *serviceHandler) EmailSend(
	ctx context.Context,
	a *amagent.Agent,
	destinations []commonaddress.Address,
	subject string,
	content string,
	attachments []ememail.Attachment,
) (*ememail.WebhookMessage, error) {

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	tmp, err := h.reqHandler.EmailV1EmailSend(ctx, a.CustomerID, uuid.Nil, destinations, subject, content, attachments)
	if err != nil {
		return nil, errors.Wrapf(err, "could not send email")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// EmailGets gets the list of email of the given customer id.
// It returns list of emails if it succeed.
func (h *serviceHandler) EmailGets(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*ememail.WebhookMessage, error) {

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	// filters
	filters := map[string]string{
		"customer_id": a.CustomerID.String(),
		"deleted":     "false", // we don't need deleted items
	}

	tmps, err := h.reqHandler.EmailV1EmailGets(ctx, token, size, filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get emails")
	}

	// create result
	res := []*ememail.WebhookMessage{}
	for _, f := range tmps {
		tmp := f.ConvertWebhookMessage()
		res = append(res, tmp)
	}

	return res, nil
}

// EmailGet gets the email of the given id.
// It returns email if it succeed.
func (h *serviceHandler) EmailGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*ememail.WebhookMessage, error) {

	tmp, err := h.emailGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get email")
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// EmailDelete deletes the email of the given id.
func (h *serviceHandler) EmailDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*ememail.WebhookMessage, error) {

	// get flow
	f, err := h.emailGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get email")
	}

	if !h.hasPermission(ctx, a, f.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	tmp, err := h.reqHandler.EmailV1EmailDelete(ctx, id)
	if err != nil {
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
