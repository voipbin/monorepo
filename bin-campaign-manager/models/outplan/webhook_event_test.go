package outplan

import (
	"encoding/json"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

func Test_CreateWebhookEvent(t *testing.T) {
	id := uuid.FromStringOrNil("6b1f0a10-0021-11ee-8a45-000000000001")
	customerID := uuid.FromStringOrNil("6b1f0a10-0022-11ee-8a45-000000000002")

	o := &Outplan{
		Identity:     commonidentity.Identity{ID: id, CustomerID: customerID},
		Name:         "test name",
		Detail:       "test detail",
		Source:       &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550100002"},
		DialTimeout:  30000,
		TryInterval:  60000,
		MaxTryCount0: 5,
		MaxTryCount4: 1,
	}

	res, err := o.CreateWebhookEvent()
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	var got map[string]any
	if errUnmarshal := json.Unmarshal(res, &got); errUnmarshal != nil {
		t.Fatalf("Wrong match. expect: valid json, got: %v", errUnmarshal)
	}

	expectValues := map[string]any{
		"id":              id.String(),
		"customer_id":     customerID.String(),
		"name":            "test name",
		"detail":          "test detail",
		"dial_timeout":    float64(30000),
		"try_interval":    float64(60000),
		"max_try_count_0": float64(5),
		"max_try_count_4": float64(1),
	}
	for k, v := range expectValues {
		if got[k] != v {
			t.Errorf("Wrong match. key: %s, expect: %v, got: %v", k, v, got[k])
		}
	}

	source, ok := got["source"].(map[string]any)
	if !ok || source["target"] != "+15550100002" {
		t.Errorf("Wrong match. source: %v", got["source"])
	}
}
