package webhookhandler

import (
	"encoding/json"
	"reflect"
	"sort"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

func TestParseWebhookOwnerData(t *testing.T) {
	data := json.RawMessage(`{"customer_id":"a1b2c3d4-0000-0000-0000-000000000001","owner_id":"98765432-0000-0000-0000-000000000002","owner_type":"agent","id":"12345678-0000-0000-0000-000000000003"}`)

	d, err := parseWebhookOwnerData(data)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if d.CustomerID.String() != "a1b2c3d4-0000-0000-0000-000000000001" {
		t.Errorf("Expected CustomerID 'a1b2c3d4-0000-0000-0000-000000000001', got '%s'", d.CustomerID.String())
	}
	if d.OwnerID.String() != "98765432-0000-0000-0000-000000000002" {
		t.Errorf("Expected OwnerID '98765432-0000-0000-0000-000000000002', got '%s'", d.OwnerID.String())
	}
}

// TestParseWebhookOwnerData_ChatFields verifies AIcallID/ChatID are extracted correctly when
// present -- these fields were untested in the initial commit (code-quality review gap).
func TestParseWebhookOwnerData_ChatFields(t *testing.T) {
	data := json.RawMessage(`{"customer_id":"a1b2c3d4-0000-0000-0000-000000000001","aicall_id":"a1ca11ff-0000-0000-0000-000000000004","chat_id":"c4a70000-0000-0000-0000-000000000005"}`)

	d, err := parseWebhookOwnerData(data)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if d.AIcallID.String() != "a1ca11ff-0000-0000-0000-000000000004" {
		t.Errorf("Expected AIcallID 'a1ca11ff-0000-0000-0000-000000000004', got '%s'", d.AIcallID.String())
	}
	if d.ChatID.String() != "c4a70000-0000-0000-0000-000000000005" {
		t.Errorf("Expected ChatID 'c4a70000-0000-0000-0000-000000000005', got '%s'", d.ChatID.String())
	}
}

// TestParseWebhookOwnerData_AbsentOptionalFields verifies the "best-effort tolerance for
// absent fields" claim in parseWebhookOwnerData's doc comment: a payload missing owner_id/
// aicall_id/chat_id entirely must NOT error, and those fields must stay at their zero value
// (uuid.Nil) rather than causing an unmarshal failure.
func TestParseWebhookOwnerData_AbsentOptionalFields(t *testing.T) {
	data := json.RawMessage(`{"customer_id":"a1b2c3d4-0000-0000-0000-000000000001"}`)

	d, err := parseWebhookOwnerData(data)
	if err != nil {
		t.Fatalf("Expected no error for a payload with absent optional fields, got %v", err)
	}

	if d.CustomerID.String() != "a1b2c3d4-0000-0000-0000-000000000001" {
		t.Errorf("Expected CustomerID to still be extracted, got '%s'", d.CustomerID.String())
	}
	if d.OwnerID != uuid.Nil {
		t.Errorf("Expected OwnerID to be zero-value (uuid.Nil) when absent, got '%s'", d.OwnerID.String())
	}
	if d.AIcallID != uuid.Nil {
		t.Errorf("Expected AIcallID to be zero-value (uuid.Nil) when absent, got '%s'", d.AIcallID.String())
	}
	if d.ChatID != uuid.Nil {
		t.Errorf("Expected ChatID to be zero-value (uuid.Nil) when absent, got '%s'", d.ChatID.String())
	}
}

// TestParseWebhookOwnerData_MalformedJSON verifies the error path: malformed JSON must return
// a non-nil error (the primary behavior parseWebhookOwnerData's signature promises), rather
// than silently succeeding with a zero-value struct.
func TestParseWebhookOwnerData_MalformedJSON(t *testing.T) {
	data := json.RawMessage(`{"customer_id": this is not valid json`)

	d, err := parseWebhookOwnerData(data)
	if err == nil {
		t.Fatal("Expected an error for malformed JSON, got nil")
	}
	if d != nil {
		t.Errorf("Expected nil result on error, got %+v", d)
	}
}

func TestCreateRoutingKeys_CustomerOnly(t *testing.T) {
	d := &webhookOwnerData{
		Identity: commonidentity.Identity{
			CustomerID: uuid.FromStringOrNil("a1b2c3d4-0000-0000-0000-000000000001"),
			ID:         uuid.FromStringOrNil("12345678-0000-0000-0000-000000000003"),
		},
	}

	keys := createRoutingKeys(d, "call", "call_updated")

	expected := []string{"customer_id.a1b2c3d4-0000-0000-0000-000000000001.call.call_updated.12345678-0000-0000-0000-000000000003"}
	if !reflect.DeepEqual(keys, expected) {
		t.Errorf("Expected %v, got %v", expected, keys)
	}
}

func TestCreateRoutingKeys_CustomerAndOwner(t *testing.T) {
	d := &webhookOwnerData{
		Identity: commonidentity.Identity{
			CustomerID: uuid.FromStringOrNil("a1b2c3d4-0000-0000-0000-000000000001"),
			ID:         uuid.FromStringOrNil("12345678-0000-0000-0000-000000000003"),
		},
		Owner: commonidentity.Owner{
			OwnerID: uuid.FromStringOrNil("98765432-0000-0000-0000-000000000002"),
		},
	}

	keys := createRoutingKeys(d, "queue", "queue_updated")

	expected := []string{
		"customer_id.a1b2c3d4-0000-0000-0000-000000000001.queue.queue_updated.12345678-0000-0000-0000-000000000003",
		"agent_id.98765432-0000-0000-0000-000000000002.queue.queue_updated.12345678-0000-0000-0000-000000000003",
	}
	sort.Strings(keys)
	sort.Strings(expected)
	if !reflect.DeepEqual(keys, expected) {
		t.Errorf("Expected %v, got %v", expected, keys)
	}
}

// TestCreateRoutingKeys_NoCustomerNoOwner verifies that when neither CustomerID nor OwnerID
// is present (uuid.Nil), createRoutingKeys returns an empty slice rather than a key with a
// nil UUID string.
func TestCreateRoutingKeys_NoCustomerNoOwner(t *testing.T) {
	d := &webhookOwnerData{}

	keys := createRoutingKeys(d, "call", "call_updated")

	if len(keys) != 0 {
		t.Errorf("Expected empty keys, got %v", keys)
	}
}
