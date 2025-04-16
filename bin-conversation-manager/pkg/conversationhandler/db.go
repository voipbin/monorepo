package conversationhandler

import (
	"context"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-conversation-manager/models/conversation"
)

// Get returns conversation
func (h *conversationHandler) Get(ctx context.Context, id uuid.UUID) (*conversation.Conversation, error) {
	return h.db.ConversationGet(ctx, id)
}

// GetByReferenceInfo returns conversation
func (h *conversationHandler) GetByReferenceInfo(ctx context.Context, customerID uuid.UUID, referenceType conversation.Type, referenceID string) (*conversation.Conversation, error) {
	return h.db.ConversationGetByTypeAndDialogID(ctx, customerID, referenceType, referenceID)
}

// GetBySelfAndPeer returns conversation
func (h *conversationHandler) GetBySelfAndPeer(ctx context.Context, self *commonaddress.Address, peer *commonaddress.Address) (*conversation.Conversation, error) {
	return h.db.ConversationGetBySelfAndPeer(ctx, self, peer)
}

// Gets returns list of conversations
func (h *conversationHandler) Gets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]*conversation.Conversation, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "Gets",
		"filters": filters,
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
	referenceType conversation.Type,
	referenceID string,
	self *commonaddress.Address,
	peer *commonaddress.Address,
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
		Type:     referenceType,
		DialogID: referenceID,
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
func (h *conversationHandler) Update(ctx context.Context, id uuid.UUID, name string, detail string) (*conversation.Conversation, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":   "Update",
		"id":     id,
		"name":   name,
		"detail": detail,
	})

	if errSet := h.db.ConversationSet(ctx, id, name, detail); errSet != nil {
		log.Errorf("Could not set conversation. err: %v", errSet)
		return nil, errSet
	}

	res, err := h.db.ConversationGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated conversation. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, conversation.EventTypeConversationUpdated, res)

	return res, nil
}
