// Golden routing-key table of the global topic exchange `bin-manager.event` (VOIP-1404/1405).
//
// It covers EVERY event type campaign-manager publishes today, across both live resource
// namespaces (campaign / campaigncall), and asserts the exact key that notifyhandler generates
// for the real event data type of each publish site. The primary defect class it guards against
// is "the right key shape carrying the wrong id space": a child resource published under its own
// id produces well-formed keys that no campaign-following binding ever matches, and no runtime
// metric can detect it. Design doc §2.3 / §4.
//
// The file lives in models/campaign because the table spans every publishing model package of the
// service and the campaign is the resource all of them address; it is an external test package so
// it can import the sibling model packages without any import-cycle risk.
//
// MAINTENANCE: this table pins CURRENT behavior. The `outplan_created/updated/deleted` constants
// in models/outplan are DEAD (no publish site anywhere in the service) and are deliberately
// excluded -- design §4 dead-constant list. If an outplan publish path is ever added, add its rows
// here in the same change.
package campaign_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-campaign-manager/models/campaign"
	"monorepo/bin-campaign-manager/models/campaigncall"
	"monorepo/bin-common-handler/models/eventtopic"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
)

// campaignID is the single subscription address every campaign-manager event of one campaign must
// carry, regardless of which resource namespace the event lives in.
var campaignID = uuid.FromStringOrNil("7c41a9e0-0000-4000-8000-000000000001")

// resolveSubscriptionID mirrors the resolution notifyhandler performs on the publish path
// (1404 design §4.2 / §5.2): the opt-in interface first, then -- ONLY when no override exists --
// the top-level "id" of the marshaled payload. Keeping it here rather than reaching into
// notifyhandler internals is deliberate -- the golden table must fail when a model stops
// implementing the interface, which is exactly what this two-step reproduction detects.
//
// The early return below is the load-bearing half: an override that EXISTS is authoritative even
// when it yields "" or uuid.Nil, so the JSON fallback must never run behind it. This matches
// notifyhandler.resolveSubscriptionOverride's hasOverride semantics exactly; if the two ever
// diverge, this table stops reproducing what the publish path actually generates.
func resolveSubscriptionID(t *testing.T, data any) string {
	t.Helper()

	if identifier, ok := data.(eventtopic.SubscriptionIdentifier); ok {
		// typed-nil guard, mirroring notifyhandler.resolveSubscriptionOverride: a nil pointer whose
		// type implements the interface still SATISFIES the assertion, and every real implementation
		// dereferences its receiver -- calling the method would panic. Production reports "no
		// override" for such a payload, so this guard falls through to the JSON half below rather
		// than returning early; `null` carries no top-level `id` either, so both halves agree on the
		// `-` placeholder.
		if v := reflect.ValueOf(data); v.Kind() != reflect.Ptr || !v.IsNil() {
			return identifier.EventSubscriptionID()
		}
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
	publisher := string(commonoutline.ServiceNameCampaignManager)

	campaignData := &campaign.Campaign{
		Identity: commonidentity.Identity{
			ID:         campaignID,
			CustomerID: uuid.Must(uuid.NewV4()),
		},
		Status: campaign.StatusRun,
	}

	campaigncallData := &campaigncall.Campaigncall{
		Identity: commonidentity.Identity{
			ID:         uuid.Must(uuid.NewV4()), // campaigncall-id: stable, but the wrong address
			CustomerID: uuid.Must(uuid.NewV4()),
		},
		CampaignID: campaignID,
		Status:     campaigncall.StatusDialing,
	}

	tests := []struct {
		name      string
		eventType string
		data      any
		expect    string
	}{
		// campaign resource -- own id is the address, resolved by the default JSON fallback.
		{
			"campaign_created",
			campaign.EventTypeCampaignCreated,
			campaignData,
			"campaign-manager.campaign.7c41a9e0-0000-4000-8000-000000000001.created",
		},
		{
			"campaign_updated",
			campaign.EventTypeCampaignUpdated,
			campaignData,
			"campaign-manager.campaign.7c41a9e0-0000-4000-8000-000000000001.updated",
		},
		{
			"campaign_deleted",
			campaign.EventTypeCampaignDeleted,
			campaignData,
			"campaign-manager.campaign.7c41a9e0-0000-4000-8000-000000000001.deleted",
		},
		// The status_* trio splits mechanically on the FIRST underscore, so the action segment
		// keeps the `status_` prefix. That is intentional -- see eventtopic.RoutingKey.
		{
			"campaign_status_run",
			campaign.EventTypeCampaignStatusRun,
			campaignData,
			"campaign-manager.campaign.7c41a9e0-0000-4000-8000-000000000001.status_run",
		},
		{
			"campaign_status_stopping",
			campaign.EventTypeCampaignStatusStopping,
			campaignData,
			"campaign-manager.campaign.7c41a9e0-0000-4000-8000-000000000001.status_stopping",
		},
		{
			"campaign_status_stop",
			campaign.EventTypeCampaignStatusStop,
			campaignData,
			"campaign-manager.campaign.7c41a9e0-0000-4000-8000-000000000001.status_stop",
		},

		// campaigncall resource -- addressed by the parent campaign-id, not the campaigncall-id.
		{
			"campaigncall_created",
			campaigncall.EventTypeCampaigncallCreated,
			campaigncallData,
			"campaign-manager.campaigncall.7c41a9e0-0000-4000-8000-000000000001.created",
		},
		{
			"campaigncall_updated",
			campaigncall.EventTypeCampaigncallUpdated,
			campaigncallData,
			"campaign-manager.campaigncall.7c41a9e0-0000-4000-8000-000000000001.updated",
		},
		{
			"campaigncall_deleted",
			campaigncall.EventTypeCampaigncallDeleted,
			campaigncallData,
			"campaign-manager.campaigncall.7c41a9e0-0000-4000-8000-000000000001.deleted",
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
// event of one campaign resolves to the same subscription address, so a consumer following that
// campaign binds `campaign-manager.<resource>.<campaign-id>.#` per namespace and receives
// everything.
func TestGoldenRoutingKeysShareOneAddress(t *testing.T) {
	expect := campaignID.String()

	tests := []struct {
		name string
		data any
	}{
		{"campaign", &campaign.Campaign{Identity: commonidentity.Identity{ID: campaignID}}},
		{"campaigncall", &campaigncall.Campaigncall{Identity: commonidentity.Identity{ID: uuid.Must(uuid.NewV4())}, CampaignID: campaignID}},
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

// TestCampaignUsesDefaultSubscriptionID pins the deliberate absence of an override on Campaign:
// its own id IS the address, so implementing the interface would be redundant and the default
// JSON `id` extraction must keep covering it.
func TestCampaignUsesDefaultSubscriptionID(t *testing.T) {
	var data any = &campaign.Campaign{Identity: commonidentity.Identity{ID: campaignID}}

	if _, ok := data.(eventtopic.SubscriptionIdentifier); ok {
		t.Errorf("Campaign must not implement SubscriptionIdentifier. its own id is the subscription address.")
	}
}
