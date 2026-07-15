package emailhandler

import (
	"context"
	stderrors "errors"
	"fmt"

	bmbilling "monorepo/bin-billing-manager/models/billing"
	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-email-manager/models/email"
	"monorepo/bin-email-manager/pkg/dbhandler"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (h *emailHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,
	activeflowID uuid.UUID,
	destinations []commonaddress.Address,
	subject string,
	content string,
	attachments []email.Attachment,
) (*email.Email, error) {
	// Canonicalize destinations through the shared address normalization
	// authority BEFORE validation (Normalize-then-Validate). Email normalization
	// is lossless; the error is discarded. Normalize by index (range copy would
	// discard the result).
	for i := range destinations {
		destinations[i].Target, _ = commonaddress.NormalizeTarget(destinations[i].Type, destinations[i].Target)
	}

	// validate destinations
	for _, destination := range destinations {
		if !h.validateEmailAddress(destination) {
			return nil, errors.New("destination is not valid")
		}
	}

	// gate: customer identity verification (fail-closed for unverified customers)
	if !h.validateCustomerIdentityVerified(ctx, customerID) {
		return nil, fmt.Errorf("customer identity verification required to send email")
	}

	// validate balance before sending
	valid, err := h.reqHandler.BillingV1AccountIsValidBalanceByCustomerID(ctx, customerID, bmbilling.ReferenceTypeEmail, "", len(destinations))
	if err != nil {
		return nil, errors.Wrap(err, "could not validate the customer's balance")
	}
	if !valid {
		return nil, errors.New("insufficient balance for email")
	}

	// gate: outbound email rate limit (fail-closed). VOIP-1259.
	if !h.validateCustomerEmailRate(ctx, customerID) {
		return nil, cerrors.ResourceExhausted(commonoutline.ServiceNameEmailManager, "RATE_LIMIT_EXCEEDED", "outbound email rate limit exceeded")
	}

	res, err := h.create(ctx, customerID, activeflowID, email.ProviderTypeSendgrid, defaultSource, destinations, subject, content, attachments)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create email")
	}

	// send the email in go routine
	go h.Send(context.Background(), res)

	return res, nil
}

// List returns list of emails
func (h *emailHandler) List(ctx context.Context, token string, size uint64, filters map[email.Field]any) ([]*email.Email, error) {
	res, err := h.db.EmailList(ctx, token, size, filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get emails")
	}

	return res, nil
}

// Delete deletes the email
func (h *emailHandler) Delete(ctx context.Context, id uuid.UUID) (*email.Email, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "Delete",
		"email_id": id,
	})
	log.Debug("Deleting the email.")

	err := h.db.EmailDelete(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not delete email")
	}

	res, err := h.db.EmailGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get deleted email")
	}

	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, email.EventTypeDeleted, res)
	return res, nil
}

// Get returns email info.
//
// When the underlying DB layer returns dbhandler.ErrNotFound, Get returns a
// typed *cerrors.VoipbinError (Status=NotFound) so the api-manager edge can
// recover the upstream domain/reason via errors.As.
func (h *emailHandler) Get(ctx context.Context, id uuid.UUID) (*email.Email, error) {
	res, err := h.db.EmailGet(ctx, id)
	if err != nil {
		if stderrors.Is(err, dbhandler.ErrNotFound) {
			return nil, cerrors.NotFound(
				commonoutline.ServiceNameEmailManager,
				"EMAIL_NOT_FOUND",
				"The email was not found.",
			).Wrap(err)
		}
		return nil, errors.Wrapf(err, "could not get email")
	}

	return res, nil
}

func (h *emailHandler) create(
	ctx context.Context,
	customerID uuid.UUID,
	activeflowID uuid.UUID,
	providerType email.ProviderType,
	source *commonaddress.Address,
	destinations []commonaddress.Address,
	subject string,
	content string,
	attachments []email.Attachment,
) (*email.Email, error) {

	id := h.utilHandler.UUIDCreate()
	e := &email.Email{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},

		ActiveflowID: activeflowID,

		ProviderType:        providerType,
		ProviderReferenceID: "",

		Source:       source,
		Destinations: destinations,

		Status:  email.StatusInitiated,
		Subject: subject,
		Content: content,

		Attachments: attachments,
	}

	if errCreate := h.db.EmailCreate(ctx, e); errCreate != nil {
		return nil, errors.Wrapf(errCreate, "could not create email.")
	}

	res, err := h.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get created email.")
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, email.EventTypeCreated, res)

	return res, nil
}

func (h *emailHandler) UpdateProviderReferenceID(ctx context.Context, id uuid.UUID, providerReferenceID string) error {

	if errUpdate := h.db.EmailUpdateProviderReferenceID(ctx, id, providerReferenceID); errUpdate != nil {
		return errors.Wrapf(errUpdate, "could not update provider reference id.")
	}

	return nil
}

func (h *emailHandler) UpdateStatus(ctx context.Context, id uuid.UUID, status email.Status) (*email.Email, error) {

	if errUpdate := h.db.EmailUpdateStatus(ctx, id, status); errUpdate != nil {
		return nil, errors.Wrapf(errUpdate, "could not update status.")
	}

	res, err := h.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get updated email.")
	}

	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, email.EventTypeUpdated, res)
	return res, nil
}
