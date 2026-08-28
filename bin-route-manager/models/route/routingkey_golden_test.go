// Golden routing-key table of the global topic exchange `bin-manager.event` (VOIP-1404/1405,
// VOIP-1419).
//
// It covers EVERY event type route-manager publishes today, across all three resource namespaces
// (route / provider / providercall), and asserts the exact key that notifyhandler generates for
// the real event data type of each publish site. Since VOIP-1419 the subscription segment is
// resolved by exactly one mechanism: the mandatory, explicit EventSubscriptionID method on the
// event data type (an empty return degrades to the `-` placeholder). All three route-manager
// models return their own id. The table is what proves the returned VALUES stay put -- a method
// silently rewired to another field changes these keys, and no runtime metric would detect it
// because the keys would still be well formed.
//
// Worth pinning explicitly here: unlike most services in this rollout, NONE of route-manager's
// three published models embeds commonidentity.Identity -- Route, Provider and ProviderCall each
// declare `ID uuid.UUID` directly, and each EventSubscriptionID reads that own field.
// TestRouteModelsResolveOwnID demonstrates that rather than assuming it.
//
// The file lives in models/route (the service's designated PRIMARY model package for this table,
// chosen for practical anchoring rather than strict aggregate semantics -- the three resources
// are related but each is addressed by its own id); it is an external test package so it can
// import the sibling model packages without any import-cycle risk.
//
// MAINTENANCE: this table pins CURRENT behavior, not what the events ought to be. There is no
// `providercall_updated` today (providercallhandler publishes created + deleted only) -- if one
// is ever added, its row belongs here in the same change.
package route_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/gofrs/uuid"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/models/eventtopic"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-route-manager/models/provider"
	"monorepo/bin-route-manager/models/providercall"
	"monorepo/bin-route-manager/models/route"
)

// The three subscription addresses. route-manager's resources are INDEPENDENT top-level
// resources, each addressed by its own id, so -- unlike the parent-axis services (transcribe,
// conference, ...) -- there is no shared address for them to converge on. Three distinct ids here
// make that explicit.
var (
	routeID        = uuid.FromStringOrNil("d3f0a512-0000-4000-8000-000000000001")
	providerID     = uuid.FromStringOrNil("d3f0a512-0000-4000-8000-000000000002")
	providerCallID = uuid.FromStringOrNil("d3f0a512-0000-4000-8000-000000000003")
)

// resolveSubscriptionID mirrors the resolution notifyhandler performs on the publish path
// (VOIP-1419): the event data's explicit EventSubscriptionID method is the only source of the
// subscription segment. Non-implementing data resolves to "" (which the key builder renders as
// the `-` placeholder), as does a typed-nil implementer -- calling the method on one would
// panic, so production guards it with the same reflect check reproduced here. Keeping the
// mirror in this file rather than reaching into notifyhandler internals is deliberate: the
// golden table must keep failing if the production resolution semantics and this reproduction
// ever diverge.
func resolveSubscriptionID(t *testing.T, data any) string {
	t.Helper()

	identifier, ok := data.(eventtopic.SubscriptionIdentifier)
	if !ok {
		return ""
	}

	// typed-nil guard, mirroring notifyhandler: a nil pointer whose type implements the
	// interface still SATISFIES the assertion, and every real implementation dereferences its
	// receiver -- calling the method would panic. Production degrades such a payload to the
	// placeholder instead.
	if v := reflect.ValueOf(data); v.Kind() == reflect.Ptr && v.IsNil() {
		return ""
	}

	return identifier.EventSubscriptionID()
}

func TestGoldenRoutingKeys(t *testing.T) {
	publisher := string(commonoutline.ServiceNameRouteManager)

	routeData := &route.Route{
		ID:         routeID,
		CustomerID: uuid.Must(uuid.NewV4()),
		Name:       "test route",
		ProviderID: providerID,
		Target:     route.TargetAll,
	}

	providerData := &provider.Provider{
		ID:       providerID,
		Type:     provider.TypeSIP,
		Hostname: "test.provider.example.com",
		Name:     "test provider",
	}

	providerCallData := &providercall.ProviderCall{
		ID:         providerCallID,
		CustomerID: uuid.Must(uuid.NewV4()),
		ProviderID: providerID,
		Source:     &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+821100000001"},
	}

	tests := []struct {
		name      string
		eventType string
		data      any
		expect    string
	}{
		// route resource -- own id is the address, returned by the explicit EventSubscriptionID
		// method. pkg/routehandler/route.go:77 / :180 / :218.
		{
			"route_created",
			route.EventTypeRouteCreated,
			routeData,
			"route-manager.route.d3f0a512-0000-4000-8000-000000000001.created",
		},
		{
			"route_deleted",
			route.EventTypeRouteDeleted,
			routeData,
			"route-manager.route.d3f0a512-0000-4000-8000-000000000001.deleted",
		},
		{
			"route_updated",
			route.EventTypeRouteUpdated,
			routeData,
			"route-manager.route.d3f0a512-0000-4000-8000-000000000001.updated",
		},

		// provider resource. pkg/providerhandler/provider.go:89 / :136 / :216.
		{
			"provider_created",
			provider.EventTypeProviderCreated,
			providerData,
			"route-manager.provider.d3f0a512-0000-4000-8000-000000000002.created",
		},
		{
			"provider_deleted",
			provider.EventTypeProviderDeleted,
			providerData,
			"route-manager.provider.d3f0a512-0000-4000-8000-000000000002.deleted",
		},
		{
			"provider_updated",
			provider.EventTypeProviderUpdated,
			providerData,
			"route-manager.provider.d3f0a512-0000-4000-8000-000000000002.updated",
		},

		// providercall resource -- a SEPARATE namespace from `provider`, not a sub-namespace of
		// it. pkg/providercallhandler/providercall.go:148 / :212. See
		// TestProviderResourcePatternDoesNotMatchProviderCall for why that separation is safe.
		{
			"providercall_created",
			providercall.EventTypeProviderCallCreated,
			providerCallData,
			"route-manager.providercall.d3f0a512-0000-4000-8000-000000000003.created",
		},
		{
			"providercall_deleted",
			providercall.EventTypeProviderCallDeleted,
			providerCallData,
			"route-manager.providercall.d3f0a512-0000-4000-8000-000000000003.deleted",
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

// matchTopic reproduces AMQP topic-exchange matching for the patterns eventtopic builds: `*`
// matches exactly one segment, `#` matches zero or more. It exists so the separation assertion
// below tests the ACTUAL broker semantics rather than a string-prefix approximation, which is
// precisely the approximation that would hide a `provider` / `providercall` collision.
func matchTopic(pattern string, key string) bool {
	return matchSegments(strings.Split(pattern, "."), strings.Split(key, "."))
}

func matchSegments(pattern []string, key []string) bool {
	switch {
	case len(pattern) == 0:
		return len(key) == 0

	case pattern[0] == "#":
		// `#` absorbs zero or more segments; try every split point.
		for i := 0; i <= len(key); i++ {
			if matchSegments(pattern[1:], key[i:]) {
				return true
			}
		}
		return false

	case len(key) == 0:
		return false

	case pattern[0] == "*" || pattern[0] == key[0]:
		return matchSegments(pattern[1:], key[1:])

	default:
		return false
	}
}

// TestMatchTopic guards the matcher itself. Without this, a matcher that returned false for
// everything would make the separation assertion below pass vacuously.
func TestMatchTopic(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		key     string
		expect  bool
	}{
		{"exact", "a.b.c.d", "a.b.c.d", true},
		{"trailing hash absorbs the rest", "a.b.#", "a.b.c.d", true},
		{"trailing hash absorbs nothing", "a.b.#", "a.b", true},
		{"star matches exactly one segment", "a.*.c", "a.b.c", true},
		{"star does not match two segments", "a.*.c", "a.b.b.c", false},
		{"segment mismatch", "a.b.#", "a.bb.c.d", false},
		{"prefix is not a match", "a.b.c.d", "a.b.c", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if res := matchTopic(tt.pattern, tt.key); res != tt.expect {
				t.Errorf("Wrong match. pattern: %s, key: %s, expect: %v, got: %v", tt.pattern, tt.key, tt.expect, res)
			}
		})
	}
}

// TestProviderResourcePatternDoesNotMatchProviderCall pins the segment-boundary property that
// makes the `provider` / `providercall` naming safe. `providercall` is a prefix-extension of
// `provider` at the STRING level, so a consumer binding `route-manager.provider.#` to follow
// providers would silently also receive every providercall event IF routing keys were matched by
// string prefix. They are not: AMQP topic matching is per-SEGMENT, and `provider` !=
// `providercall` as whole segments.
//
// This is a real consumer-contract assertion, not a theoretical one -- the two resources carry
// different id spaces (a provider id and a providercall id), so a leak here would deliver events
// whose third segment means something else entirely to the binder.
func TestProviderResourcePatternDoesNotMatchProviderCall(t *testing.T) {
	publisher := string(commonoutline.ServiceNameRouteManager)

	providerKey := eventtopic.RoutingKey(publisher, provider.EventTypeProviderCreated, providerID.String())
	providerCallKey := eventtopic.RoutingKey(publisher, providercall.EventTypeProviderCallCreated, providerCallID.String())

	providerPattern := eventtopic.PatternResource(publisher, "provider")
	providerCallPattern := eventtopic.PatternResource(publisher, "providercall")

	tests := []struct {
		name    string
		pattern string
		key     string
		expect  bool
	}{
		{"provider pattern matches provider key", providerPattern, providerKey, true},
		{"provider pattern does NOT match providercall key", providerPattern, providerCallKey, false},
		{"providercall pattern matches providercall key", providerCallPattern, providerCallKey, true},
		{"providercall pattern does NOT match provider key", providerCallPattern, providerKey, false},
		// The catch-all still sees both, as it must.
		{"publisher-wide pattern matches provider key", eventtopic.PatternAll(publisher), providerKey, true},
		{"publisher-wide pattern matches providercall key", eventtopic.PatternAll(publisher), providerCallKey, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if res := matchTopic(tt.pattern, tt.key); res != tt.expect {
				t.Errorf("Wrong match. pattern: %s, key: %s, expect: %v, got: %v", tt.pattern, tt.key, tt.expect, res)
			}
		})
	}
}

// TestRouteModelsResolveOwnID pins the specific claim made in the file header: these three
// models declare `ID` directly instead of embedding commonidentity.Identity, and each
// EventSubscriptionID method reads that own field. Resolving through the helper (rather than
// calling the methods directly) additionally proves the pointer types satisfy the interface on
// the same path the table above uses.
func TestRouteModelsResolveOwnID(t *testing.T) {
	tests := []struct {
		name   string
		data   any
		expect string
	}{
		{"route", &route.Route{ID: routeID}, routeID.String()},
		{"provider", &provider.Provider{ID: providerID}, providerID.String()},
		{"providercall", &providercall.ProviderCall{ID: providerCallID}, providerCallID.String()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := resolveSubscriptionID(t, tt.data)
			if res != tt.expect {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expect, res)
			}
		})
	}
}
