package providerhandler

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"

	"monorepo/bin-number-manager/models/number"
	"monorepo/bin-number-manager/models/provider"
	"monorepo/bin-number-manager/pkg/dbhandler"
)

// failOnCall wraps the fake db and fails the provider read at the given call number (1 based).
type failOnCall struct {
	*fakeDB

	failAt int
	calls  int
}

func (f *failOnCall) NumberProviderGet(ctx context.Context, name number.ProviderName) (*provider.Provider, error) {
	f.calls++
	if f.calls == f.failAt {
		return nil, errTestDB
	}
	return f.fakeDB.NumberProviderGet(ctx, name)
}

func Test_List_readError(t *testing.T) {
	db := newFakeDB()
	db.getErr = errTestDB
	h := newTestHandler(t, db, testKeys(1), envTelnyx())

	res, err := h.List(context.Background())
	if !errors.Is(err, errTestDB) {
		t.Errorf("Wrong match. expect: %v, got: %v", errTestDB, err)
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", res)
	}
}

func Test_ensureRow_unknownProvider(t *testing.T) {
	db := newFakeDB()
	h := newTestHandler(t, db, testKeys(1), envTelnyx())

	err := h.ensureRow(context.Background(), "nope")
	if !errors.Is(err, dbhandler.ErrNotFound) {
		t.Errorf("Wrong match. expect: not found, got: %v", err)
	}
	if len(db.rows) != 0 {
		t.Errorf("Wrong match. expect: no row created, got: %v", db.rows)
	}
}

func Test_Update_rowErrors(t *testing.T) {
	ctx := context.Background()

	t.Run("the row cannot be created", func(t *testing.T) {
		db := newFakeDB()
		db.ensureErr = errTestDB
		h := newTestHandler(t, db, testKeys(1), envTelnyx())

		_, err := h.Update(ctx, number.ProviderNameTelnyx, testActor, boolPtr(true), nil, nil)
		if !errors.Is(err, errTestDB) {
			t.Errorf("Wrong match. expect: %v, got: %v", errTestDB, err)
		}
		if len(db.updates) != 0 {
			t.Errorf("Wrong match. expect: no update, got: %v", db.updates)
		}
	})

	t.Run("the row cannot be read", func(t *testing.T) {
		db := newFakeDB()
		db.getErr = errTestDB
		h := newTestHandler(t, db, testKeys(1), envTelnyx())

		_, err := h.Update(ctx, number.ProviderNameTelnyx, testActor, boolPtr(true), nil, nil)
		if err == nil || !errors.Is(err, errTestDB) || !strings.Contains(err.Error(), "could not get the provider row") {
			t.Errorf("Wrong match. expect the row read error, got: %v", err)
		}
		if len(db.updates) != 0 {
			t.Errorf("Wrong match. expect: no update, got: %v", db.updates)
		}
	})

	t.Run("a failed candidate check after the change does not fail the update", func(t *testing.T) {
		db := newFakeDB()
		h := newTestHandler(t, db, testKeys(1), envTelnyx())
		// call 1 reads the row, call 2 is the candidate check, call 3 builds the result
		h.db = &failOnCall{fakeDB: db, failAt: 2}

		info, err := h.Update(ctx, number.ProviderNameTelnyx, testActor, boolPtr(false), nil, nil)
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if info.ProviderName != number.ProviderNameTelnyx || info.Enabled {
			t.Errorf("Wrong match. got: %+v", info)
		}
		if len(db.updates) != 1 {
			t.Errorf("Wrong match. expect: one update, got: %v", db.updates)
		}
	})
}

func Test_UpdateSecret_rowErrors(t *testing.T) {
	ctx := context.Background()

	t.Run("the row cannot be created", func(t *testing.T) {
		db := newFakeDB()
		db.ensureErr = errTestDB
		h := newTestHandler(t, db, testKeys(1), envTelnyx())

		_, err := h.UpdateSecret(ctx, number.ProviderNameTelnyx, testActor, "fake-secret-value")
		if !errors.Is(err, errTestDB) {
			t.Errorf("Wrong match. expect: %v, got: %v", errTestDB, err)
		}
		if len(db.updates) != 0 {
			t.Errorf("Wrong match. expect: no update, got: %v", db.updates)
		}
	})

	t.Run("the row cannot be read", func(t *testing.T) {
		db := newFakeDB()
		db.getErr = errTestDB
		h := newTestHandler(t, db, testKeys(1), envTelnyx())

		_, err := h.UpdateSecret(ctx, number.ProviderNameTelnyx, testActor, "fake-secret-value")
		if err == nil || !errors.Is(err, errTestDB) || !strings.Contains(err.Error(), "could not get the provider row") {
			t.Errorf("Wrong match. expect the row read error, got: %v", err)
		}
		if len(db.updates) != 0 {
			t.Errorf("Wrong match. expect: no update, got: %v", db.updates)
		}
	})
}

// purchaseCount reads the value of the purchase counter from the default registry.
func purchaseCount(t *testing.T, name, result string) float64 {
	t.Helper()

	families, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("could not gather the metrics. err: %v", err)
	}
	for _, mf := range families {
		if mf.GetName() != "number_manager_provider_purchase_total" {
			continue
		}
		for _, m := range mf.GetMetric() {
			labels := map[string]string{}
			for _, l := range m.GetLabel() {
				labels[l.GetName()] = l.GetValue()
			}
			if labels["provider"] == name && labels["result"] == result {
				return m.GetCounter().GetValue()
			}
		}
	}
	return 0
}

func Test_ObservePurchase(t *testing.T) {
	beforeRejected := purchaseCount(t, "telnyx", "rejected")
	beforeSuccess := purchaseCount(t, "telnyx", "success")

	ObservePurchase(number.ProviderNameTelnyx, "rejected")
	ObservePurchase(number.ProviderNameTelnyx, "rejected")

	if got := purchaseCount(t, "telnyx", "rejected") - beforeRejected; got != 2 {
		t.Errorf("Wrong match. expect: 2, got: %v", got)
	}
	if got := purchaseCount(t, "telnyx", "success") - beforeSuccess; got != 0 {
		t.Errorf("Wrong match. expect the other result to stay: 0, got: %v", got)
	}
}
