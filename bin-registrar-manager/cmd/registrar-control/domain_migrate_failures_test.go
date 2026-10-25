package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-registrar-manager/models/customerdomain"
	"monorepo/bin-registrar-manager/models/extension"
	"monorepo/bin-registrar-manager/pkg/customerdomainhandler"
	"monorepo/bin-registrar-manager/pkg/dbhandler"
)

var errInjectedStep = errors.New("injected step failure")

// Test_reprovisionExtension_stepFailures verifies that a failure at each ordered
// step is returned wrapped with its cause, and that no later step is attempted
// (the strict mock fails on any call that is not in the chain, and the notify
// mock has no expectation, so no event is published).
func Test_reprovisionExtension_stepFailures(t *testing.T) {
	customerID := uuid.FromStringOrNil("55555555-7f85-11ee-8f5a-1b2c3d4e5f60")
	oldRealm := "oldlabel.reg.example.test"
	newRealm := "ab12.reg.example.test"

	steps := []struct {
		name string
		want string
	}{
		{"ast endpoint delete of the old id", "could not delete the ast endpoint"},
		{"ast endpoint delete of the new id", "could not delete the ast endpoint"},
		{"ast auth delete of the old id", "could not delete the ast auth"},
		{"ast auth delete of the new id", "could not delete the ast auth"},
		{"ast aor delete of the old id", "could not delete the ast aor"},
		{"ast aor delete of the new id", "could not delete the ast aor"},
		{"ast aor create", "could not create the ast aor"},
		{"ast auth create", "could not create the ast auth"},
		{"ast endpoint create", "could not create the ast endpoint"},
		{"sip auth realm update", "could not update the sip auth realm"},
		{"stale contact purge", "could not purge the stale contacts"},
		{"extension update", "could not update the extension"},
		{"extension re-read", "could not get the updated extension"},
	}

	for failAt, step := range steps {
		t.Run(step.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDBAst := dbhandler.NewMockDBHandler(mc)
			mockDBBin := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			ctx := context.Background()

			ext := testExtension(customerID, "1001", oldRealm)
			oldEndpoint := ext.EndpointID
			newEndpoint := "1001@" + newRealm

			// the chain of calls in the order the function performs them; the chain
			// ends at the failing call, so nothing after it is expected
			result := func(idx int) error {
				if idx == failAt {
					return errInjectedStep
				}
				return nil
			}
			chain := []func() *gomock.Call{
				func() *gomock.Call { return mockDBAst.EXPECT().AstEndpointDelete(ctx, oldEndpoint).Return(result(0)) },
				func() *gomock.Call { return mockDBAst.EXPECT().AstEndpointDelete(ctx, newEndpoint).Return(result(1)) },
				func() *gomock.Call { return mockDBAst.EXPECT().AstAuthDelete(ctx, oldEndpoint).Return(result(2)) },
				func() *gomock.Call { return mockDBAst.EXPECT().AstAuthDelete(ctx, newEndpoint).Return(result(3)) },
				func() *gomock.Call { return mockDBAst.EXPECT().AstAORDelete(ctx, oldEndpoint).Return(result(4)) },
				func() *gomock.Call { return mockDBAst.EXPECT().AstAORDelete(ctx, newEndpoint).Return(result(5)) },
				func() *gomock.Call { return mockDBAst.EXPECT().AstAORCreate(ctx, gomock.Any()).Return(result(6)) },
				func() *gomock.Call { return mockDBAst.EXPECT().AstAuthCreate(ctx, gomock.Any()).Return(result(7)) },
				func() *gomock.Call { return mockDBAst.EXPECT().AstEndpointCreate(ctx, gomock.Any()).Return(result(8)) },
				func() *gomock.Call {
					return mockDBBin.EXPECT().SIPAuthUpdate(ctx, ext.ID, gomock.Any()).Return(result(9))
				},
				func() *gomock.Call {
					return mockDBAst.EXPECT().AstContactDeleteByEndpoint(ctx, oldEndpoint).Return(result(10))
				},
				func() *gomock.Call {
					return mockDBBin.EXPECT().ExtensionUpdate(ctx, ext.ID, gomock.Any()).Return(result(11))
				},
				func() *gomock.Call { return mockDBBin.EXPECT().ExtensionGet(ctx, ext.ID).Return(nil, errInjectedStep) },
			}
			calls := []any{}
			for i := 0; i <= failAt; i++ {
				calls = append(calls, chain[i]())
			}
			gomock.InOrder(calls...)

			record, err := reprovisionExtension(ctx, mockDBAst, mockDBBin, mockNotify, ext, newRealm)
			if err == nil {
				t.Fatalf("Wrong match. expect: an error, got: nil")
			}
			if !errors.Is(err, errInjectedStep) {
				t.Errorf("Wrong match. expect: the cause to stay reachable, got: %v", err)
			}
			if !strings.Contains(err.Error(), step.want) {
				t.Errorf("Wrong match. expect the error to contain %q, got: %v", step.want, err)
			}
			if record != nil {
				t.Errorf("Wrong match. expect: no record on failure, got: %+v", record)
			}
		})
	}
}

// Test_reprovisionExtension_emptyIDs verifies that stored ps ids that are empty are not
// deleted (an empty id must never reach the delete calls) and that the stale contact
// purge is skipped when the extension had no endpoint id to purge.
func Test_reprovisionExtension_emptyIDs(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDBAst := dbhandler.NewMockDBHandler(mc)
	mockDBBin := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("55555555-7f85-11ee-8f5a-1b2c3d4e5f60")
	newRealm := "ab12.reg.example.test"
	newEndpoint := "1001@" + newRealm

	ext := testExtension(customerID, "1001", "oldlabel.reg.example.test")
	ext.EndpointID = ""
	ext.AuthID = ""
	ext.AORID = ""

	// only the new ids are deleted, exactly once per table
	mockDBAst.EXPECT().AstEndpointDelete(ctx, newEndpoint).Return(nil)
	mockDBAst.EXPECT().AstAuthDelete(ctx, newEndpoint).Return(nil)
	mockDBAst.EXPECT().AstAORDelete(ctx, newEndpoint).Return(nil)
	mockDBAst.EXPECT().AstAORCreate(ctx, gomock.Any()).Return(nil)
	mockDBAst.EXPECT().AstAuthCreate(ctx, gomock.Any()).Return(nil)
	mockDBAst.EXPECT().AstEndpointCreate(ctx, gomock.Any()).Return(nil)
	mockDBBin.EXPECT().SIPAuthUpdate(ctx, ext.ID, gomock.Any()).Return(nil)
	// AstContactDeleteByEndpoint has no expectation: an empty old endpoint is not purged
	mockDBBin.EXPECT().ExtensionUpdate(ctx, ext.ID, gomock.Any()).Return(nil)
	updated := testExtension(customerID, "1001", newRealm)
	mockDBBin.EXPECT().ExtensionGet(ctx, ext.ID).Return(updated, nil)
	mockNotify.EXPECT().PublishEvent(ctx, extension.EventTypeExtensionUpdated, updated).Times(1)

	record, err := reprovisionExtension(ctx, mockDBAst, mockDBBin, mockNotify, ext, newRealm)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if record == nil || record.OldEndpointID != "" || record.NewEndpointID != newEndpoint {
		t.Errorf("Wrong match. record: %+v", record)
	}
}

// Test_migrateCustomerDomain_failures verifies the failure handling of one customer
// migration: the error is wrapped, the cause stays reachable, and the mutating steps
// that follow the failure are not started.
func Test_migrateCustomerDomain_failures(t *testing.T) {
	customerID := uuid.FromStringOrNil("88888888-7f85-11ee-8f5a-1b2c3d4e5f60")
	oldRealm := customerID.String() + ".registrar.example.test"
	newRealm := "gh78.reg.example.test"

	backfilledRow := &customerdomain.CustomerDomain{
		CustomerID:  customerID,
		DomainLabel: customerID.String(),
		Realm:       oldRealm,
	}
	migratedRow := &customerdomain.CustomerDomain{
		CustomerID:  customerID,
		DomainLabel: "gh78",
		Realm:       newRealm,
	}
	listFilter := map[extension.Field]any{
		extension.FieldCustomerID: customerID,
		extension.FieldDeleted:    false,
	}

	tests := []struct {
		name string

		setup  func(ctx context.Context, mockDBAst *dbhandler.MockDBHandler, mockDBBin *dbhandler.MockDBHandler, mockCD *customerdomainhandler.MockCustomerDomainHandler)
		logger func() *failingLog

		expectErr string
	}{
		{
			name: "domain lookup failure other than not found stops before listing",
			setup: func(ctx context.Context, mockDBAst *dbhandler.MockDBHandler, mockDBBin *dbhandler.MockDBHandler, mockCD *customerdomainhandler.MockCustomerDomainHandler) {
				mockDBBin.EXPECT().CustomerDomainGet(ctx, customerID).Return(nil, errInjectedStep)
			},
			expectErr: "could not get customer domain",
		},
		{
			name: "extension listing failure stops before any mutation",
			setup: func(ctx context.Context, mockDBAst *dbhandler.MockDBHandler, mockDBBin *dbhandler.MockDBHandler, mockCD *customerdomainhandler.MockCustomerDomainHandler) {
				mockDBBin.EXPECT().CustomerDomainGet(ctx, customerID).Return(backfilledRow, nil)
				mockDBBin.EXPECT().ExtensionList(ctx, uint64(migrationExtensionListLimit), "", listFilter).Return(nil, errInjectedStep)
			},
			expectErr: "could not list extensions",
		},
		{
			name: "domain row migration failure writes no log record",
			setup: func(ctx context.Context, mockDBAst *dbhandler.MockDBHandler, mockDBBin *dbhandler.MockDBHandler, mockCD *customerdomainhandler.MockCustomerDomainHandler) {
				mockDBBin.EXPECT().CustomerDomainGet(ctx, customerID).Return(backfilledRow, nil)
				mockDBBin.EXPECT().ExtensionList(ctx, uint64(migrationExtensionListLimit), "", listFilter).Return([]*extension.Extension{}, nil)
				mockCD.EXPECT().MigrateToShortDomain(ctx, customerID).Return(nil, errInjectedStep)
			},
			expectErr: "could not migrate the customer domain row",
		},
		{
			name: "log record write failure stops before the first re-provision",
			setup: func(ctx context.Context, mockDBAst *dbhandler.MockDBHandler, mockDBBin *dbhandler.MockDBHandler, mockCD *customerdomainhandler.MockCustomerDomainHandler) {
				mockDBBin.EXPECT().CustomerDomainGet(ctx, customerID).Return(backfilledRow, nil)
				mockDBBin.EXPECT().ExtensionList(ctx, uint64(migrationExtensionListLimit), "", listFilter).Return([]*extension.Extension{testExtension(customerID, "1001", oldRealm)}, nil)
				mockCD.EXPECT().MigrateToShortDomain(ctx, customerID).Return(migratedRow, nil)
				// no ps table call is expected: the rollback record must be durable first
			},
			logger:    func() *failingLog { return &failingLog{failAfter: 0} },
			expectErr: "could not write the migration log record",
		},
		{
			name: "completion marker write failure is reported after the record was written",
			setup: func(ctx context.Context, mockDBAst *dbhandler.MockDBHandler, mockDBBin *dbhandler.MockDBHandler, mockCD *customerdomainhandler.MockCustomerDomainHandler) {
				mockDBBin.EXPECT().CustomerDomainGet(ctx, customerID).Return(backfilledRow, nil)
				mockDBBin.EXPECT().ExtensionList(ctx, uint64(migrationExtensionListLimit), "", listFilter).Return([]*extension.Extension{}, nil)
				mockCD.EXPECT().MigrateToShortDomain(ctx, customerID).Return(migratedRow, nil)
			},
			logger:    func() *failingLog { return &failingLog{failAfter: 1} },
			expectErr: "could not write the migration log completion marker",
		},
		{
			name: "re-provision failure names the extension",
			setup: func(ctx context.Context, mockDBAst *dbhandler.MockDBHandler, mockDBBin *dbhandler.MockDBHandler, mockCD *customerdomainhandler.MockCustomerDomainHandler) {
				ext := testExtension(customerID, "1001", oldRealm)
				mockDBBin.EXPECT().CustomerDomainGet(ctx, customerID).Return(backfilledRow, nil)
				mockDBBin.EXPECT().ExtensionList(ctx, uint64(migrationExtensionListLimit), "", listFilter).Return([]*extension.Extension{ext}, nil)
				mockCD.EXPECT().MigrateToShortDomain(ctx, customerID).Return(migratedRow, nil)
				mockDBAst.EXPECT().AstEndpointDelete(ctx, ext.EndpointID).Return(errInjectedStep)
			},
			logger:    func() *failingLog { return &failingLog{failAfter: 99} },
			expectErr: "could not re-provision extension. extension_id: " + testExtension(customerID, "1001", oldRealm).ID.String(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDBAst := dbhandler.NewMockDBHandler(mc)
			mockDBBin := dbhandler.NewMockDBHandler(mc)
			mockCD := customerdomainhandler.NewMockCustomerDomainHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			ctx := context.Background()

			tt.setup(ctx, mockDBAst, mockDBBin, mockCD)

			logger := &failingLog{failAfter: 99}
			if tt.logger != nil {
				logger = tt.logger()
			}

			record, skipped, err := migrateCustomerDomain(ctx, mockDBAst, mockDBBin, mockCD, mockNotify, customerID, true, logger, &bytes.Buffer{})
			if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
				t.Fatalf("Wrong match. expect an error containing %q, got: %v", tt.expectErr, err)
			}
			if record != nil || skipped {
				t.Errorf("Wrong match. expect: no record and not skipped, got: %+v, %v", record, skipped)
			}
		})
	}
}

// failingLog is a log writer that accepts failAfter writes and fails every later one.
type failingLog struct {
	failAfter int
	writes    int
}

func (w *failingLog) Write(p []byte) (int, error) {
	if w.writes >= w.failAfter {
		return 0, errInjectedWrite
	}
	w.writes++

	return len(p), nil
}

// Test_migrateCustomerDomain_dryRunLegacyRealm verifies that, for a customer without a
// domain row, a stored extension without a realm column supplies the old realm from
// the domain part of its endpoint id, and that a dry run writes nothing.
func Test_migrateCustomerDomain_dryRunLegacyRealm(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDBAst := dbhandler.NewMockDBHandler(mc)
	mockDBBin := dbhandler.NewMockDBHandler(mc)
	mockCD := customerdomainhandler.NewMockCustomerDomainHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("88888888-7f85-11ee-8f5a-1b2c3d4e5f60")
	ext := testExtension(customerID, "1001", "")
	ext.EndpointID = "1001@legacy.reg.example.test"

	mockDBBin.EXPECT().CustomerDomainGet(ctx, customerID).Return(nil, dbhandler.ErrNotFound)
	mockDBBin.EXPECT().ExtensionList(ctx, uint64(migrationExtensionListLimit), "", gomock.Any()).Return([]*extension.Extension{ext}, nil)

	out := &bytes.Buffer{}
	record, skipped, err := migrateCustomerDomain(ctx, mockDBAst, mockDBBin, mockCD, mockNotify, customerID, false, nil, out)
	if err != nil || skipped {
		t.Fatalf("Wrong match. expect: ok and not skipped, got: %v, %v", err, skipped)
	}
	if record.OldRealm != "legacy.reg.example.test" || record.OldLabel != "legacy" {
		t.Errorf("Wrong match. record: %+v", record)
	}
	if !strings.Contains(out.String(), "[dry-run]") {
		t.Errorf("Wrong match. expect: a dry-run line, got: %q", out.String())
	}
}

// Test_domainMigrate_failFast verifies the batch counters and that the batch stops at
// the first failing customer without touching the customers after it.
func Test_domainMigrate_failFast(t *testing.T) {
	ctx := context.Background()
	ids := helperCaseIDs(1, 4)
	longRow := func(id uuid.UUID) *customerdomain.CustomerDomain {
		return &customerdomain.CustomerDomain{CustomerID: id, DomainLabel: id.String(), Realm: id.String() + ".reg.example.test"}
	}
	shortRow := func(id uuid.UUID) *customerdomain.CustomerDomain {
		return &customerdomain.CustomerDomain{CustomerID: id, DomainLabel: "ab12", Realm: "ab12.reg.example.test"}
	}

	t.Run("listing failure returns zero counts", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockDBAst := dbhandler.NewMockDBHandler(mc)
		mockDBBin := dbhandler.NewMockDBHandler(mc)
		mockCD := customerdomainhandler.NewMockCustomerDomainHandler(mc)
		mockNotify := notifyhandler.NewMockNotifyHandler(mc)

		mockDBBin.EXPECT().CustomerDomainList(ctx, uint64(migrationPageSize), "", gomock.Any()).Return(nil, errInjectedStep)

		migrated, skipped, err := domainMigrate(ctx, mockDBAst, mockDBBin, mockCD, mockNotify, false, uuid.Nil, nil, &bytes.Buffer{})
		if !errors.Is(err, errInjectedStep) {
			t.Errorf("Wrong match. expect: the cause to stay reachable, got: %v", err)
		}
		if migrated != 0 || skipped != 0 {
			t.Errorf("Wrong match. expect: 0, 0, got: %d, %d", migrated, skipped)
		}
	})

	t.Run("a failing customer stops the batch and keeps the counts so far", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockDBAst := dbhandler.NewMockDBHandler(mc)
		mockDBBin := dbhandler.NewMockDBHandler(mc)
		mockCD := customerdomainhandler.NewMockCustomerDomainHandler(mc)
		mockNotify := notifyhandler.NewMockNotifyHandler(mc)

		domains := []*customerdomain.CustomerDomain{}
		for _, id := range ids {
			domains = append(domains, &customerdomain.CustomerDomain{CustomerID: id})
		}
		mockDBBin.EXPECT().CustomerDomainList(ctx, uint64(migrationPageSize), "", gomock.Any()).Return(domains, nil)

		// customer 1: dry run, no extensions -> migrated
		mockDBBin.EXPECT().CustomerDomainGet(ctx, ids[0]).Return(longRow(ids[0]), nil)
		mockDBBin.EXPECT().ExtensionList(ctx, uint64(migrationExtensionListLimit), "", gomock.Any()).Return([]*extension.Extension{}, nil)
		// customer 2: already short and every extension done -> skipped
		mockDBBin.EXPECT().CustomerDomainGet(ctx, ids[1]).Return(shortRow(ids[1]), nil)
		mockDBBin.EXPECT().ExtensionList(ctx, uint64(migrationExtensionListLimit), "", gomock.Any()).Return([]*extension.Extension{}, nil)
		// customer 3: lookup fails -> the batch stops; customer 4 has no expectation
		mockDBBin.EXPECT().CustomerDomainGet(ctx, ids[2]).Return(nil, errInjectedStep)

		migrated, skipped, err := domainMigrate(ctx, mockDBAst, mockDBBin, mockCD, mockNotify, false, uuid.Nil, nil, &bytes.Buffer{})
		if err == nil || !errors.Is(err, errInjectedStep) || !strings.Contains(err.Error(), ids[2].String()) {
			t.Errorf("Wrong match. expect: an error naming the third customer, got: %v", err)
		}
		if migrated != 1 || skipped != 1 {
			t.Errorf("Wrong match. expect: 1 migrated and 1 skipped, got: %d, %d", migrated, skipped)
		}
	})
}
