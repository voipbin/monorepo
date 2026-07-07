package message

import (
	"encoding/json"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// Test_ConvertWebhookMessage_NeverCopiesCaseID is the explicit
// leak-prevention negative test required by contact-case-management
// design §4.3/§4.4/§4.5: Metadata.ContactCaseID-derived CaseID hints are
// purely internal case-linking plumbing and must NEVER reach the
// customer-facing webhook payload. ConvertWebhookMessage is the sole
// function that builds the customer webhook's WebhookMessage value
// (webhook.go's PublishWebhook path calls data.CreateWebhookEvent() ->
// ConvertWebhookMessage()); it must deliberately leave CaseID unset even
// when the source Message has it populated.
func Test_ConvertWebhookMessage_NeverCopiesCaseID(t *testing.T) {
	caseID := uuid.FromStringOrNil("f1b2c3d4-000c-000c-000c-000000000001")

	m := &Message{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("f1b2c3d4-000c-000c-000c-000000000002"),
		},
		CaseID: &caseID,
	}

	res := m.ConvertWebhookMessage()

	if res.CaseID != nil {
		t.Fatalf("ConvertWebhookMessage must never copy CaseID onto the customer-facing webhook payload. got: %v", *res.CaseID)
	}

	// Belt-and-suspenders: marshal the WebhookMessage as CreateWebhookEvent
	// does, and assert "case_id" is not merely nil but structurally absent
	// from the wire payload (omitempty).
	data, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}
	if _, ok := raw["case_id"]; ok {
		t.Fatalf("case_id must be structurally absent from the customer webhook payload, got: %s", data)
	}
}

// Test_WebhookMessage_CaseID_JSONRoundTrip verifies WebhookMessage itself
// (the struct bin-contact-manager's subscribehandler unmarshals the
// internal conversation_message_created event into) carries case_id when
// present -- required so the internal event consumer can read the hint.
// This is distinct from the leak-prevention test above: that test
// constrains ConvertWebhookMessage's construction; this one constrains
// the wire shape used for decoding the internal event.
func Test_WebhookMessage_CaseID_JSONRoundTrip(t *testing.T) {
	caseID := uuid.FromStringOrNil("f1b2c3d4-000c-000c-000c-000000000003")

	wm := &WebhookMessage{CaseID: &caseID}
	data, err := json.Marshal(wm)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}

	var res WebhookMessage
	if err := json.Unmarshal(data, &res); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}
	if res.CaseID == nil || *res.CaseID != caseID {
		t.Fatalf("expected CaseID to round-trip. expect: %v, got: %v", caseID, res.CaseID)
	}
}
