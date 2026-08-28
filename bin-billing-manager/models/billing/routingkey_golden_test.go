// Golden routing-key table of the global topic exchange `bin-manager.event` (VOIP-1404/1405).
//
// It covers EVERY event type billing-manager publishes today, across both live resource namespaces
// (billing / account), and asserts the exact key that notifyhandler generates for the real event
// data type of each publish site. The primary defect class it guards against is "the right key
// shape carrying the wrong id space": an event published under an id that is not the resource's
// subscription address produces well-formed keys that no instance binding ever matches, and no
// runtime metric can detect it. Design doc §2.4 / §4.
//
// billing-manager is a standard own-id service: `*billing.Billing` and `*account.Account` are
// both addressed by their OWN id (design §2.4 records this as a deliberate decision for
// billing.Billing -- address consistency with `billing_updated`, and the id is obtainable from the
// create response). Since VOIP-1419 every published type states that address through the
// mandatory eventtopic.SubscriptionIdentifier contract -- for both types here the own-id
// default is the `EventSubscriptionID()` promoted from the embedded commonidentity.Identity;
// the compile-time assertions live in each type's sibling test
// file (billing_test.go / account_test.go).
//
// The file lives in models/billing because billing is the service's designated PRIMARY model
// package; it is an external test package so it can import the sibling model packages without any
// import-cycle risk.
//
// MAINTENANCE: this table pins CURRENT behavior. Two constants are DEAD -- `billing_deleted`
// (models/billing/event.go) and `account_deleted` (models/account/event.go) have no publish site
// anywhere in this service -- and are deliberately excluded per design §4's dead-constant list.
// CAUTION: the dead `account_deleted` is billing-manager's ONLY; the identically named constants in
// conversation-manager and storage-manager are LIVE and must not be excluded there. If a delete
// publish path is ever added here, add its rows in the same change.
package billing_test

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-billing-manager/models/account"
	"monorepo/bin-billing-manager/models/billing"
	"monorepo/bin-common-handler/models/eventtopic"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
)

// billingID / accountID are the subscription addresses of the two independent resource streams
// this service publishes. Unlike a parent/child service, they deliberately do NOT converge: a
// billing entry is addressed by its own id, an account by its own id (design §2.4).
var (
	billingID = uuid.FromStringOrNil("6b28d4f1-0000-4000-8000-000000000001")
	accountID = uuid.FromStringOrNil("6b28d4f1-0000-4000-8000-000000000002")
)

// resolveSubscriptionID mirrors the resolution notifyhandler performs on the publish path
// (VOIP-1419): every published type satisfies the mandatory EventSubscriptionID contract --
// compiler-enforced at the publish call site; here via the method promoted from the embedded
// commonidentity.Identity -- and its return value IS the subscription address.
// There is no fallback: data that does not implement the interface (unrepresentable on the
// production path, but expressible through this `any`-typed helper) and a typed-nil pointer both
// resolve to "", which the routing key degrades to the `-` placeholder.
//
// The typed-nil guard is load-bearing: a nil pointer whose type implements the interface still
// SATISFIES the assertion, and every real implementation dereferences its receiver -- calling the
// method would panic. Production resolves such a payload to the placeholder, so this helper
// returns "" for it too.
func resolveSubscriptionID(t *testing.T, data any) string {
	t.Helper()

	identifier, ok := data.(eventtopic.SubscriptionIdentifier)
	if !ok {
		return ""
	}
	if v := reflect.ValueOf(data); v.Kind() == reflect.Ptr && v.IsNil() {
		return ""
	}

	return identifier.EventSubscriptionID()
}

func TestGoldenRoutingKeys(t *testing.T) {
	publisher := string(commonoutline.ServiceNameBillingManager)

	billingData := &billing.Billing{
		Identity: commonidentity.Identity{
			ID:         billingID,
			CustomerID: uuid.Must(uuid.NewV4()),
		},
		AccountID:       accountID,
		TransactionType: billing.TransactionTypeUsage,
	}

	accountData := &account.Account{
		Identity: commonidentity.Identity{
			ID:         accountID,
			CustomerID: uuid.Must(uuid.NewV4()),
		},
		Status: account.StatusActive,
	}

	tests := []struct {
		name      string
		eventType string
		data      any
		expect    string
	}{
		// billing resource -- own id is the address, returned through the EventSubscriptionID
		// promoted from the embedded commonidentity.Identity (VOIP-1419).
		// NOTE the deliberate decision recorded in design §2.4: the account-id is NOT the address
		// here, even though a billing entry is a child of an account.
		{
			"billing_created",
			billing.EventTypeBillingCreated,
			billingData,
			"billing-manager.billing.6b28d4f1-0000-4000-8000-000000000001.created",
		},
		{
			"billing_updated",
			billing.EventTypeBillingUpdated,
			billingData,
			"billing-manager.billing.6b28d4f1-0000-4000-8000-000000000001.updated",
		},

		// account resource -- own id is the address.
		{
			"account_created",
			account.EventTypeAccountCreated,
			accountData,
			"billing-manager.account.6b28d4f1-0000-4000-8000-000000000002.created",
		},
		{
			"account_updated",
			account.EventTypeAccountUpdated,
			accountData,
			"billing-manager.account.6b28d4f1-0000-4000-8000-000000000002.updated",
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

// TestGoldenRoutingKeysUseOwnID pins the property the table exists to protect: each resource
// resolves to its OWN id, returned through the promoted Identity default. A method that
// starts returning "" (or a foreign id) collapses the address to the `-` placeholder or a foreign
// address, and this assertion is what catches it.
func TestGoldenRoutingKeysUseOwnID(t *testing.T) {
	tests := []struct {
		name   string
		data   any
		expect string
	}{
		{
			"billing",
			&billing.Billing{Identity: commonidentity.Identity{ID: billingID}, AccountID: accountID},
			billingID.String(),
		},
		{
			"account",
			&account.Account{Identity: commonidentity.Identity{ID: accountID}},
			accountID.String(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := resolveSubscriptionID(t, tt.data)
			if res != tt.expect {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expect, res)
			}

			if eventtopic.IsPlaceholderSubscriptionID(res) {
				t.Errorf("%s must never resolve to the placeholder address. got: %s", tt.name, res)
			}
		})
	}
}
