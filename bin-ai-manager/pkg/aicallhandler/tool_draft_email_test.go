package aicallhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"

	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	commonidentity "monorepo/bin-common-handler/models/identity"
)

func draftEmailTool(args string) *message.ToolCall {
	return &message.ToolCall{
		ID:   "d8000000-0000-11f0-0000-000000000002",
		Type: message.ToolTypeFunction,
		Function: message.FunctionCall{
			Name:      message.FunctionCallNameDraftEmail,
			Arguments: args,
		},
	}
}

func jsonString(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// draftEmailArgs builds the JSON arguments of a draft_email call.
func draftEmailArgs(t *testing.T, to []string, subject, content string) string {
	t.Helper()
	return jsonString(t, map[string]any{"to": to, "subject": subject, "content": content})
}

// makeTestDomain builds a valid domain name of exactly n characters (labels of at most 63, last label "com").
func makeTestDomain(n int) string {
	before := n - len(".com")
	k := (before-1)/64 + 1
	labels := make([]string, 0, k)
	for i := 0; i < k-1; i++ {
		labels = append(labels, strings.Repeat("a", 63))
	}
	labels = append(labels, strings.Repeat("a", before-64*(k-1)))
	return strings.Join(labels, ".") + ".com"
}

func Test_normalizeDraftEmailAddress(t *testing.T) {
	label63 := strings.Repeat("a", 63)
	label64 := strings.Repeat("a", 64)
	// a domain of 253 characters: 3 labels of 63 and one of 61, joined by dots.
	domain253 := strings.Join([]string{label63, label63, label63, strings.Repeat("a", 61)}, ".")
	if len(domain253) != 253 {
		t.Fatalf("test domain length = %d", len(domain253))
	}
	local64 := strings.Repeat("l", 64)

	tests := []struct {
		name  string
		in    string
		want  string
		wantE bool
	}{
		{"plain address", "user@example.com", "user@example.com", false},
		{"surrounding spaces and upper case are normalized", "  User.Name@Example.COM ", "user.name@example.com", false},
		{"plus tag and atext symbols", "a+b_c!d'e@example.org", "a+b_c!d'e@example.org", false},
		{"punycode last label", "a@example.xn--p1ai", "a@example.xn--p1ai", false},
		{"two letter last label", "a@b.co", "a@b.co", false},
		{"hyphen inside a label", "a@my-host.example.net", "a@my-host.example.net", false},
		{"local part of 64 characters", local64 + "@example.com", local64 + "@example.com", false},
		{"local part of 65 characters", local64 + "x@example.com", "", true},
		{"label of 63 characters", "a@" + label63 + ".com", "a@" + label63 + ".com", false},
		{"label of 64 characters", "a@" + label64 + ".com", "", true},
		{"empty", "   ", "", true},
		{"no at sign", "example.com", "", true},
		{"display name", "Name <a@example.com>", "", true},
		{"angle brackets only", "<a@example.com>", "", true},
		{"comment", "a@example.com (comment)", "", true},
		{"quoted local part", "\"a b\"@example.com", "", true},
		{"quoted local part holding an address", "\"a@evil.example\"@example.com", "", true},
		{"two addresses", "a@example.com, b@example.com", "", true},
		{"semicolon", "a@example.com;b@example.com", "", true},
		{"line break inside", "a@example.com\nb@example.com", "", true},
		{"unicode local part", "é@example.com", "", true},
		{"unicode domain", "a@bücher.example", "", true},
		{"full width local part", "ａ@example.com", "", true},
		{"dotless host", "a@localhost", "", true},
		{"one letter last label", "a@b.c", "", true},
		{"dotted IPv4 host", "a@1.2.3.4", "", true},
		{"numeric last label", "a@example.123", "", true},
		{"bracketed IP literal", "a@[1.2.3.4]", "", true},
		{"underscore in the domain", "a@x_y.example", "", true},
		{"label starting with a hyphen", "a@-x.example", "", true},
		{"label ending with a hyphen", "a@x-.example", "", true},
		{"empty xn label", "a@example.xn--", "", true},
		{"leading dot in the local part", ".a@example.com", "", true},
		{"consecutive dots in the local part", "a..b@example.com", "", true},
		{"whole address of 254 characters", local64 + "@" + makeTestDomain(254-64-1), local64 + "@" + makeTestDomain(254-64-1), false},
		{"whole address of 255 characters", local64 + "@" + makeTestDomain(255-64-1), "", true},
		{"domain of 253 characters is too long once a local part is added", "a@" + domain253, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeDraftEmailAddress(tt.in)
			if (err != nil) != tt.wantE {
				t.Fatalf("normalizeDraftEmailAddress(%q) err = %v, wantErr %v", tt.in, err, tt.wantE)
			}
			if !tt.wantE && got != tt.want {
				t.Errorf("normalizeDraftEmailAddress(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func Test_validateEmailDraft(t *testing.T) {
	addrs := func(n int) []string {
		res := make([]string, 0, n)
		for i := 0; i < n; i++ {
			res = append(res, fmt.Sprintf("user%d@example.com", i))
		}
		return res
	}

	tests := []struct {
		name    string
		to      []string
		subject string
		content string
		want    []string
		wantErr string
	}{
		{name: "one recipient", to: []string{"a@example.com"}, subject: "s", content: "c", want: []string{"a@example.com"}},
		{name: "ten recipients", to: addrs(10), subject: "s", content: "c", want: addrs(10)},
		{name: "eleven recipients", to: addrs(11), subject: "s", content: "c", wantErr: "to has 11 recipients, the limit is 10"},
		{name: "no recipients", to: nil, subject: "s", content: "c", wantErr: "to is required"},
		{name: "empty list", to: []string{}, subject: "s", content: "c", wantErr: "to is required"},
		{name: "bad recipient names its index", to: []string{"a@example.com", "bad"}, subject: "s", content: "c", wantErr: "to[1] is not a valid email address"},
		{name: "normalized duplicate", to: []string{"a@example.com", " A@Example.com"}, subject: "s", content: "c", wantErr: "to[1] repeats to[0]"},
		{name: "order is kept and case folded", to: []string{"B@example.com", "a@example.com"}, subject: "s", content: "c", want: []string{"b@example.com", "a@example.com"}},
		{name: "subject at 200 runes", to: addrs(1), subject: strings.Repeat("a", 200), content: "c", want: addrs(1)},
		{name: "subject at 201 runes", to: addrs(1), subject: strings.Repeat("a", 201), content: "c", wantErr: "subject is too long: 201 characters, the limit is 200"},
		{name: "subject of 200 Korean characters (runes, not bytes)", to: addrs(1), subject: strings.Repeat("가", 200), content: "c", want: addrs(1)},
		{name: "subject of 201 Korean characters", to: addrs(1), subject: strings.Repeat("가", 201), content: "c", wantErr: "subject is too long: 201 characters"},
		{name: "content at 10000 runes", to: addrs(1), subject: "s", content: strings.Repeat("a", 10000), want: addrs(1)},
		{name: "content at 10001 runes", to: addrs(1), subject: "s", content: strings.Repeat("a", 10001), wantErr: "content is too long: 10001 characters, the limit is 10000"},
		{name: "content of 10000 Korean characters", to: addrs(1), subject: "s", content: strings.Repeat("가", 10000), want: addrs(1)},
		{name: "content of 10001 Korean characters", to: addrs(1), subject: "s", content: strings.Repeat("가", 10001), wantErr: "content is too long"},
		{name: "blank subject", to: addrs(1), subject: "   ", content: "c", wantErr: "subject is required"},
		{name: "blank content", to: addrs(1), subject: "s", content: " \n\t ", wantErr: "content is required"},
		{name: "subject with a line break", to: addrs(1), subject: "a\nb", content: "c", wantErr: "subject contains a control or direction character (U+000A)"},
		{name: "subject with a tab", to: addrs(1), subject: "a\tb", content: "c", wantErr: "subject contains a control or direction character (U+0009)"},
		{name: "subject with NUL", to: addrs(1), subject: "a\x00b", content: "c", wantErr: "subject contains a control or direction character (U+0000)"},
		{name: "subject with a line separator", to: addrs(1), subject: "a\u2028b", content: "c", wantErr: "(U+2028)"},
		{name: "subject with a paragraph separator", to: addrs(1), subject: "a\u2029b", content: "c", wantErr: "(U+2029)"},
		{name: "subject with a right to left override", to: addrs(1), subject: "a\u202eb", content: "c", wantErr: "(U+202E)"},
		{name: "subject with a left to right mark", to: addrs(1), subject: "a\u200eb", content: "c", wantErr: "(U+200E)"},
		{name: "subject with an Arabic letter mark", to: addrs(1), subject: "a\u061cb", content: "c", wantErr: "(U+061C)"},
		{name: "subject with an isolate", to: addrs(1), subject: "a\u2066b", content: "c", wantErr: "(U+2066)"},
		{name: "subject with a right to left mark", to: addrs(1), subject: "a\u200fb", content: "c", wantErr: "(U+200F)"},
		{name: "subject with a left to right embedding", to: addrs(1), subject: "a\u202ab", content: "c", wantErr: "(U+202A)"},
		{name: "subject with a pop directional isolate", to: addrs(1), subject: "a\u2069b", content: "c", wantErr: "(U+2069)"},
		{name: "content with a right to left mark", to: addrs(1), subject: "s", content: "a\u200fb", wantErr: "(U+200F)"},
		{name: "content with a left to right embedding", to: addrs(1), subject: "s", content: "a\u202ab", wantErr: "(U+202A)"},
		{name: "content with a pop directional isolate", to: addrs(1), subject: "s", content: "a\u2069b", wantErr: "(U+2069)"},
		{name: "subject with a C1 control", to: addrs(1), subject: "a\u0085b", content: "c", wantErr: "(U+0085)"},
		{name: "subject with the replacement character", to: addrs(1), subject: "a\ufffdb", content: "c", wantErr: "subject contains the replacement character U+FFFD"},
		{name: "subject with invalid UTF-8", to: addrs(1), subject: "a\xffb", content: "c", wantErr: "subject is not valid UTF-8"},
		{name: "content with line feed, carriage return and tab", to: addrs(1), subject: "s", content: "a\r\nb\tc", want: addrs(1)},
		{name: "content with NUL", to: addrs(1), subject: "s", content: "a\x00b", wantErr: "content contains a control or direction character (U+0000)"},
		{name: "content with a bell", to: addrs(1), subject: "s", content: "a\x07b", wantErr: "(U+0007)"},
		{name: "content with a right to left override", to: addrs(1), subject: "s", content: "a\u202eb", wantErr: "(U+202E)"},
		{name: "content with a line separator", to: addrs(1), subject: "s", content: "a\u2028b", wantErr: "(U+2028)"},
		{name: "content with the replacement character", to: addrs(1), subject: "s", content: "a\ufffdb", wantErr: "content contains the replacement character U+FFFD"},
		{name: "content with invalid UTF-8", to: addrs(1), subject: "s", content: "a\xffb", wantErr: "content is not valid UTF-8"},
		{name: "emoji joiner and zero width space are allowed", to: addrs(1), subject: "\U0001F468\u200d\U0001F469 a\u200bb", content: "ok", want: addrs(1)},
		{name: "HTML is plain text", to: addrs(1), subject: "<b>s</b>", content: "<p>c</p>", want: addrs(1)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateEmailDraft(tt.to, tt.subject, tt.content)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("validateEmailDraft() err = %v, want it to contain %q", err, tt.wantErr)
				}
				if got != nil {
					t.Errorf("a refusal must not return recipients: %v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("validateEmailDraft() err = %v", err)
			}
			if fmt.Sprint(got) != fmt.Sprint(tt.want) {
				t.Errorf("validateEmailDraft() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_toolHandleDraftEmail(t *testing.T) {
	c := &aicall.AIcall{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("d8000000-0000-11f0-0000-000000000001")}}
	x := newDraftHarness(t)

	tests := []struct {
		name string
		args string

		wantResult  string
		wantErrPart string
		wantBlock   *CardBlock
	}{
		{
			name:       "valid with title",
			args:       `{"to":["A@Example.com","b@example.org"],"subject":" Hello ","content":"Body\nline 2 ","title":"Follow up"}`,
			wantResult: "success",
			wantBlock:  &CardBlock{Type: "email_draft", Title: "Follow up", Recipients: []string{"a@example.com", "b@example.org"}, Subject: " Hello ", Content: "Body\nline 2 "},
		},
		{
			name:       "default title",
			args:       `{"to":["a@example.com"],"subject":"S","content":"C"}`,
			wantResult: "success",
			wantBlock:  &CardBlock{Type: "email_draft", Title: "Email draft", Recipients: []string{"a@example.com"}, Subject: "S", Content: "C"},
		},
		{
			name:       "title is cut at 200 characters",
			args:       `{"to":["a@example.com"],"subject":"S","content":"C","title":"` + strings.Repeat("a", 250) + `"}`,
			wantResult: "success",
			wantBlock:  &CardBlock{Type: "email_draft", Title: strings.Repeat("a", 197) + "...", Recipients: []string{"a@example.com"}, Subject: "S", Content: "C"},
		},
		{
			name:       "other unknown keys are ignored",
			args:       `{"to":["a@example.com"],"subject":"S","content":"C","priority":"high"}`,
			wantResult: "success",
			wantBlock:  &CardBlock{Type: "email_draft", Title: "Email draft", Recipients: []string{"a@example.com"}, Subject: "S", Content: "C"},
		},
		{name: "cc is refused", args: `{"to":["a@example.com"],"subject":"S","content":"C","cc":["b@example.com"]}`, wantResult: "failed", wantErrPart: "cc is not supported by draft_email"},
		{name: "bcc is refused", args: `{"to":["a@example.com"],"subject":"S","content":"C","bcc":["b@example.com"]}`, wantResult: "failed", wantErrPart: "bcc is not supported by draft_email"},
		{name: "attachments are refused", args: `{"to":["a@example.com"],"subject":"S","content":"C","attachments":[]}`, wantResult: "failed", wantErrPart: "attachments is not supported by draft_email"},
		{name: "source is refused", args: `{"to":["a@example.com"],"subject":"S","content":"C","source":{"type":"email"}}`, wantResult: "failed", wantErrPart: "source is not supported by draft_email"},
		{name: "from is refused", args: `{"to":["a@example.com"],"subject":"S","content":"C","from":"x@example.com"}`, wantResult: "failed", wantErrPart: "from is not supported by draft_email"},
		{name: "reply_to is refused", args: `{"to":["a@example.com"],"subject":"S","content":"C","reply_to":"x@example.com"}`, wantResult: "failed", wantErrPart: "reply_to is not supported by draft_email"},
		{name: "a null cc is refused too", args: `{"to":["a@example.com"],"subject":"S","content":"C","cc":null}`, wantResult: "failed", wantErrPart: "cc is not supported by draft_email"},
		{name: "an empty bcc array is refused too", args: `{"to":["a@example.com"],"subject":"S","content":"C","bcc":[]}`, wantResult: "failed", wantErrPart: "bcc is not supported by draft_email"},
		{name: "upper case CC is refused", args: `{"to":["a@example.com"],"subject":"S","content":"C","CC":["b@example.com"]}`, wantResult: "failed", wantErrPart: "cc is not supported by draft_email"},
		{name: "mixed case Source is refused", args: `{"to":["a@example.com"],"subject":"S","content":"C","Source":"x"}`, wantResult: "failed", wantErrPart: "source is not supported by draft_email"},
		{name: "to as a string", args: `{"to":"a@example.com","subject":"S","content":"C"}`, wantResult: "failed", wantErrPart: "cannot unmarshal string"},
		{name: "to as null", args: `{"to":null,"subject":"S","content":"C"}`, wantResult: "failed", wantErrPart: "to is required"},
		{name: "to with a non string element", args: `{"to":["a@example.com",5],"subject":"S","content":"C"}`, wantResult: "failed", wantErrPart: "cannot unmarshal number"},
		{name: "arguments that are not an object", args: `["a"]`, wantResult: "failed", wantErrPart: "cannot unmarshal array"},
		{name: "arguments that are not JSON", args: `{`, wantResult: "failed", wantErrPart: "unexpected end of JSON input"},
		{name: "eleven recipients", args: draftEmailArgs(t, func() []string {
			r := make([]string, 0, 11)
			for i := 0; i < 11; i++ {
				r = append(r, fmt.Sprintf("u%d@example.com", i))
			}
			return r
		}(), "S", "C"), wantResult: "failed", wantErrPart: "to has 11 recipients, the limit is 10"},
		{name: "display name recipient", args: `{"to":["Boss <boss@example.com>"],"subject":"S","content":"C"}`, wantResult: "failed", wantErrPart: "to[0] is not a valid email address"},
		{name: "blank subject", args: `{"to":["a@example.com"],"subject":" ","content":"C"}`, wantResult: "failed", wantErrPart: "subject is required"},
		{name: "blank content", args: `{"to":["a@example.com"],"subject":"S","content":""}`, wantResult: "failed", wantErrPart: "content is required"},
		{name: "content over the limit is refused, not cut", args: draftEmailArgs(t, []string{"a@example.com"}, "S", strings.Repeat("a", 10001)), wantResult: "failed", wantErrPart: "content is too long: 10001 characters"},
		{name: "subject over the limit is refused, not cut", args: draftEmailArgs(t, []string{"a@example.com"}, strings.Repeat("a", 201), "C"), wantResult: "failed", wantErrPart: "subject is too long: 201 characters"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := x.h.toolHandleDraftEmail(context.Background(), c, draftEmailTool(tt.args))

			if res.Result != tt.wantResult {
				t.Fatalf("result = %q (%s), want %q", res.Result, res.Message, tt.wantResult)
			}
			if tt.wantResult == "failed" {
				if !strings.Contains(res.Message, tt.wantErrPart) {
					t.Errorf("message = %q, want it to contain %q", res.Message, tt.wantErrPart)
				}
				if len(res.Blocks) != 0 {
					t.Errorf("a refused draft must carry no block: %+v", res.Blocks)
				}
				return
			}

			if len(res.Blocks) != 1 || fmt.Sprintf("%+v", res.Blocks[0]) != fmt.Sprintf("%+v", *tt.wantBlock) {
				t.Errorf("blocks = %+v, want %+v", res.Blocks, *tt.wantBlock)
			}
			if res.Message != "Displayed an email draft for review." || res.ResourceType != "card" {
				t.Errorf("message/resource_type = %q/%q", res.Message, res.ResourceType)
			}
			if strings.Contains(res.Message, tt.wantBlock.Content) || strings.Contains(res.Message, tt.wantBlock.Subject) {
				t.Errorf("the result message must never contain the draft body")
			}
		})
	}
}

// the handler records nothing and calls no request handler method (the mock has no expectation).
func Test_toolHandleDraftEmail_recordsNothingAndSendsNothing(t *testing.T) {
	x := newDraftHarness(t)

	res := x.h.toolHandleDraftEmail(context.Background(), x.c, draftEmailTool(draftEmailArgs(t, []string{"a@example.com"}, "S", "C")))
	if res.Result != "success" {
		t.Fatalf("result = %q (%s)", res.Result, res.Message)
	}
	if n := len(x.rows()); n != 0 {
		t.Errorf("the handler itself writes no row, got %d", n)
	}
}

// draftRowSize is the size of the stored draft row content the way the handler builds it.
func draftRowSize(t *testing.T, content string) int {
	t.Helper()
	b, err := json.Marshal(&messageContent{
		ToolCallID:   "d8000000-0000-11f0-0000-000000000002",
		Result:       "success",
		Message:      "Displayed an email draft for review.",
		ResourceType: "card",
		Blocks:       []CardBlock{{Type: "email_draft", Title: "Email draft", Recipients: []string{"a@example.com"}, Subject: "S", Content: content}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return len(b)
}

// the stored size guard: json.Marshal writes "<" as six bytes, so a draft within the rune limit
// can exceed the 65,535 byte column. The boundary is fixed to 60,000 bytes by literal comparison.
func Test_toolHandleDraftEmail_storedSizeGuard(t *testing.T) {
	x := newDraftHarness(t)
	handle := func(content string) *messageContent {
		return x.h.toolHandleDraftEmail(context.Background(), x.c, draftEmailTool(draftEmailArgs(t, []string{"a@example.com"}, "S", content)))
	}

	// find the largest number of "<" characters that is accepted.
	lo, hi := 1, 10000
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if handle(strings.Repeat("<", mid)).Result == "success" {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	if lo >= 10000 {
		t.Fatalf("10,000 \"<\" characters must be refused by the size guard")
	}
	if got := draftRowSize(t, strings.Repeat("<", lo)); got > 60000 {
		t.Errorf("the largest accepted draft has %d bytes, want at most 60000", got)
	}
	if got := draftRowSize(t, strings.Repeat("<", lo+1)); got <= 60000 {
		t.Errorf("the smallest refused draft has %d bytes, want more than 60000", got)
	}

	res := handle(strings.Repeat("<", lo+1))
	if res.Result != "failed" || !strings.Contains(res.Message, "the draft is too large to store") || len(res.Blocks) != 0 {
		t.Errorf("refused draft = %+v", res)
	}
	if strings.Contains(res.Message, "<") {
		t.Errorf("the refusal must not echo the draft")
	}

	// ordinary text up to the rune limit is far below the guard.
	if got := handle(strings.Repeat("a", 10000)); got.Result != "success" {
		t.Errorf("10,000 ordinary characters must be accepted: %s", got.Message)
	}

	// the title counts towards the stored size: a 200 character title of "<" is six times larger.
	titled := x.h.toolHandleDraftEmail(context.Background(), x.c, draftEmailTool(jsonString(t, map[string]any{
		"to": []string{"a@example.com"}, "subject": "S", "content": strings.Repeat("<", lo), "title": strings.Repeat("<", 150),
	})))
	if titled.Result != "failed" {
		t.Errorf("a title that pushes the stored size over the limit must be refused, got %s", titled.Result)
	}
}

// the info and sms_draft wire shapes are unchanged by the three new CardBlock fields.
func Test_CardBlock_oldWireShapesUnchangedByEmailFields(t *testing.T) {
	for _, b := range []CardBlock{
		{Type: "info", Title: "t"},
		{Type: message.BlockTypeSMSDraft, Title: "t", To: "+141****0100", Text: "x"},
	} {
		raw, err := json.Marshal(b)
		if err != nil {
			t.Fatal(err)
		}
		for _, key := range []string{"recipients", "subject", "content"} {
			if strings.Contains(string(raw), `"`+key+`"`) {
				t.Errorf("%s block json must not have the %q key: %s", b.Type, key, raw)
			}
		}
	}

	raw, _ := json.Marshal(CardBlock{Type: message.BlockTypeEmailDraft, Title: "t", Recipients: []string{"a@example.com"}, Subject: "S", Content: "C"})
	if string(raw) != `{"type":"email_draft","title":"t","recipients":["a@example.com"],"subject":"S","content":"C"}` {
		t.Errorf("email_draft block json = %s", raw)
	}
}

// through ToolHandle: draft_email records a card and sends nothing, its stored result carries the
// blocks (the frontend card), the LLM-facing result does not, the tool-call arguments stay in the
// stored call row, and the stored draft can then be approved end to end.
func Test_ToolHandle_draftEmail(t *testing.T) {
	x := newDraftHarness(t)
	gotCalls := x.expectEmailSend(nil)

	args := `{"to":["Person@Example.com"],"subject":"Subject-Marker-77","content":"Secret body 12345","title":"Hi"}`
	res, err := x.h.ToolHandle(context.Background(), x.c.ID, "tc-1", message.ToolTypeFunction, message.FunctionCall{
		Name:      message.FunctionCallNameDraftEmail,
		Arguments: args,
	}, uuid.Nil)
	if err != nil {
		t.Fatalf("ToolHandle() err = %v", err)
	}

	if len(*gotCalls) != 0 {
		t.Errorf("draft_email must not send")
	}
	if _, ok := res["blocks"]; ok {
		t.Errorf("the LLM-facing result must not carry blocks: %v", res)
	}
	for _, v := range res {
		if s, ok := v.(string); ok && (strings.Contains(s, "Secret body") || strings.Contains(s, "Subject-Marker") || strings.Contains(s, "person@example.com")) {
			t.Errorf("the LLM-facing result must not contain the draft: %v", res)
		}
	}
	if res["result"] != "success" || res["tool_call_id"] != "tc-1" {
		t.Errorf("result = %v", res)
	}

	var draftRow, callRow *message.Message
	for _, m := range x.rows() {
		switch m.Role {
		case message.RoleTool:
			draftRow = m
		case message.RoleAssistant:
			callRow = m
		}
	}
	if draftRow == nil || callRow == nil {
		t.Fatalf("expected a call row and a result row")
	}
	var stored messageContent
	_ = json.Unmarshal([]byte(draftRow.Content), &stored)
	if len(stored.Blocks) != 1 || stored.Blocks[0].Type != "email_draft" || stored.Blocks[0].Content != "Secret body 12345" || stored.Blocks[0].Subject != "Subject-Marker-77" ||
		len(stored.Blocks[0].Recipients) != 1 || stored.Blocks[0].Recipients[0] != "person@example.com" {
		t.Errorf("stored blocks = %+v", stored.Blocks)
	}
	if len(callRow.ToolCalls) != 1 || callRow.ToolCalls[0].Function.Arguments != args {
		t.Errorf("tool call arguments must be kept as is: %+v", callRow.ToolCalls)
	}

	// replay: blocks are removed, the arguments remain.
	msgs, err := x.h.getPipecatcallMessages(context.Background(), x.c)
	if err != nil {
		t.Fatal(err)
	}
	foundArgs := false
	for _, m := range msgs {
		if content, _ := m["content"].(string); strings.Contains(content, "blocks") || strings.Contains(content, "email_draft") || strings.Contains(content, "recipients") {
			t.Errorf("blocks must not be replayed: %s", content)
		}
		if tcs, ok := m["tool_calls"].([]message.ToolCall); ok && len(tcs) == 1 && tcs[0].Function.Arguments == args {
			foundArgs = true
		}
	}
	if !foundArgs {
		t.Errorf("the draft_email tool call arguments must stay in the replay")
	}

	// the stored draft can then be approved: end to end through the real rows.
	x.setNow(draftRow.TMCreate.Add(5_000_000_000))
	if _, err := x.h.DraftDecision(context.Background(), x.c.ID, draftRow.ID, message.DraftDecisionApprove); err != nil {
		t.Fatalf("DraftDecision() err = %v", err)
	}
	if len(*gotCalls) != 1 {
		t.Errorf("sends = %d, want 1", len(*gotCalls))
	}
}

// a failed draft_email call is a failed result without a card; nothing is stored as a draft.
func Test_ToolHandle_draftEmailRefusedHasNoCard(t *testing.T) {
	x := newDraftHarness(t)

	res, err := x.h.ToolHandle(context.Background(), x.c.ID, "tc-1", message.ToolTypeFunction, message.FunctionCall{
		Name:      message.FunctionCallNameDraftEmail,
		Arguments: `{"to":["a@example.com"],"subject":"S","content":"C","cc":["b@example.com"]}`,
	}, uuid.Nil)
	if err != nil {
		t.Fatalf("ToolHandle() err = %v", err)
	}
	if res["result"] != "failed" {
		t.Errorf("result = %v", res)
	}
	for _, m := range x.rows() {
		if m.Role == message.RoleTool && strings.Contains(m.Content, "email_draft") {
			t.Errorf("a refused draft must not store a card: %s", m.Content)
		}
	}
}

// the direct send_email path keeps its handler message (VOIP-1710 design section 4).
func Test_ToolHandle_sendEmailKeepsItsMessage(t *testing.T) {
	x := newDraftHarness(t)
	gotCalls := x.expectEmailSend(nil)

	res, err := x.h.ToolHandle(context.Background(), x.c.ID, "tc-9", message.ToolTypeFunction, message.FunctionCall{
		Name:      message.FunctionCallNameSendEmail,
		Arguments: `{"destinations":[{"type":"email","target":"a@example.com"}],"subject":"S","content":"C"}`,
	}, uuid.Nil)
	if err != nil {
		t.Fatalf("ToolHandle() err = %v", err)
	}
	if res["result"] != "success" || res["message"] != "Email sent successfully." || res["resource_type"] != "email" {
		t.Errorf("result = %v", res)
	}
	if len(*gotCalls) != 1 {
		t.Errorf("sends = %d, want 1", len(*gotCalls))
	}
}

// logs carry lengths only: no address, subject or text of a draft reaches a log entry on the
// success, refusal and decision paths.
func Test_draftEmail_logsCarryNoDraftContent(t *testing.T) {
	logger := logrus.StandardLogger()
	oldLevel := logger.GetLevel()
	logger.SetLevel(logrus.DebugLevel)
	hook := logtest.NewGlobal()
	defer func() {
		hook.Reset()
		logger.SetLevel(oldLevel)
	}()

	const (
		addr    = "log-marker-person@example.com"
		subject = "SubjectMarkerQ9"
		body    = "BodyMarkerQ9"
	)

	x := newDraftHarness(t)
	x.expectEmailSend(nil)

	// success and refusal through the handler.
	x.h.toolHandleDraftEmail(context.Background(), x.c, draftEmailTool(draftEmailArgs(t, []string{addr}, subject, body)))
	x.h.toolHandleDraftEmail(context.Background(), x.c, draftEmailTool(draftEmailArgs(t, []string{addr, "bad"}, subject, body)))
	x.h.toolHandleDraftEmail(context.Background(), x.c, draftEmailTool(jsonString(t, map[string]any{"to": []string{addr}, "subject": subject, "content": body, "cc": []string{addr}})))

	// approve, failed approve and reject through the decision path.
	approve := x.newEmailDraft([]string{addr}, subject, body)
	reject := x.newEmailDraft([]string{addr}, subject, body)
	x.setNow(approve.TMCreate.Add(1_000_000_000))
	if _, err := x.h.DraftDecision(context.Background(), x.c.ID, approve.ID, message.DraftDecisionApprove); err != nil {
		t.Fatal(err)
	}
	if _, err := x.h.DraftDecision(context.Background(), x.c.ID, reject.ID, message.DraftDecisionReject); err != nil {
		t.Fatal(err)
	}

	y := newDraftHarness(t)
	y.expectEmailSend(fmt.Errorf("provider rejected the request"))
	failed := y.newEmailDraft([]string{addr}, subject, body)
	y.setNow(failed.TMCreate.Add(1_000_000_000))
	if _, err := y.h.DraftDecision(context.Background(), y.c.ID, failed.ID, message.DraftDecisionApprove); err != nil {
		t.Fatal(err)
	}

	entries := hook.AllEntries()
	if len(entries) == 0 {
		t.Fatalf("expected log entries from the draft paths")
	}
	for _, e := range entries {
		all := e.Message
		for k, v := range e.Data {
			all += " " + k + "=" + fmt.Sprint(v)
		}
		for _, secret := range []string{addr, subject, body} {
			if strings.Contains(all, secret) {
				t.Errorf("a log entry holds draft content %q: %s", secret, all)
			}
		}
	}
}
