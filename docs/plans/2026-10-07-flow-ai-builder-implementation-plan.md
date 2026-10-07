# VOIP-1573 Flow AI Builder: Implementation Plan (retroactive)

Written after implementation from the design (`2026-10-07-flow-ai-builder-design.md`). Reviewers must check that the plan is complete, ordered, testable and consistent with the design and with the actual code; the diff is `git diff origin/main...HEAD` in this worktree.

## Phases

1. bin-flow-manager `models/action`: `meta.go` (Meta{Exposure, FlowKind}), `ref.go` (ref tags action/resource/address), tags on every option field, drift-lock test (every action type has meta, every option field has a valid ref decision), golden test for exposure.
2. bin-ai-manager pure logic: `pkg/actioncatalog` (catalog text and RequiredFields derived from metadata), `models/flowbuilder` (request/response and limits), builderhandler `flow_transcode`, `flow_option`, `flow_layout`, `flow_validate` (deterministic validation, reachability, warnings).
3. bin-ai-manager handlers: `flow_schema`, `flow_prompt`, `flow_turn`, `flow_main`, `flow_chat`, `flow_parse`, `flow_reconstruct`; cachehandler Flow daily counter; listenhandler `/v1/flow_builder/chat`; metrics namespace `flow_builder_*`.
4. Plumbing: bin-common-handler RPC client and `address.IsExternalEndpoint`; bin-openapi-manager `POST /flow_builder/chat` and regenerated server; bin-api-manager servicehandler with shared `mapBuilderRPCError`; docs.
5. Front (monorepo-javascript): `createBuilderStore` factory (Assistant behavior and tests unchanged), `views/flows/builder/*`, entry card and apply path in `flows_create.js`, logout cache clearing.
6. Verification: unit tests per phase, mutation checks of defenses, lint (golangci-lint, eslint), convention gates, front jest and build, real-browser check with mock API, scenario replay of eight realistic flows.

## Ordering and dependencies

Phase 1 before 2 (catalog derives from meta). 2 before 3. 4 depends on 3 for the RPC shape. 5 depends on the OpenAPI wire form from 4. Phase 6 runs after each phase and at the end.

Wiring steps that are easy to miss: register the route in `isBuilderRoute` and the construction in `cmd/ai-manager/main.go` (security-relevant: the input must never be logged, pinned by `Test_processFlowBuilder_neverLogsTheInput`); front round-trip test `flowDraftRoundtrip.test.js` for branch shapes; `ConfirmDialog` for replacing editor content.

Design decisions carried into the plan: no branching on action type names in builder code (everything derives from `Meta`); fail-closed drift locks; classification-based exposure with `internal` types (fetch, mute, transcribe_stop, call, goto, email_send) excluded; `sensitive_nodes` set by code; 13 warning keys shared by back and front; separate daily counter (OQ7); queue-level circuit breaker coupling accepted with timeout metrics (OQ10).

Planned versus done: the evaluation harness (design section 9) is deferred to a follow-up and is listed as such in the PR. Mutation checks, real-browser checks and the eight-scenario replay were performed by the review rounds with throwaway tests under /tmp and are not committed.

## Test strategy

TDD per unit; contract tests pin catalog statements to executor facts; adversarial random input against the parser; front store transitions with fake timers.

## Rollout and risk

Behind the existing builder availability status (`GET /ai_builder/status`): when the builder key is not configured, `flow_chat.go` returns Unavailable and the front hides the entry card. The switch is shared with the Assistant Builder, there is no Flow-only flag (by design; a new on/off flag was not requested). No schema migration. This PR also changes shared Assistant-side code (the `NewBuilderHandlers` constructor in `cmd/ai-manager/main.go`, `isBuilderRoute` and panic handling in `listenhandler/v1_ai_builder.go`, the shared `mapBuilderRPCError`, the front `createBuilderStore` factory and `BuilderPanel`, and catalog text), so the safe rollback is reverting the whole PR pair, not only the route and the entry card. Metrics added for timeouts and circuit opens to measure the queue-level circuit breaker coupling.
