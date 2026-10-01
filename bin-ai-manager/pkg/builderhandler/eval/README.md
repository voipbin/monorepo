# Builder evaluation harness

This is the adaptiveness evaluation of the Assistant Builder (design section 2.5). It exists because **a document or a unit test cannot show that the interview adapts to the user's answers. Only reading real conversations can.**

## What it does and does not prove

- It drives the production turn code (`builderhandler.RunTurn`), not a copy, against scripted scenarios and a separate **user simulator** (a different model, high temperature, a persona sheet, some deliberately uncooperative).
- Mechanical checks (deterministic, in `gate.go`): JSON parse-failure rate at most 5%, scenario 9 returns a draft in the first response, no `init_prompt` names a forbidden tool.
- Everything else is a **human** reading the transcripts against the rubric printed at the top of each transcript. The harness never judges interview quality.
- A run with fake engines (the unit tests) proves only that the wiring works. Its report starts with `NOT RUN AGAINST A REAL MODEL`.
- The gate is never `PASS` without human verdicts.

## Cost and size

36 simulated runs (3 repeats of 2b, 3, 4-A1, 4-A2, 6; 2 each of B1 and B2; 3 for scenario 9; 3 for 13; 4 for 14; one each of 1, 2a, 7, 8, 10, 11, 12), plus 5 synthetic checkpoint cases. At an average of 6 turns that is **about 216 builder calls and a similar number of simulator calls**. The 6-turn average is an assumption, not a measurement. Human judging: about 21 full transcripts and about 8 skims, plus the 5 synthetic cases.

## Run it

```
cd bin-ai-manager
BUILDER_EVAL_API_KEY=<key> \
BUILDER_EVAL_MODEL=<builder model> \
BUILDER_EVAL_SIM_MODEL=<a DIFFERENT model> \
go test -tags builder_eval -run Test_RealEvaluation -v -timeout 60m \
  ./pkg/builderhandler/eval/ -args -out /tmp/builder-eval-1
```

Useful flags (after `-args`): `-only s1,s2b` (scenario ids or group names), `-effort none|low`, `-json-mode json_schema|json_object|none`, `-data-block-in-system`. Set `BUILDER_EVAL_SYSTEM_PROMPT_FILE` to evaluate a prompt variant. `BUILDER_EVAL_BASE_URL` defaults to the Gemini OpenAI-compatible endpoint.

## Judge

1. Read `report.md`, then each `<scenario>-<n>.md` and `s15-<n>.md`.
2. Write `verdicts.json` next to them: `{"s2b#1": true, "s2b#2": false, ...}`. The judge should not be the person who wrote the prompt.
3. Recompute the report without calling any model:
   `go test -tags builder_eval -run Test_RealEvaluation -v ./pkg/builderhandler/eval/ -args -out /tmp/builder-eval-1 -judge-only`
4. A run marked ABORTED stopped on an engine or simulator error and says nothing about quality. Re-run it. A verdict recorded for an aborted run is ignored.

## Pass rule (design 2.5)

Every automatic item passes. Scenarios 2b, 3, 4-A1, 4-A2 and 6: at least 2 of 3 runs good. B1 and B2: both runs good. 9, 13 and 14: every variant good. The rest: their single run good. If a single-run scenario fails, re-run it once with the same prompt before changing the prompt, so a noisy run does not drive overfitting. If three attempts still fail, report to the CEO instead of continuing.

## Comparison axes (design 2.5)

Only two: `reasoning_effort` (`none` against a low value, scenarios 2b, 4 and 6) with model candidates, and the principle-only prompt against a prompt with a signal table (an evaluation-only file via `BUILDER_EVAL_SYSTEM_PROMPT_FILE`; it must never enter production). `-json-mode` and `-data-block-in-system` compare the JSON mode and where the session facts sit.

## Not covered here

Latency and the real `max_tokens`, the semaphore size, and the Gemini compatibility of the response format must be measured in the first real run and written down. They are initial values today.
