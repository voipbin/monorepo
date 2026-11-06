package message

import (
	"encoding/json"
	"strings"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"

	"monorepo/bin-message-manager/models/target"
)

func internalMessage() *Message {
	return &Message{
		Type:                TypeMMS,
		ProviderName:        ProviderNameTelnyx,
		ProviderReferenceID: "message-level-ref",
		Targets: []target.Target{
			{
				Destination:         commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550100002"},
				Status:              target.StatusSent,
				Parts:               3,
				ProviderParts:       2,
				ProviderReferenceID: "4031a1af-0000-4000-8000-0000000000aa",
			},
		},
	}
}

func Test_EventCopy(t *testing.T) {
	m := internalMessage()

	cp := m.EventCopy()

	if cp == m || &cp.Targets[0] == &m.Targets[0] {
		t.Errorf("The copy must not share the message or the targets slice.")
	}
	if cp.Targets[0].ProviderReferenceID != "" || cp.Targets[0].ProviderParts != 0 || cp.ProviderReferenceID != "" {
		t.Errorf("The provider keys must be cleared in the copy. got: %v", cp)
	}
	if cp.Targets[0].Parts != 3 || cp.Targets[0].Status != target.StatusSent {
		t.Errorf("The public fields must be kept. got: %v", cp.Targets[0])
	}
	if m.Targets[0].ProviderReferenceID == "" || m.Targets[0].ProviderParts != 2 || m.ProviderReferenceID == "" {
		t.Errorf("The original must keep the provider keys. got: %v", m)
	}

	raw, err := json.Marshal(cp)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "4031a1af") || strings.Contains(string(raw), "provider_parts") || strings.Contains(string(raw), "message-level-ref") {
		t.Errorf("The published json leaks a provider key: %s", raw)
	}

	if (&Message{}).EventCopy().Targets != nil {
		t.Errorf("A nil targets slice must stay nil")
	}
}

// Test_WebhookMessage_noProviderKeys is the golden test of the public shape: the API and the customer webhooks never
// carry the provider keys of a target.
func Test_WebhookMessage_noProviderKeys(t *testing.T) {
	raw, err := json.Marshal(internalMessage().ConvertWebhookMessage())
	if err != nil {
		t.Fatal(err)
	}

	for _, key := range []string{"provider_parts", "provider_reference_id", "4031a1af", "message-level-ref"} {
		if strings.Contains(string(raw), key) {
			t.Errorf("The webhook json contains %q: %s", key, raw)
		}
	}

	var decoded struct {
		Targets []map[string]any `json:"targets"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	keys := map[string]bool{}
	for k := range decoded.Targets[0] {
		keys[k] = true
	}
	for _, k := range []string{"destination", "status", "parts", "tm_update"} {
		if !keys[k] {
			t.Errorf("The target is missing the key %q. keys: %v", k, keys)
		}
	}
	if len(keys) != 4 {
		t.Errorf("The target has other keys than the public four. keys: %v", keys)
	}
}

// Test_Target_oldFormat checks that a stored target without the new keys is read with zero values, and that the new
// keys are omitted when empty, so old code and new code read each other's rows.
func Test_Target_oldFormat(t *testing.T) {
	var old target.Target
	if err := json.Unmarshal([]byte(`{"destination":{"type":"tel","target":"+15550100002"},"status":"sent","parts":2,"tm_update":null}`), &old); err != nil {
		t.Fatal(err)
	}
	if old.ProviderParts != 0 || old.ProviderReferenceID != "" || old.Parts != 2 {
		t.Errorf("Wrong old target. got: %v", old)
	}

	raw, _ := json.Marshal(target.Target{Parts: 1})
	if strings.Contains(string(raw), "provider_") {
		t.Errorf("Empty provider keys must be omitted. got: %s", raw)
	}
}
