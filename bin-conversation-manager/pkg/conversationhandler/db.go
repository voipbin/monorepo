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

	res, err := h.db.ConversationGetBySelfAndPeer(ctx, self, peer)
	if err != nil {
		log.Debugf("Could not find conversation. Create a new conversation. err: %v", err)

		res, err = h.Create(
			ctx,
			customerID,
			"conversation with "+peer.TargetName,
			"conversation with "+peer.TargetName,
			conversation.TypeMessage,
			dialogID, // because it's sms conversation, there is no dialog id
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
	log := logrus.WithFields(logrus.Fields{
		"func":   "Update",
		"id":     id,
		"fields": fields,
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
		if ownerID == uuid.Nil {
			fields[conversation.FieldOwnerType] = commonidentity.OwnerTypeNone
		} else {
			fields[conversation.FieldOwnerType] = commonidentity.OwnerTypeAgent
		}
	}

	if errUpdate := h.db.ConversationUpdate(ctx, id, fields); errUpdate != nil {
		return nil, errors.Wrapf(errUpdate, "Could not update conversation. err: %v", errUpdate)
	}

	res, err := h.db.ConversationGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not get updated conversation. err: %v", err)
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, conversation.EventTypeConversationUpdated, res)

	return res, nil
}
