package eval

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"monorepo/bin-ai-manager/models/builder"
	"monorepo/bin-ai-manager/pkg/builderhandler"
)

// RunResult is the record of one run. Everything the gate needs is a plain
// field, so results.json can be re-judged later without re-running.
type RunResult struct {
	RunID      string `json:"run_id"`
	ScenarioID string `json:"scenario_id"`
	Group      string `json:"group"`
	Title      string `json:"title"`

	Turns []builder.Message `json:"turns"`

	BuilderCalls  int `json:"builder_calls"`
	ParseFailures int `json:"parse_failures"`

	FinalDraft            *builder.Draft `json:"final_draft,omitempty"`
	FirstResponseHasDraft bool           `json:"first_response_has_draft"`
	ForbiddenToolWarnings []string       `json:"forbidden_tool_warnings,omitempty"`
	Warnings              []string       `json:"warnings,omitempty"`
	Assumptions           []string       `json:"assumptions,omitempty"`

	// Aborted is non-empty when the run stopped early for a reason that says
	// nothing about interview quality (engine error, simulator failure). Such a
	// run is not judgeable and must be re-run.
	Aborted string `json:"aborted,omitempty"`

	BuilderPromptTokens     int `json:"builder_prompt_tokens"`
	BuilderCompletionTokens int `json:"builder_completion_tokens"`
	SimPromptTokens         int `json:"sim_prompt_tokens"`
	SimCompletionTokens     int `json:"sim_completion_tokens"`
}

const warnForbiddenPrefix = builderhandler.WarnForbiddenToolMentioned

// RunConversation plays one scenario: the builder (production RunTurn) and the
// user simulator alternate until the first draft (plus ContinueAfterDraft more
// turns), the turn cap, or a failure.
func RunConversation(ctx context.Context, builderSender, simSender builderhandler.Sender, cfg builderhandler.Config, simCfg SimConfig, spec RunSpec) RunResult {
	sc := spec.Scenario
	res := RunResult{RunID: spec.RunID, ScenarioID: sc.ID, Group: sc.Group, Title: sc.Title}
	res.Turns = []builder.Message{{Role: builder.RoleUser, Content: sc.FirstMessage}}

	var current *builder.Draft
	drafted := false
	extra := 0

	for call := 1; call <= sc.MaxTurns; call++ {
		req := &builder.ChatRequest{Messages: append([]builder.Message(nil), res.Turns...), CurrentDraft: current}
		turn, err := builderhandler.RunTurn(ctx, builderSender, cfg, req)
		res.BuilderCalls++
		if turn != nil {
			res.BuilderPromptTokens += turn.Usage.PromptTokens
			res.BuilderCompletionTokens += turn.Usage.CompletionTokens
		}
		if err != nil {
			if errors.Is(err, builderhandler.ErrInvalidResponse) || errors.Is(err, builderhandler.ErrTruncated) {
				res.ParseFailures++
			}
			res.Aborted = fmt.Sprintf("builder call %d: %v", call, err)
			return res
		}

		p := turn.Parsed
		res.Turns = append(res.Turns, builder.Message{Role: builder.RoleAssistant, Content: p.Message})
		for _, w := range p.Warnings {
			res.Warnings = append(res.Warnings, w)
			if strings.HasPrefix(w, warnForbiddenPrefix) {
				res.ForbiddenToolWarnings = append(res.ForbiddenToolWarnings, w)
			}
		}

		if p.Draft != nil {
			current = p.Draft
			res.FinalDraft = p.Draft
			res.Assumptions = p.Assumptions
			if call == 1 {
				res.FirstResponseHasDraft = true
			}
			if !drafted {
				drafted = true
				extra = sc.ContinueAfterDraft
			} else {
				extra--
			}
		} else if drafted {
			extra--
		}
		if drafted && extra <= 0 {
			return res
		}
		if call == sc.MaxTurns {
			return res
		}

		reply, usage, err := SimulateUser(ctx, simSender, simCfg, sc.Persona, res.Turns)
		res.SimPromptTokens += usage.PromptTokens
		res.SimCompletionTokens += usage.CompletionTokens
		if err != nil {
			res.Aborted = err.Error()
			return res
		}
		res.Turns = append(res.Turns, builder.Message{Role: builder.RoleUser, Content: reply})
	}
	return res
}
