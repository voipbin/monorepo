package servicehandler

import (
	"context"
	"fmt"
	"sort"

	amagent "monorepo/bin-agent-manager/models/agent"
	tkchat "monorepo/bin-talk-manager/models/chat"
	tkmessage "monorepo/bin-talk-manager/models/message"
	tkparticipant "monorepo/bin-talk-manager/models/participant"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

// ServiceAgentTalkGet gets a talk by ID
func (h *serviceHandler) ServiceAgentTalkChatGet(ctx context.Context, a *amagent.Agent, talkID uuid.UUID) (*tkchat.WebhookMessage, error) {
	// Get talk
	tmp, err := h.talkGet(ctx, talkID)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not get talk.")
	}

	// Check permission - must be a participant
	if !h.isParticipantOfTalk(ctx, a.ID, talkID) {
		return nil, fmt.Errorf("agent is not a participant of this talk")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ServiceAgentTalkList gets list of talks for the agent
// Returns:
// - All public "talk" type chats for the customer (regardless of participation)
// - "group" type chats where the agent is a participant
// - "direct" type chats where the agent is a participant
func (h *serviceHandler) ServiceAgentTalkChatList(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*tkchat.WebhookMessage, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	// Query A: All public talk-type chats for the customer (no participant filter)
	talkFilters := map[string]any{
		"customer_id": a.CustomerID.String(),
		"type":        string(tkchat.TypeTalk),
		"deleted":     false,
	}
	talkChats, err := h.reqHandler.TalkV1ChatList(ctx, talkFilters, token, size)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not get talk-type chats.")
	}

	// Query B: Private chats (group/direct) where agent is participant
	privateFilters := map[string]any{
		"owner_type": "agent",
		"owner_id":   a.ID.String(),
		"deleted":    false,
	}
	privateChats, err := h.reqHandler.TalkV1ChatList(ctx, privateFilters, token, size)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not get private chats.")
	}

	// Combine results and deduplicate
	seen := make(map[uuid.UUID]bool)
	var allChats []*tkchat.Chat

	// Add talk-type chats first
	for _, c := range talkChats {
		if !seen[c.ID] {
			seen[c.ID] = true
			allChats = append(allChats, c)
		}
	}

	// Add private chats (group/direct only, skip talk-type to avoid duplicates)
	for _, c := range privateChats {
		if !seen[c.ID] && c.Type != tkchat.TypeTalk {
			seen[c.ID] = true
			allChats = append(allChats, c)
		}
	}

	// Sort by tm_create descending
	sort.Slice(allChats, func(i, j int) bool {
		return allChats[i].TMCreate > allChats[j].TMCreate
	})

	// Limit to requested size
	if uint64(len(allChats)) > size {
		allChats = allChats[:size]
	}

	// Convert to webhook messages
	res := []*tkchat.WebhookMessage{}
	for _, t := range allChats {
		res = append(res, t.ConvertWebhookMessage())
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

// ServiceAgentTalkChatUpdate updates a talk's name and/or detail
func (h *serviceHandler) ServiceAgentTalkChatUpdate(ctx context.Context, a *amagent.Agent, talkID uuid.UUID, name *string, detail *string) (*tkchat.WebhookMessage, error) {
	// Check permission
	if !h.isParticipantOfTalk(ctx, a.ID, talkID) {
		return nil, fmt.Errorf("agent is not a participant of this talk")
	}

	// Update talk via RPC
	tmp, err := h.reqHandler.TalkV1ChatUpdate(ctx, talkID, name, detail)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not update talk.")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ServiceAgentTalkDelete deletes a talk
func (h *serviceHandler) ServiceAgentTalkChatDelete(ctx context.Context, a *amagent.Agent, talkID uuid.UUID) (*tkchat.WebhookMessage, error) {
	// Check permission
	if !h.isParticipantOfTalk(ctx, a.ID, talkID) {
		return nil, fmt.Errorf("agent is not a participant of this talk")
	}

	// Delete talk via RPC
	tmp, err := h.reqHandler.TalkV1ChatDelete(ctx, talkID)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not delete talk.")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ServiceAgentTalkParticipantList gets list of participants in a talk
func (h *serviceHandler) ServiceAgentTalkParticipantList(ctx context.Context, a *amagent.Agent, talkID uuid.UUID) ([]*tkparticipant.WebhookMessage, error) {
	// Check permission
	if !h.isParticipantOfTalk(ctx, a.ID, talkID) {
		return nil, fmt.Errorf("agent is not a participant of this talk")
	}

	// Get participants via RPC
	participants, err := h.reqHandler.TalkV1ParticipantList(ctx, talkID)
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

// ServiceAgentTalkParticipantCreate adds a participant to a talk
func (h *serviceHandler) ServiceAgentTalkParticipantCreate(ctx context.Context, a *amagent.Agent, talkID uuid.UUID, ownerType string, ownerID uuid.UUID) (*tkparticipant.WebhookMessage, error) {
	// Check permission
	if !h.isParticipantOfTalk(ctx, a.ID, talkID) {
		return nil, fmt.Errorf("agent is not a participant of this talk")
	}

	// Add participant via RPC
	tmp, err := h.reqHandler.TalkV1ParticipantCreate(ctx, talkID, ownerType, ownerID)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not add participant.")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ServiceAgentTalkParticipantDelete removes a participant from a talk
func (h *serviceHandler) ServiceAgentTalkParticipantDelete(ctx context.Context, a *amagent.Agent, talkID uuid.UUID, participantID uuid.UUID) (*tkparticipant.WebhookMessage, error) {
	// Check permission
	if !h.isParticipantOfTalk(ctx, a.ID, talkID) {
		return nil, fmt.Errorf("agent is not a participant of this talk")
	}

	// Delete participant via RPC
	tmp, err := h.reqHandler.TalkV1ParticipantDelete(ctx, talkID, participantID)
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

	// Check permission - must be participant of the talk
	if !h.isParticipantOfTalk(ctx, a.ID, tmp.ChatID) {
		return nil, fmt.Errorf("agent is not a participant of this talk")
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
	messages, err := h.talkMessageListByTalkIDs(ctx, []uuid.UUID{chatID}, size, token)
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

// isParticipantOfTalk checks if an agent is a participant of a talk
func (h *serviceHandler) isParticipantOfTalk(ctx context.Context, agentID uuid.UUID, talkID uuid.UUID) bool {
	chat, err := h.talkGet(ctx, talkID)
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
func (h *serviceHandler) canAccessChat(ctx context.Context, agentID uuid.UUID, customerID uuid.UUID, talkID uuid.UUID) bool {
	chat, err := h.talkGet(ctx, talkID)
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

// talkGet gets a talk by ID via RPC
func (h *serviceHandler) talkGet(ctx context.Context, talkID uuid.UUID) (*tkchat.Chat, error) {
	return h.reqHandler.TalkV1ChatGet(ctx, talkID)
}

// talkMessageGet gets a message by ID via RPC
func (h *serviceHandler) talkMessageGet(ctx context.Context, messageID uuid.UUID) (*tkmessage.Message, error) {
	return h.reqHandler.TalkV1MessageGet(ctx, messageID)
}

// talkMessageListByTalkIDs gets messages by talk IDs
func (h *serviceHandler) talkMessageListByTalkIDs(ctx context.Context, talkIDs []uuid.UUID, size uint64, token string) ([]*tkmessage.Message, error) {
	// Build filters with chat_ids (note: currently only supports single chat_id due to filter limitations)
	// For multiple chat IDs, this would need to make multiple requests or enhanced backend support
	if len(talkIDs) == 0 {
		return []*tkmessage.Message{}, nil
	}

	// For now, filter by first talk ID (simplified implementation)
	// TODO: Enhance to support multiple talk IDs when backend supports it
	filters := map[string]any{
		"chat_id": talkIDs[0].String(),
	}

	return h.reqHandler.TalkV1MessageListWithFilters(ctx, filters, token, size)
}
