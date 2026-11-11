# VOIP-1715: Sync Conversation owner on Case Assign and Unassign

## 1. Issue review (ticket claims were assumed wrong and re-checked on origin/main 7a741becb)

Verdict: valid.

| Claim | Result | Evidence |
|---|---|---|
| Case Assign only writes Case.Owner | Confirmed | `bin-contact-manager/pkg/casehandler/assign.go` `Assign`: `CaseGetByID`, tenant check, `CaseUpdateOwner`, `CaseGetByID`. No conversation call. Callers: `listenhandler/v1_cases.go` `processV1CasesIDAssignPost` only. |
| Unassign goes through the same function | Confirmed | `requesthandler.ContactV1CaseUnassign` posts to `/v1/cases/{id}/assign` with `OwnerType=""` and `OwnerID=uuid.Nil` (`bin-common-handler/pkg/requesthandler/contact_cases.go`). |
| Flow dispatch is decided by Conversation owner only | Confirmed | `bin-conversation-manager/pkg/conversationhandler/execute_mode.go` `getExecuteMode`; used by `event_webchat.go`, `message.go`, `hook.go` (two sites). Agent owner returns `ExecuteModeAgent` and `runExecuteModeAgent` triggers no flow. |
| A Conversation owner update path exists | Confirmed | `conversationhandler/db.go` `Update`: for `owner_id != Nil` it validates the agent exists and belongs to the conversation's customer, then derives `owner_type=agent`; for `Nil` it derives `owner_type=""`. RPC: `requesthandler.ConversationV1ConversationUpdate`. |
| Conversation to Case link is `Metadata.ContactCaseID` | Partly wrong as a lookup key | It is written only by `bin-api-manager/pkg/servicehandler/case_message.go` (agent sends a message from a Case). The `case_create` flow action and AI tool do not set it (`conversation/metadata.go` comment). Conversation list has no metadata filter (`Field*` list in `models/conversation/conversation.go`). It cannot be used to find conversations from a Case. |
| Case to Conversation link is `ReferenceType`/`ReferenceID` | Confirmed with a gap | `case_create` (flow `actionhandle.go` actionHandleCaseCreate, AI `tool.go` toolHandleCaseCreate) stores `ReferenceType="conversation_message"` and `ReferenceID=<conversation id>` for conversation-origin Cases. Gap: `Continue` (`lifecycle.go`) calls `insertWithRetry(..., source.ReferenceType, &source.ID, now, "")`, so a continued Case has the same ReferenceType, an empty ReferenceID and `PreviousCaseID` set. |

Why the AI keeps answering: each inbound message creates a new activeflow only when `getExecuteMode` returns Flow; ai-manager then reuses the conversation's aicall (`startReferenceTypeConversation`). The Conversation owner stays empty after Case Assign, so the Flow path is taken every time.

Not reproduced against a running system. The code path is deterministic and pinned by the unit tests in section 4.

Facts the ticket could not state, resolved here:
- Case to Conversation: the Case stores one conversation id (ReferenceID), reached directly or through the PreviousCaseID chain. The open-Case uniqueness key is `customer_id|peer_type|peer_target|reference_type` (`migrations/000001_baseline.up.sql`, `open_peer_uk`), so the same peer can reach the same Case from a second Conversation with a different self (second number or account). `case_create` on that second Conversation ends with AlreadyExists (silently, `actionhandle.go` errCreate path), so the second Conversation is not linked by ReferenceID. This is a known limit (see section 3): only the Conversation recorded in the Case chain is synchronized.
- Call-origin Cases (`ReferenceType="call"`) have no Conversation: no-op.
- Conversations linked to a call-origin Case only through `Metadata.ContactCaseID` (agent SMS from a Case) cannot be found from the Case side (no metadata filter) and are out of scope.

Blast radius: Case Assign is called by `ServiceAgentCaseAssign`, `CaseAssign`, and Unassign by `ServiceAgentCaseUnassign`, `CaseUnassign` in api-manager. All reach the single contact-manager function.

## 2. Design

### 2.1 Where

In `bin-contact-manager/pkg/casehandler` (`assign.go`, `assign_conversation.go`, `lifecycle.go`), not in api-manager.

- Assign and Unassign (admin and service-agent routes, four api-manager call sites) all reach `casehandler.Assign`; Close reaches `casehandler.Close`. api-manager would need four duplicated edits and extra Case reads.
- contact-manager owns the Case (ReferenceID, PreviousCaseID) and already holds `reqHandler` (used by `CaseTagAdd`).
- No new dependency edge and no cycle: `requesthandler` already has `ConversationV1ConversationGet` and `ConversationV1ConversationUpdate`; contact-manager already `replace`s `bin-conversation-manager`; conversation-manager is reached over RPC only and does not import contact-manager (no Go import of `bin-contact-manager` there). The only go.mod change moves `bin-conversation-manager` from indirect to direct (the conversation model is imported).
- Authorization stays in api-manager. conversation-manager `Update` validates the agent and the customer. contact-manager adds a Case/Conversation customer equality check.
- No OpenAPI, RST or response contract change: the Case response is identical; only a side effect is added. No config flag.
- Rejected: making `getExecuteMode` consult the Case (couples conversation-manager to contact-manager on the inbound hot path); carrying ReferenceID in `Continue` (changes `reference_id` filter results); a conversation list lookup by peer (`Continue` blanks Local, unverified JSON-column match semantics, would also silence other channels of the same peer).

### 2.2 Lookup (a)(b)(c)

Starting from the Case: if `ReferenceType==conversation_message` and `ReferenceID` is a non-nil UUID, that is the conversation. Otherwise follow `PreviousCaseID` (customer-checked, at most 10 hops). No conversation means no-op (call-origin, no reference, a chain ending at a missing or other-customer Case, over-long chain). A database error while walking the chain is returned as an error (never skipped silently). `Metadata.ContactCaseID` is not used (not always set, not filterable).

### 2.3 Behavior and ordering

| Operation | Conversation owner change | Order and failure |
|---|---|---|
| Assign (agent, id not Nil) | set to the agent; reassign overwrites; unchanged owner makes no update | conversation first, then `CaseUpdateOwner`. A conversation failure returns an error and leaves the Case unchanged; a Case failure after a successful sync leaves the conversation owned and the retry converges (both steps idempotent). After the Case write the Case is re-read; if it was open at the first read and is closed now (a Close committed in between and had already released the conversation) the conversation is released again. A failure of that re-release is returned (unlike Close) because the Case write is done and the retry converges. A Case that is already closed at the first read gets no conversation sync at all for Assign and Unassign: a continued open Case can point at the same conversation through the chain and must not lose its owner. |
| Unassign (type none, id Nil) | release only if the conversation owner equals the previous Case owner; skipped when the Case had no owner | same order. The previous owner is read before the write, so syncing first keeps the retry convergent. |
| Close (genuine close only) | release unconditionally; `Case.Owner` is kept (Continue's authorization relies on it) | the close is committed first. A sync failure is logged at error level and not returned: a retry only reports AlreadyClosed, and re-releasing from a stale retry could take away a conversation that a continued Case has since assigned. The agent can release it through the conversation API. |
| Close already closed, Continue | none | |

Other (ownerType, ownerID) combinations skip the sync. A vanished conversation (typed `StatusNotFound`) and a conversation of another customer are no-ops (the latter with a warning).

(d) A running aicall gets no action: new inbound messages skip the Flow because `getExecuteMode` returns Agent, the idle aicall expires by its own rule, and a reply already generating may still be delivered once. On Unassign the next message reuses the aicall if it is not idle-expired.
(e) Reassign A to B sets B. Continue neither has an owner nor touches the conversation; a later Assign finds the conversation through the chain.
(g) Permission and customer checks are unchanged (api-manager; conversation-manager `Update`).

Known limits (accepted):
- Assign and Unassign (or two Assigns) on the same Case are not serialized. Concurrent calls can leave the Case and the conversation with different owners until the next Assign or Unassign.
- conversation-manager `Update` classifies a transient agent-manager failure as an invalid argument, so Assign can now return a 400 or 5xx from the conversation step; the retry converges.
- Cases assigned before this change keep an unowned conversation (backfill is a data operation, out of scope).
- contact-manager `Assign` does not reject a closed Case (it only updates `Case.Owner` as before and skips the conversation sync); api-manager's closed guards reject such calls.

## 3. Policy decisions

- Q1 (several conversations): only the conversation found through `ReferenceID` and the `PreviousCaseID` chain is synchronized. A second conversation of the same peer opened through another self address (second number or account; the open-Case key is customer, peer and reference_type) stays unlinked and keeps the Flow.
- Q2 (Case close): Close releases only the conversation owner, unconditionally, and keeps `Case.Owner`.
- Q3 (Unassign): release only when the conversation owner equals the previous Case owner.

Q2 (unconditional) and Q3 (conditional) are different rules on purpose. Close ends the interaction, so the conversation must return to the Flow whoever holds it. Unassign only withdraws one agent from a still open Case, so it must not take away a conversation another agent holds.

## 4. Verification

Unit tests in `pkg/casehandler/assign_conversation_test.go` (gomock request handler, sqlite): assign, reassign, same-owner idempotence, call-origin, empty or bad reference, chain resolution, broken, cross-customer and over-long chain, chain database error, conversation NotFound, other-customer conversation, Get and Update failures with retry convergence for Assign and Unassign, closed-between-read-and-write re-release, closed Case Assign and Unassign without a conversation call, unsupported combination; Unassign for matching, foreign and already-unowned owner and for an unowned Case; Close release with kept Case owner, call-origin Close, Close sync failure, already-closed Close. The receiving side is covered by conversation-manager `Test_getExecuteMode` (agent owner returns Agent, none returns Flow) and the `Update` tests of `conversationhandler/db_test.go`; the tests above pin the exact field map `{owner_id: <uuid>}` this change sends. Mutation checks in a scratch copy (drop the Update call, the customer check, the Unassign equality guard, the chain customer check, the idempotence check, the chain error return, the Close sync, the closed re-release, the closed-Case skip, the conversation-first order) each fail at least one test.

Commands in `bin-contact-manager`: `go mod tidy && go mod vendor && go generate ./... && go build ./... && go vet ./... && go test ./... && golangci-lint run -v --timeout 5m`.

Docs: `docs/domain.md` (behavior) and `docs/dependencies.md` (new RPC use). RST and OpenAPI are unchanged because no external contract changes.

## 5. Out of scope

- Backfill of already assigned Cases.
- Syncing conversations linked to a call-origin Case only through `Metadata.ContactCaseID` (needs a metadata filter in conversation-manager).
- Frontend: the Case response is identical, no square-talk change.
