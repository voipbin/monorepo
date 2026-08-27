// Golden routing-key table of the global topic exchange `bin-manager.event` (VOIP-1404/1405).
//
// It covers EVERY event type storage-manager publishes today, across both resource namespaces
// (account / file), and asserts the exact key that notifyhandler generates for the real event
// data type of each publish site. storage-manager carries NO subscription-id override: both
// published types address their own stream by their own id, so the default JSON top-level `id`
// fallback is the whole resolution. The table exists to keep it that way -- an override silently
// added to either type would move the address without any runtime metric noticing.
//
// The file lives in models/account because the table spans both model packages of the service and
// account is the designated PRIMARY package for storage-manager (1405 plan §Phase 1 anchoring
// list); it is an external test package so it can import the sibling model package without any
// import-cycle risk.
//
// MAINTENANCE: the table pins CURRENT behavior.
//   - `file_updated` (models/file.EventTypeFileUpdated) is a DEAD constant -- no publish site
//     references it (design §4 dead-constant list). It is deliberately absent; if a publish site
//     for it ever appears, add its row here in the same change.
//   - `Account_created` / `Account_updated` / `Account_deleted` are spelled with a CAPITAL A in
//     the source constants, unlike every other event type in the monorepo. The routing key is
//     lowercased by eventtopic.RoutingKey, so the wire keys read `storage-manager.account.<id>.
//     <action>`. Both halves are pinned below (the raw constant AND the normalized key) so a
//     "cleanup" rename of the constants shows up here as a routing-key change, which is what it
//     would actually be for every subscriber.
package account_test

import (
	"encoding/json"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/eventtopic"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-storage-manager/models/account"
	"monorepo/bin-storage-manager/models/file"
)

var (
	// accountID is the subscription address of every account event.
	accountID = uuid.FromStringOrNil("a51c0a9e-0000-4000-8000-000000000001")

	// fileID is the subscription address of every file event. A file is an independent resource
	// addressed by its own id -- it does NOT collapse onto the account axis (design §2.4).
	fileID = uuid.FromStringOrNil("a51c0a9e-0000-4000-8000-000000000002")
)

// resolveSubscriptionID mirrors the resolution notifyhandler performs on the publish path
// (1404 design §4.2 / §5.2): the opt-in interface first, then -- ONLY when no override exists --
// the top-level "id" of the marshaled payload. Keeping it here rather than reaching into
// notifyhandler internals is deliberate -- the golden table must fail when a model STARTS or
// STOPS implementing the interface, which is exactly what this two-step reproduction detects.
//
// The early return below is the load-bearing half: an override that EXISTS is authoritative even
// when it yields "" or uuid.Nil, so the JSON fallback must never run behind it. This matches
// notifyhandler.resolveSubscriptionOverride's hasOverride semantics exactly.
func resolveSubscriptionID(t *testing.T, data any) string {
	t.Helper()

	if identifier, ok := data.(eventtopic.SubscriptionIdentifier); ok {
		return identifier.EventSubscriptionID()
	}

	m, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("Could not marshal the event data. err: %v", err)
	}

	d := struct {
		ID string `json:"id"`
	}{}
	if errUnmarshal := json.Unmarshal(m, &d); errUnmarshal != nil {
		return ""
	}

	return d.ID
}

func TestGoldenRoutingKeys(t *testing.T) {
	publisher := string(commonoutline.ServiceNameStorageManager)

	accountData := &account.Account{
		ID:         accountID,
		CustomerID: uuid.Must(uuid.NewV4()),
	}

	fileData := &file.File{
		Identity: commonidentity.Identity{
			ID:         fileID,
			CustomerID: uuid.Must(uuid.NewV4()),
		},
		AccountID: accountID,
		Name:      "record.wav",
	}

	tests := []struct {
		name      string
		eventType string
		data      any
		expect    string
	}{
		// account resource -- the raw constants are CAPITALIZED (`Account_created`, ...);
		// eventtopic.RoutingKey lowercases before splitting, so the resource segment normalizes
		// to `account`. Publish sites: pkg/accounthandler/account.go:52 (created), :162 and :188
		// (updated, from IncreaseFileInfo / DecreaseFileInfo), :136 (deleted).
		{
			"Account_created (uppercase constant, normalized key)",
			account.EventTypeAccountCreated,
			accountData,
			"storage-manager.account.a51c0a9e-0000-4000-8000-000000000001.created",
		},
		{
			"Account_updated (uppercase constant, normalized key)",
			account.EventTypeAccountUpdated,
			accountData,
			"storage-manager.account.a51c0a9e-0000-4000-8000-000000000001.updated",
		},
		{
			"Account_deleted (uppercase constant, normalized key)",
			account.EventTypeAccountDeleted,
			accountData,
			"storage-manager.account.a51c0a9e-0000-4000-8000-000000000001.deleted",
		},

		// file resource -- lowercase constants, own id is the address.
		// Publish sites: pkg/filehandler/file.go:140 (created), :219 (deleted).
		{
			"file_created",
			file.EventTypeFileCreated,
			fileData,
			"storage-manager.file.a51c0a9e-0000-4000-8000-000000000002.created",
		},
		{
			"file_deleted",
			file.EventTypeFileDeleted,
			fileData,
			"storage-manager.file.a51c0a9e-0000-4000-8000-000000000002.deleted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subscriptionID := resolveSubscriptionID(t, tt.data)

			res := eventtopic.RoutingKey(publisher, tt.eventType, subscriptionID)
			if res != tt.expect {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expect, res)
			}
		})
	}
}

// TestGoldenRoutingKeysAccountEventTypeLiterals pins the RAW event-type constants of the account
// namespace next to the normalized keys asserted above. storage-manager is the only service whose
// event types start with an uppercase letter, and the normalization that hides it lives in
// eventtopic.RoutingKey, not here -- so if someone "fixes" the capitalization, this test is what
// says out loud that the published payload's `type` field changed for every fanout subscriber,
// even though the topic routing key would not move.
func TestGoldenRoutingKeysAccountEventTypeLiterals(t *testing.T) {
	tests := []struct {
		name   string
		got    string
		expect string
	}{
		{"created", account.EventTypeAccountCreated, "Account_created"},
		{"updated", account.EventTypeAccountUpdated, "Account_updated"},
		{"deleted", account.EventTypeAccountDeleted, "Account_deleted"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expect {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expect, tt.got)
			}
		})
	}
}

// TestStorageTypesUseDefaultSubscriptionID pins the deliberate ABSENCE of a subscription-id
// override on both published types (design §2.4: storage account/file address by their own id).
// Adding an override to either would silently relocate every subscriber's binding address, and no
// runtime metric would flag it -- the keys would stay well-formed.
func TestStorageTypesUseDefaultSubscriptionID(t *testing.T) {
	tests := []struct {
		name string
		data any
	}{
		{"account", &account.Account{ID: accountID}},
		{"file", &file.File{Identity: commonidentity.Identity{ID: fileID}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, ok := tt.data.(eventtopic.SubscriptionIdentifier); ok {
				t.Errorf("%s must not implement SubscriptionIdentifier. its own id is the subscription address.", tt.name)
			}
		})
	}
}
