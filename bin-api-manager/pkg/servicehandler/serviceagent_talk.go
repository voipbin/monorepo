package servicehandler

import (
	"context"
	"fmt"

	amagent "monorepo/bin-agent-manager/models/agent"
	tkchat "monorepo/bin-talk-manager/models/chat"
	tkmessage "monorepo/bin-talk-manager/models/message"
	tkparticipant "monorepo/bin-talk-manager/models/participant"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

// ServiceAgentTalkChatGet gets a chat by ID
func (h *serviceHandler) ServiceAgentTalkChatGet(ctx context.Context, a *amagent.Agent, chatID uuid.UUID) (*tkchat.WebhookMessage, error) {
	// Check permission - agent must have access to the chat
	// Public "talk" type chats are accessible to all agents in the customer
	// For group/direct chats, agent must be a participant
	if !h.canAccessChat(ctx, a.ID, a.CustomerID, chatID) {
		return nil, fmt.Errorf("agent has no permission to access this chat")
	}

	// Get chat
	tmp, err := h.talkGet(ctx, chatID)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not get chat.")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ServiceAgentTalkChatList gets list of chats where the agent is a participant
// Returns only chats where the agent has joined (talk, group, or direct types)
func (h *serviceHandler) ServiceAgentTalkChatList(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*tkchat.WebhookMessage, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	// Get chats where agent is a participant
	filters := map[string]any{
		"owner_type": "agent",
		"owner_id":   a.ID.String(),
		"deleted":    false,
	}
	chats, err := h.reqHandler.TalkV1ChatList(ctx, filters, token, size)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not get chats.")
	}

	// Convert to webhook messages (exclude participants for list response)
	res := []*tkchat.WebhookMessage{}
	for _, c := range chats {
		wm := c.ConvertWebhookMessage()
		wm.Participants = nil
		res = append(res, wm)
	}

	return res, nil
}

// ServiceAgentTalkChannelList gets all public "talk" type channels for the customer
// Returns all public channels regardless of agent participation (for discovery/joining)
func (h *serviceHandler) ServiceAgentTalkChannelList(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*tkchat.WebhookMessage, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	// Get all public talk-type channels for the customer
	filters := map[string]any{
		"customer_id": a.CustomerID.String(),
		"type":        string(tkchat.TypeTalk),
		"deleted":     false,
	}
	channels, err := h.reqHandler.TalkV1ChatList(ctx, filters, token, size)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not get channels.")
	}

	// Convert to webhook messages (exclude participants for list response)
	res := []*tkchat.WebhookMessage{}
	for _, c := range channels {
		wm := c.ConvertWebhookMessage()
		wm.Participants = nil
		res = append(res, wm)
	}

	return res, nil
}

// ServiceAgentTalkCreate creates a new talk
func (h *serviceHandler) ServiceAgentTalkChatCreate(ctx context.Context, a *amagent.Agent, talkType tkchat.Type, name string, detail string, participants []tkparticipant.ParticipantInput) (*tkchat.WebhookMessage, error) {
	// Create talk via RPC with name, detail, and participants
	// Agent is automatically added as first participant by talk-manager
	tmp, err := h.reqHandler.TalkV1ChatCreate(ctx, a.CustomerID, talkType, name, detail, "agent", a.ID, participants)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not create talk.")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ServiceAgentTalkChatUpdate updates a chat's name and/or detail
func (h *serviceHandler) ServiceAgentTalkChatUpdate(ctx context.Context, a *amagent.Agent, chatID uuid.UUID, name *string, detail *string) (*tkchat.WebhookMessage, error) {
	// Check permission - must be a participant to modify
	if !h.isParticipantOfTalk(ctx, a.ID, chatID) {
		return nil, fmt.Errorf("agent is not a participant of this chat")
	}

	// Update chat via RPC
	tmp, err := h.reqHandler.TalkV1ChatUpdate(ctx, chatID, name, detail)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not update talk.")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ServiceAgentTalkChatDelete deletes a chat
func (h *serviceHandler) ServiceAgentTalkChatDelete(ctx context.Context, a *amagent.Agent, chatID uuid.UUID) (*tkchat.WebhookMessage, error) {
	// Check permission - must be a participant to delete
	if !h.isParticipantOfTalk(ctx, a.ID, chatID) {
		return nil, fmt.Errorf("agent is not a participant of this chat")
	}

	// Delete chat via RPC
	tmp, err := h.reqHandler.TalkV1ChatDelete(ctx, chatID)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not delete talk.")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ServiceAgentTalkParticipantList gets list of participants in a chat
func (h *serviceHandler) ServiceAgentTalkParticipantList(ctx context.Context, a *amagent.Agent, chatID uuid.UUID) ([]*tkparticipant.WebhookMessage, error) {
	// Check permission - agent must have access to the chat
	// Public "talk" type chats are accessible to all agents in the customer
	// For group/direct chats, agent must be a participant
	if !h.canAccessChat(ctx, a.ID, a.CustomerID, chatID) {
		return nil, fmt.Errorf("agent has no permission to access this chat")
	}

	// Get participants via RPC
	participants, err := h.reqHandler.TalkV1ParticipantList(ctx, chatID)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not get participants.")
	}

	// Convert to webhook messages
	res := []*tkparticipant.WebhookMessage{}
	for _, p := range participants {
		res = append(res, p.ConvertWebhookMessage())
	}

	return res, nil
}

// ServiceAgentTalkParticipantCreate adds a participant to a chat
func (h *serviceHandler) ServiceAgentTalkParticipantCreate(ctx context.Context, a *amagent.Agent, chatID uuid.UUID, ownerType string, ownerID uuid.UUID) (*tkparticipant.WebhookMessage, error) {
	// Check permission - must be a participant OR joining a public "talk" type chat
	// For "talk" type chats (public channels): anyone in the customer can add themselves (join)
	// For group/direct chats: only existing participants can add others
	if !h.canAddParticipant(ctx, a, chatID, ownerType, ownerID) {
		return nil, fmt.Errorf("agent is not a participant of this chat")
	}

	// Add participant via RPC
	tmp, err := h.reqHandler.TalkV1ParticipantCreate(ctx, chatID, ownerType, ownerID)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not add participant.")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ServiceAgentTalkParticipantDelete removes a participant from a chat
func (h *serviceHandler) ServiceAgentTalkParticipantDelete(ctx context.Context, a *amagent.Agent, chatID uuid.UUID, participantID uuid.UUID) (*tkparticipant.WebhookMessage, error) {
	// Check permission - must be a participant to remove others
	if !h.isParticipantOfTalk(ctx, a.ID, chatID) {
		return nil, fmt.Errorf("agent is not a participant of this chat")
	}

	// Delete participant via RPC
	tmp, err := h.reqHandler.TalkV1ParticipantDelete(ctx, chatID, participantID)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not delete participant.")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ServiceAgentTalkMessageGet gets a message by ID
func (h *serviceHandler) ServiceAgentTalkMessageGet(ctx context.Context, a *amagent.Agent, messageID uuid.UUID) (*tkmessage.WebhookMessage, error) {
	// Get message
	tmp, err := h.talkMessageGet(ctx, messageID)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not get message.")
	}

	// Check permission - agent must have access to the chat
	// Public "talk" type chats are accessible to all agents in the customer
	// For group/direct chats, agent must be a participant
	if !h.canAccessChat(ctx, a.ID, a.CustomerID, tmp.ChatID) {
		return nil, fmt.Errorf("agent has no permission to access this chat")
	}

	// Convert to WebhookMessage
	res, err := tmp.ConvertWebhookMessage()
	if err != nil {
		return nil, errors.Wrapf(err, "Could not convert message.")
	}

	return res, nil
}

// ServiceAgentTalkMessageList gets list of messages for a specific chat
func (h *serviceHandler) ServiceAgentTalkMessageList(ctx context.Context, a *amagent.Agent, chatID uuid.UUID, size uint64, token string) ([]*tkmessage.WebhookMessage, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	// Check permission - agent must have access to the chat
	// Public "talk" type chats are accessible to all agents in the customer
	// For group/direct chats, agent must be a participant
	if !h.canAccessChat(ctx, a.ID, a.CustomerID, chatID) {
		return nil, fmt.Errorf("agent is not a participant of this chat")
	}

	// Get messages for this specific chat
	messages, err := h.talkMessageListByChatIDs(ctx, []uuid.UUID{chatID}, size, token)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not get messages.")
	}

	// Convert to WebhookMessages
	res := []*tkmessage.WebhookMessage{}
	for _, m := range messages {
		wm, err := m.ConvertWebhookMessage()
		if err != nil {
			continue
		}
		res = append(res, wm)
	}

	return res, nil
}

// ServiceAgentTalkMessageCreate creates a new message
func (h *serviceHandler) ServiceAgentTalkMessageCreate(ctx context.Context, a *amagent.Agent, chatID uuid.UUID, parentID *uuid.UUID, msgType tkmessage.Type, text string, medias []tkmessage.Media) (*tkmessage.WebhookMessage, error) {
	// Check permission
	if !h.isParticipantOfTalk(ctx, a.ID, chatID) {
		return nil, fmt.Errorf("agent is not a participant of this talk")
	}

	// Create message via RPC
	tmp, err := h.reqHandler.TalkV1MessageCreate(ctx, chatID, parentID, "agent", a.ID, msgType, text, medias)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not create message.")
	}

	// Convert to WebhookMessage
	res, err := tmp.ConvertWebhookMessage()
	if err != nil {
		return nil, errors.Wrapf(err, "Could not convert message.")
	}

	return res, nil
}

// ServiceAgentTalkMessageDelete deletes a message
func (h *serviceHandler) ServiceAgentTalkMessageDelete(ctx context.Context, a *amagent.Agent, messageID uuid.UUID) (*tkmessage.WebhookMessage, error) {
	// Get message to check ownership and talk
	tmp, err := h.talkMessageGet(ctx, messageID)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not get message.")
	}

	// Check permission - must own the message
	if tmp.OwnerID != a.ID {
		return nil, fmt.Errorf("agent does not own this message")
	}

	// Delete message via RPC
	deleted, err := h.reqHandler.TalkV1MessageDelete(ctx, messageID)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not delete message.")
	}

	// Convert to WebhookMessage
	res, err := deleted.ConvertWebhookMessage()
	if err != nil {
		return nil, errors.Wrapf(err, "Could not convert message.")
	}

	return res, nil
}

// ServiceAgentTalkMessageReactionCreate adds a reaction to a message
func (h *serviceHandler) ServiceAgentTalkMessageReactionCreate(ctx context.Context, a *amagent.Agent, messageID uuid.UUID, emoji string) (*tkmessage.WebhookMessage, error) {
	// Get message to check talk
	tmp, err := h.talkMessageGet(ctx, messageID)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not get message.")
	}

	// Check permission - must be participant of the talk
	if !h.isParticipantOfTalk(ctx, a.ID, tmp.ChatID) {
		return nil, fmt.Errorf("agent is not a participant of this talk")
	}

	// Add reaction via RPC
	updated, err := h.reqHandler.TalkV1MessageReactionCreate(ctx, messageID, "agent", a.ID, emoji)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not add reaction.")
	}

	// Convert to WebhookMessage
	res, err := updated.ConvertWebhookMessage()
	if err != nil {
		return nil, errors.Wrapf(err, "Could not convert message.")
	}

	return res, nil
}

// Helper functions

// isParticipantOfTalk checks if an agent is a participant of a chat
func (h *serviceHandler) isParticipantOfTalk(ctx context.Context, agentID uuid.UUID, chatID uuid.UUID) bool {
	chat, err := h.talkGet(ctx, chatID)
	if err != nil {
		return false
	}

	for _, p := range chat.Participants {
		if p.OwnerType == "agent" && p.OwnerID == agentID {
			return true
		}
	}

	return false
}

// canAccessChat checks if an agent can access a chat (view messages)
// Returns true if:
// - Chat is a public "talk" type (anyone in the customer can view)
// - Agent is a participant of the chat (for group/direct types)
func (h *serviceHandler) canAccessChat(ctx context.Context, agentID uuid.UUID, customerID uuid.UUID, chatID uuid.UUID) bool {
	chat, err := h.talkGet(ctx, chatID)
	if err != nil {
		return false
	}

	// Public talk-type chats are accessible to all agents in the customer
	if chat.Type == tkchat.TypeTalk && chat.CustomerID == customerID {
		return true
	}

	// For group/direct chats, check if agent is a participant
	for _, p := range chat.Participants {
		if p.OwnerType == "agent" && p.OwnerID == agentID {
			return true
		}
	}

	return false
}

// talkGet gets a chat by ID via RPC
func (h *serviceHandler) talkGet(ctx context.Context, chatID uuid.UUID) (*tkchat.Chat, error) {
	return h.reqHandler.TalkV1ChatGet(ctx, chatID)
}

// talkMessageGet gets a message by ID via RPC
func (h *serviceHandler) talkMessageGet(ctx context.Context, messageID uuid.UUID) (*tkmessage.Message, error) {
	return h.reqHandler.TalkV1MessageGet(ctx, messageID)
}

// talkMessageListByChatIDs gets messages by chat IDs
func (h *serviceHandler) talkMessageListByChatIDs(ctx context.Context, chatIDs []uuid.UUID, size uint64, token string) ([]*tkmessage.Message, error) {
	// Build filters with chat_ids (note: currently only supports single chat_id due to filter limitations)
	// For multiple chat IDs, this would need to make multiple requests or enhanced backend support
	if len(chatIDs) == 0 {
		return []*tkmessage.Message{}, nil
	}

	// For now, filter by first chat ID (simplified implementation)
	// TODO: Enhance to support multiple chat IDs when backend supports it
	filters := map[string]any{
		"chat_id": chatIDs[0].String(),
	}

	return h.reqHandler.TalkV1MessageListWithFilters(ctx, filters, token, size)
}

// canAddParticipant checks if an agent can add a participant to a chat
// Returns true if:
// - Agent is already a participant (can add anyone)
// - Chat is a "talk" type (public channel) and agent is adding themselves (joining)
func (h *serviceHandler) canAddParticipant(ctx context.Context, a *amagent.Agent, chatID uuid.UUID, ownerType string, ownerID uuid.UUID) bool {
	chat, err := h.talkGet(ctx, chatID)
	if err != nil {
		return false
	}

	// Check if agent is already a participant - can add anyone
	for _, p := range chat.Participants {
		if p.OwnerType == "agent" && p.OwnerID == a.ID {
			return true
		}
	}

	// For "talk" type (public channels): allow agent to add themselves (join)
	if chat.Type == tkchat.TypeTalk && chat.CustomerID == a.CustomerID {
		// Agent can only add themselves to a public channel
		if ownerType == "agent" && ownerID == a.ID {
			return true
		}
	}

	return false
}
