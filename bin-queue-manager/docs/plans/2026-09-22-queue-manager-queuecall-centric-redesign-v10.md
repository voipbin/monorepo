# Queue-manager Queuecall-centric Event-driven Redesign (v10)

Status: APPROVED (v10 확정 — iter-16·17 2연속 양쪽 APPROVE, 2026-09-23). 구현 착수 대기(대표님 승인)
Supersedes: `2026-09-22-queue-manager-event-driven-routing-redesign.md` (v9, "큐 스케줄러" 방식 — 대안 보존)
Worktree: `NOJIRA-Queue-manager-event-driven-routing-redesign`

## 0. v9 → v10 전환 이유

v9는 큐 스케줄러(`queue.execute` Run/Stop + `execute_run` self-RPC 루프)를 유지한 채 이벤트 wake를 얹었다. 대표님 반박: "큐의 `execute_run` 개념은 폴링 잔재. `waiting_queuecall_ids`는 모니터링용." 제어축은 큐가 아니라 **큐콜.status + 상담원 가용성**. v10은 큐 스케줄러를 제거하고 **큐콜 중심 순수 이벤트 매칭**으로 재설계한다.

**[iter-7/8 핵심] 병렬화의 대가.** v9 스케줄러는 매칭을 우연히 직렬화해 큐콜 status 전이의 경합을 가렸다. v10은 스케줄러를 제거해 이벤트 병렬 매칭이 되므로, **status 전이 전부가 서로/이벤트와 경합한다**. 따라서 **무조건 부수효과(webhook/confbridge delete/모니터링 배열 add·remove/변수 삭제/메트릭/RPC)를 가진 모든 status 전이 메서드는 예외 없이 blind→CAS 승자 게이팅**(§5.0에 전수 명세).

## 0.1 agent-manager 가용성 이벤트 실측

- agent-manager 구독은 `groupcall_created`, `groupcall_progressing`(→busy), `customer_*` **4종**(`subscribehandler/main.go:28-33`). hangup 구독 없음 → busy→available은 상담원 **수동 status API**로만.
- **[iter-4 결정] available 복귀는 제품/클라이언트 정책**(wrap-up 후 상담원 Ready). 백엔드는 available 복귀 자동화 안 함.
- `dbUpdateStatus`(`db.go:391`)는 모든 status 변경에 `agent_status_updated` 발행 → 매칭 트리거는 status==available만 필터(§8).
- **[iter-2] agent leg는 hangup 이벤트 경로 없음**: 큐콜 `ReferenceID`는 고객 콜 → 상담원 leg 정리는 능동 hangup 필수(§5).

## 0.2 [iter-4/5] agent `ringing` status 제거

**기능적 역할.** agent status 6개 중 `ringing`의 라우팅 기능은 available 후보 배제뿐. 세팅은 `EventGroupcallCreated`(`event.go:54`) 한 곳, 소비는 후보 조회 `status=available` 필터뿐. **방법 B 예약 필드가 그 배제를 대체**하므로 잉여. exit 경로 없어 "유령 ringing" 결함 클래스 근원.

**[iter-5 정정] 계약/문서 소비 표면 존재**(라우팅/UI 아닌 공개 계약):
- `openapi.yaml:283`(`- ringing`, `AgentManagerAgentStatus`), `:290`(varname)
- `bin-openapi-manager/gens/models/gen.go:636,652`(재생성물 1), `bin-api-manager/gens/openapi_server/gen.go`(재생성물 2)
- docsdev RST: `agent_struct_agent.rst:83`, `agent_overview.rst:10,75`, `queue_overview.rst:199,272`
- agent-manager 문서(domain/operations/architecture/README/CLAUDE)

**결정(대표님): enum 완전 제거(방법 B).** `ringing`은 백엔드가 **실제로 발행하던 라이브 라우팅 상태**라, 제거하려면 계약·문서·백필 전수 처리(파괴적 계약 변경 감수). **call status `ringing`(openapi 967/974 `CallManagerCallStatus`, 1113/1117 `tm_ringing`)은 별개 타입, 불변.**

**효과:** available→busy 직행(`EventGroupcallProgressing` 무가드 busy write). dial 중 배제는 예약 전담. 유령 ringing 소멸. `EventGroupcallCreated` 핸들러/구독 제거.

## 1. 핵심 모델

### 1.1 제어축 = 큐콜.status + 상담원 예약(별도 필드)
큐는 설정+통계 그릇. 라우팅 진실은 큐콜 `status`와 상담원 가용성(status)+예약(별도 필드).

### 1.2 매칭 = 이벤트 시점 1회 시도
큐콜 waiting / 상담원 available → 매칭 1회. 실패 시 예약 않고 끝. 놓친 엣지는 (나) 백스톱.

### 1.3 큐 배열 필드 = 모니터링 전용
`wait/service_queue_call_ids`는 고객 노출 모니터링. 라우팅 안 읽음.

## 2. Concurrency Model

| 실패축 | 증상 | 장치 |
|---|---|---|
| 과잉(OVER) | 한 상담원 2콜 | §3 큐콜 CAS + 상담원 예약 CAS(방법 B) |
| 부족(UNDER) | 대기 큐콜 미매칭 | §5 (나) 백스톱 |

- **I1 (iter-6~9 확정): 경합 status 전이는 조건부 CAS, 부수효과는 승자(affected>0)만.** "부수효과"=신규+기존 전부. **적용 범위 = 무조건 부수효과를 가진 큐콜 status 전이 메서드(§5.0). 부수효과 없는 죽은 setter는 대상 아님(§5.0 주).**
- **부수효과 소유자 단일 원칙(iter-9)**: 각 전이의 승자 부수효과는 **전이 메서드 내부**에만 귀속. 호출부(백스톱 등)는 메서드 호출만.
- I2: 멱등 재시도. CAS 패배 경로는 부수효과 없는 no-op, `(현재상태 res, nil)` 반환(Get 에러 시에만 `(nil, err)`). 호출부가 non-nil res 역참조(health.go:50 `res.ID` 등)하므로 계약 유지.
- **직렬화 기전(iter-9 정정)**: 같은 행 원자 CAS가 경합 전이를 직렬화(승자 1명). 술어가 논리적으로 disjoint일 필요 없음(abandon `!= 'service'`는 connecting과 겹치나 행잠금으로 안전).

## 3. 매칭 직렬화 (과잉 방지) — 방법 B

status는 이벤트 구동(available↔busy), 예약은 매칭 흐름만. 이벤트 경합 구조적 제거(iter-1).

### 3.1 예약 필드 (agent_agents 신규 3컬럼)
| 컬럼 | 타입 | 의미 |
|---|---|---|
| `reserve_reference_type` | varchar(255) | `"queuecall"`. 미예약=빈 값 |
| `reserve_reference_id` | binary(16) | 예약 주체 id(상관관계 토큰). zero=미예약 |
| `tm_reserve` | datetime(6) | 예약 시각. 좀비 판정 |

### 3.2 크로스서비스 경계 — 예약은 agent-manager RPC (iter-2)
queue-manager는 `agent_agents` DB 접근 없음(RPC만).
- `AgentV1AgentReserve(agentID, refType, refID)`: `WHERE status='available' AND reserve_reference_id=zero` CAS.
- `AgentV1AgentReserveRelease(agentID, refID)`: `WHERE reserve_reference_id=refID`.
- 좀비-sweep: agent-manager 자체, `tm_reserve < now-Tres` 시간 단독. REST `/v1/agents/{id}/reserve`, `/reserve_release`.

### 3.3 매칭 절차 (match())
```
1. cands = AgentV1AgentGetsByTagIDsAndStatus(q.tags, available); 없으면 실패
2. targetAgent 선택
3. ok = AgentV1AgentReserve(target, "queuecall", qc.ID); !ok → 다음 후보/실패
4. dial: calls, groupcalls = CallV1CallsCreate(..., [TypeAgent target])
   err → ReserveRelease; 끝 / len(groupcalls)==0 → ReserveRelease; 실패
   agentGroupcallID = groupcalls[0].ID
5. 진입 CAS(UpdateStatusConnecting, §5.0): WHERE status='waiting'
   affected=0 → CallV1GroupcallHangup(agentGroupcallID) + ReserveRelease; 끝
6. forward. 통화 성사 시 groupcall_progressing이 available→busy
```
dial→CAS 순서로 groupcall_id 확보. ringing 제거로 dial~CAS 사이 status 전이 없음.

### 3.4 예약 생명주기
- connecting 동안 유지, service 확정 승자 시 ReserveRelease. **Tres ≥ Tconn**. 좀비 sweep 시간 단독.
- 이중배정 방어축: 예약 필드 단독. dial 중 status=available라 후보 재등장하나 예약 CAS가 재선택 거부.

## 4. (가) 모니터링 reconcile — Lazy
```
GET queue: fresh=redis.EXISTS("queue:{id}:reconciled")
  fresh → DB 배열; stale/miss → 재계산 → DB write-back → redis.SET(key,1,EX=5s)
```
배열=DB, 신선도=Redis TTL. 멱등. Redis 장애→매 GET 재계산. 최대 5s stale.

## 5. status 전이 CAS 게이팅 + kick TOCTOU + (나) 백스톱

### 5.0 [iter-8/9] 큐콜 status 전이 메서드 전수 CAS 승자 게이팅

**무조건 부수효과를 가진 status 전이 메서드**가 blind + 무조건 부수효과(코드 전수 확인). 예외 없이 blind→CAS+affected 반환, 부수효과를 `affected>0` 블록으로. **각 전이는 전용 핸들러 메서드 1개 ↔ 전용 setter 1개**:

| # | 핸들러 메서드 | setter(→CAS) | CAS 술어 | 승자 부수효과(메서드 내부) |
|---|---|---|---|---|
| 1 | `UpdateStatusWaiting`(db.go:336, **initiating→waiting 전용**) | `QueuecallSetStatusWaitingIfInitiating`(신규) | `WHERE status='initiating'` | webhook Waiting(353), `AddWaitQueueCallID`(358) |
| 2 | `UpdateStatusWaitingRollback`(**신규, connecting→waiting 롤백 전용**) | `QueuecallSetStatusWaitingIfConnecting`(신규) | `WHERE status='connecting'` | service_agent_id·groupcall_id clear, `AgentV1AgentReserveRelease`(service_agent_id!=zero), 재큐잉 |
| 3 | `UpdateStatusConnecting`(db.go:177, waiting→connecting) | `QueuecallSetStatusConnecting`(→CAS) | `WHERE status='waiting'` | webhook Connecting(195), service_agent_id·groupcall_id write |
| 4 | `UpdateStatusService`(db.go:201, connecting→service) | `QueuecallSetStatusService`(→CAS) | `WHERE status='connecting'` | prom(221), webhook serviced(222), `AddServiceQueuecallID`(224), timeout RPC(231), ReserveRelease |
| 5 | `UpdateStatusDone`(db.go:288, service→done) | `QueuecallSetStatusDone`(→CAS) | `WHERE status='service'` | prom(308), webhook done(309), `RemoveQueuecallID`(312), **CallV1ConfbridgeDelete(321)**, 변수(328) |
| 6 | `UpdateStatusAbandoned`(db.go:240, 공용→abandoned) | `QueuecallSetStatusAbandoned`(→CAS) | `WHERE tm_end IS NULL AND status != 'service'` | prom(260), webhook abandoned(261), `RemoveQueuecallID`(264), **CallV1ConfbridgeDelete(273)**, 변수(280), leg hangup(groupcall_id!=zero), ReserveRelease(service_agent_id!=zero) |

**공통 패턴:**
```
QueuecallSetStatusX: blind → CAS ... WHERE id=? AND <술어>; return affected
UpdateStatusX(qc):
    affected = QueuecallSetStatusX(...)
    if affected == 0:                              // CAS 패배 → 멱등 no-op
        res, err = h.Get(qc.ID); if err!=nil: return nil,err
        return res, nil                            // 현재상태(경합 승자) 반환. 부수효과 전무
    res, err = h.Get(qc.ID); if err!=nil: return nil,err
    <해당 메서드 전 부수효과>                        // 승자만, 메서드 내부에서만
    return res, nil
```

**[iter-9] 공유 메서드/setter 분리 (핸들러 계층 API 명확화):**
- 기존 `UpdateStatusWaiting(ctx, id)`는 **initiating→waiting 전용**으로 유지(#1). 호출부: `v1_queues.go:506`(단일).
- connecting→waiting 롤백(#2)은 **신규 메서드 `UpdateStatusWaitingRollback(ctx, qc)`**로 분리. 부수효과(clear·ReserveRelease·재큐잉)를 **이 메서드 내부**에서 수행. 호출부: §5.2 백스톱(유일).
- 기존 blind setter `QueuecallSetStatusWaiting`은 **두 술어 setter로 대체**: `...IfInitiating`(#1), `...IfConnecting`(#2). 단일 메서드 셀렉터 방식 안 씀.
- **소유자 단일(I1)**: #2 부수효과는 `UpdateStatusWaitingRollback` 내부에만. 백스톱은 호출만.

**완결성 근거(코드 grep 전수).** db.go 무조건 부수효과 소비처: PublishWebhookEvent(132/171/195/222/261/309/353), AddWaitQueueCallID(358), AddServiceQueuecallID(224), RemoveQueuecallID(264/312), ConfbridgeDelete(273/321). 이 중 status 전이는 위 6행이 **전부**. Create(132)/Delete(171)는 status 전이 아닌 생성/삭제(범위 밖).

**[iter-10 정정] 죽은 코드 `StatusKicking`는 건드리지 않는다(오버엔지니어링 회피).**
`StatusKicking`(모델 queuecall.go:77) + `QueuecallSetStatusKicking`(dbhandler:440)은 핸들러 계층 호출자 0(grep: def/interface/mock/test만). **부수효과 없고 라이브 경합 없음 → CAS 게이팅 대상이 아니다.** iter-9는 이를 제거해 "잔존 blind setter 0" 수사적 완결을 노렸으나, 두 리뷰어(iter-10)가 지적: 제거는 CAS 정합성에 불필요하고, 공개 계약 표면(openapi `QueueManagerQueuecallStatus` `kicking` enum, 재생성물 2, docsdev RST 2)까지 흔드는 파괴적 크로스서비스 변경을 유발한다(대표님 오버엔지니어링 지양 원칙 충돌). **결정: 무해한 죽은 상수·setter·enum을 그대로 둔다. "잔존 blind setter 0" 주장 철회.** `ringing`(백엔드가 실제 발행하던 라이브 상태 → 전수 제거)과 `kicking`(원래부터 미발행 죽은 enum, gens 자체 상수라 무해 → 유지)의 처리 비대칭은 이 근거로 정당. 게이팅 범위는 "부수효과 가진 전이 6종"으로 충분하고 완결.

**왜 6종 전부 CAS인가(경합 실재, 오버엔지니어링 아님 — iter-8 적대 리뷰 코드 확인):**
- waiting(1): abandon(initiating 중 고객 hangup, `event.go:16-38`이 initiating 배제 안 함) 승리 후 blind waiting-write가 **버려진 콜 부활**.
- rollback(2): join(connecting→service)과 경합.
- connecting(3): 병렬 매처 2개가 같은 waiting 큐콜 경합 → 패자 blind write가 **중복 connecting webhook + 승자 agent 배정 덮어씀**.
- service(4): abandon 패배해도 blind면 **유령 serviced webhook + 유령 배열 + 유령 timeout RPC**.
- done(5): 2 writer(ConfbridgeLeaved event.go:90 + kickForce kick.go:107) 동시 → **중복 + 재삭제**.
- abandoned(6): service 패배 시 **라이브 콜 confbridge 삭제**(치명).
각 전이 전용 setter의 술어로 같은 행 원자 CAS → 승자 1명.

**불변식 I-CB**: confbridge delete(done/abandon)·모니터링 배열 add(waiting/service)/remove(done/abandon)는 **반드시 승자 블록 안**. service가 큐콜 confbridge 재사용(`UpdateStatusService` ConfbridgeID 미재할당, `EventCallConfbridgeJoined` 동일성 검증)하므로 패배자의 confbridge delete는 라이브 콜을 끊는다. 향후 리팩터 보존.

### 5.1 [iter-8] kick.go 선행 부수효과 TOCTOU (Kick + kickForce 둘 다)

`Kick`(kick.go:15)/`kickForce`(kick.go:~90) 모두 `FlowV1ActiveflowServiceStop`을 status 전이 CAS **전에, stale `qc.Status`(Get 후 미재조회) 기반**으로 실행. 콜이 그 사이 service 전이되면 라이브 콜 flow 중단 + 후속 stale 분기 오판. **flow-stop 후 status 재조회 필수.**

- **`Kick`**(자발적): 재조회 후 `service`면 confbridge_leaved 위임(return), 아니면 `UpdateStatusAbandoned`.
```
qc=Get(id); if terminal: err
FlowV1ActiveflowServiceStop(...)
qcFresh=Get(id)
if qcFresh.Status==service: return qcFresh,nil
UpdateStatusAbandoned(qcFresh)
```
- **`kickForce`**(force-kill, 위임 불가 — health.go:45 헬스체크 소진, event.go:120 EventCUCustomerDeleted): 재조회 후 `service`면 `UpdateStatusDone`(강제 종료), 아니면 `UpdateStatusAbandoned`.
```
qcFresh=Get(id)
if qcFresh.Status==service: UpdateStatusDone(qcFresh)
else: UpdateStatusAbandoned(qcFresh)
```
- **[iter-6 과잉주장 철회]** "공용 메서드라 전 호출부 자동 안전"은 메서드 내부 부수효과에만 성립. kick 선행 flow-stop은 게이트 밖 → 재조회로 별도. 잔여 창(재조회~내부 CAS)은 양성.

### 5.2 (나) 매칭 백스톱 — 저빈도 능동
```
매 30초 (로컬 타이머):
  1. Redis 리스 SET reconcile:match {inst} NX EX 35; 실패 → skip
  2. [복구 A] connecting-stale: status=connecting AND tm_update < now-Tconn
        gid=groupcall_id; if gid!=zero: if !CallV1GroupcallHangup(gid): continue
        UpdateStatusWaitingRollback(qc)   // §5.0 #2. 부수효과는 메서드 내부. 백스톱은 호출만
  3. [복구 B] waiting 큐콜(tm_create ASC) → match() 재호출(멱등)
  4. 리스 DEL
```
Tconn > call-manager 최대 agent-dial 타임아웃 + 마진. 좀비 예약 해제 백업은 agent-manager sweep.

### 5.3 join CAS 패배 teardown
백스톱 A가 롤백 시 leg hangup+clear하므로, join 핸들러(`EventCallConfbridgeJoined → UpdateStatusService`, §5.0 #4)가 CAS affected=0을 만나면 groupcall_id는 이미 zero → no-op 반환(백스톱이 유일 정리 주체).

## 6. DB 변경

| 테이블 | 변경 |
|---|---|
| `queue_queues` | `execute` 컬럼 **drop** (**[iter-11] 최후행: §10 step 5, 스케줄러 제거 배포 후. 구 바이너리가 struct SELECT 로드하므로 선DROP 시 런타임 실패**) |
| `queue_queuecalls` | `(queue_id, status)` 인덱스 **add** + `groupcall_id` binary(16) **add** |
| `agent_agents` | `reserve_reference_type`, `reserve_reference_id`, `tm_reserve` **3컬럼 add** |

status CAS는 스키마 무변경(WHERE절만). Alembic은 bin-dbscheme-manager 생성.
**[iter-6] agent `ringing` 잔존 행 백필 필수**. enum 제거 배포 전 `UPDATE agent_agents SET status='available' WHERE status='ringing'`.

## 7. 상태 전이 CAS 술어 (§5.0 6종과 1:1)

| 전이 | 술어 | 승자 부수효과 |
|---|---|---|
| initiating→waiting | `WHERE status='initiating'` | webhook Waiting + AddWaitQueueCallID |
| connecting→waiting (롤백) | `WHERE status='connecting'` | clear·ReserveRelease·재큐잉 |
| waiting→connecting | `WHERE status='waiting'` | **webhook Connecting** + service_agent_id·groupcall_id |
| connecting→service | `WHERE status='connecting'` | prom·webhook serviced·**AddServiceQueuecallID**·timeout·ReserveRelease |
| service→done | `WHERE status='service'` | prom·webhook done·RemoveQueuecallID·**confbridge delete**·변수 |
| →abandoned (공용) | `WHERE tm_end IS NULL AND status != 'service'` | prom·webhook abandoned·RemoveQueuecallID·**confbridge delete**·변수·leg hangup·ReserveRelease |

**I-CB**: confbridge delete·모니터링 배열 add/remove는 반드시 승자 블록 안.

상담원(agent-manager): available→busy(`groupcall_progressing` 무가드 직행) / busy→available(수동 API) / 예약 획득·해제·좀비(§3.2 술어).

## 8. 이벤트 구독 / RPC

- queue-manager 신규 구독: `agent.*.status_updated`, status==available만.
- 신규 RPC(agent-manager): `AgentV1AgentReserve`, `AgentV1AgentReserveRelease`.
- 재사용: `AgentV1AgentGetsByTagIDsAndStatus`, `CallV1GroupcallHangup`, `CallV1ConfbridgeDelete`.
- 제거: `execute_run` self-RPC, agent-manager `EventGroupcallCreated`.

## 9. 상수

| 상수 | 값(초기) | 용도 |
|---|---|---|
| Tconn | 최대 dial 타임아웃+마진(예: 90s) | connecting-stale |
| Tres | ≥ Tconn(예: 120s) | 좀비 예약 |
| 백스톱 주기 | 30s | |
| 리스 TTL | 35s | |
| (가) 신선도 TTL | 5s | |
| agent 좀비 sweep 주기 | 60s | |

## 10. Phase 구성 + 크로스서비스 배포 순서

- **Phase 1** (CAS 안전화 + 스케줄러 제거 + 예약):
  1. **CAS 인프라: 큐콜 status 전이 6종(§5.0) blind→CAS+affected 반환+부수효과 승자 게이팅. `QueuecallSetStatusWaiting`→`...IfInitiating`/`...IfConnecting` 2 setter, `UpdateStatusWaitingRollback` 신규. kick.go Kick·kickForce 재조회(§5.1). mock/test 재배선(§11).** (StatusKicking은 건드리지 않음, §5.0)
  2. 큐 스케줄러 제거: `execute_run`/`UpdateExecute` 루프 삭제 **+ [iter-12] `models/queue/queue.go`의 `Execute` 필드(`db:"execute"` 태그 포함, :28)·`Execute` 타입(:58)·`ExecuteRun`/`ExecuteStop` 상수(:62-63) 제거 + `models/queue/field.go` `FieldExecute`(:19)·`models/queue/filters.go` `FieldStruct.Execute`(:12) 제거 + execute write 소비처(`queuehandler/create.go:97`, `queuehandler/queue.go:35-37`) 제거 + `bin-common-handler` `QueueV1QueueExecuteRun`/`QueueV1QueueUpdateExecute`(+mock) 제거**. **이것이 step 5(DB 컬럼 DROP) 안전의 전제**: struct에서 `Execute` 필드를 빼야 `queueGetFromDB`의 `GetDBFields(&queue.Queue{})`(dbhandler/queue.go:135) 리플렉션 SELECT에서 `execute`가 빠지고, 그래야 새 바이너리가 execute 컬럼을 조회하지 않는다. **필드를 남기면 새 바이너리도 `SELECT ... execute ...`를 발행해 step 5 DROP이 새 바이너리를 크래시**(iter-12).
     - **[iter-13] 스코프 성격**: 위 목록은 전수 파일 나열이 아니라 **① silent-risk(struct `Execute` 필드 — Go가 unused 필드에 컴파일 에러를 안 내므로 유일하게 조용히 방치될 위험)와 ② 주요 소비처**를 명시한다. 나머지 execute 참조(`listenhandler` execute/execute_run 라우트·processor·request model, `cmd/queue-control` cmdUpdateExecute, 관련 test 전수, docs/architecture·operations 서술)는 **`Execute` 타입 제거 시 컴파일러가 강제 노출**하므로 구현자가 `go build`로 전수 포착(전수 나열 불요). **크로스서비스 파손 없음**: execute RPC 소비처는 bin-queue-manager 내부 self-RPC뿐(flow/call/api-manager 등 타 서비스 0건, iter-13 grep 확증).
  3. agent `ringing` 백필 선행 → 전 표면 제거(상수, EventGroupcallCreated, openapi enum+양쪽 재생성물, docs). call ringing 불변.
  4. 예약 3컬럼 + agent-manager 예약/해제 RPC + 좀비 sweep. 큐콜 `groupcall_id` 컬럼.
  5. 이벤트 매칭(§3.3).
  6. (나) 백스톱(§5.2) + connecting-stale.
  7. `(queue_id,status)` 인덱스.
- **Phase 2**: (가) Lazy 모니터링. connecting 정밀 타임아웃.
- **Phase 3**: 튜닝, 라우팅 전략.

**[iter-10] 크로스서비스 배포 순서 제약** (Phase 1은 6 리포, 리포당 1 PR — 런타임 의존 순서 준수):
1. **`bin-dbscheme-manager`**: 예약 3컬럼 + queuecall `groupcall_id` + `(queue_id,status)` 인덱스 마이그레이션. **최선행**(컬럼 없으면 이후 코드 런타임 실패).
2. **`bin-common-handler`** 예약 RPC 클라이언트 + **`bin-agent-manager`** 예약/해제 RPC 서버·좀비 sweep·`agent_agents` 3필드 사용. **agent 예약 엔드포인트가 라이브된 뒤**에야 queue 소비 가능.
3. **`bin-queue-manager`** match()가 예약 RPC 소비 + CAS 게이팅 + 스케줄러 제거. **step 2 이후 배포**(RPC 미라이브 시 매칭 실패). ← 배포 게이팅.
4. **agent `ringing` 백필(§6) → `bin-openapi-manager`/`bin-api-manager` enum 제거·재생성·docs**. 백필이 enum 제거보다 **선행**(enum 밖 잔존 행 방지).
5. **[iter-11/12] `bin-dbscheme-manager` `queue_queues.execute` 컬럼 DROP = 최후행(별도 2차 마이그레이션).** step 1의 dbscheme 마이그레이션은 **additive만**(예약 3컬럼 + queuecall `groupcall_id` + `(queue_id,status)` 인덱스). `execute` **DB 컬럼 DROP**은 **파괴적**이다. **안전 전제(iter-12)**: step 2에서 `Execute` struct 필드를 제거해야 새 queue-manager의 `GetDBFields` 리플렉션 SELECT에서 execute가 빠진다. 이 전제가 충족된 **step 3(스케줄러 제거·Execute 필드 제거된 새 queue-manager) 배포 확인 후** DROP. 필드 제거 없이 DROP하면 구·신 바이너리 모두 크래시.
의존은 단방향(순환 없음). 각 PR은 이전 단계 배포 확인 후 머지. **dbscheme는 2회 배포**(step 1 additive 최선행, step 5 execute DROP 최후행).

## 11. Affected Files

| 파일 | 변경 |
|---|---|
| `bin-agent-manager models/agent/agent.go` | `StatusRinging` 제거 + 예약 3필드 |
| `bin-agent-manager agenthandler/event.go` | `EventGroupcallCreated` 제거. progressing→busy 유지 |
| `bin-agent-manager subscribehandler/main.go` | `groupcall_created` 구독 제거(golden/syncOps 갱신) |
| `bin-agent-manager dbhandler/agent.go` | AgentSetStatus blind→CAS, 예약/해제/좀비 CAS |
| `bin-agent-manager listenhandler/` | `/reserve`, `/reserve_release` |
| `bin-agent-manager (신규)` | 좀비 sweep |
| `bin-agent-manager docs (domain/operations/architecture/README/CLAUDE)` | ringing 제거/개정 |
| `bin-openapi-manager openapi/openapi.yaml` | `AgentManagerAgentStatus` ringing(283)·varname(290) 제거. call ringing·queuecall `kicking`(6494/6502) 불변 |
| `bin-openapi-manager gens/models/gen.go` | agent enum(636)+`Valid()`(652) 재생성(수기 금지) |
| `bin-api-manager gens/openapi_server/gen.go` | 재생성물(수기 금지) |
| `bin-api-manager docsdev/source/{agent_struct_agent,agent_overview,queue_overview}.rst` | agent ringing 행/전이 서술 제거·개정 |
| `bin-queue-manager models/queuecall/queuecall.go` | `GroupcallID` 필드 add. (**StatusKicking 유지**, §5.0) |
| **`bin-queue-manager models/queue/queue.go`** | **[iter-12] `Execute` 필드(`db:"execute"`, :28)·`Execute` 타입(:58)·`ExecuteRun`/`ExecuteStop` 상수(:62-63) 제거. step 5 DROP 안전 전제(§10)** |
| **`bin-queue-manager models/queue/field.go`, `models/queue/filters.go`** | **[iter-12] `FieldExecute`(field.go:19), `FieldStruct.Execute`(filters.go:12, `filter:"execute"`) 제거. filters.go는 비스케줄러 코드라 `Execute` 타입 제거 시 컴파일 파손(누락 주의)** |
| **`bin-queue-manager queuehandler/create.go`, `queuehandler/queue.go`** | **[iter-12] execute write 소비처(create.go:97, queue.go:35-37 FieldExecute) 제거** |
| **`bin-queue-manager dbhandler/queuecall.go`** | **status 전이 setter 6종 blind→CAS+affected: `QueuecallSetStatusConnecting/Service/Done/Abandoned` + `QueuecallSetStatusWaitingIfInitiating`·`...IfConnecting`(기존 Waiting 대체). (QueuecallSetStatusKicking 유지)**. SetStatus*(groupcall_id), ListWaitingFIFO(ASC), connecting-stale 조회 |
| **`bin-queue-manager queuecallhandler/db.go`** | **`UpdateStatus{Waiting,Connecting,Service,Done,Abandoned}` + 신규 `UpdateStatusWaitingRollback` 전 부수효과 `affected>0` 게이팅 + 패배 시 (res,nil) no-op(§5.0)** |
| **`bin-queue-manager queuecallhandler/main.go`** | **인터페이스: `UpdateStatusWaitingRollback` 추가, setter 시그니처 `(int64,error)`** |
| **`bin-queue-manager queuecallhandler/kick.go`** | **Kick·kickForce flow-stop 후 status 재조회 → fresh 분기(§5.1)** |
| `bin-queue-manager queuecallhandler/execute.go` | groupcalls[0].ID 영속화+len 가드, 진입 CAS, dial/CAS 실패 시 hangup |
| `bin-queue-manager queuecallhandler/event.go` | status 전이는 db.go 게이팅으로 자동 안전, join CAS 패배 no-op |
| `bin-queue-manager queuehandler/{execute,db}.go` | 스케줄러 루프/UpdateExecute 제거→match() |
| `bin-queue-manager subscribehandler/` | agent status_updated 구독+available 필터 |
| `bin-queue-manager queuehandler/(신규)` | (나) 백스톱 |
| `bin-queue-manager cachehandler/` | (가) 신선도, (나) 리스 |
| **`bin-queue-manager {queuecallhandler,dbhandler}/mock_main.go`** | **setter 시그니처 `(int64,error)`, `UpdateStatusWaitingRollback` mock 재생성** |
| **`bin-queue-manager {queuecallhandler/db_test,queuecallhandler/event_test,dbhandler/queuecall_test,listenhandler/v1_queuecalls_test}.go`** | **`.Return(nil)`→`.Return(affected,nil)`. UpdateStatus* 테스트에 CAS 패배(affected=0) 케이스** |
| `bin-common-handler requesthandler/agent_agents.go` | `AgentV1AgentReserve`, `AgentV1AgentReserveRelease` |
| **`bin-common-handler requesthandler/queue_queue.go`(+`main.go` 인터페이스+mock)** | **[iter-12/13] `QueueV1QueueExecuteRun`/`QueueV1QueueUpdateExecute` 및 인터페이스 정의·mock 제거(스케줄러 제거 완결). self-RPC만 소비, 타 서비스 0건** |
| `bin-dbscheme-manager` | **2회 배포(§10)**: (1차 최선행) (queue_id,status) 인덱스, 예약 3컬럼, queuecall groupcall_id — additive. (2차 최후행, step 5) execute drop — 파괴적, 스케줄러 제거 후 |

## 12. Open Questions

| # | 질문 | 권고 |
|---|---|---|
| Q1 | 예약 RPC 소관 | agent-manager 확정 |
| Q2 | Tres | ≥ Tconn(120s) |
| Q3 | Tconn | 최대 dial 타임아웃+마진 |
| Q4 | 예약 해제 시점 | service 확정 승자 |
| Q5 | available 복귀 | 제품/클라이언트 정책 |
| Q6 | 좀비 sweep 역조회 | 불필요(시간 단독) |
| Q7 | dial-then-CAS 순서 | 확정 |
| Q8 | agent ringing | enum 완전 제거(라이브 상태). call ringing·queuecall kicking 불변 |
| Q9 | 승자 게이팅 범위 | 부수효과 가진 큐콜 status 전이 6종(§5.0). 각 전이 전용 메서드↔전용 setter. **StatusKicking(죽은 코드)은 대상 아님, 유지** |
| Q10 | 롤백 핸들러 API | `UpdateStatusWaitingRollback` 신규(connecting→waiting 전용). 부수효과 메서드 내부 단일 귀속 |

## 13. Rollout / Risk

- 큐 스케줄러 제거 = 큰 blast radius. Phase 1 신중. **크로스서비스 배포 순서는 §10 참조**(dbscheme→agent RPC→queue, ringing 백필→enum 제거. 단방향, 순환 없음).
- [iter-1~5 요약] 방법 B, connecting 복구 leg hangup, Tres≥Tconn, 예약 RPC, groupcall_id 영속, ringing 제거(유령 소멸), enum 완전 제거(파괴적, 감수, call ringing 불변).
- **[iter-6~9] status 전이 6종 CAS 승자 게이팅**(부분 배선이 라이브 콜 teardown/버려진 콜 부활/유령 webhook/배정 덮어씀). kick Kick·kickForce 재조회. 롤백 전용 메서드 분리.
- **[iter-10] StatusKicking 제거 철회**(오버엔지니어링): 죽은 코드 유지, 계약 표면 미변경. 게이팅 범위는 부수효과 6종으로 완결.
- available 복귀는 제품/클라이언트 정책.

## 14. Review Log

### iter-1~5 (요약)
방법 A→B / Tres≥Tconn·connecting leg hangup·예약 RPC / groupcall_id 영속 / 유령 ringing→agent ringing 제거·available 복귀 정책화 / enum 완전 제거(B).

### iter-6 (둘 다 CHANGES): abandon 기존 부수효과(confbridge delete) 무게이팅 → 라이브 콜 teardown. `UpdateStatusAbandoned` 전체 승자 게이팅.

### iter-7 (스플릿 1A/1C): service/done에도 동일. 종단 3전이 게이팅 + kick TOCTOU(Kick). 과잉주장 철회.

### iter-8 (스플릿 적대A/표준C): forward 전이(waiting/connecting) 미배선. status 전이 5종 완결 테이블 + setter 분리 + kickForce. 적대 리뷰 오버엔지니어링 반증.

### iter-9 (둘 다 CHANGES, 핵심 CAS 양쪽 건전 확정): iter-8 setter 분리의 문서 모호성 → 롤백 전용 메서드 분리·부수효과 단일귀속·명칭 정정. (StatusKicking 제거 시도 — iter-10에서 철회)

### iter-10 (둘 다 CHANGES, 핵심 설계 양쪽 건전 재확정 — StatusKicking 오버엔지니어링) → v10
| ID | 지적 | 반영 |
|---|---|---|
| **StatusKicking 제거 = 오버엔지니어링 (양쪽)** | 저 스스로 "부수효과 없고 라이브 경합 없음"이라 인정 → 제거는 CAS 정합성에 불필요. 죽은 상수 1개 지우려 openapi enum+재생성물 2+docs RST 2 파괴적 크로스서비스 변경 유발(§11 누락으로 불완전하기까지). 대표님 오버엔지니어링 지양 원칙 충돌 | §5.0·§10·§11·§13·Q9: **StatusKicking 제거 철회, 죽은 코드 유지**. "잔존 blind setter 0" 수사 철회. ringing(라이브)/kicking(죽은 enum) 비대칭 근거 명시 |
| **Phase 1 배포 순서 미명시 (적대)** | 6 리포 크로스서비스인데 런타임 의존 순서(dbscheme→agent RPC 라이브→queue 소비, 백필→enum) 롤아웃 제약 부재 | §10에 **배포 순서 4단계 + 게이팅** 명문화. 단방향(순환 없음) |
| **핵심 설계 (양쪽)** | 6전이 게이팅·롤백 분리·단일귀속·no-op·I-CB·kick TOCTOU·배포 단방향 | 검증 통과(양쪽 건전 재확정) |

iter-10 두 리뷰어가 **핵심 설계는 건전** 재확정, 남은 건 iter-9에서 내가 추가한 **StatusKicking 제거(오버엔지니어링)**와 배포 순서 누락. StatusKicking을 죽은 코드로 되돌려(스코프 축소) 대표님 원칙에 정합, 배포 순서 명문화. iter-11 fresh 재검토.

### iter-11 (스플릿: 적대 APPROVE, 표준 CHANGES) → v10
| ID | 지적 | 반영 |
|---|---|---|
| **execute DROP 배포 순서 (표준)** | §10 step 1(dbscheme 최선행)이 additive만 열거, 파괴적 `execute` DROP 순서 누락. 구 queue-manager가 `queue` struct(`queue.go:28 db:"execute"`) SELECT 로드하므로 선DROP 시 런타임 실패(expand/contract 위반) | §10 step 5·§6·§11: **execute DROP을 별도 2차 마이그레이션, 스케줄러 제거(step 3) 배포 후 최후행**. dbscheme 2회 배포 명시 |
| **APPROVE (적대)** | 구현 착수 가능(파일별 라인 명세)·Phase 1 크기는 구조적 필연(원칙 위반 아님)·예약/Redis 리스 선제 인프라 아님(각각 유일 방어축/기존 의존 재사용)·핵심 CAS 무결함·Q1~Q10 대표님 결정 대기 없음 | 착수 준비 확정 |

iter-11 적대 리뷰어 APPROVE(구현 착수 가능·오버엔지니어링 없음·대표님 결정 대기 항목 0 확정), 표준 리뷰어가 execute DROP 배포 순서 1건. 반영으로 expand/contract 준수. iter-12 fresh 재검토.

### iter-12 (둘 다 CHANGES, 동일 결함 수렴 — execute struct 필드 제거 누락) → v10
| ID | 지적 | 반영 |
|---|---|---|
| **execute struct 필드 미제거 (양쪽 독립 수렴, 치명)** | iter-11이 execute DROP을 step 5 최후행으로 옮겼으나, contract 안전의 진짜 전제는 **struct에서 `Execute` 필드 제거**. `queueGetFromDB`(dbhandler/queue.go:135)가 `GetDBFields(&queue.Queue{})`(mapping.go:50 리플렉션)로 SELECT 컬럼 생성 → `db:"execute"` 필드가 struct에 남으면 **스케줄러 코드만 지운 새 바이너리도 `SELECT ... execute ...` 발행** → step 5 DROP이 **새 바이너리 크래시**. §11이 `models/queue/queue.go` 필드 제거를 누락(Go unused 필드 컴파일 에러 없어 방치됨) | §10 step 2/5·§11: **`models/queue/queue.go` Execute 필드·타입·상수 제거, `field.go` FieldExecute 제거, execute write 소비처·`bin-common-handler` execute RPC 제거를 step 2에 명시**. "execute drop"을 "struct 필드/코드 제거(step 2)" vs "DB 컬럼 DROP(step 5)"로 분리 |
| **핵심 설계 (양쪽)** | 6전이 CAS·롤백 분리·kick TOCTOU·StatusKicking 유지·배포 단방향 재확인 | 재제기 없음, 무회귀 확정 |

iter-12 두 리뷰어가 독립적으로 "execute DB 컬럼 DROP 안전의 전제 = struct 필드 제거"에 수렴(리플렉션 SELECT가 필드 구동이라 코드로 검증). §11에 struct/field/RPC 제거를 명시해 contract 완결. iter-13 fresh 재검토.

### iter-13 (스플릿: 적대 APPROVE, 표준 CHANGES — 동일 사실 상반 해석) → v10
| ID | 지적 | 반영 |
|---|---|---|
| **execute 제거 체인 나열 (해석 갈림)** | 표준: §11이 filters.go/listenhandler/cmd/test 미열거 → "완결 주장" 거짓 = CHANGES. 적대: 이들은 **전부 `Execute` 타입 제거 시 컴파일러 강제 노출**, silent-risk는 struct 필드뿐(iter-12 반영됨), 크로스서비스 소비 0건(self-RPC) = **APPROVE, 치명 결함 없음** | **적대 해석 채택(CPO 판단)**: 설계 문서 목적은 전수 파일 나열이 아니라 silent-risk 식별. §10 step2에 **스코프 성격 명시(silent-risk=struct 필드 1개 / 나머지=컴파일러 포착 / 크로스서비스 0건)**, "완결" 과잉표현 축소. 컴파일 파손하는 비스케줄러 코드 `filters.go:12`만 콕 집어 §11 추가(구현자 착각 방지). 나머지는 `go build` 위임 |
| **APPROVE (적대)** | execute struct 필드가 유일 non-compile-caught SELECT 구동체 정확 겨냥, 미열거 소비처 전부 컴파일러 포착, 크로스서비스 0건, iter-9~12 핵심 CAS 무회귀 | 착수 준비 확정 |
| **비차단 관찰 (적대)** | prod execute 컬럼 NOT NULL이면 step5 이전 INSERT 실패 가능하나 테스트 스키마 nullable(queues.sql:18) 시사 | Alembic 작성 시 nullable 확인 권고(비차단) |

iter-13 적대 리뷰어 APPROVE(구현 착수 막는 치명 결함 없음 확증: silent-risk 정확 겨냥+컴파일러 포착+크로스서비스 0건), 표준 리뷰어는 전수 나열 미비를 CHANGES로 봄. CPO 판단: 컴파일러가 잡는 파일 전수 나열은 문서 비대화(오버엔지니어링)이므로, 스코프 성격 규정+silent-risk(filters.go 포함) 명시로 대체. iter-14 fresh 재검토.

### iter-14 (둘 다 APPROVE ✓ — 첫 양쪽 APPROVE) → v10
| ID | 지적 | 반영 |
|---|---|---|
| **양쪽 APPROVE** | (표준) silent-risk 규정 코드상 정확(`db:"execute"` struct는 queue.go:28 단 하나, SELECT 구동=GetDBFields 리플렉션 2곳뿐, raw SQL execute 경로 0건). filters.go 반영 정확. 구현 착수 가능, 치명 결함 없음. (적대) step-2 필드 제거가 step-5 DROP 크래시를 정확 차단, 크로스서비스 0건, 6전이 게이팅 정당. 배포/데이터손실/런타임 크래시급 결함 미발견 | 착수 준비 완료 |
| **표기 오류 (표준, 비차단)** | §11 `FilterStruct.Execute` → 실제 타입명 `FieldStruct`(filters.go:7) | `FieldStruct.Execute`로 정정 |
| **경미 관찰 (양쪽, 비차단)** | 문서 경로 `pkg/` 접두어 축약(일관됨), Alembic 작성 시 prod execute 컬럼 nullable 확인(queues.sql:18 nullable 시사) | 구현 시 처리(비차단) |

iter-14 **두 리뷰어 양쪽 APPROVE**. 핵심 설계는 iter-9~13 다섯 라운드 양쪽 건전에 이어 iter-14 양쪽 APPROVE로 완결. 표기 오류 1건만 정정(FieldStruct). 정책상 2연속 양쪽 APPROVE를 위해 iter-15 확인 라운드(표기 정정본) 1회 예정. 남은 비차단 관찰은 구현 중 `go build`/Alembic이 처리.

### iter-15 (스플릿: 적대 APPROVE, 표준 CHANGES — 표기 정정 누락) → v10
| ID | 지적 | 반영 |
|---|---|---|
| **표기 정정 불완전 (표준)** | iter-14가 `FilterStruct`→`FieldStruct` 정정을 §11(l.267)만 하고 **§10 step2(l.232)의 동일 참조 누락**. 같은 filters.go:12를 두 곳이 가리키는데 한 곳만 고침. 실제 타입명 `FieldStruct`(filters.go:7) | §10 step2도 `FieldStruct.Execute`로 정정. 문서 내 지시 참조 2곳 통일(Review Log 이력 서술만 과거 기록으로 보존) |
| **APPROVE (적대)** | execute struct 필드→step5 DROP 크래시 전제 실증(SELECT(fields...) 리플렉션, 별표 아님), 크로스서비스 0건, prod nullable, StatusKicking 유지, 6 setter blind 실재, ringing 백필 순서 전부 코드 대조 일치. 배포/데이터손실/런타임 크래시급 신규 결함 0 | 착수 가능 재확정 |

iter-15 적대 리뷰어 APPROVE(코드 대조 6항목 전부 일치, 신규 치명 결함 0), 표준 리뷰어가 표기 정정 누락 1건(제 부주의 — §11만 고치고 §10 누락). 정정으로 문서 지시 참조 통일. **연속 APPROVE 카운트: iter-14의 1 → iter-15 스플릿으로 0 리셋.** iter-16 fresh 재검토.

### iter-16 (둘 다 APPROVE ✓ — 연속 1/2) → v10
문서 변경 없이 iter-15 표기 정정본을 fresh 재검토. 양쪽 리뷰어 코드 라인 단위 대조:
- (표준) FieldStruct 표기가 §10 step2·§11 두 곳 모두 코드(filters.go:7 `type FieldStruct`)와 일치, 지시 참조에 잔존 FilterStruct 0건(Review Log 이력 서술만 남음). 무회귀, 구현 착수 가능.
- (적대) step2 struct 필드 제거→step5 DROP 크래시 방어 실증(GetDBFields 리플렉션 필드 구동 SELECT, 별표 아님), 6 blind setter 실재(324/338/372/406/453), 부수효과 소비처 라인 전부 일치, 크로스서비스 0건. 배포/데이터손실/런타임 크래시급 신규 결함 0.
**연속 APPROVE 카운트: 1/2.** iter-17 확인 라운드(문서 무변경) 1회로 2연속 달성 목표.

### iter-17 (둘 다 APPROVE ✓ — 연속 2/2 달성, 설계 확정) → v10 APPROVED
문서 무변경 최종 확인 라운드. 양쪽 리뷰어 독립 코드 대조 후 APPROVE:
- (표준) 모든 라인 앵커(queue.go:28/58/62-63, filters.go:7/12, field.go:19, GetDBFields queue.go:135/193, db.go 6전이 부수효과 소비처, kick.go TOCTOU, subscribehandler 4종 구독)가 코드와 일치. 구현 착수 가능. 비차단: iter-16 로그의 "6 blind setter" 앵커 5개 표기는 Waiting 분리로 6종 되는 구조라 실질 오류 아님.
- (적대) execute 크래시 방어·크로스서비스 0건·6 blind setter·confbridge delete 부수효과·agent ringing 라이브 발행 전부 코드 실측 일치. 배포 순서 단방향 expand/contract 준수. 배포/데이터손실/런타임 크래시급 신규 결함 0.

**설계 확정: iter-16·17 2연속 양쪽 APPROVE 달성(정책 요건 충족). v10 APPROVED.** 핵심 설계는 iter-9~17 아홉 라운드 건전. 구현 착수는 대표님 승인 후.

## 리뷰 여정 총괄 (17 라운드)
- iter-1~6: 동시성 근본 결함(방법 A→B 전환, 라이브 콜 teardown 등 correctness 버그).
- iter-7~8: 결함 클래스 완결(게이팅을 종단→forward 전이 전수 확장). 적대 리뷰 오버엔지니어링 반증.
- iter-9: 문서 명료화(롤백 메서드 분리, 부수효과 단일귀속).
- iter-10: 스코프 축소(StatusKicking 오버엔지니어링 철회).
- iter-11~13: 배포 안전(expand/contract, execute 필드 제거).
- iter-14~17: 표기 정정 + 최종 확인. iter-14/16/17 양쪽 APPROVE, iter-16·17 2연속으로 확정.
방지한 프로덕션 장애: 라이브 콜 teardown, 버려진 콜 부활, 유령 webhook, 배포 크래시(execute DROP).
