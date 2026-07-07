package servicehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	commonaddress "monorepo/bin-common-handler/models/address"
	cmkase "monorepo/bin-contact-manager/models/kase"
	cvconversation "monorepo/bin-conversation-manager/models/conversation"
	cvmessage "monorepo/bin-conversation-manager/models/message"
	nmnumber "monorepo/bin-number-manager/models/number"
)

// caseMessagePeerTypeToConversationType maps a Case's PeerType (design
// §3.1, commonaddress.Type -- the vocabulary Case.PeerType is stored in)
// to the conversation Type ConversationV1ConversationGetOrCreateBySelfAndPeer
// expects (design §4.5 step 1 / round-12 correction). Only the address
// types that are actually reachable through a Case today are mapped;
// anything else (e.g. commonaddress.TypeAgent, TypeConference -- types
// that can never be a Case's PeerType per the design's channel scope)
// falls through to cvconversation.TypeMessage as the conservative default,
// since design §4.5 restricts this endpoint to genuinely message-capable
// channels in the first place.
func caseMessagePeerTypeToConversationType(peerType commonaddress.Type) cvconversation.Type {
	switch peerType {
	case commonaddress.TypeLine:
		return cvconversation.TypeLine
	case commonaddress.TypeWhatsApp:
		return cvconversation.TypeWhatsApp
	case commonaddress.TypeEmail:
		return cvconversation.TypeEmail
	default:
		// commonaddress.TypeTel and any other/unknown value default to
		// TypeMessage (sms/mms) -- the common case for a phone-number
		// peer_target.
		return cvconversation.TypeMessage
	}
}

// CaseMessageSend implements design §4.5: send a message from a known,
// open Case with an explicit source/destination, validated in the exact
// 6-step order specified by the design (case validation, destination-to-
// case binding, source-ownership validation, conversation resolution,
// fail-open metadata write, send). Each step's rationale/security
// invariant is documented inline at its call site below.
func (h *serviceHandler) CaseMessageSend(
	ctx context.Context,
	a *auth.AuthIdentity,
	caseID uuid.UUID,
	source string,
	destination string,
	text string,
) (*cvmessage.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CaseMessageSend",
		"customer_id": a.CustomerID,
		"case_id":     caseID,
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	// Step 1: case validation. caseGet tenant-checks automatically (a
	// cross-tenant caseID returns the same not-found error a genuinely
	// missing case would -- see case.go's caseGet doc comment).
	c, err := h.caseGet(ctx, a.CustomerID, caseID)
	if err != nil {
		log.Errorf("Could not get the case info. err: %v", err)
		return nil, err
	}

	// Permission check: design §4.5's 6-step validation sequence does not
	// enumerate an explicit permission-check step, but every other Case
	// endpoint in this package (case.go, case_note.go) gates on
	// PermissionCustomerAdmin|PermissionCustomerManager immediately after
	// caseGet, and skipping it here would let any authenticated agent of
	// this customer (not just admins/managers) send messages through
	// Cases -- inconsistent with the rest of the Case surface. Conservative
	// judgment call: apply the same gate here, before step 1's status
	// check, mirroring case.go's ordering.
	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	if c.Status != cmkase.StatusOpen {
		log.Infof("Case is not open, status: %s", c.Status)
		return nil, serviceerrors.ErrCaseClosed
	}

	// Step 2: destination-to-case binding (design §4.5's anti-oracle
	// property). Both sub-checks below MUST return the exact same
	// serviceerrors.ErrCaseDestinationNotAssociated sentinel on failure --
	// no branch-specific error or log message is allowed to leak which
	// branch failed via the returned error itself. (Debug logging here is
	// server-side only and never reaches the client; the client only ever
	// sees the single generic sentinel via error_translate.go.)
	if c.ContactID != nil {
		addresses, errAddr := h.reqHandler.ContactV1AddressGet(ctx, *c.ContactID)
		if errAddr != nil {
			log.Errorf("Could not get contact addresses. err: %v", errAddr)
			return nil, serviceerrors.ErrCaseDestinationNotAssociated
		}

		matched := false
		for _, addr := range addresses {
			if addr.Target == destination {
				matched = true
				break
			}
		}
		if !matched {
			return nil, serviceerrors.ErrCaseDestinationNotAssociated
		}
	} else {
		if destination != c.PeerTarget {
			return nil, serviceerrors.ErrCaseDestinationNotAssociated
		}
	}

	// Step 3: source-ownership validation (round-17 correction). Queries
	// against case.CustomerID (the case's own, already tenant-verified
	// customer_id) -- not a.CustomerID directly, though the two are
	// identical by this point since caseGet already tenant-checked the
	// case against a.CustomerID.
	numbers, err := h.reqHandler.NumberV1NumberList(ctx, "", 1, map[nmnumber.Field]any{
		nmnumber.FieldCustomerID: c.CustomerID,
		nmnumber.FieldNumber:     source,
		nmnumber.FieldType:       nmnumber.TypeNormal,
		nmnumber.FieldStatus:     nmnumber.StatusActive,
		nmnumber.FieldDeleted:    false,
	})
	if err != nil {
		log.Errorf("Could not list numbers. err: %v", err)
		return nil, err
	}
	if len(numbers) == 0 {
		return nil, serviceerrors.ErrCaseSourceNotOwned
	}

	// Step 4: resolve conversationID via get-or-create.
	selfAddr := commonaddress.Address{
		Type:   commonaddress.TypeTel,
		Target: source,
	}
	peerAddr := commonaddress.Address{
		Type:   c.PeerType,
		Target: destination,
	}
	conversationType := caseMessagePeerTypeToConversationType(c.PeerType)

	conv, err := h.reqHandler.ConversationV1ConversationGetOrCreateBySelfAndPeer(ctx, c.CustomerID, conversationType, "", selfAddr, peerAddr)
	if err != nil {
		log.Errorf("Could not get or create conversation. err: %v", err)
		return nil, err
	}

	// Step 5: metadata write, FAIL-OPEN. If this RPC errors, log it and
	// continue to step 6 regardless -- the message must still be sent.
	if _, errMeta := h.reqHandler.ConversationV1ConversationUpdateMetadata(ctx, conv.ID, cvconversation.Metadata{ContactCaseID: &caseID}); errMeta != nil {
		log.Errorf("Could not update conversation metadata, continuing to send anyway (fail-open). err: %v", errMeta)
	}

	// Step 6: send. Uses the existing private no-permission-check helper
	// directly -- permission for this endpoint was already verified
	// against the Case's own customer_id (via caseGet's tenant check
	// above), not the conversation's, so ConversationMessageSend's own
	// permission check (against the conversation's customer_id) would be
	// redundant/wrong here.
	msg, err := h.conversationMessageSend(ctx, a, conv.ID, text, nil)
	if err != nil {
		log.Errorf("Could not send the message. err: %v", err)
		return nil, err
	}

	return msg.ConvertWebhookMessage(), nil
}
