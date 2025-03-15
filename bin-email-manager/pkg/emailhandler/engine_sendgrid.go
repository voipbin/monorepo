package emailhandler

//go:generate mockgen -package emailhandler -destination ./mock_engine_sendgrid.go -source engine_sendgrid.go -build_flags=-mod=mod

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-email-manager/models/email"
	smbucketfile "monorepo/bin-storage-manager/models/bucketfile"
	"net/http"

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

	res := resp.Headers["X-Message-Id"][0]
	return res, nil
}

// getAttachment returns the sendgrid attachment from the given email attachment
func (h *engineSendgrid) getAttachment(ctx context.Context, e *email.Attachment) (*mail.Attachment, error) {
	var f *smbucketfile.BucketFile
	var err error

	fileType := ""
	filename := ""
	switch e.ReferenceType {
	case email.AttachmentReferenceTypeRecording:
		f, err = h.reqHandler.StorageV1RecordingGet(ctx, e.ReferenceID, 60000)
		fileType = "audio/wav"
		filename = fmt.Sprintf("%s.wav", f.ReferenceID)

	default:
		return nil, errors.Errorf("unknown attachment reference type: %v", e.ReferenceType)
	}

	if err != nil {
		return nil, errors.Wrapf(err, "could not get attachment")
	}

	encodedAttachment, err := h.downloadToBase64(ctx, f.DownloadURI)
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

// downloadToBase64 downloads the file from the given uri and returns the base64 encoded string
func (h *engineSendgrid) downloadToBase64(ctx context.Context, downloadURI string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", downloadURI, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", errors.Wrapf(err, "failed to download file")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.Wrapf(err, "failed to download file, status code: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrapf(err, "failed to read response body")
	}

	return base64.StdEncoding.EncodeToString(data), nil
}
