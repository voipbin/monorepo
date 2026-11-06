package storetest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/pkg/confighandler"
)
// leave no history behind.
// contractEnv holds what every case of the contract needs.
type contractEnv struct {
	ctx       context.Context
	actor     uuid.UUID
	now       time.Time
	newStore  func(t *testing.T) confighandler.Store
	histories func(t *testing.T, s confighandler.Store) int
}

func (e *contractEnv) newRow(key string) *confighandler.Row {
	return &confighandler.Row{
		Key: key, ValueType: "string", ValueJSON: []byte(`"v1"`), Revision: 1,
		UpdatedBy: &e.actor, TMCreate: e.now, TMUpdate: e.now,
	}
}

func (e *contractEnv) newHist(key, action string) *confighandler.History {
	return &confighandler.History{
		ID: uuid.Must(uuid.NewV4()), Key: key, Action: action, ActorID: e.actor, Source: "cli", TMCreate: e.now,
	}
}

// Contract runs the behavior every confighandler.Store must have. newStore returns an empty store;
// histories returns how many history rows that store holds, so that a failed write is shown to
// leave no history behind.
func Contract(t *testing.T, newStore func(t *testing.T) confighandler.Store, histories func(t *testing.T, s confighandler.Store) int) {
	t.Helper()

	e := &contractEnv{
		ctx:       context.Background(),
		actor:     uuid.Must(uuid.NewV4()),
		now:       time.Date(2026, 1, 2, 3, 4, 5, 123000000, time.UTC),
		newStore:  newStore,
		histories: histories,
	}

	t.Run("get missing", e.getMissing)
	t.Run("insert and get round trip", e.insertAndGet)
	t.Run("duplicate insert is a conflict and leaves no history", e.duplicateInsert)
	t.Run("update with the right and a stale revision", e.updateRevision)
	t.Run("delete with the right and a stale revision", e.deleteRevision)
}

func (e *contractEnv) getMissing(t *testing.T) {
	s := e.newStore(t)
	if _, err := s.Get(e.ctx, "nothing"); !errors.Is(err, confighandler.ErrNotFound) {
		t.Errorf("Wrong match. expect: ErrNotFound, got: %v", err)
	}
}

func (e *contractEnv) insertAndGet(t *testing.T) {
	s := e.newStore(t)

	row := e.newRow("a.value")
	if err := s.Insert(e.ctx, row, e.newHist("a.value", "set")); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	secret := &confighandler.Row{
		Key: "a.secret", ValueType: "secret", SecretCiphertext: []byte{1, 2, 3, 4, 5}, KeyVersion: 2, Revision: 1,
		UpdatedBy: &e.actor, TMCreate: e.now, TMUpdate: e.now,
	}
	h := e.newHist("a.secret", "set")
	h.IsSecret, h.NewKeyVersion = true, 2
	if err := s.Insert(e.ctx, secret, h); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	e.checkRoundTrip(t, s, secret)
}

// checkRoundTrip reads the two rows written by insertAndGet and compares every field.
func (e *contractEnv) checkRoundTrip(t *testing.T, s confighandler.Store, secret *confighandler.Row) {
	t.Helper()

	got, err := s.Get(e.ctx, "a.value")
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if got.Key != "a.value" || got.ValueType != "string" || string(got.ValueJSON) != `"v1"` || got.Revision != 1 ||
		got.KeyVersion != 0 || got.UpdatedBy == nil || *got.UpdatedBy != e.actor || len(got.SecretCiphertext) != 0 {
		t.Errorf("Wrong match. got: %+v", got)
	}
	if !got.TMCreate.Equal(e.now) || !got.TMUpdate.Equal(e.now) {
		t.Errorf("Wrong match. times: %v %v", got.TMCreate, got.TMUpdate)
	}

	gs, err := s.Get(e.ctx, "a.secret")
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if gs.ValueType != "secret" || len(gs.ValueJSON) != 0 || string(gs.SecretCiphertext) != string(secret.SecretCiphertext) || gs.KeyVersion != 2 {
		t.Errorf("Wrong match. got: %+v", gs)
	}

	list, err := s.List(e.ctx)
	if err != nil || len(list) != 2 {
		t.Errorf("Wrong match. expect: 2 rows, got: %d, %v", len(list), err)
	}
	if e.histories(t, s) != 2 {
		t.Errorf("Wrong match. expect: 2 history rows, got: %d", e.histories(t, s))
	}
}

func (e *contractEnv) duplicateInsert(t *testing.T) {
	s := e.newStore(t)
	if err := s.Insert(e.ctx, e.newRow("k"), e.newHist("k", "set")); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	before := e.histories(t, s)

	if err := s.Insert(e.ctx, e.newRow("k"), e.newHist("k", "set")); !errors.Is(err, confighandler.ErrConflict) {
		t.Errorf("Wrong match. expect: ErrConflict, got: %v", err)
	}
	if e.histories(t, s) != before {
		t.Errorf("Wrong match. a failed insert left a history row")
	}
}

func (e *contractEnv) updateRevision(t *testing.T) {
	s := e.newStore(t)
	if err := s.Insert(e.ctx, e.newRow("k"), e.newHist("k", "set")); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	upd := e.newRow("k")
	upd.ValueJSON = []byte(`"v2"`)
	upd.Revision = 2
	if err := s.Update(e.ctx, upd, 1, e.newHist("k", "set")); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	got, _ := s.Get(e.ctx, "k")
	if got.Revision != 2 || string(got.ValueJSON) != `"v2"` {
		t.Errorf("Wrong match. got: %+v", got)
	}

	before := e.histories(t, s)
	stale := e.newRow("k")
	stale.ValueJSON = []byte(`"v3"`)
	if err := s.Update(e.ctx, stale, 1, e.newHist("k", "set")); !errors.Is(err, confighandler.ErrConflict) {
		t.Errorf("Wrong match. expect: ErrConflict, got: %v", err)
	}
	got, _ = s.Get(e.ctx, "k")
	if got.Revision != 2 || string(got.ValueJSON) != `"v2"` {
		t.Errorf("Wrong match. a stale update changed the row: %+v", got)
	}
	if e.histories(t, s) != before {
		t.Errorf("Wrong match. a stale update left a history row")
	}

	if err := s.Update(e.ctx, e.newRow("missing"), 1, e.newHist("missing", "set")); !errors.Is(err, confighandler.ErrConflict) {
		t.Errorf("Wrong match. missing row expect: ErrConflict, got: %v", err)
	}
}

func (e *contractEnv) deleteRevision(t *testing.T) {
	s := e.newStore(t)
	if err := s.Insert(e.ctx, e.newRow("k"), e.newHist("k", "set")); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	before := e.histories(t, s)
	if err := s.Delete(e.ctx, "k", 7, e.newHist("k", "clear")); !errors.Is(err, confighandler.ErrConflict) {
		t.Errorf("Wrong match. expect: ErrConflict, got: %v", err)
	}
	if _, err := s.Get(e.ctx, "k"); err != nil {
		t.Errorf("Wrong match. a stale delete removed the row: %v", err)
	}
	if e.histories(t, s) != before {
		t.Errorf("Wrong match. a stale delete left a history row")
	}

	if err := s.Delete(e.ctx, "k", 1, e.newHist("k", "clear")); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if _, err := s.Get(e.ctx, "k"); !errors.Is(err, confighandler.ErrNotFound) {
		t.Errorf("Wrong match. expect: ErrNotFound, got: %v", err)
	}
	if e.histories(t, s) != before+1 {
		t.Errorf("Wrong match. the delete did not write its history row")
	}
}
