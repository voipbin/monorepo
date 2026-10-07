# VOIP-1573 Flow AI Builder: Issue Analysis (retroactive)

Written after implementation, to satisfy the issue-analysis checkpoint. Reviewers must verify every claim against the code, not against this document.

## 1. Request

CEO request: let a user create a Flow in square-admin by conversation, like the Assistant Builder (VOIP-1558). Standing constraints from the CEO:
- Flow action components change over time, so the builder must not hardcode per-action knowledge. Adding or changing an action must not require builder changes, and missing metadata must fail CI (fail-closed).
- Nothing is saved automatically; the user reviews the draft in the editor and saves it.

## 2. Validity and current code state (to be verified)

- No Flow builder exists today. Only the Assistant Builder: `bin-ai-manager/pkg/builderhandler` (stateless synchronous multi-turn), `POST /ai_builder/chat`, `GET /ai_builder/status`, front `square-admin/src/views/ais/builder`.
- Flow action types are defined in `bin-flow-manager/models/action` (43 types). Option structs exist per type, but there is no metadata saying which types are safe or useful to expose to a model, and resource references are not marked.
- An action catalog already exists on main: `bin-ai-manager/pkg/actioncatalog/main.go`, a hand-written entry per action type with drift guards (`TestActionCatalogMatchesTypeListAll`, `TestActionCatalogFieldsMatchOptionStructs`). It is consumed by the Assistant's `describe_action` tool (`pkg/aicallhandler/tool_describeaction.go`) and by the tool definitions enum (`pkg/toolhandler/definitions.go`). What is missing is exposure classification, flow-shape (FlowKind), resource-reference marking and required-field derivation; the descriptive text is reused and corrected, not replaced.
- The editor (`square-admin/src/views/actiongraph`, `views/flows/flows_create.js`) can take initial actions and positions.
- Not already solved elsewhere: no other ticket or code path generates Flow drafts from conversation.

## 3. Feasibility and risks

- Reuse: the stateless multi-turn structure, semaphore, circuit breaker, status endpoint and the front store can be reused. Flow handlers are written next to the Assistant ones (not by reusing Assistant types), sharing only the semaphore.
- Risks identified: customer resource UUIDs leaking through the draft (other customers' ids), the model producing actions that the executor treats differently than their names suggest, cost and side-effect actions (outbound send, webhook) chosen without user intent, SSRF through fetch, regression of the Assistant Builder when the front store is generalized.
- Mitigations chosen: classification-based exposure (Meta registry), ref tags that blank resource fields every turn, an address allowlist, deterministic validation in code, drift-lock tests, a shared front store factory with Assistant tests unchanged.

- Shared-asset impact: correcting the catalog text (for example condition_datetime is UTC, some options are not read by the executor) also changes what the Assistant's `describe_action` tool returns. The corrections state executor facts, so the effect is more accurate output; it is covered by the existing catalog tests plus the new statement-pinning tests.
- Two repositories: the front is in `monorepo-javascript` (separate PR). The front calls the new backend route, so the backend PR should merge and deploy first.
- Editor round-trip limit: the editor does not restore `next_id` for `goto`, so goto is excluded from v1 and the front round-trip test pins the remaining branch shapes.
- Prompt injection: user text reaches the model, but the model output is never executed; it is parsed, validated and stripped of resource ids and unknown option keys in code, then shown to the user for review before any save.

## 4. Decision

Proceed. Alternatives considered and rejected: a separate service (duplicates infrastructure), a hand-written per-action prompt (violates the change-tolerance requirement), automatic save (violates the review-before-save requirement).

## 5. Out of scope

Evaluation harness, multi-node loop detection, executor defects found on the way (mute and transcribe_stop have no handler, number-in-array substitution breaks condition_datetime.weekdays, fetch has no URL validation, message-manager nil source).
