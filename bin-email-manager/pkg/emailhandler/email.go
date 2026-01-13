package emailhandler

import (
	"context"
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-email-manager/models/email"

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
	// validate destinations
	for _, destination := range destinations {
		if !h.validateEmailAddress(destination) {
			return nil, errors.New("destination is not valid")
		}
	}

	res, err := h.create(ctx, customerID, activeflowID, email.ProviderTypeSendgrid, defaultSource, destinations, subject, content, attachments)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create email")
	}

	// send the email in go routine
	go h.Send(context.Background(), res)

	return res, nil
}

// Gets returns list of emails
func (h *emailHandler) Gets(ctx context.Context, token string, size uint64, filters map[email.Field]any) ([]*email.Email, error) {
	res, err := h.db.EmailGets(ctx, token, size, filters)
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

func (h *emailHandler) Get(ctx context.Context, id uuid.UUID) (*email.Email, error) {
	res, err := h.db.EmailGet(ctx, id)
	if err != nil {
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
