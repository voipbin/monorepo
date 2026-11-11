// Package ttsconfig wires the stored configuration of bin-tts-manager: the master key, the
// registry, the environment lookup and the suppliers that hand the provider credentials to the
// handlers at the time of use.
package ttsconfig

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-common-handler/pkg/confighandler"
	"monorepo/bin-common-handler/pkg/secretcrypto"

	"monorepo/bin-tts-manager/internal/config"
	"monorepo/bin-tts-manager/internal/configkeys"
)

const (
	// maxSecretPlaintext is the longest secret the cipher accepts. It is the second guard behind
	// the MaxLen of each key and fits the column (2048 bytes minus nonce and tag).
	maxSecretPlaintext = 2020

	// defaultReadTimeout bounds one read of the keys of a call (the whole pair for AWS).
	defaultReadTimeout = 5 * time.Second
)

// ErrKeyUnavailable is returned when a provider key could not be read: a database read error, a
// stored secret that can not be decrypted, a read that ran out of time, or an AWS pair whose two
// keys come from different sources. It is never replaced by the value of the environment. A key
// that is simply not configured is not an error (an empty value). The error holds no key text.
var ErrKeyUnavailable = errors.New("the TTS provider key could not be read")

// ErrAWSSourceMismatch is wrapped by ErrKeyUnavailable when exactly one of the two AWS keys is
// stored in the database: a stored key and an environment key are never combined.
var ErrAWSSourceMismatch = errors.New("the AWS access key and secret key come from different sources")

// KeyFunc hands one provider key to a handler at the time of use. An empty value with no error
// means "not configured".
type KeyFunc func(ctx context.Context) (string, error)

// AWSCredentialsFunc hands the AWS key pair to a handler at the time of use. Empty values with no
// error mean "not configured".
type AWSCredentialsFunc func(ctx context.Context) (accessKey, secretKey string, err error)

// New creates the config handler of the service on the given store.
//
// An unparsable master key list never stops the service: the error is logged (without any key
// text) and the handler works without a master key. Secrets can not be written then, and a stored
// secret is reported as undecryptable, but the values of the environment keep working for the
// keys that have no stored row. The only error is an invalid registry, which is a programming
// error.
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

// supplierConfig is the setting of a supplier.
type supplierConfig struct {
	timeout time.Duration
}

// SupplierOption configures a supplier.
type SupplierOption func(*supplierConfig)

// WithTimeout sets the timeout of one whole read of a supplier (default 5 seconds).
func WithTimeout(d time.Duration) SupplierOption {
	return func(c *supplierConfig) {
		c.timeout = d
	}
}

func newSupplierConfig(opts []SupplierOption) *supplierConfig {
	c := &supplierConfig{timeout: defaultReadTimeout}
	for _, opt := range opts {
		opt(c)
	}

	return c
}

// ElevenlabsKey returns the supplier of the ElevenLabs key. It reads the key on every call and
// keeps nothing, so a changed key is used by the next call on every replica. A read error is
// returned as an error (ErrKeyUnavailable) and is never turned into an empty value or into the
// environment value.
func ElevenlabsKey(h confighandler.Handler, opts ...SupplierOption) KeyFunc {
	c := newSupplierConfig(opts)

	return func(ctx context.Context) (string, error) {
		cctx, cancel := context.WithTimeout(ctx, c.timeout)
		defer cancel()

		secret, _, err := h.Secret(cctx, configkeys.KeyElevenlabsAPIKey)
		if err != nil {
			return "", fmt.Errorf("%w: %w", ErrKeyUnavailable, err)
		}

		return secret.Reveal(), nil
	}
}

// AWSCredentials returns the supplier of the AWS key pair. It reads both keys on every call with
// one time bound for the whole read, and keeps nothing. When exactly one of the two keys is stored
// in the database it returns ErrKeyUnavailable wrapping ErrAWSSourceMismatch: a stored key and an
// environment key are never combined. Both stored, both from the environment, or none: the pair
// is returned. This is a check of the sources, not a transaction.
func AWSCredentials(h confighandler.Handler, opts ...SupplierOption) AWSCredentialsFunc {
	c := newSupplierConfig(opts)

	return func(ctx context.Context) (string, string, error) {
		cctx, cancel := context.WithTimeout(ctx, c.timeout)
		defer cancel()

		access, accessRes, err := h.Secret(cctx, configkeys.KeyAWSAccessKey)
		if err != nil {
			return "", "", fmt.Errorf("%w: %w", ErrKeyUnavailable, err)
		}

		secret, secretRes, err := h.Secret(cctx, configkeys.KeyAWSSecretKey)
		if err != nil {
			return "", "", fmt.Errorf("%w: %w", ErrKeyUnavailable, err)
		}

		if awsSourcesDiffer(accessRes.Source, secretRes.Source) {
			return "", "", fmt.Errorf("%w: %w", ErrKeyUnavailable, ErrAWSSourceMismatch)
		}

		return access.Reveal(), secret.Reveal(), nil
	}
}

// awsSourcesDiffer tells whether exactly one of the two sources is the database.
func awsSourcesDiffer(a, b confighandler.Source) bool {
	return (a == confighandler.SourceDB) != (b == confighandler.SourceDB)
}

// LogSources logs, once, where each key comes from (key name and source only, never a value). A
// key that is not configured is logged at info level, because every provider key is optional. A
// read error is logged as a warning: the service starts anyway.
func LogSources(ctx context.Context, h confighandler.Handler) {
	log := logrus.WithField("func", "LogSources")

	cctx, cancel := context.WithTimeout(ctx, defaultReadTimeout)
	defer cancel()

	list, err := h.Describe(cctx)
	if err != nil {
		log.Warnf("Could not read the config sources. err: %v", err)
		return
	}

	sources := map[string]confighandler.Source{}
	for _, r := range list {
		sources[r.Key] = r.Source
		entry := log.WithFields(logrus.Fields{"key": r.Key, "source": r.Source, "env_shadowed": r.EnvShadowed})

		switch r.Source {
		case confighandler.SourceDBUndecryptable, confighandler.SourceDBInvalid:
			entry.Errorf("The stored config value can not be used. key: %s, source: %s", r.Key, r.Source)
		default:
			entry.Infof("Config key. key: %s, source: %s, env_shadowed: %v", r.Key, r.Source, r.EnvShadowed)
		}
	}

	if awsSourcesDiffer(sources[configkeys.KeyAWSAccessKey], sources[configkeys.KeyAWSSecretKey]) {
		log.Warnf("The AWS keys come from different sources, so the AWS credentials are refused. Store or clear both. keys: %s, %s", configkeys.KeyAWSAccessKey, configkeys.KeyAWSSecretKey)
	}
}
