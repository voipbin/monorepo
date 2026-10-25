package campaign

import (
	"encoding/json"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

func Test_CreateWebhookEvent(t *testing.T) {
	id := uuid.FromStringOrNil("6b1f0a10-0001-11ee-8a45-000000000001")
	customerID := uuid.FromStringOrNil("6b1f0a10-0002-11ee-8a45-000000000002")
	flowID := uuid.FromStringOrNil("6b1f0a10-0003-11ee-8a45-000000000003")
	tmCreate := time.Date(2020, 10, 10, 3, 30, 17, 0, time.UTC)

	c := &Campaign{
		Identity: commonidentity.Identity{ID: id, CustomerID: customerID},
		Type:     TypeCall,
		Execute:  ExecuteRun,
		Name:     "test name",
		Detail:   "test detail",
		Status:   StatusRun,
		FlowID:   flowID,
		TMCreate: &tmCreate,
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
		"id":          id.String(),
		"customer_id": customerID.String(),
		"type":        string(TypeCall),
		"name":        "test name",
		"detail":      "test detail",
		"status":      string(StatusRun),
		"tm_create":   "2020-10-10T03:30:17Z",
	}
	for k, v := range expectValues {
		if got[k] != v {
			t.Errorf("Wrong match. key: %s, expect: %v, got: %v", k, v, got[k])
		}
	}

	// internal fields must not be exposed through the webhook.
	for _, k := range []string{"execute", "flow_id"} {
		if _, ok := got[k]; ok {
			t.Errorf("Wrong match. key %s must not be exposed", k)
		}
	}
}
