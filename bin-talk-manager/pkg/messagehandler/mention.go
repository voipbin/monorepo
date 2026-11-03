package messagehandler

import (
	"context"
	"regexp"

	"github.com/gofrs/uuid"

	"monorepo/bin-talk-manager/models/message"
)

// maxMentionsPerMessage bounds the mention rows one message can create.
const maxMentionsPerMessage = 50

// regAgentMention matches an agent mention written by the clients: [@display name](mention:<agent-id>).
// The id is restricted to a UUID, so a contact mention (mention:contact:<id>) never matches, and a
// display name containing "]" is not a mention, as on the clients.
var regAgentMention = regexp.MustCompile(`\[@([^\]]+)\]\(mention:([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})\)`)

// extractAgentMentions returns the distinct agent ids mentioned in the text in the order of first
// appearance, without the author, and at most maxMentionsPerMessage of them.
func extractAgentMentions(text string, authorID uuid.UUID) []uuid.UUID {
	var res []uuid.UUID
	seen := map[uuid.UUID]bool{}

	for _, match := range regAgentMention.FindAllStringSubmatch(text, -1) {
		id := uuid.FromStringOrNil(match[2])
		if id == uuid.Nil || id == authorID || seen[id] {
			continue
		}
		seen[id] = true
		res = append(res, id)

		if len(res) >= maxMentionsPerMessage {
			break
		}
	}

	return res
}

// MentionList returns the messages that mention the agent, newest first.
func (h *messageHandler) MentionList(ctx context.Context, customerID, agentID uuid.UUID, token string, size uint64) ([]*message.Message, error) {
	return h.dbHandler.MentionList(ctx, customerID, agentID, token, size)
}
