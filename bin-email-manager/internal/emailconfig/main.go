// Package emailconfig wires the configuration of bin-email-manager: the suppliers that hand the
// provider credentials to the email handler at the time of use.
//
// A credential is read from its environment variable only (SENDGRID_API_KEY, MAILGUN_API_KEY,
// MAILGUN_WEBHOOK_SIGNING_KEY).
package emailconfig

import (
	"context"

	"github.com/sirupsen/logrus"

	"monorepo/bin-common-handler/pkg/envsecret"

	"monorepo/bin-email-manager/internal/configkeys"
	"monorepo/bin-email-manager/pkg/emailhandler"
)

// Supplier returns the function that hands the secret of one environment variable to the email
// handler at the time of use. It reads the variable on every call and keeps nothing. A variable that
// is unset, empty or an unresolved placeholder is not configured: an empty value and no error,
// which the email handler treats as "not configured".
func Supplier(envName string) emailhandler.SecretFunc {
	return func(ctx context.Context) (string, error) {
		value, _ := envsecret.Lookup(envName)

		return value, nil
	}
}

// Suppliers returns the suppliers of the SendGrid API key, the Mailgun API key and the Mailgun
// webhook signing key, in the argument order of emailhandler.NewEmailHandler. It is the one place
// that pairs a supplier with its variable.
func Suppliers() (sendgrid, mailgun, mailgunSigning emailhandler.SecretFunc) {
	return Supplier(configkeys.EnvSendgridAPIKey), Supplier(configkeys.EnvMailgunAPIKey), Supplier(configkeys.EnvMailgunWebhookSigningKey)
}

// LogStates logs, once, the state of each variable (name and state only, never a value). A variable
// that is not set and a placeholder are warnings: the provider that needs the key can not be used.
func LogStates() {
	log := logrus.WithField("func", "LogStates")

	for _, name := range []string{configkeys.EnvSendgridAPIKey, configkeys.EnvMailgunAPIKey, configkeys.EnvMailgunWebhookSigningKey} {
		_, state := envsecret.Lookup(name)
		entry := log.WithFields(logrus.Fields{"variable": name, "state": state.String()})

		switch state {
		case envsecret.Placeholder:
			entry.Warnf("The variable holds an unresolved placeholder, so the key is not configured. variable: %s", name)
		case envsecret.Unset:
			entry.Warnf("The variable is not set, so the key is not configured. variable: %s", name)
		default:
			entry.Infof("The variable is set. variable: %s", name)
		}
	}
}
