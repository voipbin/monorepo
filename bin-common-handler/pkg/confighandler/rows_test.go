package confighandler_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/pkg/confighandler"
)

// Test_resolve_storedValues covers every type decoded from a stored row, valid and invalid.
func Test_resolve_storedValues(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		key       string
		valueType string
		json      string

		expectErr bool
		check     func(t *testing.T, f *fixture)
	}{
		{name: "string", key: "a.url", valueType: "string", json: `"https://example.invalid"`, check: func(t *testing.T, f *fixture) {
			if s, _, err := f.h.String(ctx, "a.url"); err != nil || s != "https://example.invalid" {
				t.Errorf("Wrong match. got: %q %v", s, err)
			}
		}},
		{name: "string too long", key: "p.model", valueType: "string", json: `"0123456789012345678901234567890123456789"`, expectErr: true},
		{name: "string given a number", key: "a.url", valueType: "string", json: `5`, expectErr: true},
		{name: "enum", key: "a.mode", valueType: "enum", json: `"y"`, check: func(t *testing.T, f *fixture) {
			if s, _, err := f.h.String(ctx, "a.mode"); err != nil || s != "y" {
				t.Errorf("Wrong match. got: %q %v", s, err)
			}
		}},
		{name: "enum outside", key: "a.mode", valueType: "enum", json: `"z"`, expectErr: true},
		{name: "duration", key: "a.timeout", valueType: "duration", json: `"2s"`, check: func(t *testing.T, f *fixture) {
			if d, _, err := f.h.Duration(ctx, "a.timeout"); err != nil || d != 2*time.Second {
				t.Errorf("Wrong match. got: %v %v", d, err)
			}
		}},
		{name: "duration below min", key: "a.timeout", valueType: "duration", json: `"1ms"`, expectErr: true},
		{name: "duration not a string", key: "a.timeout", valueType: "duration", json: `5`, expectErr: true},
		{name: "duration malformed", key: "a.timeout", valueType: "duration", json: `"soon"`, expectErr: true},
		{name: "empty json", key: "a.limit", valueType: "int", json: ``, expectErr: true},
		{name: "int above max", key: "a.limit", valueType: "int", json: `101`, expectErr: true},
		{name: "wrong stored type name", key: "a.mode", valueType: "string", json: `"x"`, expectErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t, testKeyList(1, 1))
			f.putValueRow(tt.key, tt.valueType, tt.json)
			if tt.key == "p.model" {
				// a member of an atomic group is read from the database only when the anchor has a row.
				f.putSecretRow(t, "p.api_key", "db-key", 1)
			}

			var err error
			var res confighandler.Resolved
			switch tt.key {
			case "a.limit":
				_, res, err = f.h.Int(ctx, tt.key)
			case "a.timeout":
				_, res, err = f.h.Duration(ctx, tt.key)
			default:
				_, res, err = f.h.String(ctx, tt.key)
			}

			if tt.expectErr {
				if !errors.Is(err, confighandler.ErrInvalid) || res.Source != confighandler.SourceDBInvalid {
					t.Errorf("Wrong match. expect: db_invalid, got: %s %v", res.Source, err)
				}
				return
			}
			if err != nil || res.Source != confighandler.SourceDB {
				t.Fatalf("Wrong match. expect: ok, got: %s %v", res.Source, err)
			}
			tt.check(t, f)
		})
	}
}

// Test_resolve_boolMember reads the bool select key of an atomic group from the database and from the env.
func Test_resolve_boolMember(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, testKeyList(1, 1))

	f.env["P_ENABLED"] = "false"
	if b, res, err := f.h.Bool(ctx, "p.enabled"); err != nil || b || res.Source != confighandler.SourceEnv {
		t.Errorf("Wrong match. got: %v %s %v", b, res.Source, err)
	}
	f.env["P_ENABLED"] = "maybe"
	if _, _, err := f.h.Bool(ctx, "p.enabled"); !errors.Is(err, confighandler.ErrInvalid) {
		t.Errorf("Wrong match. expect: ErrInvalid, got: %v", err)
	}
	delete(f.env, "P_ENABLED")

	f.putSecretRow(t, "p.api_key", "db-key", 1)
	f.putValueRow("p.enabled", "bool", `false`)
	if b, res, err := f.h.Bool(ctx, "p.enabled"); err != nil || b || res.Source != confighandler.SourceDB {
		t.Errorf("Wrong match. got: %v %s %v", b, res.Source, err)
	}
	f.putValueRow("p.enabled", "bool", `"yes"`)
	if _, _, err := f.h.Bool(ctx, "p.enabled"); !errors.Is(err, confighandler.ErrInvalid) {
		t.Errorf("Wrong match. expect: ErrInvalid, got: %v", err)
	}
}

// failAfterStore fails every Get from the n-th call on.
type failAfterStore struct {
	confighandler.Store

	n     int
	calls int
}

func (s *failAfterStore) Get(ctx context.Context, key string) (*confighandler.Row, error) {
	s.calls++
	if s.calls >= s.n {
		return nil, errors.New("database is down")
	}

	return s.Store.Get(ctx, key)
}

// A write that is committed but cannot be read back is reported as stored, not as a failed write,
// so that a caller does not retry a change that already happened.
func Test_write_readBackFailureIsReportedAsStored(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name string
		run  func(h confighandler.Handler) error
		// failAt is the number of the Get call that fails: the first one is the read of the existing row.
		failAt int
	}{
		{name: "set", failAt: 2, run: func(h confighandler.Handler) error {
			_, err := h.Set(ctx, testActor, "a.limit", "5")
			return err
		}},
		{name: "clear", failAt: 2, run: func(h confighandler.Handler) error {
			_, err := h.Clear(ctx, testActor, "a.limit")
			return err
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t, testKeyList(1, 1))
			f.putValueRow("a.limit", "int", `3`)

			wrapped := &failAfterStore{Store: f.store, n: tt.failAt}
			h := confighandler.NewHandler(testRegistry(t), wrapped, f.crypto, func(string) (string, bool) { return "", false })

			err := tt.run(h)
			if err == nil || !strings.Contains(err.Error(), "is stored, but reading it back failed") {
				t.Errorf("Wrong match. expect: stored but unreadable, got: %v", err)
			}
			if len(f.store.Histories()) != 1 {
				t.Errorf("Wrong match. expect: the change to be committed, histories: %d", len(f.store.Histories()))
			}
		})
	}
}

func Test_write_canceledContext(t *testing.T) {
	f := newFixture(t, testKeyList(1, 1))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := f.h.Set(ctx, testActor, "a.limit", "5"); !errors.Is(err, context.Canceled) {
		t.Errorf("Wrong match. expect: canceled, got: %v", err)
	}
	if _, err := f.h.Clear(ctx, testActor, "a.limit"); !errors.Is(err, context.Canceled) {
		t.Errorf("Wrong match. expect: canceled, got: %v", err)
	}
	if f.store.GetCalls != 0 || len(f.store.Histories()) != 0 {
		t.Errorf("Wrong match. a canceled write touched the store: %d, %d", f.store.GetCalls, len(f.store.Histories()))
	}
}

// Members of an atomic group: a read error of the member row is an error, never "no row", and
// env_shadowed follows the source (a member without its own row is default and not shadowed).
func Test_resolve_groupMemberRows(t *testing.T) {
	ctx := context.Background()

	t.Run("member read error is an error", func(t *testing.T) {
		f := newFixture(t, testKeyList(1, 1))
		f.putSecretRow(t, "p.api_key", "db-key", 1)
		f.store.GetErrByKey = map[string]error{"p.model": errors.New("database is down")}
		f.env["P_MODEL"] = "env-model"

		model, res, err := f.h.String(ctx, "p.model")
		if err == nil || model != "" || res.Source == confighandler.SourceDefault || res.Source == confighandler.SourceEnv {
			t.Errorf("Wrong match. expect: error and no value, got: %q %s %v", model, res.Source, err)
		}
	})

	t.Run("member without a row is default and not shadowed", func(t *testing.T) {
		f := newFixture(t, testKeyList(1, 1))
		f.putSecretRow(t, "p.api_key", "db-key", 1)
		f.env["P_MODEL"] = "env-model"

		model, res, err := f.h.String(ctx, "p.model")
		if err != nil || model != "model-default" || res.Source != confighandler.SourceDefault || res.EnvShadowed {
			t.Errorf("Wrong match. got: %q %s shadowed=%v %v", model, res.Source, res.EnvShadowed, err)
		}
	})

	t.Run("member with a row and an environment value is shadowed", func(t *testing.T) {
		f := newFixture(t, testKeyList(1, 1))
		f.putSecretRow(t, "p.api_key", "db-key", 1)
		f.putValueRow("p.model", "string", `"db-model"`)
		f.env["P_MODEL"] = "env-model"

		model, res, err := f.h.String(ctx, "p.model")
		if err != nil || model != "db-model" || res.Source != confighandler.SourceDB || !res.EnvShadowed {
			t.Errorf("Wrong match. got: %q %s shadowed=%v %v", model, res.Source, res.EnvShadowed, err)
		}
	})
}

// The create time is kept by later writes while the update time follows the clock.
func Test_write_keepsCreateTime(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, testKeyList(1, 1))

	if _, err := f.h.Set(ctx, testActor, "a.limit", "10"); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	later := testNow.Add(time.Hour)
	n := 100
	confighandler.SetClock(f.h, func() time.Time { return later }, func() uuid.UUID {
		n++
		return uuid.FromStringOrNil(fmt.Sprintf("00000000-0000-4000-8000-%012d", n))
	})
	if _, err := f.h.Set(ctx, testActor, "a.limit", "20"); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	row, _ := f.store.Get(ctx, "a.limit")
	if !row.TMCreate.Equal(testNow) || !row.TMUpdate.Equal(later) {
		t.Errorf("Wrong match. expect: create %v update %v, got: %v %v", testNow, later, row.TMCreate, row.TMUpdate)
	}
}

// Concurrent writers are last-writer-wins: every write succeeds or reports a conflict, nothing
// is lost silently and the race detector stays quiet.
func Test_write_concurrent(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, testKeyList(1, 1))

	// the id source of the fixture is not safe for concurrent use
	var idCounter atomic.Int64
	confighandler.SetClock(f.h, func() time.Time { return testNow }, func() uuid.UUID {
		return uuid.FromStringOrNil(fmt.Sprintf("00000000-0000-4000-8000-%012d", idCounter.Add(1)))
	})

	var wg sync.WaitGroup
	results := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := f.h.Set(ctx, testActor, "a.limit", fmt.Sprintf("%d", 10+i))
			results <- err
		}(i)
	}
	wg.Wait()
	close(results)

	ok := 0
	for err := range results {
		switch {
		case err == nil:
			ok++
		case !errors.Is(err, confighandler.ErrConflict):
			t.Errorf("Wrong match. expect: ok or conflict, got: %v", err)
		}
	}

	row, err := f.store.Get(ctx, "a.limit")
	if err != nil || int(row.Revision) != ok || len(f.store.Histories()) != ok {
		t.Errorf("Wrong match. expect: revision and histories equal to the %d successful writes, got: %v %v %d", ok, row, err, len(f.store.Histories()))
	}
}
