// Package storetest holds an in-memory confighandler.Store and the contract test that every Store
// implementation must pass. It is used by tests only.
package storetest

import (
	"bytes"
	"context"
	"sort"
	"sync"

	"monorepo/bin-common-handler/pkg/confighandler"
)

// Memory is an in-memory Store. The exported fields let a test inject failures.
type Memory struct {
	mu sync.Mutex

	rows  map[string]*confighandler.Row
	hists []*confighandler.History

	// GetErr and ListErr are returned by Get and List when set.
	GetErr error
	// GetErrByKey is returned by Get for the given key only.
	GetErrByKey map[string]error
	ListErr     error

	// ListHistoriesErr is returned by ListHistories when set.
	ListHistoriesErr error

	// ConflictWrites is the number of following Insert, Update and Delete calls that return
	// ErrConflict without changing anything.
	ConflictWrites int

	// GetCalls counts Get calls.
	GetCalls int
}

// NewMemory creates an empty store.
func NewMemory() *Memory {
	return &Memory{rows: map[string]*confighandler.Row{}}
}

func cloneRow(r *confighandler.Row) *confighandler.Row {
	c := *r
	c.ValueJSON = append([]byte(nil), r.ValueJSON...)
	c.SecretCiphertext = append([]byte(nil), r.SecretCiphertext...)
	if r.UpdatedBy != nil {
		u := *r.UpdatedBy
		c.UpdatedBy = &u
	}

	return &c
}

func cloneHistory(h *confighandler.History) *confighandler.History {
	c := *h
	if h.OldValueJSON != nil {
		c.OldValueJSON = append([]byte(nil), h.OldValueJSON...)
	}
	if h.NewValueJSON != nil {
		c.NewValueJSON = append([]byte(nil), h.NewValueJSON...)
	}

	return &c
}

// Get implements confighandler.Store.
func (m *Memory) Get(_ context.Context, key string) (*confighandler.Row, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.GetCalls++
	if m.GetErr != nil {
		return nil, m.GetErr
	}
	if err := m.GetErrByKey[key]; err != nil {
		return nil, err
	}

	r, ok := m.rows[key]
	if !ok {
		return nil, confighandler.ErrNotFound
	}

	return cloneRow(r), nil
}

// List implements confighandler.Store.
func (m *Memory) List(_ context.Context) ([]*confighandler.Row, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.ListErr != nil {
		return nil, m.ListErr
	}

	keys := make([]string, 0, len(m.rows))
	for k := range m.rows {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	res := make([]*confighandler.Row, 0, len(keys))
	for _, k := range keys {
		res = append(res, cloneRow(m.rows[k]))
	}

	return res, nil
}

func (m *Memory) injectedConflict() bool {
	if m.ConflictWrites > 0 {
		m.ConflictWrites--
		return true
	}

	return false
}

// Insert implements confighandler.Store.
func (m *Memory) Insert(_ context.Context, row *confighandler.Row, hist *confighandler.History) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.injectedConflict() {
		return confighandler.ErrConflict
	}
	if _, ok := m.rows[row.Key]; ok {
		return confighandler.ErrConflict
	}

	m.rows[row.Key] = cloneRow(row)
	m.hists = append(m.hists, cloneHistory(hist))

	return nil
}

// Update implements confighandler.Store.
func (m *Memory) Update(_ context.Context, row *confighandler.Row, expectRevision uint32, hist *confighandler.History) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.injectedConflict() {
		return confighandler.ErrConflict
	}
	cur, ok := m.rows[row.Key]
	if !ok || cur.Revision != expectRevision {
		return confighandler.ErrConflict
	}

	c := cloneRow(row)
	c.Revision = expectRevision + 1
	m.rows[row.Key] = c
	m.hists = append(m.hists, cloneHistory(hist))

	return nil
}

// Delete implements confighandler.Store.
func (m *Memory) Delete(_ context.Context, key string, expectRevision uint32, hist *confighandler.History) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.injectedConflict() {
		return confighandler.ErrConflict
	}
	cur, ok := m.rows[key]
	if !ok || cur.Revision != expectRevision {
		return confighandler.ErrConflict
	}

	delete(m.rows, key)
	m.hists = append(m.hists, cloneHistory(hist))

	return nil
}

// Histories returns a copy of the stored history rows in write order.
func (m *Memory) Histories() []*confighandler.History {
	m.mu.Lock()
	defer m.mu.Unlock()

	res := make([]*confighandler.History, 0, len(m.hists))
	for _, h := range m.hists {
		res = append(res, cloneHistory(h))
	}

	return res
}

// Put stores a row directly (test setup), without history.
func (m *Memory) Put(row *confighandler.Row) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.rows[row.Key] = cloneRow(row)
}

// ListHistories implements confighandler.Store.
func (m *Memory) ListHistories(_ context.Context, key string, after confighandler.HistoryCursor, limit int) ([]*confighandler.History, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.ListHistoriesErr != nil {
		return nil, m.ListHistoriesErr
	}

	var rows []*confighandler.History
	for _, h := range m.hists {
		if h.Key != key {
			continue
		}
		if !after.TMCreate.IsZero() {
			older := h.TMCreate.Before(after.TMCreate)
			sameTimeSmallerID := h.TMCreate.Equal(after.TMCreate) && bytes.Compare(h.ID.Bytes(), after.ID.Bytes()) < 0
			if !older && !sameTimeSmallerID {
				continue
			}
		}
		rows = append(rows, h)
	}

	sort.SliceStable(rows, func(i, j int) bool {
		if !rows[i].TMCreate.Equal(rows[j].TMCreate) {
			return rows[i].TMCreate.After(rows[j].TMCreate)
		}

		return bytes.Compare(rows[i].ID.Bytes(), rows[j].ID.Bytes()) > 0
	})
	if limit >= 0 && len(rows) > limit {
		rows = rows[:limit]
	}

	res := make([]*confighandler.History, 0, len(rows))
	for _, h := range rows {
		res = append(res, cloneHistory(h))
	}

	return res, nil
}
