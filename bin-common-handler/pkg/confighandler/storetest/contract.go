package storetest

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/pkg/confighandler"
)

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
	t.Run("list histories newest first with a cursor", e.listHistories)
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

// listHistories checks the read of the history: the newest first, the cursor (time, id) that also
// separates rows of the same time, the other keys left out, and the stored fields.
func (e *contractEnv) listHistories(t *testing.T) {
	s := e.newStore(t)

	at := func(sec int) time.Time {
		return e.now.Add(time.Duration(sec) * time.Second)
	}
	id := func(n int) uuid.UUID {
		return uuid.FromStringOrNil(fmt.Sprintf("00000000-0000-4000-8000-%012d", n))
	}
	hist := func(key, action string, n, sec int) *confighandler.History {
		h := e.newHist(key, action)
		h.ID, h.TMCreate = id(n), at(sec)
		h.IsSecret, h.NewKeyVersion, h.Source = true, 2, "api"
		return h
	}

	// "k": ids 1, 2, 3 share a time, id 4 is later, id 5 is earlier. "other": id 6.
	if err := s.Insert(e.ctx, e.newRow("k"), hist("k", "set", 1, 10)); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	for i, h := range []*confighandler.History{hist("k", "set", 2, 10), hist("k", "set", 3, 10), hist("k", "clear", 4, 20), hist("k", "set", 5, 5)} {
		upd := e.newRow("k")
		upd.Revision = uint32(i) + 2
		if err := s.Update(e.ctx, upd, uint32(i)+1, h); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
	}
	if err := s.Insert(e.ctx, e.newRow("other"), hist("other", "set", 6, 10)); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	expect := []uuid.UUID{id(4), id(3), id(2), id(1), id(5)}

	got, err := s.ListHistories(e.ctx, "k", confighandler.HistoryCursor{}, 10)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if len(got) != len(expect) {
		t.Fatalf("Wrong match. expect: %d rows, got: %d", len(expect), len(got))
	}
	for i, h := range got {
		if h.ID != expect[i] || h.Key != "k" {
			t.Errorf("Wrong match. position %d expect: %s, got: %s %s", i, expect[i], h.ID, h.Key)
		}
	}
	e.checkHistoryFields(t, got[1], at(10))

	e.walkHistories(t, s, expect)

	e.checkHistoryEdges(t, s, at(10), id(2), id(1), id(5))
}

// walkHistories walks the history of "k" with a limit of 2 and the cursor of the last row of each
// page, and expects the rows in the given order.
func (e *contractEnv) walkHistories(t *testing.T, s confighandler.Store, expect []uuid.UUID) {
	var walked []uuid.UUID
	cursor := confighandler.HistoryCursor{}
	for page := 0; page < 10; page++ {
		rows, err := s.ListHistories(e.ctx, "k", cursor, 2)
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		for _, h := range rows {
			walked = append(walked, h.ID)
		}
		if len(rows) < 2 {
			break
		}
		last := rows[len(rows)-1]
		cursor = confighandler.HistoryCursor{TMCreate: last.TMCreate, ID: last.ID}
	}
	if len(walked) != len(expect) {
		t.Fatalf("Wrong match. expect: %v, got: %v", expect, walked)
	}
	for i := range walked {
		if walked[i] != expect[i] {
			t.Errorf("Wrong match. position %d expect: %s, got: %s", i, expect[i], walked[i])
		}
	}
}

// checkHistoryFields checks the fields that a history row keeps (a secret row keeps no value).
func (e *contractEnv) checkHistoryFields(t *testing.T, first *confighandler.History, at time.Time) {
	if first.Action != "set" || !first.IsSecret || first.NewKeyVersion != 2 || first.Source != "api" ||
		first.ActorID != e.actor || !first.TMCreate.Equal(at) || first.OldValueJSON != nil || first.NewValueJSON != nil {
		t.Errorf("Wrong match. got: %+v", first)
	}
}

// checkHistoryEdges checks that a cursor in another time zone selects the same rows, and that a key
// without a change has no rows.
func (e *contractEnv) checkHistoryEdges(t *testing.T, s confighandler.Store, at time.Time, after, expectFirst, expectSecond uuid.UUID) {
	// a cursor in another time zone selects the same rows
	zone := time.FixedZone("test", 9*60*60)
	rows, err := s.ListHistories(e.ctx, "k", confighandler.HistoryCursor{TMCreate: at.In(zone), ID: after}, 10)
	if err != nil || len(rows) != 2 || rows[0].ID != expectFirst || rows[1].ID != expectSecond {
		t.Errorf("Wrong match. got: %v %v", rows, err)
	}

	none, err := s.ListHistories(e.ctx, "nothing", confighandler.HistoryCursor{}, 10)
	if err != nil || len(none) != 0 {
		t.Errorf("Wrong match. expect: no rows, got: %v %v", none, err)
	}
}
