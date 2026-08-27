// Golden routing-key table of the global topic exchange `bin-manager.event` (VOIP-1404/1405).
//
// It covers EVERY event type customer-manager publishes today, across both live resource
// namespaces (customer / accesskey), and asserts the exact key that notifyhandler generates for the
// real event data type of each publish site. The primary defect class it guards against is "the
// right key shape carrying the wrong id space": an event published under an id that is not the
// resource's subscription address produces well-formed keys that no instance binding ever matches,
// and no runtime metric can detect it. Design doc §2.4 / §4.
//
// customer-manager is a default-fallback service: `*customer.Customer` and
// `*accesskey.Accesskey` are both addressed by their OWN id, so NO type here carries an
// eventtopic.SubscriptionIdentifier override.
//
// THE TRAP THIS FILE EXISTS FOR (design §2.4, explicit): `customer.CustomerCreatedEvent`
// ANONYMOUSLY EMBEDS `*Customer`. Two consequences are pinned below and must never be broken:
//
//  1. The embed's `id` is PROMOTED into the wrapper's JSON, so `customer_created` resolves to the
//     customer id through the ordinary default fallback -- no special handling anywhere.
//  2. Because the embed is anonymous, ANY method added to `*Customer` -- including an
//     `EventSubscriptionID()` override -- is promoted to `*CustomerCreatedEvent` too. Adding one to
//     `*Customer` would therefore silently re-address BOTH types at once. Design §2.4 states it
//     normatively: no override may ever be added to `*Customer`. The assertions below fail the
//     moment either type starts satisfying the interface.
//
// A nil embed is the third pinned case: it marshals WITHOUT an `id` (encoding/json skips fields
// reached through a nil anonymous pointer), so the address collapses to the `-` placeholder. That
// is the correct outcome for a payload carrying no identity, and the row exists so the behavior is
// a pinned fact rather than an assumption.
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
	"encoding/json"
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
	// happens if one ever does: no `id` in the marshaled payload, hence the `-` placeholder.
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
		// customer resource, created -- published as the CustomerCreatedEvent WRAPPER. The address
		// comes from the anonymously embedded *Customer's promoted `id`, so it lands in exactly the
		// same key space as every other customer event below.
		{
			"customer_created (wrapper, embedded id promoted)",
			customer.EventTypeCustomerCreated,
			createdEventData,
			"customer-manager.customer.d5c9e720-0000-4000-8000-000000000001.created",
		},
		// Same wrapper with a nil embed: encoding/json skips the promoted fields, so no `id` is
		// present and the address collapses to the `-` placeholder segment.
		{
			"customer_created (wrapper, nil embed -> placeholder)",
			customer.EventTypeCustomerCreated,
			createdEventNilEmbed,
			"customer-manager.customer.-.created",
		},

		// customer resource, remaining lifecycle events -- published as the bare *Customer, own id
		// is the address, resolved by the default JSON fallback.
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

// TestCustomerCreatedEventPromotesEmbeddedID pins the id-promotion half of the wrapper trap
// directly, independent of the routing-key formatting above: the wrapper and the bare Customer must
// resolve to the SAME address, so `customer-manager.customer.<customer-id>.#` delivers the created
// event alongside every later lifecycle event.
func TestCustomerCreatedEventPromotesEmbeddedID(t *testing.T) {
	customerData := &customer.Customer{ID: customerID}
	wrapper := &customer.CustomerCreatedEvent{Customer: customerData, Headless: true}

	bare := resolveSubscriptionID(t, customerData)
	promoted := resolveSubscriptionID(t, wrapper)

	if bare != customerID.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", customerID.String(), bare)
	}
	if promoted != bare {
		t.Errorf("Wrong match. the wrapper must resolve to the embedded customer id. expect: %s, got: %s", bare, promoted)
	}
	if eventtopic.IsPlaceholderSubscriptionID(promoted) {
		t.Errorf("CustomerCreatedEvent with a non-nil embed must never resolve to the placeholder address. got: %s", promoted)
	}
}

// TestCustomerCreatedEventNilEmbedIsPlaceholder pins the other half: with a nil embed there is no
// promoted `id` at all, so the address is the `-` placeholder rather than some accidental value.
// This is the metered, visible degradation path (topic_placeholder_total), not a silent one.
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

// TestCustomerUsesDefaultSubscriptionID pins the normative rule of design §2.4: no override may
// ever be added to *Customer. Because CustomerCreatedEvent anonymously embeds *Customer, a method
// on *Customer promotes to the wrapper as well -- so BOTH types are asserted here, and both must
// stay on the default JSON `id` fallback. *Accesskey is asserted for the same reason its own id is
// the address: an independent persistent resource (design §2.4).
func TestCustomerUsesDefaultSubscriptionID(t *testing.T) {
	tests := []struct {
		name string
		data any
	}{
		{"Customer", &customer.Customer{ID: customerID}},
		{"CustomerCreatedEvent", &customer.CustomerCreatedEvent{Customer: &customer.Customer{ID: customerID}}},
		{"Accesskey", &accesskey.Accesskey{ID: accesskeyID, CustomerID: customerID}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, ok := tt.data.(eventtopic.SubscriptionIdentifier); ok {
				t.Errorf("%s must not implement SubscriptionIdentifier. its own id is the subscription address, and an override on *Customer would promote to *CustomerCreatedEvent (design §2.4).", tt.name)
			}
		})
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
