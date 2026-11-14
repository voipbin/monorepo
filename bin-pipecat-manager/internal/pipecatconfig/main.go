// Package pipecatconfig wires the stored configuration of bin-pipecat-manager: the master key,
// the registry and the resolver that hands the platform keys of a session to the runner request.
package pipecatconfig

import (
	"context"
	"sort"
	"time"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	amai "monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-common-handler/pkg/confighandler"
	"monorepo/bin-common-handler/pkg/secretcrypto"

	"monorepo/bin-pipecat-manager/internal/config"
	"monorepo/bin-pipecat-manager/internal/configkeys"
)

const (
	// maxSecretPlaintext is the longest secret the cipher accepts. It is the second guard behind
	// the MaxLen of each key and fits the column (2048 bytes minus nonce and tag).
	maxSecretPlaintext = 2020

	// readTimeout bounds all reads of one session together (not one read per key). A key that is
	// not read in time is left out of the request, and the runner rejects a session that needs it.
	readTimeout = 2 * time.Second

	// logSourcesTimeout bounds the read of the sources at start up.
	logSourcesTimeout = 5 * time.Second
)

// New creates the config handler of the service on the given store.
//
// The handler gets no environment lookup and no key declares an environment variable, so the
// database is the only source of a key. A key without a stored row has no value. The runner uses
// the keys of its request only.
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

// KeyResolver resolves the platform keys of a session. It implements
// pipecatcallhandler.PlatformKeyResolver.
type KeyResolver struct {
	h       confighandler.Handler
	timeout time.Duration
}

// NewKeyResolver creates the resolver on a config handler.
func NewKeyResolver(h confighandler.Handler) *KeyResolver {
	return &KeyResolver{h: h, timeout: readTimeout}
}

// Resolve reads the named keys from the database and returns the values by key name.
// It reads on every call and keeps nothing, so a changed key is used by
// the next session on every replica.
//
// It never returns an error and never blocks a session start for longer than the read bound (2
// seconds for all keys together). A key that can not be read (database error, undecryptable
// value, invalid value, out of time) is left out of the result and one warning is logged with the
// session id and the key name and a reason class, never with a value or the error text. The runner
// rejects a session that needs a key that is missing in the request. A key without a value is
// left out silently. An empty result is nil.
func (r *KeyResolver) Resolve(ctx context.Context, pipecatcallID uuid.UUID, keyNames []string) map[string]amai.RedactedString {
	names := uniqueSorted(keyNames)
	if len(names) == 0 {
		return nil
	}

	cctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	var out map[string]amai.RedactedString
	for _, name := range names {
		var secret string
		var err error
		if err = cctx.Err(); err == nil {
			var s confighandler.Secret
			s, _, err = r.h.Secret(cctx, name)
			secret = s.Reveal()
		}
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"func":           "Resolve",
				"pipecatcall_id": pipecatcallID,
				"key":            name,
				"reason":         reasonOf(cctx, err),
			}).Warn("Could not read the platform key. It is left out of the runner request.")
			continue
		}
		if secret == "" {
			continue
		}

		if out == nil {
			out = make(map[string]amai.RedactedString, len(names))
		}
		out[name] = amai.RedactedString(secret)
	}

	return out
}

// reasonOf classifies a read error into a fixed word. The error text is never logged because a
// driver error can hold parts of a connection string.
func reasonOf(ctx context.Context, err error) string {
	switch {
	case errors.Is(err, confighandler.ErrUndecryptable):
		return "undecryptable"
	case errors.Is(err, confighandler.ErrInvalid):
		return "invalid"
	case ctx.Err() != nil:
		return "timeout"
	default:
		return "read_error"
	}
}

func uniqueSorted(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, n := range in {
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		out = append(out, n)
	}
	sort.Strings(out)

	return out
}

// LogSources logs, once, where each key comes from (key name and source only, never a value). A
// key that is not configured is an info line, because every vendor key is optional. A read error
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
		entry := log.WithFields(logrus.Fields{"key": r.Key, "source": r.Source})

		switch r.Source {
		case confighandler.SourceNone:
			entry.Infof("The config key is not set in the database. A session that needs it is rejected. key: %s", r.Key)
		case confighandler.SourceDBUndecryptable, confighandler.SourceDBInvalid:
			entry.Errorf("The stored config value can not be used. key: %s, source: %s", r.Key, r.Source)
		default:
			entry.Infof("Config key. key: %s, source: %s", r.Key, r.Source)
		}
	}
}
