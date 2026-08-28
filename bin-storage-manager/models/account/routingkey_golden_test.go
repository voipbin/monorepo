// Golden routing-key table of the global topic exchange `bin-manager.event` (VOIP-1404/1405).
//
// It covers EVERY event type storage-manager publishes today, across both resource namespaces
// (account / file), and asserts the exact key that notifyhandler generates for the real event
// data type of each publish site. Both published types address their own stream by their own id
// -- account.Account through its explicit EventSubscriptionID method, file.File through the
// default promoted from the embedded commonidentity.Identity (VOIP-1419: the contract is
// mandatory; an empty return degrades to the `-` placeholder). The table exists to keep those
// addresses
// pinned -- a change to either resolution would move the address without any runtime metric noticing.
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
	"reflect"
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
// (VOIP-1419): the event data's EventSubscriptionID method (explicit on account.Account,
// promoted from the embedded commonidentity.Identity on file.File) is the whole mechanism --
// the contract is mandatory, and an empty result degrades to the `-` placeholder. The `data any`
// parameter is kept so the table can also feed values that do NOT implement the interface; those
// resolve to "" (→ placeholder), same as production. Keeping the helper here rather than reaching
// into notifyhandler internals is deliberate -- the golden table must fail when a model's method
// changes what it returns, which is exactly what this reproduction detects.
func resolveSubscriptionID(t *testing.T, data any) string {
	t.Helper()

	identifier, ok := data.(eventtopic.SubscriptionIdentifier)
	if !ok {
		return ""
	}

	// typed-nil guard, mirroring notifyhandler: a nil pointer whose type implements the interface
	// still SATISFIES the assertion, and every real implementation dereferences its receiver --
	// calling the method would panic. Production resolves such a payload to the `-` placeholder.
	if v := reflect.ValueOf(data); v.Kind() == reflect.Ptr && v.IsNil() {
		return ""
	}

	return identifier.EventSubscriptionID()
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
