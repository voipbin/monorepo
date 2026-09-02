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
// (VOIP-1419): every published event data type satisfies the mandatory
// EventSubscriptionID contract -- campaign.Campaign through the own-id
// default promoted from the embedded commonidentity.Identity, campaigncall.Campaigncall through
// its explicit parent-campaign override -- and an empty return degrades to the `-` placeholder. There is no
// JSON fallback anymore. Keeping the reproduction here rather than reaching into notifyhandler
// internals is deliberate -- the golden table must fail when a model's method stops returning
// the pinned address, which is exactly what this mirror detects.
func resolveSubscriptionID(t *testing.T, data any) string {
	t.Helper()

	if identifier, ok := data.(eventtopic.SubscriptionIdentifier); ok {
		// typed-nil guard, mirroring notifyhandler.resolveSubscriptionID: a nil pointer whose
		// type implements the interface still SATISFIES the assertion, and every real
		// implementation dereferences its receiver -- calling the method would panic. Production
		// resolves such a payload to "" (the `-` placeholder), so this guard does the same.
		if v := reflect.ValueOf(data); v.Kind() != reflect.Pointer || !v.IsNil() {
			return identifier.EventSubscriptionID()
		}
	}

	return ""
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
		// campaign resource -- own id is the address, returned through the promoted Identity default.
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
