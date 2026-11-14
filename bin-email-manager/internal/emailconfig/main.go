// Package emailconfig wires the configuration of bin-email-manager: the suppliers that hand the
// provider credentials to the email handler at the time of use, and the master key and registry of
// the stored settings code.
//
// A credential is read from its environment variable only (SENDGRID_API_KEY, MAILGUN_API_KEY,
// MAILGUN_WEBHOOK_SIGNING_KEY). The stored settings are not read.
package emailconfig

import (
	"context"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-common-handler/pkg/confighandler"
	"monorepo/bin-common-handler/pkg/envsecret"
	"monorepo/bin-common-handler/pkg/secretcrypto"

	"monorepo/bin-email-manager/internal/config"
	"monorepo/bin-email-manager/internal/configkeys"
	"monorepo/bin-email-manager/pkg/emailhandler"
)

const (
	// maxSecretPlaintext is the longest secret the cipher accepts. It is the second guard behind
	// the MaxLen of each key and fits the column (2048 bytes minus nonce and tag).
	maxSecretPlaintext = 2020
)

// New creates the config handler of the stored settings (the control commands and the admin API)
// on the given store. No read path of the service consumes it.
//
// An unparsable master key list never stops the service: the error is logged (without any key
// text) and the handler works without a master key. Secrets can not be written then, and a stored
// secret is reported as undecryptable.
// The only error is an invalid registry, which is a programming error.
func New(cfg *config.Config, store confighandler.Store, opts ...confighandler.Option) (confighandler.Handler, error) {
	reg, err := configkeys.NewRegistry()
	if err != nil {
		return nil, errors.Wrap(err, "could not create the config registry")
	}

	crypto, errCrypto := secretcrypto.New(cfg.ConfigEncryptionKeys, secretcrypto.WithMaxPlaintext(maxSecretPlaintext))
	if errCrypto != nil {
		logrus.WithField("func", "New").Errorf("The %s value is invalid, so stored secrets can not be used. err: %v", config.EnvConfigEncryptionKeys, errCrypto)
		crypto, _ = secretcrypto.New("", secretcrypto.WithMaxPlaintext(maxSecretPlaintext))
	}

	return confighandler.NewHandler(reg, store, crypto, nil, opts...), nil
}

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
