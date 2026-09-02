// Golden routing-key table of the global topic exchange `bin-manager.event` (VOIP-1404/1405).
//
// It covers EVERY event type conference-manager publishes today, across both resource namespaces
// (conference / conferencecall), and asserts the exact key that notifyhandler generates for the
// real event data type of each publish site. The primary defect class it guards against is "the
// right key shape carrying the wrong id space": a participant published under its own id produces
// well-formed keys that no conference-following binding ever matches, and no runtime metric can
// detect it. Design doc §2.3 / §4.
//
// The file lives in models/conference because the table spans every publishing model package of
// the service and the conference is the resource all of them address; it is an external test
// package so it can import the sibling model packages without any import-cycle risk.
//
// MAINTENANCE: this table pins CURRENT behavior, not what the events ought to be.
package conference_test

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/eventtopic"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-conference-manager/models/conference"
	"monorepo/bin-conference-manager/models/conferencecall"
)

// conferenceID is the single subscription address every conference-manager event of one
// conference session must carry, regardless of which resource namespace the event lives in.
var conferenceID = uuid.FromStringOrNil("3b52d7c1-0000-4000-8000-000000000001")

// resolveSubscriptionID mirrors the resolution notifyhandler performs on the publish path
// (VOIP-1419): every published event data type satisfies the mandatory
// eventtopic.SubscriptionIdentifier contract -- conference.Conference through the own-id
// default promoted from the embedded commonidentity.Identity, conferencecall.Conferencecall
// through its explicit parent-conference override -- and an empty return
// degrades to the `-` placeholder. Keeping the reproduction here rather than reaching into
// notifyhandler internals is deliberate -- the golden table must fail when a model stops
// implementing the interface or changes what its method returns.
//
// The parameter stays `any` (not the interface) so the helper can also be fed non-implementing
// payloads, which resolve to "" -- the same placeholder degrade the production path yields.
func resolveSubscriptionID(t *testing.T, data any) string {
	t.Helper()

	identifier, ok := data.(eventtopic.SubscriptionIdentifier)
	if !ok {
		return ""
	}

	// typed-nil guard, mirroring notifyhandler: a nil pointer whose type implements the
	// interface still SATISFIES the assertion, and every real implementation dereferences its
	// receiver -- calling the method would panic. Production resolves such a payload to the
	// placeholder, so the helper returns "" instead of calling the method.
	if v := reflect.ValueOf(data); v.Kind() == reflect.Pointer && v.IsNil() {
		return ""
	}

	return identifier.EventSubscriptionID()
}

func TestGoldenRoutingKeys(t *testing.T) {
	publisher := string(commonoutline.ServiceNameConferenceManager)

	conferenceData := &conference.Conference{
		Identity: commonidentity.Identity{
			ID:         conferenceID,
			CustomerID: uuid.Must(uuid.NewV4()),
		},
		Type:   conference.TypeConference,
		Status: conference.StatusProgressing,
	}

	conferencecallData := &conferencecall.Conferencecall{
		Identity: commonidentity.Identity{
			ID:         uuid.Must(uuid.NewV4()), // conferencecall-id: stable, but the wrong address
			CustomerID: uuid.Must(uuid.NewV4()),
		},
		ConferenceID: conferenceID,
		Status:       conferencecall.StatusJoined,
	}

	tests := []struct {
		name      string
		eventType string
		data      any
		expect    string
	}{
		// conference resource -- own id is the address, returned through the EventSubscriptionID
		// promoted from the embedded commonidentity.Identity (VOIP-1419).
		{
			"conference_created",
			conference.EventTypeConferenceCreated,
			conferenceData,
			"conference-manager.conference.3b52d7c1-0000-4000-8000-000000000001.created",
		},
		{
			"conference_updated",
			conference.EventTypeConferenceUpdated,
			conferenceData,
			"conference-manager.conference.3b52d7c1-0000-4000-8000-000000000001.updated",
		},
		{
			"conference_deleted",
			conference.EventTypeConferenceDeleted,
			conferenceData,
			"conference-manager.conference.3b52d7c1-0000-4000-8000-000000000001.deleted",
		},

		// conferencecall resource -- addressed by the parent conference-id, not the
		// conferencecall-id. All four lifecycle events of a participant land on the conference
		// address.
		{
			"conferencecall_joining",
			conferencecall.EventTypeConferencecallJoining,
			conferencecallData,
			"conference-manager.conferencecall.3b52d7c1-0000-4000-8000-000000000001.joining",
		},
		{
			"conferencecall_joined",
			conferencecall.EventTypeConferencecallJoined,
			conferencecallData,
			"conference-manager.conferencecall.3b52d7c1-0000-4000-8000-000000000001.joined",
		},
		{
			"conferencecall_leaving",
			conferencecall.EventTypeConferencecallLeaving,
			conferencecallData,
			"conference-manager.conferencecall.3b52d7c1-0000-4000-8000-000000000001.leaving",
		},
		{
			"conferencecall_leaved",
			conferencecall.EventTypeConferencecallLeaved,
			conferencecallData,
			"conference-manager.conferencecall.3b52d7c1-0000-4000-8000-000000000001.leaved",
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

// TestGoldenRoutingKeysShareOneAddress pins the property the table above exists to protect: every
// event of one conference session -- including the events of DIFFERENT participants -- resolves
// to the same subscription address, so a consumer following that conference binds
// `conference-manager.<resource>.<conference-id>.#` per namespace and receives everything.
func TestGoldenRoutingKeysShareOneAddress(t *testing.T) {
	expect := conferenceID.String()

	tests := []struct {
		name string
		data any
	}{
		{"conference", &conference.Conference{Identity: commonidentity.Identity{ID: conferenceID}}},
		{"conferencecall participant a", &conferencecall.Conferencecall{Identity: commonidentity.Identity{ID: uuid.Must(uuid.NewV4())}, ConferenceID: conferenceID}},
		{"conferencecall participant b", &conferencecall.Conferencecall{Identity: commonidentity.Identity{ID: uuid.Must(uuid.NewV4())}, ConferenceID: conferenceID}},
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
