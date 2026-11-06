// Package confighandler resolves typed configuration keys of a service from its database, its
// environment and the code defaults, in this order: db, env, default.
//
// A service declares its keys once in a Registry (name, kind, type, range, env name, default).
// The database only stores values that passed the registry, in two tables of the service: one row
// per key and one history row per change. The package is not a feature flag store: a bool key is
// accepted only as a provider selection inside an atomic group.
//
// Secrets are encrypted with the shared secretcrypto package; a stored ciphertext is not bound to
// its row.
// A secret that is stored but cannot be decrypted is never replaced by the environment value:
// the read fails with ErrUndecryptable and the resolved source is db_undecryptable.
//
// History rows never carry a secret value, a hash, a length, or any other derived value: only the
// action, the actor, the key versions and the time.
package confighandler

//go:generate mockgen -package confighandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"errors"
	"time"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/pkg/secretcrypto"
)

// Source tells where a resolved value comes from.
type Source string

// list of sources
const (
	SourceDB              Source = "db"
	SourceEnv             Source = "env"
	SourceDefault         Source = "default"
	SourceNone            Source = "none"
	SourceDBUndecryptable Source = "db_undecryptable"
	SourceDBInvalid       Source = "db_invalid"
)

// Errors. Messages never contain a value.
var (
	// ErrNotFound is returned by a Store when the row does not exist.
	ErrNotFound = errors.New("config row not found")

	// ErrConflict is returned by a Store when the row changed since it was read (or exists already
	// on insert), and by the Handler when the retries are used up.
	ErrConflict = errors.New("config row changed concurrently")

	// ErrUnknownKey is returned when the key is not in the registry.
	ErrUnknownKey = errors.New("unknown config key")

	// ErrWrongKind is returned when an operation does not match the kind or type of the key.
	ErrWrongKind = errors.New("config key kind or type does not match the operation")

	// ErrUndecryptable is returned when a stored secret can not be decrypted. It wraps the
	// secretcrypto error.
	ErrUndecryptable = errors.New("stored secret can not be decrypted")

	// ErrInvalid is returned for a value that is malformed or violates the declaration of the key.
	ErrInvalid = errors.New("invalid config value")

	// ErrKeyNotConfigured is returned when a secret is written and no master key exists.
	ErrKeyNotConfigured = secretcrypto.ErrKeyNotConfigured

	// ErrTooLong is returned when a secret is longer than the key allows.
	ErrTooLong = secretcrypto.ErrTooLong
)

// EnvLookup returns the value of an environment variable. A missing or empty value means "not set".
type EnvLookup func(name string) (string, bool)

// Resolved describes where the value of a key comes from. It never holds a value.
type Resolved struct {
	Key  string
	Kind Kind
	Type Type

	Source      Source
	HasSecret   bool // a secret value exists (database, env)
	EnvShadowed bool // the source is the database and the env value is set but not used
	KeyVersion  int
	Revision    uint32
	UpdatedBy   *uuid.UUID
	TMUpdate    *time.Time
}

// Handler reads and writes the configuration of one service.
//
// Reads return a Resolved even when they return an error, so a caller can report
// db_undecryptable and db_invalid. A secret key with source none returns an empty Secret and no
// error; the caller decides what a missing secret means.
type Handler interface {
	String(ctx context.Context, key string) (string, Resolved, error)
	Int(ctx context.Context, key string) (int64, Resolved, error)
	Bool(ctx context.Context, key string) (bool, Resolved, error)
	Duration(ctx context.Context, key string) (time.Duration, Resolved, error)
	Secret(ctx context.Context, key string) (Secret, Resolved, error)

	// Set, SetSecret and Clear are overwrites: the last writer wins and every write has its own
	// history row. actor must not be uuid.Nil.
	Set(ctx context.Context, actor uuid.UUID, key, raw string) (*Resolved, error)
	SetSecret(ctx context.Context, actor uuid.UUID, key, secret string) (*Resolved, error)
	Clear(ctx context.Context, actor uuid.UUID, key string) (*Resolved, error)

	// Describe lists every registered key in registry order, with or without a row, and never a
	// value. Rows of keys that are not registered are ignored.
	Describe(ctx context.Context) ([]*Resolved, error)

	Registry() *Registry
}

// Option configures NewHandler.
type Option func(*handler)

// WithSource sets the source recorded in the history rows ("cli" or "api"). Default "api".
func WithSource(s string) Option {
	return func(h *handler) {
		h.source = s
	}
}

const (
	defaultHistorySource = "api"
	maxWriteAttempts     = 3

	actionSet   = "set"
	actionClear = "clear"
)

type handler struct {
	reg    *Registry
	store  Store
	crypto *secretcrypto.Crypto
	env    EnvLookup
	source string

	now   func() time.Time
	newID func() uuid.UUID
}

// NewHandler creates a Handler. crypto may be nil or hold no key: secrets can not be written, and a
// stored secret is reported as undecryptable.
func NewHandler(reg *Registry, store Store, crypto *secretcrypto.Crypto, env EnvLookup, opts ...Option) Handler {
	h := &handler{
		reg:    reg,
		store:  store,
		crypto: crypto,
		env:    env,
		source: defaultHistorySource,

		now:   time.Now,
		newID: func() uuid.UUID { return uuid.Must(uuid.NewV4()) },
	}
	for _, opt := range opts {
		opt(h)
	}

	return h
}

func (h *handler) Registry() *Registry {
	return h.reg
}
