// Golden routing-key table of the global topic exchange `bin-manager.event` (VOIP-1404/1405).
//
// It covers EVERY event type customer-manager publishes today, across both live resource
// namespaces (customer / accesskey), and asserts the exact key that notifyhandler generates for the
// real event data type of each publish site. The primary defect class it guards against is "the
// right key shape carrying the wrong id space": an event published under an id that is not the
// resource's subscription address produces well-formed keys that no instance binding ever matches,
// and no runtime metric can detect it. Design doc §2.4 / §4.
//
// Since VOIP-1419 every published type carries an explicit `EventSubscriptionID()` method --
// mandatory, compiler-enforced; there is no JSON fallback anymore, and an empty return is the
// only degrade path (the `-` placeholder). `*customer.Customer` and `*accesskey.Accesskey` are
// both addressed by their OWN id.
//
// THE TRAP THIS FILE EXISTS FOR: `customer.CustomerCreatedEvent` ANONYMOUSLY EMBEDS `*Customer`,
// so `*Customer`'s method is promoted to the wrapper -- but a promoted call on a nil embed
// panics (the receiver chain is computed before any promoted access). The wrapper therefore
// carries its OWN explicit shadowing method with a nil guard: non-nil embed returns the
// embedded Customer's id (same key space as every other customer event), nil embed returns ""
// explicitly. Both branches are pinned below and must never be broken.
//
// The nil-embed row is the second pinned case: the wrapper's nil guard returns "" so the
// address collapses to the `-` placeholder. That is the correct outcome for a payload carrying
// no identity, and the row exists so the behavior is a pinned fact rather than an assumption.
//
// The file lives in models/customer because customer is the service's designated PRIMARY model
// package and the resource the wrapper addresses; it is an external test package so it can import
// the sibling model packages without any import-cycle risk.
//
// MAINTENANCE: this table pins CURRENT behavior. Every constant in models/customer/event.go and
// models/accesskey/event.go is LIVE -- there is no dead constant to exclude here. When a publish
// path is added or removed, update this table in the same change.
package customer_test

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/eventtopic"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-customer-manager/models/accesskey"
	"monorepo/bin-customer-manager/models/customer"
)

// customerID / accesskeyID are the subscription addresses of the two resource streams this service
// publishes. They deliberately do NOT converge: an accesskey is an independent persistent resource
// addressed by its own id (design §2.4), not by the customer it belongs to.
var (
	customerID  = uuid.FromStringOrNil("d5c9e720-0000-4000-8000-000000000001")
	accesskeyID = uuid.FromStringOrNil("d5c9e720-0000-4000-8000-000000000002")
)

// resolveSubscriptionID mirrors the resolution notifyhandler performs on the publish path
// (VOIP-1419): the payload's explicit `EventSubscriptionID()` method is the single source of
// the subscription address -- implementation is mandatory (compiler-enforced at the publish
// sites), and an empty return degrades to the `-` placeholder. Keeping the mirror here rather
// than reaching into notifyhandler internals is deliberate -- the golden table must fail if a
// method starts returning a different id space than the one pinned below.
//
// The parameter stays `any` so the table can also feed values that do not implement the
// interface; they resolve to "" (→ placeholder), matching what production's narrowed
// signature makes unrepresentable at compile time.
func resolveSubscriptionID(t *testing.T, data any) string {
	t.Helper()

	identifier, ok := data.(eventtopic.SubscriptionIdentifier)
	if !ok {
		return ""
	}

	// typed-nil guard, mirroring notifyhandler.resolveSubscriptionID: a nil pointer whose type
	// implements the interface still SATISFIES the assertion, and every real implementation
	// dereferences its receiver -- calling the method would panic. Production resolves such a
	// payload to the `-` placeholder instead.
	if v := reflect.ValueOf(data); v.Kind() == reflect.Pointer && v.IsNil() {
		return ""
	}

	return identifier.EventSubscriptionID()
}

func TestGoldenRoutingKeys(t *testing.T) {
	publisher := string(commonoutline.ServiceNameCustomerManager)

	customerData := &customer.Customer{
		ID:     customerID,
		Name:   "test customer",
		Status: customer.StatusActive,
	}

	// The wrapper actually published by customerhandler/db.go:127 and customerhandler/signup.go:109.
	createdEventData := &customer.CustomerCreatedEvent{
		Customer: customerData,
		Headless: false,
	}

	// A wrapper whose embed is nil -- no publish site produces this today, but the row pins what
	// happens if one ever does: the wrapper's nil guard returns "", hence the `-` placeholder.
	createdEventNilEmbed := &customer.CustomerCreatedEvent{
		Headless: true,
	}

	accesskeyData := &accesskey.Accesskey{
		ID:         accesskeyID,
		CustomerID: customerID,
		Name:       "default",
	}

	tests := []struct {
		name      string
		eventType string
		data      any
		expect    string
	}{
		// customer resource, created -- published as the CustomerCreatedEvent WRAPPER. The wrapper's
		// explicit method returns the embedded *Customer's id, so it lands in exactly the same key
		// space as every other customer event below.
		{
			"customer_created (wrapper, embedded customer id)",
			customer.EventTypeCustomerCreated,
			createdEventData,
			"customer-manager.customer.d5c9e720-0000-4000-8000-000000000001.created",
		},
		// Same wrapper with a nil embed: the wrapper's nil guard returns "", so the address
		// collapses to the `-` placeholder segment.
		{
			"customer_created (wrapper, nil embed -> placeholder)",
			customer.EventTypeCustomerCreated,
			createdEventNilEmbed,
			"customer-manager.customer.-.created",
		},

		// customer resource, remaining lifecycle events -- published as the bare *Customer, own id
		// is the address, returned by its explicit EventSubscriptionID method.
		{
			"customer_updated",
			customer.EventTypeCustomerUpdated,
			customerData,
			"customer-manager.customer.d5c9e720-0000-4000-8000-000000000001.updated",
		},
		{
			"customer_deleted",
			customer.EventTypeCustomerDeleted,
			customerData,
			"customer-manager.customer.d5c9e720-0000-4000-8000-000000000001.deleted",
		},
		{
			"customer_frozen",
			customer.EventTypeCustomerFrozen,
			customerData,
			"customer-manager.customer.d5c9e720-0000-4000-8000-000000000001.frozen",
		},
		{
			"customer_recovered",
			customer.EventTypeCustomerRecovered,
			customerData,
			"customer-manager.customer.d5c9e720-0000-4000-8000-000000000001.recovered",
		},
		// Splits mechanically on the FIRST underscore, so the resource stays `customer` and the
		// whole `identity_verification_updated` tail becomes the action segment. That is
		// intentional -- see eventtopic.RoutingKey -- and it keeps this event inside the same
		// `customer-manager.customer.<customer-id>.#` binding as the lifecycle events above.
		{
			"customer_identity_verification_updated",
			customer.EventTypeCustomerIdentityVerificationUpdated,
			customerData,
			"customer-manager.customer.d5c9e720-0000-4000-8000-000000000001.identity_verification_updated",
		},

		// accesskey resource -- own id is the address, NOT the owning customer's.
		{
			"accesskey_created",
			accesskey.EventTypeAccesskeyCreated,
			accesskeyData,
			"customer-manager.accesskey.d5c9e720-0000-4000-8000-000000000002.created",
		},
		{
			"accesskey_updated",
			accesskey.EventTypeAccesskeyUpdated,
			accesskeyData,
			"customer-manager.accesskey.d5c9e720-0000-4000-8000-000000000002.updated",
		},
		{
			"accesskey_deleted",
			accesskey.EventTypeAccesskeyDeleted,
			accesskeyData,
			"customer-manager.accesskey.d5c9e720-0000-4000-8000-000000000002.deleted",
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

// TestCustomerCreatedEventUsesEmbeddedID pins the address-equality half of the wrapper contract
// directly, independent of the routing-key formatting above: the wrapper's explicit method and the
// bare Customer's must resolve to the SAME address, so `customer-manager.customer.<customer-id>.#`
// delivers the created event alongside every later lifecycle event.
func TestCustomerCreatedEventUsesEmbeddedID(t *testing.T) {
	customerData := &customer.Customer{ID: customerID}
	wrapper := &customer.CustomerCreatedEvent{Customer: customerData, Headless: true}

	bare := resolveSubscriptionID(t, customerData)
	wrapped := resolveSubscriptionID(t, wrapper)

	if bare != customerID.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", customerID.String(), bare)
	}
	if wrapped != bare {
		t.Errorf("Wrong match. the wrapper must resolve to the embedded customer id. expect: %s, got: %s", bare, wrapped)
	}
	if eventtopic.IsPlaceholderSubscriptionID(wrapped) {
		t.Errorf("CustomerCreatedEvent with a non-nil embed must never resolve to the placeholder address. got: %s", wrapped)
	}
}

// TestCustomerCreatedEventNilEmbedIsPlaceholder pins the other half: with a nil embed the
// wrapper's nil guard returns "", so the address is the `-` placeholder rather than a panic or
// some accidental value. This is the metered, visible degradation path (topic_placeholder_total),
// not a silent one.
func TestCustomerCreatedEventNilEmbedIsPlaceholder(t *testing.T) {
	wrapper := &customer.CustomerCreatedEvent{Headless: true}

	res := resolveSubscriptionID(t, wrapper)
	if res != "" {
		t.Errorf("Wrong match. a nil embed marshals without an id. expect: \"\", got: %s", res)
	}
	if !eventtopic.IsPlaceholderSubscriptionID(res) {
		t.Errorf("A nil-embed CustomerCreatedEvent must resolve to the placeholder address. got: %s", res)
	}
}

// TestAccesskeyUsesOwnIDNotCustomerID pins the address choice that is easiest to get wrong by
// analogy with the parent-axis services: an accesskey event is NOT addressed by the customer it
// belongs to. A consumer following one accesskey binds
// `customer-manager.accesskey.<accesskey-id>.#`.
func TestAccesskeyUsesOwnIDNotCustomerID(t *testing.T) {
	data := &accesskey.Accesskey{ID: accesskeyID, CustomerID: customerID}

	res := resolveSubscriptionID(t, data)
	if res != accesskeyID.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", accesskeyID.String(), res)
	}
	if res == customerID.String() {
		t.Errorf("Accesskey must not be addressed by its customer id. got: %s", res)
	}
}
