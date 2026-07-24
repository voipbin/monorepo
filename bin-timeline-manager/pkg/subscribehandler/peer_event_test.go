package subscribehandler

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"

	call "monorepo/bin-call-manager/models/call"
	convconversation "monorepo/bin-conversation-manager/models/conversation"
	convmessage "monorepo/bin-conversation-manager/models/message"
)

func mustJSON(t *testing.T, v interface{}) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("could not marshal fixture: %v", err)
	}
	return b
}

func Test_buildPeerEventRows_CallEligible(t *testing.T) {
	customerID := uuid.Must(uuid.NewV4())
	callID := uuid.Must(uuid.NewV4())
	src := commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551110000"}
	dst := commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15552220000"}

	msg := call.WebhookMessage{
		Identity:    commonidentity.Identity{ID: callID, CustomerID: customerID},
		Source:      src,
		Destination: dst,
		Direction:   call.DirectionIncoming,
	}

	entries := []eventEntry{
		{
			event: &sock.Event{
				Publisher: string(commonoutline.ServiceNameCallManager),
				Type:      call.EventTypeCallHangup,
				Data:      mustJSON(t, msg),
			},
			receivedAt: time.Now(),
		},
	}

	rows := buildPeerEventRows(entries)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	r := rows[0]
	if r.Publisher != "call" {
		t.Errorf("Publisher = %q, want %q", r.Publisher, "call")
	}
	if r.ReferenceID != callID {
		t.Errorf("ReferenceID = %v, want %v", r.ReferenceID, callID)
	}
	// incoming: peer = source, local = destination
	if r.PeerType != string(commonaddress.TypeTel) || r.PeerTarget != src.Target {
		t.Errorf("Peer = (%q,%q), want (%q,%q)", r.PeerType, r.PeerTarget, commonaddress.TypeTel, src.Target)
	}
	if r.LocalType != string(commonaddress.TypeTel) || r.LocalTarget != dst.Target {
		t.Errorf("Local = (%q,%q), want (%q,%q)", r.LocalType, r.LocalTarget, commonaddress.TypeTel, dst.Target)
	}
}

func Test_buildPeerEventRows_ConversationMessageEligible(t *testing.T) {
	customerID := uuid.Must(uuid.NewV4())
	msgID := uuid.Must(uuid.NewV4())
	src := commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551110000"}
	dst := commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15552220000"}

	msg := convmessage.WebhookMessage{
		Identity:    commonidentity.Identity{ID: msgID, CustomerID: customerID},
		Source:      src,
		Destination: dst,
		Direction:   convmessage.DirectionOutgoing,
	}

	entries := []eventEntry{
		{
			event: &sock.Event{
				Publisher: string(commonoutline.ServiceNameConversationManager),
				Type:      convmessage.EventTypeMessageCreated,
				Data:      mustJSON(t, msg),
			},
			receivedAt: time.Now(),
		},
	}

	rows := buildPeerEventRows(entries)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	r := rows[0]
	if r.Publisher != "conversation_message" {
		t.Errorf("Publisher = %q, want %q", r.Publisher, "conversation_message")
	}
	// outgoing: peer = destination, local = source
	if r.PeerTarget != dst.Target {
		t.Errorf("PeerTarget = %q, want %q", r.PeerTarget, dst.Target)
	}
	if r.LocalTarget != src.Target {
		t.Errorf("LocalTarget = %q, want %q", r.LocalTarget, src.Target)
	}
}

func Test_buildPeerEventRows_ConversationParentEligible(t *testing.T) {
	customerID := uuid.Must(uuid.NewV4())
	convID := uuid.Must(uuid.NewV4())
	self := commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551110000"}
	peer := commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15552220000"}

	msg := convconversation.WebhookMessage{
		Identity: commonidentity.Identity{ID: convID, CustomerID: customerID},
		Self:     self,
		Peer:     peer,
	}

	entries := []eventEntry{
		{
			event: &sock.Event{
				Publisher: string(commonoutline.ServiceNameConversationManager),
				Type:      convconversation.EventTypeConversationCreated,
				Data:      mustJSON(t, msg),
			},
			receivedAt: time.Now(),
		},
	}

	rows := buildPeerEventRows(entries)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	r := rows[0]
	if r.Publisher != "conversation" {
		t.Errorf("Publisher = %q, want %q", r.Publisher, "conversation")
	}
	if r.Direction != "" {
		t.Errorf("Direction = %q, want empty (conversation parent has no direction)", r.Direction)
	}
	if r.PeerTarget != peer.Target {
		t.Errorf("PeerTarget = %q, want %q", r.PeerTarget, peer.Target)
	}
	if r.LocalTarget != self.Target {
		t.Errorf("LocalTarget = %q, want %q", r.LocalTarget, self.Target)
	}
}

// Test_buildPeerEventRows_AllowlistMiss verifies the silent-skip path: a
// (Publisher, EventType) pair NOT in eligiblePeerEvents produces no row
// (steady-state path, distinct from the malformed-payload path below).
func Test_buildPeerEventRows_AllowlistMiss(t *testing.T) {
	entries := []eventEntry{
		{
			event: &sock.Event{
				Publisher: string(commonoutline.ServiceNameCallManager),
				Type:      "groupcall_created", // not in the allowlist
				Data:      json.RawMessage(`{}`),
			},
			receivedAt: time.Now(),
		},
	}

	rows := buildPeerEventRows(entries)
	if len(rows) != 0 {
		t.Fatalf("expected 0 rows for allowlist-miss, got %d", len(rows))
	}
}

// Test_buildPeerEventRows_MalformedPayload verifies each of the three
// unmarshal branches (call, conversation_message, conversation) skips the
// row on malformed JSON rather than producing a zero-value garbage row.
func Test_buildPeerEventRows_MalformedPayload(t *testing.T) {
	tests := []struct {
		name      string
		publisher string
		eventType string
	}{
		{"call branch", string(commonoutline.ServiceNameCallManager), call.EventTypeCallCreated},
		{"conversation_message branch", string(commonoutline.ServiceNameConversationManager), convmessage.EventTypeMessageCreated},
		{"conversation branch", string(commonoutline.ServiceNameConversationManager), convconversation.EventTypeConversationCreated},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entries := []eventEntry{
				{
					event: &sock.Event{
						Publisher: tt.publisher,
						Type:      tt.eventType,
						Data:      json.RawMessage(`{not valid json`),
					},
					receivedAt: time.Now(),
				},
			}

			rows := buildPeerEventRows(entries)
			if len(rows) != 0 {
				t.Fatalf("expected 0 rows for malformed payload, got %d", len(rows))
			}
		})
	}
}

// Test_buildPeerEventRows_DirectionUnset verifies an eligible event with an
// unset/unknown Direction still produces a row (per the no-eligibility-
// filter decision), with zero-value peer/local fields (not skipped, not an
// error).
func Test_buildPeerEventRows_DirectionUnset(t *testing.T) {
	customerID := uuid.Must(uuid.NewV4())
	callID := uuid.Must(uuid.NewV4())

	msg := call.WebhookMessage{
		Identity:    commonidentity.Identity{ID: callID, CustomerID: customerID},
		Source:      commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551110000"},
		Destination: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15552220000"},
		Direction:   "", // unset
	}

	entries := []eventEntry{
		{
			event: &sock.Event{
				Publisher: string(commonoutline.ServiceNameCallManager),
				Type:      call.EventTypeCallCreated,
				Data:      mustJSON(t, msg),
			},
			receivedAt: time.Now(),
		},
	}

	rows := buildPeerEventRows(entries)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row (not skipped), got %d", len(rows))
	}
	r := rows[0]
	if r.PeerType != "" || r.PeerTarget != "" {
		t.Errorf("expected zero-value peer, got (%q,%q)", r.PeerType, r.PeerTarget)
	}
	if r.LocalType != "" || r.LocalTarget != "" {
		t.Errorf("expected zero-value local, got (%q,%q)", r.LocalType, r.LocalTarget)
	}
}
