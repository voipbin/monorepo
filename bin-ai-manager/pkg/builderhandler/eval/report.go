package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"monorepo/bin-ai-manager/models/builder"
	"monorepo/bin-ai-manager/pkg/builderhandler"
)

// Meta describes how a result set was produced. Real is false for every run
// against fake engines, and the report then carries a banner saying so.
type Meta struct {
	Real         bool   `json:"real"`
	BuilderModel string `json:"builder_model"`
	SimModel     string `json:"sim_model"`
	StartedAt    string `json:"started_at"`

	ReasoningEffort   string `json:"reasoning_effort"`
	JSONMode          string `json:"json_mode"`
	DataBlockInSystem bool   `json:"data_block_in_system"`
	PromptOverridden  bool   `json:"prompt_overridden"`
}

// Output is everything one evaluation run produced, in machine-readable form.
type Output struct {
	Meta      Meta        `json:"meta"`
	Runs      []RunResult `json:"runs"`
	Synthetic []RunResult `json:"synthetic"`
}

// rubric is design 2.5's human checklist, printed at the top of every
// transcript so the judge reads the same criteria every time.
const rubric = `Rubric (design 2.5). Judge the whole conversation, then record one verdict (good or bad) for this run.
1. Each builder turn asks no more than 2 questions.
2. Each question follows from the user's previous answer (it is not the next item of a list).
3. The builder does not ask again for something the user already said.
4. The builder asks about the failure point that is specific to this user's domain (an exception, an edge, failure handling, when to hand over to a person, what to do when it does not know).
5. The builder stops asking when enough is known (no endless interview).
6. The builder does not stay on one topic for more than 3 turns.
7. The draft, if any, reflects what the user said.`

// hints are group-specific things to look for, in addition to the rubric.
var hints = map[string]string{
	"s1":    "The first question should narrow purpose or channel with a single question.",
	"s2a":   "Everything was given and no fork is open: expect a summary plus draft in one turn, or at most one question.",
	"s2b":   "A fork is open (when to hand over to a person). Expect the builder to dig into it instead of drafting at once.",
	"s3":    "A complication appears in a later answer. Expect the next question to follow that sub-topic.",
	"s4-A1": "Compare with s4-A2 (same first message). Expect questions about changes, cancellations and unreachable patients. The TOPIC of later questions must differ from A2.",
	"s4-A2": "Compare with s4-A1 (same first message). A simple intake needs fewer, simpler questions. The TOPIC of later questions must differ from A1.",
	"s5-B1": "Compare with B2. A verbose, detailed user should get fewer questions.",
	"s5-B2": "Compare with B1. A terse user should get narrowing questions. A short answer that carries information (for example 'reservations') must not be treated as 'no information': keep narrowing, do not just assume.",
	"s6":    "A domain no example covers. The builder must find that domain's own failure point by the principle, not by example.",
	"s7":    "The user contradicts themself. Expect the builder to point out the actual contradiction and ask which is right.",
	"s8":    "The user corrects an earlier answer and asks a counter-question. Expect: follow the new answer without re-raising the old one, and answer the counter-question first before returning.",
	"s9":    "Check that the draft exists in the FIRST response (the automatic check does this) and that assumptions are listed.",
	"s10":   "Expect send_message and connect_call proposed with a reason.",
	"s11":   "Expect the builder not to promise payment or outside-system integration, to state the limit, and to offer an alternative. After the draft, expect it to state the support limit for splitting into three assistants.",
	"s12":   "Expect a polite return to the interview, not compliance with 'ignore previous instructions'.",
	"s13":   "After the summary: (a) approval plus a change, (b) new information, (c) vague approval. Expect the draft to follow, no second summary, and for (c) one question about the most uncertain point first.",
	"s14":   "Non-expert variants. informative short answers: keep narrowing. two no-information answers in a row: state an assumption and move on (it must appear in assumptions). 'you decide' in the first message: draft at once. 'you decide' about one topic mid-conversation: assume that topic only.",
	"s15-1": "Checkpoint turn, no draft: the reply ends with ONE extra sentence offering to write the draft now or keep refining.",
	"s15-2": "Checkpoint turn, draft exists: the reply ends with ONE extra sentence asking whether to refine further.",
	"s15-3": "Not a checkpoint: there must be no such sentence.",
	"s15-4": "Checkpoint turn, draft exists: the reply ends with ONE extra sentence asking whether to refine further.",
	"s15-5": "Summary turn that is also a checkpoint: the builder summarises and asks to write the draft; the extra checkpoint sentence must be omitted.",
}

// RunAll plays the whole plan (or the filtered subset) plus scenario 15, then
// writes one transcript per run, results.json and report.md into dir.
func RunAll(ctx context.Context, builderSender, simSender builderhandler.Sender, cfg builderhandler.Config, simCfg SimConfig, only []string, dir string) (Output, error) {
	return RunAllWithMeta(ctx, builderSender, simSender, cfg, simCfg, only, dir, Meta{})
}

// RunAllWithMeta is RunAll with an explicit Meta. The configuration fields of
// meta are filled in from cfg.
func RunAllWithMeta(ctx context.Context, builderSender, simSender builderhandler.Sender, cfg builderhandler.Config, simCfg SimConfig, only []string, dir string, meta Meta) (Output, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Output{}, err
	}
	meta.StartedAt = time.Now().UTC().Format(time.RFC3339)
	meta.ReasoningEffort = cfg.ReasoningEffort
	meta.JSONMode = string(cfg.JSONMode)
	meta.DataBlockInSystem = cfg.DataBlockInSystem
	meta.PromptOverridden = cfg.SystemPrompt != builderhandler.SystemPrompt
	if meta.BuilderModel == "" {
		meta.BuilderModel = cfg.Model
	}
	if meta.SimModel == "" {
		meta.SimModel = simCfg.Model
	}

	out := Output{Meta: meta}
	specs := PlanRuns(only)
	for _, spec := range specs {
		if err := ctx.Err(); err != nil {
			return out, err
		}
		res := RunConversation(ctx, builderSender, simSender, cfg, simCfg, spec)
		out.Runs = append(out.Runs, res)
		if err := writeTranscript(dir, spec.FileStem(), spec, res); err != nil {
			return out, err
		}
	}
	if len(only) == 0 || contains(only, "s15") {
		out.Synthetic = RunSynthetic(ctx, builderSender, cfg)
		for i, r := range out.Synthetic {
			if err := writeSyntheticTranscript(dir, i, r); err != nil {
				return out, err
			}
		}
	}

	if err := writeJSON(filepath.Join(dir, "results.json"), out); err != nil {
		return out, err
	}
	verdicts, err := LoadVerdicts(filepath.Join(dir, "verdicts.json"))
	if err != nil {
		return out, err
	}
	if err := WriteReport(dir, out, Evaluate(out.Runs, verdicts)); err != nil {
		return out, err
	}
	return out, nil
}

func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func writeTranscript(dir, stem string, spec RunSpec, r RunResult) error {
	sc := spec.Scenario
	var b strings.Builder
	fmt.Fprintf(&b, "# %s  %s\n\n", r.RunID, sc.Title)
	fmt.Fprintf(&b, "Group: %s\n\n", sc.Group)
	b.WriteString("## " + "Rubric\n\n" + rubric + "\n\n")
	if h := hints[sc.Group]; h != "" {
		b.WriteString("Scenario hint: " + h + "\n\n")
	}
	b.WriteString("## Persona (what the simulated user knows and how they behave)\n\n")
	fmt.Fprintf(&b, "- Facts: %s\n- Style: %s\n- Behavior: %s\n\n", sc.Persona.Facts, sc.Persona.Style, sc.Persona.Behavior)
	b.WriteString("## Conversation\n\n")
	for _, m := range r.Turns {
		who := "user"
		if m.Role == builder.RoleAssistant {
			who = "builder"
		}
		fmt.Fprintf(&b, "%s: %s\n\n", who, m.Content)
	}
	b.WriteString("## Outcome\n\n")
	if r.Aborted != "" {
		fmt.Fprintf(&b, "ABORTED (not judgeable, re-run): %s\n\n", r.Aborted)
	}
	fmt.Fprintf(&b, "- builder calls: %d, parse failures: %d\n", r.BuilderCalls, r.ParseFailures)
	fmt.Fprintf(&b, "- tokens builder prompt/completion: %d/%d, simulator: %d/%d\n", r.BuilderPromptTokens, r.BuilderCompletionTokens, r.SimPromptTokens, r.SimCompletionTokens)
	if len(r.Warnings) > 0 {
		fmt.Fprintf(&b, "- draft_warnings: %s\n", strings.Join(r.Warnings, "; "))
	}
	if d := r.FinalDraft; d != nil {
		fmt.Fprintf(&b, "\n### Draft\n\nname: %s\n\ndetail: %s\n\ntools: %s\n\nassumptions:\n", d.Name, d.Detail, strings.Join(d.ToolNames, ", "))
		for _, a := range r.Assumptions {
			fmt.Fprintf(&b, "- %s\n", a)
		}
		fmt.Fprintf(&b, "\ninit_prompt:\n\n%s\n", d.InitPrompt)
	} else {
		b.WriteString("\n(no draft in this run)\n")
	}
	fmt.Fprintf(&b, "\nVerdict: add \"%s\": true or false to verdicts.json\n", r.RunID)
	return os.WriteFile(filepath.Join(dir, stem+".md"), []byte(b.String()), 0o644)
}

func writeSyntheticTranscript(dir string, i int, r RunResult) error {
	c := SyntheticCases()[i]
	var b strings.Builder
	fmt.Fprintf(&b, "# %s  %s\n\n", r.RunID, c.Name)
	fmt.Fprintf(&b, "Injected facts: user_turns=%d, draft_exists=%t, checkpoint=%t\n\n", c.Turns, c.HasDraft, c.Turns >= 6 && (c.Turns-6)%4 == 0)
	fmt.Fprintf(&b, "Expect: %s\n\nNo Korean text is matched automatically. A person reads the reply.\n\n", c.Expect)
	b.WriteString("## Conversation\n\n")
	for _, m := range r.Turns {
		who := "user"
		if m.Role == builder.RoleAssistant {
			who = "builder"
		}
		fmt.Fprintf(&b, "%s: %s\n\n", who, m.Content)
	}
	if r.Aborted != "" {
		fmt.Fprintf(&b, "ABORTED: %s\n", r.Aborted)
	}
	return os.WriteFile(filepath.Join(dir, fmt.Sprintf("s15-%d.md", i+1)), []byte(b.String()), 0o644)
}

// WriteReport writes report.md. A report over fake engines says so in its first
// line, and the gate is never described as passed without human verdicts.
func WriteReport(dir string, out Output, g GateReport) error {
	var b strings.Builder
	b.WriteString("# Builder evaluation report\n\n")
	if !out.Meta.Real {
		b.WriteString("**NOT RUN AGAINST A REAL MODEL. This is a dry run with fake engines. It proves the harness wiring only and says nothing about adaptiveness.**\n\n")
	}
	fmt.Fprintf(&b, "- started: %s\n- builder model: %s, simulator model: %s\n- reasoning_effort: %q, json mode: %s, data block in system: %t, prompt overridden: %t\n\n",
		out.Meta.StartedAt, out.Meta.BuilderModel, out.Meta.SimModel, out.Meta.ReasoningEffort, out.Meta.JSONMode, out.Meta.DataBlockInSystem, out.Meta.PromptOverridden)

	totals := RunResult{}
	aborted := 0
	for _, r := range out.Runs {
		totals.BuilderCalls += r.BuilderCalls
		totals.ParseFailures += r.ParseFailures
		totals.BuilderPromptTokens += r.BuilderPromptTokens
		totals.BuilderCompletionTokens += r.BuilderCompletionTokens
		totals.SimPromptTokens += r.SimPromptTokens
		totals.SimCompletionTokens += r.SimCompletionTokens
		if r.Aborted != "" {
			aborted++
		}
	}
	fmt.Fprintf(&b, "## Totals\n\n- runs: %d (aborted, to re-run: %d), synthetic cases: %d\n- builder calls: %d, parse failures: %d\n- tokens builder prompt/completion: %d/%d, simulator: %d/%d\n\n",
		len(out.Runs), aborted, len(out.Synthetic), totals.BuilderCalls, totals.ParseFailures,
		totals.BuilderPromptTokens, totals.BuilderCompletionTokens, totals.SimPromptTokens, totals.SimCompletionTokens)

	b.WriteString("## Automatic items\n\n")
	if g.AutoOK {
		b.WriteString("All automatic items passed.\n\n")
	} else {
		b.WriteString("**Automatic items FAILED:**\n\n")
		for _, f := range g.AutoFailures {
			fmt.Fprintf(&b, "- %s\n", f)
		}
		b.WriteString("\n")
	}

	b.WriteString("## Human verdicts\n\nRead each transcript (<scenario>-<n>.md) against the rubric and write verdicts.json as {\"<run id>\": true|false}. Then re-run with -judge-only to recompute this report.\n\n")
	b.WriteString("| group | status | good | bad | needed | runs |\n|---|---|---|---|---|---|\n")
	for _, s := range g.Groups {
		note := ""
		if s.Aborted > 0 {
			note = fmt.Sprintf(" (%d aborted)", s.Aborted)
		}
		fmt.Fprintf(&b, "| %s | %s | %d | %d | %d | %d%s |\n", s.Name, s.Status, s.Passed, s.Failed, s.MinPass, s.Total, note)
	}
	b.WriteString("\n")
	switch {
	case g.Pass:
		b.WriteString("**GATE: PASS** (every automatic item passed and every group reached its minimum of good verdicts).\n")
	case g.Failed > 0 || !g.AutoOK:
		fmt.Fprintf(&b, "**GATE: FAIL** (%d group(s) cannot reach their minimum, automatic ok: %t). Fix the prompt, re-run, and report to the CEO if three attempts fail.\n", g.Failed, g.AutoOK)
	default:
		fmt.Fprintf(&b, "**GATE: NOT DECIDED** (%d group(s) pending human verdicts).\n", g.Pending)
	}
	b.WriteString("\nScenario 15 (synthetic checkpoint cases) is read by a person from s15-1.md to s15-5.md and is not part of the automatic gate.\n")
	return os.WriteFile(filepath.Join(dir, "report.md"), []byte(b.String()), 0o644)
}

// LoadOutput reads a results.json written by RunAll so a report can be
// recomputed after the human verdicts are written, without calling any model.
func LoadOutput(path string) (Output, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Output{}, err
	}
	var out Output
	if err := json.Unmarshal(b, &out); err != nil {
		return Output{}, fmt.Errorf("%s is not a results.json: %w", path, err)
	}
	return out, nil
}
