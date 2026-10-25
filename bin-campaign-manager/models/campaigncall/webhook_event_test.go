package campaigncall

import (
	"encoding/json"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

func Test_CreateWebhookEvent(t *testing.T) {
	id := uuid.FromStringOrNil("6b1f0a10-0011-11ee-8a45-000000000001")
	customerID := uuid.FromStringOrNil("6b1f0a10-0012-11ee-8a45-000000000002")
	campaignID := uuid.FromStringOrNil("6b1f0a10-0013-11ee-8a45-000000000003")

	c := &Campaigncall{
		Identity:         commonidentity.Identity{ID: id, CustomerID: customerID},
		CampaignID:       campaignID,
		Status:           StatusProgressing,
		Result:           ResultSuccess,
		Destination:      &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550100001"},
		DestinationIndex: 2,
		TryCount:         3,
	}

	res, err := c.CreateWebhookEvent()
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	var got map[string]any
	if errUnmarshal := json.Unmarshal(res, &got); errUnmarshal != nil {
		t.Fatalf("Wrong match. expect: valid json, got: %v", errUnmarshal)
	}

	expectValues := map[string]any{
		"id":                id.String(),
		"customer_id":       customerID.String(),
		"campaign_id":       campaignID.String(),
		"status":            string(StatusProgressing),
		"result":            string(ResultSuccess),
		"destination_index": float64(2),
		"try_count":         float64(3),
	}
	for k, v := range expectValues {
		if got[k] != v {
			t.Errorf("Wrong match. key: %s, expect: %v, got: %v", k, v, got[k])
		}
	}

	dest, ok := got["destination"].(map[string]any)
	if !ok || dest["target"] != "+15550100001" {
		t.Errorf("Wrong match. destination: %v", got["destination"])
	}
}
