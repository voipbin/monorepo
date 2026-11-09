package confighandler_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/pkg/confighandler"
)

const testSecretValue = "test-not-real"

// A write with ExpectRevision succeeds only when the stored revision is the expected one (0 means
// no row), is tried once, and never retries.
func Test_SetSecret_expectRevision(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name string

		storedRevision uint32 // 0: no row
		expect         uint32

		expectErr      error
		expectRevision uint32
		expectHistory  int
	}{
		{name: "match", storedRevision: 3, expect: 3, expectRevision: 4, expectHistory: 1},
		{name: "mismatch", storedRevision: 3, expect: 2, expectErr: confighandler.ErrStale},
		{name: "no row and expect zero", storedRevision: 0, expect: 0, expectRevision: 1, expectHistory: 1},
		{name: "no row and expect one", storedRevision: 0, expect: 1, expectErr: confighandler.ErrStale},
		{name: "row and expect zero", storedRevision: 1, expect: 0, expectErr: confighandler.ErrStale},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t, testKeyList(1, 1))
			if tt.storedRevision > 0 {
				f.putSecretRow(t, "s.token", "old-value", tt.storedRevision)
			}

			res, err := f.h.SetSecret(ctx, testActor, "s.token", testSecretValue, confighandler.ExpectRevision(tt.expect))
			if tt.expectErr != nil {
				if !errors.Is(err, tt.expectErr) || !errors.Is(err, confighandler.ErrConflict) {
					t.Fatalf("Wrong match. expect: %v wrapping ErrConflict, got: %v", tt.expectErr, err)
				}
				if len(f.store.Histories()) != 0 {
					t.Errorf("Wrong match. a stale write left a history row")
				}
				if tt.storedRevision > 0 {
					row, _ := f.store.Get(ctx, "s.token")
					if row == nil || row.Revision != tt.storedRevision {
						t.Errorf("Wrong match. a stale write changed the row: %+v", row)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res.Revision != tt.expectRevision || res.Source != confighandler.SourceDB || !res.HasSecret {
				t.Errorf("Wrong match. got: %+v", res)
			}
			if len(f.store.Histories()) != tt.expectHistory {
				t.Errorf("Wrong match. expect: %d histories, got: %d", tt.expectHistory, len(f.store.Histories()))
			}
		})
	}
}

// A store conflict between the read and the write is a stale write, and the write is not
// retried: with a single injected conflict a retry would have succeeded.
func Test_SetSecret_expectRevisionIsNotRetried(t *testing.T) {
	ctx := context.Background()

	f := newFixture(t, testKeyList(1, 1))
	f.putSecretRow(t, "s.token", "old-value", 1)
	f.store.ConflictWrites = 1

	_, err := f.h.SetSecret(ctx, testActor, "s.token", testSecretValue, confighandler.ExpectRevision(1))
	if !errors.Is(err, confighandler.ErrStale) {
		t.Fatalf("Wrong match. expect: ErrStale, got: %v", err)
	}
	if f.store.GetCalls != 1 {
		t.Errorf("Wrong match. expect: 1 read, got: %d", f.store.GetCalls)
	}
	if len(f.store.Histories()) != 0 {
		t.Errorf("Wrong match. a stale write left a history row")
	}
}

// Without the option the write keeps retrying (the command line behavior is unchanged).
func Test_SetSecret_withoutOptionStillRetries(t *testing.T) {
	ctx := context.Background()

	f := newFixture(t, testKeyList(1, 1))
	f.putSecretRow(t, "s.token", "old-value", 1)
	f.store.ConflictWrites = 1

	if _, err := f.h.SetSecret(ctx, testActor, "s.token", testSecretValue); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
}

// The expected revision does not read the stored secret: a row that can not be decrypted is
// replaced like any other row.
func Test_SetSecret_expectRevisionDoesNotDecrypt(t *testing.T) {
	ctx := context.Background()

	f := newFixture(t, testKeyList(1, 1))
	f.store.Put(&confighandler.Row{
		Key: "s.token", ValueType: "secret", SecretCiphertext: []byte("not-a-valid-ciphertext-at-all-0123456789"),
		KeyVersion: 9, Revision: 2, TMCreate: testNow, TMUpdate: testNow,
	})

	res, err := f.h.SetSecret(ctx, testActor, "s.token", testSecretValue, confighandler.ExpectRevision(2))
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.Source != confighandler.SourceDB || res.Revision != 3 {
		t.Errorf("Wrong match. got: %+v", res)
	}

	hists := f.store.Histories()
	if len(hists) != 1 || hists[0].OldKeyVersion != 9 || hists[0].NewKeyVersion != 1 {
		t.Errorf("Wrong match. got: %+v", hists)
	}
}

func Test_Set_expectRevision(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name string

		storedRevision uint32
		expect         uint32

		expectErr      error
		expectRevision uint32
	}{
		{name: "match", storedRevision: 2, expect: 2, expectRevision: 3},
		{name: "mismatch", storedRevision: 2, expect: 1, expectErr: confighandler.ErrStale},
		{name: "no row and expect zero", storedRevision: 0, expect: 0, expectRevision: 1},
		{name: "no row and expect one", storedRevision: 0, expect: 1, expectErr: confighandler.ErrStale},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t, testKeyList(1, 1))
			if tt.storedRevision > 0 {
				f.putValueRow("a.limit", "int", `3`)
				row, _ := f.store.Get(ctx, "a.limit")
				row.Revision = tt.storedRevision
				f.store.Put(row)
			}

			res, err := f.h.Set(ctx, testActor, "a.limit", "5", confighandler.ExpectRevision(tt.expect))
			if tt.expectErr != nil {
				if !errors.Is(err, tt.expectErr) {
					t.Fatalf("Wrong match. expect: %v, got: %v", tt.expectErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res.Revision != tt.expectRevision {
				t.Errorf("Wrong match. expect: %d, got: %d", tt.expectRevision, res.Revision)
			}
		})
	}
}

// Clear with the expected revision: the three cases (row matches, row differs, no row).
func Test_Clear_expectRevision(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name string

		storedRevision uint32 // 0: no row
		expect         uint32

		expectErr     error
		expectSource  confighandler.Source
		expectHistory int
	}{
		{name: "match", storedRevision: 2, expect: 2, expectSource: confighandler.SourceNone, expectHistory: 1},
		{name: "mismatch", storedRevision: 2, expect: 1, expectErr: confighandler.ErrStale},
		{name: "no row and expect zero", storedRevision: 0, expect: 0, expectSource: confighandler.SourceNone, expectHistory: 0},
		{name: "no row and expect one", storedRevision: 0, expect: 1, expectErr: confighandler.ErrStale},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t, testKeyList(1, 1))
			if tt.storedRevision > 0 {
				f.putSecretRow(t, "s.token", "old-value", tt.storedRevision)
			}

			res, err := f.h.Clear(ctx, testActor, "s.token", confighandler.ExpectRevision(tt.expect))
			if tt.expectErr != nil {
				if !errors.Is(err, tt.expectErr) {
					t.Fatalf("Wrong match. expect: %v, got: %v", tt.expectErr, err)
				}
				if len(f.store.Histories()) != 0 {
					t.Errorf("Wrong match. a stale clear left a history row")
				}
				return
			}
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res.Source != tt.expectSource {
				t.Errorf("Wrong match. expect: %s, got: %+v", tt.expectSource, res)
			}
			if len(f.store.Histories()) != tt.expectHistory {
				t.Errorf("Wrong match. expect: %d histories, got: %d", tt.expectHistory, len(f.store.Histories()))
			}
		})
	}
}

// A clear is not retried either, and it works on a row that can not be decrypted.
func Test_Clear_expectRevisionIsNotRetriedAndDoesNotDecrypt(t *testing.T) {
	ctx := context.Background()

	f := newFixture(t, testKeyList(1, 1))
	f.putSecretRow(t, "s.token", "old-value", 1)
	f.store.ConflictWrites = 1
	if _, err := f.h.Clear(ctx, testActor, "s.token", confighandler.ExpectRevision(1)); !errors.Is(err, confighandler.ErrStale) {
		t.Fatalf("Wrong match. expect: ErrStale, got: %v", err)
	}

	g := newFixture(t, testKeyList(1, 1))
	g.store.Put(&confighandler.Row{
		Key: "s.token", ValueType: "secret", SecretCiphertext: []byte("not-a-valid-ciphertext-at-all-0123456789"),
		KeyVersion: 9, Revision: 4, TMCreate: testNow, TMUpdate: testNow,
	})
	if _, err := g.h.Clear(ctx, testActor, "s.token", confighandler.ExpectRevision(4)); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
}

// The checks that come before the write are the same with or without the option.
func Test_write_expectRevisionValidatesFirst(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, testKeyList(1, 1))
	opt := confighandler.ExpectRevision(0)

	if _, err := f.h.SetSecret(ctx, uuid.Nil, "s.token", testSecretValue, opt); !errors.Is(err, confighandler.ErrInvalid) {
		t.Errorf("Wrong match. expect: ErrInvalid, got: %v", err)
	}
	if _, err := f.h.SetSecret(ctx, testActor, "nothing", testSecretValue, opt); !errors.Is(err, confighandler.ErrUnknownKey) {
		t.Errorf("Wrong match. expect: ErrUnknownKey, got: %v", err)
	}
	if _, err := f.h.SetSecret(ctx, testActor, "a.limit", testSecretValue, opt); !errors.Is(err, confighandler.ErrWrongKind) {
		t.Errorf("Wrong match. expect: ErrWrongKind, got: %v", err)
	}
	if _, err := f.h.SetSecret(ctx, testActor, "s.token", strings.Repeat("a", 17), opt); !errors.Is(err, confighandler.ErrTooLong) {
		t.Errorf("Wrong match. expect: ErrTooLong, got: %v", err)
	}
	if _, err := f.h.Clear(ctx, uuid.Nil, "s.token", opt); !errors.Is(err, confighandler.ErrInvalid) {
		t.Errorf("Wrong match. expect: ErrInvalid, got: %v", err)
	}

	noKey := newFixture(t, "")
	if _, err := noKey.h.SetSecret(ctx, testActor, "s.token", testSecretValue, opt); !errors.Is(err, confighandler.ErrKeyNotConfigured) {
		t.Errorf("Wrong match. expect: ErrKeyNotConfigured, got: %v", err)
	}
	if len(f.store.Histories()) != 0 || len(noKey.store.Histories()) != 0 {
		t.Errorf("Wrong match. a rejected write left a history row")
	}
}

// A committed write that can not be read back is an ErrWriteVerify, with and without the option.
func Test_write_readBackFailureIsErrWriteVerify(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name string
		run  func(h confighandler.Handler) error
	}{
		{name: "set secret", run: func(h confighandler.Handler) error {
			_, err := h.SetSecret(ctx, testActor, "s.token", testSecretValue)
			return err
		}},
		{name: "set secret expect", run: func(h confighandler.Handler) error {
			_, err := h.SetSecret(ctx, testActor, "s.token", testSecretValue, confighandler.ExpectRevision(1))
			return err
		}},
		{name: "clear", run: func(h confighandler.Handler) error {
			_, err := h.Clear(ctx, testActor, "s.token")
			return err
		}},
		{name: "clear expect", run: func(h confighandler.Handler) error {
			_, err := h.Clear(ctx, testActor, "s.token", confighandler.ExpectRevision(1))
			return err
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t, testKeyList(1, 1))
			f.putSecretRow(t, "s.token", "old-value", 1)

			wrapped := &failAfterStore{Store: f.store, n: 2}
			h := confighandler.NewHandler(testRegistry(t), wrapped, f.crypto, func(string) (string, bool) { return "", false })

			err := tt.run(h)
			if !errors.Is(err, confighandler.ErrWriteVerify) {
				t.Errorf("Wrong match. expect: ErrWriteVerify, got: %v", err)
			}
			if err != nil && strings.Contains(err.Error(), testSecretValue) {
				t.Errorf("Wrong match. the error repeats the secret: %v", err)
			}
			if len(f.store.Histories()) != 1 {
				t.Errorf("Wrong match. expect: the change to be committed, histories: %d", len(f.store.Histories()))
			}
		})
	}
}

// Two writers that expect the same revision: exactly one wins, the other is stale, and only the
// winner leaves a history row.
func Test_SetSecret_expectRevisionConcurrent(t *testing.T) {
	ctx := context.Background()

	f := newFixture(t, testKeyList(1, 1))
	f.putSecretRow(t, "s.token", "old-value", 1)

	const writers = 8
	var wg sync.WaitGroup
	errs := make([]error, writers)
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = f.h.SetSecret(ctx, testActor, "s.token", testSecretValue, confighandler.ExpectRevision(1))
		}(i)
	}
	wg.Wait()

	ok, stale := 0, 0
	for _, err := range errs {
		switch {
		case err == nil:
			ok++
		case errors.Is(err, confighandler.ErrStale):
			stale++
		default:
			t.Errorf("Wrong match. unexpected error: %v", err)
		}
	}
	if ok != 1 || stale != writers-1 {
		t.Errorf("Wrong match. expect: 1 ok, %d stale, got: %d ok, %d stale", writers-1, ok, stale)
	}
	if len(f.store.Histories()) != 1 {
		t.Errorf("Wrong match. expect: 1 history, got: %d", len(f.store.Histories()))
	}
}

// The accepted limit: after a clear and a new set the revision is 1 again, so a writer that
// still holds revision 1 is not detected. Both changes stay in the history.
func Test_SetSecret_expectRevisionAfterRecreateIsAccepted(t *testing.T) {
	ctx := context.Background()

	f := newFixture(t, testKeyList(1, 1))
	f.putSecretRow(t, "s.token", "old-value", 1)

	if _, err := f.h.Clear(ctx, testActor, "s.token", confighandler.ExpectRevision(1)); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if _, err := f.h.SetSecret(ctx, testActor, "s.token", "other-value", confighandler.ExpectRevision(0)); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if _, err := f.h.SetSecret(ctx, testActor, "s.token", testSecretValue, confighandler.ExpectRevision(1)); err != nil {
		t.Fatalf("Wrong match. expect: ok (accepted limit), got: %v", err)
	}
	if len(f.store.Histories()) != 3 {
		t.Errorf("Wrong match. expect: 3 histories, got: %d", len(f.store.Histories()))
	}
}

func Test_SecretWritable(t *testing.T) {
	tests := []struct {
		name    string
		keyList string
		expect  bool
	}{
		{name: "with a key", keyList: testKeyList(1, 1), expect: true},
		{name: "without a key", keyList: "", expect: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t, tt.keyList)
			if got := f.h.SecretWritable(); got != tt.expect {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expect, got)
			}
		})
	}

	nilCrypto := confighandler.NewHandler(testRegistry(t), nil, nil, func(string) (string, bool) { return "", false })
	if nilCrypto.SecretWritable() {
		t.Errorf("Wrong match. a handler without crypto reports a writable secret")
	}
}
