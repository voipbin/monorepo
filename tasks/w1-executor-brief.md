# W1 Executor Brief (VOIP-1419) — read fully before touching anything

You are one of several parallel executors. You work ONLY inside your assigned service
directory under the worktree root
`/home/pchero/gitvoipbin/monorepo/.worktrees/VOIP-1419-Enforce-explicit-event-subscription-id`.

Normative sources (read both BEFORE coding):
- `docs/plans/2026-08-28-voip-1419-explicit-subscription-id-design.md` (esp. §1 invariance
  rule, §3 method template + special cases, §4 tests)
- `tasks/todo.md` — "Implementation Plan (stage 3)" W1 bullet (all carve-outs live there)

## Hard rules

1. NEVER run any `git add`/`commit`/`checkout`/`stash` — the orchestrator owns the index.
2. Touch files ONLY under your assigned `bin-<service>` directory.
3. Use `/usr/bin/grep` (a shell hook rewrites bare `grep`); exclude `vendor/`.
4. Do NOT modify `bin-common-handler` in any way. The old `PublishEvent(data interface{})`
   signature still stands in W1 — your additions are purely additive and must compile
   against it.
5. ROUTING-KEY VALUES MUST NOT CHANGE. Golden `expect` strings are read-only. If any test
   forces you to touch an expect string, STOP and report — do not "fix" it.

## What you do, per assigned type

1. **Method**: add `EventSubscriptionID() string` per the design template — pointer
   receiver, returns the same value the JSON fallback yields today (normally
   `h.ID.String()`; special cases per design §3 table). Placement: next to the type
   declaration, EXCEPT types declared in `models/<entity>/webhook.go` → put the method in
   a sibling file (e.g. `subscription.go`) with a short comment explaining why (see
   `bin-ai-manager/models/message/subscription.go` for the precedent). Method comment: the
   short standard form from design §3 (2-3 lines, no essay).
2. **Compile assertion**: `var _ eventtopic.SubscriptionIdentifier = (*T)(nil)` in the
   package's sibling `_test.go` (create one if absent; import
   `monorepo/bin-common-handler/models/eventtopic`). Exactly ONE per type, in the sibling
   test file — never in the golden file.
3. **Behavioral test**: in the type's OWN package, mutation-checked (set distinct UUIDs on
   the id field and any plausible wrong-answer field; assert the method returns the right
   one and not the wrong one — copy the shape of
   `bin-ai-manager/models/message/main_test.go:60` /
   customer golden `TestAccesskeyUsesOwnIDNotCustomerID`). Wrapper/nil-embed types
   additionally assert the nil-embed branch returns `""` without panicking.
4. **Golden file** (`models/*/routingkey_golden_test.go` in your service):
   - Simplify `resolveSubscriptionID`: DROP the JSON-unmarshal half; KEEP the `data any`
     parameter and the typed-nil reflect guard; non-implementing data now returns `""`.
     Rewrite the helper's doc comment for the new mechanism (no more "top-level id"
     description).
   - Negative assertions ("must not implement SubscriptionIdentifier") on types that now
     GAIN methods: DELETE the runtime negative test entirely (the sibling `var _`
     assertion replaces it). Do NOT convert it in place.
   - Refresh stale prose comments that describe the deleted JSON fallback (headers, var
     comments) — the mechanism is now "explicit method, mandatory; empty return →
     placeholder".
   - Expected key strings: untouched, byte-for-byte.
5. **Verification** (mandatory, from your service dir):
   `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`
   If a PRE-EXISTING failure unrelated to your change surfaces (mockgen drift, old lint
   errors): do NOT fix it and do NOT revert your work — report it verbatim and let the
   orchestrator decide.

## Report format (return this, nothing else)

- Service, types done, files created/modified (paths)
- `git status --short -- bin-<service>` output
- Verification: each of the 5 steps → pass/fail, with failing output verbatim if any
- Any deviation from the brief, with reason
