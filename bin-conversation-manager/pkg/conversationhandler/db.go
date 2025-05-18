package conversationhandler

import (
	"context"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-conversation-manager/models/conversation"
)

// Get returns conversation
func (h *conversationHandler) Get(ctx context.Context, id uuid.UUID) (*conversation.Conversation, error) {
	return h.db.ConversationGet(ctx, id)
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

// Gets returns list of conversations
func (h *conversationHandler) Gets(ctx context.Context, pageToken string, pageSize uint64, filters map[conversation.Field]any) ([]*conversation.Conversation, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":   "Gets",
		"filers": filters,
	})
	log.Debugf("Getting a list of conversations.")

	res, err := h.db.ConversationGets(ctx, pageSize, pageToken, filters)
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
func (h *conversationHandler) Update(ctx context.Context, id uuid.UUID, fields map[conversation.Field]any) (*conversation.Conversation, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":   "Update",
		"id":     id,
		"fields": fields,
	})
	log.Debugf("Updating conversation. conversation_id: %s", id)

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
