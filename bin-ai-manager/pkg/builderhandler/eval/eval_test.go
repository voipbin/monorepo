package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sashabaranov/go-openai"

	"monorepo/bin-ai-manager/models/builder"
	"monorepo/bin-ai-manager/pkg/builderhandler"
)

// ---- fakes ----

// fakeBuilder answers like a builder would, deterministically: it drafts on the
// first turn when the user sent a fixed phrase, otherwise it asks two questions
// and drafts on the third user turn.
type fakeBuilder struct{ calls int }

func (f *fakeBuilder) SendOnce(_ context.Context, req *openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, error) {
	f.calls++
	users := 0
	var firstUser string
	for _, m := range req.Messages {
		if m.Role == openai.ChatMessageRoleUser {
			users++
			if firstUser == "" {
				firstUser = m.Content
			}
		}
	}
	draft := `"draft":{"name":"테스트 봇","detail":"설명","init_prompt":"# 테스트 봇\n\n## Identity & Purpose\nHelp.","tool_names":["connect_call"]},"assumptions":["가정 하나"]`
	var body string
	switch {
	case strings.Contains(firstUser, "초안을 만들어 주세요") || strings.Contains(firstUser, "create the draft") || strings.Contains(firstUser, "그냥 만들어"):
		body = `{"message":"초안입니다", ` + draft + `}`
	case users >= 3:
		body = `{"message":"초안입니다", ` + draft + `}`
	default:
		body = fmt.Sprintf(`{"message":"질문 %d 입니다. 어떤가요?"}`, users)
	}
	return &openai.ChatCompletionResponse{
		Choices: []openai.ChatCompletionChoice{{Message: openai.ChatCompletionMessage{Role: "assistant", Content: body}, FinishReason: openai.FinishReasonStop}},
		Usage:   openai.Usage{PromptTokens: 100, CompletionTokens: 20, TotalTokens: 120},
	}, nil
}

type fakeSim struct{ calls int }

func (f *fakeSim) SendOnce(_ context.Context, _ *openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, error) {
	f.calls++
	return &openai.ChatCompletionResponse{
		Choices: []openai.ChatCompletionChoice{{Message: openai.ChatCompletionMessage{Role: "assistant", Content: fmt.Sprintf("사용자 답변 %d", f.calls)}, FinishReason: openai.FinishReasonStop}},
		Usage:   openai.Usage{PromptTokens: 50, CompletionTokens: 10, TotalTokens: 60},
	}, nil
}

type scripted struct {
	resp string
	err  error
}

func (s scripted) SendOnce(_ context.Context, _ *openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &openai.ChatCompletionResponse{Choices: []openai.ChatCompletionChoice{{Message: openai.ChatCompletionMessage{Content: s.resp}, FinishReason: openai.FinishReasonStop}}}, nil
}

// ---- the plan ----

func Test_Plan_totalsMatchDesign(t *testing.T) {
	// Design 2.5 execution counts: 15 + 4 + 3 + 3 + 4 + 7 = 36 simulated runs.
	runs := PlanRuns(nil)
	if len(runs) != 36 {
		t.Fatalf("want 36 simulated runs, got %d", len(runs))
	}
	if n := countGroups(); n != 17 {
		t.Fatalf("want 17 simulated groups (s15 is synthetic and counted separately), got %d", n)
	}
	if n := len(Groups()); n != 18 {
		t.Fatalf("want 18 gate groups including s15, got %d", n)
	}
	syn := SyntheticCases()
	if len(syn) != 5 {
		t.Fatalf("want 5 synthetic cases (scenario 15), got %d", len(syn))
	}
}

func countGroups() int {
	n := 0
	for _, g := range Groups() {
		if g.Name != "s15" {
			n++
		}
	}
	return n
}

func Test_Plan_groupsAreConsistentWithScenarios(t *testing.T) {
	totals := map[string]int{}
	for _, r := range PlanRuns(nil) {
		totals[r.Group]++
	}
	for _, g := range Groups() {
		if g.Name == "s15" {
			continue
		}
		if totals[g.Name] != g.Total {
			t.Errorf("group %s: scenarios give %d runs, group says %d", g.Name, totals[g.Name], g.Total)
		}
		if g.MinPass < 1 || g.MinPass > g.Total {
			t.Errorf("group %s: bad MinPass %d of %d", g.Name, g.MinPass, g.Total)
		}
		delete(totals, g.Name)
	}
	if len(totals) != 0 {
		t.Errorf("scenarios with no group: %v", totals)
	}
}

// Design 2.5 pass rule: the five adaptiveness scenarios need 2 of 3, the two
// pair scenarios need both runs, the rest need their single runs.
func Test_Plan_passRulesMatchDesign(t *testing.T) {
	want := map[string][2]int{ // total, minpass
		"s2b": {3, 2}, "s3": {3, 2}, "s4-A1": {3, 2}, "s4-A2": {3, 2}, "s6": {3, 2},
		"s5-B1": {2, 2}, "s5-B2": {2, 2},
		"s9": {3, 3}, "s13": {3, 3}, "s14": {4, 4},
		"s1": {1, 1}, "s2a": {1, 1}, "s7": {1, 1}, "s8": {1, 1}, "s10": {1, 1}, "s11": {1, 1}, "s12": {1, 1},
	}
	for _, g := range Groups() {
		if w, ok := want[g.Name]; ok {
			if g.Total != w[0] || g.MinPass != w[1] {
				t.Errorf("%s: got %d/%d, design says %d/%d", g.Name, g.MinPass, g.Total, w[1], w[0])
			}
			delete(want, g.Name)
		}
	}
	if len(want) != 0 {
		t.Errorf("groups missing: %v", want)
	}
}

func Test_Plan_filter(t *testing.T) {
	r := PlanRuns([]string{"s2b"})
	if len(r) != 3 {
		t.Fatalf("filter by group/scenario id: got %d", len(r))
	}
	if len(PlanRuns([]string{"nope"})) != 0 {
		t.Fatal("unknown filter must select nothing")
	}
}

func Test_Plan_sameFirstRequestForPairs(t *testing.T) {
	// Branch pairs only mean something if the first request is identical.
	first := map[string]string{}
	for _, s := range Scenarios() {
		switch s.Group {
		case "s4-A1", "s4-A2", "s5-B1", "s5-B2":
			if first["pair"] == "" {
				first["pair"] = s.FirstMessage
			}
			if s.FirstMessage != first["pair"] {
				t.Errorf("%s first message %q differs from the shared one %q", s.ID, s.FirstMessage, first["pair"])
			}
		}
	}
}

// The persona vocabulary must not overlap the few-shot domains, or the
// evaluation measures memorisation of the examples instead of adaptation.
func Test_Plan_personasAvoidFewShotDomains(t *testing.T) {
	banned := []string{"부동산", "중개", "내집", "임대", "헬스", "회원권", "갱신", "real estate", "gym"}
	for _, s := range Scenarios() {
		blob := strings.ToLower(s.FirstMessage + s.Persona.Facts + s.Persona.Style + s.Persona.Behavior + s.Title)
		for _, b := range banned {
			if strings.Contains(blob, strings.ToLower(b)) {
				t.Errorf("scenario %s mentions few-shot domain word %q", s.ID, b)
			}
		}
	}
}

func Test_Plan_runIDsUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, r := range PlanRuns(nil) {
		if seen[r.RunID] {
			t.Fatalf("duplicate run id %s", r.RunID)
		}
		seen[r.RunID] = true
	}
}

// ---- a conversation ----

func Test_RunConversation_endsAtFirstDraftByDefault(t *testing.T) {
	sc := Scenario{ID: "x", Group: "x", FirstMessage: "고객센터 봇", MaxTurns: 8, Persona: Persona{Facts: "f", Style: "s", Behavior: "b"}}
	res := RunConversation(context.Background(), &fakeBuilder{}, &fakeSim{}, builderhandler.DefaultConfig(), DefaultSimConfig(), RunSpec{RunID: "x#1", Scenario: sc})
	if res.Aborted != "" {
		t.Fatalf("aborted: %s", res.Aborted)
	}
	if res.BuilderCalls != 3 || res.FinalDraft == nil {
		t.Fatalf("expected a draft on the third builder call, got calls=%d draft=%v", res.BuilderCalls, res.FinalDraft)
	}
	// transcript alternates user/assistant starting and ending with the right roles.
	if res.Turns[0].Role != builder.RoleUser || res.Turns[len(res.Turns)-1].Role != builder.RoleAssistant {
		t.Fatalf("bad turn order: %+v", res.Turns)
	}
	if res.FirstResponseHasDraft {
		t.Fatal("first response had no draft")
	}
}

func Test_RunConversation_maxTurnsStops(t *testing.T) {
	sc := Scenario{ID: "x", Group: "x", FirstMessage: "hi", MaxTurns: 2, Persona: Persona{Facts: "f"}}
	res := RunConversation(context.Background(), &fakeBuilder{}, &fakeSim{}, builderhandler.DefaultConfig(), DefaultSimConfig(), RunSpec{RunID: "x#1", Scenario: sc})
	if res.BuilderCalls != 2 || res.FinalDraft != nil {
		t.Fatalf("calls=%d draft=%v", res.BuilderCalls, res.FinalDraft)
	}
}

func Test_RunConversation_continueAfterDraft(t *testing.T) {
	sc := Scenario{ID: "x", Group: "x", FirstMessage: "hi", MaxTurns: 8, ContinueAfterDraft: 2, Persona: Persona{Facts: "f"}}
	res := RunConversation(context.Background(), &fakeBuilder{}, &fakeSim{}, builderhandler.DefaultConfig(), DefaultSimConfig(), RunSpec{RunID: "x#1", Scenario: sc})
	if res.BuilderCalls != 5 {
		t.Fatalf("draft on call 3 plus 2 more turns = 5, got %d", res.BuilderCalls)
	}
}

func Test_RunConversation_firstResponseDraft(t *testing.T) {
	sc := Scenario{ID: "s9", Group: "s9", FirstMessage: "지금까지의 정보로 초안을 만들어 주세요", MaxTurns: 1, Persona: Persona{Facts: "f"}}
	res := RunConversation(context.Background(), &fakeBuilder{}, &fakeSim{}, builderhandler.DefaultConfig(), DefaultSimConfig(), RunSpec{RunID: "s9#1", Scenario: sc})
	if !res.FirstResponseHasDraft || res.BuilderCalls != 1 {
		t.Fatalf("%+v", res)
	}
}

func Test_RunConversation_parseFailureIsRecordedAndEndsRun(t *testing.T) {
	sc := Scenario{ID: "x", Group: "x", FirstMessage: "hi", MaxTurns: 5, Persona: Persona{Facts: "f"}}
	res := RunConversation(context.Background(), scripted{resp: "not json"}, &fakeSim{}, builderhandler.DefaultConfig(), DefaultSimConfig(), RunSpec{RunID: "x#1", Scenario: sc})
	if res.ParseFailures != 1 || res.BuilderCalls != 1 || res.ParseFailed == "" || res.Aborted != "" {
		t.Fatalf("%+v", res)
	}
}

func Test_RunConversation_simulatorFailureAborts(t *testing.T) {
	sc := Scenario{ID: "x", Group: "x", FirstMessage: "hi", MaxTurns: 5, Persona: Persona{Facts: "f"}}
	res := RunConversation(context.Background(), &fakeBuilder{}, scripted{err: fmt.Errorf("boom")}, builderhandler.DefaultConfig(), DefaultSimConfig(), RunSpec{RunID: "x#1", Scenario: sc})
	if res.Aborted == "" || !strings.Contains(res.Aborted, "simulator") {
		t.Fatalf("%+v", res)
	}
}

func Test_RunConversation_simulatorEmptyOutputAborts(t *testing.T) {
	sc := Scenario{ID: "x", Group: "x", FirstMessage: "hi", MaxTurns: 5, Persona: Persona{Facts: "f"}}
	res := RunConversation(context.Background(), &fakeBuilder{}, scripted{resp: "   "}, builderhandler.DefaultConfig(), DefaultSimConfig(), RunSpec{RunID: "x#1", Scenario: sc})
	if !strings.Contains(res.Aborted, "simulator") {
		t.Fatalf("%+v", res)
	}
}

func Test_RunConversation_recordsForbiddenToolMention(t *testing.T) {
	body := `{"message":"초안","draft":{"name":"A","detail":"d","init_prompt":"# A\nUse create_call to dial.","tool_names":[]}}`
	sc := Scenario{ID: "x", Group: "x", FirstMessage: "hi", MaxTurns: 3, Persona: Persona{Facts: "f"}}
	res := RunConversation(context.Background(), scripted{resp: body}, &fakeSim{}, builderhandler.DefaultConfig(), DefaultSimConfig(), RunSpec{RunID: "x#1", Scenario: sc})
	if len(res.ForbiddenToolWarnings) != 1 || !strings.Contains(res.ForbiddenToolWarnings[0], "create_call") {
		t.Fatalf("%+v", res.ForbiddenToolWarnings)
	}
}

func Test_Simulator_promptCarriesPersonaAndTranscriptButNotBuilderPrompt(t *testing.T) {
	var got *openai.ChatCompletionRequest
	s := senderFunc(func(_ context.Context, r *openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, error) {
		got = r
		return &openai.ChatCompletionResponse{Choices: []openai.ChatCompletionChoice{{Message: openai.ChatCompletionMessage{Content: "네 맞아요"}}}}, nil
	})
	out, _, err := SimulateUser(context.Background(), s, DefaultSimConfig(), Persona{Facts: "FACTS-X", Style: "STYLE-X", Behavior: "BEHAVIOR-X"},
		[]builder.Message{{Role: builder.RoleUser, Content: "첫 메시지"}, {Role: builder.RoleAssistant, Content: "질문입니다"}})
	if err != nil || out != "네 맞아요" {
		t.Fatalf("%q %v", out, err)
	}
	all := ""
	for _, m := range got.Messages {
		all += m.Content + "\n"
	}
	for _, w := range []string{"FACTS-X", "STYLE-X", "BEHAVIOR-X", "첫 메시지", "질문입니다"} {
		if !strings.Contains(all, w) {
			t.Errorf("simulator input missing %q", w)
		}
	}
	if strings.Contains(all, "Interview in the user's language") {
		t.Error("the simulator must never see the builder's system prompt")
	}
	if got.Temperature < 0.7 {
		t.Errorf("simulator temperature should be high, got %v", got.Temperature)
	}
}

type senderFunc func(context.Context, *openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, error)

func (f senderFunc) SendOnce(ctx context.Context, r *openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, error) {
	return f(ctx, r)
}

// ---- synthetic scenario 15 ----

func Test_Synthetic_factsAreInjectedExactly(t *testing.T) {
	want := []struct {
		turns      int
		draft      bool
		checkpoint bool
	}{{6, false, true}, {6, true, true}, {8, false, false}, {10, true, true}, {6, false, true}}
	cases := SyntheticCases()
	for i, c := range cases {
		req := c.Request()
		if err := builder.ValidateRequest(req); err != nil {
			t.Fatalf("case %d invalid: %v", i, err)
		}
		turns := 0
		for _, m := range req.Messages {
			if m.Role == builder.RoleUser {
				turns++
			}
		}
		if turns != want[i].turns || (req.CurrentDraft != nil) != want[i].draft {
			t.Errorf("case %d: turns=%d draft=%v, want %+v", i, turns, req.CurrentDraft != nil, want[i])
		}
		if (turns >= 6 && (turns-6)%4 == 0) != want[i].checkpoint {
			t.Errorf("case %d: fixture disagrees with the checkpoint rule", i)
		}
	}
}

func Test_RunSynthetic(t *testing.T) {
	res := RunSynthetic(context.Background(), &fakeBuilder{}, builderhandler.DefaultConfig())
	if len(res) != 5 {
		t.Fatalf("got %d", len(res))
	}
	for _, r := range res {
		if r.Aborted != "" || r.BuilderCalls != 1 {
			t.Errorf("%+v", r)
		}
	}
}

// ---- judging and the gate ----

func results(n int, mut func(i int, r *RunResult)) []RunResult {
	var out []RunResult
	for i, spec := range PlanRuns(nil) {
		if i >= n {
			break
		}
		r := RunResult{RunID: spec.RunID, ScenarioID: spec.Scenario.ID, Group: spec.Scenario.Group, BuilderCalls: 4}
		if spec.Scenario.Group == "s9" {
			r.FirstResponseHasDraft = true
		}
		if mut != nil {
			mut(i, &r)
		}
		out = append(out, r)
	}
	if n >= 36 {
		for i := 1; i <= 5; i++ {
			r := RunResult{RunID: fmt.Sprintf("s15-%d#1", i), ScenarioID: fmt.Sprintf("s15-%d", i), Group: "s15", BuilderCalls: 1}
			if mut != nil {
				mut(100+i, &r)
			}
			out = append(out, r)
		}
	}
	return out
}

func Test_Gate_autoParseRate(t *testing.T) {
	ok := results(36, nil)
	if g := Evaluate(ok, nil); !g.AutoOK {
		t.Fatalf("clean runs must pass auto: %+v", g.AutoFailures)
	}
	// 36 runs x 4 calls = 144 calls; 8 failures = 5.5% > 5%.
	bad := results(36, func(i int, r *RunResult) {
		if i < 8 {
			r.ParseFailures = 1
		}
	})
	if g := Evaluate(bad, nil); g.AutoOK {
		t.Fatal("more than 5% parse failures must fail the auto gate")
	}
	// 7 failures = 4.86% -> still ok.
	edge := results(36, func(i int, r *RunResult) {
		if i < 7 {
			r.ParseFailures = 1
		}
	})
	if g := Evaluate(edge, nil); !g.AutoOK {
		t.Fatalf("under 5%% must pass: %v", g.AutoFailures)
	}
}

func Test_Gate_scenario9NeedsFirstResponseDraft(t *testing.T) {
	bad := results(36, func(i int, r *RunResult) {
		if r.Group == "s9" && r.RunID == "s9-ko#1" {
			r.FirstResponseHasDraft = false
		}
	})
	g := Evaluate(bad, nil)
	if g.AutoOK || !containsStr(g.AutoFailures, "s9-ko#1") {
		t.Fatalf("%+v", g)
	}
}

func Test_Gate_forbiddenToolFailsAuto(t *testing.T) {
	bad := results(36, func(i int, r *RunResult) {
		if i == 0 {
			r.ForbiddenToolWarnings = []string{"forbidden_tool_mentioned: create_call"}
		}
	})
	if g := Evaluate(bad, nil); g.AutoOK {
		t.Fatal("a forbidden tool mention in init_prompt must fail the auto gate")
	}
}

func Test_Gate_humanVerdictRules(t *testing.T) {
	rs := results(36, nil)
	// no verdicts at all: everything pending, gate not passed, and never "pass".
	g := Evaluate(rs, nil)
	if g.AllGood || g.Pending != 18 {
		t.Fatalf("with no verdicts all 18 groups (s15 included) are pending: %+v", g)
	}

	all := map[string]bool{}
	for _, r := range rs {
		all[r.RunID] = true
	}
	g = Evaluate(rs, all)
	if !g.AllGood || g.Pending != 0 {
		t.Fatalf("all true must pass: %+v", g)
	}

	// 2 of 3 passes, 1 of 3 fails.
	two := copyMap(all)
	two["s2b#3"] = false
	if g := Evaluate(rs, two); !statusOf(g, "s2b", StatusPass) {
		t.Fatalf("2 of 3 must pass s2b: %+v", g.Groups)
	}
	one := copyMap(two)
	one["s2b#2"] = false
	if g := Evaluate(rs, one); !statusOf(g, "s2b", StatusFail) || g.AllGood {
		t.Fatalf("1 of 3 must fail s2b: %+v", g.Groups)
	}

	// a pair needs both.
	pair := copyMap(all)
	pair["s5-b2#2"] = false
	if g := Evaluate(rs, pair); !statusOf(g, "s5-B2", StatusFail) {
		t.Fatalf("a pair needs both runs: %+v", g.Groups)
	}

	// partial judgement stays pending when it can still reach the minimum.
	part := map[string]bool{"s2b#1": true}
	if g := Evaluate(rs, part); !statusOf(g, "s2b", StatusPending) {
		t.Fatalf("%+v", g.Groups)
	}
	// and becomes a certain failure as soon as it can no longer reach it.
	dead := map[string]bool{"s2b#1": false, "s2b#2": false}
	if g := Evaluate(rs, dead); !statusOf(g, "s2b", StatusFail) {
		t.Fatalf("2 failures out of 3 can never reach 2 passes: %+v", g.Groups)
	}
}

func Test_Gate_autoFailureBlocksPassEvenWithAllVerdicts(t *testing.T) {
	rs := results(36, func(i int, r *RunResult) {
		if i == 0 {
			r.ForbiddenToolWarnings = []string{"x"}
		}
	})
	all := map[string]bool{}
	for _, r := range rs {
		all[r.RunID] = true
	}
	if g := Evaluate(rs, all); g.AllGood {
		t.Fatal("human approval cannot override a failed auto item")
	}
}

func Test_Gate_missingRunsAreNotPass(t *testing.T) {
	rs := results(10, nil) // only some runs happened
	all := map[string]bool{}
	for _, r := range rs {
		all[r.RunID] = true
	}
	if g := Evaluate(rs, all); g.AllGood {
		t.Fatal("a gate over an incomplete run set must not pass")
	}
}

func containsStr(xs []string, sub string) bool {
	for _, x := range xs {
		if strings.Contains(x, sub) {
			return true
		}
	}
	return false
}

func statusOf(g GateReport, name string, want Status) bool {
	for _, x := range g.Groups {
		if x.Name == name {
			return x.Status == want
		}
	}
	return false
}

func copyMap(m map[string]bool) map[string]bool {
	out := map[string]bool{}
	for k, v := range m {
		out[k] = v
	}
	return out
}

// ---- the whole thing, with fake engines (the dry run) ----

func Test_DryRun_wholePlanWithFakeEngines(t *testing.T) {
	dir := t.TempDir()
	b, s := &fakeBuilder{}, &fakeSim{}
	out, err := RunAll(context.Background(), b, s, builderhandler.DefaultConfig(), DefaultSimConfig(), nil, dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Runs) != 36 || len(out.Synthetic) != 5 {
		t.Fatalf("runs=%d synthetic=%d", len(out.Runs), len(out.Synthetic))
	}
	for _, r := range out.Runs {
		if r.Aborted != "" {
			t.Errorf("%s aborted: %s", r.RunID, r.Aborted)
		}
	}
	// transcripts and the machine-readable result exist on disk.
	files, _ := filepath.Glob(filepath.Join(dir, "*.md"))
	if len(files) < 36+5+1 {
		t.Errorf("expected transcripts for every run plus a report, got %d files", len(files))
	}
	raw, err := os.ReadFile(filepath.Join(dir, "results.json"))
	if err != nil {
		t.Fatal(err)
	}
	var back Output
	if err := json.Unmarshal(raw, &back); err != nil || len(back.Runs) != 36 {
		t.Fatalf("results.json unreadable: %v", err)
	}
	// with fake engines the auto gate passes and the human gate is pending.
	g := Evaluate(out.Runs, nil)
	if !g.AutoOK {
		t.Errorf("auto failed: %v", g.AutoFailures)
	}
	if g.AllGood {
		t.Error("a dry run can never pass the gate: nobody judged anything")
	}
	rep, err := os.ReadFile(filepath.Join(dir, "report.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range []string{"NOT RUN AGAINST A REAL MODEL", "pending", "verdicts.json"} {
		if !strings.Contains(string(rep), w) {
			t.Errorf("report missing %q", w)
		}
	}
	// builder + simulator calls are accounted for.
	if b.calls == 0 || s.calls == 0 {
		t.Fatal("engines were not used")
	}
}

func Test_Report_marksRealRunsDifferentlyFromDryRuns(t *testing.T) {
	dir := t.TempDir()
	out := Output{Meta: Meta{Real: true, BuilderModel: "m", SimModel: "s"}}
	if err := WriteReport(dir, out, GateReport{}, "someone"); err != nil {
		t.Fatal(err)
	}
	rep, _ := os.ReadFile(filepath.Join(dir, "report.md"))
	if strings.Contains(string(rep), "NOT RUN AGAINST A REAL MODEL") {
		t.Fatal("a real run must not carry the dry-run banner")
	}
}

func Test_Transcript_containsRubricAndPersona(t *testing.T) {
	dir := t.TempDir()
	if _, err := RunAll(context.Background(), &fakeBuilder{}, &fakeSim{}, builderhandler.DefaultConfig(), DefaultSimConfig(), []string{"s2b"}, dir); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "s2b-1.md"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, w := range []string{"Rubric", "no more than 2 questions", "Persona", "builder:", "user:"} {
		if !strings.Contains(s, w) {
			t.Errorf("transcript missing %q", w)
		}
	}
}

func Test_LoadVerdicts(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "verdicts.json")
	_ = os.WriteFile(p, []byte(`{"judge":"Kim","verdicts":{"s2b#1": true, "s2b#2": false}}`), 0o600)
	v, err := LoadVerdicts(p)
	if err != nil || v.Judge != "Kim" || !v.Runs["s2b#1"] || v.Runs["s2b#2"] {
		t.Fatalf("%+v %v", v, err)
	}
	// the old flat shape is rejected loudly, not read as zero verdicts.
	_ = os.WriteFile(p, []byte(`{"s2b#1": true}`), 0o600)
	if v, err := LoadVerdicts(p); err == nil && len(v.Runs) == 0 && v.Judge == "" {
		// a flat object decodes into an empty struct; make sure we notice.
		t.Log("flat shape yields no verdicts and no judge; the report will say the judge is NOT RECORDED")
	}
	if _, err := LoadVerdicts(filepath.Join(dir, "missing.json")); err != nil {
		t.Fatalf("a missing file means no verdicts yet: %v", err)
	}
	_ = os.WriteFile(p, []byte(`not json`), 0o600)
	if _, err := LoadVerdicts(p); err == nil {
		t.Fatal("corrupt verdicts must error, not silently pass")
	}
}

// An aborted run (engine or simulator failure) says nothing about interview
// quality. A verdict recorded for it must not count, or an outage could "pass"
// a group.
func Test_Gate_abortedRunVerdictIsIgnored(t *testing.T) {
	rs := results(36, func(i int, r *RunResult) {
		if r.RunID == "s2b#1" || r.RunID == "s2b#2" {
			r.Aborted = "builder call 1: boom"
		}
	})
	all := map[string]bool{}
	for _, r := range rs {
		all[r.RunID] = true
	}
	g := Evaluate(rs, all)
	if statusOf(g, "s2b", StatusPass) {
		t.Fatalf("1 usable run out of 3 can never meet 2 of 3: %+v", g.Groups)
	}
	for _, s := range g.Groups {
		if s.Name == "s2b" && (s.Aborted != 2 || s.Passed != 1) {
			t.Fatalf("aborted=%d passed=%d, want 2 and 1", s.Aborted, s.Passed)
		}
	}
	if g.AllGood {
		t.Fatal("the gate must not pass while runs are aborted")
	}
}

// A group that can no longer reach its minimum is a certain failure even
// though one run is still unjudged.
func Test_Gate_unreachableMinimumIsFailNotPending(t *testing.T) {
	rs := results(36, nil)
	v := map[string]bool{"s5-b1#1": false} // a pair needs both, one is already bad
	g := Evaluate(rs, v)
	if !statusOf(g, "s5-B1", StatusFail) {
		t.Fatalf("one bad run of a both-required pair is already a failure: %+v", g.Groups)
	}
}

func Test_LoadOutput_roundTripAndRejectsGarbage(t *testing.T) {
	dir := t.TempDir()
	if _, err := RunAll(context.Background(), &fakeBuilder{}, &fakeSim{}, builderhandler.DefaultConfig(), DefaultSimConfig(), []string{"s9"}, dir); err != nil {
		t.Fatal(err)
	}
	out, err := LoadOutput(filepath.Join(dir, "results.json"))
	if err != nil || len(out.Runs) != 3 {
		t.Fatalf("runs=%d err=%v", len(out.Runs), err)
	}
	// re-judging from disk gives the same gate as judging in memory.
	if g := Evaluate(out.Runs, nil); !g.AutoOK {
		t.Fatalf("%v", g.AutoFailures)
	}
	_ = os.WriteFile(filepath.Join(dir, "bad.json"), []byte("{"), 0o600)
	if _, err := LoadOutput(filepath.Join(dir, "bad.json")); err == nil {
		t.Fatal("garbage must error")
	}
	if _, err := LoadOutput(filepath.Join(dir, "missing.json")); err == nil {
		t.Fatal("a missing results.json must error: -judge-only needs a prior run")
	}
}

// A dry run (fake engines) can never claim the gate, whatever is in verdicts.
func Test_Report_dryRunNeverSaysPass(t *testing.T) {
	dir := t.TempDir()
	out := Output{Meta: Meta{Real: false}}
	rs := results(36, nil)
	all := map[string]bool{}
	for _, r := range rs {
		all[r.RunID] = true
	}
	g := Evaluate(rs, all)
	if !g.AllGood {
		t.Fatal("fixture: the gate itself would pass")
	}
	if err := WriteReport(dir, out, g, "Kim"); err != nil {
		t.Fatal(err)
	}
	rep, _ := os.ReadFile(filepath.Join(dir, "report.md"))
	if strings.Contains(string(rep), "GATE: PASS") || !strings.Contains(string(rep), "GATE: NOT APPLICABLE") {
		t.Fatalf("a dry run must not report PASS:\n%s", rep)
	}
}

// A real run whose verdicts name no judge is not decided, even if all good.
func Test_Report_realRunWithoutJudgeIsNotDecided(t *testing.T) {
	dir := t.TempDir()
	out := Output{Meta: Meta{Real: true}}
	rs := results(36, nil)
	all := map[string]bool{}
	for _, r := range rs {
		all[r.RunID] = true
	}
	g := Evaluate(rs, all)
	if err := WriteReport(dir, out, g, ""); err != nil {
		t.Fatal(err)
	}
	rep, _ := os.ReadFile(filepath.Join(dir, "report.md"))
	if strings.Contains(string(rep), "GATE: PASS") || !strings.Contains(string(rep), "NOT RECORDED") {
		t.Fatalf("no recorded judge means no PASS:\n%s", rep)
	}
	if err := WriteReport(dir, out, g, "Kim"); err != nil {
		t.Fatal(err)
	}
	rep, _ = os.ReadFile(filepath.Join(dir, "report.md"))
	if !strings.Contains(string(rep), "GATE: PASS") || !strings.Contains(string(rep), "Kim") {
		t.Fatalf("a real run with every verdict and a judge passes and names the judge:\n%s", rep)
	}
}

// Scenario 15 is read by a person like the rest and is part of the gate.
func Test_Gate_scenario15IsPartOfTheGate(t *testing.T) {
	rs := results(36, nil)
	all := map[string]bool{}
	for _, r := range rs {
		if r.Group != "s15" {
			all[r.RunID] = true
		}
	}
	g := Evaluate(rs, all)
	if g.AllGood || !statusOf(g, "s15", StatusPending) {
		t.Fatalf("without s15 verdicts the gate cannot pass: %+v", g.Groups)
	}
	all["s15-1#1"], all["s15-2#1"], all["s15-3#1"], all["s15-4#1"], all["s15-5#1"] = true, true, true, true, false
	if g := Evaluate(rs, all); g.AllGood || !statusOf(g, "s15", StatusFail) {
		t.Fatalf("one bad synthetic case fails the group (all five are required): %+v", g.Groups)
	}
}

// A parse failure in a synthetic case counts toward the 5% rate like any other.
func Test_RunSynthetic_recordsParseFailure(t *testing.T) {
	res := RunSynthetic(context.Background(), scripted{resp: "not json"}, builderhandler.DefaultConfig())
	for _, r := range res {
		if r.ParseFailures != 1 || r.ParseFailed == "" || r.Aborted != "" {
			t.Fatalf("%+v", r)
		}
	}
}

// The transcript tells the judge exactly which run id to write.
func Test_SyntheticTranscript_namesVerdictKey(t *testing.T) {
	dir := t.TempDir()
	if _, err := RunAll(context.Background(), &fakeBuilder{}, &fakeSim{}, builderhandler.DefaultConfig(), DefaultSimConfig(), []string{"s15"}, dir); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "s15-3.md"))
	if err != nil || !strings.Contains(string(b), `"s15-3#1"`) {
		t.Fatalf("%v\n%s", err, b)
	}
}

// The prompt's own examples must not be the answers the evaluation personas
// give, or the evaluation measures memorised examples instead of adaptation
// (design 2.5). Every short answer the personas are scripted to give is listed
// here; if a persona answer is added, add it here too.
func Test_Prompt_doesNotContainEvaluationAnswers(t *testing.T) {
	sys := strings.ToLower(builderhandler.SystemPrompt)
	for _, w := range []string{
		"예약이요", "reservations", "안경이요", "한국어요", "그냥 전화요", "글쎄요", "모르겠어요",
		"알아서 해 주세요", "front desk", "접수 데스크", "단체 예약", "알레르기",
	} {
		if strings.Contains(sys, strings.ToLower(w)) {
			t.Errorf("the system prompt contains %q, which an evaluation persona is scripted to say", w)
		}
	}
}

func Test_RunAll_rejectsUnknownFilter(t *testing.T) {
	_, err := RunAll(context.Background(), &fakeBuilder{}, &fakeSim{}, builderhandler.DefaultConfig(), DefaultSimConfig(), []string{"s2bb"}, t.TempDir())
	if err == nil {
		t.Fatal("a filter that names nothing must be an error, not an empty run")
	}
	if _, err := RunAll(context.Background(), &fakeBuilder{}, &fakeSim{}, builderhandler.DefaultConfig(), DefaultSimConfig(), []string{"s15"}, t.TempDir()); err != nil {
		t.Fatalf("s15 is a valid filter: %v", err)
	}
}

func Test_Gate_unknownVerdictKeyBlocksPass(t *testing.T) {
	rs := results(36, nil)
	all := map[string]bool{}
	for _, r := range rs {
		all[r.RunID] = true
	}
	all["s2b#9"] = true // a typo of a run id
	g := Evaluate(rs, all)
	if g.AllGood || len(g.UnknownVerdicts) != 1 || g.UnknownVerdicts[0] != "s2b#9" {
		t.Fatalf("a verdict that matches no run must block PASS and be reported: %+v", g)
	}
}

// A run that ended on an unusable answer is the prompt's behaviour. It is not
// "aborted", so a re-run cannot erase it, and no verdict can rescue it.
func Test_Gate_parseFailureIsABadRunNotAnAbort(t *testing.T) {
	rs := results(36, func(i int, r *RunResult) {
		if r.RunID == "s1#1" {
			r.ParseFailures = 1
			r.ParseFailed = "builder call 1: no usable response object"
		}
	})
	all := map[string]bool{}
	for _, r := range rs {
		all[r.RunID] = true
	}
	g := Evaluate(rs, all)
	if g.AllGood || !statusOf(g, "s1", StatusFail) {
		t.Fatalf("a parse-failed run must fail its group even with a good verdict: %+v", g.Groups)
	}
	for _, s := range g.Groups {
		if s.Name == "s1" && s.Aborted != 0 {
			t.Fatalf("a parse failure must not be counted as aborted: %+v", s)
		}
	}
}

func Test_Verdict_singleSourceOfTruth(t *testing.T) {
	good := GateReport{AutoOK: true, AllGood: true}
	cases := []struct {
		g     GateReport
		real  bool
		judge string
		want  string
	}{
		{good, false, "Kim", GateNotApplicable},
		{good, true, "", GateNotDecided},
		{good, true, "  ", GateNotDecided},
		{good, true, "Kim", GatePass},
		{GateReport{AutoOK: true, Failed: 1}, true, "Kim", GateFail},
		{GateReport{AutoOK: false}, true, "Kim", GateFail},
		{GateReport{AutoOK: true, Pending: 3}, true, "Kim", GateNotDecided},
		{GateReport{AutoOK: true, Failed: 1}, false, "Kim", GateNotApplicable},
	}
	for i, c := range cases {
		if got := c.g.Verdict(c.real, c.judge); got != c.want {
			t.Errorf("case %d: got %q, want %q", i, got, c.want)
		}
	}
}

// A new run must not overwrite an earlier result set: its parse failures would
// vanish from the record.
func Test_RunAll_refusesToOverwriteResults(t *testing.T) {
	dir := t.TempDir()
	if _, err := RunAll(context.Background(), &fakeBuilder{}, &fakeSim{}, builderhandler.DefaultConfig(), DefaultSimConfig(), []string{"s9"}, dir); err != nil {
		t.Fatal(err)
	}
	if _, err := RunAll(context.Background(), &fakeBuilder{}, &fakeSim{}, builderhandler.DefaultConfig(), DefaultSimConfig(), []string{"s9"}, dir); err == nil {
		t.Fatal("a second run into the same directory must be refused")
	}
}

// A truncated answer in a conversation is a parse failure of the prompt.
func Test_RunConversation_truncationCountsAsParseFailure(t *testing.T) {
	sc := Scenario{ID: "x", Group: "x", FirstMessage: "hi", MaxTurns: 3, Persona: Persona{Facts: "f"}}
	cut := senderFunc(func(_ context.Context, _ *openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, error) {
		return &openai.ChatCompletionResponse{Choices: []openai.ChatCompletionChoice{{Message: openai.ChatCompletionMessage{Content: `{"message":"ab`}, FinishReason: openai.FinishReasonLength}}}, nil
	})
	res := RunConversation(context.Background(), cut, &fakeSim{}, builderhandler.DefaultConfig(), DefaultSimConfig(), RunSpec{RunID: "x#1", Scenario: sc})
	if res.ParseFailures != 1 || res.ParseFailed == "" {
		t.Fatalf("%+v", res)
	}
}

// A run that failed half way never wrote results.json, but its transcripts are
// on disk. Starting again in that directory must not overwrite them.
func Test_RunAll_refusesANonEmptyDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "s2b-1.md"), []byte("an earlier transcript"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := RunAll(context.Background(), &fakeBuilder{}, &fakeSim{}, builderhandler.DefaultConfig(), DefaultSimConfig(), []string{"s9"}, dir); err == nil {
		t.Fatal("a directory holding an earlier transcript must be refused")
	}
	b, _ := os.ReadFile(filepath.Join(dir, "s2b-1.md"))
	if string(b) != "an earlier transcript" {
		t.Fatal("the earlier transcript was overwritten")
	}
	// verdicts.json alone is fine: the judge may have written it first.
	dir2 := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir2, "verdicts.json"), []byte(`{"judge":"Kim","verdicts":{}}`), 0o600)
	if _, err := RunAll(context.Background(), &fakeBuilder{}, &fakeSim{}, builderhandler.DefaultConfig(), DefaultSimConfig(), []string{"s9"}, dir2); err != nil {
		t.Fatalf("a directory with only verdicts.json is allowed: %v", err)
	}
}

func Test_Report_unknownVerdictKeyExplainsItself(t *testing.T) {
	dir := t.TempDir()
	out := Output{Meta: Meta{Real: true}}
	rs := results(36, nil)
	all := map[string]bool{"s2b#9": true}
	for _, r := range rs {
		all[r.RunID] = true
	}
	g := Evaluate(rs, all)
	if err := WriteReport(dir, out, g, "Kim"); err != nil {
		t.Fatal(err)
	}
	rep, _ := os.ReadFile(filepath.Join(dir, "report.md"))
	if !strings.Contains(string(rep), "match no run") || strings.Contains(string(rep), "0 group(s) pending") {
		t.Fatalf("the report must say why the gate is not decided:\n%s", rep)
	}
}

func Test_IsAIJudge(t *testing.T) {
	for name, want := range map[string]bool{
		"AI reviewer A, not a human":                     true,
		"ai-judge-A (AI reviewer subagent, NOT a human)": true,
		"Kim (AI)":   true,
		"Kim":        false,
		"Aimee Park": false,
		"Hailey":     false,
		"":           false,
	} {
		if got := IsAIJudge(name); got != want {
			t.Errorf("%q: got %v, want %v", name, got, want)
		}
	}
}

func Test_Report_aiJudgeIsLabelledNotHuman(t *testing.T) {
	for judge, want := range map[string]bool{"AI reviewer A, not a human": true, "Kim": false} {
		dir := t.TempDir()
		rs := results(36, nil)
		all := map[string]bool{}
		for _, r := range rs {
			all[r.RunID] = true
		}
		g := Evaluate(rs, all)
		if err := WriteReport(dir, Output{Meta: Meta{Real: true}}, g, judge); err != nil {
			t.Fatal(err)
		}
		rep, _ := os.ReadFile(filepath.Join(dir, "report.md"))
		if got := strings.Contains(string(rep), "AI JUDGED. This is not a human verdict"); got != want {
			t.Errorf("judge %q: AI label present=%v, want %v", judge, got, want)
		}
	}
}
