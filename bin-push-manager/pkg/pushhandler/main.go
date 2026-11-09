package pushhandler

//go:generate mockgen -package pushhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"errors"
	"os"
	"strings"

	"github.com/sirupsen/logrus"

	"monorepo/bin-push-manager/models/notification"
)

// Sender delivers one notification to one device token.
type Sender interface {
	// Send delivers the notification. It returns ErrNotConfigured when no credential is available and
	// ErrUnregistered when the provider reports that the token is not valid any more.
	Send(ctx context.Context, token string, kind notification.Kind, data map[string]string) error
}

// sender errors
var (
	ErrNotConfigured = errors.New("the push sender is not configured")
	ErrUnregistered  = errors.New("the push token is not registered")
)

// NewSender returns the FCM sender that uses the service account JSON file. It never fails: when the file
// is not set, can not be read or is not a valid service account, the returned sender reports
// ErrNotConfigured for every send. The content of the file is never logged.
func NewSender(serviceAccountFile string) Sender {
	log := logrus.WithField("func", "NewSender")

	if strings.TrimSpace(serviceAccountFile) == "" {
		log.Warn("FCM is not configured: the service account file is not set.")
		return &notConfiguredSender{}
	}

	content, err := os.ReadFile(serviceAccountFile) //nolint:gosec // G304: the path comes from the service configuration
	if err != nil {
		log.Warn("FCM is not configured: could not read the service account file.")
		return &notConfiguredSender{}
	}

	if strings.TrimSpace(string(content)) == "" {
		log.Warn("FCM is not configured: the service account file is empty.")
		return &notConfiguredSender{}
	}

	res, err := newFCMSenderFromJSON(content)
	if err != nil {
		log.Warn("FCM is not configured: the service account is not valid.")
		return &notConfiguredSender{}
	}

	return res
}

// notConfiguredSender is the sender used when FCM has no credential.
type notConfiguredSender struct{}

// Send always fails with ErrNotConfigured.
func (s *notConfiguredSender) Send(ctx context.Context, token string, kind notification.Kind, data map[string]string) error {
	return ErrNotConfigured
}
