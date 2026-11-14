package emailhandler

//go:generate mockgen -package emailhandler -destination ./mock_engine_sendgrid.go -source engine_sendgrid.go -build_flags=-mod=mod

import (
	"context"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-email-manager/models/email"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"github.com/sirupsen/logrus"
)

type engineSendgrid struct {
	utilHandler utilhandler.UtilHandler
	reqHandler  requesthandler.RequestHandler

	// apiKey returns the API key at the time of the send.
	apiKey SecretFunc

	// client, when set, is used as is (tests inject a client with a test base URL).
	client *sendgrid.Client

	// newClient builds the client from the API key. nil means sendgrid.NewSendClient.
	newClient func(apiKey string) *sendgrid.Client
}

type EngineSendgrid interface {
	Send(ctx context.Context, m *email.Email) (string, error)
}

func NewEngineSendgrid(reqHandler requesthandler.RequestHandler, apiKey SecretFunc) EngineSendgrid {
	return &engineSendgrid{
		utilHandler: utilhandler.NewUtilHandler(),
		reqHandler:  reqHandler,

		apiKey: apiKey,
	}
}

// getClient returns the client of the send. The API key is read now, not at construction, so a
// rotated key is used by the next send. An empty key builds the client as before and the send
// fails at the provider.
func (h *engineSendgrid) getClient(ctx context.Context) (*sendgrid.Client, error) {
	if h.client != nil {
		return h.client, nil
	}
	if h.apiKey == nil {
		return nil, errors.New("the sendgrid api key supplier is not set")
	}

	key, err := h.apiKey(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not get the sendgrid api key")
	}

	build := h.newClient
	if build == nil {
		build = sendgrid.NewSendClient
	}

	return build(key), nil
}

// Send sends the email using sendgrid
func (h *engineSendgrid) Send(ctx context.Context, m *email.Email) (string, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "Send",
		"email_id": m.ID,
	})
	log.Debugf("Sending an email. email_id: %v", m.ID)

	// Create a V3 mail object
	message := mail.NewV3Mail()

	// Set the source
	source := mail.NewEmail(m.Source.TargetName, m.Source.Target)
	message.SetFrom(source)

	// set the subject
	message.Subject = m.Subject

	// Set the content
	content := mail.NewContent("text/plain", m.Content)
	message.AddContent(content)

	// set destinations
	personalize := mail.NewPersonalization()
	for _, d := range m.Destinations {
		personalize.AddTos(mail.NewEmail(d.TargetName, d.Target))
	}

	// set message id
	personalize.SetCustomArg("voipbin_message_id", m.ID.String())
	message.AddPersonalizations(personalize)

	// add attachments
	usedNames := map[string]bool{}
	for _, a := range m.Attachments {
		attach, err := h.getAttachment(ctx, m.CustomerID, &a, usedNames)
		if err != nil {
			return "", errors.Wrapf(err, "could not get attachment. reference_type: %s, reference_id: %s", a.ReferenceType, a.ReferenceID)
		}

		_ = message.AddAttachment(attach)
	}

	// Send the email
	client, err := h.getClient(ctx)
	if err != nil {
		return "", err
	}

	resp, err := client.Send(message)
	if err != nil {
		return "", errors.Wrapf(err, "could not send email.")
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.WithFields(logrus.Fields{
			"status_code":   resp.StatusCode,
			"response_body": resp.Body,
		}).Errorf("Unexpected response status code.")
		return "", errors.Errorf("unexpected response status code: %d", resp.StatusCode)
	}

	if resp.Headers == nil {
		log.WithFields(logrus.Fields{
			"status_code":   resp.StatusCode,
			"response_body": resp.Body,
		}).Errorf("Response headers are nil, could not get message id from response headers.")
		return "", errors.Errorf("response headers are nil, could not get message id from response headers")
	}

	messageIDs := resp.Headers["X-Message-Id"]
	if len(messageIDs) == 0 {
		log.WithFields(logrus.Fields{
			"response_headers": resp.Headers,
			"status_code":      resp.StatusCode,
			"response_body":    resp.Body,
		}).Errorf("Could not get message id from response headers.")
		return "", errors.Errorf("could not get message id from response headers")
	}

	res := messageIDs[0]
	return res, nil
}

// getAttachment returns the sendgrid attachment from the given email attachment
func (h *engineSendgrid) getAttachment(ctx context.Context, customerID uuid.UUID, e *email.Attachment, usedNames map[string]bool) (*mail.Attachment, error) {
	filename, fileType, downloadURI, err := resolveAttachment(ctx, h.reqHandler, customerID, e)
	if err != nil {
		return nil, err
	}

	encodedAttachment, err := downloadToBase64(ctx, downloadURI)
	if err != nil {
		return nil, errors.Wrapf(err, "could not download attachment")
	}

	res := mail.NewAttachment()
	res.SetContent(encodedAttachment)
	res.SetType(fileType)
	res.SetFilename(uniqueFilename(usedNames, filename))
	res.SetDisposition("attachment") //  "attachment" = download, "inline" = display in email if possible

	return res, nil
}
