package builderhandler

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"monorepo/bin-ai-manager/models/builder"
)

func rep(s string, n int) string { return strings.Repeat(s, n) }

const okDraft = `"draft":{"name":"Clinic Bot","detail":"books visits","init_prompt":"# Clinic Bot\n\n## Identity & Purpose\nHelp callers.","tool_names":["connect_call","send_message"]},"assumptions":["callers speak Korean"]`

func Test_Parse_normal(t *testing.T) {
	p, err := Parse(`{"message":"어떤 채널인가요?"}`)
	if err != nil {
		t.Fatal(err)
	}
	if p.Message != "어떤 채널인가요?" || p.Draft != nil || len(p.Warnings) != 0 {
		t.Fatalf("unexpected: %+v", p)
	}

	p, err = Parse(`{"message":"초안입니다", ` + okDraft + `}`)
	if err != nil {
		t.Fatal(err)
	}
	if p.Draft == nil || p.Draft.Name != "Clinic Bot" || len(p.Draft.ToolNames) != 2 {
		t.Fatalf("draft missing: %+v", p)
	}
	if len(p.Assumptions) != 1 {
		t.Fatalf("assumptions: %+v", p.Assumptions)
	}
}

func Test_Parse_codefenceAndSurroundingText(t *testing.T) {
	for name, raw := range map[string]string{
		"fence":      "```json\n{\"message\":\"hi\"}\n```",
		"prefix":     "Sure, here you go: {\"message\":\"hi\"} thanks",
		"braces":     "use {placeholders} like this {\"x\":1}\n{\"message\":\"hi\"}",
		"othervalid": `{"foo":"bar"} {"message":"hi"}`,
	} {
		t.Run(name, func(t *testing.T) {
			p, err := Parse(raw)
			if err != nil || p.Message != "hi" {
				t.Fatalf("err=%v p=%+v", err, p)
			}
		})
	}
}

func Test_Parse_noUsableObject(t *testing.T) {
	for name, raw := range map[string]string{
		"empty":          "",
		"not json":       "I cannot do that",
		"empty object":   "{}",
		"empty message":  `{"message":""}`,
		"message number": `{"message":42}`,
		"message null":   `{"message":null}`,
		"truncated":      `{"message":"abc`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := Parse(raw); !errors.Is(err, ErrInvalidResponse) {
				t.Fatalf("want ErrInvalidResponse, got %v", err)
			}
		})
	}
}

// A leading object with an empty message must not stop the scan.
func Test_Parse_emptyMessageObjectThenValid(t *testing.T) {
	p, err := Parse(`{"message":""} {"message":"real"}`)
	if err != nil || p.Message != "real" {
		t.Fatalf("err=%v p=%+v", err, p)
	}
}

// The scan tries at most 5 '{' positions.
func Test_Parse_fiveBraceLimit(t *testing.T) {
	junk5 := rep("{x ", 5)
	if _, err := Parse(junk5 + `{"message":"late"}`); !errors.Is(err, ErrInvalidResponse) {
		t.Fatalf("object after 5 junk braces must not be reached, got %v", err)
	}
	junk4 := rep("{x ", 4)
	if p, err := Parse(junk4 + `{"message":"ok"}`); err != nil || p.Message != "ok" {
		t.Fatalf("5th brace must be tried: err=%v p=%+v", err, p)
	}
}

func Test_Parse_typeMismatch(t *testing.T) {
	// assumptions wrong type is dropped; the response survives.
	p, err := Parse(`{"message":"hi","draft":{"name":"A","detail":"d","init_prompt":"# A\nbody","tool_names":["connect_call"]},"assumptions":"oops"}`)
	if err != nil {
		t.Fatal(err)
	}
	if p.Draft == nil || len(p.Assumptions) != 0 {
		t.Fatalf("unexpected: %+v", p)
	}

	// nested tool_names type mismatch drops that field only, not the draft.
	p, err = Parse(`{"message":"hi","draft":{"name":"A","detail":"d","init_prompt":"# A\nbody","tool_names":"connect_call"}}`)
	if err != nil {
		t.Fatal(err)
	}
	if p.Draft == nil || len(p.Draft.ToolNames) != 0 {
		t.Fatalf("draft must survive a bad tool_names: %+v", p)
	}
	if !hasWarning(p.Warnings, WarnToolNamesInvalid) {
		t.Fatalf("want %q warning, got %v", WarnToolNamesInvalid, p.Warnings)
	}
}

func Test_Parse_unknownFieldsIgnored(t *testing.T) {
	p, err := Parse(`{"message":"hi","captured":["x"],"suggested_replies":["a"]}`)
	if err != nil || p.Message != "hi" {
		t.Fatalf("err=%v p=%+v", err, p)
	}
}

func Test_Parse_draftRequiresNameAndInitPrompt(t *testing.T) {
	for name, d := range map[string]string{
		"no name":        `{"name":"","detail":"d","init_prompt":"# A\nbody","tool_names":[]}`,
		"no init_prompt": `{"name":"A","detail":"d","init_prompt":"","tool_names":[]}`,
		"missing fields": `{"detail":"only"}`,
	} {
		t.Run(name, func(t *testing.T) {
			p, err := Parse(`{"message":"hi","draft":` + d + `,"assumptions":["a"]}`)
			if err != nil {
				t.Fatal(err)
			}
			if p.Draft != nil {
				t.Fatalf("draft must be discarded: %+v", p.Draft)
			}
			// assumptions travel with the draft.
			if len(p.Assumptions) != 0 {
				t.Fatalf("assumptions must be discarded with the draft: %v", p.Assumptions)
			}
			if !hasWarning(p.Warnings, WarnDraftDiscarded) {
				t.Fatalf("want %q warning, got %v", WarnDraftDiscarded, p.Warnings)
			}
		})
	}
}

func Test_Parse_assumptionsDefaultsToEmpty(t *testing.T) {
	p, err := Parse(`{"message":"hi","draft":{"name":"A","detail":"d","init_prompt":"# A\nbody","tool_names":[]}}`)
	if err != nil {
		t.Fatal(err)
	}
	if p.Draft == nil || p.Assumptions == nil || len(p.Assumptions) != 0 {
		t.Fatalf("a draft without assumptions must carry an empty (non-nil) slice: %+v", p)
	}
}

func Test_Parse_toolIntersection(t *testing.T) {
	p, err := Parse(`{"message":"hi","draft":{"name":"A","detail":"d","init_prompt":"# A\nbody","tool_names":["connect_call","create_call","join_queue","send_email","connect_call"]}}`)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(p.Draft.ToolNames, ",")
	if got != "connect_call,send_email" {
		t.Fatalf("want allow-listed unique tools in order, got %q", got)
	}
	if !hasWarning(p.Warnings, WarnToolRemoved+": create_call") || !hasWarning(p.Warnings, WarnToolRemoved+": join_queue") {
		t.Fatalf("removed tools must be recorded: %v", p.Warnings)
	}
}

func Test_Parse_toolsSectionRemoval(t *testing.T) {
	body := "# A\n\n## Identity\nhelp\n\n%s\n- connect_call: transfer\n\n## Response Guidelines\nbe brief\n"
	cases := map[string]string{
		"h2":     "## Tools & Capabilities",
		"h3":     "### Tools & Capabilities",
		"case":   "## TOOLS & CAPABILITIES",
		"spaces": "##   tools & capabilities  ",
	}
	for name, header := range cases {
		t.Run(name, func(t *testing.T) {
			ip := strings.Replace(body, "%s", header, 1)
			p := mustParseDraft(t, ip)
			if strings.Contains(strings.ToLower(p.Draft.InitPrompt), "tools & capabilities") {
				t.Fatalf("section must be removed:\n%s", p.Draft.InitPrompt)
			}
			if !strings.Contains(p.Draft.InitPrompt, "## Response Guidelines") {
				t.Fatalf("the following same-level section must be kept:\n%s", p.Draft.InitPrompt)
			}
			if strings.Contains(p.Draft.InitPrompt, "connect_call: transfer") {
				t.Fatalf("section body must be removed:\n%s", p.Draft.InitPrompt)
			}
			if !hasWarning(p.Warnings, WarnToolsSectionRemoved) {
				t.Fatalf("want %q warning: %v", WarnToolsSectionRemoved, p.Warnings)
			}
		})
	}

	t.Run("to end of document", func(t *testing.T) {
		ip := "# A\n\n## Identity\nhelp\n\n## Tools & Capabilities\n- connect_call\n"
		p := mustParseDraft(t, ip)
		if strings.Contains(p.Draft.InitPrompt, "connect_call") || !strings.Contains(p.Draft.InitPrompt, "## Identity") {
			t.Fatalf("got:\n%s", p.Draft.InitPrompt)
		}
	})

	t.Run("duplicates", func(t *testing.T) {
		ip := "# A\n\n## Tools & Capabilities\n- one\n\n## Mid\nkeep\n\n## Tools & Capabilities\n- two\n"
		p := mustParseDraft(t, ip)
		if strings.Contains(p.Draft.InitPrompt, "one") || strings.Contains(p.Draft.InitPrompt, "two") || !strings.Contains(p.Draft.InitPrompt, "## Mid") {
			t.Fatalf("got:\n%s", p.Draft.InitPrompt)
		}
	})

	t.Run("deeper subsections of the removed section go too", func(t *testing.T) {
		ip := "# A\n\n## Tools & Capabilities\nintro\n### Sub\ndetail\n\n## After\nkeep\n"
		p := mustParseDraft(t, ip)
		if strings.Contains(p.Draft.InitPrompt, "detail") || !strings.Contains(p.Draft.InitPrompt, "## After") {
			t.Fatalf("got:\n%s", p.Draft.InitPrompt)
		}
	})

	t.Run("h3 header ends at the next h3 or higher", func(t *testing.T) {
		ip := "# A\n\n## Parent\n### Tools & Capabilities\n- x\n#### deeper\nstill removed\n### Sibling\nkeep\n"
		p := mustParseDraft(t, ip)
		if strings.Contains(p.Draft.InitPrompt, "still removed") || !strings.Contains(p.Draft.InitPrompt, "### Sibling") {
			t.Fatalf("got:\n%s", p.Draft.InitPrompt)
		}
	})

	t.Run("empty after removal discards the draft", func(t *testing.T) {
		ip := "## Tools & Capabilities\n- connect_call\n"
		p, err := Parse(`{"message":"hi","draft":{"name":"A","detail":"d","init_prompt":` + jsonString(ip) + `,"tool_names":[]},"assumptions":["a"]}`)
		if err != nil {
			t.Fatal(err)
		}
		if p.Draft != nil || len(p.Assumptions) != 0 || !hasWarning(p.Warnings, WarnDraftDiscarded) {
			t.Fatalf("got: %+v", p)
		}
	})

	t.Run("a header that merely mentions the words is not a section", func(t *testing.T) {
		ip := "# A\n\n## Tools & Capabilities Overview\nkeep this\n"
		p := mustParseDraft(t, ip)
		if !strings.Contains(p.Draft.InitPrompt, "keep this") {
			t.Fatalf("different header must not be removed:\n%s", p.Draft.InitPrompt)
		}
	})
}

func Test_Parse_wordBoundaryForbiddenTools(t *testing.T) {
	// stop_service is allowed, stop_flow is not. A prefix match must not fire.
	p := mustParseDraft(t, "# A\nCall stop_service when done.\n")
	if hasWarningPrefix(p.Warnings, WarnForbiddenToolMentioned) {
		t.Fatalf("stop_service must not trigger a stop_flow warning: %v", p.Warnings)
	}
	p = mustParseDraft(t, "# A\nCall stop_flow when done.\n")
	if !hasWarning(p.Warnings, WarnForbiddenToolMentioned+": stop_flow") {
		t.Fatalf("stop_flow mention must be flagged: %v", p.Warnings)
	}
	p = mustParseDraft(t, "# A\nuse create_call, then join_queue.\n")
	if !hasWarning(p.Warnings, WarnForbiddenToolMentioned+": create_call") || !hasWarning(p.Warnings, WarnForbiddenToolMentioned+": join_queue") {
		t.Fatalf("got %v", p.Warnings)
	}
	// substring inside a longer identifier is not a word match.
	p = mustParseDraft(t, "# A\nmy_create_caller is not a tool.\n")
	if hasWarningPrefix(p.Warnings, WarnForbiddenToolMentioned) {
		t.Fatalf("got %v", p.Warnings)
	}
}

func Test_Parse_lengthTruncationByRune(t *testing.T) {
	long := "# A\n" + rep("가", builder.MaxInitPromptRunes+50)
	p := mustParseDraft(t, long)
	if n := utf8.RuneCountInString(p.Draft.InitPrompt); n != builder.MaxInitPromptRunes {
		t.Fatalf("want %d runes, got %d", builder.MaxInitPromptRunes, n)
	}
	if !utf8.ValidString(p.Draft.InitPrompt) {
		t.Fatal("truncation must not split a rune")
	}
	if !hasWarning(p.Warnings, WarnInitPromptTruncated) {
		t.Fatalf("got %v", p.Warnings)
	}

	d, err := Parse(`{"message":"hi","draft":{"name":` + jsonString(rep("n", 150)) + `,"detail":` + jsonString(rep("d", 600)) + `,"init_prompt":"# A\nbody","tool_names":[]}}`)
	if err != nil {
		t.Fatal(err)
	}
	if utf8.RuneCountInString(d.Draft.Name) != builder.MaxNameRunes || utf8.RuneCountInString(d.Draft.Detail) != builder.MaxDetailRunes {
		t.Fatalf("name/detail must be truncated: %d %d", utf8.RuneCountInString(d.Draft.Name), utf8.RuneCountInString(d.Draft.Detail))
	}
}

func Test_Parse_doesNotEchoRawIntoErrors(t *testing.T) {
	_, err := Parse("SECRET-입력-문자열 not json")
	if err == nil || strings.Contains(err.Error(), "SECRET") {
		t.Fatalf("error must not echo model output: %v", err)
	}
}

// ---- helpers ----

func mustParseDraft(t *testing.T, initPrompt string) *ParsedResponse {
	t.Helper()
	p, err := Parse(`{"message":"hi","draft":{"name":"A","detail":"d","init_prompt":` + jsonString(initPrompt) + `,"tool_names":[]}}`)
	if err != nil {
		t.Fatal(err)
	}
	if p.Draft == nil {
		t.Fatalf("draft unexpectedly discarded; warnings=%v", p.Warnings)
	}
	return p
}

func jsonString(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

func hasWarning(ws []string, want string) bool {
	for _, w := range ws {
		if w == want {
			return true
		}
	}
	return false
}

func hasWarningPrefix(ws []string, prefix string) bool {
	for _, w := range ws {
		if strings.HasPrefix(w, prefix) {
			return true
		}
	}
	return false
}

// An H1 titled like the section is the document title, not a tools section.
func Test_Parse_h1ToolsTitleIsKept(t *testing.T) {
	ip := "# Tools & Capabilities\n\n## Identity\nhelp\n"
	p := mustParseDraft(t, ip)
	if !strings.Contains(p.Draft.InitPrompt, "# Tools & Capabilities") || !strings.Contains(p.Draft.InitPrompt, "## Identity") {
		t.Fatalf("an H1 must not be treated as the tools section:\n%s", p.Draft.InitPrompt)
	}
	if hasWarning(p.Warnings, WarnToolsSectionRemoved) {
		t.Fatalf("no removal expected: %v", p.Warnings)
	}
}

// Assumptions belong to the draft. When tool-section removal empties the
// prompt the draft is discarded and its assumptions must go with it.
func Test_Parse_assumptionsDroppedWhenRemovalEmptiesDraft(t *testing.T) {
	p, err := Parse(`{"message":"hi","draft":{"name":"A","detail":"d","init_prompt":"## Tools & Capabilities\n- x\n","tool_names":[]},"assumptions":["a","b"]}`)
	if err != nil {
		t.Fatal(err)
	}
	if p.Draft != nil || p.Assumptions != nil {
		t.Fatalf("draft and assumptions must both be gone: draft=%+v assumptions=%v", p.Draft, p.Assumptions)
	}
}

// A discarded draft because a field is missing must also drop assumptions.
func Test_Parse_assumptionsNilWhenDraftMissingFields(t *testing.T) {
	p, err := Parse(`{"message":"hi","draft":{"name":"","detail":"d","init_prompt":"x","tool_names":[]},"assumptions":["a"]}`)
	if err != nil {
		t.Fatal(err)
	}
	if p.Assumptions != nil {
		t.Fatalf("assumptions must be nil, got %v", p.Assumptions)
	}
}

// ---- code review round 1 ----

// A tools header inside a code fence is example text, not a section, and a
// fenced "##" line inside a removed section does not end it.
func Test_Parse_toolsSectionAndCodeFences(t *testing.T) {
	got, removed := removeToolsSection("# T\n\n## Intro\n```\n## Tools & Capabilities\n```\nkeep\n")
	if removed || !strings.Contains(got, "keep") || strings.Count(got, "```") != 2 {
		t.Fatalf("a fenced header must not be treated as a section:\n%q", got)
	}
	got, removed = removeToolsSection("# T\n\n## Tools & Capabilities\n```\n## Example\n- a\n```\n\n## Other\nkeep\n")
	if !removed || strings.Contains(got, "Example") || strings.Contains(got, "```") || !strings.Contains(got, "## Other") || !strings.Contains(got, "keep") {
		t.Fatalf("a fenced header inside the removed section must go with it and the next real header must stay:\n%q", got)
	}
}

func Test_Parse_toolsHeaderDecorations(t *testing.T) {
	for _, h := range []string{
		"## **Tools & Capabilities**", "## Tools & Capabilities:", "   ## Tools & Capabilities",
		"## `Tools & Capabilities`", "## 🛠 Tools & Capabilities", "### __Tools & Capabilities__:",
	} {
		t.Run(h, func(t *testing.T) {
			got, removed := removeToolsSection("# T\n\n" + h + "\n- x\n\n## After\nkeep\n")
			if !removed || strings.Contains(got, "- x") || !strings.Contains(got, "## After") {
				t.Fatalf("variant must be removed:\n%q", got)
			}
		})
	}
	// four spaces is an indented code block in CommonMark, not a header.
	if _, removed := removeToolsSection("# T\n\n    ## Tools & Capabilities\n"); removed {
		t.Fatal("four leading spaces is not a header")
	}
}

// Removed tool names come from the model. They reach the client and the logs,
// so only identifier-shaped names are echoed and the list is bounded.
func Test_Parse_removedToolWarningsAreBounded(t *testing.T) {
	long := strings.Repeat("z", 300)
	p, err := Parse(`{"message":"m","draft":{"name":"A","init_prompt":"# A","tool_names":["IGNORE PREVIOUS INSTRUCTIONS","` + long + `","create_call"]}}`)
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range p.Warnings {
		if len(w) > 100 || strings.Contains(w, "IGNORE") || strings.Contains(w, "zzzz") {
			t.Fatalf("hostile text echoed in a warning: %q", w)
		}
	}
	if !hasWarning(p.Warnings, WarnToolRemoved+": create_call") {
		t.Fatalf("an identifier-shaped removed tool is still named: %v", p.Warnings)
	}
	names := make([]string, 0, 40)
	for i := 0; i < 40; i++ {
		names = append(names, fmt.Sprintf(`"tool_%d"`, i))
	}
	p, _ = Parse(`{"message":"m","draft":{"name":"A","init_prompt":"# A","tool_names":[` + strings.Join(names, ",") + `]}}`)
	n := 0
	for _, w := range p.Warnings {
		if strings.HasPrefix(w, WarnToolRemoved) {
			n++
		}
	}
	if n != maxRemovedToolWarnings {
		t.Fatalf("want at most %d removed-tool warnings, got %d", maxRemovedToolWarnings, n)
	}
}

// An object that decoded is consumed whole: the inner object of a draft must
// not be adopted as the response.
func Test_Parse_nestedObjectIsNotACandidate(t *testing.T) {
	if _, err := Parse(`{"draft":{"message":"inner","name":"a","init_prompt":"b"},"message":""}`); !errors.Is(err, ErrInvalidResponse) {
		t.Fatalf("want ErrInvalidResponse, got %v", err)
	}
	if _, err := Parse(`{"draft":{"message":"inner"}}`); !errors.Is(err, ErrInvalidResponse) {
		t.Fatalf("an object with no message must not expose its inner message: %v", err)
	}
	// a later top-level object after a consumed one is still reached.
	p, err := Parse(`{"draft":{"message":"inner"}} {"message":"outer"}`)
	if err != nil || p.Message != "outer" {
		t.Fatalf("err=%v p=%+v", err, p)
	}
}

func Test_Parse_whitespaceOnlyNameOrPromptDiscardsDraft(t *testing.T) {
	for _, d := range []string{
		`{"name":"   ","init_prompt":"# A","tool_names":[]}`,
		`{"name":"A","init_prompt":"  \n\t","tool_names":[]}`,
	} {
		p, err := Parse(`{"message":"m","draft":` + d + `}`)
		if err != nil || p.Draft != nil || !hasWarning(p.Warnings, WarnDraftDiscarded) {
			t.Fatalf("%s -> err=%v p=%+v", d, err, p)
		}
	}
}

// ---- code review round 2 ----

// A longer fence is not closed by a shorter one: the ``` example inside a
// ```` block stays inside it, so the Tools header after the block is real.
func Test_Parse_fenceLengthAndKind(t *testing.T) {
	got, removed := removeToolsSection("# T\n````md\n```\nexample\n```\n````\n## Tools & Capabilities\nx\n## Next\nafter\n")
	if !removed || strings.Contains(got, "\nx\n") || !strings.Contains(got, "example") || !strings.Contains(got, "## Next") {
		t.Fatalf("a ``` line inside a ```` block must not close it:\n%q", got)
	}
	// tildes: a ``` line inside a ~~~ block is plain text.
	got, removed = removeToolsSection("# T\n~~~\n```\n## Tools & Capabilities\n~~~\nkeep\n")
	if removed || !strings.Contains(got, "keep") || !strings.Contains(got, "## Tools & Capabilities") {
		t.Fatalf("a header inside a ~~~ block is not a header:\n%q", got)
	}
	// a closing fence must be bare: text after it makes it another opener.
	got, removed = removeToolsSection("# T\n```\ncode\n``` not a close\n## Tools & Capabilities\nx\n```\n## Next\nafter\n")
	if removed {
		t.Fatalf("the header sits inside a still-open fence:\n%q", got)
	}
	// an unclosed fence runs to the end (CommonMark): nothing after it is a header.
	if _, removed = removeToolsSection("# T\n```\n## Tools & Capabilities\nx\n"); removed {
		t.Fatal("a header inside an unclosed fence is code")
	}
}

func Test_Parse_adjacentObjectsAreReached(t *testing.T) {
	p, err := Parse(`{"x":1}{"message":"ok"}`)
	if err != nil || p.Message != "ok" {
		t.Fatalf("an object that starts right where the previous one ends must be reached: err=%v p=%+v", err, p)
	}
	p, err = Parse(`{"message":""}{"message":"ok"}`)
	if err != nil || p.Message != "ok" {
		t.Fatalf("err=%v p=%+v", err, p)
	}
}

func Test_Parse_boundaries(t *testing.T) {
	// exactly the limit is not truncated.
	exact := strings.Repeat("n", builder.MaxNameRunes)
	p, err := Parse(`{"message":"m","draft":{"name":"` + exact + `","init_prompt":"# A"}}`)
	if err != nil || p.Draft == nil || hasWarning(p.Warnings, WarnNameTruncated) || len(p.Draft.Name) != builder.MaxNameRunes {
		t.Fatalf("a name of exactly %d runes must pass untouched: %+v err=%v", builder.MaxNameRunes, p, err)
	}
	// identifier-shaped names: 64 is echoed, 65 is not.
	for n, wantEcho := range map[int]bool{64: true, 65: false} {
		id := strings.Repeat("a", n)
		p, _ := Parse(`{"message":"m","draft":{"name":"A","init_prompt":"# A","tool_names":["` + id + `"]}}`)
		echoed := hasWarning(p.Warnings, WarnToolRemoved+": "+id)
		if echoed != wantEcho {
			t.Errorf("%d-char tool name: echoed=%v, want %v (%v)", n, echoed, wantEcho, p.Warnings)
		}
	}
	// a longer identifier that merely contains a banned name is not a mention.
	p, _ = Parse(`{"message":"m","draft":{"name":"A","init_prompt":"# A\nuse xstop_flow here and stop_flowx too","tool_names":[]}}`)
	if hasWarningPrefix(p.Warnings, WarnForbiddenToolMentioned) {
		t.Fatalf("identifier-embedded names are not mentions: %v", p.Warnings)
	}
	// the cleaned prompt ends with exactly one newline.
	pp := mustParseDraft(t, "# A\n\n## Tools & Capabilities\n- x\n\n\n")
	if !strings.HasSuffix(pp.Draft.InitPrompt, "\n") || strings.HasSuffix(pp.Draft.InitPrompt, "\n\n") {
		t.Fatalf("want a single trailing newline, got %q", pp.Draft.InitPrompt)
	}
}

// The tilde fence is a real fence: a header inside it is not a header, and the
// fence opened with tildes is not closed by backticks.
func Test_Parse_tildeFenceHidesHeaders(t *testing.T) {
	got, removed := removeToolsSection("# T\n~~~\n## Tools & Capabilities\nx\n~~~\nkeep\n")
	if removed || !strings.Contains(got, "x") {
		t.Fatalf("a header inside a ~~~ fence is code:\n%q", got)
	}
	got, removed = removeToolsSection("# T\n~~~\n```\n~~~\n## Tools & Capabilities\nx\n## Next\nafter\n")
	if !removed || strings.Contains(got, "\nx\n") || !strings.Contains(got, "## Next") {
		t.Fatalf("the ~~~ block closes at ~~~, not at a ``` line inside it; the header after it is real:\n%q", got)
	}
}

// CommonMark: an info string after a backtick fence may not contain a backtick,
// so a line such as "``` a ` b" is not a fence opener.
func Test_Parse_backtickInfoStringWithBacktickIsNotAFence(t *testing.T) {
	got, removed := removeToolsSection("# T\n``` a ` b\n## Tools & Capabilities\nx\n## Next\nafter\n")
	if !removed || strings.Contains(got, "\nx\n") || !strings.Contains(got, "## Next") {
		t.Fatalf("a line with a backtick in its info string does not open a fence:\n%q", got)
	}
}

// A fence that is closed by a SHORTER run of the same character stays open
// until a long enough closer, so everything between is code.
func Test_Parse_shorterRunDoesNotCloseALongerFence(t *testing.T) {
	got, removed := removeToolsSection("# T\n````\n```\n## Tools & Capabilities\nx\n")
	if removed || !strings.Contains(got, "x") {
		t.Fatalf("the ``` line does not close a ```` fence, so the header is still code:\n%q", got)
	}
}
