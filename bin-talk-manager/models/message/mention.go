package message

import (
	"regexp"

	"github.com/gofrs/uuid"
)

// maxMentionsPerMessage bounds the mentions one message can have.
const maxMentionsPerMessage = 50

// regAgentMention matches an agent mention written by the clients: [@display name](mention:<agent-id>).
// The id is restricted to a UUID, so a contact mention (mention:contact:<id>) never matches, and a
// display name containing "]" is not a mention, as on the clients.
//
// Other services parse the same text with ExtractAgentMentions (bin-push-manager decides the
// recipients of a mention notification this way). A service that is built from its own directory is
// not rebuilt when only this file changes, so when this expression changes, deploy those services
// together with bin-talk-manager.
var regAgentMention = regexp.MustCompile(`\[@([^\]]+)\]\(mention:([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})\)`)

// ExtractAgentMentions returns the distinct agent ids mentioned in the text in the order of first
// appearance, without the author, and at most maxMentionsPerMessage of them. It is a pure function.
func ExtractAgentMentions(text string, authorID uuid.UUID) []uuid.UUID {
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
