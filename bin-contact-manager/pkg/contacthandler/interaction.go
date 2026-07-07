package contacthandler

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	commonaddress "monorepo/bin-common-handler/models/address"

	call "monorepo/bin-call-manager/models/call"
	message "monorepo/bin-conversation-manager/models/message"

	"monorepo/bin-contact-manager/models/interaction"
)

// deriveEndpoints resolves which address is the remote peer and which is our
// local endpoint based on the call/message direction.
//
//   - incoming: the remote party is the source (they called/wrote us)
//   - outgoing: the remote party is the destination (we called/wrote them)
//   - unknown:  both are zero values; the caller should still persist the row
//     (the direction column itself carries the raw value for diagnostics).
func deriveEndpoints(direction string, source, dest commonaddress.Address) (peer commonaddress.Address, local commonaddress.Address) {
	switch direction {
	case "incoming":
		return source, dest
	case "outgoing":
		return dest, source
	default:
		return commonaddress.Address{}, commonaddress.Address{}
	}
}

// crmIneligiblePeerTypes lists address types that never represent a
// re-identifiable external contact (customer). Interactions whose peer
// resolves to one of these types are internal-resource noise (agent
// extensions, conference legs, AI resources, PSTN trunk direct-SIP legs,
// etc.) and must not be projected into the CRM interaction timeline: they
// can never be matched to a contact_address, so they would otherwise sit in
// the unresolved queue forever.
//
// Note: PSTN trunk partner calls are NOT affected by excluding TypeSIP here.
// bin-call-manager normalizes trunk-domain and SIP-domain incoming/outgoing
// call endpoints to TypeTel before publishing the webhook event (see
// AddressGetSource/AddressGetDestination call sites in
// bin-call-manager/pkg/callhandler), so a real external SIP trunk partner is
// always observed here as peer_type=tel, never peer_type=sip.
var crmIneligiblePeerTypes = map[commonaddress.Type]struct{}{
	commonaddress.TypeAgent:      {},
	commonaddress.TypeAI:         {},
	commonaddress.TypeAITeam:     {},
	commonaddress.TypeConference: {},
	commonaddress.TypeExtension:  {},
	commonaddress.TypeSIP:        {},
	"web_session":                {}, // synthetic type; not in commonaddress.Type enum
}

// isCRMEligiblePeer reports whether the given peer address type can ever
// represent an external, re-identifiable contact. Interactions with an
// ineligible peer type are dropped at projection time (never persisted),
// not merely filtered out of the unresolved queue at read time, since they
// can never legitimately resolve to a contact.
func isCRMEligiblePeer(peerType commonaddress.Type) bool {
	_, ineligible := crmIneligiblePeerTypes[peerType]
	return !ineligible
}

// EventCallCreated projects a call-created webhook event into the CRM
// interaction timeline.
func (h *contactHandler) EventCallCreated(ctx context.Context, m *call.WebhookMessage) error {
	log := logrus.WithFields(logrus.Fields{
		"func":      "EventCallCreated",
		"call_id":   m.ID,
		"direction": m.Direction,
	})

	peer, local := deriveEndpoints(string(m.Direction), m.Source, m.Destination)

	if !isCRMEligiblePeer(peer.Type) {
		log.Debugf("Peer type is not CRM-eligible; skipping interaction projection. peer_type: %s", peer.Type)
		return nil
	}

	peerTarget, err := commonaddress.NormalizeTarget(peer.Type, peer.Target)
	if err != nil {
		log.WithError(err).Warnf("could not normalize peer target; storing raw value. peer_type: %s peer_target: %s", peer.Type, peer.Target)
		peerTarget = peer.Target
	}

	localTarget, err := commonaddress.NormalizeTarget(local.Type, local.Target)
	if err != nil {
		log.WithError(err).Warnf("could not normalize local target; storing raw value. local_type: %s local_target: %s", local.Type, local.Target)
		localTarget = local.Target
	}

	id := h.utilHandler.UUIDCreate()
	now := h.utilHandler.TimeNow()

	c, err := h.caseHandler.GetOrCreate(ctx, m.CustomerID, peer.Type, peerTarget, "call", nil)
	if err != nil {
		return fmt.Errorf("could not get-or-create case. EventCallCreated. err: %v", err)
	}

	i := interaction.Interaction{
		ID:            id,
		CustomerID:    m.CustomerID,
		Direction:     string(m.Direction),
		PeerType:      string(peer.Type),
		PeerTarget:    peerTarget,
		LocalType:     string(local.Type),
		LocalTarget:   localTarget,
		ReferenceType: "call",
		ReferenceID:   m.ID,
		CaseID:        &c.ID,
		TMInteraction: m.TMCreate,
		TMCreate:      now,
	}

	return h.db.InteractionCreate(ctx, &i)
}

// EventConversationMessageCreated projects a conversation-message-created
// webhook event into the CRM interaction timeline.
func (h *contactHandler) EventConversationMessageCreated(ctx context.Context, m *message.WebhookMessage) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "EventConversationMessageCreated",
		"message_id": m.ID,
		"direction":  m.Direction,
	})

	peer, local := deriveEndpoints(string(m.Direction), m.Source, m.Destination)

	if !isCRMEligiblePeer(peer.Type) {
		log.Debugf("Peer type is not CRM-eligible; skipping interaction projection. peer_type: %s", peer.Type)
		return nil
	}

	peerTarget, err := commonaddress.NormalizeTarget(peer.Type, peer.Target)
	if err != nil {
		log.WithError(err).Warnf("could not normalize peer target; storing raw value. peer_type: %s peer_target: %s", peer.Type, peer.Target)
		peerTarget = peer.Target
	}

	localTarget, err := commonaddress.NormalizeTarget(local.Type, local.Target)
	if err != nil {
		log.WithError(err).Warnf("could not normalize local target; storing raw value. local_type: %s local_target: %s", local.Type, local.Target)
		localTarget = local.Target
	}

	id := h.utilHandler.UUIDCreate()
	now := h.utilHandler.TimeNow()

	c, err := h.caseHandler.GetOrCreate(ctx, m.CustomerID, peer.Type, peerTarget, "conversation_message", nil)
	if err != nil {
		return fmt.Errorf("could not get-or-create case. EventConversationMessageCreated. err: %v", err)
	}

	// m.ID comes from the embedded commonidentity.Identity; use it directly.
	i := interaction.Interaction{
		ID:            id,
		CustomerID:    m.CustomerID,
		Direction:     string(m.Direction),
		PeerType:      string(peer.Type),
		PeerTarget:    peerTarget,
		LocalType:     string(local.Type),
		LocalTarget:   localTarget,
		ReferenceType: "conversation_message",
		ReferenceID:   m.ID,
		CaseID:        &c.ID,
		TMInteraction: m.TMCreate,
		TMCreate:      now,
	}

	return h.db.InteractionCreate(ctx, &i)
}
