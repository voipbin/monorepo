package dbhandler

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-number-manager/models/number"
	"monorepo/bin-number-manager/models/provider"
	"monorepo/bin-number-manager/pkg/cachehandler"
)

func newProviderTestHandler(t *testing.T) (*handler, *utilhandler.MockUtilHandler) {
	t.Helper()

	if _, err := dbTest.Exec("delete from number_providers"); err != nil {
		t.Fatal(err)
	}

	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	return &handler{
		utilHandler: mockUtil,
		cache:       cachehandler.NewMockCacheHandler(mc),
		db:          dbTest,
	}, mockUtil
}

func Test_NumberProvider_EnsureGetUpdate(t *testing.T) {
	ctx := context.Background()
	h, mockUtil := newProviderTestHandler(t)

	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	later := now.Add(time.Hour)
	actor := uuid.FromStringOrNil("a0000000-0000-4000-8000-0000000000a1")

	if _, err := h.NumberProviderGet(ctx, number.ProviderNameTelnyx); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Wrong match. expect: not found, got: %v", err)
	}

	// ensure creates the row with the given defaults
	mockUtil.EXPECT().TimeNow().Return(&now)
	if err := h.NumberProviderEnsure(ctx, &provider.Provider{ProviderName: number.ProviderNameTelnyx, Enabled: true, Priority: 10, Config: map[string]string{}}); err != nil {
		t.Fatal(err)
	}

	res, err := h.NumberProviderGet(ctx, number.ProviderNameTelnyx)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Enabled || res.Priority != 10 || len(res.SecretCiphertext) != 0 || res.UpdatedBy != nil || res.SecretUpdatedBy != nil || res.TMSecretUpdate != nil || res.TMCreate == nil {
		t.Errorf("Wrong created row. got: %+v", res)
	}

	// a second ensure leaves the row untouched (insert-if-absent)
	mockUtil.EXPECT().TimeNow().Return(&later)
	if err := h.NumberProviderEnsure(ctx, &provider.Provider{ProviderName: number.ProviderNameTelnyx, Enabled: false, Priority: 99}); err != nil {
		t.Fatalf("Wrong match. expect: ok on a duplicate, got: %v", err)
	}
	res, _ = h.NumberProviderGet(ctx, number.ProviderNameTelnyx)
	if !res.Enabled || res.Priority != 10 {
		t.Errorf("The existing row was overwritten. got: %+v", res)
	}

	// update writes only the given columns, with NULL-able audit columns
	mockUtil.EXPECT().TimeNow().Return(&later).Times(2)
	if err := h.NumberProviderUpdate(ctx, number.ProviderNameTelnyx, map[provider.Field]any{
		provider.FieldPriority:         20,
		provider.FieldConfig:           map[string]string{"connection_id": "c1"},
		provider.FieldSecretCiphertext: []byte{0x01, 0x02, 0xff},
		provider.FieldKeyVersion:       3,
		provider.FieldSecretUpdatedBy:  actor,
		provider.FieldTMSecretUpdate:   &later,
	}); err != nil {
		t.Fatal(err)
	}

	res, _ = h.NumberProviderGet(ctx, number.ProviderNameTelnyx)
	if !res.Enabled || res.Priority != 20 || res.KeyVersion != 3 || !reflect.DeepEqual(res.Config, map[string]string{"connection_id": "c1"}) {
		t.Errorf("Wrong updated row. got: %+v", res)
	}
	if !reflect.DeepEqual(res.SecretCiphertext, []byte{0x01, 0x02, 0xff}) {
		t.Errorf("The ciphertext did not round trip as raw bytes. got: %v", res.SecretCiphertext)
	}
	if res.SecretUpdatedBy == nil || *res.SecretUpdatedBy != actor || res.UpdatedBy != nil || res.TMSecretUpdate == nil || res.TMUpdate == nil {
		t.Errorf("Wrong audit columns. got: %+v", res)
	}

	// clearing the secret stores NULL
	if err := h.NumberProviderUpdate(ctx, number.ProviderNameTelnyx, map[provider.Field]any{
		provider.FieldSecretCiphertext: []byte(nil),
		provider.FieldKeyVersion:       0,
	}); err != nil {
		t.Fatal(err)
	}
	res, _ = h.NumberProviderGet(ctx, number.ProviderNameTelnyx)
	if len(res.SecretCiphertext) != 0 || res.KeyVersion != 0 {
		t.Errorf("The secret was not cleared. got: %+v", res)
	}
	var isNull bool
	if err := dbTest.QueryRow("select secret_ciphertext is null from number_providers where provider_name = 'telnyx'").Scan(&isNull); err != nil || !isNull {
		t.Errorf("The cleared secret must be NULL. err: %v", err)
	}
}

func Test_NumberProvider_Errors(t *testing.T) {
	ctx := context.Background()
	h, mockUtil := newProviderTestHandler(t)

	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	mockUtil.EXPECT().TimeNow().Return(&now).Times(2)
	_ = h.NumberProviderEnsure(ctx, &provider.Provider{ProviderName: "zeta", Enabled: true, Priority: 1})
	_ = h.NumberProviderEnsure(ctx, &provider.Provider{ProviderName: "alpha", Enabled: false, Priority: 2})

	// nothing to update is a no-op, an update of a missing row is not found
	if err := h.NumberProviderUpdate(ctx, "alpha", map[provider.Field]any{}); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
	mockUtil.EXPECT().TimeNow().Return(&now)
	if err := h.NumberProviderUpdate(ctx, "missing", map[provider.Field]any{provider.FieldPriority: 1}); !errors.Is(err, ErrNotFound) {
		t.Errorf("Wrong match. expect: not found, got: %v", err)
	}

	// the table is gone: every method reports the failure
	if _, err := dbTest.Exec("alter table number_providers rename to number_providers_off"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = dbTest.Exec("alter table number_providers_off rename to number_providers") })

	mockUtil.EXPECT().TimeNow().Return(&now).Times(2)
	if _, err := h.NumberProviderGet(ctx, "alpha"); err == nil || errors.Is(err, ErrNotFound) {
		t.Errorf("Wrong match. expect: a query error that is not not-found, got: %v", err)
	}
	if err := h.NumberProviderEnsure(ctx, &provider.Provider{ProviderName: "x"}); err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
	if err := h.NumberProviderUpdate(ctx, "alpha", map[provider.Field]any{provider.FieldPriority: 1}); err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}

func Test_isDuplicateKeyErr(t *testing.T) {
	if isDuplicateKeyErr(nil) {
		t.Errorf("nil is not a duplicate")
	}
	if !isDuplicateKeyErr(errors.New("UNIQUE constraint failed: number_providers.provider_name")) {
		t.Errorf("a sqlite unique violation is a duplicate")
	}
	if isDuplicateKeyErr(errors.New("something with 1062 in the text")) {
		t.Errorf("a plain message is not a duplicate")
	}
}

func Test_NumberGetsByTMRenew_excludeIDs(t *testing.T) {
	ctx := context.Background()

	mc := gomock.NewController(t)
	defer mc.Finish()
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{utilHandler: mockUtil, cache: mockCache, db: dbTest}

	if _, err := dbTest.Exec("delete from number_numbers"); err != nil {
		t.Fatal(err)
	}

	base := time.Date(2020, 4, 10, 18, 26, 49, 0, time.UTC)
	ids := []uuid.UUID{
		uuid.FromStringOrNil("b0000000-0000-4000-8000-000000000001"),
		uuid.FromStringOrNil("b0000000-0000-4000-8000-000000000002"),
		uuid.FromStringOrNil("b0000000-0000-4000-8000-000000000003"),
	}
	for i, id := range ids {
		tm := base.Add(time.Duration(i) * time.Minute)
		mockUtil.EXPECT().TimeNow().Return(&tm)
		mockCache.EXPECT().NumberSet(gomock.Any(), gomock.Any())

		n := &number.Number{}
		n.ID = id
		if err := h.NumberCreate(ctx, n); err != nil {
			t.Fatal(err)
		}
	}

	filters := map[number.Field]any{number.FieldDeleted: false}
	later := "2030-01-01 00:00:00.000000"

	tests := []struct {
		name    string
		exclude []uuid.UUID
		expect  []uuid.UUID
	}{
		{"no exclusion", nil, []uuid.UUID{ids[2], ids[1], ids[0]}},
		{"empty exclusion", []uuid.UUID{}, []uuid.UUID{ids[2], ids[1], ids[0]}},
		{"one excluded", []uuid.UUID{ids[1]}, []uuid.UUID{ids[2], ids[0]}},
		{"several excluded", []uuid.UUID{ids[0], ids[2]}, []uuid.UUID{ids[1]}},
		{"all excluded", ids, []uuid.UUID{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := h.NumberGetsByTMRenew(ctx, later, 100, filters, tt.exclude)
			if err != nil {
				t.Fatal(err)
			}

			got := []uuid.UUID{}
			for _, n := range res {
				got = append(got, n.ID)
			}
			if !reflect.DeepEqual(got, tt.expect) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expect, got)
			}
		})
	}
}
