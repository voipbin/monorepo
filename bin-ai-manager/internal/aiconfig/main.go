// Package aiconfig wires the stored configuration of bin-ai-manager: the master key, the
// registry, the suppliers that hand the platform LLM keys to the handlers at the time of use, and
// the resolver of the MCP OAuth app of a vendor.
package aiconfig

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-common-handler/pkg/confighandler"
	"monorepo/bin-common-handler/pkg/secretcrypto"

	"monorepo/bin-ai-manager/internal/config"
	"monorepo/bin-ai-manager/internal/configkeys"
)

const (
	// maxSecretPlaintext is the longest secret the cipher accepts. It is the second guard behind
	// the MaxLen of each key and fits the column (2048 bytes minus nonce and tag).
	maxSecretPlaintext = 2020

	// defaultReadTimeout bounds one read of a key. The read runs inside the deadline of the call
	// that needs the key when there is one, so this is only the upper bound.
	defaultReadTimeout = 5 * time.Second

	// geminiURLMarker marks the base URL of the Gemini OpenAI compatible endpoint. It is the rule
	// that picks the key of a provider, and it is written here only.
	geminiURLMarker = "generativelanguage"
)

// ErrKeyUnavailable is returned when a platform key could not be read: a database read error, a
// stored secret that can not be decrypted, or a read that ran out of time. It is never replaced by
// an empty value. A key that is simply not configured is not an error (an empty
// value). The error holds no key text.
var ErrKeyUnavailable = errors.New("the platform LLM key could not be read")

// ErrMCPOAuthIncomplete is returned when the client_secret of an MCP OAuth app has a stored row
// but the client_id has none (or an empty one). The group is then read from the database only, so
// the environment value of the client_id is not used. The error holds no value.
var ErrMCPOAuthIncomplete = errors.New("the MCP OAuth app is stored incompletely")

// KeyFunc hands a platform key to a handler at the time of use. An empty value with no error means
// "not configured".
type KeyFunc = func(ctx context.Context) (string, error)

// New creates the config handler of the service on the given store.
//
// An unparsable master key list never stops the service: the error is logged (without any key
// text) and the handler works without a master key. Secrets can not be written then, and a stored
// secret is reported as undecryptable. The two
// older key lists of this service stop the start on an error; this one does not, so that a
// mistake in a list that is new does not take the LLM calls down.
// The only error is an invalid registry, which is a programming error.
//
// The environment is read with os.LookupEnv for the keys that declare an environment name (the MCP
// OAuth app keys). The platform LLM keys declare none, so they are never read from the environment.
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

	return confighandler.NewHandler(reg, store, crypto, os.LookupEnv, opts...), nil
}

// supplierConfig is the setting of a supplier.
type supplierConfig struct {
	timeout time.Duration
}

// SupplierOption configures Supplier.
type SupplierOption func(*supplierConfig)

// WithTimeout sets the timeout of one read of the key (default 5 seconds).
func WithTimeout(d time.Duration) SupplierOption {
	return func(c *supplierConfig) {
		c.timeout = d
	}
}

// Supplier returns the function that hands the secret of the given key to a handler at the time of
// use. It reads the key on every call and keeps nothing, so a changed key is used by the next call
// on every replica. A source of none gives an empty value. A read error is returned as an error
// (ErrKeyUnavailable) and is never turned into an empty value.
func Supplier(h confighandler.Handler, key string, opts ...SupplierOption) KeyFunc {
	c := &supplierConfig{timeout: defaultReadTimeout}
	for _, opt := range opts {
		opt(c)
	}

	return func(ctx context.Context) (string, error) {
		cctx, cancel := context.WithTimeout(ctx, c.timeout)
		defer cancel()

		secret, _, err := h.Secret(cctx, key)
		if err != nil {
			return "", fmt.Errorf("%w: %w", ErrKeyUnavailable, err)
		}

		return secret.Reveal(), nil
	}
}

// mcpOAuthReadAttempts is the number of times the pair is read when the two reads disagree about
// the source of the group (a write landed between them).
const mcpOAuthReadAttempts = 2

// MCPOAuthClient resolves the (client_id, client_secret) pair of the MCP OAuth app of a vendor.
//
// The pair comes from one source: the database when the client_secret (the anchor of the group) has
// a stored row, the environment otherwise. With a stored client_secret and no stored client_id the
// result is ErrMCPOAuthIncomplete, never the environment client_id. A read error or a stored value
// that can not be decrypted is returned as ErrKeyUnavailable and is never replaced by the
// environment. Neither set is not an error: the result is an empty pair ("not configured").
// An unknown vendor is an error. No value is part of an error text.
func MCPOAuthClient(ctx context.Context, h confighandler.Handler, vendor string, opts ...SupplierOption) (clientID, clientSecret string, err error) {
	secretKey, ok := configkeys.MCPOAuthClientSecretKey(vendor)
	idKey, _ := configkeys.MCPOAuthClientIDKey(vendor)
	if !ok {
		return "", "", fmt.Errorf("unknown MCP OAuth vendor: %q", vendor)
	}

	c := &supplierConfig{timeout: defaultReadTimeout}
	for _, opt := range opts {
		opt(c)
	}

	cctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	for attempt := 0; attempt < mcpOAuthReadAttempts; attempt++ {
		secret, rs, errSecret := h.Secret(cctx, secretKey)
		if errSecret != nil {
			return "", "", fmt.Errorf("%w: %w", ErrKeyUnavailable, errSecret)
		}

		id, ri, errID := h.String(cctx, idKey)
		if errID != nil {
			return "", "", fmt.Errorf("%w: %w", ErrKeyUnavailable, errID)
		}

		switch {
		case rs.Source == confighandler.SourceDB:
			// The group is read from the database only.
			if ri.Source != confighandler.SourceDB || id == "" {
				return "", "", fmt.Errorf("%w: key %s has no stored client_id", ErrMCPOAuthIncomplete, secretKey)
			}

			return id, secret.Reveal(), nil

		case ri.Source == confighandler.SourceDB:
			// The client_secret had no row but the client_id has one: the client_secret row was
			// written between the two reads. Read the pair again.
			continue

		default:
			// Environment or nothing, for both. One side alone is passed on as it is, as before.
			return id, secret.Reveal(), nil
		}
	}

	return "", "", fmt.Errorf("%w: the MCP OAuth app of %q changed while it was read", ErrKeyUnavailable, vendor)
}

// IsGeminiBaseURL tells whether a base URL is the Gemini OpenAI compatible endpoint.
func IsGeminiBaseURL(baseURL string) bool {
	return strings.Contains(baseURL, geminiURLMarker)
}

// KeyForBaseURL returns the supplier of the key that belongs to the provider at the base URL: the
// Gemini key for the Gemini endpoint, the OpenAI compatible key otherwise (the OpenAI default and
// any other compatible provider). It is the one place of this rule.
func KeyForBaseURL(h confighandler.Handler, baseURL string, opts ...SupplierOption) KeyFunc {
	if IsGeminiBaseURL(baseURL) {
		return GeminiKey(h, opts...)
	}

	return Supplier(h, configkeys.KeyOpenAICompatAPIKey, opts...)
}

// GeminiKey returns the supplier of the Gemini key, for the handlers that always call the Gemini
// endpoint (audit and prompt proposal).
func GeminiKey(h confighandler.Handler, opts ...SupplierOption) KeyFunc {
	return Supplier(h, configkeys.KeyGeminiAPIKey, opts...)
}

// Configured tells whether the key has a value now. A read error is returned as an error.
func Configured(ctx context.Context, f KeyFunc) (bool, error) {
	v, err := f(ctx)
	if err != nil {
		return false, err
	}

	return v != "", nil
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
		entry := log.WithFields(logrus.Fields{"key": r.Key, "source": r.Source})

		switch r.Source {
		case confighandler.SourceNone:
			entry.Warnf("The config key is not configured. key: %s", r.Key)
		case confighandler.SourceDBUndecryptable, confighandler.SourceDBInvalid:
			entry.Errorf("The stored config value can not be used. key: %s, source: %s", r.Key, r.Source)
		default:
			entry.Infof("Config key. key: %s, source: %s", r.Key, r.Source)
		}
	}
}
