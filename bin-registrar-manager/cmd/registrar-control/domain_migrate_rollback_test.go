package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/pkg/notifyhandler"

	"monorepo/bin-registrar-manager/models/customerdomain"
	"monorepo/bin-registrar-manager/pkg/customerdomainhandler"
	"monorepo/bin-registrar-manager/pkg/dbhandler"
)

func Test_domainMigrateRollback_failures(t *testing.T) {
	customerID := uuid.FromStringOrNil("99999999-7f85-11ee-8f5a-1b2c3d4e5f60")
	extID := uuid.FromStringOrNil("aaaaaaaa-7f85-11ee-8f5a-1b2c3d4e5f60")
	oldRealm := "old1.reg.example.test"
	newRealm := "new1.reg.example.test"

	errInjected := errors.New("injected failure")

	newRecord := func() *migrationRecord {
		return &migrationRecord{
			CustomerID: customerID,
			OldLabel:   "old1",
			NewLabel:   "new1",
			OldRealm:   oldRealm,
			NewRealm:   newRealm,
			Extensions: []extensionMigrationRecord{{ID: extID, Extension: "1001", OldRealm: oldRealm, NewRealm: newRealm}},
		}
	}

	tests := []struct {
		name      string
		record    func() *migrationRecord
		setup     func(ctx context.Context, cd *customerdomainhandler.MockCustomerDomainHandler, bin *dbhandler.MockDBHandler, ast *dbhandler.MockDBHandler)
		expectErr error
		expectMsg string
	}{
		{
			name: "record without old label is rejected before any call",
			record: func() *migrationRecord {
				r := newRecord()
				r.OldLabel = ""
				return r
			},
			setup: func(ctx context.Context, cd *customerdomainhandler.MockCustomerDomainHandler, bin *dbhandler.MockDBHandler, ast *dbhandler.MockDBHandler) {
			},
			expectMsg: "missing old_label/old_realm",
		},
		{
			name: "record without old realm is rejected before any call",
			record: func() *migrationRecord {
				r := newRecord()
				r.OldRealm = ""
				return r
			},
			setup: func(ctx context.Context, cd *customerdomainhandler.MockCustomerDomainHandler, bin *dbhandler.MockDBHandler, ast *dbhandler.MockDBHandler) {
			},
			expectMsg: "missing old_label/old_realm",
		},
		{
			name:   "domain row restore failure stops the rollback",
			record: newRecord,
			setup: func(ctx context.Context, cd *customerdomainhandler.MockCustomerDomainHandler, bin *dbhandler.MockDBHandler, ast *dbhandler.MockDBHandler) {
				cd.EXPECT().UpdateByCustomerID(ctx, customerID, "old1", oldRealm).Return(nil, errInjected)
			},
			expectErr: errInjected,
			expectMsg: "could not restore the customer domain row",
		},
		{
			name:   "extension lookup failure other than not found is returned",
			record: newRecord,
			setup: func(ctx context.Context, cd *customerdomainhandler.MockCustomerDomainHandler, bin *dbhandler.MockDBHandler, ast *dbhandler.MockDBHandler) {
				cd.EXPECT().UpdateByCustomerID(ctx, customerID, "old1", oldRealm).Return(&customerdomain.CustomerDomain{}, nil)
				bin.EXPECT().ExtensionGet(ctx, extID).Return(nil, errInjected)
			},
			expectErr: errInjected,
			expectMsg: "could not get the extension",
		},
		{
			name:   "reprovision failure is returned",
			record: newRecord,
			setup: func(ctx context.Context, cd *customerdomainhandler.MockCustomerDomainHandler, bin *dbhandler.MockDBHandler, ast *dbhandler.MockDBHandler) {
				cd.EXPECT().UpdateByCustomerID(ctx, customerID, "old1", oldRealm).Return(&customerdomain.CustomerDomain{}, nil)
				bin.EXPECT().ExtensionGet(ctx, extID).Return(testExtension(customerID, "1001", newRealm), nil)
				ast.EXPECT().AstEndpointDelete(ctx, gomock.Any()).Return(errInjected)
			},
			expectErr: errInjected,
			expectMsg: "could not roll back the extension",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCD := customerdomainhandler.NewMockCustomerDomainHandler(mc)
			mockBin := dbhandler.NewMockDBHandler(mc)
			mockAst := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			ctx := context.Background()

			tt.setup(ctx, mockCD, mockBin, mockAst)

			var out bytes.Buffer
			count, err := domainMigrateRollback(ctx, mockAst, mockBin, mockCD, mockNotify, true, []*migrationRecord{tt.record()}, &out)
			if err == nil {
				t.Fatalf("expected an error, got nil")
			}
			if tt.expectErr != nil && !errors.Is(err, tt.expectErr) {
				t.Errorf("expected the injected failure to be wrapped, got: %v", err)
			}
			if !strings.Contains(err.Error(), tt.expectMsg) {
				t.Errorf("expected message containing %q, got: %v", tt.expectMsg, err)
			}
			if count != 0 {
				t.Errorf("expected nothing rolled back, got: %d", count)
			}
		})
	}
}

func Test_domainMigrateRollback_missingExtensionIsSkipped(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCD := customerdomainhandler.NewMockCustomerDomainHandler(mc)
	mockBin := dbhandler.NewMockDBHandler(mc)
	mockAst := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("99999999-7f85-11ee-8f5a-1b2c3d4e5f60")
	extID := uuid.FromStringOrNil("aaaaaaaa-7f85-11ee-8f5a-1b2c3d4e5f60")
	record := &migrationRecord{
		CustomerID: customerID,
		OldLabel:   "old1",
		OldRealm:   "old1.reg.example.test",
		NewRealm:   "new1.reg.example.test",
		Extensions: []extensionMigrationRecord{{ID: extID, OldRealm: "old1.reg.example.test"}},
	}

	mockCD.EXPECT().UpdateByCustomerID(ctx, customerID, "old1", record.OldRealm).Return(&customerdomain.CustomerDomain{}, nil)
	mockBin.EXPECT().ExtensionGet(ctx, extID).Return(nil, dbhandler.ErrNotFound)

	var out bytes.Buffer
	count, err := domainMigrateRollback(ctx, mockAst, mockBin, mockCD, mockNotify, true, []*migrationRecord{record}, &out)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 customer rolled back, got: %d", count)
	}
	if !strings.Contains(out.String(), "no longer exists; skipping") {
		t.Errorf("expected a skip warning, got: %q", out.String())
	}
}

func Test_domainMigrateRollback_dryRunMakesNoChanges(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	// no expectations: any handler call fails the test
	mockCD := customerdomainhandler.NewMockCustomerDomainHandler(mc)
	mockBin := dbhandler.NewMockDBHandler(mc)
	mockAst := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	records := []*migrationRecord{
		{CustomerID: uuid.FromStringOrNil("99999999-7f85-11ee-8f5a-1b2c3d4e5f60"), OldLabel: "a", OldRealm: "a.reg.example.test", NewRealm: "b.reg.example.test"},
		{CustomerID: uuid.FromStringOrNil("88888888-7f85-11ee-8f5a-1b2c3d4e5f60"), OldLabel: "c", OldRealm: "c.reg.example.test", NewRealm: "d.reg.example.test"},
	}

	var out bytes.Buffer
	count, err := domainMigrateRollback(context.Background(), mockAst, mockBin, mockCD, mockNotify, false, records, &out)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 counted, got: %d", count)
	}
	if strings.Count(out.String(), "[dry-run] would roll back customer") != 2 {
		t.Errorf("expected two dry-run lines, got: %q", out.String())
	}
	// reverse order: the last record is reported first
	if strings.Index(out.String(), "c.reg.example.test") > strings.Index(out.String(), "a.reg.example.test") {
		t.Errorf("expected reverse replay order, got: %q", out.String())
	}
}
