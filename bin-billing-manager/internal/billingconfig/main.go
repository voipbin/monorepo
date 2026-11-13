// Package billingconfig wires the stored configuration of bin-billing-manager: the master key, the
// registry and the suppliers that hand the Paddle credentials to the Paddle handler at the time of
// use.
//
// A credential is resolved in this order: a stored value (the database), then the environment
// variable of the key while no value is stored. A read or decrypt error is an error: it never
// falls through to the environment.
package billingconfig

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-common-handler/pkg/confighandler"
	"monorepo/bin-common-handler/pkg/secretcrypto"

	"monorepo/bin-billing-manager/internal/config"
	"monorepo/bin-billing-manager/internal/configkeys"
	"monorepo/bin-billing-manager/pkg/paddlehandler"
)

const (
	// maxSecretPlaintext is the longest secret the cipher accepts. It is the second guard behind
	// the MaxLen of each key and fits the column (2048 bytes minus nonce and tag).
	maxSecretPlaintext = 2020

	// readTimeout bounds the read of one credential. It stays below the time that a caller waits
	// for a portal session or a webhook answer.
	readTimeout = 2 * time.Second

	// logSourcesTimeout bounds the read of the sources at start up.
	logSourcesTimeout = 5 * time.Second
)

// New creates the config handler of the service on the given store. The environment variable of a
// key is read with os.LookupEnv, so a key without a stored row keeps the value that the container
// environment holds.
//
// An unparsable master key list never stops the service: the error is logged (without any key
// text) and the handler works without a master key. Secrets can not be written then, and a stored
// secret is reported as undecryptable.
// The only error is an invalid registry, which is a programming error.
func New(cfg *config.Config, store confighandler.Store, opts ...confighandler.Option) (confighandler.Handler, error) {
	return newWithEnv(cfg, store, os.LookupEnv, opts...)
}

// newWithEnv is New with the lookup of the environment injected, for tests.
func newWithEnv(cfg *config.Config, store confighandler.Store, env confighandler.EnvLookup, opts ...confighandler.Option) (confighandler.Handler, error) {
	reg, err := configkeys.NewRegistry()
	if err != nil {
		return nil, errors.Wrap(err, "could not create the config registry")
	}

	crypto, errCrypto := secretcrypto.New(cfg.ConfigEncryptionKeys, secretcrypto.WithMaxPlaintext(maxSecretPlaintext))
	if errCrypto != nil {
		logrus.WithField("func", "New").Errorf("The %s value is invalid, so stored secrets can not be used. err: %v", config.EnvConfigEncryptionKeys, errCrypto)
		crypto, _ = secretcrypto.New("", secretcrypto.WithMaxPlaintext(maxSecretPlaintext))
	}

	return confighandler.NewHandler(reg, store, crypto, env, opts...), nil
}

// Supplier returns the function that hands one secret key to the Paddle handler at the time of
// use. It reads the key on every call and keeps nothing, so a changed key is used by the next call
// on every replica.
//
// A key that is not configured is not an error (an empty value). A read error, a stored secret
// that can not be decrypted, and a read that ran out of time are returned as an error
// (paddlehandler.ErrKeyUnavailable) and are never turned into an empty value or into the value of
// the environment.
func Supplier(h confighandler.Handler, key string) paddlehandler.KeyFunc {
	return func(ctx context.Context) (string, error) {
		cctx, cancel := context.WithTimeout(ctx, readTimeout)
		defer cancel()

		secret, _, err := h.Secret(cctx, key)
		if err != nil {
			return "", fmt.Errorf("%w: %w", paddlehandler.ErrKeyUnavailable, err)
		}

		return secret.Reveal(), nil
	}
}

// PaddleKeys returns the suppliers of the API key and of the webhook secret.
func PaddleKeys(h confighandler.Handler) (apiKey, webhookSecret paddlehandler.KeyFunc) {
	return Supplier(h, configkeys.KeyPaddleAPIKey), Supplier(h, configkeys.KeyPaddleWebhookSecretKey)
}

// LogSources logs, once, where each key comes from (key name and source only, never a value). A
// webhook secret that is not configured is a warning, because the signature is not checked then.
// An API key that is not configured is an info line: Paddle is an optional feature of a self
// hosted setup. A stored value that can not be used is an error. A read error is logged as a
// warning: the service starts anyway.
func LogSources(ctx context.Context, h confighandler.Handler) {
	log := logrus.WithField("func", "LogSources")

	cctx, cancel := context.WithTimeout(ctx, logSourcesTimeout)
	defer cancel()

	list, err := h.Describe(cctx)
	if err != nil {
		log.Warnf("Could not read the config sources. err: %v", err)
		return
	}

	for _, r := range list {
		entry := log.WithFields(logrus.Fields{"key": r.Key, "source": r.Source})

		switch r.Source {
		case confighandler.SourceNone:
			if r.Key == configkeys.KeyPaddleWebhookSecretKey {
				entry.Warnf("The config key is not configured, so the webhook signature is not checked. key: %s", r.Key)
			} else {
				entry.Infof("The config key is not configured (optional). key: %s", r.Key)
			}
		case confighandler.SourceDBUndecryptable, confighandler.SourceDBInvalid:
			entry.Errorf("The stored config value can not be used. key: %s, source: %s", r.Key, r.Source)
		default:
			entry.Infof("Config key. key: %s, source: %s", r.Key, r.Source)
		}
	}
}
