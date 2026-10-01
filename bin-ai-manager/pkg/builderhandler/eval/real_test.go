//go:build builder_eval

// Real-model entry point of the evaluation harness. Excluded from normal builds
// and from CI: it needs a real API key and spends money.
//
//	cd bin-ai-manager
//	BUILDER_EVAL_API_KEY=... \
//	BUILDER_EVAL_MODEL=gemini-3.8-flash BUILDER_EVAL_SIM_MODEL=<a different model> \
//	go test -tags builder_eval -run Test_RealEvaluation -v -timeout 60m \
//	  ./pkg/builderhandler/eval/ -args -out /path/to/out
//
// See README.md in this directory for what to do with the output.
package eval

import (
	"context"
	"flag"
	"os"
	"strings"
	"testing"
	"time"

	"monorepo/bin-ai-manager/pkg/builderhandler"
	"monorepo/bin-ai-manager/pkg/engine_openai_handler"
)

var (
	flagOut    = flag.String("out", "", "output directory (required)")
	flagOnly   = flag.String("only", "", "comma separated scenario ids or group names to run (default: all)")
	flagEffort = flag.String("effort", "none", "builder reasoning_effort")
	flagJSON   = flag.String("json-mode", "json_schema", "json_schema | json_object | none")
	flagSysBlk = flag.Bool("data-block-in-system", false, "evaluation variant: merge the session facts into the system prompt")
	flagJudge  = flag.Bool("judge-only", false, "do not call any model: recompute report.md from results.json and verdicts.json")
)

const defaultBaseURL = "https://generativelanguage.googleapis.com/v1beta/openai/"

func Test_RealEvaluation(t *testing.T) {
	if *flagOut == "" {
		t.Skip("set -args -out <dir> to run the real evaluation")
	}
	if *flagJudge {
		judgeOnly(t, *flagOut)
		return
	}

	key := os.Getenv("BUILDER_EVAL_API_KEY")
	model := os.Getenv("BUILDER_EVAL_MODEL")
	simModel := os.Getenv("BUILDER_EVAL_SIM_MODEL")
	if key == "" || model == "" || simModel == "" {
		t.Skip("BUILDER_EVAL_API_KEY, BUILDER_EVAL_MODEL and BUILDER_EVAL_SIM_MODEL must all be set")
	}
	if model == simModel {
		t.Fatal("the user simulator must use a different model from the builder")
	}
	base := os.Getenv("BUILDER_EVAL_BASE_URL")
	if base == "" {
		base = defaultBaseURL
	}

	var sender builderhandler.Sender = engine_openai_handler.NewEngineOpenaiHandlerWithConfig(key, base)

	cfg := builderhandler.DefaultConfig()
	cfg.Model = model
	cfg.ReasoningEffort = *flagEffort
	switch m := builderhandler.JSONMode(*flagJSON); m {
	case builderhandler.JSONModeSchema, builderhandler.JSONModeObject, builderhandler.JSONModeNone:
		cfg.JSONMode = m
	default:
		t.Fatalf("-json-mode must be json_schema, json_object or none, got %q", *flagJSON)
	}
	cfg.DataBlockInSystem = *flagSysBlk
	if v := os.Getenv("BUILDER_EVAL_SYSTEM_PROMPT_FILE"); v != "" {
		b, err := os.ReadFile(v)
		if err != nil {
			t.Fatal(err)
		}
		cfg.SystemPrompt = string(b)
	}
	cfg.LLMTimeout = 90 * time.Second

	simCfg := DefaultSimConfig()
	simCfg.Model = simModel

	var only []string
	if *flagOnly != "" {
		only = strings.Split(*flagOnly, ",")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 55*time.Minute)
	defer cancel()
	out, err := RunAllWithMeta(ctx, sender, sender, cfg, simCfg, only, *flagOut,
		Meta{Real: true, BuilderModel: model, SimModel: simModel})
	if err != nil {
		t.Fatalf("evaluation stopped early: %v", err)
	}
	t.Logf("done: %d runs, %d synthetic cases. Read %s/report.md, judge the transcripts, write verdicts.json, then run again with -judge-only.", len(out.Runs), len(out.Synthetic), *flagOut)
}

func judgeOnly(t *testing.T, dir string) {
	t.Helper()
	out, err := LoadOutput(dir + "/results.json")
	if err != nil {
		t.Fatal(err)
	}
	verdicts, err := LoadVerdicts(dir + "/verdicts.json")
	if err != nil {
		t.Fatal(err)
	}
	all := append(append([]RunResult{}, out.Runs...), out.Synthetic...)
	g := Evaluate(all, verdicts.Runs)
	if err := WriteReport(dir, out, g, verdicts.Judge); err != nil {
		t.Fatal(err)
	}
	t.Logf("report rewritten: pass=%t pending=%d failed=%d", g.Pass, g.Pending, g.Failed)
}
