package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-registrar-manager/models/astaor"
	"monorepo/bin-registrar-manager/models/astauth"
	"monorepo/bin-registrar-manager/models/astendpoint"
	"monorepo/bin-registrar-manager/models/customerdomain"
	"monorepo/bin-registrar-manager/models/extension"
	"monorepo/bin-registrar-manager/models/sipauth"
	"monorepo/bin-registrar-manager/pkg/customerdomainhandler"
	"monorepo/bin-registrar-manager/pkg/dbhandler"
)

// ============================================================================
// reprovisionExtension — identity invariants
// ============================================================================

func testExtension(customerID uuid.UUID, ext string, realm string) *extension.Extension {
	endpoint := ext + "@" + realm

	return &extension.Extension{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("aaaaaaaa-7f85-11ee-8f5a-1b2c3d4e5f60"),
			CustomerID: customerID,
		},

		Name:   "test name",
		Detail: "test detail",

		EndpointID: endpoint,
		AORID:      endpoint,
		AuthID:     endpoint,

		Extension:  ext,
		DomainName: customerID.String(),

		Realm:    realm,
		Username: ext,
		Password: "secret-password",

		DirectID:   uuid.FromStringOrNil("dddddddd-7f85-11ee-8f5a-1b2c3d4e5f60"),
		DirectHash: "direct-hash-value",
	}
}

func Test_reprovisionExtension_identityInvariants(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDBAst := dbhandler.NewMockDBHandler(mc)
	mockDBBin := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("55555555-7f85-11ee-8f5a-1b2c3d4e5f60")
	oldRealm := customerID.String() + ".registrar.voipbin.net"
	newRealm := "ab12.reg.voipbin.net"
	ext := testExtension(customerID, "1001", oldRealm)
	oldEndpoint := ext.EndpointID
	newEndpoint := "1001@" + newRealm

	// 1. ps_* deletes: old ids AND (defensively) new ids
	mockDBAst.EXPECT().AstEndpointDelete(ctx, oldEndpoint).Return(nil)
	mockDBAst.EXPECT().AstEndpointDelete(ctx, newEndpoint).Return(nil)
	mockDBAst.EXPECT().AstAuthDelete(ctx, oldEndpoint).Return(nil)
	mockDBAst.EXPECT().AstAuthDelete(ctx, newEndpoint).Return(nil)
	mockDBAst.EXPECT().AstAORDelete(ctx, oldEndpoint).Return(nil)
	mockDBAst.EXPECT().AstAORDelete(ctx, newEndpoint).Return(nil)

	// 2. ps_* creates: new ids, SAME username/password, new realm
	mockDBAst.EXPECT().AstAORCreate(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, aor *astaor.AstAOR) error {
		if *aor.ID != newEndpoint {
			t.Errorf("Wrong match. expect aor id: %s, got: %s", newEndpoint, *aor.ID)
		}
		if *aor.MaxContacts != migrationDefaultMaxContacts || *aor.RemoveExisting != migrationDefaultRemoveExisting {
			t.Errorf("Wrong match. aor defaults changed: %v %v", *aor.MaxContacts, *aor.RemoveExisting)
		}
		return nil
	})
	mockDBAst.EXPECT().AstAuthCreate(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, auth *astauth.AstAuth) error {
		if *auth.ID != newEndpoint {
			t.Errorf("Wrong match. expect auth id: %s, got: %s", newEndpoint, *auth.ID)
		}
		if *auth.Username != "1001" {
			t.Errorf("Wrong match. username changed: %s", *auth.Username)
		}
		if *auth.Password != "secret-password" {
			t.Errorf("Wrong match. password changed: %s", *auth.Password)
		}
		if *auth.Realm != newRealm {
			t.Errorf("Wrong match. expect realm: %s, got: %s", newRealm, *auth.Realm)
		}
		if *auth.AuthType != migrationDefaultAuthType {
			t.Errorf("Wrong match. auth type changed: %s", *auth.AuthType)
		}
		return nil
	})
	mockDBAst.EXPECT().AstEndpointCreate(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, ep *astendpoint.AstEndpoint) error {
		if *ep.ID != newEndpoint || *ep.AORs != newEndpoint || *ep.Auth != newEndpoint {
			t.Errorf("Wrong match. endpoint ids: %s %s %s", *ep.ID, *ep.AORs, *ep.Auth)
		}
		return nil
	})

	// 3. sipauth realm update on the extension id
	mockDBBin.EXPECT().SIPAuthUpdate(ctx, ext.ID, map[sipauth.Field]any{
		sipauth.FieldRealm: newRealm,
	}).Return(nil)

	// 4. stale contact purge for the OLD endpoint
	mockDBAst.EXPECT().AstContactDeleteByEndpoint(ctx, oldEndpoint).Return(nil)

	// 5. extension update IN PLACE: ONLY realm/domain_name/endpoint_id/aor_id/auth_id.
	//    id/extension/username/password/direct_id/direct_hash are NOT in the field map.
	mockDBBin.EXPECT().ExtensionUpdate(ctx, ext.ID, gomock.Any()).DoAndReturn(func(_ context.Context, id uuid.UUID, fields map[extension.Field]any) error {
		expectFields := map[extension.Field]any{
			extension.FieldRealm:      newRealm,
			extension.FieldDomainName: newRealm,
			extension.FieldEndpointID: newEndpoint,
			extension.FieldAORID:      newEndpoint,
			extension.FieldAuthID:     newEndpoint,
		}
		if len(fields) != len(expectFields) {
			t.Errorf("Wrong match. expect: %d fields, got: %d (%v)", len(expectFields), len(fields), fields)
		}
		for k, v := range expectFields {
			if fields[k] != v {
				t.Errorf("Wrong match. field %s: expect %v, got %v", k, v, fields[k])
			}
		}
		for _, forbidden := range []extension.Field{
			extension.FieldID, extension.FieldExtension, extension.FieldUsername,
			extension.FieldPassword, extension.FieldDirectID, extension.FieldDirectHash,
		} {
			if _, ok := fields[forbidden]; ok {
				t.Errorf("Wrong match. identity field %s must not be updated", forbidden)
			}
		}
		return nil
	})

	// 6. exactly ONE extension_updated event (gomock default Times(1); no
	//    created/deleted event expectations exist, so those calls would fail)
	updated := testExtension(customerID, "1001", newRealm)
	mockDBBin.EXPECT().ExtensionGet(ctx, ext.ID).Return(updated, nil)
	mockNotify.EXPECT().PublishEvent(ctx, extension.EventTypeExtensionUpdated, updated).Times(1)

	// NOTE: no requesthandler is involved at all — the re-provision path is
	// structurally incapable of making a billing RPC.
	record, err := reprovisionExtension(ctx, mockDBAst, mockDBBin, mockNotify, ext, newRealm)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	if record == nil {
		t.Fatalf("Wrong match. expect: record, got: nil")
	}
	if record.ID != ext.ID || record.OldRealm != oldRealm || record.NewRealm != newRealm {
		t.Errorf("Wrong match. record: %+v", record)
	}
	if record.OldEndpointID != oldEndpoint || record.NewEndpointID != newEndpoint {
		t.Errorf("Wrong match. record endpoints: %+v", record)
	}
}

func Test_reprovisionExtension_idempotentSkip(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDBAst := dbhandler.NewMockDBHandler(mc)
	mockDBBin := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("66666666-7f85-11ee-8f5a-1b2c3d4e5f60")
	newRealm := "cd34.reg.voipbin.net"
	ext := testExtension(customerID, "1001", newRealm) // already on the target realm

	// no expectations at all: ANY db/notify call fails the test — a re-run
	// publishes zero additional events for already-migrated extensions
	record, err := reprovisionExtension(ctx, mockDBAst, mockDBBin, mockNotify, ext, newRealm)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if record != nil {
		t.Errorf("Wrong match. expect: nil record (skip), got: %+v", record)
	}
}

func Test_reprovisionExtension_legacyNullRealmRow(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDBAst := dbhandler.NewMockDBHandler(mc)
	mockDBBin := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("77777777-7f85-11ee-8f5a-1b2c3d4e5f60")
	oldRealm := customerID.String() + ".registrar.voipbin.net"
	newRealm := "ef56.reg.voipbin.net"

	// legacy pre-Feb-2024 row: realm column NULL, endpoint ids present
	ext := testExtension(customerID, "1001", oldRealm)
	ext.Realm = ""

	mockDBAst.EXPECT().AstEndpointDelete(ctx, gomock.Any()).Return(nil).Times(2)
	mockDBAst.EXPECT().AstAuthDelete(ctx, gomock.Any()).Return(nil).Times(2)
	mockDBAst.EXPECT().AstAORDelete(ctx, gomock.Any()).Return(nil).Times(2)
	mockDBAst.EXPECT().AstAORCreate(ctx, gomock.Any()).Return(nil)
	mockDBAst.EXPECT().AstAuthCreate(ctx, gomock.Any()).Return(nil)
	mockDBAst.EXPECT().AstEndpointCreate(ctx, gomock.Any()).Return(nil)
	mockDBBin.EXPECT().SIPAuthUpdate(ctx, ext.ID, gomock.Any()).Return(nil)
	mockDBAst.EXPECT().AstContactDeleteByEndpoint(ctx, ext.EndpointID).Return(nil)
	mockDBBin.EXPECT().ExtensionUpdate(ctx, ext.ID, gomock.Any()).Return(nil)
	updated := testExtension(customerID, "1001", newRealm)
	mockDBBin.EXPECT().ExtensionGet(ctx, ext.ID).Return(updated, nil)
	mockNotify.EXPECT().PublishEvent(ctx, extension.EventTypeExtensionUpdated, updated).Times(1)

	record, err := reprovisionExtension(ctx, mockDBAst, mockDBBin, mockNotify, ext, newRealm)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	// the record's old realm is recovered from the endpoint id
	if record.OldRealm != oldRealm {
		t.Errorf("Wrong match. expect: %s, got: %s", oldRealm, record.OldRealm)
	}
}

// ============================================================================
// migrateCustomerDomain / domainMigrate
// ============================================================================

func Test_migrateCustomerDomain_execute(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDBAst := dbhandler.NewMockDBHandler(mc)
	mockDBBin := dbhandler.NewMockDBHandler(mc)
	mockCD := customerdomainhandler.NewMockCustomerDomainHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("88888888-7f85-11ee-8f5a-1b2c3d4e5f60")
	oldRealm := customerID.String() + ".registrar.voipbin.net"
	newRealm := "gh78.reg.voipbin.net"

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

	ext := testExtension(customerID, "1001", oldRealm)

	mockDBBin.EXPECT().CustomerDomainGet(ctx, customerID).Return(backfilledRow, nil)
	mockDBBin.EXPECT().ExtensionList(ctx, uint64(migrationExtensionListLimit), "", map[extension.Field]any{
		extension.FieldCustomerID: customerID,
		extension.FieldDeleted:    false,
	}).Return([]*extension.Extension{ext}, nil)
	mockCD.EXPECT().MigrateToShortDomain(ctx, customerID).Return(migratedRow, nil)

	// re-provision plumbing (details covered by the reprovision tests)
	mockDBAst.EXPECT().AstEndpointDelete(ctx, gomock.Any()).Return(nil).Times(2)
	mockDBAst.EXPECT().AstAuthDelete(ctx, gomock.Any()).Return(nil).Times(2)
	mockDBAst.EXPECT().AstAORDelete(ctx, gomock.Any()).Return(nil).Times(2)
	mockDBAst.EXPECT().AstAORCreate(ctx, gomock.Any()).Return(nil)
	mockDBAst.EXPECT().AstAuthCreate(ctx, gomock.Any()).Return(nil)
	mockDBAst.EXPECT().AstEndpointCreate(ctx, gomock.Any()).Return(nil)
	mockDBBin.EXPECT().SIPAuthUpdate(ctx, ext.ID, gomock.Any()).Return(nil)
	mockDBAst.EXPECT().AstContactDeleteByEndpoint(ctx, ext.EndpointID).Return(nil)
	mockDBBin.EXPECT().ExtensionUpdate(ctx, ext.ID, gomock.Any()).Return(nil)
	updated := testExtension(customerID, "1001", newRealm)
	mockDBBin.EXPECT().ExtensionGet(ctx, ext.ID).Return(updated, nil)
	mockNotify.EXPECT().PublishEvent(ctx, extension.EventTypeExtensionUpdated, updated).Times(1)

	out := &bytes.Buffer{}
	logBuf := &bytes.Buffer{}
	record, skipped, err := migrateCustomerDomain(ctx, mockDBAst, mockDBBin, mockCD, mockNotify, customerID, true, logBuf, out)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if skipped {
		t.Fatalf("Wrong match. expect: migrated, got: skipped")
	}

	if record.OldLabel != customerID.String() || record.OldRealm != oldRealm {
		t.Errorf("Wrong match. record old: %+v", record)
	}
	if record.NewLabel != "gh78" || record.NewRealm != newRealm {
		t.Errorf("Wrong match. record new: %+v", record)
	}
	if len(record.Extensions) != 1 {
		t.Fatalf("Wrong match. expect: 1 extension record, got: %d", len(record.Extensions))
	}
	if record.Extensions[0].OldRealm != oldRealm || record.Extensions[0].NewRealm != newRealm {
		t.Errorf("Wrong match. extension record: %+v", record.Extensions[0])
	}
}

func Test_migrateCustomerDomain_alreadyMigratedIsSkipped(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDBAst := dbhandler.NewMockDBHandler(mc)
	mockDBBin := dbhandler.NewMockDBHandler(mc)
	mockCD := customerdomainhandler.NewMockCustomerDomainHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("99999999-7f85-11ee-8f5a-1b2c3d4e5f60")
	newRealm := "ij90.reg.voipbin.net"

	shortRow := &customerdomain.CustomerDomain{
		CustomerID:  customerID,
		DomainLabel: "ij90",
		Realm:       newRealm,
	}
	ext := testExtension(customerID, "1001", newRealm)
	ext.EndpointID = "1001@" + newRealm

	mockDBBin.EXPECT().CustomerDomainGet(ctx, customerID).Return(shortRow, nil)
	mockDBBin.EXPECT().ExtensionList(ctx, uint64(migrationExtensionListLimit), "", gomock.Any()).Return([]*extension.Extension{ext}, nil)
	// no MigrateToShortDomain / re-provision / event expectations: any call fails

	out := &bytes.Buffer{}
	logBuf := &bytes.Buffer{}
	record, skipped, err := migrateCustomerDomain(ctx, mockDBAst, mockDBBin, mockCD, mockNotify, customerID, true, logBuf, out)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !skipped {
		t.Errorf("Wrong match. expect: skipped, got: migrated (%+v)", record)
	}
	if logBuf.Len() != 0 {
		t.Errorf("Wrong match. expect: no log line for a skipped customer, got: %s", logBuf.String())
	}
}

func Test_migrateCustomerDomain_resumeRecoversOldRealmFromExtension(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDBAst := dbhandler.NewMockDBHandler(mc)
	mockDBBin := dbhandler.NewMockDBHandler(mc)
	mockCD := customerdomainhandler.NewMockCustomerDomainHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("ababab00-7f85-11ee-8f5a-1b2c3d4e5f60")
	oldRealm := customerID.String() + ".registrar.voipbin.net"
	newRealm := "kl12.reg.voipbin.net"

	// resume: the domain row is ALREADY short (previous run crashed
	// mid-customer) but one extension still carries the old realm
	shortRow := &customerdomain.CustomerDomain{
		CustomerID:  customerID,
		DomainLabel: "kl12",
		Realm:       newRealm,
	}
	pendingExt := testExtension(customerID, "1001", oldRealm)

	mockDBBin.EXPECT().CustomerDomainGet(ctx, customerID).Return(shortRow, nil)
	mockDBBin.EXPECT().ExtensionList(ctx, uint64(migrationExtensionListLimit), "", gomock.Any()).Return([]*extension.Extension{pendingExt}, nil)
	mockCD.EXPECT().MigrateToShortDomain(ctx, customerID).Return(shortRow, nil)

	mockDBAst.EXPECT().AstEndpointDelete(ctx, gomock.Any()).Return(nil).Times(2)
	mockDBAst.EXPECT().AstAuthDelete(ctx, gomock.Any()).Return(nil).Times(2)
	mockDBAst.EXPECT().AstAORDelete(ctx, gomock.Any()).Return(nil).Times(2)
	mockDBAst.EXPECT().AstAORCreate(ctx, gomock.Any()).Return(nil)
	mockDBAst.EXPECT().AstAuthCreate(ctx, gomock.Any()).Return(nil)
	mockDBAst.EXPECT().AstEndpointCreate(ctx, gomock.Any()).Return(nil)
	mockDBBin.EXPECT().SIPAuthUpdate(ctx, pendingExt.ID, gomock.Any()).Return(nil)
	mockDBAst.EXPECT().AstContactDeleteByEndpoint(ctx, pendingExt.EndpointID).Return(nil)
	mockDBBin.EXPECT().ExtensionUpdate(ctx, pendingExt.ID, gomock.Any()).Return(nil)
	updated := testExtension(customerID, "1001", newRealm)
	mockDBBin.EXPECT().ExtensionGet(ctx, pendingExt.ID).Return(updated, nil)
	mockNotify.EXPECT().PublishEvent(ctx, extension.EventTypeExtensionUpdated, updated).Times(1)

	out := &bytes.Buffer{}
	logBuf := &bytes.Buffer{}
	record, skipped, err := migrateCustomerDomain(ctx, mockDBAst, mockDBBin, mockCD, mockNotify, customerID, true, logBuf, out)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if skipped {
		t.Fatalf("Wrong match. expect: migrated, got: skipped")
	}

	// the TRUE old realm is recovered from the pending extension, not the
	// already-migrated domain row
	if record.OldRealm != oldRealm {
		t.Errorf("Wrong match. expect: %s, got: %s", oldRealm, record.OldRealm)
	}
	if record.OldLabel != customerID.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", customerID.String(), record.OldLabel)
	}
}

func Test_migrateCustomerDomain_dryRunMakesNoWrites(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDBAst := dbhandler.NewMockDBHandler(mc)
	mockDBBin := dbhandler.NewMockDBHandler(mc)
	mockCD := customerdomainhandler.NewMockCustomerDomainHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("cdcdcd00-7f85-11ee-8f5a-1b2c3d4e5f60")
	oldRealm := customerID.String() + ".registrar.voipbin.net"
	backfilledRow := &customerdomain.CustomerDomain{
		CustomerID:  customerID,
		DomainLabel: customerID.String(),
		Realm:       oldRealm,
	}
	ext := testExtension(customerID, "1001", oldRealm)

	// reads only; any write/event call fails the test
	mockDBBin.EXPECT().CustomerDomainGet(ctx, customerID).Return(backfilledRow, nil)
	mockDBBin.EXPECT().ExtensionList(ctx, uint64(migrationExtensionListLimit), "", gomock.Any()).Return([]*extension.Extension{ext}, nil)

	out := &bytes.Buffer{}
	logBuf := &bytes.Buffer{}
	record, skipped, err := migrateCustomerDomain(ctx, mockDBAst, mockDBBin, mockCD, mockNotify, customerID, false, logBuf, out)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if skipped {
		t.Fatalf("Wrong match. expect: not skipped, got: skipped")
	}
	if record.OldRealm != oldRealm {
		t.Errorf("Wrong match. expect: %s, got: %s", oldRealm, record.OldRealm)
	}
	if !strings.Contains(out.String(), "[dry-run]") {
		t.Errorf("Wrong match. expect: dry-run output, got: %s", out.String())
	}
	if logBuf.Len() != 0 {
		t.Errorf("Wrong match. expect: no log writes on dry-run, got: %s", logBuf.String())
	}
}

// Test_migrateCustomerDomain_noRowDerivesOldRealmFromStoredFields: with no
// domain row, the customer's old realm comes from the STORED extension fields
// (post-cutover there is no computed legacy fallback).
func Test_migrateCustomerDomain_noRowDerivesOldRealmFromStoredFields(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDBAst := dbhandler.NewMockDBHandler(mc)
	mockDBBin := dbhandler.NewMockDBHandler(mc)
	mockCD := customerdomainhandler.NewMockCustomerDomainHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("dedede00-7f85-11ee-8f5a-1b2c3d4e5f60")
	oldRealm := customerID.String() + ".registrar.voipbin.net" // stored on the extension row
	newRealm := "zz88.reg.voipbin.net"

	migratedRow := &customerdomain.CustomerDomain{
		CustomerID:  customerID,
		DomainLabel: "zz88",
		Realm:       newRealm,
	}
	ext := testExtension(customerID, "1001", oldRealm)

	mockDBBin.EXPECT().CustomerDomainGet(ctx, customerID).Return(nil, dbhandler.ErrNotFound)
	mockDBBin.EXPECT().ExtensionList(ctx, uint64(migrationExtensionListLimit), "", gomock.Any()).Return([]*extension.Extension{ext}, nil)
	mockCD.EXPECT().MigrateToShortDomain(ctx, customerID).Return(migratedRow, nil)

	mockDBAst.EXPECT().AstEndpointDelete(ctx, gomock.Any()).Return(nil).Times(2)
	mockDBAst.EXPECT().AstAuthDelete(ctx, gomock.Any()).Return(nil).Times(2)
	mockDBAst.EXPECT().AstAORDelete(ctx, gomock.Any()).Return(nil).Times(2)
	mockDBAst.EXPECT().AstAORCreate(ctx, gomock.Any()).Return(nil)
	mockDBAst.EXPECT().AstAuthCreate(ctx, gomock.Any()).Return(nil)
	mockDBAst.EXPECT().AstEndpointCreate(ctx, gomock.Any()).Return(nil)
	mockDBBin.EXPECT().SIPAuthUpdate(ctx, ext.ID, gomock.Any()).Return(nil)
	mockDBAst.EXPECT().AstContactDeleteByEndpoint(ctx, ext.EndpointID).Return(nil)
	mockDBBin.EXPECT().ExtensionUpdate(ctx, ext.ID, gomock.Any()).Return(nil)
	updated := testExtension(customerID, "1001", newRealm)
	mockDBBin.EXPECT().ExtensionGet(ctx, ext.ID).Return(updated, nil)
	mockNotify.EXPECT().PublishEvent(ctx, extension.EventTypeExtensionUpdated, updated).Times(1)

	out := &bytes.Buffer{}
	logBuf := &bytes.Buffer{}
	record, skipped, err := migrateCustomerDomain(ctx, mockDBAst, mockDBBin, mockCD, mockNotify, customerID, true, logBuf, out)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if skipped {
		t.Fatalf("Wrong match. expect: migrated, got: skipped")
	}

	// old realm/label recovered from the extension's STORED realm
	if record.OldRealm != oldRealm {
		t.Errorf("Wrong match. expect: %s, got: %s", oldRealm, record.OldRealm)
	}
	if record.OldLabel != customerID.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", customerID.String(), record.OldLabel)
	}
}

// Test_migrateCustomerDomain_noRowNoRecoverableRealm_errors: extensions exist
// but carry no recoverable realm (no stored realm, endpoint id without a
// domain part) — the batch refuses instead of writing a rollback record that
// cannot restore them.
func Test_migrateCustomerDomain_noRowNoRecoverableRealm_errors(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDBAst := dbhandler.NewMockDBHandler(mc)
	mockDBBin := dbhandler.NewMockDBHandler(mc)
	mockCD := customerdomainhandler.NewMockCustomerDomainHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("fafafa00-7f85-11ee-8f5a-1b2c3d4e5f60")

	ext := testExtension(customerID, "1001", "")
	ext.Realm = ""
	ext.EndpointID = "1001" // no domain part

	mockDBBin.EXPECT().CustomerDomainGet(ctx, customerID).Return(nil, dbhandler.ErrNotFound)
	mockDBBin.EXPECT().ExtensionList(ctx, uint64(migrationExtensionListLimit), "", gomock.Any()).Return([]*extension.Extension{ext}, nil)
	// no MigrateToShortDomain / write expectations: any mutation fails the test

	out := &bytes.Buffer{}
	logBuf := &bytes.Buffer{}
	if _, _, err := migrateCustomerDomain(ctx, mockDBAst, mockDBBin, mockCD, mockNotify, customerID, true, logBuf, out); err == nil {
		t.Fatalf("Wrong match. expect: error, got: ok")
	}
	if logBuf.Len() != 0 {
		t.Errorf("Wrong match. expect: no log writes, got: %s", logBuf.String())
	}
}

// Test_migrateCustomerDomain_noRowNoExtensions_noRollbackRecord: a customer
// with no previous domain state at all gets a fresh short row and NO rollback
// record (there is nothing to restore); only the completion marker is logged.
func Test_migrateCustomerDomain_noRowNoExtensions_noRollbackRecord(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDBAst := dbhandler.NewMockDBHandler(mc)
	mockDBBin := dbhandler.NewMockDBHandler(mc)
	mockCD := customerdomainhandler.NewMockCustomerDomainHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("bcbcbc00-7f85-11ee-8f5a-1b2c3d4e5f60")
	shortRow := &customerdomain.CustomerDomain{
		CustomerID:  customerID,
		DomainLabel: "yy77",
		Realm:       "yy77.reg.voipbin.net",
	}

	mockDBBin.EXPECT().CustomerDomainGet(ctx, customerID).Return(nil, dbhandler.ErrNotFound)
	mockDBBin.EXPECT().ExtensionList(ctx, uint64(migrationExtensionListLimit), "", gomock.Any()).Return([]*extension.Extension{}, nil)
	mockCD.EXPECT().MigrateToShortDomain(ctx, customerID).Return(shortRow, nil)

	out := &bytes.Buffer{}
	logBuf := &bytes.Buffer{}
	record, skipped, err := migrateCustomerDomain(ctx, mockDBAst, mockDBBin, mockCD, mockNotify, customerID, true, logBuf, out)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if skipped {
		t.Fatalf("Wrong match. expect: migrated, got: skipped")
	}
	if record.NewRealm != shortRow.Realm {
		t.Errorf("Wrong match. expect: %s, got: %s", shortRow.Realm, record.NewRealm)
	}

	// the completion marker is present but the rollback parser sees NO records
	if !strings.Contains(logBuf.String(), `"status":"completed"`) {
		t.Errorf("Wrong match. expect: completion marker, got: %s", logBuf.String())
	}
	records, errParse := parseMigrationLog(bytes.NewReader(logBuf.Bytes()))
	if errParse != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", errParse)
	}
	if len(records) != 0 {
		t.Errorf("Wrong match. expect: 0 rollback records, got: %+v", records)
	}
}

func Test_domainMigrate_writesJSONLinesLog(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDBAst := dbhandler.NewMockDBHandler(mc)
	mockDBBin := dbhandler.NewMockDBHandler(mc)
	mockCD := customerdomainhandler.NewMockCustomerDomainHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("efefef00-7f85-11ee-8f5a-1b2c3d4e5f60")
	oldRealm := customerID.String() + ".registrar.voipbin.net"
	newRealm := "mn34.reg.voipbin.net"

	backfilledRow := &customerdomain.CustomerDomain{
		CustomerID:  customerID,
		DomainLabel: customerID.String(),
		Realm:       oldRealm,
	}
	migratedRow := &customerdomain.CustomerDomain{
		CustomerID:  customerID,
		DomainLabel: "mn34",
		Realm:       newRealm,
	}
	ext := testExtension(customerID, "1001", oldRealm)

	mockDBBin.EXPECT().CustomerDomainGet(ctx, customerID).Return(backfilledRow, nil)
	mockDBBin.EXPECT().ExtensionList(ctx, uint64(migrationExtensionListLimit), "", gomock.Any()).Return([]*extension.Extension{ext}, nil)
	mockCD.EXPECT().MigrateToShortDomain(ctx, customerID).Return(migratedRow, nil)
	mockDBAst.EXPECT().AstEndpointDelete(ctx, gomock.Any()).Return(nil).Times(2)
	mockDBAst.EXPECT().AstAuthDelete(ctx, gomock.Any()).Return(nil).Times(2)
	mockDBAst.EXPECT().AstAORDelete(ctx, gomock.Any()).Return(nil).Times(2)
	mockDBAst.EXPECT().AstAORCreate(ctx, gomock.Any()).Return(nil)
	mockDBAst.EXPECT().AstAuthCreate(ctx, gomock.Any()).Return(nil)
	mockDBAst.EXPECT().AstEndpointCreate(ctx, gomock.Any()).Return(nil)
	mockDBBin.EXPECT().SIPAuthUpdate(ctx, ext.ID, gomock.Any()).Return(nil)
	mockDBAst.EXPECT().AstContactDeleteByEndpoint(ctx, ext.EndpointID).Return(nil)
	mockDBBin.EXPECT().ExtensionUpdate(ctx, ext.ID, gomock.Any()).Return(nil)
	updated := testExtension(customerID, "1001", newRealm)
	mockDBBin.EXPECT().ExtensionGet(ctx, ext.ID).Return(updated, nil)
	mockNotify.EXPECT().PublishEvent(ctx, extension.EventTypeExtensionUpdated, updated).Times(1)

	logBuffer := &bytes.Buffer{}
	out := &bytes.Buffer{}
	migrated, skipped, err := domainMigrate(ctx, mockDBAst, mockDBBin, mockCD, mockNotify, true, customerID, logBuffer, out)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if migrated != 1 || skipped != 0 {
		t.Errorf("Wrong match. expect: migrated=1 skipped=0, got: migrated=%d skipped=%d", migrated, skipped)
	}

	// the completion marker is present for the operator...
	if !strings.Contains(logBuffer.String(), `"status":"completed"`) {
		t.Errorf("Wrong match. expect: completion marker in the log, got: %s", logBuffer.String())
	}

	// ...and the log round-trips through the rollback parser, which ignores it
	records, err := parseMigrationLog(logBuffer)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("Wrong match. expect: 1 record, got: %d", len(records))
	}
	record := records[0]
	if record.CustomerID != customerID || record.OldRealm != oldRealm || record.NewRealm != newRealm {
		t.Errorf("Wrong match. record: %+v", record)
	}
	if record.OldLabel != customerID.String() || record.NewLabel != "mn34" {
		t.Errorf("Wrong match. record labels: %+v", record)
	}
	if len(record.Extensions) != 1 || record.Extensions[0].OldRealm != oldRealm {
		t.Errorf("Wrong match. record extensions: %+v", record.Extensions)
	}
}

// ============================================================================
// rollback
// ============================================================================

func Test_parseMigrationLog(t *testing.T) {
	customerID := uuid.FromStringOrNil("0f0f0f00-7f86-11ee-8f5a-1b2c3d4e5f60")

	record := &migrationRecord{
		CustomerID: customerID,
		OldLabel:   customerID.String(),
		NewLabel:   "op56",
		OldRealm:   customerID.String() + ".registrar.voipbin.net",
		NewRealm:   "op56.reg.voipbin.net",
		Extensions: []extensionMigrationRecord{},
	}
	line, _ := json.Marshal(record)

	// valid log with a blank line
	input := string(line) + "\n\n" + string(line) + "\n"
	records, err := parseMigrationLog(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if len(records) != 2 {
		t.Errorf("Wrong match. expect: 2 records, got: %d", len(records))
	}

	// completion markers (and any status-carrying line) are ignored gracefully
	withMarker := string(line) + "\n" + `{"customer_id":"` + customerID.String() + `","status":"completed"}` + "\n"
	records, err = parseMigrationLog(strings.NewReader(withMarker))
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if len(records) != 1 {
		t.Errorf("Wrong match. expect: 1 record (marker ignored), got: %d", len(records))
	}

	// invalid json line
	if _, errParse := parseMigrationLog(strings.NewReader("{not json}\n")); errParse == nil {
		t.Errorf("Wrong match. expect: error, got: ok")
	}

	// missing customer_id
	if _, errParse := parseMigrationLog(strings.NewReader(`{"old_label":"x"}` + "\n")); errParse == nil {
		t.Errorf("Wrong match. expect: error, got: ok")
	}
}

func Test_domainMigrateRollback(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDBAst := dbhandler.NewMockDBHandler(mc)
	mockDBBin := dbhandler.NewMockDBHandler(mc)
	mockCD := customerdomainhandler.NewMockCustomerDomainHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	ctx := context.Background()

	firstCustomer := uuid.FromStringOrNil("1a1a1a00-7f86-11ee-8f5a-1b2c3d4e5f60")
	secondCustomer := uuid.FromStringOrNil("2b2b2b00-7f86-11ee-8f5a-1b2c3d4e5f60")

	extID := uuid.FromStringOrNil("aaaaaaaa-7f85-11ee-8f5a-1b2c3d4e5f60")
	oldRealmSecond := secondCustomer.String() + ".registrar.voipbin.net"
	newRealmSecond := "qr78.reg.voipbin.net"

	records := []*migrationRecord{
		{
			CustomerID: firstCustomer,
			OldLabel:   firstCustomer.String(),
			NewLabel:   "st90",
			OldRealm:   firstCustomer.String() + ".registrar.voipbin.net",
			NewRealm:   "st90.reg.voipbin.net",
			Extensions: []extensionMigrationRecord{},
		},
		{
			CustomerID: secondCustomer,
			OldLabel:   secondCustomer.String(),
			NewLabel:   "qr78",
			OldRealm:   oldRealmSecond,
			NewRealm:   newRealmSecond,
			Extensions: []extensionMigrationRecord{
				{
					ID:            extID,
					Extension:     "1001",
					OldRealm:      oldRealmSecond,
					NewRealm:      newRealmSecond,
					OldEndpointID: "1001@" + oldRealmSecond,
					NewEndpointID: "1001@" + newRealmSecond,
				},
			},
		},
	}

	// REVERSE order: second customer is rolled back before the first
	migratedExt := testExtension(secondCustomer, "1001", newRealmSecond)

	gomock.InOrder(
		mockCD.EXPECT().UpdateByCustomerID(ctx, secondCustomer, secondCustomer.String(), oldRealmSecond).Return(&customerdomain.CustomerDomain{}, nil),
		mockDBBin.EXPECT().ExtensionGet(ctx, extID).Return(migratedExt, nil),
		mockCD.EXPECT().UpdateByCustomerID(ctx, firstCustomer, firstCustomer.String(), firstCustomer.String()+".registrar.voipbin.net").Return(&customerdomain.CustomerDomain{}, nil),
	)

	// the extension re-provision back to the old realm (same identity-preserving path)
	mockDBAst.EXPECT().AstEndpointDelete(ctx, gomock.Any()).Return(nil).Times(2)
	mockDBAst.EXPECT().AstAuthDelete(ctx, gomock.Any()).Return(nil).Times(2)
	mockDBAst.EXPECT().AstAORDelete(ctx, gomock.Any()).Return(nil).Times(2)
	mockDBAst.EXPECT().AstAORCreate(ctx, gomock.Any()).Return(nil)
	mockDBAst.EXPECT().AstAuthCreate(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, auth *astauth.AstAuth) error {
		if *auth.Realm != oldRealmSecond {
			t.Errorf("Wrong match. expect realm: %s, got: %s", oldRealmSecond, *auth.Realm)
		}
		if *auth.Password != "secret-password" {
			t.Errorf("Wrong match. password changed on rollback: %s", *auth.Password)
		}
		return nil
	})
	mockDBAst.EXPECT().AstEndpointCreate(ctx, gomock.Any()).Return(nil)
	mockDBBin.EXPECT().SIPAuthUpdate(ctx, extID, map[sipauth.Field]any{
		sipauth.FieldRealm: oldRealmSecond,
	}).Return(nil)
	mockDBAst.EXPECT().AstContactDeleteByEndpoint(ctx, "1001@"+newRealmSecond).Return(nil)
	mockDBBin.EXPECT().ExtensionUpdate(ctx, extID, gomock.Any()).Return(nil)
	restored := testExtension(secondCustomer, "1001", oldRealmSecond)
	mockDBBin.EXPECT().ExtensionGet(ctx, extID).Return(restored, nil)
	mockNotify.EXPECT().PublishEvent(ctx, extension.EventTypeExtensionUpdated, restored).Times(1)

	out := &bytes.Buffer{}
	rolledBack, err := domainMigrateRollback(ctx, mockDBAst, mockDBBin, mockCD, mockNotify, true, records, out)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if rolledBack != 2 {
		t.Errorf("Wrong match. expect: 2, got: %d", rolledBack)
	}
}

func Test_domainMigrateRollback_dryRun(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDBAst := dbhandler.NewMockDBHandler(mc)
	mockDBBin := dbhandler.NewMockDBHandler(mc)
	mockCD := customerdomainhandler.NewMockCustomerDomainHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("3c3c3c00-7f86-11ee-8f5a-1b2c3d4e5f60")
	records := []*migrationRecord{
		{
			CustomerID: customerID,
			OldLabel:   customerID.String(),
			NewLabel:   "uv12",
			OldRealm:   customerID.String() + ".registrar.voipbin.net",
			NewRealm:   "uv12.reg.voipbin.net",
		},
	}

	// no expectations: any write fails the test
	out := &bytes.Buffer{}
	rolledBack, err := domainMigrateRollback(ctx, mockDBAst, mockDBBin, mockCD, mockNotify, false, records, out)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if rolledBack != 1 {
		t.Errorf("Wrong match. expect: 1, got: %d", rolledBack)
	}
	if !strings.Contains(out.String(), "[dry-run]") {
		t.Errorf("Wrong match. expect: dry-run output, got: %s", out.String())
	}
}

func Test_domainMigrateRollback_invalidRecord(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDBAst := dbhandler.NewMockDBHandler(mc)
	mockDBBin := dbhandler.NewMockDBHandler(mc)
	mockCD := customerdomainhandler.NewMockCustomerDomainHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	ctx := context.Background()

	records := []*migrationRecord{
		{
			CustomerID: uuid.FromStringOrNil("4d4d4d00-7f86-11ee-8f5a-1b2c3d4e5f60"),
			// missing old_label/old_realm
		},
	}

	out := &bytes.Buffer{}
	if _, err := domainMigrateRollback(ctx, mockDBAst, mockDBBin, mockCD, mockNotify, true, records, out); err == nil {
		t.Errorf("Wrong match. expect: error, got: ok")
	}
}

// ============================================================================
// helpers
// ============================================================================

func Test_extensionLegacyRealm(t *testing.T) {
	customerID := uuid.FromStringOrNil("5e5e5e00-7f86-11ee-8f5a-1b2c3d4e5f60")

	tests := []struct {
		name string

		ext *extension.Extension

		expectRes string
	}{
		{
			name: "from endpoint id",

			ext: &extension.Extension{
				Identity:   commonidentity.Identity{CustomerID: customerID},
				EndpointID: "1001@old.registrar.voipbin.net",
			},

			expectRes: "old.registrar.voipbin.net",
		},
		{
			name: "no endpoint id: empty (stored fields only, no computed fallback)",

			ext: &extension.Extension{
				Identity: commonidentity.Identity{CustomerID: customerID},
			},

			expectRes: "",
		},
		{
			name: "endpoint id without domain part: empty",

			ext: &extension.Extension{
				Identity:   commonidentity.Identity{CustomerID: customerID},
				EndpointID: "1001",
			},

			expectRes: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if res := extensionLegacyRealm(tt.ext); res != tt.expectRes {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectRes, res)
			}
		})
	}
}

func Test_firstDomainLabel(t *testing.T) {
	tests := []struct {
		name string

		realm string

		expectRes string
	}{
		{"uuid realm", "5e5e5e00-7f86-11ee-8f5a-1b2c3d4e5f60.registrar.voipbin.net", "5e5e5e00-7f86-11ee-8f5a-1b2c3d4e5f60"},
		{"short realm", "ab12.reg.voipbin.net", "ab12"},
		{"no dot", "ab12", "ab12"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if res := firstDomainLabel(tt.realm); res != tt.expectRes {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectRes, res)
			}
		})
	}
}

// ============================================================================
// log-before-execute (interrupted/resumed run rollback coverage)
// ============================================================================

func testExtensionWithID(id uuid.UUID, customerID uuid.UUID, ext string, realm string) *extension.Extension {
	e := testExtension(customerID, ext, realm)
	e.ID = id
	return e
}

// Test_migrateCustomerDomain_logIsWrittenBeforeExecution proves the
// flush-before-execute contract: when a later extension of the same customer
// fails mid-run, the log ALREADY holds the full record covering every pending
// extension (including the ones that were never touched).
func Test_migrateCustomerDomain_logIsWrittenBeforeExecution(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDBAst := dbhandler.NewMockDBHandler(mc)
	mockDBBin := dbhandler.NewMockDBHandler(mc)
	mockCD := customerdomainhandler.NewMockCustomerDomainHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("6f6f6f00-7f86-11ee-8f5a-1b2c3d4e5f60")
	oldRealm := customerID.String() + ".registrar.voipbin.net"
	newRealm := "wx34.reg.voipbin.net"

	backfilledRow := &customerdomain.CustomerDomain{
		CustomerID:  customerID,
		DomainLabel: customerID.String(),
		Realm:       oldRealm,
	}
	migratedRow := &customerdomain.CustomerDomain{
		CustomerID:  customerID,
		DomainLabel: "wx34",
		Realm:       newRealm,
	}

	ext1 := testExtensionWithID(uuid.FromStringOrNil("e1e1e1e1-7f86-11ee-8f5a-1b2c3d4e5f60"), customerID, "1001", oldRealm)
	ext2 := testExtensionWithID(uuid.FromStringOrNil("e2e2e2e2-7f86-11ee-8f5a-1b2c3d4e5f60"), customerID, "1002", oldRealm)

	mockDBBin.EXPECT().CustomerDomainGet(ctx, customerID).Return(backfilledRow, nil)
	mockDBBin.EXPECT().ExtensionList(ctx, uint64(migrationExtensionListLimit), "", gomock.Any()).Return([]*extension.Extension{ext1, ext2}, nil)
	mockCD.EXPECT().MigrateToShortDomain(ctx, customerID).Return(migratedRow, nil)

	// the FIRST extension fails immediately: nothing of ext2 is ever touched
	mockDBAst.EXPECT().AstEndpointDelete(ctx, ext1.EndpointID).Return(fmt.Errorf("db down"))

	out := &bytes.Buffer{}
	logBuf := &bytes.Buffer{}
	_, _, err := migrateCustomerDomain(ctx, mockDBAst, mockDBBin, mockCD, mockNotify, customerID, true, logBuf, out)
	if err == nil {
		t.Fatalf("Wrong match. expect: error, got: ok")
	}

	// the log line was written BEFORE the failing mutation and covers BOTH extensions
	records, errParse := parseMigrationLog(bytes.NewReader(logBuf.Bytes()))
	if errParse != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", errParse)
	}
	if len(records) != 1 {
		t.Fatalf("Wrong match. expect: 1 pre-logged record, got: %d", len(records))
	}
	record := records[0]
	if record.CustomerID != customerID || record.OldRealm != oldRealm || record.NewRealm != newRealm {
		t.Errorf("Wrong match. record: %+v", record)
	}
	if len(record.Extensions) != 2 {
		t.Fatalf("Wrong match. expect: 2 extensions in the pre-logged record, got: %d", len(record.Extensions))
	}
	for i, expectID := range []uuid.UUID{ext1.ID, ext2.ID} {
		if record.Extensions[i].ID != expectID || record.Extensions[i].OldRealm != oldRealm || record.Extensions[i].NewRealm != newRealm {
			t.Errorf("Wrong match. extension record %d: %+v", i, record.Extensions[i])
		}
	}

	// no completion marker: the customer never completed
	if strings.Contains(logBuf.String(), migrationStatusCompleted) {
		t.Errorf("Wrong match. expect: no completion marker, got: %s", logBuf.String())
	}
}

// Test_domainMigrateRollback_preLoggedNeverMigratedExtension proves that
// replaying a pre-logged record whose extension was never actually migrated
// (crash before its re-provision) is a clean no-op skip.
func Test_domainMigrateRollback_preLoggedNeverMigratedExtension(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDBAst := dbhandler.NewMockDBHandler(mc)
	mockDBBin := dbhandler.NewMockDBHandler(mc)
	mockCD := customerdomainhandler.NewMockCustomerDomainHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("7a7a7a00-7f86-11ee-8f5a-1b2c3d4e5f60")
	oldRealm := customerID.String() + ".registrar.voipbin.net"
	newRealm := "yz56.reg.voipbin.net"

	extID := uuid.FromStringOrNil("e3e3e3e3-7f86-11ee-8f5a-1b2c3d4e5f60")
	records := []*migrationRecord{
		{
			CustomerID: customerID,
			OldLabel:   customerID.String(),
			NewLabel:   "yz56",
			OldRealm:   oldRealm,
			NewRealm:   newRealm,
			Extensions: []extensionMigrationRecord{
				{
					ID:            extID,
					Extension:     "1001",
					OldRealm:      oldRealm,
					NewRealm:      newRealm,
					OldEndpointID: "1001@" + oldRealm,
					NewEndpointID: "1001@" + newRealm,
				},
			},
		},
	}

	// the extension is STILL on the old realm (never migrated before the crash)
	neverMigrated := testExtensionWithID(extID, customerID, "1001", oldRealm)

	mockCD.EXPECT().UpdateByCustomerID(ctx, customerID, customerID.String(), oldRealm).Return(&customerdomain.CustomerDomain{}, nil)
	mockDBBin.EXPECT().ExtensionGet(ctx, extID).Return(neverMigrated, nil)
	// NO ast/sipauth/update/notify expectations: any re-provision call fails the test

	out := &bytes.Buffer{}
	rolledBack, err := domainMigrateRollback(ctx, mockDBAst, mockDBBin, mockCD, mockNotify, true, records, out)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok (clean skip), got: %v", err)
	}
	if rolledBack != 1 {
		t.Errorf("Wrong match. expect: 1, got: %d", rolledBack)
	}
}

// Test_domainMigrate_interruptedThenResumed_rollbackRestoresAll is the exact
// review scenario: run 1 pre-logs a record with 5 extensions and dies after 3;
// the resume run pre-logs the remaining 2; a rollback over the ACCUMULATED log
// restores all 5 extensions and the customer domain row.
func Test_domainMigrate_interruptedThenResumed_rollbackRestoresAll(t *testing.T) {
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("8b8b8b00-7f86-11ee-8f5a-1b2c3d4e5f60")
	oldRealm := customerID.String() + ".registrar.voipbin.net"
	newRealm := "qq77.reg.voipbin.net"

	backfilledRow := &customerdomain.CustomerDomain{
		CustomerID:  customerID,
		DomainLabel: customerID.String(),
		Realm:       oldRealm,
	}
	shortRow := &customerdomain.CustomerDomain{
		CustomerID:  customerID,
		DomainLabel: "qq77",
		Realm:       newRealm,
	}

	extIDs := []uuid.UUID{
		uuid.FromStringOrNil("f1f1f1f1-7f87-11ee-8f5a-1b2c3d4e5f60"),
		uuid.FromStringOrNil("f2f2f2f2-7f87-11ee-8f5a-1b2c3d4e5f60"),
		uuid.FromStringOrNil("f3f3f3f3-7f87-11ee-8f5a-1b2c3d4e5f60"),
		uuid.FromStringOrNil("f4f4f4f4-7f87-11ee-8f5a-1b2c3d4e5f60"),
		uuid.FromStringOrNil("f5f5f5f5-7f87-11ee-8f5a-1b2c3d4e5f60"),
	}
	extNumbers := []string{"1001", "1002", "1003", "1004", "1005"}

	logBuf := &bytes.Buffer{}

	// ---- RUN 1: all 5 pending; dies when starting extension 4 ----
	{
		mc := gomock.NewController(t)

		mockDBAst := dbhandler.NewMockDBHandler(mc)
		mockDBBin := dbhandler.NewMockDBHandler(mc)
		mockCD := customerdomainhandler.NewMockCustomerDomainHandler(mc)
		mockNotify := notifyhandler.NewMockNotifyHandler(mc)

		exts := make([]*extension.Extension, 5)
		for i := range extIDs {
			exts[i] = testExtensionWithID(extIDs[i], customerID, extNumbers[i], oldRealm)
		}

		mockDBBin.EXPECT().CustomerDomainGet(ctx, customerID).Return(backfilledRow, nil)
		mockDBBin.EXPECT().ExtensionList(ctx, uint64(migrationExtensionListLimit), "", gomock.Any()).Return(exts, nil)
		mockCD.EXPECT().MigrateToShortDomain(ctx, customerID).Return(shortRow, nil)

		// extensions 1-3 re-provision fully
		for i := 0; i < 3; i++ {
			e := exts[i]
			newEndpoint := extNumbers[i] + "@" + newRealm
			mockDBAst.EXPECT().AstEndpointDelete(ctx, e.EndpointID).Return(nil)
			mockDBAst.EXPECT().AstEndpointDelete(ctx, newEndpoint).Return(nil)
			mockDBAst.EXPECT().AstAuthDelete(ctx, e.AuthID).Return(nil)
			mockDBAst.EXPECT().AstAuthDelete(ctx, newEndpoint).Return(nil)
			mockDBAst.EXPECT().AstAORDelete(ctx, e.AORID).Return(nil)
			mockDBAst.EXPECT().AstAORDelete(ctx, newEndpoint).Return(nil)
			mockDBAst.EXPECT().AstAORCreate(ctx, gomock.Any()).Return(nil)
			mockDBAst.EXPECT().AstAuthCreate(ctx, gomock.Any()).Return(nil)
			mockDBAst.EXPECT().AstEndpointCreate(ctx, gomock.Any()).Return(nil)
			mockDBBin.EXPECT().SIPAuthUpdate(ctx, e.ID, gomock.Any()).Return(nil)
			mockDBAst.EXPECT().AstContactDeleteByEndpoint(ctx, e.EndpointID).Return(nil)
			mockDBBin.EXPECT().ExtensionUpdate(ctx, e.ID, gomock.Any()).Return(nil)
			migrated := testExtensionWithID(e.ID, customerID, extNumbers[i], newRealm)
			mockDBBin.EXPECT().ExtensionGet(ctx, e.ID).Return(migrated, nil)
			mockNotify.EXPECT().PublishEvent(ctx, extension.EventTypeExtensionUpdated, migrated)
		}

		// extension 4: the process "dies" (first mutation errors)
		mockDBAst.EXPECT().AstEndpointDelete(ctx, exts[3].EndpointID).Return(fmt.Errorf("process killed"))

		out := &bytes.Buffer{}
		_, _, err := domainMigrate(ctx, mockDBAst, mockDBBin, mockCD, mockNotify, true, customerID, logBuf, out)
		if err == nil {
			t.Fatalf("Wrong match. expect: error (interrupted run), got: ok")
		}
		mc.Finish()
	}

	// run 1 pre-logged one record covering ALL 5 extensions
	run1Records, err := parseMigrationLog(bytes.NewReader(logBuf.Bytes()))
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if len(run1Records) != 1 || len(run1Records[0].Extensions) != 5 {
		t.Fatalf("Wrong match. expect: 1 record with 5 extensions, got: %+v", run1Records)
	}

	// ---- RUN 2 (resume): 1-3 already migrated, 4-5 still pending ----
	{
		mc := gomock.NewController(t)

		mockDBAst := dbhandler.NewMockDBHandler(mc)
		mockDBBin := dbhandler.NewMockDBHandler(mc)
		mockCD := customerdomainhandler.NewMockCustomerDomainHandler(mc)
		mockNotify := notifyhandler.NewMockNotifyHandler(mc)

		exts := make([]*extension.Extension, 5)
		for i := range extIDs {
			realm := newRealm
			if i >= 3 {
				realm = oldRealm // 4-5 still on the old realm
			}
			exts[i] = testExtensionWithID(extIDs[i], customerID, extNumbers[i], realm)
		}

		mockDBBin.EXPECT().CustomerDomainGet(ctx, customerID).Return(shortRow, nil)
		mockDBBin.EXPECT().ExtensionList(ctx, uint64(migrationExtensionListLimit), "", gomock.Any()).Return(exts, nil)
		mockCD.EXPECT().MigrateToShortDomain(ctx, customerID).Return(shortRow, nil)

		for i := 3; i < 5; i++ {
			e := exts[i]
			newEndpoint := extNumbers[i] + "@" + newRealm
			mockDBAst.EXPECT().AstEndpointDelete(ctx, e.EndpointID).Return(nil)
			mockDBAst.EXPECT().AstEndpointDelete(ctx, newEndpoint).Return(nil)
			mockDBAst.EXPECT().AstAuthDelete(ctx, e.AuthID).Return(nil)
			mockDBAst.EXPECT().AstAuthDelete(ctx, newEndpoint).Return(nil)
			mockDBAst.EXPECT().AstAORDelete(ctx, e.AORID).Return(nil)
			mockDBAst.EXPECT().AstAORDelete(ctx, newEndpoint).Return(nil)
			mockDBAst.EXPECT().AstAORCreate(ctx, gomock.Any()).Return(nil)
			mockDBAst.EXPECT().AstAuthCreate(ctx, gomock.Any()).Return(nil)
			mockDBAst.EXPECT().AstEndpointCreate(ctx, gomock.Any()).Return(nil)
			mockDBBin.EXPECT().SIPAuthUpdate(ctx, e.ID, gomock.Any()).Return(nil)
			mockDBAst.EXPECT().AstContactDeleteByEndpoint(ctx, e.EndpointID).Return(nil)
			mockDBBin.EXPECT().ExtensionUpdate(ctx, e.ID, gomock.Any()).Return(nil)
			migrated := testExtensionWithID(e.ID, customerID, extNumbers[i], newRealm)
			mockDBBin.EXPECT().ExtensionGet(ctx, e.ID).Return(migrated, nil)
			mockNotify.EXPECT().PublishEvent(ctx, extension.EventTypeExtensionUpdated, migrated)
		}

		out := &bytes.Buffer{}
		migrated, skipped, errMigrate := domainMigrate(ctx, mockDBAst, mockDBBin, mockCD, mockNotify, true, customerID, logBuf, out)
		if errMigrate != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", errMigrate)
		}
		if migrated != 1 || skipped != 0 {
			t.Errorf("Wrong match. expect: migrated=1 skipped=0, got: migrated=%d skipped=%d", migrated, skipped)
		}
		mc.Finish()
	}

	// accumulated log: record(5 exts) + record(2 exts); completion marker ignored
	records, err := parseMigrationLog(bytes.NewReader(logBuf.Bytes()))
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if len(records) != 2 || len(records[0].Extensions) != 5 || len(records[1].Extensions) != 2 {
		t.Fatalf("Wrong match. expect: records with 5 and 2 extensions, got: %+v", records)
	}
	// the resume record's old realms are the TRUE old realms of the pending extensions
	for _, e := range records[1].Extensions {
		if e.OldRealm != oldRealm {
			t.Errorf("Wrong match. expect: %s, got: %s", oldRealm, e.OldRealm)
		}
	}

	// ---- ROLLBACK over the accumulated log: ALL 5 must come back ----
	{
		mc := gomock.NewController(t)

		mockDBAst := dbhandler.NewMockDBHandler(mc)
		mockDBBin := dbhandler.NewMockDBHandler(mc)
		mockCD := customerdomainhandler.NewMockCustomerDomainHandler(mc)
		mockNotify := notifyhandler.NewMockNotifyHandler(mc)

		// live state: every extension is on the new realm; ExtensionGet serves
		// from state, ExtensionUpdate(realm rollback) mutates it
		state := map[uuid.UUID]string{}
		for _, id := range extIDs {
			state[id] = newRealm
		}
		extNumberByID := map[uuid.UUID]string{}
		for i, id := range extIDs {
			extNumberByID[id] = extNumbers[i]
		}

		mockDBBin.EXPECT().ExtensionGet(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, id uuid.UUID) (*extension.Extension, error) {
			return testExtensionWithID(id, customerID, extNumberByID[id], state[id]), nil
		}).AnyTimes()
		mockDBBin.EXPECT().ExtensionUpdate(ctx, gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, id uuid.UUID, fields map[extension.Field]any) error {
			state[id] = fields[extension.FieldRealm].(string)
			return nil
		}).Times(5) // exactly the 5 extensions, once each
		mockDBAst.EXPECT().AstEndpointDelete(ctx, gomock.Any()).Return(nil).AnyTimes()
		mockDBAst.EXPECT().AstAuthDelete(ctx, gomock.Any()).Return(nil).AnyTimes()
		mockDBAst.EXPECT().AstAORDelete(ctx, gomock.Any()).Return(nil).AnyTimes()
		mockDBAst.EXPECT().AstAORCreate(ctx, gomock.Any()).Return(nil).AnyTimes()
		mockDBAst.EXPECT().AstAuthCreate(ctx, gomock.Any()).Return(nil).AnyTimes()
		mockDBAst.EXPECT().AstEndpointCreate(ctx, gomock.Any()).Return(nil).AnyTimes()
		mockDBAst.EXPECT().AstContactDeleteByEndpoint(ctx, gomock.Any()).Return(nil).AnyTimes()
		mockDBBin.EXPECT().SIPAuthUpdate(ctx, gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		// exactly one extension_updated per restored extension
		mockNotify.EXPECT().PublishEvent(ctx, extension.EventTypeExtensionUpdated, gomock.Any()).Times(5)
		// the domain row is restored once per record (idempotent replay)
		mockCD.EXPECT().UpdateByCustomerID(ctx, customerID, customerID.String(), oldRealm).Return(backfilledRow, nil).Times(2)

		out := &bytes.Buffer{}
		rolledBack, errRollback := domainMigrateRollback(ctx, mockDBAst, mockDBBin, mockCD, mockNotify, true, records, out)
		if errRollback != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", errRollback)
		}
		if rolledBack != 2 {
			t.Errorf("Wrong match. expect: 2 records replayed, got: %d", rolledBack)
		}

		// EVERY extension is back on the old realm — nothing stranded
		for id, realm := range state {
			if realm != oldRealm {
				t.Errorf("Wrong match. extension %s stranded on realm %s", id, realm)
			}
		}
		mc.Finish()
	}
}
