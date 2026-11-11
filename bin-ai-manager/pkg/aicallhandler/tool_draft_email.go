package aicallhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/sirupsen/logrus"
)

// draft_email limits (docs/plans/2026-11-11-insight-assistant-email-design.md,
// section 3). Every limit is enforced by rejecting, never by silently cutting,
// because what the agent approves is exactly what is sent. The card title is the
// one cosmetic cut, as for the SMS card.
const (
	draftEmailMaxRecipients     = 10
	draftEmailSubjectMaxRunes   = 200
	draftEmailContentMaxRunes   = 10000
	draftEmailAddressMaxLen     = 254
	draftEmailLocalPartMaxLen   = 64
	draftEmailDomainMaxLen      = 253
	draftEmailDomainLabelMaxLen = 63
	draftEmailTitleDefault      = "Email draft"
	draftEmailDisplayedMessage  = "Displayed an email draft for review."
	draftEmailAcceptedMessage   = "Email accepted for sending. Delivery is not confirmed."
	draftEmailRejectedMessage   = "The agent rejected this draft. The email was not sent."
	draftEmailInvalidMessage    = "The draft recipients, subject or content are not valid."
	draftEmailAddressMinLabels  = 2
)

// draftStoredMaxBytes is the largest size of the JSON of a draft row content.
// ai_messages.content is a text column (65,535 bytes) and json.Marshal writes
// characters such as "<", ">", "&", U+2028 and U+2029 as six bytes, so content
// within the rune limit can be much larger once stored. The guard keeps a margin
// and turns a failing insert into a refusal before anything is recorded.
const draftStoredMaxBytes = 60000

var (
	// draftEmailLabelRegex is one label of a domain name (already lower case).
	draftEmailLabelRegex = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

	// draftEmailTLDRegex is a last label made of letters only.
	draftEmailTLDRegex = regexp.MustCompile(`^[a-z]{2,}$`)

	// draftEmailUnsupportedKeys are arguments of other email tools that draft_email
	// does not support. A key that is present is refused, whatever its value, so
	// the assistant cannot report a copy or an attachment that was not included.
	draftEmailUnsupportedKeys = []string{"cc", "bcc", "attachments", "source", "from", "reply_to"}
)

// toolHandleDraftEmail handles the draft_email tool. It records an
// "email_draft" card block in the session's own message stream and makes no
// external call: nothing is sent until the agent approves the card through
// the draft-decision path. The id of the stored tool result row is the draft
// id.
//
// The result message never contains the draft body. Logs carry lengths only,
// never an address, a subject or a text.
func (h *aicallHandler) toolHandleDraftEmail(ctx context.Context, c *aicall.AIcall, tool *message.ToolCall) *messageContent {
	log := logrus.WithFields(logrus.Fields{
		"func":      "toolHandleDraftEmail",
		"aicall_id": c.ID,
	})
	log.Debugf("handling tool draft_email.")

	res := newToolResult(tool.ID)

	var args struct {
		To      []string `json:"to"`
		Subject string   `json:"subject"`
		Content string   `json:"content"`
		Title   string   `json:"title"`
	}
	if errUnmarshal := json.Unmarshal([]byte(tool.Function.Arguments), &args); errUnmarshal != nil {
		fillFailed(res, errUnmarshal)
		return res
	}

	// the struct decode matches keys case-insensitively, so the refusal does too.
	var keys map[string]json.RawMessage
	if errUnmarshal := json.Unmarshal([]byte(tool.Function.Arguments), &keys); errUnmarshal != nil {
		fillFailed(res, errUnmarshal)
		return res
	}
	for key := range keys {
		for _, unsupported := range draftEmailUnsupportedKeys {
			if strings.EqualFold(key, unsupported) {
				fillFailed(res, fmt.Errorf("%s is not supported by draft_email: put every recipient in to, and attachments and a sender are not available", strings.ToLower(key)))
				return res
			}
		}
	}

	recipients, err := validateEmailDraft(args.To, args.Subject, args.Content)
	if err != nil {
		fillFailed(res, err)
		return res
	}

	title := strings.TrimSpace(args.Title)
	if title == "" {
		title = draftEmailTitleDefault
	}
	title = truncateCardText(title, cardTitleMaxLen)

	res.Blocks = []CardBlock{
		{
			Type:       message.BlockTypeEmailDraft,
			Title:      title,
			Recipients: recipients,
			Subject:    args.Subject,
			Content:    args.Content,
		},
	}
	fillSuccess(res, "card", "", draftEmailDisplayedMessage)

	stored, errMarshal := json.Marshal(res)
	if errMarshal != nil {
		res = newToolResult(tool.ID)
		fillFailed(res, errMarshal)
		return res
	}
	if len(stored) > draftStoredMaxBytes {
		res = newToolResult(tool.ID)
		fillFailed(res, fmt.Errorf("the draft is too large to store: %d bytes after encoding, the limit is %d. Shorten the content and draft again", len(stored), draftStoredMaxBytes))
		return res
	}

	log.Debugf("Created email draft. recipients: %d, title_len: %d, subject_len: %d, content_len: %d", len(recipients), len([]rune(title)), len([]rune(args.Subject)), len([]rune(args.Content)))

	return res
}

// validateEmailDraft checks the recipients, the subject and the content of an
// email draft and returns the normalized recipients (trimmed, lower case, in the
// order given). It refuses, it never cuts. The same function runs when the draft
// is created and when it is approved, so a stored draft is decided by the rules
// it was created under.
func validateEmailDraft(to []string, subject, content string) ([]string, error) {
	if len(to) == 0 {
		return nil, fmt.Errorf("to is required: give 1 to %d recipient addresses", draftEmailMaxRecipients)
	}
	if len(to) > draftEmailMaxRecipients {
		return nil, fmt.Errorf("to has %d recipients, the limit is %d", len(to), draftEmailMaxRecipients)
	}

	recipients := make([]string, 0, len(to))
	seen := make(map[string]int, len(to))
	for i, raw := range to {
		normalized, err := normalizeDraftEmailAddress(raw)
		if err != nil {
			return nil, fmt.Errorf("to[%d] is not a valid email address: %w", i, err)
		}
		if first, ok := seen[normalized]; ok {
			return nil, fmt.Errorf("to[%d] repeats to[%d]", i, first)
		}
		seen[normalized] = i
		recipients = append(recipients, normalized)
	}

	if err := validateDraftText("subject", subject, draftEmailSubjectMaxRunes, false); err != nil {
		return nil, err
	}
	if err := validateDraftText("content", content, draftEmailContentMaxRunes, true); err != nil {
		return nil, err
	}

	return recipients, nil
}

// normalizeDraftEmailAddress applies the rule of bin-email-manager
// (commonaddress.NormalizeTarget, then mail.ParseAddress) and makes it stricter:
// the parsed address must equal the normalized value (so a display name, a
// comment or a quoted local part is refused) and the address is ASCII only with
// the length limits of RFC 5321 and the DNS. bin-email-manager's own check is an
// unexported method of another service, so the two calls are repeated here.
func normalizeDraftEmailAddress(raw string) (string, error) {
	normalized, _ := commonaddress.NormalizeTarget(commonaddress.TypeEmail, raw)

	if normalized == "" {
		return "", fmt.Errorf("it is empty")
	}
	if len(normalized) > draftEmailAddressMaxLen {
		return "", fmt.Errorf("it is longer than %d characters", draftEmailAddressMaxLen)
	}
	for _, r := range normalized {
		if r < 0x21 || r > 0x7e {
			return "", fmt.Errorf("only plain ASCII characters without spaces are allowed")
		}
	}

	parsed, err := mail.ParseAddress(normalized)
	if err != nil {
		return "", fmt.Errorf("the address is malformed")
	}
	if parsed.Address != normalized {
		return "", fmt.Errorf("give a plain address without a display name, comment or quotes")
	}

	at := strings.LastIndex(normalized, "@")
	if at <= 0 || at == len(normalized)-1 {
		return "", fmt.Errorf("the address is malformed")
	}
	local, domain := normalized[:at], normalized[at+1:]
	if len(local) > draftEmailLocalPartMaxLen {
		return "", fmt.Errorf("the part before @ is longer than %d characters", draftEmailLocalPartMaxLen)
	}
	if len(domain) > draftEmailDomainMaxLen {
		return "", fmt.Errorf("the domain is longer than %d characters", draftEmailDomainMaxLen)
	}

	labels := strings.Split(domain, ".")
	if len(labels) < draftEmailAddressMinLabels {
		return "", fmt.Errorf("the domain needs at least two labels")
	}
	for _, label := range labels {
		if len(label) > draftEmailDomainLabelMaxLen || !draftEmailLabelRegex.MatchString(label) {
			return "", fmt.Errorf("the domain is not valid")
		}
	}
	last := labels[len(labels)-1]
	if !draftEmailTLDRegex.MatchString(last) && !strings.HasPrefix(last, "xn--") {
		return "", fmt.Errorf("the last label of the domain must be letters only")
	}

	return normalized, nil
}

// validateDraftText checks the subject or the content of an email draft: valid
// UTF-8 without U+FFFD (the JSON decoder turns invalid UTF-8 and a lone
// surrogate escape into U+FFFD, which would be a silent substitution), not
// blank, within maxRunes, and free of control, line and paragraph separator and
// bidirectional control characters. Line feed, carriage return and tab are
// allowed when allowLineBreaks is set.
func validateDraftText(name, value string, maxRunes int, allowLineBreaks bool) error {
	if !utf8.ValidString(value) {
		return fmt.Errorf("%s is not valid UTF-8", name)
	}
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s is required", name)
	}
	if n := utf8.RuneCountInString(value); n > maxRunes {
		return fmt.Errorf("%s is too long: %d characters, the limit is %d. Shorten it and draft again", name, n, maxRunes)
	}

	for _, r := range value {
		switch {
		case r == utf8.RuneError:
			return fmt.Errorf("%s contains the replacement character U+FFFD", name)
		case allowLineBreaks && (r == '\n' || r == '\r' || r == '\t'):
			continue
		case unicode.Is(unicode.Cc, r), unicode.Is(unicode.Zl, r), unicode.Is(unicode.Zp, r), isBidiControl(r):
			return fmt.Errorf("%s contains a control or direction character (U+%04X)", name, r)
		}
	}

	return nil
}

// isBidiControl reports whether r is a bidirectional control character that can
// change how a text is displayed.
func isBidiControl(r rune) bool {
	switch {
	case r == 0x061C, r == 0x200E, r == 0x200F:
		return true
	case r >= 0x202A && r <= 0x202E:
		return true
	case r >= 0x2066 && r <= 0x2069:
		return true
	}
	return false
}
