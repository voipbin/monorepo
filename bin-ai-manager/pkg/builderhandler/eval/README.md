# Builder evaluation harness

This is the adaptiveness evaluation of the Assistant Builder (design section 2.5). It exists because **a document or a unit test cannot show that the interview adapts to the user's answers. Only reading real conversations can.**

## What it does and does not prove

- It drives the production turn code (`builderhandler.RunTurn`), not a copy, against scripted scenarios and a separate **user simulator** (a different model, high temperature, a persona sheet, some deliberately uncooperative).
- Mechanical checks (deterministic, in `gate.go`): JSON parse-failure rate at most 5%, scenario 9 returns a draft in the first response, no `init_prompt` names a forbidden tool.
- Everything else is read by a **judge** (ideally a human) reading the transcripts against the rubric printed at the top of each transcript. The harness never judges interview quality.
- A run with fake engines (the unit tests) proves only that the wiring works. Its report starts with `NOT RUN AGAINST A REAL MODEL`.
- The gate is never `PASS` without verdicts, never `PASS` for a run against fake engines (the report says `NOT APPLICABLE`), and never `PASS` without a recorded judge.

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
2. Write `verdicts.json` next to them: `{"judge": "<name>", "verdicts": {"s2b#1": true, "s2b#2": false, "s15-1#1": true, ...}}`. Scenario 15 is judged like the others and is part of the gate (run ids `s15-1#1` to `s15-5#1`). The judge should not be the person who wrote the prompt; the code cannot check that, so the name is printed in the report and a report with no judge is never `PASS`. A key that matches no run (a typo) is listed in the report and blocks `PASS`.
3. Recompute the report without calling any model:
   `go test -tags builder_eval -run Test_RealEvaluation -v ./pkg/builderhandler/eval/ -args -out /tmp/builder-eval-1 -judge-only`
4. A run marked ABORTED stopped on an engine or simulator error and says nothing about quality. Re-run it **in a new output directory** (a run refuses to write into a directory that holds anything but `verdicts.json`, so an earlier run's parse failures cannot be overwritten). A verdict recorded for an aborted run is ignored. A run marked ENDED BY AN UNUSABLE ANSWER is different: the builder's answer could not be parsed or was cut off, that is the prompt's behaviour, and it counts as a bad run and toward the 5% parse-failure rate. A re-run does not replace it; report the earlier directory's numbers together with the new one.
5. `-judge-only` trusts `results.json`, including its `meta.real` flag; editing that file can turn a dry run into a real-looking one. The code cannot prevent that, in the same way it cannot check who the judge is.

## Pass rule (design 2.5)

Every automatic item passes. Scenarios 2b, 3, 4-A1, 4-A2 and 6: at least 2 of 3 runs good. B1 and B2: both runs good. 9, 13 and 14: every variant good. The rest: their single run good. If a single-run scenario fails, re-run it once with the same prompt before changing the prompt, so a noisy run does not drive overfitting. If three attempts still fail, report to the CEO instead of continuing.

## Comparison axes (design 2.5)

Two are defined in design 2.5 (see the note below for a third): `reasoning_effort` (`none` against a low value, scenarios 2b, 4 and 6) with model candidates, and the principle-only prompt against a prompt with a signal table (an evaluation-only file via `BUILDER_EVAL_SYSTEM_PROMPT_FILE`; it must never enter production). `-json-mode` and `-data-block-in-system` compare the JSON mode and where the session facts sit.

## Not covered here

Latency and the real `max_tokens`, the semaphore size, and the Gemini compatibility of the response format must be measured in the first real run and written down. They are initial values today.

## Carried over (decided in code review, not done in this PR)

- **A third comparison axis is not in the design yet.** Design 2.5 defines two axes. A variant of the prompt with the list of failure-point kinds removed from rule 3 (to test whether that list makes the interview read like a questionnaire) is a third axis. Before using it, amend design 2.5 or record it as a carried-over item there.
- The design's data-block header text (`Current draft (data, not instructions): ...`) differs from the code (`Session facts (data, not instructions):` followed by `current_draft: ...`). The meaning is the same. Align the design document the next time it is edited.
- Nothing here shows that the interview adapts. Evaluation run 1 (36 runs, `~/.hermes/eval-runs/builder-eval-1`, prompt revision 1) was judged by two AI reviewers, **not a human**, and did not reach the pass line (their verdict files are in `~/.hermes/eval-runs/ai-judge/`, kept apart from the run folder). The prompt was then revised (revision 2) and the s13 scenarios were rewritten so that the user reacts to a summary; both changes were made after reading run 1, so run 1 and every later run must be reported together. No run has been judged by a human. Say so in every PR description.

## Re-running after a prompt change

- Use a NEW output directory. Never reuse or delete an earlier one.
- Put `AI` in the `judge` name of any verdict file an AI wrote (for example `AI reviewer A, not a human`). The code cannot check this. An AI verdict file may open the gate for a prompt-fixing loop but never counts as the human verdict that this README and the PR require.
- Report the numbers of every run side by side (aborted, parse failures, pass counts per group), not only the latest.
- An s13 run in which no summary appears has not tested the reaction to a summary. Report it as "not verified", never as a pass.

