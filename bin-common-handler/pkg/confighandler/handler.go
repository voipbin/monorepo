package confighandler

import (
	"context"
	"errors"
	"fmt"

	"github.com/gofrs/uuid"
)

// writeFn builds the row and the history for one attempt from the freshly read row (nil when the
// key has no row yet).
type writeFn func(existing *Row) (*Row, *History)

// write stores a change with the optimistic lock. A conflict re-reads the row and builds the row
// and the history again, at most maxWriteAttempts times. The concurrent writers are last-writer-wins
// and every write keeps its own history row.
func (h *handler) write(ctx context.Context, k Key, build writeFn) (*Resolved, error) {
	for i := 0; i < maxWriteAttempts; i++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		existing, err := h.store.Get(ctx, k.Name)
		if err != nil {
			if !errors.Is(err, ErrNotFound) {
				return nil, err
			}
			existing = nil
		}

		row, hist := build(existing)

		if existing == nil {
			row.Revision = 1
			err = h.store.Insert(ctx, row, hist)
		} else {
			row.Revision = existing.Revision + 1
			row.TMCreate = existing.TMCreate
			err = h.store.Update(ctx, row, existing.Revision, hist)
		}
		if errors.Is(err, ErrConflict) {
			continue
		}
		if err != nil {
			return nil, err
		}

		return h.describeStored(ctx, k)
	}

	return nil, fmt.Errorf("%w: key %q", ErrConflict, k.Name)
}

// describeStored reads back the result of a write that is already committed. A failure here must
// not look like a failed write, because a caller may retry a failed write.
func (h *handler) describeStored(ctx context.Context, k Key) (*Resolved, error) {
	res, err := h.describeOne(ctx, k)
	if err != nil {
		return nil, fmt.Errorf("the change of key %q is stored, but reading it back failed: %w", k.Name, err)
	}

	return res, nil
}

func (h *handler) newHistory(k Key, action string, actor uuid.UUID) *History {
	return &History{
		ID:       h.newID(),
		Key:      k.Name,
		Action:   action,
		IsSecret: k.hidden(),
		ActorID:  actor,
		Source:   h.source,
		TMCreate: h.now(),
	}
}

func (h *handler) Set(ctx context.Context, actor uuid.UUID, key, raw string) (*Resolved, error) {
	if actor == uuid.Nil {
		return nil, fmt.Errorf("%w: the actor is required", ErrInvalid)
	}
	k, err := h.lookupKey(key)
	if err != nil {
		return nil, err
	}
	if k.Kind == KindSecret {
		return nil, fmt.Errorf("%w: key %q is a secret", ErrWrongKind, key)
	}

	typed, err := k.parse(raw)
	if err != nil {
		return nil, fmt.Errorf("%w: the value of key %q does not fit its declaration", ErrInvalid, key)
	}
	encoded, err := k.encodeJSON(typed)
	if err != nil {
		return nil, fmt.Errorf("%w: the value of key %q can not be stored", ErrInvalid, key)
	}

	return h.write(ctx, k, func(existing *Row) (*Row, *History) {
		now := h.now()
		a := actor

		row := &Row{
			Key:       k.Name,
			ValueType: k.rowType(),
			ValueJSON: encoded,
			UpdatedBy: &a,
			TMCreate:  now,
			TMUpdate:  now,
		}

		hist := h.newHistory(k, actionSet, actor)
		hist.TMCreate = now
		if !hist.IsSecret {
			hist.NewValueJSON = encoded
			if existing != nil && existing.ValueType != secretRowType {
				hist.OldValueJSON = existing.ValueJSON
			}
		}

		return row, hist
	})
}

func (h *handler) SetSecret(ctx context.Context, actor uuid.UUID, key, secret string) (*Resolved, error) {
	if actor == uuid.Nil {
		return nil, fmt.Errorf("%w: the actor is required", ErrInvalid)
	}
	k, err := h.lookupKey(key)
	if err != nil {
		return nil, err
	}
	if k.Kind != KindSecret {
		return nil, fmt.Errorf("%w: key %q is not a secret", ErrWrongKind, key)
	}
	if secret == "" {
		return nil, fmt.Errorf("%w: the secret of key %q is empty", ErrInvalid, key)
	}
	if len(secret) > k.MaxLen {
		return nil, fmt.Errorf("%w: key %q", ErrTooLong, key)
	}

	blob, version, err := h.crypto.Encrypt(secret)
	if err != nil {
		// the library errors carry no secret text
		return nil, fmt.Errorf("could not encrypt the secret of key %q: %w", key, err)
	}

	return h.write(ctx, k, func(existing *Row) (*Row, *History) {
		now := h.now()
		a := actor

		row := &Row{
			Key:              k.Name,
			ValueType:        k.rowType(),
			SecretCiphertext: blob,
			KeyVersion:       version,
			UpdatedBy:        &a,
			TMCreate:         now,
			TMUpdate:         now,
		}

		// a secret history row has no value fields: only the key versions.
		hist := h.newHistory(k, actionSet, actor)
		hist.TMCreate = now
		hist.NewKeyVersion = version
		if existing != nil {
			hist.OldKeyVersion = existing.KeyVersion
		}

		return row, hist
	})
}

func (h *handler) Clear(ctx context.Context, actor uuid.UUID, key string) (*Resolved, error) {
	if actor == uuid.Nil {
		return nil, fmt.Errorf("%w: the actor is required", ErrInvalid)
	}
	k, err := h.lookupKey(key)
	if err != nil {
		return nil, err
	}

	for i := 0; i < maxWriteAttempts; i++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		existing, err := h.store.Get(ctx, k.Name)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				// nothing to clear: no write and no history.
				return h.describeOne(ctx, k)
			}
			return nil, err
		}

		hist := h.newHistory(k, actionClear, actor)
		if k.Kind == KindSecret {
			hist.OldKeyVersion = existing.KeyVersion
		} else if !hist.IsSecret && existing.ValueType != secretRowType {
			hist.OldValueJSON = existing.ValueJSON
		}

		err = h.store.Delete(ctx, k.Name, existing.Revision, hist)
		if errors.Is(err, ErrConflict) {
			continue
		}
		if err != nil {
			return nil, err
		}

		return h.describeStored(ctx, k)
	}

	return nil, fmt.Errorf("%w: key %q", ErrConflict, k.Name)
}
