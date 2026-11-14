// Package billingconfig wires the configuration of bin-billing-manager: the suppliers that hand the
// Paddle credentials to the Paddle handler at the time of use, and the master key and registry of
// the stored settings code.
//
// A credential is read from its environment variable only (PADDLE_API_KEY and
// PADDLE_WEBHOOK_SECRET). The stored settings are not read: the handler of New has no environment
// lookup and no supplier consumes it. A variable that is unset or an unresolved placeholder means
// that the key is not configured (an empty value, no error).
package billingconfig

import (
	"context"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-common-handler/pkg/confighandler"
	"monorepo/bin-common-handler/pkg/envsecret"
	"monorepo/bin-common-handler/pkg/secretcrypto"

	"monorepo/bin-billing-manager/internal/config"
	"monorepo/bin-billing-manager/internal/configkeys"
	"monorepo/bin-billing-manager/pkg/paddlehandler"
)

const (
	// maxSecretPlaintext is the longest secret the cipher accepts. It is the second guard behind
	// the MaxLen of each key and fits the column (2048 bytes minus nonce and tag).
	maxSecretPlaintext = 2020
)

// New creates the config handler of the stored settings (the control commands and the admin API)
// on the given store. The handler has no environment lookup (nil) and no read path of the service
// consumes it.
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

// Supplier returns the function that hands the secret of one environment variable to the Paddle
// handler at the time of use. It reads the variable on every call and keeps nothing.
//
// A variable that is unset, empty or an unresolved placeholder is not configured: an empty value
// and no error. The caller decides what that means (paddlehandler.ErrCredentialNotConfigured).
func Supplier(envName string) paddlehandler.KeyFunc {
	return func(ctx context.Context) (string, error) {
		value, _ := envsecret.Lookup(envName)

		return value, nil
	}
}

// PaddleKeys returns the suppliers of the API key and of the webhook secret.
func PaddleKeys() (apiKey, webhookSecret paddlehandler.KeyFunc) {
	return Supplier(configkeys.EnvPaddleAPIKey), Supplier(configkeys.EnvPaddleWebhookSecret)
}

// LogStates logs, once, the state of each variable (name and state only, never a value). A webhook
// secret that is not usable is a warning, because every Paddle webhook is rejected then. An API key
// that is not set is an info line: Paddle is an optional feature of a self hosted setup. An
// unresolved placeholder is a warning for both.
func LogStates() {
	log := logrus.WithField("func", "LogStates")

	for _, name := range []string{configkeys.EnvPaddleAPIKey, configkeys.EnvPaddleWebhookSecret} {
		_, state := envsecret.Lookup(name)
		entry := log.WithFields(logrus.Fields{"variable": name, "state": state.String()})

		switch {
		case state == envsecret.Placeholder:
			entry.Warnf("The variable holds an unresolved placeholder, so the key is not configured. variable: %s", name)
		case state == envsecret.Unset && name == configkeys.EnvPaddleWebhookSecret:
			entry.Warnf("The variable is not set, so every Paddle webhook is rejected. variable: %s", name)
		case state == envsecret.Unset:
			entry.Infof("The variable is not set (optional), so a portal session can not be created. variable: %s", name)
		default:
			entry.Infof("The variable is set. variable: %s", name)
		}
	}
}
