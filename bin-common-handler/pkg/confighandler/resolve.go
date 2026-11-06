package confighandler

import (
	"context"
	"errors"
	"fmt"
	"time"

	"monorepo/bin-common-handler/pkg/secretcrypto"
)

// rowGetter returns the row of a key or ErrNotFound.
type rowGetter func(ctx context.Context, key string) (*Row, error)

// resolution is the internal result of resolving one key.
type resolution struct {
	res    Resolved
	value  any    // typed value of a non secret key; nil when there is none
	secret string // plaintext of a secret key
}

// lookupEnv returns the env value of the key; an empty value is "not set".
func (h *handler) lookupEnv(k Key) (string, bool) {
	if k.Env == "" || h.env == nil {
		return "", false
	}

	v, ok := h.env(k.Env)
	if !ok || v == "" {
		return "", false
	}

	return v, true
}

// resolve resolves one key: db, then env, then default. A store read error (anything but
// ErrNotFound) is returned as is, without a resolution: it is never treated as "no row".
// For undecryptable, invalid and unparsable values the error is returned together with a
// resolution whose Source tells which case it is.
func (h *handler) resolve(ctx context.Context, k Key, get rowGetter) (*resolution, error) {
	r := &resolution{res: Resolved{Key: k.Name, Kind: k.Kind, Type: k.Type, Source: SourceNone}}

	// the row that decides whether the group (or the key itself) is read from the database.
	anchorName := h.reg.anchorOf(k)
	anchorRow, err := get(ctx, anchorName)
	dbMode := true
	if err != nil {
		if !errors.Is(err, ErrNotFound) {
			return nil, err
		}
		dbMode = false
	}

	envVal, envSet := h.lookupEnv(k)

	if dbMode {
		row := anchorRow
		if anchorName != k.Name {
			row, err = get(ctx, k.Name)
			if err != nil {
				if !errors.Is(err, ErrNotFound) {
					return nil, err
				}
				row = nil
			}
		}

		if row != nil {
			// shadowed only when the database is the source and the environment value is set.
			r.res.EnvShadowed = envSet

			return h.fromRow(k, row, r)
		}

		// a member of an atomic group without its own row: default only, never env.
		return fromDefault(k, r), nil
	}

	if envSet {
		return fromEnv(k, envVal, r)
	}

	return fromDefault(k, r), nil
}

func fromDefault(k Key, r *resolution) *resolution {
	if k.Default == nil {
		return r
	}

	r.res.Source = SourceDefault
	r.value = k.Default

	return r
}

func fromEnv(k Key, raw string, r *resolution) (*resolution, error) {
	r.res.Source = SourceEnv

	if k.Kind == KindSecret {
		r.secret = raw
		r.res.HasSecret = true
		return r, nil
	}

	v, err := k.parse(raw)
	if err != nil {
		return r, fmt.Errorf("%w: the environment value of key %q is not valid", ErrInvalid, k.Name)
	}
	r.value = v

	return r, nil
}

func (h *handler) fromRow(k Key, row *Row, r *resolution) (*resolution, error) {
	r.res.Source = SourceDB
	r.res.KeyVersion = row.KeyVersion
	r.res.Revision = row.Revision
	if row.UpdatedBy != nil {
		u := *row.UpdatedBy
		r.res.UpdatedBy = &u
	}
	t := row.TMUpdate
	r.res.TMUpdate = &t

	if row.ValueType != k.rowType() {
		r.res.Source = SourceDBInvalid
		return r, fmt.Errorf("%w: the stored type of key %q does not match its declaration", ErrInvalid, k.Name)
	}

	if k.Kind == KindSecret {
		r.res.HasSecret = true
		if len(row.SecretCiphertext) == 0 {
			r.res.Source = SourceDBInvalid
			return r, fmt.Errorf("%w: the stored secret of key %q is empty", ErrInvalid, k.Name)
		}

		pt, err := h.crypto.Decrypt(row.SecretCiphertext, row.KeyVersion, secretcrypto.AAD(h.table, k.Name))
		if err != nil {
			r.res.Source = SourceDBUndecryptable
			return r, fmt.Errorf("%w: key %q: %w", ErrUndecryptable, k.Name, err)
		}
		r.secret = pt

		return r, nil
	}

	v, err := k.decodeJSON(row.ValueJSON)
	if err != nil {
		r.res.Source = SourceDBInvalid
		return r, fmt.Errorf("%w: the stored value of key %q is not valid", ErrInvalid, k.Name)
	}
	r.value = v

	return r, nil
}

// lookupKey finds the declaration or returns ErrUnknownKey.
func (h *handler) lookupKey(name string) (Key, error) {
	k, ok := h.reg.Lookup(name)
	if !ok {
		return Key{}, fmt.Errorf("%w: %q", ErrUnknownKey, name)
	}

	return k, nil
}

// readTyped resolves a non secret key of the wanted types.
func (h *handler) readTyped(ctx context.Context, name string, types ...Type) (*resolution, error) {
	k, err := h.lookupKey(name)
	if err != nil {
		return nil, err
	}

	ok := k.Kind != KindSecret
	if ok {
		ok = false
		for _, t := range types {
			if k.Type == t {
				ok = true
			}
		}
	}
	if !ok {
		return &resolution{res: Resolved{Key: k.Name, Kind: k.Kind, Type: k.Type, Source: SourceNone}}, fmt.Errorf("%w: key %q", ErrWrongKind, name)
	}

	return h.resolve(ctx, k, h.store.Get)
}

func (h *handler) String(ctx context.Context, key string) (string, Resolved, error) {
	r, err := h.readTyped(ctx, key, TypeString, TypeEnum)
	if r == nil {
		return "", Resolved{Key: key, Source: SourceNone}, err
	}
	if err != nil {
		return "", r.res, err
	}

	s, _ := r.value.(string)
	return s, r.res, nil
}

func (h *handler) Int(ctx context.Context, key string) (int64, Resolved, error) {
	r, err := h.readTyped(ctx, key, TypeInt)
	if r == nil {
		return 0, Resolved{Key: key, Source: SourceNone}, err
	}
	if err != nil {
		return 0, r.res, err
	}

	n, _ := r.value.(int64)
	return n, r.res, nil
}

func (h *handler) Bool(ctx context.Context, key string) (bool, Resolved, error) {
	r, err := h.readTyped(ctx, key, TypeBool)
	if r == nil {
		return false, Resolved{Key: key, Source: SourceNone}, err
	}
	if err != nil {
		return false, r.res, err
	}

	b, _ := r.value.(bool)
	return b, r.res, nil
}

func (h *handler) Duration(ctx context.Context, key string) (time.Duration, Resolved, error) {
	r, err := h.readTyped(ctx, key, TypeDuration)
	if r == nil {
		return 0, Resolved{Key: key, Source: SourceNone}, err
	}
	if err != nil {
		return 0, r.res, err
	}

	d, _ := r.value.(time.Duration)
	return d, r.res, nil
}

func (h *handler) Secret(ctx context.Context, key string) (Secret, Resolved, error) {
	k, err := h.lookupKey(key)
	if err != nil {
		return Secret{}, Resolved{Key: key, Source: SourceNone}, err
	}
	if k.Kind != KindSecret {
		return Secret{}, Resolved{Key: k.Name, Kind: k.Kind, Type: k.Type, Source: SourceNone}, fmt.Errorf("%w: key %q", ErrWrongKind, key)
	}

	r, err := h.resolve(ctx, k, h.store.Get)
	if r == nil {
		return Secret{}, Resolved{Key: k.Name, Kind: k.Kind, Type: k.Type, Source: SourceNone}, err
	}
	if err != nil {
		return Secret{}, r.res, err
	}

	return NewSecret(r.secret), r.res, nil
}

// tolerable reports whether an error only describes the state of a value (undecryptable,
// invalid) so that Describe and the write results can still show the Resolved.
func tolerable(err error) bool {
	return errors.Is(err, ErrUndecryptable) || errors.Is(err, ErrInvalid)
}

func (h *handler) describeOne(ctx context.Context, k Key) (*Resolved, error) {
	r, err := h.resolve(ctx, k, h.store.Get)
	if err != nil && (r == nil || !tolerable(err)) {
		return nil, err
	}

	return &r.res, nil
}

func (h *handler) Describe(ctx context.Context) ([]*Resolved, error) {
	rows, err := h.store.List(ctx)
	if err != nil {
		return nil, err
	}

	byKey := make(map[string]*Row, len(rows))
	for _, row := range rows {
		byKey[row.Key] = row
	}
	get := func(_ context.Context, key string) (*Row, error) {
		if row, ok := byKey[key]; ok {
			return row, nil
		}
		return nil, ErrNotFound
	}

	keys := h.reg.Keys()
	res := make([]*Resolved, 0, len(keys))
	for _, k := range keys {
		r, err := h.resolve(ctx, k, get)
		if err != nil && (r == nil || !tolerable(err)) {
			return nil, err
		}
		res = append(res, &r.res)
	}

	return res, nil
}
