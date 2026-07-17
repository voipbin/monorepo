package conversationhandler

import (
	"context"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-conversation-manager/internal/convtitle"
	"monorepo/bin-conversation-manager/models/conversation"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"
)

// CreateAndExecuteFlow creates a brand-new Conversation and immediately
// Create+Executes an activeflow against it (ReferenceType=Conversation).
// Unlike GetOrCreateBySelfAndPeer, this deliberately always creates --
// no dedup lookup -- callers are expected to pass a self/peer pair that
// is unique per call (e.g. bin-webchat-manager passes a fresh
// Session.ID as peer on every session-create call, so the dedup key is
// unreachable by construction). See design doc
// 2026-07-17-webchat-widget-session-message-flow-split-design.md §3.3.
func (h *conversationHandler) CreateAndExecuteFlow(
	ctx context.Context,
	customerID uuid.UUID,
	flowID uuid.UUID,
	conversationType conversation.Type,
	dialogID string,
	self commonaddress.Address,
	peer commonaddress.Address,
) (*conversation.Conversation, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CreateAndExecuteFlow",
		"customer_id": customerID,
		"flow_id":     flowID,
		"self":        self,
		"peer":        peer,
	})
	log.Debug("Creating a new conversation and triggering its flow.")

	id := h.utilHandler.UUIDCreate()

	// Canonicalize self/peer through the shared address normalization
	// authority, matching Create/GetOrCreateBySelfAndPeer exactly --
	// this guarantees a later GetOrCreateBySelfAndPeer call (e.g.
	// conversation-manager's own async webchat_message_created
	// subscriber) resolves to a GET against THIS row, not a second
	// CREATE.
	self.Target, _ = commonaddress.NormalizeTarget(self.Type, self.Target)
	peer.Target, _ = commonaddress.NormalizeTarget(peer.Type, peer.Target)

	name, detail := convtitle.Build(conversationType, peer)

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

	if flowID == uuid.Nil {
		log.Debug("No flow configured. Skipping activeflow.")
		return res, nil
	}

	af, errCreate := h.reqHandler.FlowV1ActiveflowCreate(
		ctx,
		uuid.Nil,
		customerID,
		flowID,
		fmactiveflow.ReferenceTypeConversation,
		res.ID,
		uuid.Nil,
		nil,
		"",
		fmactiveflow.WebhookMethodNone,
	)
	if errCreate != nil {
		// Best-effort: the Conversation itself was already created
		// successfully. A Flow-trigger failure must not fail the
		// visitor-facing session-creation response (mirrors
		// bin-webchat-manager's own triggerFirstMessageFlow's
		// best-effort framing).
		log.Errorf("Could not create activeflow. flow_id: %s, err: %v", flowID, errors.Wrapf(errCreate, "could not create activeflow"))
		return res, nil
	}
	log.WithField("activeflow_id", af.ID).Debug("Created activeflow.")

	if errExecute := h.reqHandler.FlowV1ActiveflowExecute(ctx, af.ID); errExecute != nil {
		log.Errorf("Could not execute activeflow. activeflow_id: %s, err: %v", af.ID, errExecute)
		return res, nil
	}
	log.Debug("Executed activeflow.")

	return res, nil
}
