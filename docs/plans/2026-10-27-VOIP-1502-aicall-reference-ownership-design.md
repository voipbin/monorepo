# VOIP-1502: aicall reference_type allowlist and reference ownership

## 1. Goal

`POST /aicalls` and `POST /service_agents/aicalls` take `reference_type` and
`reference_id`. They validate the type against an allowlist and check that the
referenced resource belongs to the customer before anything is created, and
ai-manager looks up an existing aicall by customer and reference id.

## 2. Behavior after this change

bin-api-manager, both create paths, one shared servicehandler helper
`aicallReferenceValidate(ctx, customerID, referenceType, referenceID)`:

| input | result |
|---|---|
| `reference_type` not in {"", call, conversation, contact_case} (including `task`, other case, padded) | 400 `INVALID_ARGUMENT` |
| `reference_type` = "" (none) | `reference_id` is forced to the nil UUID |
| call / conversation / contact_case with nil `reference_id` (a malformed id on the admin path parses to nil) | 400 `INVALID_ARGUMENT` |
| referenced resource missing, deleted, or owned by another customer | 404 `RESOURCE_NOT_FOUND`, identical for all three types |
| backend error that is not a not-found (including typed errors of other statuses) | wrapped and propagated with its original classification |

Not-found normalization: the backend not-found forms (typed `VoipbinError` with
`StatusNotFound`, `requesthandler.ErrNotFound`) are mapped to a bare
`serviceerrors.ErrNotFound` that does not keep the backend error in its chain,
because the error translator passes typed errors through unchanged and a
backend reason code would otherwise differ between "missing" and "foreign".
Ownership mismatch and a non-nil `TMDelete` (call, conversation) map to the same
bare sentinel. The helper calls `CallV1CallGet`, `ConversationV1ConversationGet`
and `ContactV1CaseGet` directly (not `callGet`, which returns a state error for
a deleted call). contact_case has no deletion concept and a closed case is accepted (out of scope). Rejections log customer id, type and reference id at info.

Reference customer: the assistance owner's customer on the admin and accesskey
path, `a.CustomerID` on the service agent path.

Order (reference lookups only; the existing assistance lookup is unchanged): permission check, then (admin path) the direct-token forcing, then the
helper, then `FlowV1ActiveflowCreate`. A refusal therefore never creates an
activeflow and a caller without permission gets 403 before any reference
lookup. On the service agent path the helper replaces the old contact_case
block and runs right after `hasPermission`, before insight AI resolution. The
direct path keeps forcing none/nil unconditionally (an unknown type from a
direct token is overwritten, not rejected, as before).

`task` is internal and is not in the allowlist. The generated `Valid()`
accepts it, so it is not used. The OpenAPI enum is unchanged (the type is also
a response schema); only descriptions are extended.

The old service agent contact_case branch (mismatch 403) goes away; a case of
another customer is already a not-found from contact-manager, so the response
is now 404 in both cases. A grep of the repository finds no consumer branching on that 403.

bin-ai-manager:

- New `AIcallGetByCustomerIDAndReferenceID(ctx, customerID, referenceID)` on
  dbhandler and `GetByCustomerIDAndReferenceID` on aicallhandler. Direct
  database query on `customer_id` and `reference_id`, newest first, same row selection as the existing query (no `reference_type` or
  `tm_delete` filter), no cache
  (the existing cache key has no customer), no new index (existing
  customer_id and reference_id indexes). `AIcallGetByReferenceID`, its cache
  key, event.go (its ids are call ids) and ai-control are unchanged. The
  conversation reuse path loses its cache hit and does one indexed query.
- `startReferenceTypeConversation` and the duplicate-key branch of
  `startReferenceTypeContactCase` use the new lookup, then check
  `existing.CustomerID == a.CustomerID` as a defensive invariant (unreachable
  while the lookup is scoped; it protects against a future unscoped swap).
  conversation: a mismatching row is not reused and not touched, a fresh
  aicall is created (the local `res` is cleared so the idle-expiry termination
  branch never touches the row). contact_case: the mismatching row is refused with an
  error returned immediately (no `continue`, no retry). For a team the starting
  member AI is assumed to belong to the team's customer.

## 3. Alternatives

Scoping `AIcallGetByReferenceID` itself touches the cache key, event.go and the
CLI: out of scope. Rejecting (instead of ignoring) a non-nil `reference_id`
with type none: the direct path already forces it silently, same here.

## 4. Out of scope

`IsErrDuplicate` string matching; flow-manager ai_talk assistance check (it
calls Start with a trusted reference and is unaffected); cleaning existing
production rows.

## 5. Verification plan

api-manager servicehandler tests, both paths, types call / conversation /
contact_case: success, foreign owner, missing (typed 404 and bare 404), deleted
(call, conversation), type none with non-nil id, nil id with a typed
reference, malformed admin id, unknown types (`task`, `Call`, padded), non
not-found backend error, permission denied before any lookup, admin pointing at
an assistance of another customer (reference customer is the assistance owner),
accesskey identity; the error codes of missing / foreign / deleted are equal.
Gomock enforces that no activeflow is created on any refusal. Existing tests
changed: `Test_AIcallCreate` rows with call references (add `CallV1CallGet`),
`Test_ServiceAgentAIcallCreate` (mismatch row becomes not-found, harness
rewritten), tenant isolation test; `Test_AIcallCreate_direct_bindsAndForcesReference`
unchanged plus one unknown-type row. Direct tokens are forced to none before the helper, so no reference lookup
happens for them (the existing direct test expects none). Also: a typed non-not-found error is propagated, type
none with the assistance id omitted keeps the existing 400. Server handler tests: one row each for 400
and 404 mapping.

ai-manager: dbhandler test with two customers sharing a reference id (filter,
newest first, not found, rows error table), aicallhandler wrapper test, start.go
tests moved to the new lookup (conversation and contact_case reuse paths) plus
mismatch cases (conversation: create called, no status update, no stop
listening; contact_case: error, create called once). Mocks regenerated.

Existing ai-manager tests moved to the new lookup, by file: start_test.go
(about 21 expectations), start_duplicate_test.go, service_test.go; unchanged:
event_test.go, logpolicy_test.go, listen_test.go.

Docs: OpenAPI request descriptions (allowed values, 400/404 on both paths;
enum unchanged) regenerated with go generate in bin-openapi-manager then
bin-api-manager; aicall RST: allowed values and the sentence that promised 403
for a foreign case replaced by 400/404; clean rebuild and `git add -f` of
build/.
