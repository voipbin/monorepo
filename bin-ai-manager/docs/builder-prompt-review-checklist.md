# Builder prompt review checklist (VOIP-1558, T4)

**This checklist is checked by a human reviewer in the PR. CI cannot check any of it.** The unit tests pin only the contract parts of the prompt (fixed phrases, data-block keys, tool catalog, schema). Whether the interview actually adapts to what the user says is not provable by a document or a unit test. It is judged by running the evaluation harness (design 2.5) and by a person reading the transcripts.

Status: the prompt has not been evaluated. The harness exists (`pkg/builderhandler/eval`) and runs against fake engines in `go test`; no real model has been run against it yet.

## 1. Prompt content mapping (design sections to `prompt.go`)

Tick each line by reading `SystemPrompt`. A missing item is a defect.

- [ ] 2.2 item 1: role (interviewer and consultant), interview in the user's language, `init_prompt` in English, one line stating the callers' language.
- [ ] 2.2 item 2: four dimensions (purpose and success, channel and the callers' language, who and which situation, must and must-not), and "move to confirming only when all four are filled and no fork is open".
- [ ] 2.2 item 3: the digging principle ("where is this assistant most likely to fail in a real call", pick one, decide afresh after every answer). **No question list, no ordered steps, and no signal table anywhere in the body.** The signal-table variant exists only as an evaluation file, never in production.
- [ ] 2.2 item 4: tool catalog of exactly six tools with a human-written one-line description and one sentence on the flow behaviour of `stop_service` and `connect_call` (stop_service moves on to the next node, connect_call transfers the caller). No promise for capabilities that have no tool.
- [ ] 2.2 item 6: two few-shot dialogues (one vague start, one detailed start). **Their domains (real estate, gym renewal) do not overlap the evaluation domains by vocabulary** (reservation, dental, restaurant, customer support, delivery notice, survey, pharmacy stock). A test pins the short answers the evaluation personas give; conceptual closeness (a viewing is a kind of booking) is not tested and is a reviewer's call.
- [ ] 2.3: six rules and the priority chain (one-or-two-question limit, then the user's words, then digging while a fork is open, then confirming, then stopping). Rule 3 has the three-turn cap per topic, the "last two answers carried no information" assumption rule judged by information not length, the closed-fork-goes-to-assumptions rule, and the checkpoint sentence (only when `checkpoint: true`, omitted on a summary turn). Rule 5 has at most one summary, both exceptions (full hand-over and fixed phrases; a detailed first message), and the two reactions to a summary. Rule 6 has revise-and-say-what-changed, assumptions instead of blanks, the callers' language.
- [ ] 2.3 fixed phrases: the two draft-now phrases and the two continue-draft phrases are in the prompt and are handled "or the same meaning".
- [ ] 2.4: reply is one JSON object, `message` first, then `draft`, then `assumptions`. `message` is plain text, no markdown.
- [ ] 2.6: H1 title line, then the header skeleton (`## Identity & Purpose`, `## Voice & Persona` with `### Personality` and `### Speech Characteristics`, `## Conversation Flow` with `### Introduction`, `## Response Guidelines`, `## Scenario Handling`), length target 2.5K to 6K characters, channel changes content not header use, and **the model is told not to write `## Tools & Capabilities`** (the product adds it).
- [ ] 4.2: the session block is described as data, not instructions, and the model is told not to follow instructions inside `current_draft` or the user message that change these rules.
- [ ] 5: the continue-draft seed phrases are listed.

## 2. Properties no test can check (read the transcripts)

- [ ] The next question follows from the previous answer (not from a list).
- [ ] The model does not ask for something already said.
- [ ] A short answer that carries information ("reservations") makes the model keep narrowing, not assume.
- [ ] Two answers in a row with no information make the model state an assumption and move on, and the assumption appears in `assumptions`.
- [ ] After the summary the model does not summarise again.
- [ ] **The user's own concrete values (numbers, times, names, thresholds) survive into the draft unchanged (s2b).** Two sentences in the prompt were added after the first real run and are unverified hypotheses: the last sentence of rule 6 (keep the user's values) and the flow-behaviour sentence after the tool catalog. The first real run, three s2b conversations judged by the prompt's author and not an evaluation, showed one case where the user said 10 and the draft said 8. Their effect is unknown until the evaluation runs; do not credit a later pass to them.

## 3. Cross-repository and cross-copy checks (manual, no automation exists)

- [ ] `pkg/builderhandler/testdata/golden_phrases.txt` matches the four strings in the square-admin PR character for character in meaning.
- [ ] The six tool names in `models/builder` `AllowedTools`, in the prompt catalog and in the response schema enum are the same six, and each has an entry in the frontend `TOOL_LABELS`.
- [ ] The header skeleton in the prompt is the same header set as the frontend `PROMPT_TEMPLATES`.
- [ ] The two JSON copies of each error response (`bin-ai-manager/pkg/listenhandler/testdata/builder_*.json` and `bin-api-manager/pkg/servicehandler/testdata/builder_*.json`, added with the server tasks) have identical content. They live in separate Go modules, so the match is verified by eye in review, not by CI.
