package confighandler_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/pkg/confighandler"
)

// writeSecrets stores n changes of the key s.token, each with its own history row.
func writeSecrets(t *testing.T, f *fixture, n int) {
	t.Helper()

	for i := 0; i < n; i++ {
		if _, err := f.h.SetSecret(context.Background(), testActor, "s.token", fmt.Sprintf("value-%d", i)); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
	}
}

// collect reads every page and returns the history ids in the order they were returned.
func collect(t *testing.T, h confighandler.Handler, key string, pageSize int) []uuid.UUID {
	t.Helper()

	var res []uuid.UUID
	token := ""
	for page := 0; page < 50; page++ {
		got, next, err := h.Histories(context.Background(), key, token, pageSize)
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if len(got) > pageSize && pageSize > 0 {
			t.Fatalf("Wrong match. a page holds %d rows, more than %d", len(got), pageSize)
		}
		for _, hist := range got {
			res = append(res, hist.ID)
		}
		if next == "" {
			return res
		}
		token = next
	}

	t.Fatalf("Wrong match. the pages do not end")
	return nil
}

// Every change is returned once, the newest first, whatever the page size is. The fixture gives
// every change the same time, so the order inside the same time follows the id.
func Test_Histories_sameTimestampPages(t *testing.T) {
	f := newFixture(t, testKeyList(1, 1))
	writeSecrets(t, f, 5)

	all := f.store.Histories()
	for _, size := range []int{1, 2, 3, 5, 20} {
		t.Run(fmt.Sprintf("size %d", size), func(t *testing.T) {
			ids := collect(t, f.h, "s.token", size)
			if len(ids) != len(all) {
				t.Fatalf("Wrong match. expect: %d rows, got: %d", len(all), len(ids))
			}
			seen := map[uuid.UUID]bool{}
			for i, id := range ids {
				if seen[id] {
					t.Errorf("Wrong match. row %s is returned twice", id)
				}
				seen[id] = true
				// the newest first: the last written is the first read
				if want := all[len(all)-1-i].ID; id != want {
					t.Errorf("Wrong match. position %d expect: %s, got: %s", i, want, id)
				}
			}
		})
	}
}

// The order follows the time first, and the cursor works across different times and across a
// time zone that is not UTC.
func Test_Histories_differentTimes(t *testing.T) {
	zone := time.FixedZone("test", 9*60*60)

	for _, loc := range []*time.Location{time.UTC, zone} {
		t.Run(loc.String(), func(t *testing.T) {
			f := newFixture(t, testKeyList(1, 1))

			n, m := 0, 0
			confighandler.SetClock(f.h, func() time.Time {
				// the first three changes share a time, the later ones are later
				n++
				return testNow.Add(time.Duration((n-1)/3) * time.Second).In(loc)
			}, func() uuid.UUID {
				m++
				return uuid.FromStringOrNil(fmt.Sprintf("00000000-0000-4000-8000-%012d", m))
			})
			writeSecrets(t, f, 7)

			all := f.store.Histories()
			ids := collect(t, f.h, "s.token", 2)
			if len(ids) != len(all) {
				t.Fatalf("Wrong match. expect: %d rows, got: %d", len(all), len(ids))
			}
			for i, id := range ids {
				if want := all[len(all)-1-i].ID; id != want {
					t.Errorf("Wrong match. position %d expect: %s, got: %s", i, want, id)
				}
			}
		})
	}
}

// Only the changes of the key are returned, and a secret history never holds a value.
func Test_Histories_keyAndSecretFields(t *testing.T) {
	ctx := context.Background()

	f := newFixture(t, testKeyList(1, 1))
	writeSecrets(t, f, 2)
	if _, err := f.h.Set(ctx, testActor, "a.limit", "5"); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if _, err := f.h.Clear(ctx, testActor, "s.token"); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	got, next, err := f.h.Histories(ctx, "s.token", "", 20)
	if err != nil || next != "" {
		t.Fatalf("Wrong match. expect: ok and the last page, got: %v %q", err, next)
	}
	if len(got) != 3 {
		t.Fatalf("Wrong match. expect: 3 rows, got: %d", len(got))
	}
	if got[0].Action != "clear" || got[1].Action != "set" {
		t.Errorf("Wrong match. got: %s, %s", got[0].Action, got[1].Action)
	}
	for _, hist := range got {
		if hist.Key != "s.token" || !hist.IsSecret || hist.OldValueJSON != nil || hist.NewValueJSON != nil {
			t.Errorf("Wrong match. got: %+v", hist)
		}
	}

	other, _, err := f.h.Histories(ctx, "a.limit", "", 20)
	if err != nil || len(other) != 1 || other[0].IsSecret {
		t.Errorf("Wrong match. got: %+v %v", other, err)
	}

	// a key without a change
	none, next, err := f.h.Histories(ctx, "a.mode", "", 20)
	if err != nil || len(none) != 0 || next != "" {
		t.Errorf("Wrong match. got: %+v %q %v", none, next, err)
	}
}

func Test_Histories_errors(t *testing.T) {
	ctx := context.Background()

	f := newFixture(t, testKeyList(1, 1))
	writeSecrets(t, f, 3)

	_, token, err := f.h.Histories(ctx, "s.token", "", 1)
	if err != nil || token == "" {
		t.Fatalf("Wrong match. expect: a next token, got: %q %v", token, err)
	}

	tests := []struct {
		name  string
		key   string
		token string
		size  int

		expectErr error
	}{
		{name: "unknown key", key: "nothing", expectErr: confighandler.ErrUnknownKey},
		{name: "not base64", key: "s.token", token: "!!!", expectErr: confighandler.ErrInvalid},
		{name: "decodes to nonsense", key: "s.token", token: "bm9uc2Vuc2U", expectErr: confighandler.ErrInvalid},
		{name: "token of another key", key: "a.limit", token: token, expectErr: confighandler.ErrInvalid},
		{name: "negative size", key: "s.token", size: -1, expectErr: confighandler.ErrInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := f.h.Histories(ctx, tt.key, tt.token, tt.size)
			if !errors.Is(err, tt.expectErr) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectErr, err)
			}
		})
	}
}

// The default page size is 20 and the largest is 100.
func Test_Histories_pageSizeLimits(t *testing.T) {
	ctx := context.Background()

	f := newFixture(t, testKeyList(1, 1))
	writeSecrets(t, f, 130)

	got, next, err := f.h.Histories(ctx, "s.token", "", 0)
	if err != nil || len(got) != 20 || next == "" {
		t.Errorf("Wrong match. expect: 20 rows and a next token, got: %d %q %v", len(got), next, err)
	}

	got, next, err = f.h.Histories(ctx, "s.token", "", 500)
	if err != nil || len(got) != 100 || next == "" {
		t.Errorf("Wrong match. expect: 100 rows and a next token, got: %d %q %v", len(got), next, err)
	}
}

// A read error of the store is returned, never an empty page.
func Test_Histories_storeError(t *testing.T) {
	f := newFixture(t, testKeyList(1, 1))
	f.store.ListHistoriesErr = errors.New("database is down")

	if _, _, err := f.h.Histories(context.Background(), "s.token", "", 20); err == nil {
		t.Errorf("Wrong match. expect: an error, got: nil")
	}
}
