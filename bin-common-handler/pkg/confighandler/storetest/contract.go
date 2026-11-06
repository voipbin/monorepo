package storetest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/pkg/confighandler"
)

// Contract runs the behavior every confighandler.Store must have. newStore returns an empty store;
// histories returns how many history rows that store holds, so that a failed write is shown to
// leave no history behind.
func Contract(t *testing.T, newStore func(t *testing.T) confighandler.Store, histories func(t *testing.T, s confighandler.Store) int) {
	t.Helper()

	ctx := context.Background()
	actor := uuid.Must(uuid.NewV4())
	now := time.Date(2026, 1, 2, 3, 4, 5, 123000000, time.UTC)

	newRow := func(key string) *confighandler.Row {
		return &confighandler.Row{
			Key: key, ValueType: "string", ValueJSON: []byte(`"v1"`), Revision: 1,
			UpdatedBy: &actor, TMCreate: now, TMUpdate: now,
		}
	}
	newHist := func(key, action string) *confighandler.History {
		return &confighandler.History{
			ID: uuid.Must(uuid.NewV4()), Key: key, Action: action, ActorID: actor, Source: "cli", TMCreate: now,
		}
	}

	t.Run("get missing", func(t *testing.T) {
		s := newStore(t)
		if _, err := s.Get(ctx, "nothing"); !errors.Is(err, confighandler.ErrNotFound) {
			t.Errorf("Wrong match. expect: ErrNotFound, got: %v", err)
		}
	})

	t.Run("insert and get round trip", func(t *testing.T) {
		s := newStore(t)

		row := newRow("a.value")
		if err := s.Insert(ctx, row, newHist("a.value", "set")); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}

		secret := &confighandler.Row{
			Key: "a.secret", ValueType: "secret", SecretCiphertext: []byte{1, 2, 3, 4, 5}, KeyVersion: 2, Revision: 1,
			UpdatedBy: &actor, TMCreate: now, TMUpdate: now,
		}
		h := newHist("a.secret", "set")
		h.IsSecret, h.NewKeyVersion = true, 2
		if err := s.Insert(ctx, secret, h); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}

		got, err := s.Get(ctx, "a.value")
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if got.Key != "a.value" || got.ValueType != "string" || string(got.ValueJSON) != `"v1"` || got.Revision != 1 ||
			got.KeyVersion != 0 || got.UpdatedBy == nil || *got.UpdatedBy != actor || len(got.SecretCiphertext) != 0 {
			t.Errorf("Wrong match. got: %+v", got)
		}
		if !got.TMCreate.Equal(now) || !got.TMUpdate.Equal(now) {
			t.Errorf("Wrong match. times: %v %v", got.TMCreate, got.TMUpdate)
		}

		gs, err := s.Get(ctx, "a.secret")
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if gs.ValueType != "secret" || len(gs.ValueJSON) != 0 || string(gs.SecretCiphertext) != string(secret.SecretCiphertext) || gs.KeyVersion != 2 {
			t.Errorf("Wrong match. got: %+v", gs)
		}

		list, err := s.List(ctx)
		if err != nil || len(list) != 2 {
			t.Errorf("Wrong match. expect: 2 rows, got: %d, %v", len(list), err)
		}
		if histories(t, s) != 2 {
			t.Errorf("Wrong match. expect: 2 history rows, got: %d", histories(t, s))
		}
	})

	t.Run("duplicate insert is a conflict and leaves no history", func(t *testing.T) {
		s := newStore(t)
		if err := s.Insert(ctx, newRow("k"), newHist("k", "set")); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		before := histories(t, s)

		if err := s.Insert(ctx, newRow("k"), newHist("k", "set")); !errors.Is(err, confighandler.ErrConflict) {
			t.Errorf("Wrong match. expect: ErrConflict, got: %v", err)
		}
		if histories(t, s) != before {
			t.Errorf("Wrong match. a failed insert left a history row")
		}
	})

	t.Run("update with the right and a stale revision", func(t *testing.T) {
		s := newStore(t)
		if err := s.Insert(ctx, newRow("k"), newHist("k", "set")); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}

		upd := newRow("k")
		upd.ValueJSON = []byte(`"v2"`)
		upd.Revision = 2
		if err := s.Update(ctx, upd, 1, newHist("k", "set")); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		got, _ := s.Get(ctx, "k")
		if got.Revision != 2 || string(got.ValueJSON) != `"v2"` {
			t.Errorf("Wrong match. got: %+v", got)
		}

		before := histories(t, s)
		stale := newRow("k")
		stale.ValueJSON = []byte(`"v3"`)
		if err := s.Update(ctx, stale, 1, newHist("k", "set")); !errors.Is(err, confighandler.ErrConflict) {
			t.Errorf("Wrong match. expect: ErrConflict, got: %v", err)
		}
		got, _ = s.Get(ctx, "k")
		if got.Revision != 2 || string(got.ValueJSON) != `"v2"` {
			t.Errorf("Wrong match. a stale update changed the row: %+v", got)
		}
		if histories(t, s) != before {
			t.Errorf("Wrong match. a stale update left a history row")
		}

		if err := s.Update(ctx, newRow("missing"), 1, newHist("missing", "set")); !errors.Is(err, confighandler.ErrConflict) {
			t.Errorf("Wrong match. missing row expect: ErrConflict, got: %v", err)
		}
	})

	t.Run("delete with the right and a stale revision", func(t *testing.T) {
		s := newStore(t)
		if err := s.Insert(ctx, newRow("k"), newHist("k", "set")); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}

		before := histories(t, s)
		if err := s.Delete(ctx, "k", 7, newHist("k", "clear")); !errors.Is(err, confighandler.ErrConflict) {
			t.Errorf("Wrong match. expect: ErrConflict, got: %v", err)
		}
		if _, err := s.Get(ctx, "k"); err != nil {
			t.Errorf("Wrong match. a stale delete removed the row: %v", err)
		}
		if histories(t, s) != before {
			t.Errorf("Wrong match. a stale delete left a history row")
		}

		if err := s.Delete(ctx, "k", 1, newHist("k", "clear")); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if _, err := s.Get(ctx, "k"); !errors.Is(err, confighandler.ErrNotFound) {
			t.Errorf("Wrong match. expect: ErrNotFound, got: %v", err)
		}
		if histories(t, s) != before+1 {
			t.Errorf("Wrong match. the delete did not write its history row")
		}
	})
}
