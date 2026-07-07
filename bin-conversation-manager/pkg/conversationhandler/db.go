package conversationhandler

import (
	"context"
	stderrors "errors"
	"fmt"

	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-conversation-manager/internal/convtitle"
	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/pkg/dbhandler"
)

// Get returns conversation.
//
// When the underlying DB layer returns dbhandler.ErrNotFound, Get returns a
// typed *cerrors.VoipbinError (Status=NotFound) so the api-manager edge can
// recover the upstream domain/reason via errors.As.
func (h *conversationHandler) Get(ctx context.Context, id uuid.UUID) (*conversation.Conversation, error) {
	res, err := h.db.ConversationGet(ctx, id)
	if err != nil {
		if stderrors.Is(err, dbhandler.ErrNotFound) {
			return nil, cerrors.NotFound(
				commonoutline.ServiceNameConversationManager,
				"CONVERSATION_NOT_FOUND",
				"The conversation was not found.",
			).Wrap(err)
		}
		return nil, err
	}
	return res, nil
}

// GetBySelfAndPeer is a get-only lookup (deliberately never creates).
// Used by bin-contact-manager's proactive Case-linking write path
// (contact-case-management design §4.4, round-7 correction): a miss
// must not create a Conversation, since doing so purely as case-linking
// plumbing -- with no message ever having been sent -- would fire a
// genuine, real, customer-facing conversation_created webhook for a
// thread that doesn't actually exist yet from the customer's
// perspective. Contrast with GetOrCreateBySelfAndPeer below, which is
// correct to create on miss because it is only ever called when a real
// message is genuinely about to be sent (§4.5).
func (h *conversationHandler) GetBySelfAndPeer(ctx context.Context, self commonaddress.Address, peer commonaddress.Address) (*conversation.Conversation, error) {
	// Canonicalize self/peer BEFORE the lookup so the lookup key and the
	// stored value share one canonical form, matching
	// GetOrCreateBySelfAndPeer's normalization. NormalizeTarget is
	// loss-proof, so the error is discarded.
	self.Target, _ = commonaddress.NormalizeTarget(self.Type, self.Target)
	peer.Target, _ = commonaddress.NormalizeTarget(peer.Type, peer.Target)

	res, err := h.db.ConversationGetBySelfAndPeer(ctx, self, peer)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// GetBySelfAndPeer returns conversation
func (h *conversationHandler) GetOrCreateBySelfAndPeer(
	ctx context.Context,
	customerID uuid.UUID,
	conversationType conversation.Type,
	dialogID string,
	self commonaddress.Address,
	peer commonaddress.Address,
) (*conversation.Conversation, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "GetOrCreateBySelfAndPeer",
		"self": self,
		"peer": peer,
	})

	// Canonicalize self/peer BEFORE the dedup lookup so the lookup key and the
	// stored value share one canonical form. NormalizeTarget is loss-proof, so
	// the error is discarded. DialogID is the provider wire/dedup key and is NOT
	// normalized.
	self.Target, _ = commonaddress.NormalizeTarget(self.Type, self.Target)
	peer.Target, _ = commonaddress.NormalizeTarget(peer.Type, peer.Target)

	res, err := h.db.ConversationGetBySelfAndPeer(ctx, self, peer)
	if err != nil {
		log.Debugf("Could not find conversation. Create a new conversation. err: %v", err)

		name, detail := convtitle.Build(conversationType, peer)
		res, err = h.Create(
			ctx,
			customerID,
			name,
			detail,
			conversationType,
			dialogID,
			self,
			peer,
		)
		if err != nil {
			return nil, errors.Wrapf(err, "Could not create a new conversation")
		}
		log.WithField("conversation", res).Debugf("Created a new conversation. conversation_id: %s", res.ID)
	}

	return res, nil
}

// List returns list of conversations
func (h *conversationHandler) List(ctx context.Context, pageToken string, pageSize uint64, filters map[conversation.Field]any) ([]*conversation.Conversation, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "List",
		"filers": filters,
	})
	log.Debugf("Getting a list of conversations.")

	res, err := h.db.ConversationList(ctx, pageSize, pageToken, filters)
	if err != nil {
		log.Errorf("Could not get conversations. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Create creates a new conversation and return a created conversation.
func (h *conversationHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,
	name string,
	detail string,
	conversationType conversation.Type,
	dialogID string,
	self commonaddress.Address,
	peer commonaddress.Address,
) (*conversation.Conversation, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "Create",
	})

	id := h.utilHandler.UUIDCreate()

	// Canonicalize self/peer through the shared address normalization authority.
	// Idempotent, so callers that already normalized (GetOrCreateBySelfAndPeer)
	// pass through unchanged. Loss-proof, so the error is discarded. DialogID is
	// the provider wire/dedup key and is NOT normalized.
	self.Target, _ = commonaddress.NormalizeTarget(self.Type, self.Target)
	peer.Target, _ = commonaddress.NormalizeTarget(peer.Type, peer.Target)

	tmp := &conversation.Conversation{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
		Owner: commonidentity.Owner{
			OwnerType: commonidentity.OwnerTypeNone,
			OwnerID:   uuid.Nil,
		},

		Name:     name,
		Detail:   detail,
		Type:     conversationType,
		DialogID: dialogID,
		Self:     self,
		Peer:     peer,
	}

	if errCreate := h.db.ConversationCreate(ctx, tmp); errCreate != nil {
		log.Errorf("Could not create conversation. err: %v", errCreate)
		return nil, errCreate
	}

	res, err := h.db.ConversationGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get created conversation. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, conversation.EventTypeConversationCreated, res)

	return res, nil
}

// Update updates conversation and return a updated conversation.
//
// When owner_id is present in the partial-update fields, the server derives
// owner_type from the owner_id value (clients never send owner_type directly):
//   - owner_id == uuid.Nil  → owner_type = "" (OwnerTypeNone, unassigned)
//   - owner_id != uuid.Nil  → owner_type = "agent" (OwnerTypeAgent, the only
//     valid type today)
//
// Caller-supplied owner_type is silently overridden by the derived value.
func (h *conversationHandler) Update(ctx context.Context, id uuid.UUID, fields map[conversation.Field]any) (*conversation.Conversation, error) {
	// Avoid logging field values: the partial-update map can carry owner UUIDs
	// and other caller-controlled values. Log the field names only as a
	// defense-in-depth measure against accidental disclosure in structured
	// log streams.
	fieldNames := make([]string, 0, len(fields))
	for k := range fields {
		fieldNames = append(fieldNames, string(k))
	}
	log := logrus.WithFields(logrus.Fields{
		"func":        "Update",
		"id":          id,
		"field_names": fieldNames,
	})
	log.Debugf("Updating conversation. conversation_id: %s", id)

	if v, ok := fields[conversation.FieldOwnerID]; ok {
		ownerID, okType := v.(uuid.UUID)
		if !okType {
			return nil, cerrors.InvalidArgument(
				commonoutline.ServiceNameConversationManager,
				"INVALID_OWNER_ID_TYPE",
				fmt.Sprintf("invalid owner_id type: %T", v),
			)
		}
		if ownerID != uuid.Nil {
			// Need cv.CustomerID for the same-customer constraint — fetch the
			// existing conversation. Update did not previously load it.
			cv, errGet := h.Get(ctx, id)
			if errGet != nil {
				return nil, errors.Wrapf(errGet, "could not load conversation for validation. id: %s", id)
			}

			// Validate agent existence and the same-customer constraint via
			// per-id Get. Two distinct error reasons surface through the
			// typed VoipbinError envelope so the api-manager edge can
			// distinguish them in 400 response bodies (design §5.4).
			//
			// NOTE: bin-agent-manager collapses ErrNotFound to HTTP 500 over
			// RPC today, so a real not-found and a transport error look the
			// same here. We treat any AgentV1AgentGet error as not-found per
			// the design's canonical wording (the dominant case); operators
			// can correlate transport errors via the wrapped underlying error.
			ag, errGetAgent := h.reqHandler.AgentV1AgentGet(ctx, ownerID)
			if errGetAgent != nil {
				return nil, cerrors.InvalidArgument(
					commonoutline.ServiceNameConversationManager,
					"AGENT_NOT_FOUND",
					fmt.Sprintf("agent not found. owner_id: %s", ownerID),
				).Wrap(errGetAgent)
			}
			if ag.CustomerID != cv.CustomerID {
				// Deviation from design §5.4 verbatim wording: do NOT include the agent's
				// CustomerID or the conversation's CustomerID in the user-facing message.
				// Returning agent_customer_id would leak a cross-tenant customer UUID to the
				// caller — even an admin/manager who is authorized for THIS conversation has
				// no need to know the customer of an unrelated agent. The reason code
				// AGENT_CUSTOMER_MISMATCH still distinguishes this case from AGENT_NOT_FOUND
				// for diagnostic purposes; operators can correlate via server-side logs which
				// retain the full triple (id is logged in the structured fields).
				log.WithFields(logrus.Fields{
					"owner_id":               ownerID,
					"agent_customer_id":      ag.CustomerID,
					"conversation_customer_id": cv.CustomerID,
				}).Info("Agent customer mismatch on assignment.")
				return nil, cerrors.InvalidArgument(
					commonoutline.ServiceNameConversationManager,
					"AGENT_CUSTOMER_MISMATCH",
					fmt.Sprintf("agent customer mismatch. owner_id: %s", ownerID),
				)
			}
		}

		if ownerID == uuid.Nil {
			fields[conversation.FieldOwnerType] = commonidentity.OwnerTypeNone
		} else {
			fields[conversation.FieldOwnerType] = commonidentity.OwnerTypeAgent
		}
	}

	if errUpdate := h.db.ConversationUpdate(ctx, id, fields); errUpdate != nil {
		return nil, errors.Wrapf(errUpdate, "could not update conversation. id: %s", id)
	}

	res, err := h.db.ConversationGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get updated conversation. id: %s", id)
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, conversation.EventTypeConversationUpdated, res)

	return res, nil
}
