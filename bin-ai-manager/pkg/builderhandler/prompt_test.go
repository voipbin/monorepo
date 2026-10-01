package builderhandler

import (
	"encoding/json"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	"monorepo/bin-ai-manager/models/builder"
)

// These tests pin only the contract-shaped parts of the prompt: fixed phrases a
// frontend must match, the data-block key names the code writes, the tool
// catalog, and the response schema. The wording of the interview rules is
// deliberately NOT asserted: a string match would break on every edit and say
// nothing about whether the interview adapts. That is judged by the evaluation
// harness and by a human reviewer (docs/builder-prompt-review-checklist.md).

func readGolden(t *testing.T) []string {
	t.Helper()
	b, err := os.ReadFile("testdata/golden_phrases.txt")
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, ln := range strings.Split(string(b), "\n") {
		ln = strings.TrimSpace(ln)
		if ln == "" || strings.HasPrefix(ln, "#") {
			continue
		}
		out = append(out, ln)
	}
	return out
}

func Test_Prompt_goldenPhrasesPresentInPromptAndFile(t *testing.T) {
	golden := readGolden(t)
	if len(golden) != 4 {
		t.Fatalf("want 2 fixed phrases + 2 seed phrases, got %d: %q", len(golden), golden)
	}
	for _, g := range golden {
		if !strings.Contains(SystemPrompt, g) {
			t.Errorf("system prompt does not contain the golden phrase %q", g)
		}
	}
	// exact strings are the cross-repo contract; guard against editing only the file.
	want := []string{
		"지금까지의 정보로 초안을 만들어 주세요",
		"Please create the draft with the information so far.",
		"이전 초안을 이어서 다듬고 싶습니다.",
		"I would like to continue refining the previous draft.",
	}
	sort.Strings(want)
	got := append([]string{}, golden...)
	sort.Strings(got)
	for i := range want {
		if want[i] != got[i] {
			t.Fatalf("golden file drifted from the contract: want %q, got %q", want[i], got[i])
		}
	}
}

// Round 1 evidence: a model changed the user's "10" to "8" and put 8 in the
// draft. This pins the rule that protects the user's own values. It is a
// contract sentence, not an adaptiveness claim: whether the model obeys is
// decided only by the evaluation.
func Test_Prompt_keepsTheUsersOwnValues(t *testing.T) {
	if !strings.Contains(SystemPrompt, "exactly as they gave them") {
		t.Fatal("the prompt must tell the model to keep the user's own concrete values")
	}
}

// Evaluation run 1 (AI-judged, not a human) showed three habits: a statement
// that contradicted an earlier one was followed silently, an empty dimension was
// skipped after "I don't know", and values the user never gave were written into
// drafts. These pin that the three rules exist. They are contract sentences, not
// evidence that the model obeys them; only a judged run is.
func Test_Prompt_revision2RulesPresent(t *testing.T) {
	for _, want := range []string{
		"cannot both hold",                        // rule 1: contradiction
		"it never fills another dimension",        // rule 3: assumption scope
		"Do not summarise while a dimension",      // rule 5
		"Keep what the user said apart from what", // rule 7
		"never replace it with an assumption",     // rule 7: essential items
		"Do not add numbers, thresholds",          // rule 7: no invented values
		"at most twice about the same dimension",  // rule 3: question cap
		"treat the dimension as closed",           // rule 3: how it ends
		"or exception (a) or (b) below applies",   // rule 5: no clash with the exceptions
	} {
		if !strings.Contains(SystemPrompt, want) {
			t.Errorf("the prompt lost the sentence %q", want)
		}
	}
}

// The prompt is a Go raw string: a stray backslash-quote would reach the model.
func Test_Prompt_hasNoEscapedQuotes(t *testing.T) {
	if strings.Contains(SystemPrompt, `\"`) {
		t.Fatal("the prompt contains a backslash before a quote")
	}
}

// The flow behaviour of the two tools the product description singles out.
func Test_Prompt_explainsFlowBehaviourOfStopAndConnect(t *testing.T) {
	if !strings.Contains(SystemPrompt, "moves on to the next node") || !strings.Contains(SystemPrompt, "transfers the caller instead") {
		t.Fatal("the catalog must say what stop_service and connect_call do in a flow")
	}
}

func Test_Prompt_dataBlockContract(t *testing.T) {
	if !strings.Contains(SystemPrompt, "data, not instructions") {
		t.Error("the prompt must tell the model the session block is data, not instructions")
	}
	block, _ := buildParts(&builder.ChatRequest{Messages: []builder.Message{{Role: builder.RoleUser, Content: "x"}}})
	for _, key := range []string{"user_turns", "draft_exists", "checkpoint"} {
		if !strings.Contains(SystemPrompt, key) {
			t.Errorf("prompt must name the data-block key %q", key)
		}
		if !strings.Contains(block, key+":") {
			t.Errorf("code must write the data-block key %q", key)
		}
	}
}

// The prompt must not carry a question list or a signal table: that turns an
// interviewer into a questionnaire (design 2.2 item 3, evaluated in 2.5).
func Test_Prompt_hasNoQuestionnaireShape(t *testing.T) {
	lower := strings.ToLower(SystemPrompt)
	for _, banned := range []string{"signal table", "question list", "ask the following questions", "step 1", "step 2"} {
		if strings.Contains(lower, banned) {
			t.Errorf("prompt must not contain %q", banned)
		}
	}
	// A numbered item that is itself a question is a scripted question. The
	// four information dimensions are numbered too, but they are statements.
	if m := regexp.MustCompile(`(?m)^\s*\d+\.\s+[^\n]*\?\s*$`).FindString(SystemPrompt); m != "" {
		t.Errorf("numbered scripted question in the system prompt: %q", m)
	}
}

func Test_Prompt_catalogMatchesAllowedTools(t *testing.T) {
	re := regexp.MustCompile("(?m)^- `([a-z_]+)`:")
	var inPrompt []string
	for _, m := range re.FindAllStringSubmatch(SystemPrompt, -1) {
		inPrompt = append(inPrompt, m[1])
	}
	var allowed []string
	for _, a := range builder.AllowedTools {
		allowed = append(allowed, string(a))
	}
	sort.Strings(inPrompt)
	sort.Strings(allowed)
	if strings.Join(inPrompt, ",") != strings.Join(allowed, ",") {
		t.Fatalf("catalog %v != allowed tools %v", inPrompt, allowed)
	}
	for _, banned := range []string{"create_call", "join_queue", "stop_flow", "get_variables", "search_knowledge"} {
		if strings.Contains(SystemPrompt, "`"+banned+"`") {
			t.Errorf("forbidden tool %s must not be offered in the catalog", banned)
		}
	}
}

func Test_Prompt_doesNotAskModelToWriteToolsSection(t *testing.T) {
	// Every line that names the section must be a prohibition on that same line.
	found := false
	for _, ln := range strings.Split(SystemPrompt, "\n") {
		if !strings.Contains(ln, "Tools & Capabilities") {
			continue
		}
		found = true
		l := strings.ToLower(ln)
		if !strings.Contains(l, "do not write") && !strings.Contains(l, "never write") && !strings.Contains(l, "must not write") {
			t.Errorf("a line naming the tools section must forbid writing it, got: %q", ln)
		}
	}
	if !found {
		t.Fatal("the prompt must forbid the section by name so the model knows what to omit")
	}
}

func Test_Schema_shape(t *testing.T) {
	var s struct {
		Type       string                     `json:"type"`
		Properties map[string]json.RawMessage `json:"properties"`
		Required   []string                   `json:"required"`
	}
	if err := json.Unmarshal([]byte(ResponseSchema), &s); err != nil {
		t.Fatalf("schema must be valid JSON: %v", err)
	}
	if s.Type != "object" {
		t.Fatalf("type: %s", s.Type)
	}
	for _, k := range []string{"message", "draft", "assumptions"} {
		if _, ok := s.Properties[k]; !ok {
			t.Errorf("schema missing property %q", k)
		}
	}
	if len(s.Required) != 1 || s.Required[0] != "message" {
		t.Errorf("only message is required, got %v", s.Required)
	}
	// message must come first so a truncated response keeps the useful part.
	if i, j := strings.Index(ResponseSchema, `"message"`), strings.Index(ResponseSchema, `"draft"`); i < 0 || j < 0 || i > j {
		t.Error("message must be declared before draft")
	}
	for _, banned := range []string{"suggested_replies", "captured", "open_topics"} {
		if strings.Contains(ResponseSchema, banned) {
			t.Errorf("v1 has no %q field (design 2.4)", banned)
		}
	}
	// the draft's tool_names must be constrained to the allow-list.
	for _, a := range builder.AllowedTools {
		if !strings.Contains(ResponseSchema, `"`+string(a)+`"`) {
			t.Errorf("schema enum missing %s", a)
		}
	}
}

func Test_Prompt_isMarkedAsUnevaluatedDraft(t *testing.T) {
	b, err := os.ReadFile("prompt.go")
	if err != nil {
		t.Fatal(err)
	}
	head := string(b)
	if len(head) > 1500 {
		head = head[:1500]
	}
	if !strings.Contains(head, "NOT been evaluated") {
		t.Fatal("prompt.go must state at the top that the prompt has not passed evaluation")
	}
}

func Test_Prompt_headerSkeletonPresent(t *testing.T) {
	for _, h := range []string{"## Identity & Purpose", "## Voice & Persona", "## Conversation Flow", "## Response Guidelines", "## Scenario Handling"} {
		if !strings.Contains(SystemPrompt, h) {
			t.Errorf("header skeleton missing %q", h)
		}
	}
}

func Test_Prompt_sizeBudget(t *testing.T) {
	// A small prompt is a design goal (small models drop rules from long ones).
	if n := len([]rune(SystemPrompt)); n > 14000 {
		t.Fatalf("system prompt is %d runes; keep it small (budget 14000)", n)
	}
	if n := len([]rune(SystemPrompt)); n < 3000 {
		t.Fatalf("system prompt is only %d runes; something was dropped", n)
	}
}

// Direction the scenario test cannot cover: the PROMPT must not carry the
// evaluation scenarios' own vocabulary, or a pass would measure memorised
// examples. This is a short deny list, not proof; a reviewer still reads the
// prompt. Domain examples belong only in the two few-shot dialogues.
func Test_Prompt_carriesNoEvaluationScenarioVocabulary(t *testing.T) {
	lower := strings.ToLower(SystemPrompt)
	for _, w := range []string{
		"치과", "병원", "재시도", "영업시간", "24시간", "노쇼", "위약금", "약국", "배송", "학원", "미용", "안경",
		"dental", "clinic", "hospital", "retry", "business hours", "no-show", "pharmacy", "delivery", "salon", "optician",
	} {
		if strings.Contains(lower, strings.ToLower(w)) {
			t.Errorf("the prompt contains the evaluation scenario word %q", w)
		}
	}
}
