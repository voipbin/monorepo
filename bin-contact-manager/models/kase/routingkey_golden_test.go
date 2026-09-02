// Golden routing-key table of the global topic exchange `bin-manager.event` (VOIP-1404/1405).
//
// It covers EVERY event type contact-manager publishes today, across both resource namespaces
// (contact / case), and asserts the exact key that notifyhandler generates for the real event
// data type of each publish site. The primary defect class it guards against is "the right key
// shape carrying the wrong id space": an id that is well-formed but is not the address anybody
// binds to produces keys that no instance binding ever matches, and no runtime metric can detect
// it. Design doc 1404 §4.2 / 1405 §2.3.
//
// The file lives in models/kase because the table spans every model package of the service and
// the case is the axis all internal events of this service address; it is an external test
// package so it can import the sibling model packages without any import-cycle risk.
//
// MAINTENANCE: this table pins CURRENT behavior. Adding a publish site to contact-manager without
// adding its row here is the failure mode the table exists to prevent.
package kase_test

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/eventtopic"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"monorepo/bin-contact-manager/models/casenote"
	"monorepo/bin-contact-manager/models/contact"
	"monorepo/bin-contact-manager/models/kase"
)

// caseID is the single subscription address every case-scoped contact-manager event must carry,
// whichever of the four payload types the publish site uses.
var caseID = uuid.FromStringOrNil("c0a7e1d2-0000-4000-8000-000000000001")

// contactID is the address of the customer-facing contact lifecycle events, whose own id IS the
// address, returned through the payload's EventSubscriptionID promoted from the embedded
// commonidentity.Identity (VOIP-1419).
var contactID = uuid.FromStringOrNil("c0a7e1d2-0000-4000-8000-000000000002")

// resolveSubscriptionID mirrors the resolution notifyhandler performs on the publish path
// (VOIP-1419): every published event data type satisfies the EventSubscriptionID contract
// (own-id types via the default promoted from the embedded commonidentity.Identity, the
// case-scoped event wrappers via their explicit case-id overrides) --
// the contract is mandatory, enforced at compile time by the narrowed PublishEvent signature --
// and an empty return degrades to the `-` placeholder. Keeping the reproduction here rather than
// reaching into notifyhandler internals is deliberate -- the golden table must fail when a model's
// method starts returning the wrong id space, which is exactly the defect class this file pins.
//
// The parameter stays `any` so the table can also feed payloads that do not implement the
// interface (e.g. the []byte regression row below); such data resolves to "" -> placeholder.
func resolveSubscriptionID(t *testing.T, data any) string {
	t.Helper()

	identifier, ok := data.(eventtopic.SubscriptionIdentifier)
	if !ok {
		return ""
	}

	// typed-nil guard, mirroring notifyhandler.resolveSubscriptionID: a nil pointer whose type
	// implements the interface still SATISFIES the assertion, and every real implementation
	// dereferences its receiver -- calling the method would panic. Such a payload marshals to
	// `null` and production resolves it to the `-` placeholder.
	if v := reflect.ValueOf(data); v.Kind() == reflect.Pointer && v.IsNil() {
		return ""
	}

	return identifier.EventSubscriptionID()
}

func TestGoldenRoutingKeys(t *testing.T) {
	publisher := string(commonoutline.ServiceNameContactManager)

	// contacthandler.publishEvent passes the *WebhookMessage (VOIP-1405 §3.1 -- it used to pass
	// the []byte of CreateWebhookEvent, which double-encoded the payload and carried no address
	// at all). Its EventSubscriptionID method returns the contact's own id -- the address
	// subscribers use.
	contactData := (&contact.Contact{
		Identity: commonidentity.Identity{
			ID:         contactID,
			CustomerID: uuid.Must(uuid.NewV4()),
		},
		FirstName: "sungtae",
		LastName:  "kim",
	}).ConvertWebhookMessage()

	// casehandler.CaseNoteCreate publishes the CaseNote itself -- own id present, but overridden
	// to the case axis.
	caseNoteData := &casenote.CaseNote{
		ID:         uuid.Must(uuid.NewV4()),
		CustomerID: uuid.Must(uuid.NewV4()),
		CaseID:     caseID,
		AuthorType: casenote.AuthorTypeAgent,
		Text:       "note",
	}

	// casehandler.CaseNoteDelete publishes this struct. SILENT-FAILURE CLASS (design §2.3): the
	// payload carries a top-level `id` (the note id), so without the override the default
	// resolution would address it by that note id -- a well-formed key nobody binds to, invisible
	// to the placeholder metric.
	caseNoteDeletedData := &casenote.CaseNoteDeletedEvent{
		ID:         uuid.Must(uuid.NewV4()),
		CaseID:     caseID,
		CustomerID: uuid.Must(uuid.NewV4()),
	}

	// casehandler.CaseTagAdd / CaseTagRemove -- one struct, two event types.
	caseTagData := &kase.CaseTagEvent{
		CaseID: caseID,
		TagID:  uuid.Must(uuid.NewV4()),
	}

	// casehandler.UpdateContact -- one site, one struct, TWO event types selected dynamically by
	// whether contactID is uuid.Nil. Both branches are enumerated below.
	caseContactAttachedData := &kase.CaseContactEvent{
		CaseID:    caseID,
		ContactID: contactID,
	}
	caseContactDetachedData := &kase.CaseContactEvent{
		CaseID:    caseID,
		ContactID: uuid.Nil,
	}

	tests := []struct {
		name      string
		eventType string
		data      any
		expect    string
	}{
		// contact resource -- own id is the address, returned through the promoted Identity default.
		{
			"contact_created",
			contact.EventTypeContactCreated,
			contactData,
			"contact-manager.contact.c0a7e1d2-0000-4000-8000-000000000002.created",
		},
		{
			"contact_updated",
			contact.EventTypeContactUpdated,
			contactData,
			"contact-manager.contact.c0a7e1d2-0000-4000-8000-000000000002.updated",
		},
		{
			"contact_deleted",
			contact.EventTypeContactDeleted,
			contactData,
			"contact-manager.contact.c0a7e1d2-0000-4000-8000-000000000002.deleted",
		},

		// case resource -- every internal case event normalizes to resource `case` (the event
		// type splits on the FIRST underscore), and every one of them is addressed by the case id.
		{
			"case_note_created",
			casenote.EventTypeCaseNoteCreated,
			caseNoteData,
			"contact-manager.case.c0a7e1d2-0000-4000-8000-000000000001.note_created",
		},
		{
			// MANDATORY row (design §2.3): address must be the case id, NOT the note id in the
			// payload.
			"case_note_deleted",
			casenote.EventTypeCaseNoteDeleted,
			caseNoteDeletedData,
			"contact-manager.case.c0a7e1d2-0000-4000-8000-000000000001.note_deleted",
		},
		{
			"case_tag_added",
			kase.EventTypeCaseTagAdded,
			caseTagData,
			"contact-manager.case.c0a7e1d2-0000-4000-8000-000000000001.tag_added",
		},
		{
			"case_tag_removed",
			kase.EventTypeCaseTagRemoved,
			caseTagData,
			"contact-manager.case.c0a7e1d2-0000-4000-8000-000000000001.tag_removed",
		},
		{
			// contact_update dynamic branch 1 of 2.
			"case_contact_attributed",
			kase.EventTypeCaseContactAttributed,
			caseContactAttachedData,
			"contact-manager.case.c0a7e1d2-0000-4000-8000-000000000001.contact_attributed",
		},
		{
			// contact_update dynamic branch 2 of 2 -- contact_id is uuid.Nil here, and the
			// address must not degrade with it.
			"case_contact_detached",
			kase.EventTypeCaseContactDetached,
			caseContactDetachedData,
			"contact-manager.case.c0a7e1d2-0000-4000-8000-000000000001.contact_detached",
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

// TestGoldenRoutingKeysCaseAxisConvergence pins the property the case rows above exist to
// protect (design §4 address-convergence note): notes, tags and contact attribution all resolve
// to the SAME address, so a consumer following one case binds
// `contact-manager.case.<case-id>.#` and receives the whole case lifecycle.
//
// Each payload below carries a different own id (or none at all); none of them may leak into the
// address.
func TestGoldenRoutingKeysCaseAxisConvergence(t *testing.T) {
	expect := caseID.String()

	tests := []struct {
		name string
		data any
	}{
		{"case note", &casenote.CaseNote{ID: uuid.Must(uuid.NewV4()), CaseID: caseID}},
		{"case note deleted", &casenote.CaseNoteDeletedEvent{ID: uuid.Must(uuid.NewV4()), CaseID: caseID}},
		{"case tag", &kase.CaseTagEvent{CaseID: caseID, TagID: uuid.Must(uuid.NewV4())}},
		{"case contact", &kase.CaseContactEvent{CaseID: caseID, ContactID: uuid.Must(uuid.NewV4())}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := resolveSubscriptionID(t, tt.data)
			if res != expect {
				t.Errorf("Wrong match. expect: %s, got: %s", expect, res)
			}
		})
	}
}

// TestGoldenRoutingKeysContactPayloadIsNotBytes pins VOIP-1405 §3.1. Before the fix,
// contacthandler.publishEvent handed PublishEvent the []byte of CreateWebhookEvent(); a []byte
// carries no EventSubscriptionID method, so every contact event would have published under the
// `-` placeholder address. The row asserts what the fixed path produces AND what the old one
// would have, so a regression cannot pass silently.
func TestGoldenRoutingKeysContactPayloadIsNotBytes(t *testing.T) {
	c := &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         contactID,
			CustomerID: uuid.Must(uuid.NewV4()),
		},
	}

	got := resolveSubscriptionID(t, c.ConvertWebhookMessage())
	if got != contactID.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", contactID, got)
	}

	raw, err := c.CreateWebhookEvent()
	if err != nil {
		t.Fatalf("Could not create the webhook event. err: %v", err)
	}
	if regressed := resolveSubscriptionID(t, raw); regressed != "" {
		t.Errorf("The []byte payload must not resolve to any address. got: %s", regressed)
	}
	if key := eventtopic.RoutingKey(string(commonoutline.ServiceNameContactManager), contact.EventTypeContactCreated, ""); key != "contact-manager.contact.-.created" {
		t.Errorf("Wrong match. expect: contact-manager.contact.-.created, got: %s", key)
	}
}
