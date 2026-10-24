package number

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	commonidentity "monorepo/bin-common-handler/models/identity"
)

func newWebhookTestNumber() *Number {
	tmPurchase := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	tmRenew := time.Date(2024, 2, 3, 4, 5, 6, 0, time.UTC)
	tmCreate := time.Date(2024, 3, 4, 5, 6, 7, 0, time.UTC)
	tmUpdate := time.Date(2024, 4, 5, 6, 7, 8, 0, time.UTC)
	tmDelete := time.Date(2024, 5, 6, 7, 8, 9, 0, time.UTC)

	return &Number{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("11111111-aaaa-4bbb-8ccc-000000000001"),
			CustomerID: uuid.FromStringOrNil("22222222-aaaa-4bbb-8ccc-000000000002"),
		},
		Number:              "+15550100001",
		Type:                TypeNormal,
		CallFlowID:          uuid.FromStringOrNil("33333333-aaaa-4bbb-8ccc-000000000003"),
		MessageFlowID:       uuid.FromStringOrNil("44444444-aaaa-4bbb-8ccc-000000000004"),
		Name:                "test name",
		Detail:              "test detail",
		ProviderName:        ProviderNameTelnyx,
		ProviderReferenceID: "provider-ref-1",
		Status:              StatusActive,
		T38Enabled:          true,
		EmergencyEnabled:    false,
		Metadata:            Metadata{RTPDebug: true},
		TMPurchase:          &tmPurchase,
		TMRenew:             &tmRenew,
		TMCreate:            &tmCreate,
		TMUpdate:            &tmUpdate,
		TMDelete:            &tmDelete,
	}
}

func Test_ConvertWebhookMessage(t *testing.T) {
	n := newWebhookTestNumber()

	tmPurchase := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	tmRenew := time.Date(2024, 2, 3, 4, 5, 6, 0, time.UTC)
	tmCreate := time.Date(2024, 3, 4, 5, 6, 7, 0, time.UTC)
	tmUpdate := time.Date(2024, 4, 5, 6, 7, 8, 0, time.UTC)
	tmDelete := time.Date(2024, 5, 6, 7, 8, 9, 0, time.UTC)

	expect := &WebhookMessage{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("11111111-aaaa-4bbb-8ccc-000000000001"),
			CustomerID: uuid.FromStringOrNil("22222222-aaaa-4bbb-8ccc-000000000002"),
		},
		Number:           "+15550100001",
		Type:             TypeNormal,
		CallFlowID:       uuid.FromStringOrNil("33333333-aaaa-4bbb-8ccc-000000000003"),
		MessageFlowID:    uuid.FromStringOrNil("44444444-aaaa-4bbb-8ccc-000000000004"),
		Name:             "test name",
		Detail:           "test detail",
		Status:           StatusActive,
		T38Enabled:       true,
		EmergencyEnabled: false,
		Metadata:         Metadata{RTPDebug: true},
		TMPurchase:       &tmPurchase,
		TMRenew:          &tmRenew,
		TMCreate:         &tmCreate,
		TMUpdate:         &tmUpdate,
		TMDelete:         &tmDelete,
	}

	res := n.ConvertWebhookMessage()
	if !reflect.DeepEqual(expect, res) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expect, res)
	}
}

func Test_CreateWebhookEvent(t *testing.T) {
	n := newWebhookTestNumber()

	res, err := n.CreateWebhookEvent()
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	expect := `{"id":"11111111-aaaa-4bbb-8ccc-000000000001","customer_id":"22222222-aaaa-4bbb-8ccc-000000000002","number":"+15550100001","type":"normal","call_flow_id":"33333333-aaaa-4bbb-8ccc-000000000003","message_flow_id":"44444444-aaaa-4bbb-8ccc-000000000004","name":"test name","detail":"test detail","status":"active","t38_enabled":true,"emergency_enabled":false,"metadata":{"rtp_debug":true},"tm_purchase":"2024-01-02T03:04:05Z","tm_renew":"2024-02-03T04:05:06Z","tm_create":"2024-03-04T05:06:07Z","tm_update":"2024-04-05T06:07:08Z","tm_delete":"2024-05-06T07:08:09Z"}`

	var gotMap, expectMap map[string]any
	if err := json.Unmarshal(res, &gotMap); err != nil {
		t.Fatalf("Could not unmarshal the result: %v", err)
	}
	if err := json.Unmarshal([]byte(expect), &expectMap); err != nil {
		t.Fatalf("Could not unmarshal the expectation: %v", err)
	}

	if !reflect.DeepEqual(expectMap, gotMap) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expectMap, gotMap)
	}

	// provider internals must not be exposed
	for _, key := range []string{"provider_name", "provider_reference_id"} {
		if _, ok := gotMap[key]; ok {
			t.Errorf("Wrong match. expect: %s is not exposed, got: exposed", key)
		}
	}
}
