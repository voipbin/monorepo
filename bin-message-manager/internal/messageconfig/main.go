// Package messageconfig wires the stored configuration of bin-message-manager: the master key,
// the registry and the suppliers that hand a provider token to the
// request layer at the time of use.
package messageconfig

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-common-handler/pkg/confighandler"
	"monorepo/bin-common-handler/pkg/secretcrypto"

	"monorepo/bin-message-manager/internal/config"
	"monorepo/bin-message-manager/internal/configkeys"
)

const (
	// maxSecretPlaintext is the longest secret the cipher accepts. It is the second guard behind
	// the MaxLen of each key and fits the column (2048 bytes minus nonce and tag).
	maxSecretPlaintext = 2020

	// readTimeout bounds the read of one token. A send runs in the background, so the bound only
	// keeps a stuck database from holding a send for ever.
	readTimeout = 2 * time.Second

	// logSourcesTimeout bounds the read of the sources at start up.
	logSourcesTimeout = 5 * time.Second
)

// ErrKeyUnavailable is returned when a token could not be read: a database read error, a stored
// secret that can not be decrypted, or a read that ran out of time. It is never replaced by an
// empty token. A token that is simply not configured is not an error (an empty
// token, as before the configs existed). The error holds no key text.
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

// MessagebirdToken returns the supplier of the MessageBird token.
func MessagebirdToken(h confighandler.Handler) TokenFunc {
	return token(h, configkeys.KeyMessagebirdAuthToken)
}

// TelnyxToken returns the supplier of the Telnyx token. It reads only the key of this service:
// the Telnyx token of bin-number-manager is a different setting and is never consulted.
func TelnyxToken(h confighandler.Handler) TokenFunc {
	return token(h, configkeys.KeyTelnyxAuthToken)
}

// token returns the function that reads one token on every call and keeps nothing, so a changed
// token is used by the next call on every replica. The value is the stored one, or empty
// when the key has no row.
//
// A read error (including a stored secret that can not be decrypted and a read that ran out of
// time) is returned as ErrKeyUnavailable and is never turned into an
// empty token.
func token(h confighandler.Handler, key string) TokenFunc {
	return func(ctx context.Context) (string, error) {
		cctx, cancel := context.WithTimeout(ctx, readTimeout)
		defer cancel()

		secret, _, err := h.Secret(cctx, key)
		if err != nil {
			return "", fmt.Errorf("%w: %w", ErrKeyUnavailable, err)
		}

		return secret.Reveal(), nil
	}
}

// LogSources logs, once, where each key comes from (key name and source only, never a value). The
// tokens are stored in the database only. A Telnyx token that is not configured is a warning,
// because Telnyx is the first provider of every send; a MessageBird token that is not configured is
// an info line, because MessageBird only serves the targets that Telnyx refused. A read error is
// logged as a warning: the service starts anyway.
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
			if r.Key == configkeys.KeyTelnyxAuthToken {
				entry.Warnf("The config key is not configured; it is read from the database only. key: %s", r.Key)
			} else {
				entry.Infof("The config key is not configured; it is read from the database only. key: %s", r.Key)
			}
		case confighandler.SourceDBUndecryptable, confighandler.SourceDBInvalid:
			entry.Errorf("The stored config value can not be used. key: %s, source: %s", r.Key, r.Source)
		default:
			entry.Infof("Config key. key: %s, source: %s", r.Key, r.Source)
		}
	}
}
