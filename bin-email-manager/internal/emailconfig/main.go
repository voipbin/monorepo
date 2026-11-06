// Package emailconfig wires the stored configuration of bin-email-manager: the master key, the
// registry, the environment lookup and the suppliers that hand the provider credentials to the
// email handler at the time of use.
package emailconfig

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-common-handler/pkg/confighandler"
	"monorepo/bin-common-handler/pkg/secretcrypto"

	"monorepo/bin-email-manager/internal/config"
	"monorepo/bin-email-manager/internal/configkeys"
	"monorepo/bin-email-manager/pkg/emailhandler"
)

const (
	// maxSecretPlaintext is the longest secret the cipher accepts. It is the second guard behind
	// the MaxLen of each key and fits the column (2048 bytes minus nonce and tag).
	maxSecretPlaintext = 2020

	// defaultReadTimeout bounds one read of a credential. A send is started with a context
	// without deadline, so a stalled database must not block a send forever.
	defaultReadTimeout = 5 * time.Second
)

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

	return confighandler.NewHandler(reg, store, crypto, cfg.EnvValue, configkeys.Table, opts...), nil
}

// supplierConfig is the setting of a supplier.
type supplierConfig struct {
	timeout time.Duration
}

// SupplierOption configures Supplier.
type SupplierOption func(*supplierConfig)

// WithTimeout sets the timeout of one read of the credential (default 5 seconds).
func WithTimeout(d time.Duration) SupplierOption {
	return func(c *supplierConfig) {
		c.timeout = d
	}
}

// Supplier returns the function that hands the secret of the given key to the email handler at
// the time of use. A source of none gives an empty value, which the email handler treats as "not
// configured". A read error is returned as an error and is never turned into an empty value.
func Supplier(h confighandler.Handler, key string, opts ...SupplierOption) emailhandler.SecretFunc {
	c := &supplierConfig{timeout: defaultReadTimeout}
	for _, opt := range opts {
		opt(c)
	}

	return func(ctx context.Context) (string, error) {
		cctx, cancel := context.WithTimeout(ctx, c.timeout)
		defer cancel()

		secret, _, err := h.Secret(cctx, key)
		if err != nil {
			return "", err
		}

		return secret.Reveal(), nil
	}
}

// LogSources logs, once, where each key comes from (key name and source only, never a value) and
// warns about the keys that are not configured. A read error is logged as a warning: the service
// starts anyway.
func LogSources(ctx context.Context, h confighandler.Handler) {
	log := logrus.WithField("func", "LogSources")

	cctx, cancel := context.WithTimeout(ctx, defaultReadTimeout)
	defer cancel()

	list, err := h.Describe(cctx)
	if err != nil {
		log.Warnf("Could not read the config sources. err: %v", err)
		return
	}

	for _, r := range list {
		entry := log.WithFields(logrus.Fields{"key": r.Key, "source": r.Source, "env_shadowed": r.EnvShadowed})

		switch r.Source {
		case confighandler.SourceNone:
			entry.Warnf("The config key is not configured. key: %s", r.Key)
		case confighandler.SourceDBUndecryptable, confighandler.SourceDBInvalid:
			entry.Errorf("The stored config value can not be used. key: %s, source: %s", r.Key, r.Source)
		default:
			entry.Infof("Config key. key: %s, source: %s, env_shadowed: %v", r.Key, r.Source, r.EnvShadowed)
		}
	}
}
