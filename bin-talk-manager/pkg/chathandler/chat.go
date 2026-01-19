package chathandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"monorepo/bin-talk-manager/models/chat"
	"monorepo/bin-talk-manager/models/participant"
)

// ChatCreate creates a new talk
func (h *chatHandler) ChatCreate(ctx context.Context, customerID uuid.UUID, chatType chat.Type, name string, detail string, creatorType string, creatorID uuid.UUID, participants []participant.ParticipantInput) (*chat.Chat, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":              "ChatCreate",
		"customer_id":       customerID,
		"type":              chatType,
		"name":              name,
		"detail":            detail,
		"creator_type":      creatorType,
		"creator_id":        creatorID,
		"participants_count": len(participants),
	})
	log.Debug("Creating a new talk")

	// Validate input
	if customerID == uuid.Nil {
		log.Error("Invalid customer ID: nil UUID")
		return nil, errors.New("customer ID cannot be nil")
	}

	// Validate chat type
	if chatType != chat.TypeDirect && chatType != chat.TypeGroup && chatType != chat.TypeTalk {
		log.Errorf("Invalid chat type: %s", chatType)
		return nil, errors.Errorf("invalid chat type: %s", chatType)
	}

	// Validate creator (creator is optional)
	if creatorType != "" && creatorID == uuid.Nil {
		log.Error("Invalid creator ID: nil UUID with non-empty type")
		return nil, errors.New("creator ID cannot be nil when creator type is specified")
	}

	// Validate participants based on chat type
	var otherParticipant *participant.ParticipantInput
	switch chatType {
	case chat.TypeDirect:
		// Direct chats require exactly 1 additional participant (the other party)
		// If creator included themselves in participants, we need exactly 2
		// If creator is not in participants, we need exactly 1
		otherParticipants := 0
		for i := range participants {
			p := &participants[i]
			if p.OwnerType != creatorType || p.OwnerID != creatorID {
				otherParticipants++
				otherParticipant = p
			}
		}
		if otherParticipants != 1 {
			log.Errorf("Direct chat requires exactly 1 other participant, got %d", otherParticipants)
			return nil, errors.Errorf("direct chat requires exactly 1 other participant, got %d", otherParticipants)
		}

		// Check if a direct chat already exists between these two participants
		existingChat, err := h.dbHandler.FindDirectChatByParticipants(ctx, customerID, creatorType, creatorID, otherParticipant.OwnerType, otherParticipant.OwnerID)
		if err != nil {
			log.Errorf("Failed to check for existing direct chat. err: %v", err)
			// Continue with creation on error (don't block)
		} else if existingChat != nil {
			// Found existing direct chat, load participants and return it
			log.WithField("existing_chat_id", existingChat.ID).Debug("Found existing direct chat, returning it instead of creating new")
			result, err := h.ChatGet(ctx, existingChat.ID)
			if err != nil {
				log.Errorf("Failed to reload existing chat with participants. err: %v", err)
				// Return the existing chat without participants if reload fails
				return existingChat, nil
			}
			return result, nil
		}
	case chat.TypeGroup:
		// Group chats can start with just the creator
		// Members can be added/removed later
		// No validation needed for participants
	case chat.TypeTalk:
		// Talk type doesn't require additional participants (only creator)
		// No validation needed for participants
	}

	// Create chat object using utilHandler for UUID generation
	t := &chat.Chat{
		Identity: commonidentity.Identity{
			ID:         h.utilHandler.UUIDCreate(),
			CustomerID: customerID,
		},
		Type:   chatType,
		Name:   name,
		Detail: detail,
	}

	// Save to database
	err := h.dbHandler.ChatCreate(ctx, t)
	if err != nil {
		log.Errorf("Failed to create chat in database. err: %v", err)
		return nil, errors.Wrap(err, "failed to create chat in database")
	}

	// Check if creator is already in the participants list
	creatorInParticipants := false
	for _, p := range participants {
		if p.OwnerType == creatorType && p.OwnerID == creatorID {
			creatorInParticipants = true
			break
		}
	}

	// Add creator as participant if provided and not already in participants list
	if creatorType != "" && creatorID != uuid.Nil && !creatorInParticipants {
		_, err = h.participantHandler.ParticipantAdd(ctx, t.ID, creatorID, creatorType)
		if err != nil {
			log.Errorf("Failed to add creator as participant. err: %v", err)
			// Note: We don't fail the entire chat creation if participant addition fails
			// The chat is already created; this is a best-effort participant addition
		} else {
			log.WithFields(logrus.Fields{
				"chat_id":      t.ID,
				"creator_type": creatorType,
				"creator_id":   creatorID,
			}).Debug("Creator added as participant")
		}
	}

	// Add all participants from the list
	for _, p := range participants {
		_, err = h.participantHandler.ParticipantAdd(ctx, t.ID, p.OwnerID, p.OwnerType)
		if err != nil {
			log.Errorf("Failed to add participant. owner_type: %s, owner_id: %v, err: %v", p.OwnerType, p.OwnerID, err)
			// Note: We don't fail the entire chat creation if participant addition fails
			// The chat is already created; this is a best-effort participant addition
		} else {
			log.WithFields(logrus.Fields{
				"chat_id":    t.ID,
				"owner_type": p.OwnerType,
				"owner_id":   p.OwnerID,
			}).Debug("Participant added")
		}
	}

	// Load chat with participants before returning
	result, err := h.ChatGet(ctx, t.ID)
	if err != nil {
		log.Errorf("Failed to reload chat with participants. err: %v", err)
		// Return original chat without participants if reload fails
		result = t
	}

	// Publish webhook event
	h.notifyHandler.PublishWebhookEvent(ctx, result.CustomerID, chat.EventTypeChatCreated, result)

	log.WithField("chat_id", result.ID).Debugf("Chat created successfully. chat_id: %s", result.ID)
	return result, nil
}

// ChatGet retrieves a chat by ID
func (h *chatHandler) ChatGet(ctx context.Context, id uuid.UUID) (*chat.Chat, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "ChatGet",
		"chat_id": id,
	})

	t, err := h.dbHandler.ChatGet(ctx, id)
	if err != nil {
		log.Errorf("Failed to get talk. err: %v", err)
		return nil, errors.Wrap(err, "failed to get talk")
	}

	// Load participants for this chat
	participants, err := h.dbHandler.ParticipantListByChatIDs(ctx, []uuid.UUID{t.ID})
	if err != nil {
		log.Errorf("Failed to load participants: %v", err)
		// Continue without participants rather than failing entire request
		t.Participants = []*participant.Participant{}
	} else {
		t.Participants = participants
	}

	return t, nil
}

// ChatList retrieves talks with filters and pagination
func (h *chatHandler) ChatList(ctx context.Context, filters map[chat.Field]any, token string, size uint64) ([]*chat.Chat, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "ChatList",
		"filters": filters,
		"token":   token,
		"size":    size,
	})

	talks, err := h.dbHandler.ChatList(ctx, filters, token, size)
	if err != nil {
		log.Errorf("Failed to list talks. err: %v", err)
		return nil, errors.Wrap(err, "failed to list talks")
	}

	return talks, nil
}

// ChatUpdate updates a chat's name and/or detail
func (h *chatHandler) ChatUpdate(ctx context.Context, id uuid.UUID, name *string, detail *string) (*chat.Chat, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "ChatUpdate",
		"chat_id": id,
	})
	log.Debug("Updating chat")

	// Validate input
	if id == uuid.Nil {
		log.Error("Invalid chat ID: nil UUID")
		return nil, errors.New("chat ID cannot be nil")
	}

	// Check at least one field is being updated
	if name == nil && detail == nil {
		log.Error("No fields to update")
		return nil, errors.New("at least one field must be provided for update")
	}

	// Verify chat exists
	_, err := h.dbHandler.ChatGet(ctx, id)
	if err != nil {
		log.Errorf("Failed to get chat before update. err: %v", err)
		return nil, errors.Wrap(err, "failed to get chat before update")
	}

	// Build update fields
	fields := make(map[chat.Field]any)
	if name != nil {
		fields[chat.FieldName] = *name
	}
	if detail != nil {
		fields[chat.FieldDetail] = *detail
	}

	// Update in database
	err = h.dbHandler.ChatUpdate(ctx, id, fields)
	if err != nil {
		log.Errorf("Failed to update chat. err: %v", err)
		return nil, errors.Wrap(err, "failed to update chat")
	}

	// Get updated chat with participants
	result, err := h.ChatGet(ctx, id)
	if err != nil {
		log.Errorf("Failed to get chat after update. err: %v", err)
		return nil, errors.Wrap(err, "failed to get chat after update")
	}

	// Publish webhook event
	h.notifyHandler.PublishWebhookEvent(ctx, result.CustomerID, chat.EventTypeChatUpdated, result)

	log.Debugf("Chat updated successfully. chat_id: %s", id)
	return result, nil
}

// ChatDelete soft deletes a talk
func (h *chatHandler) ChatDelete(ctx context.Context, id uuid.UUID) (*chat.Chat, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "ChatDelete",
		"chat_id": id,
	})
	log.Debug("Deleting talk")

	// Get chat before deletion for webhook
	t, err := h.dbHandler.ChatGet(ctx, id)
	if err != nil {
		log.Errorf("Failed to get chat before deletion. err: %v", err)
		return nil, errors.Wrap(err, "failed to get chat before deletion")
	}

	// Soft delete in database
	err = h.dbHandler.ChatDelete(ctx, id)
	if err != nil {
		log.Errorf("Failed to delete talk. err: %v", err)
		return nil, errors.Wrap(err, "failed to delete talk")
	}

	// Publish webhook event
	h.notifyHandler.PublishWebhookEvent(ctx, t.CustomerID, chat.EventTypeChatDeleted, t)

	log.Debugf("Chat deleted successfully. chat_id: %s", id)
	return t, nil
}
