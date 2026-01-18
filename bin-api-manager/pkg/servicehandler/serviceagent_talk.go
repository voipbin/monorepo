package servicehandler

import (
	"context"
	"fmt"

	amagent "monorepo/bin-agent-manager/models/agent"
	tkmessage "monorepo/bin-talk-manager/models/message"
	tkparticipant "monorepo/bin-talk-manager/models/participant"
	tkchat "monorepo/bin-talk-manager/models/chat"

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
func (h *serviceHandler) ServiceAgentTalkChatList(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*tkchat.WebhookMessage, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	// Build filters to get talks where agent is a participant
	// Exclude deleted chats
	filters := map[string]any{
		"owner_type": "agent",
		"owner_id":   a.ID.String(),
		"deleted":    false,
	}

	// Get talks with filters using consolidated method
	talks, err := h.reqHandler.TalkV1ChatList(ctx, filters, token, size)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not get talks.")
	}

	// Convert to webhook messages
	res := []*tkchat.WebhookMessage{}
	for _, t := range talks {
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

	// Check permission - agent must be participant of the chat
	if !h.isParticipantOfTalk(ctx, a.ID, chatID) {
		return nil, fmt.Errorf("agent is not a participant of this chat")
	}

	// Get messages for this specific chat
	messages, err := h.talkMessageListByTalkIDs(ctx, []uuid.UUID{chatID}, size, token)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not get messages.")
	}

	// Convert to webhook messages
	res := []*tkmessage.WebhookMessage{}
	for _, m := range messages {
		wm, err := m.ConvertWebhookMessage()
		if err != nil {
			continue // Skip messages that can't be converted
		}
		res = append(res, wm)
	}

	return res, nil
}

// ServiceAgentTalkMessageCreate creates a new message
func (h *serviceHandler) ServiceAgentTalkMessageCreate(ctx context.Context, a *amagent.Agent, chatID uuid.UUID, parentID *uuid.UUID, msgType tkmessage.Type, text string) (*tkmessage.WebhookMessage, error) {
	// Check permission
	if !h.isParticipantOfTalk(ctx, a.ID, chatID) {
		return nil, fmt.Errorf("agent is not a participant of this talk")
	}

	// Create message via RPC
	tmp, err := h.reqHandler.TalkV1MessageCreate(ctx, chatID, parentID, "agent", a.ID, msgType, text)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not create message.")
	}

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
	result, err := h.reqHandler.TalkV1MessageReactionCreate(ctx, messageID, "agent", a.ID, emoji)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not add reaction.")
	}

	res, err := result.ConvertWebhookMessage()
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
