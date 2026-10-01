# Builder evaluation harness

This is the adaptiveness evaluation of the Assistant Builder (design section 2.5). It exists because **a document or a unit test cannot show that the interview adapts to the user's answers. Only reading real conversations can.**

## What it does and does not prove

- It drives the production turn code (`builderhandler.RunTurn`), not a copy, against scripted scenarios and a separate **user simulator** (a different model, high temperature, a persona sheet, some deliberately uncooperative).
- Mechanical checks (deterministic, in `gate.go`): JSON parse-failure rate at most 5%, scenario 9 returns a draft in the first response, no `init_prompt` names a forbidden tool.
- Everything else is read by a **judge** (ideally a human) reading the transcripts against the rubric printed at the top of each transcript. The harness never judges interview quality.
- A run with fake engines (the unit tests) proves only that the wiring works. Its report starts with `NOT RUN AGAINST A REAL MODEL`.
- The gate is never `PASS` without verdicts, never `PASS` for a run against fake engines (the report says `NOT APPLICABLE`), and never `PASS` without a recorded judge. A judge name containing `AI` makes the report say `AI JUDGED. This is not a human verdict`; an AI verdict does not count as the human verdict this README and the PR require (see "Re-running after a prompt change").

## Cost and size

36 simulated runs (3 repeats of 2b, 3, 4-A1, 4-A2, 6; 2 each of B1 and B2; 3 for scenario 9; 3 for 13; 4 for 14; one each of 1, 2a, 7, 8, 10, 11, 12), plus 5 synthetic checkpoint cases. At an average of 6 turns that is **about 216 builder calls and a similar number of simulator calls**. The 6-turn average is an assumption, not a measurement: evaluation run 1 measured 138 builder calls (about 3.8 per run) and run 2 measured 166 (about 4.6 per run), both with 0 parse failures. Judging: about 21 full transcripts and about 8 skims, plus the 5 synthetic cases.

## Runs side by side (AI judged; no run has been judged by a human)

Both runs used 36 simulated runs plus 5 synthetic cases, a builder `gemini-3.8-flash` and a simulator `gemini-3.7-flash`. Every verdict below was written by an AI reviewer, **not a human**. Counts are verdicts marked true out of 41 per judge (`~/.hermes/eval-runs/ai-judge/`).

| | Run 1 | Run 2 |
|---|---|---|
| Output folder | `builder-eval-1` | `builder-eval-2` |
| Prompt | revision 1 (commit `776efb063`) | revision 2 (commit `01f794f37`; `prompt.go` has not changed since, so this is the shipped prompt) |
| Builder calls (average per run) | 138 (3.8) | 166 (4.6) |
| Parse failures | 0 | 0 |
| Aborted | 0 | 0 |
| AI judge A, true of 41 | 32 | 36 |
| AI judge B, true of 41 | 30 | 34 |
| Judges agree | 33 of 41 | 35 of 41 |
| Pass line, computed by `Evaluate` over each AI judge's verdicts | not reached by either judge (A: s4-A1, s5-B1, s5-B2, s7, s8 below the line; B: s2b, s3, s4-A1, s5-B1, s5-B2, s7) | not reached by either judge (A: s1, s2a, s13, s15 below the line; B: s2a, s2b, s13, s15). No human verdict exists, so the gate is not decided as a human gate |

Read this with two cautions. Revision 2 was written after reading run 1's transcripts and verdicts, and the s13 scenarios were rewritten at the same time, so the improvement is partly fitted to the same scenarios; a third run on new scenarios has not been made. And the two judges disagree on 6 to 8 items per run, so the totals are not exact.

What run 2 still got wrong. The verdict files hold only good or bad per run and no reasons, so these items were read from the run 2 transcripts (`~/.hermes/eval-runs/builder-eval-2/`) for the runs the AI judges marked bad:
- s2b (the restaurant booking scenario) stayed below the line for judge B (s2b#1 and s2b#3 marked bad): one question and then a draft, no question about a failure point specific to the business, and in s2b#3 a behaviour the user never mentioned (hand over to a staff member on request) written into the draft.
- The retry policy the user called essential in s4-A1 was still not asked about.
- Values the user never said still reach the draft ("arrive 5 to 10 minutes early" in s2a, a date of birth in s4-A1, an entrance password in s6-delivery).
- s13-c (an ambiguous approval) and s15-1 (the checkpoint sentence without a choice to keep refining) failed for both judges.
- Two defects visible in drafts: the prompt's own rule name leaked into a draft assumption, and `send_email` was mentioned in a draft body but missing from its tool list.

None of this was fixed in this PR, on purpose: fixing it by reading these transcripts would fit the prompt to them more tightly.

## Where the run folders are

`~/.hermes/eval-runs/builder-eval-1/report.md` (run 1) was written before any verdict existed and still says `Judge: NOT RECORDED` and `NOT DECIDED`. The two AI verdict files are in `~/.hermes/eval-runs/ai-judge/`, kept apart on purpose. Do not read the run folder's report as a result; read the counts above and the verdict files.

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

Latency, the real `max_tokens` and the semaphore size were not measured in run 1 or run 2; measure them in the next run and write them down. They are initial values today. (Gemini accepted the response format in both runs: 138 and 166 calls, 0 parse failures. That is two runs, not a guarantee.)

## Carried over (decided in code review, not done in this PR)

- When the evaluation variants (the signal-table prompt, the JSON-mode and data-block switches, the failure-point-list variant) are deleted: after the first real comparison run, delete every variant that was not used, in the same PR as that run's report.
- `Config.LLMTimeout` of zero or less means no deadline in `RunTurn`. The server wiring must reject such a value when it reads the setting, because the 40 second limit protects the circuit breaker (design 4.3).
- Known interaction in the prompt, to be watched in the next run: a fork on an item the user called essential that is closed by the three-turn limit (rule 3) must still go into the draft as a requirement and into assumptions (rule 7). Check it with s4-A1 and s7. Rule 3 is also long and may be split into items in the next revision.

- Not measured yet: latency, the real `max_tokens`, the semaphore size, and the Gemini response-format compatibility beyond the 138 calls of run 1 and the 166 of run 2, with 0 parse failures. Before activation the design's section 7 items must also be settled (load balancer and ingress timeouts, the Loki log source, a per-minute rate limit).

- **A third comparison axis is not in the design yet.** Design 2.5 defines two axes. A variant of the prompt with the list of failure-point kinds removed from rule 3 (to test whether that list makes the interview read like a questionnaire) is a third axis. Before using it, amend design 2.5 or record it as a carried-over item there.
- The design's data-block header text (`Current draft (data, not instructions): ...`) differs from the code (`Session facts (data, not instructions):` followed by `current_draft: ...`). The meaning is the same. Align the design document the next time it is edited.
- Nothing here shows that the interview adapts. Evaluation run 1 (36 runs, `~/.hermes/eval-runs/builder-eval-1`, prompt revision 1) was judged by two AI reviewers, **not a human**, and did not reach the pass line (their verdict files are in `~/.hermes/eval-runs/ai-judge/`, kept apart from the run folder). The prompt was then revised (revision 2) and the s13 scenarios were rewritten so that the user reacts to a summary; both changes were made after reading run 1, so run 1 and every later run must be reported together. No run has been judged by a human. Say so in every PR description.

## Re-running after a prompt change

- Use a NEW output directory. Never reuse or delete an earlier one.
- Put `AI` in the `judge` name of any verdict file an AI wrote (for example `AI reviewer A, not a human`). The code cannot check this. An AI verdict file may open the gate for a prompt-fixing loop but never counts as the human verdict that this README and the PR require.
- Report the numbers of every run side by side (aborted, parse failures, pass counts per group), not only the latest.
- An s13 run in which no summary appears has not tested the reaction to a summary. Report it as "not verified", never as a pass.

