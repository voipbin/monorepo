package aicallhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"

	"github.com/sirupsen/logrus"
)

// emit_info_card size guards (design doc D1/§1.6). The tool's JSON Schema
// declares matching maxLength/maxItems hints for the LLM, but those are a
// model-facing hint only -- function calls are not run in `strict` mode
// (bin-pipecat-manager/scripts/pipecat/run.py never sets function.strict), so
// schema compliance is not guaranteed. These constants are therefore the
// PRIMARY and only real enforcement.
const (
	cardTitleMaxLen       = 200
	cardDescriptionMaxLen = 1000
	cardFieldLabelMaxLen  = 50
	cardFieldValueMaxLen  = 500
	cardFieldsMaxCount    = 20
)

// truncateCardText clamps s to at most maxLen runes (not bytes, to avoid
// splitting a multi-byte UTF-8 character), appending a trailing "..." when
// truncation actually occurs and maxLen leaves room for it.
func truncateCardText(s string, maxLen int) string {
	r := []rune(s)
	if len(r) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return string(r[:maxLen])
	}
	return string(r[:maxLen-3]) + "..."
}

// toolHandleEmitInfoCard handles the emit_info_card tool: it parses and
// validates the LLM-supplied title/description/fields, applies the handler-
// side size guards above, and returns a *messageContent whose Blocks field
// carries the resulting CardBlock. It performs no DB write and no external
// API call -- its only effect is the message it writes into its own
// session's message stream, the same surface a plain assistant-text turn
// already writes to (design doc §1.4).
//
// Message is deliberately a short, title-only trace -- e.g. "Displayed an
// info card titled '<title>'." -- never the field values. It is the same
// value across the stored/frontend-facing representation, the first-turn
// LLM feedback, and (after getPipecatcallMessages' strip) the N-turn LLM
// history value, so it must never contain content the frontend should not
// literally render as a caption and the LLM should not restate. See design
// doc D2/D5.
func (h *aicallHandler) toolHandleEmitInfoCard(ctx context.Context, c *aicall.AIcall, tool *message.ToolCall) *messageContent {
	log := logrus.WithFields(logrus.Fields{
		"func":      "toolHandleEmitInfoCard",
		"aicall_id": c.ID,
	})
	log.Debugf("handling tool emit_info_card.")

	res := newToolResult(tool.ID)

	var args struct {
		Title       string      `json:"title"`
		Description string      `json:"description,omitempty"`
		Fields      []CardField `json:"fields,omitempty"`
	}
	if errUnmarshal := json.Unmarshal([]byte(tool.Function.Arguments), &args); errUnmarshal != nil {
		fillFailed(res, errUnmarshal)
		return res
	}

	if strings.TrimSpace(args.Title) == "" {
		fillFailed(res, fmt.Errorf("title is required"))
		return res
	}

	title := truncateCardText(args.Title, cardTitleMaxLen)
	description := truncateCardText(args.Description, cardDescriptionMaxLen)

	fields := args.Fields
	if len(fields) > cardFieldsMaxCount {
		fields = fields[:cardFieldsMaxCount]
	}
	truncatedFields := make([]CardField, len(fields))
	for i, f := range fields {
		truncatedFields[i] = CardField{
			Label: truncateCardText(f.Label, cardFieldLabelMaxLen),
			Value: truncateCardText(f.Value, cardFieldValueMaxLen),
		}
	}

	res.Blocks = []CardBlock{
		{
			Type:        "info",
			Title:       title,
			Description: description,
			Fields:      truncatedFields,
		},
	}

	// resource_type "card" / resource_id "" follows the actual codebase
	// precedent (every real fillSuccess call site passes a non-empty
	// resource_type; empty resource_id has precedent in case_create) -- there
	// is no addressable resource id for a card the way a case/call/contact
	// has one. See design doc D2 (round 7 correction).
	fillSuccess(res, "card", "", fmt.Sprintf("Displayed an info card titled '%s'.", title))

	log.Debugf("Created info card. title: %s, field_count: %d", title, len(truncatedFields))

	return res
}
