# VOIP-1578 Flow Builder fails on follow-up question turns

This document holds the issue analysis, the design and the implementation plan of one small change. Reviewers must verify every claim against the code (`git show origin/main:<path>`). The measurements come from a throwaway probe whose raw output is quoted below; the probe file itself is not part of the change.

## 1. Issue analysis

### Observed

- The CEO used the Flow AI Builder in square-admin (production, after VOIP-1577). Turn 1 ("a typical parcel delivery company") got a question. Turn 2 ("I want delivery tracking") ended with "No usable response. Trying again may use another daily attempt." and a Try again button, which is the `BUILDER_RESPONSE_INVALID` message.
- Production diagnostic lines (VOIP-1576) of the last 90 minutes for that conversation: turn 1 `outcome=ok`; every turn with `user_turns` 2, 3 and 4 (5 calls in total) `outcome=invalid_response`, `invalid_kind=unparsable`, `finish_reason=stop`, 2.3 to 3.6 s, 82 to 110 completion tokens, 144 to 193 response chars, `json_mode=object`. The model answered quickly and completely, and the answer was not accepted by the parser.

### Reproduction (local, developer key, `gemini-3.8-flash`, reasoning none, 27 to 30 allowed types, the production `flowMessages` and system prompt; the conversation of the screenshot)

| response_format | history sent as today (assistant turn as plain text) | assistant turn wrapped as `{"message": "..."}` |
|---|---|---|
| `json_object` | 4 of 4 plain Korean sentences, `FlowParse` fails ("no usable response object") | 8 of 8 `{"message": ...}`, `FlowParse` accepts |
| none | 4 of 4 plain sentences, fails | not run |
| `json_schema` (before VOIP-1577) | 4 of 4 JSON, accepted | not run |

So with `json_object` the provider does not force the answer shape and, on a later turn, the model copies the form of the previous assistant message in the history, which is plain text.

### Code facts

- `flowMessages` (`flow_turn.go`) sends each history message as `{Role, Content: m.Content}`; the assistant entries are the visible `message` text that the client stores and sends back (`flowbuilder.ChatRequest.Messages`).
- `FlowParse` accepts a JSON object that has a non-empty `message`; text without such an object returns `ErrInvalidResponse` (`flow_parse.go`).
- Before VOIP-1577 the schema forced the JSON form on every turn, which hid the problem. The change of VOIP-1577 (`FlowConfig` fixes `json_object`) exposed it. Its checks covered a first turn, the fixed phrase and a change request; they did not cover a second question turn. This is a regression of that change.
- The Assistant Builder builds its own messages (`turn.go`) and is not affected.

### Conclusion and not known

- Cause: plain-text assistant turns in the history make the model answer in plain text when no schema is enforced. Wrapping them in the JSON form that the prompt demands fixes the reproduction (8 of 8).
- Not measured: conversations with more than 4 turns, a history that contains an assistant turn which came with a draft (only its message text is in the history), other models, and the production key. The post-deploy check repeats the screenshot scenario against production.
- Proceed: yes. Every conversation with a follow-up question fails today, and the fix is small and reversible. A revert of VOIP-1577 is not better: it brings back the missing draft and the runaway generations.

## 2. Design

- `flowMessages` sends each assistant history entry as the JSON object that the prompt asks the model to produce: `{"message": "<the text>"}`, encoded with HTML escaping off (so `&`, `<` stay as written) and non-ASCII text left as is. User entries and the session facts block are unchanged.
- This is a server-side change. The client keeps sending the visible text, so the API, the OpenAPI spec and the frontend do not change.
- An assistant entry that is already a JSON object with a `message` is not specially handled: the client only stores the visible message text, never the raw model answer.
- No change to the parser, the prompt, the response format, error mapping, metrics or the Assistant Builder. The parser is not made lenient on purpose: accepting free text as a message would hide a real format failure; if invalid answers continue after this change, that is the next step to design.
- Risk: the model now sees its own earlier turns in the JSON form, which is what the prompt describes, so no new behaviour is asked of it. Messages that contain quotes or newlines are escaped by the encoder.
- Rollback: revert the commit and redeploy ai-manager.

## 3. Implementation plan

1. `bin-ai-manager/pkg/builderhandler/flow_turn.go`: in `flowMessages`, encode assistant history entries as `{"message": content}` (a small helper, HTML escaping off) and keep user entries as they are. Update the doc comment of `flowMessages`.
2. Tests in `flow_diag_test.go` (or a new `flow_messages_test.go`), reached through the real `Chat` path with the recording sender so the request that is sent is what is checked:
   - an assistant turn with Korean text, a double quote, a backslash, a newline, `&` and `<` is sent as a valid JSON object whose `message` equals the original text (decoded and compared as values), user turns are sent as typed, and the last user turn keeps the session facts block;
   - a history without assistant turns is unchanged;
   - the first message of the request is still the system prompt.
3. `bin-ai-manager/docs/operations.md`: one sentence in the Flow Builder section that history assistant turns are sent as JSON and why (VOIP-1578).
4. Gates: `cd bin-ai-manager && go test ./... && golangci-lint run`; from the repository root, after committing and fetching `origin/main`, `bash scripts/check-test-conventions.sh`.
5. Verification after deploy (read-only plus the minimum LLM calls, a temporary admin that is deleted afterwards, no calls, SMS, email or number purchases): repeat the screenshot conversation (turn 1, a question, turn 2, a question, then the fixed phrase) and the three product scenarios of VOIP-1577. Expected: no `invalid_response`, every turn `outcome=ok`, a draft on the fixed phrase and on a change request.
6. Only `bin-ai-manager/` changes, so CI builds and deploys only ai-manager; the release approval after merge belongs to the CEO.
