package emailhandler

//go:generate mockgen -package emailhandler -destination ./mock_engine_mailgun.go -source engine_mailgun.go -build_flags=-mod=mod

import (
	"context"
	"fmt"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-email-manager/models/email"
	smbucketfile "monorepo/bin-storage-manager/models/bucketfile"
	"time"

	"github.com/mailgun/mailgun-go/v4"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	defaultMailgunDomain         = "mailgun.voipbin.net"
	defaultMailgunRequestTimeout = 30 * time.Second
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

func NewEngineMailgun(reqHandler requesthandler.RequestHandler, apiKey string) EngineMailgun {
	mg := mailgun.NewMailgun(defaultMailgunDomain, apiKey)

	return &engineMailgun{
		utilHandler: utilhandler.NewUtilHandler(),
		reqHandler:  reqHandler,

		client: mg,
		domain: defaultMailgunDomain,
	}
}

func (h *engineMailgun) Send(ctx context.Context, m *email.Email) (string, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "Send",
		"email_id": m.ID,
	})
	log.Debugf("Sending an email via Mailgun. email_id: %v", m.ID)

	message := mailgun.NewMessage(
		fmt.Sprintf("%s <%s>", m.Source.TargetName, m.Source.Target),
		m.Subject,
		m.Content,
	)

	for _, d := range m.Destinations {
		if errAdd := message.AddRecipient(fmt.Sprintf("%s <%s>", d.TargetName, d.Target)); errAdd != nil {
			log.Errorf("Could not add recipient: %s <%s>. err: %v", d.TargetName, d.Target, errAdd)
			return "", errors.Wrapf(errAdd, "could not add recipient. target: %s, target_name: %s", d.Target, d.TargetName)
		}
	}

	message.AddHeader("X-Voipbin-Message-Id", m.ID.String())
	for _, a := range m.Attachments {
		filename, data, err := h.getAttachment(ctx, &a)
		if err != nil {
			return "", errors.Wrapf(err, "could not get attachment. reference_type: %s, reference_id: %s", a.ReferenceType, a.ReferenceID)
		}

		message.AddBufferAttachment(filename, data)
	}

	cctx, cancel := context.WithTimeout(ctx, defaultMailgunRequestTimeout)
	defer cancel()

	resp, id, err := h.client.Send(cctx, message)
	if err != nil {
		return "", errors.Wrapf(err, "could not send mailgun email. message: %v", message)
	}

	log.WithField("mailgun_response", resp).Debugf("Mailgun email sent. message_id: %s", id)

	return id, nil
}

func (h *engineMailgun) getAttachment(ctx context.Context, e *email.Attachment) (string, []byte, error) {
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
			return "", nil, errors.Wrapf(err, "could not get attachment. reference_type: %s, reference_id: %s", e.ReferenceType, e.ReferenceID)
		}
		log.WithField("recording", f).Debugf("Got recording attachment. recording_id: %s", f.ReferenceID)

		filename = fmt.Sprintf("%s.zip", f.ReferenceID)

	default:
		return "", nil, errors.Errorf("unknown attachment reference type: %v", e.ReferenceType)
	}

	data, err := download(ctx, f.DownloadURI)
	if err != nil {
		return "", nil, errors.Wrapf(err, "could not download attachment")
	}

	return filename, data, nil
}
