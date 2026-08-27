// Golden routing-key table of the global topic exchange `bin-manager.event` (VOIP-1404/1405).
//
// It covers EVERY event type billing-manager publishes today, across both live resource namespaces
// (billing / account), and asserts the exact key that notifyhandler generates for the real event
// data type of each publish site. The primary defect class it guards against is "the right key
// shape carrying the wrong id space": an event published under an id that is not the resource's
// subscription address produces well-formed keys that no instance binding ever matches, and no
// runtime metric can detect it. Design doc §2.4 / §4.
//
// billing-manager is a default-fallback service: `*billing.Billing` and `*account.Account` are
// both addressed by their OWN id (design §2.4 records this as a deliberate decision for
// billing.Billing -- address consistency with `billing_updated`, and the id is obtainable from the
// create response), so NO type in this service carries an eventtopic.SubscriptionIdentifier
// override. That absence is asserted explicitly below.
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
	"encoding/json"
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
// (1404 design §4.2 / §5.2): the opt-in interface first, then -- ONLY when no override exists --
// the top-level "id" of the marshaled payload. Keeping it here rather than reaching into
// notifyhandler internals is deliberate -- the golden table must fail when a model STARTS or STOPS
// implementing the interface, which is exactly what this two-step reproduction detects.
//
// The early return below is the load-bearing half: an override that EXISTS is authoritative even
// when it yields "" or uuid.Nil, so the JSON fallback must never run behind it. This matches
// notifyhandler.resolveSubscriptionOverride's hasOverride semantics exactly; if the two ever
// diverge, this table stops reproducing what the publish path actually generates.
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
		// billing resource -- own id is the address, resolved by the default JSON fallback.
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
// resolves to its OWN id, taken from the marshaled payload's top-level `id`. A payload that loses
// that field (or gains an override returning something else) collapses to the `-` placeholder or a
// foreign address, and this assertion is what catches it.
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

// TestBillingAndAccountUseDefaultSubscriptionID pins the deliberate absence of an override on both
// published types: their own ids ARE the addresses, so implementing the interface would be
// redundant and the default JSON `id` extraction must keep covering them. billing-manager has no
// override type at all (design §2.4).
func TestBillingAndAccountUseDefaultSubscriptionID(t *testing.T) {
	tests := []struct {
		name string
		data any
	}{
		{"Billing", &billing.Billing{Identity: commonidentity.Identity{ID: billingID}}},
		{"Account", &account.Account{Identity: commonidentity.Identity{ID: accountID}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, ok := tt.data.(eventtopic.SubscriptionIdentifier); ok {
				t.Errorf("%s must not implement SubscriptionIdentifier. its own id is the subscription address.", tt.name)
			}
		})
	}
}
