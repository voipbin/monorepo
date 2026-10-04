# VOIP-1561: Inherit groupcall owner for agent-bound chained calls

Status: Draft
Ticket: VOIP-1561 (sibling: SQUARE-87)
Source: Fully re-verified via 8-round analysis review loop (rounds 7-8 consecutive APPROVED), see `../../../.worktrees/SQUARE-87-In-call-caller-context-panel-and-live-transcript/square-talk/docs/plans/2026-10-04-incall-context-and-live-transcript-analysis.md` §4.1, §4.3 (a-2). CEO confirmed decision (1)/(6): implement (a-2), linear 2nd+ destination path left unsupported (follow-up).

## 1. Problem statement

Every inbound-to-agent call leg (queue assignment, flow `connect`, AI tool agent-transfer, direct extension dial, blind/attended transfer landing on an agent) is created with `owner_id = Nil`, while agent-initiated outbound legs get `owner_id = agent`. Root cause: these inbound legs are always the inner SIP "chained call" of a 2-level groupcall structure, and the function that creates that chained call (`CreateCallOutgoing`) determines owner purely from `getAddressOwner(destination)`, which cannot resolve an agent from a raw SIP contact URI (it returns `OwnerTypeNone`) and has no fallback.

Consequence for square-talk (SQUARE-87, blocked by this ticket):
- `GET /service_agents/calls` (owner-filtered) never lists the agent's own inbound calls.
- `GET /service_agents/calls/{id}` 403s for inbound calls (`serviceagent_call.go:77`, `AgentID() != OwnerID`).
- The `agent_id:<id>:call` websocket topic never fires for inbound calls (routing key is built from `OwnerID`, `bin-webhook-manager/pkg/webhookhandler/routingkey.go:49-51`).

## 2. Goals

1. An agent-bound chained SIP call leg created under an owner=agent parent groupcall is itself created with `owner_type=agent, owner_id=<that agent>`.
2. No change to any RPC request/response shape, proto, or OpenAPI contract. No vendor/mock regeneration in any other service.
3. No change to the owner of groupcalls or calls whose destination is independently resolvable (tel/sip NOT nested under an agent groupcall) — today's `getAddressOwner(destination)` result continues to take priority.
4. Verified test coverage for every call path identified in the analysis: queue, flow `connect`, AI tool, direct `POST /calls` to an agent, blind transfer, attended transfer, both `ring_method=ringall` and `ring_method=linear` agents. `CreateCallOutgoing` rejects any destination whose `Type` is not `TypeSIP`/`TypeTel` at its own entry guard (`outgoing_call.go:148-150`) — by the time any of the above paths reaches this function, the destination has already been resolved down to a SIP contact URI (`getDialDestinationsAddressTypeExtension`, `dial.go:179-209`) or is a directly-dialed tel/sip address. `getAddressOwner`'s `TypeAgent` branch (`start.go:755-761`, backed by `AgentV1AgentGet`) is therefore unreachable from this call site — it only exists in `getAddressOwner` because the same helper is also consulted elsewhere in the package. Test coverage at this level is correctly expressed entirely in terms of SIP/Tel destinations (§6 below); no `TypeAgent`/`TypeExtension` destination test case is meaningful or addable here.

## 3. Non-goals

- Fixing `ring_method=linear` 2nd+ destination inheriting owner (`groupcallhandler/dial.go:90` path) — parent groupcall owner is structurally `None` there (no owner to inherit from without a separate RPC/lookup change). CEO-confirmed non-goal (decision 6); filed as follow-up once a concrete need surfaces.
- Fixing `ring_method=linear` being effectively ignored for agent destinations (pre-existing defect, `start.go:325-360` fires all agent addresses concurrently via goroutines) — pre-existing, out of scope, filed as follow-up defect candidate.
- Backfilling historical (pre-deploy) `owner_id=Nil` call rows — CEO-confirmed non-goal (decision 4).
- Any bin-common-handler RPC signature change (the a-1 alternative) — explicitly rejected in favor of (a-2)'s local-only change.
- square-talk frontend changes — tracked entirely in SQUARE-87.

## 4. Affected files

| File | Why |
|---|---|
| `bin-call-manager/pkg/callhandler/outgoing_call.go` | `CreateCallOutgoing`: add parent-groupcall owner fallback right after the existing `getAddressOwner` call (~line 287). |
| `bin-call-manager/pkg/callhandler/outgoing_call_test.go` | New unit tests for the fallback (table-driven, add cases to/near `Test_CreateCallOutgoing_TypeSIP`). |
| `bin-call-manager/pkg/callhandler/main.go` | No signature change expected; `groupcallHandler` field already present on `callHandler` (line 160) and already used elsewhere in the same file (`outgoing_call.go:84,507`) — confirms `h.groupcallHandler.Get` is reachable without any wiring change. |

No other file needs to change. `groupcallhandler/start.go` (chained-call creation call sites at `:153`, `:230`, `:355`) and `groupcallhandler/dial.go:125` are NOT touched — they already call `CallV1CallCreateWithID` → eventually `CreateCallOutgoing`, which is the single inheritance point. `bin-common-handler/pkg/requesthandler/call_calls.go:157-172` (`CallV1CallCreateWithID` signature) is explicitly NOT touched (confirms no RPC/vendor impact).

## 5. Exact change

### 5.1 Current code (`outgoing_call.go`, around line 285-291)

```go
	// get address owner info
	ownerType, ownerID, err := h.getAddressOwner(ctx, customerID, &destination)
	if err != nil {
		// we could not find owner info, but just write the log here.
		log.Errorf("Could not get address owner info. err: %v", err)
	}
```

### 5.2 New code

```go
	// get address owner info
	ownerType, ownerID, err := h.getAddressOwner(ctx, customerID, &destination)
	if err != nil {
		// we could not find owner info, but just write the log here.
		log.Errorf("Could not get address owner info. err: %v", err)
	}

	// Fallback: if the destination address itself has no resolvable owner
	// (e.g. a raw SIP registrar-contact URI for an agent's extension) but
	// this call is the inner leg of a groupcall whose OWN owner is known
	// (e.g. the outer agent-destination groupcall created by
	// groupcallhandler.startWithDestination), inherit that owner. This is
	// a call-manager-internal fallback only: it does not change the
	// priority of a directly-resolvable destination owner, and it never
	// fires for a groupcall whose own owner is None (ringall/linear outer
	// groupcalls, and the linear 2nd+ destination path -- see
	// docs/plans/2026-10-04-groupcall-owner-inheritance.md §3 non-goals).
	if ownerType == commonidentity.OwnerTypeNone && groupcallID != uuid.Nil {
		parentGroupcall, errGroupcall := h.groupcallHandler.Get(ctx, groupcallID)
		switch {
		case errGroupcall != nil:
			log.Errorf("Could not get parent groupcall for owner inheritance. groupcall_id: %s, err: %v", groupcallID, errGroupcall)
		case parentGroupcall.OwnerType != commonidentity.OwnerTypeNone:
			log.Debugf("Inheriting owner from parent groupcall. groupcall_id: %s, owner_type: %s, owner_id: %s", groupcallID, parentGroupcall.OwnerType, parentGroupcall.OwnerID)
			ownerType = parentGroupcall.OwnerType
			ownerID = parentGroupcall.OwnerID
		}
	}
```

Placement: strictly after the existing `getAddressOwner` call and strictly before the `h.Create(...)` call (currently ~line 293) that consumes `ownerType`/`ownerID`. No other line in the function changes.

### 5.3 Wire-field checklist (verified against this repo 2026-10-04)

| Field | Source | Verified shape |
|---|---|---|
| `groupcall.Groupcall.OwnerType` | `bin-call-manager/models/groupcall/main.go:13-15` (`Groupcall` embeds BOTH `commonidentity.Identity` (ID/CustomerID, `identity.go`) AND `commonidentity.Owner` (OwnerType/OwnerID, `owner.go`) per hard-copy-forbidden convention) | `commonidentity.OwnerType`, field promoted from the embedded `Owner` struct; confirmed via `groupcallhandler/db.go:121` `Get` returning `*groupcall.Groupcall` |
| `groupcallHandler.Get(ctx, id uuid.UUID) (*groupcall.Groupcall, error)` | `bin-call-manager/pkg/groupcallhandler/db.go:121-133` | Returns typed `cerrors.NotFound` (`Status=NotFound`) on `dbhandler.ErrNotFound`, wrapped generic error otherwise — handled above via the `errGroupcall != nil` branch, no special-casing needed since both outcomes just skip inheritance and log. |
| `commonidentity.OwnerTypeNone` / `OwnerTypeAgent` | `bin-common-handler/models/identity/owner.go:16-17` | `OwnerTypeNone OwnerType = ""`, `OwnerTypeAgent OwnerType = "agent"` |
| `h.groupcallHandler` field | `bin-call-manager/pkg/callhandler/main.go:160,315,330` | Already wired into `callHandler`; already called in the same file at `outgoing_call.go:84` (`IsGroupcallTypeAddress`) and `:507` (`Start`) — no new dependency injection needed. |
| `groupcallID` parameter | `CreateCallOutgoing` signature, `outgoing_call.go` (~line 107-122) | Already a parameter of the function; passed through from every call site (`groupcallhandler/start.go:153,230,355`, `dial.go:125`) — confirmed via `grep -n groupcallID bin-call-manager/pkg/groupcallhandler/start.go` before drafting. |

## 6. Call-path coverage matrix (test plan, maps to decision (1)/(4.1))

| Path | Outer groupcall owner | Inner chained call before fix | After fix |
|---|---|---|---|
| Queue → agent, `ring_method=ringall`, single agent in queue | agent (`startWithDestination`) | None | agent (inherited) |
| Queue → agent, `ring_method=ringall`, multiple agents (fan-out) | agent per-branch (`startRingall:136-147`, `mapGroupcalls` goroutine fan-out, each branch recurses into `startWithDestination(ctx, chainedGroupcallID, ...)` with its OWN `chainedGroupcallID`, not the outer `id`) | None | agent (inherited per-branch; `groupcallID` passed into `CreateCallOutgoing` is each branch's own `chainedGroupcallID`, which independently resolves to `OwnerTypeAgent` via `h.groupcallHandler.Get`) |
| Queue → agent, `ring_method=linear` (single agent) | agent (`startWithDestination`, via `startLinear` nested groupcall) | None | agent (inherited) |
| Flow `connect` → agent | agent | None | agent (inherited) |
| AI tool → agent | agent | None | agent (inherited) |
| `POST /calls` (API) → agent destination | agent | None | agent (inherited) |
| Blind transfer → agent | agent | None | agent (inherited) |
| Attended transfer → agent | agent | None | agent (inherited) |
| Agent-initiated outbound (browser dial) | n/a (no groupcall) | agent (via `getAddressOwner(source)`, unaffected path) | agent (unchanged, regression check only) |
| `ring_method=linear`, 2nd+ destination is an extension | None (`dial.go:90` nested groupcall) | None | None (unchanged — non-goal §3) |
| Tel/SIP destination NOT nested under an agent groupcall | directly resolved by `getAddressOwner` | directly resolved | unchanged (fallback never triggers, `ownerType != OwnerTypeNone`) |

## 7. Verification plan

1. `cd bin-call-manager && go build ./...` — must pass with zero new warnings.
2. `go test ./pkg/callhandler/... -run CreateCallOutgoing -v` — all existing tests green (regression), plus new table-driven cases covering every row of §6 above.
3. `go vet ./...` and the repo's standard lint target (per `CLAUDE.md` build/test commands — confirm exact command before running).
4. Manual trace re-check: after implementing, re-grep `groupcallhandler/start.go:153,230,355` and `dial.go:125` to confirm none of their call sites needed changes (they shouldn't — single inheritance point is `outgoing_call.go`).
5. No changes expected to `go.sum`/vendor in any other service — confirm with `git status` scoped to the worktree showing only `bin-call-manager/pkg/callhandler/outgoing_call.go` and its test file.

## 8. Rollout / risk

- **Risk: increased exposure.** Previously-`owner=Nil` inbound call rows become visible to the owning agent in list/detail/delete. This is the intended effect (SQUARE-87 depends on it) but widens what an agent can see/delete about their own calls. No cross-tenant exposure (owner is still scoped to the correct agent/customer).
- **Risk: event volume.** `agent_id:<id>:call` webhook/websocket events will fire for calls that previously generated no agent-scoped event. Expected and required for SQUARE-87's "On call now" / live list features; no fan-out to other customers.
- **Risk: none for RPC/schema compatibility** — single-file, call-manager-internal change; no proto/OpenAPI/vendor touch confirmed in §4/§7.5.
- **Rollback:** revert the single diff in `outgoing_call.go`; no data migration involved (no non-goal-4 backfill means no destructive or stateful rollback concern).

## 9. Open questions

None outstanding — all CEO decision points for this ticket's scope (decisions 1, 4, 6) were locked before this doc was drafted. Reviewer should focus on: correctness of the fallback condition (especially interaction with the `ownerType == OwnerTypeNone` check when `getAddressOwner` itself errored vs. cleanly returned None), and whether §6's coverage matrix is actually exhaustive against the current code.

## 10. Approval status

**APPROVED** — Design Review→Fix loop closed. Round 1: CHANGES_REQUESTED (2 items, fixed). Round 2: APPROVED. Round 3: APPROVED (2 consecutive, min-3-round floor satisfied). Non-blocking round-3 note (explicit extension-dial test case) folded into §2 Goal 4 above.

## Iter-1 review response summary

- 1 (§5.3 모델 인용 오류): `groupcall.Groupcall.OwnerType`의 실제 정의 파일을 `models/groupcall/main.go:13-15`로 정정, `Identity`(ID/CustomerID)와 `Owner`(OwnerType/OwnerID)를 별도로 embed한다고 정정. 본인 재확인(`sed -n '1,20p' models/groupcall/main.go`).
- 2 (§6 매트릭스 ringall 다중 목적지 fan-out 미반영): `startRingall:136-147`의 `mapGroupcalls` 고루틴 fan-out(각 branch가 outer `id`가 아닌 자신의 `chainedGroupcallID`로 `startWithDestination` 재귀 호출) 행을 매트릭스에 추가. 본인 재확인(`sed -n '130,160p' pkg/groupcallhandler/start.go`).

## Iter-2 review response summary (round 3, non-blocking)

- Round 3 승인, 비차단 보완 2건 중 1건(Extension dial 테스트 명시성)만 반영: §2 Goal 4에 direct-extension-dial 테스트 케이스 요구사항 명시. 나머지 1건(§3 비고의 `start.go:325-360` 줄 번호 근사치)은 사소한 범위 오차로 수정 불필요(실제 322-363, 본문 서술 내용 자체는 정확).

## Iter-3 review response summary (PR review round 3 finding, design doc correction)

- **이 응답은 코드가 아니라 §2 Goal 4 자체의 오류를 바로잡습니다.** PR 코드 리뷰 3회차에서 "TypeAgent/TypeExtension 목적지 테스트 케이스 누락"을 CHANGES_REQUESTED로 지적했으나, 재검증 결과 그 요청의 전제(Iter-2가 추가한 "`IsGroupcallTypeAddress`/`getDialDestinationsAddressTypeExtension`/`getAddressOwner`가 TypeAgent/TypeExtension을 동일 코드 경로로 처리한다")가 사실과 다릅니다.
  - `CreateCallOutgoing`은 자체 진입 가드에서 `destination.Type`이 `TypeSIP`/`TypeTel`이 아니면 즉시 에러를 반환합니다(`outgoing_call.go:148-150`, 본인 재확인). 따라서 이 함수 내부에서 `getAddressOwner`가 호출될 때 `destination.Type`은 항상 SIP 또는 Tel이며, `TypeAgent` 분기(`start.go:755-761`, `AgentV1AgentGet` 사용)는 **도달 불가능**합니다.
  - Extension으로 가는 모든 경로는 `getDialDestinationsAddressTypeExtension`(`dial.go:179-209`)에서 registrar contact URI로 변환된 뒤 `TypeSIP` 목적지로 `CreateCallOutgoing`에 들어옵니다. 즉 기존 5개 테스트 케이스의 SIP 목적지(`registrar-contact-uri@test.com`, `sip-contact-2@test.com` 등)가 이미 "direct extension dial" 시나리오를 정확히 표현하고 있습니다.
  - §2 Goal 4를 위 사실에 맞게 재작성했습니다. `TypeAgent`/`TypeExtension` 목적지 전용 테스트는 이 호출부에서는 **추가할 수 없는(의미 없는) 테스트**이므로 추가하지 않습니다. Iter-2의 해당 문구는 잘못된 근거였음을 여기 기록합니다.



