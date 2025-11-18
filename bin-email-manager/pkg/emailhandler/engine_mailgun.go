package emailhandler

//go:generate mockgen -package emailhandler -destination ./mock_engine_mailgun.go -source engine_mailgun.go -build_flags=-mod=mod

import (
	"context"
	"fmt"
	"io"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-email-manager/models/email"
	smbucketfile "monorepo/bin-storage-manager/models/bucketfile"
	"net/http"
	"time"

	"github.com/mailgun/mailgun-go"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type engineMailgun struct {
	utilHandler utilhandler.UtilHandler
	reqHandler  requesthandler.RequestHandler

	client *mailgun.MailgunImpl
	domain string
}

type EngineMailgun interface {
	Send(ctx context.Context, m *email.Email) (string, error)
}

func NewEngineMailgun(reqHandler requesthandler.RequestHandler, domain string, apiKey string) EngineMailgun {
	mg := mailgun.NewMailgun(domain, apiKey)

	return &engineMailgun{
		utilHandler: utilhandler.NewUtilHandler(),
		reqHandler:  reqHandler,

		client: mg,
		domain: domain,
	}
}

func (h *engineMailgun) Send(ctx context.Context, m *email.Email) (string, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "Send",
		"email_id": m.ID,
	})
	log.Debugf("Sending an email via Mailgun. email_id: %v", m.ID)

	message := h.client.NewMessage(
		fmt.Sprintf("%s <%s>", m.Source.TargetName, m.Source.Target),
		m.Subject,
		m.Content,
	)

	// Add recipients
	for _, d := range m.Destinations {
		message.AddRecipient(fmt.Sprintf("%s <%s>", d.TargetName, d.Target))
	}

	// Custom message-id
	message.AddHeader("X-Voipbin-Message-Id", m.ID.String())

	// Attachments (Mailgun is simpler)
	for _, a := range m.Attachments {
		attach, err := h.getAttachment(ctx, &a)
		if err != nil {
			log.WithField("attachment", a).Errorf("Could not get attachment. err: %v", err)
			continue
		}

		message.AddBufferAttachment(attach.Filename, attach.Bytes)
	}

	// Mailgun timeout recommendation: context with timeout
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	resp, id, err := h.client.Send(message)
	if err != nil {
		return "", errors.Wrapf(err, "could not send mailgun email. message: %v", message)
	}

	log.WithField("mailgun_response", resp).Debugf("Mailgun email sent. message_id: %s", id)

	return id, nil
}

/*
We define a dedicated struct instead of mail.Attachment
*/
type MGAttachment struct {
	Filename string
	Bytes    []byte
}

func (h *engineMailgun) getAttachment(ctx context.Context, e *email.Attachment) (*MGAttachment, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "getAttachment",
		"attachment": e,
	})

	var f *smbucketfile.BucketFile
	var err error

	filename := ""
	switch e.ReferenceType {
	case email.AttachmentReferenceTypeRecording:
		f, err = h.reqHandler.StorageV1RecordingGet(ctx, e.ReferenceID, 60000)
		if err != nil {
			return nil, errors.Wrapf(err, "could not get attachment. type=%s id=%s", e.ReferenceType, e.ReferenceID)
		}
		log.WithField("recording", f).Debugf("Got recording attachment. recording_id: %s", f.ReferenceID)

		filename = fmt.Sprintf("%s.zip", f.ReferenceID)

	default:
		return nil, errors.Errorf("unknown attachment reference type: %v", e.ReferenceType)
	}

	data, err := h.download(ctx, f.DownloadURI)
	if err != nil {
		return nil, errors.Wrapf(err, "could not download attachment")
	}

	return &MGAttachment{
		Filename: filename,
		Bytes:    data,
	}, nil
}

func (h *engineMailgun) download(ctx context.Context, downloadURI string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", downloadURI, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to download file")
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download file: status=%d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read response body")
	}

	return data, nil
}
