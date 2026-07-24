package subscribehandler

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonoutline "monorepo/bin-common-handler/models/outline"

	call "monorepo/bin-call-manager/models/call"
	convconversation "monorepo/bin-conversation-manager/models/conversation"
	convmessage "monorepo/bin-conversation-manager/models/message"

	"monorepo/bin-timeline-manager/pkg/dbhandler"
)

// eligiblePeerEvents is the exhaustive (Publisher, EventType) allowlist for
// projection into peer_events. Any (Publisher, EventType) pair not in this
// set is left in `events` only — never attempted for peer/local derivation.
//
// Explicitly EXCLUDED (same publisher, incompatible or non-webhook-message
// payload): call-manager's groupcall_*, recording_*, confbridge_* events
// (different struct shape — groupcall.WebhookMessage has a pointer Source
// and a plural Destinations, no Direction field at all), dtmf_received
// (dtmf.DTMF carries only CallID/Digit/Duration), and
// call.outbound_whitelist_rejected (publishes a raw map[string]interface{},
// not call.WebhookMessage); conversation-manager's account_created/
// account_updated/account_deleted (account.WebhookMessage carries no
// Source/Destination/Self/Peer). See the design doc for full details:
// bin-timeline-manager/docs/plans/2026-07-24-add-peer-events-table-design.md
var eligiblePeerEvents = map[string]map[string]struct{}{
	string(commonoutline.ServiceNameCallManager): {
		call.EventTypeCallCreated:     {},
		call.EventTypeCallUpdated:     {},
		call.EventTypeCallDeleted:     {},
		call.EventTypeCallDialing:     {},
		call.EventTypeCallRinging:     {},
		call.EventTypeCallProgressing: {},
		call.EventTypeCallTerminating: {},
		call.EventTypeCallCanceling:   {},
		call.EventTypeCallHangup:      {},
	},
	string(commonoutline.ServiceNameConversationManager): {
		convmessage.EventTypeMessageCreated:           {}, // message.WebhookMessage shape
		convmessage.EventTypeMessageUpdated:           {},
		convmessage.EventTypeMessageDeleted:           {},
		convconversation.EventTypeConversationCreated: {}, // conversation.WebhookMessage shape (different struct!)
		convconversation.EventTypeConversationUpdated: {},
		convconversation.EventTypeConversationDeleted: {},
	},
}

// buildPeerEventRows converts each eligible entry into a PeerEventRow. The
// EventType (already matched against eligiblePeerEvents) determines both
// which WebhookMessage shape to unmarshal into and the synthetic
// PeerEventRow.Publisher label — event.Publisher alone cannot distinguish
// conversation_message_* from conversation_* (both carry the identical raw
// "conversation-manager" wire value).
func buildPeerEventRows(entries []eventEntry) []dbhandler.PeerEventRow {
	log := logrus.WithField("func", "buildPeerEventRows")

	var rows []dbhandler.PeerEventRow
	for _, e := range entries {
		types, ok := eligiblePeerEvents[e.event.Publisher]
		if !ok {
			continue
		}
		if _, ok := types[e.event.Type]; !ok {
			continue
		}

		switch e.event.Publisher {
		case string(commonoutline.ServiceNameCallManager):
			var m call.WebhookMessage
			if err := json.Unmarshal(e.event.Data, &m); err != nil {
				log.Warnf("Could not unmarshal call webhook for peer_events. err: %v", err)
				continue
			}
			peer, local := commonaddress.DeriveEndpoints(string(m.Direction), m.Source, m.Destination)
			rows = append(rows, newPeerEventRow(e.receivedAt, m.CustomerID, "call", e.event.Type, m.ID, string(m.Direction), peer, local, e.event.Data))

		case string(commonoutline.ServiceNameConversationManager):
			switch {
			case strings.HasPrefix(e.event.Type, "conversation_message_"):
				var m convmessage.WebhookMessage
				if err := json.Unmarshal(e.event.Data, &m); err != nil {
					log.Warnf("Could not unmarshal conversation message webhook for peer_events. err: %v", err)
					continue
				}
				peer, local := commonaddress.DeriveEndpoints(string(m.Direction), m.Source, m.Destination)
				rows = append(rows, newPeerEventRow(e.receivedAt, m.CustomerID, "conversation_message", e.event.Type, m.ID, string(m.Direction), peer, local, e.event.Data))
			default: // "conversation_created" / "_updated" / "_deleted"
				var m convconversation.WebhookMessage
				if err := json.Unmarshal(e.event.Data, &m); err != nil {
					log.Warnf("Could not unmarshal conversation webhook for peer_events. err: %v", err)
					continue
				}
				rows = append(rows, newPeerEventRow(e.receivedAt, m.CustomerID, "conversation", e.event.Type, m.ID, "", m.Peer, m.Self, e.event.Data))
			}
		}
	}
	return rows
}

// newPeerEventRow serializes peer/local into their JSON (response-facing)
// form AND flattens Type/Target into the internal-only search columns, in
// one place -- the Go-level equivalent of contact_interactions' MySQL
// STORED GENERATED COLUMN (see PeerEventRow's doc comment). If JSON
// marshaling somehow fails (it practically never does for this struct),
// the row is still inserted with an empty peer/local JSON string rather
// than being dropped -- the flat search columns are unaffected either way.
func newPeerEventRow(
	timestamp time.Time,
	customerID uuid.UUID,
	publisher, eventType string,
	referenceID uuid.UUID,
	direction string,
	peer, local commonaddress.Address,
	data json.RawMessage,
) dbhandler.PeerEventRow {
	peerJSON, err := json.Marshal(peer)
	if err != nil {
		logrus.WithField("func", "newPeerEventRow").Warnf("Could not marshal peer address. err: %v", err)
		peerJSON = []byte{}
	}
	localJSON, err := json.Marshal(local)
	if err != nil {
		logrus.WithField("func", "newPeerEventRow").Warnf("Could not marshal local address. err: %v", err)
		localJSON = []byte{}
	}

	return dbhandler.PeerEventRow{
		Timestamp:   timestamp,
		CustomerID:  customerID,
		Publisher:   publisher,
		EventType:   eventType,
		ReferenceID: referenceID,
		Direction:   direction,
		Peer:        string(peerJSON),
		Local:       string(localJSON),
		PeerType:    string(peer.Type),
		PeerTarget:  peer.Target,
		LocalType:   string(local.Type),
		LocalTarget: local.Target,
		Data:        string(data),
	}
}
