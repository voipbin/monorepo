// Package transcribeconfig wires the stored configuration of bin-transcribe-manager: the master
// key, the registry, the environment lookup and the supplier that hands the AWS credential to the
// streaming handler at the time of use.
package transcribeconfig

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-common-handler/pkg/confighandler"
	"monorepo/bin-common-handler/pkg/secretcrypto"

	"monorepo/bin-transcribe-manager/internal/config"
	"monorepo/bin-transcribe-manager/internal/configkeys"
)

const (
	// maxSecretPlaintext is the longest secret the cipher accepts. It is the second guard behind
	// the MaxLen of each key and fits the column (2048 bytes minus nonce and tag).
	maxSecretPlaintext = 2020

	// readTimeout bounds the two reads of the credential together. It stays below the time that a
	// caller waits for a transcribe start.
	readTimeout = 2 * time.Second

	// logSourcesTimeout bounds the read of the sources at start up.
	logSourcesTimeout = 5 * time.Second
)

// ErrKeyUnavailable is returned when the AWS credential could not be read: a database read
// error, a stored secret that can not be decrypted, or a read that ran out of time. It is never
// replaced by the value of the environment. A credential that is simply not configured is not an
// error (an empty pair). The error holds no key text.
var ErrKeyUnavailable = errors.New("the AWS credential could not be read")

// AWSKeysFunc hands the AWS credential to the streaming handler at the time of use. An empty
// pair with no error means "AWS is not available" (not configured, or the two keys are not both
// set from the same source).
type AWSKeysFunc = func(ctx context.Context) (accessKey, secretKey string, err error)

// New creates the config handler of the service on the given store.
//
// An unparsable master key list never stops the service: the error is logged (without any key
// text) and the handler works without a master key. Secrets can not be written then, and a stored
// secret is reported as undecryptable, but the values of the environment keep working.
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

	return confighandler.NewHandler(reg, store, crypto, cfg.EnvValue, opts...), nil
}

// AWSKeys returns the function that hands the AWS credential to the streaming handler at the
// time of use. It reads both keys on every call and keeps nothing, so a changed key is used by
// the next call on every replica.
//
// The two keys are one credential, so a pair is handed out only when both have a value from the
// same source, the database or the environment. Every other state is "AWS is not available" (an
// empty pair, no error): one key only (as before the configs existed), or one key from the
// database and the other from the environment, which would make a pair that was never entered
// together. A state that involves a stored row is logged as a warning (key names and sources
// only); the states of the environment alone keep the debug level they had.
//
// A read error is returned as an error (ErrKeyUnavailable) and is never turned into "not
// available" or into the environment value.
func AWSKeys(h confighandler.Handler) AWSKeysFunc {
	return func(ctx context.Context) (string, string, error) {
		log := logrus.WithField("func", "AWSKeys")

		cctx, cancel := context.WithTimeout(ctx, readTimeout)
		defer cancel()

		access, accessRes, err := h.Secret(cctx, configkeys.KeyAWSAccessKey)
		if err != nil {
			return "", "", fmt.Errorf("%w: %w", ErrKeyUnavailable, err)
		}

		secret, secretRes, err := h.Secret(cctx, configkeys.KeyAWSSecretKey)
		if err != nil {
			return "", "", fmt.Errorf("%w: %w", ErrKeyUnavailable, err)
		}

		accessKey, secretKey := access.Reveal(), secret.Reveal()
		if accessKey != "" && secretKey != "" && accessRes.Source == secretRes.Source {
			return accessKey, secretKey, nil
		}

		entry := log.WithFields(logrus.Fields{
			"access_key_source": accessRes.Source,
			"secret_key_source": secretRes.Source,
		})
		if accessRes.Source == confighandler.SourceDB || secretRes.Source == confighandler.SourceDB {
			entry.Warn("The AWS credential is not available: the two keys are not both set from the same source.")
		} else {
			entry.Debug("The AWS credential is not available: the two keys are not both set.")
		}

		return "", "", nil
	}
}

// LogSources logs, once, where each key comes from (key name and source only, never a value). A
// key that is not configured is an info line, because AWS is an optional provider. A read error
// is logged as a warning: the service starts anyway.
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
		entry := log.WithFields(logrus.Fields{"key": r.Key, "source": r.Source, "env_shadowed": r.EnvShadowed})

		switch r.Source {
		case confighandler.SourceDBUndecryptable, confighandler.SourceDBInvalid:
			entry.Errorf("The stored config value can not be used. key: %s, source: %s", r.Key, r.Source)
		default:
			entry.Infof("Config key. key: %s, source: %s, env_shadowed: %v", r.Key, r.Source, r.EnvShadowed)
		}
	}
}
