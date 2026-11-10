package aicallhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"

	"github.com/sirupsen/logrus"
)

// draft_message limits. The text limit is enforced by rejecting, never by
// silently cutting, because what the agent approves is exactly what is sent.
const (
	draftTextMaxRunes = 1600
	draftTitleDefault = "SMS draft"
)

// draftToRegex is the E.164 form a draft recipient must have.
var draftToRegex = regexp.MustCompile(`^\+[1-9][0-9]{6,14}$`)

// toolHandleDraftMessage handles the draft_message tool. It records an
// "sms_draft" card block in the session's own message stream and makes no
// external call: nothing is sent until the agent approves the card through
// the draft-decision path. The id of the stored tool result row is the draft
// id.
//
// The result message never contains the draft body, for the same reason the
// info card message does not contain field values.
func (h *aicallHandler) toolHandleDraftMessage(ctx context.Context, c *aicall.AIcall, tool *message.ToolCall) *messageContent {
	log := logrus.WithFields(logrus.Fields{
		"func":      "toolHandleDraftMessage",
		"aicall_id": c.ID,
	})
	log.Debugf("handling tool draft_message.")

	res := newToolResult(tool.ID)

	var args struct {
		To    string `json:"to"`
		Text  string `json:"text"`
		Title string `json:"title"`
	}
	if errUnmarshal := json.Unmarshal([]byte(tool.Function.Arguments), &args); errUnmarshal != nil {
		fillFailed(res, errUnmarshal)
		return res
	}

	to := strings.TrimSpace(args.To)
	if !draftToRegex.MatchString(to) {
		fillFailed(res, fmt.Errorf("to must be a phone number in E.164 format, for example +14155550100"))
		return res
	}

	if strings.TrimSpace(args.Text) == "" {
		fillFailed(res, fmt.Errorf("text is required"))
		return res
	}
	if n := len([]rune(args.Text)); n > draftTextMaxRunes {
		fillFailed(res, fmt.Errorf("text is too long: %d characters, the limit is %d. Shorten it and draft again", n, draftTextMaxRunes))
		return res
	}

	title := strings.TrimSpace(args.Title)
	if title == "" {
		title = draftTitleDefault
	}
	title = truncateCardText(title, cardTitleMaxLen)

	res.Blocks = []CardBlock{
		{
			Type:  message.BlockTypeSMSDraft,
			Title: title,
			To:    to,
			Text:  args.Text,
		},
	}
	fillSuccess(res, "card", "", "Displayed an SMS draft for review.")

	log.Debugf("Created sms draft. title_len: %d, text_len: %d", len(title), len([]rune(args.Text)))

	return res
}
