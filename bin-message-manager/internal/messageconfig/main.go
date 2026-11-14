// Package messageconfig wires the configuration of bin-message-manager: the handler of the stored
// settings (the master key and the registry, used by the config commands and the admin API) and the
// suppliers that hand a provider token to the request layer at the time of use. The tokens come
// from the environment only; the stored settings are not read.
package messageconfig

import (
	"context"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-common-handler/pkg/confighandler"
	"monorepo/bin-common-handler/pkg/envsecret"
	"monorepo/bin-common-handler/pkg/secretcrypto"

	"monorepo/bin-message-manager/internal/config"
	"monorepo/bin-message-manager/internal/configkeys"
)

// maxSecretPlaintext is the longest secret the cipher accepts. It is the second guard behind the
// MaxLen of each key and fits the column (2048 bytes minus nonce and tag).
const maxSecretPlaintext = 2020

// ErrKeyUnavailable stays for the stored settings code until its removal. The environment read can
// not fail, so the suppliers never return it.
var ErrKeyUnavailable = errors.New("the provider token could not be read")

// TokenFunc hands a provider token to the request layer at the time of use. An empty token with
// no error means that the token is not configured.
type TokenFunc = func(ctx context.Context) (string, error)

// New creates the config handler of the service on the given store.
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

// MessagebirdToken returns the supplier of the MessageBird token. The token is the value of
// MESSAGEBIRD_API_KEY at the time of use; the stored value is not read.
func MessagebirdToken() TokenFunc {
	return token(configkeys.EnvMessagebirdAPIKey)
}

// TelnyxToken returns the supplier of the Telnyx token (TELNYX_API_KEY, the variable that
// bin-number-manager reads too).
func TelnyxToken() TokenFunc {
	return token(configkeys.EnvTelnyxAPIKey)
}

// token returns the function that reads one variable on every call and keeps nothing, so a changed
// variable is used by the next call after a restart of the replica. A variable that is not set,
// is empty or holds an unresolved placeholder is not configured: an empty token and no error.
func token(name string) TokenFunc {
	return func(ctx context.Context) (string, error) {
		value, _ := envsecret.Lookup(name)

		return value, nil
	}
}

// LogStates logs, once, the state of each variable (name and state only, never a value). A Telnyx
// variable that is not set is a warning, because Telnyx is the first provider of every send; a
// MessageBird variable that is not set is an info line, because MessageBird only serves the targets
// that Telnyx refused. A placeholder (the text of an unresolved deployment variable) is a warning.
func LogStates() {
	log := logrus.WithField("func", "LogStates")

	for _, name := range []string{configkeys.EnvMessagebirdAPIKey, configkeys.EnvTelnyxAPIKey} {
		_, state := envsecret.Lookup(name)
		entry := log.WithFields(logrus.Fields{"variable": name, "state": state.String()})

		switch state {
		case envsecret.Set:
			entry.Infof("The variable is set. variable: %s", name)
		case envsecret.Placeholder:
			entry.Warnf("The variable holds an unresolved placeholder. variable: %s", name)
		default:
			if name == configkeys.EnvTelnyxAPIKey {
				entry.Warnf("The variable is not set. variable: %s", name)
			} else {
				entry.Infof("The variable is not set. variable: %s", name)
			}
		}
	}
}
