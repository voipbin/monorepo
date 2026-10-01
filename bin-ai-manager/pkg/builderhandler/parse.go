package builderhandler

import (
	"bytes"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"unicode/utf8"

	"monorepo/bin-ai-manager/models/builder"
	"monorepo/bin-ai-manager/models/tool"
)

// Sentinel errors produced by this package. RunTurn returns them and the
// handler (Chat) maps them to reasons and metric labels. None of them carries
// model output or caller input.
var (
	ErrInvalidResponse = errors.New("builder: no usable response object")
	ErrTruncated       = errors.New("builder: response truncated")
	ErrTimeout         = errors.New("builder: llm deadline exceeded")
	ErrLLM             = errors.New("builder: llm error")
)

// Warning strings recorded in draft_warnings. They are facts the code
// established, never model prose. The client may key on the prefix before ": ".
const (
	WarnDraftDiscarded         = "draft_discarded"
	WarnToolNamesInvalid       = "tool_names_invalid"
	WarnToolRemoved            = "tool_removed"
	WarnToolsSectionRemoved    = "tools_section_removed"
	WarnForbiddenToolMentioned = "forbidden_tool_mentioned"
	WarnInitPromptTruncated    = "init_prompt_truncated"
	WarnNameTruncated          = "name_truncated"
	WarnDetailTruncated        = "detail_truncated"
)

// maxObjectAttempts is how many '{' positions Parse tries.
const maxObjectAttempts = 5

// ParsedResponse is the validated result of one model turn.
type ParsedResponse struct {
	Message     string
	Draft       *builder.Draft
	Assumptions []string
	Warnings    []string
}

// Parse extracts and validates the model's JSON response (design 4.4).
//
// The provider endpoint does not enforce the schema strictly, so the model may
// wrap the object in a code fence or prose. Parse scans up to five '{'
// positions, decodes one object from each with json.Decoder and adopts the
// first whose "message" is a non-empty string. Fields are then decoded
// independently so a field of the wrong type is dropped without losing the
// response. Server behaviour never depends on the message prose.
func Parse(raw string) (*ParsedResponse, error) {
	fields, ok := firstUsableObject(raw)
	if !ok {
		return nil, ErrInvalidResponse
	}

	out := &ParsedResponse{}
	_ = json.Unmarshal(fields["message"], &out.Message)

	draftRaw, hasDraft := fields["draft"]
	if !hasDraft || isJSONNull(draftRaw) {
		return out, nil
	}

	draft, toolsInvalid := decodeDraft(draftRaw)
	if draft == nil {
		out.discardDraft()
		return out, nil
	}
	if toolsInvalid {
		out.Warnings = append(out.Warnings, WarnToolNamesInvalid)
	}

	// Order: decode, required check, section removal, re-check, truncation.
	if draft.Name == "" || draft.InitPrompt == "" {
		out.discardDraft()
		return out, nil
	}

	stripped, removed := removeToolsSection(draft.InitPrompt)
	if removed {
		out.Warnings = append(out.Warnings, WarnToolsSectionRemoved)
		draft.InitPrompt = stripped
		if strings.TrimSpace(draft.InitPrompt) == "" {
			out.discardDraft()
			return out, nil
		}
	}

	draft.ToolNames = filterTools(draft.ToolNames, &out.Warnings)

	var cut bool
	if draft.InitPrompt, cut = truncateRunes(draft.InitPrompt, builder.MaxInitPromptRunes); cut {
		out.Warnings = append(out.Warnings, WarnInitPromptTruncated)
	}
	if draft.Name, cut = truncateRunes(draft.Name, builder.MaxNameRunes); cut {
		out.Warnings = append(out.Warnings, WarnNameTruncated)
	}
	if draft.Detail, cut = truncateRunes(draft.Detail, builder.MaxDetailRunes); cut {
		out.Warnings = append(out.Warnings, WarnDetailTruncated)
	}

	out.Warnings = append(out.Warnings, forbiddenToolMentions(draft.InitPrompt)...)

	out.Draft = draft
	out.Assumptions = []string{}
	if ar, ok := fields["assumptions"]; ok {
		var a []string
		if err := json.Unmarshal(ar, &a); err == nil && a != nil {
			out.Assumptions = a
		}
	}
	return out, nil
}

// discardDraft drops the draft. Assumptions are only ever filled after a draft
// has been accepted (see Parse), so a discarded draft never carries any; the
// tests pin that invariant.
func (p *ParsedResponse) discardDraft() {
	p.Draft = nil
	p.Warnings = append(p.Warnings, WarnDraftDiscarded)
}

func isJSONNull(b json.RawMessage) bool {
	return bytes.Equal(bytes.TrimSpace(b), []byte("null"))
}

// firstUsableObject returns the field map of the first decodable object whose
// "message" is a non-empty string.
func firstUsableObject(raw string) (map[string]json.RawMessage, bool) {
	attempts := 0
	for i := 0; i < len(raw) && attempts < maxObjectAttempts; i++ {
		if raw[i] != '{' {
			continue
		}
		attempts++
		dec := json.NewDecoder(strings.NewReader(raw[i:]))
		var fields map[string]json.RawMessage
		if err := dec.Decode(&fields); err != nil {
			continue
		}
		mr, ok := fields["message"]
		if !ok {
			continue
		}
		var msg string
		if err := json.Unmarshal(mr, &msg); err != nil || msg == "" {
			continue
		}
		return fields, true
	}
	return nil, false
}

// decodeDraft decodes the draft object field by field. A field of the wrong
// type is dropped alone. It returns nil only when raw is not an object.
func decodeDraft(raw json.RawMessage) (d *builder.Draft, toolNamesInvalid bool) {
	var f map[string]json.RawMessage
	if err := json.Unmarshal(raw, &f); err != nil || f == nil {
		return nil, false
	}
	d = &builder.Draft{ToolNames: []string{}}
	_ = json.Unmarshal(f["name"], &d.Name)
	_ = json.Unmarshal(f["detail"], &d.Detail)
	_ = json.Unmarshal(f["init_prompt"], &d.InitPrompt)
	if tr, ok := f["tool_names"]; ok && !isJSONNull(tr) {
		var names []string
		if err := json.Unmarshal(tr, &names); err != nil {
			toolNamesInvalid = true
		} else {
			d.ToolNames = names
		}
	}
	return d, toolNamesInvalid
}

// filterTools keeps only allow-listed tools, de-duplicated, in first-seen
// order, and records each removed name.
func filterTools(names []string, warnings *[]string) []string {
	kept := make([]string, 0, len(names))
	seen := map[string]bool{}
	removed := map[string]bool{}
	for _, n := range names {
		if builder.IsAllowedTool(n) {
			if !seen[n] {
				seen[n] = true
				kept = append(kept, n)
			}
			continue
		}
		if !removed[n] {
			removed[n] = true
			*warnings = append(*warnings, WarnToolRemoved+": "+n)
		}
	}
	return kept
}

func truncateRunes(s string, max int) (string, bool) {
	if utf8.RuneCountInString(s) <= max {
		return s, false
	}
	r := []rune(s)
	return string(r[:max]), true
}

var headerRE = regexp.MustCompile(`^(#{1,6})[ \t]+(.*?)[ \t]*$`)

// removeToolsSection deletes every "Tools & Capabilities" section. The frontend
// builds that section from its own label table, so a model-written copy would
// duplicate it. A section runs from its header to the next header of the same
// or a higher level, or to the end of the document.
func removeToolsSection(s string) (string, bool) {
	lines := strings.Split(s, "\n")
	kept := make([]string, 0, len(lines))
	removed := false
	skipLevel := 0
	for _, ln := range lines {
		m := headerRE.FindStringSubmatch(strings.TrimRight(ln, "\r"))
		if skipLevel > 0 {
			if m != nil && len(m[1]) <= skipLevel {
				skipLevel = 0
			} else {
				continue
			}
		}
		if m != nil && isToolsHeader(m[2]) && len(m[1]) >= 2 {
			skipLevel = len(m[1])
			removed = true
			continue
		}
		kept = append(kept, ln)
	}
	if !removed {
		return s, false
	}
	return strings.TrimRight(strings.Join(kept, "\n"), "\n \t") + "\n", true
}

func isToolsHeader(title string) bool {
	return strings.EqualFold(strings.Join(strings.Fields(title), " "), "tools & capabilities")
}

// forbiddenToolMentions lists tool identifiers that exist in the product but
// are not on the allow-list and appear in init_prompt as whole identifiers.
// Matching is on word boundaries because stop_service (allowed) and stop_flow
// (not allowed) share a prefix. It informs the user; it is not a security
// control.
func forbiddenToolMentions(initPrompt string) []string {
	allowed := map[tool.ToolName]bool{}
	for _, t := range builder.AllowedTools {
		allowed[t] = true
	}
	var out []string
	for _, t := range tool.AllToolNames {
		if allowed[t] {
			continue
		}
		name := string(t)
		if containsIdentifier(initPrompt, name) {
			out = append(out, WarnForbiddenToolMentioned+": "+name)
		}
	}
	return out
}

// containsIdentifier reports whether name occurs in s bounded by non-identifier
// characters on both sides.
func containsIdentifier(s, name string) bool {
	for from := 0; ; {
		i := strings.Index(s[from:], name)
		if i < 0 {
			return false
		}
		start := from + i
		end := start + len(name)
		if (start == 0 || !isIdentByte(s[start-1])) && (end == len(s) || !isIdentByte(s[end])) {
			return true
		}
		from = start + 1
	}
}

func isIdentByte(b byte) bool {
	return b == '_' || (b >= '0' && b <= '9') || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}
