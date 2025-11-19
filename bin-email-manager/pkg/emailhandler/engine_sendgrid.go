package emailhandler

//go:generate mockgen -package emailhandler -destination ./mock_engine_sendgrid.go -source engine_sendgrid.go -build_flags=-mod=mod

import (
	"context"
	"fmt"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-email-manager/models/email"
	smbucketfile "monorepo/bin-storage-manager/models/bucketfile"

	"github.com/pkg/errors"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"github.com/sirupsen/logrus"
)

type engineSendgrid struct {
	utilHandler utilhandler.UtilHandler
	reqHandler  requesthandler.RequestHandler

	client *sendgrid.Client
}

type EngineSendgrid interface {
	Send(ctx context.Context, m *email.Email) (string, error)
}

func NewEngineSendgrid(reqHandler requesthandler.RequestHandler, apikey string) EngineSendgrid {
	client := sendgrid.NewSendClient(apikey)

	return &engineSendgrid{
		utilHandler: utilhandler.NewUtilHandler(),
		reqHandler:  reqHandler,

		client: client,
	}
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
	for _, a := range m.Attachments {
		attach, err := h.getAttachment(ctx, &a)
		if err != nil {
			log.WithField("attachment", a).Errorf("Could not get attachment. err: %v", err)
			continue
		}

		_ = message.AddAttachment(attach)
	}

	// Send the email
	resp, err := h.client.Send(message)
	if err != nil {
		return "", errors.Wrapf(err, "could not send email.")
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
func (h *engineSendgrid) getAttachment(ctx context.Context, e *email.Attachment) (*mail.Attachment, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "getAttachment",
		"attachment": e,
	})

	var f *smbucketfile.BucketFile
	var err error

	fileType := ""
	filename := ""
	switch e.ReferenceType {
	case email.AttachmentReferenceTypeRecording:
		f, err = h.reqHandler.StorageV1RecordingGet(ctx, e.ReferenceID, 60000)
		if err != nil {
			return nil, errors.Wrapf(err, "could not get attachment. reference_type: %s, reference_id: %s", e.ReferenceType, e.ReferenceID)
		}
		log.WithField("recording", f).Debugf("Got recording attachment. recording_id: %s", f.ReferenceID)

		fileType = "application/zip"
		filename = fmt.Sprintf("%s.zip", f.ReferenceID)

	default:
		return nil, errors.Errorf("unknown attachment reference type: %v", e.ReferenceType)
	}

	encodedAttachment, err := downloadToBase64(ctx, f.DownloadURI)
	if err != nil {
		return nil, errors.Wrapf(err, "could not download attachment")
	}

	res := mail.NewAttachment()
	res.SetContent(encodedAttachment)
	res.SetType(fileType)
	res.SetFilename(filename)
	res.SetDisposition("attachment") //  "attachment" = download, "inline" = display in email if possible

	return res, nil
}
