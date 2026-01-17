package servicehandler

import (
	"context"
	"fmt"

	amagent "monorepo/bin-agent-manager/models/agent"
	tkmessage "monorepo/bin-talk-manager/models/message"
	tkparticipant "monorepo/bin-talk-manager/models/participant"
	tktalk "monorepo/bin-talk-manager/models/talk"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

// ServiceAgentTalkGet gets a talk by ID
func (h *serviceHandler) ServiceAgentTalkGet(ctx context.Context, a *amagent.Agent, talkID uuid.UUID) (*tktalk.WebhookMessage, error) {
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
func (h *serviceHandler) ServiceAgentTalkList(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*tktalk.WebhookMessage, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	// Get all talks the agent participates in
	participants, err := h.talkParticipantListByOwner(ctx, a.CustomerID, "agent", a.ID)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not get participant list.")
	}

	// Collect unique talk IDs
	talkIDs := make([]uuid.UUID, 0, len(participants))
	for _, p := range participants {
		talkIDs = append(talkIDs, p.ChatID)
	}

	if len(talkIDs) == 0 {
		return []*tktalk.WebhookMessage{}, nil
	}

	// Get talks
	talks, err := h.talkListByIDs(ctx, talkIDs, size, token)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not get talks.")
	}

	// Convert to webhook messages
	res := []*tktalk.WebhookMessage{}
	for _, t := range talks {
		res = append(res, t.ConvertWebhookMessage())
	}

	return res, nil
}

// ServiceAgentTalkCreate creates a new talk
func (h *serviceHandler) ServiceAgentTalkCreate(ctx context.Context, a *amagent.Agent, talkType tktalk.Type) (*tktalk.WebhookMessage, error) {
	// Create talk via RPC
	tmp, err := h.reqHandler.TalkV1TalkCreate(ctx, a.CustomerID, talkType)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not create talk.")
	}

	// Add agent as first participant
	_, err = h.reqHandler.TalkV1TalkParticipantCreate(ctx, tmp.ID, "agent", a.ID)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not add agent as participant.")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ServiceAgentTalkDelete deletes a talk
func (h *serviceHandler) ServiceAgentTalkDelete(ctx context.Context, a *amagent.Agent, talkID uuid.UUID) (*tktalk.WebhookMessage, error) {
	// Check permission
	if !h.isParticipantOfTalk(ctx, a.ID, talkID) {
		return nil, fmt.Errorf("agent is not a participant of this talk")
	}

	// Delete talk via RPC
	tmp, err := h.reqHandler.TalkV1TalkDelete(ctx, talkID)
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
	participants, err := h.reqHandler.TalkV1TalkParticipantList(ctx, talkID)
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
	tmp, err := h.reqHandler.TalkV1TalkParticipantCreate(ctx, talkID, ownerType, ownerID)
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
	tmp, err := h.reqHandler.TalkV1TalkParticipantDelete(ctx, talkID, participantID)
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

// ServiceAgentTalkMessageList gets list of messages for the agent
func (h *serviceHandler) ServiceAgentTalkMessageList(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*tkmessage.WebhookMessage, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	// Get all talks the agent participates in
	participants, err := h.talkParticipantListByOwner(ctx, a.CustomerID, "agent", a.ID)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not get participant list.")
	}

	// Collect unique talk IDs
	talkIDs := make([]uuid.UUID, 0, len(participants))
	for _, p := range participants {
		talkIDs = append(talkIDs, p.ChatID)
	}

	if len(talkIDs) == 0 {
		return []*tkmessage.WebhookMessage{}, nil
	}

	// Get messages from those talks
	messages, err := h.talkMessageListByTalkIDs(ctx, talkIDs, size, token)
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
	tmp, err := h.reqHandler.TalkV1TalkMessageCreate(ctx, chatID, parentID, "agent", a.ID, msgType, text)
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
	deleted, err := h.reqHandler.TalkV1TalkMessageDelete(ctx, messageID)
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
	result, err := h.reqHandler.TalkV1TalkMessageReactionCreate(ctx, messageID, "agent", a.ID, emoji)
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
	participants, err := h.reqHandler.TalkV1TalkParticipantList(ctx, talkID)
	if err != nil {
		return false
	}

	for _, p := range participants {
		if p.OwnerType == "agent" && p.OwnerID == agentID {
			return true
		}
	}

	return false
}

// talkGet gets a talk by ID via RPC
func (h *serviceHandler) talkGet(ctx context.Context, talkID uuid.UUID) (*tktalk.Talk, error) {
	return h.reqHandler.TalkV1TalkGet(ctx, talkID)
}

// talkListByIDs gets talks by list of IDs
func (h *serviceHandler) talkListByIDs(ctx context.Context, talkIDs []uuid.UUID, size uint64, token string) ([]*tktalk.Talk, error) {
	// This is a simplified implementation - in production, you'd filter by IDs
	// For now, get all and filter client-side
	talks := []*tktalk.Talk{}
	idMap := make(map[uuid.UUID]bool)
	for _, id := range talkIDs {
		idMap[id] = true
	}

	// Get each talk individually (inefficient but works)
	for id := range idMap {
		talk, err := h.talkGet(ctx, id)
		if err != nil {
			continue // Skip talks that can't be fetched
		}
		talks = append(talks, talk)
		if uint64(len(talks)) >= size {
			break
		}
	}

	return talks, nil
}

// talkParticipantListByOwner gets participants by owner
func (h *serviceHandler) talkParticipantListByOwner(ctx context.Context, customerID uuid.UUID, ownerType string, ownerID uuid.UUID) ([]*tkparticipant.Participant, error) {
	// This would need a proper RPC method in talk-manager that filters by owner
	// For now, this is a placeholder - needs implementation in talk-manager
	return []*tkparticipant.Participant{}, fmt.Errorf("not implemented - needs talk-manager support")
}

// talkMessageGet gets a message by ID via RPC
func (h *serviceHandler) talkMessageGet(ctx context.Context, messageID uuid.UUID) (*tkmessage.Message, error) {
	return h.reqHandler.TalkV1TalkMessageGet(ctx, messageID)
}

// talkMessageListByTalkIDs gets messages by talk IDs
func (h *serviceHandler) talkMessageListByTalkIDs(ctx context.Context, talkIDs []uuid.UUID, size uint64, token string) ([]*tkmessage.Message, error) {
	// This is a simplified implementation
	// For production, you'd need a proper RPC method that filters by multiple talk IDs
	return []*tkmessage.Message{}, fmt.Errorf("not implemented - needs talk-manager support")
}
