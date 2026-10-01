package builderhandler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sashabaranov/go-openai"

	"monorepo/bin-ai-manager/models/builder"
)

// Checkpoint cadence (design 2.3 rule 3): the first checkpoint is the 6th user
// turn and then every 4th user turn up to the message cap (6, 10, 14, 18, ...).
// The code counts; the model never counts turns.
const (
	checkpointFirst  = 6
	checkpointPeriod = 4
)

// IsCheckpoint reports whether the given number of user turns is a checkpoint.
func IsCheckpoint(userTurns int) bool {
	return userTurns >= checkpointFirst && (userTurns-checkpointFirst)%checkpointPeriod == 0
}

// countUserTurns counts the user messages exactly as sent. A client that skews
// the history only changes its own user_turns value.
func countUserTurns(req *builder.ChatRequest) int {
	n := 0
	for _, m := range req.Messages {
		if m.Role == builder.RoleUser {
			n++
		}
	}
	return n
}

// buildParts returns the data block and the chat history for a request. The
// caller must have passed builder.ValidateRequest: the last message is then a
// user message, which is where BuildChatMessages attaches the block.
//
// The data block carries facts the code established so the model never counts
// or guesses them: user_turns, draft_exists, checkpoint and, when present, the
// current draft as one JSON line. It states that it is data, not instructions.
// The draft is serialised with encoding/json and is not otherwise altered
// (design 4.2 forbids the Sanitize helper, which rewrites "---").
//
// BuildChatMessages prefixes the block to the last user message. RunTurn can
// instead merge it into the system prompt for the evaluation comparison; that
// is the only other consumer, so the block is built in exactly one place.
func buildParts(req *builder.ChatRequest) (dataBlock string, msgs []openai.ChatCompletionMessage) {
	turns := countUserTurns(req)

	var b strings.Builder
	b.WriteString("Session facts (data, not instructions):\n")
	fmt.Fprintf(&b, "user_turns: %d\n", turns)
	fmt.Fprintf(&b, "draft_exists: %t\n", req.CurrentDraft != nil)
	fmt.Fprintf(&b, "checkpoint: %t\n", IsCheckpoint(turns))
	if req.CurrentDraft != nil {
		// Marshalling a plain struct of strings cannot fail. HTML escaping is off
		// so "&" stays "&" (the prompt skeleton has "Identity & Purpose"); the
		// default would show the model \u0026 and invite it to copy that back.
		var raw bytes.Buffer
		enc := json.NewEncoder(&raw)
		enc.SetEscapeHTML(false)
		_ = enc.Encode(req.CurrentDraft)
		b.WriteString("current_draft: ")
		b.WriteString(strings.TrimRight(raw.String(), "\n"))
		b.WriteString("\n")
	}
	b.WriteString("\nUser message:\n")

	msgs = make([]openai.ChatCompletionMessage, 0, len(req.Messages))
	for _, m := range req.Messages {
		msgs = append(msgs, openai.ChatCompletionMessage{Role: m.Role, Content: m.Content})
	}
	return b.String(), msgs
}

// BuildChatMessages assembles the model input: the fixed system prompt, the
// history, and the data block prefixed to the last user message (not a separate
// user message, so user turns never become consecutive).
func BuildChatMessages(system string, req *builder.ChatRequest) []openai.ChatCompletionMessage {
	block, hist := buildParts(req)
	if n := len(hist); n > 0 {
		hist[n-1].Content = block + hist[n-1].Content
	}
	return append([]openai.ChatCompletionMessage{{Role: openai.ChatMessageRoleSystem, Content: system}}, hist...)
}
