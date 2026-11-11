package eventhandler

import (
	"context"
	"errors"
	"fmt"
	"sort"

	commonidentity "monorepo/bin-common-handler/models/identity"

	tkmessage "monorepo/bin-talk-manager/models/message"
	tkparticipant "monorepo/bin-talk-manager/models/participant"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-push-manager/models/notification"
)

// EventChatMessageCreated handles the talk-manager's chatmessage_created event.
//
// The mentioned agents get chat_mentioned. They may be outside the chat, and a muted chat does not silence a
// mention. The agents who take part in the chat get chat_message, except the author and the mentioned agents:
// one message gives an agent one notification at most. An agent who switched chat_mentioned off therefore gets
// nothing for the message that mentions him, even if chat_message is on. The recipients are cut to the limit
// after the settings are applied, and the oldest participants are cut first.
func (h *eventHandler) EventChatMessageCreated(ctx context.Context, m *tkmessage.Message) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "EventChatMessageCreated",
		"customer_id": m.CustomerID,
		"chat_id":     m.ChatID,
		"message_id":  m.ID,
	})

	if m.Type != tkmessage.TypeNormal || m.CustomerID == uuid.Nil || m.ChatID == uuid.Nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, h.eventTimeout)
	defer cancel()

	data := map[string]string{
		"chat_id":    m.ChatID.String(),
		"message_id": m.ID.String(),
	}

	// the author is never a mention target
	mentions := tkmessage.ExtractAgentMentions(m.Text, m.OwnerID)

	errMention := h.notifyMentions(ctx, m.CustomerID, mentions, data)
	errMessage := h.notifyParticipants(ctx, m, mentions, data)

	if err := errors.Join(errMention, errMessage); err != nil {
		log.Errorf("Could not notify everybody. err: %v", err)
		return err
	}

	return nil
}

// notifyMentions sends chat_mentioned to the mentioned agents who want it.
func (h *eventHandler) notifyMentions(ctx context.Context, customerID uuid.UUID, mentions []uuid.UUID, data map[string]string) error {
	if len(mentions) == 0 {
		return nil
	}

	targets, err := h.settingsHandler.FilterRecipients(ctx, customerID, mentions, notification.KindChatMentioned, uuid.Nil)
	if err != nil {
		return fmt.Errorf("could not apply the settings of the mentions. err: %w", err)
	}

	h.sendAll(ctx, customerID, targets, notification.KindChatMentioned, data)

	return nil
}

// notifyParticipants sends chat_message to the agents in the chat who want it.
func (h *eventHandler) notifyParticipants(ctx context.Context, m *tkmessage.Message, mentions []uuid.UUID, data map[string]string) error {
	participants, err := h.reqHandler.TalkV1ParticipantList(ctx, m.ChatID)
	if err != nil {
		return fmt.Errorf("could not get the participants. err: %w", err)
	}

	sortParticipants(participants)

	mentioned := make(map[uuid.UUID]bool, len(mentions))
	for _, id := range mentions {
		mentioned[id] = true
	}

	seen := map[uuid.UUID]bool{}
	candidates := []uuid.UUID{}
	for _, p := range participants {
		switch {
		case p == nil,
			p.OwnerType != commonidentity.OwnerTypeAgent,
			p.OwnerID == uuid.Nil,
			p.CustomerID != m.CustomerID, // an agent of another customer is not a recipient
			p.OwnerID == m.OwnerID,
			mentioned[p.OwnerID],
			seen[p.OwnerID]:
			continue
		}

		seen[p.OwnerID] = true
		candidates = append(candidates, p.OwnerID)
	}

	if len(candidates) == 0 {
		return nil
	}

	targets, err := h.settingsHandler.FilterRecipients(ctx, m.CustomerID, candidates, notification.KindChatMessage, m.ChatID)
	if err != nil {
		return fmt.Errorf("could not apply the settings of the chat message. err: %w", err)
	}

	if len(targets) > h.maxRecipients {
		logrus.WithFields(logrus.Fields{
			"func":        "notifyParticipants",
			"customer_id": m.CustomerID,
			"chat_id":     m.ChatID,
			"wanted":      len(targets),
			"limit":       h.maxRecipients,
		}).Warn("The recipients of the chat message are cut to the limit. The oldest participants are cut first.")
		promRecipientTruncated.WithLabelValues(string(notification.KindChatMessage)).Inc()

		targets = targets[:h.maxRecipients]
	}

	h.sendAll(ctx, m.CustomerID, targets, notification.KindChatMessage, data)

	return nil
}

// sortParticipants sorts the participants by the join time, the newest first. A participant without a join time
// is last, and participants who joined at the same time are ordered by id, so the cut at the limit is stable.
func sortParticipants(list []*tkparticipant.Participant) {
	sort.SliceStable(list, func(i, j int) bool {
		a, b := list[i], list[j]
		if a == nil || b == nil {
			return a != nil
		}

		switch {
		case a.TMJoined == nil && b.TMJoined == nil:
			return a.ID.String() < b.ID.String()
		case a.TMJoined == nil:
			return false
		case b.TMJoined == nil:
			return true
		case !a.TMJoined.Equal(*b.TMJoined):
			return a.TMJoined.After(*b.TMJoined)
		}

		return a.ID.String() < b.ID.String()
	})
}
