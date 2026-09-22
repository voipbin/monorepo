# Queue-manager Event-driven Routing Redesign

Status: Draft → in review (v9, iter-1~8 반영)

## 1. Problem Statement

`bin-queue-manager`의 라우팅 루프(`pkg/queuehandler/execute.go`)는 순수 폴링으로 동작하며, 다음의 구조적 낭비와 정확성 결함을 가진다. 모두 실제 코드로 확인되었다.

1. **유휴 폴링 낭비 (핵심)**: 대기콜이 있고 available 에이전트가 0명이어도 큐당 1초마다 `QueuecallList` + `AgentList`가 무한 반복(`execute.go:70-73`).
2. **쿼리 순서 비효율**: 대기콜 먼저 뽑고(`execute.go:43`) 에이전트 나중 확인(`execute.go:63`) → 에이전트 없으면 대기콜 버림.
3. **대기콜 선택 LIFO (버그)**: `QueuecallList`가 `tm_create DESC` + `token=현재시각`/`size=1` → 최신 콜 우선. FIFO 기대와 반대.
4. **`connecting` 전용 타임아웃 부재**: `connecting`에 자체 타임아웃/복구 없음 → 에이전트 무응답+고객 생존 시 무기한 정체.
5. **대기콜 이중 소스**: status 컬럼(A, 라우팅) vs `wait_queuecall_ids` 배열(B, webhook 표시). B는 과대계상(connecting/service 미제거).

### 진단의 근원: 비대칭
새 대기콜 진입은 이미 이벤트 드리븐(`AddWaitQueueCallID`→`UpdateExecute(Run)`)인데, 에이전트 available만 이벤트가 없어 폴링. agent-manager는 이미 `agent_status_updated`를 global topic exchange로 발행 중(§5.3), queue-manager가 구독만 안 함.

## 5.0 Concurrency Model (설계의 척추)

**[iter-2~7 수렴]** 현행 queue-manager는 큐당 **단일 in-flight** self-paced 폴링 루프라 동시성이 없었고, 그래서 모든 상태 전이가 blind read-then-write / blind UPDATE여도 안전했다. 이벤트 드리븐 전환은 이벤트당 고루틴(`subscribehandler/main.go:139` `go h.processEvent`)으로 **동시성을 새로 도입**한다. 따라서 여태 잠자던 race가 전부 실재화된다. v8은 이를 3개 불변식으로 봉쇄한다. **개별 전이는 전부 여기서 파생되며, 예외를 두지 않는다.**

### 상태공간 확정 (iter-5 A-5)
queuecall status 상태공간은 **{initiating, waiting, connecting, service, done, abandoned}**로 확정한다. `StatusKicking`(`models/queuecall/queuecall.go:77`)과 `QueuecallSetStatusKicking`(dbhandler)은 **라이브 호출자가 없는 死상태**(db/mock/test만 참조)이므로 본 설계 상태공간에서 제외하고, 별도 cleanup 이슈로 제거 대상 등재한다.

### 불변식 I1 — 경합하는 모든 상태 전이는 조건부 CAS (예외 없음)
읽고-쓰기 사이 다른 고루틴이 끼어들 수 있는 **모든** 전이를 `UPDATE ... WHERE <expected-precondition>`로 원자화하고 affected 행 수를 반환한다. **부수효과(webhook 발행, 배열/큐 갱신, 타임아웃 예약, confbridge/flow 정리)는 CAS가 affected=1(승리)일 때만 실행한다. 패배(affected=0) 경로는 순수 no-op이다.**

대상 전이와 술어(전이 그래프 전수 — iter-6·7 검증):
| 전이 | 함수(신규) | CAS 술어 | Phase |
|---|---|---|---|
| queue: Stop→Run | `SetExecuteRunIfStop` | `WHERE execute='stop'` | 1 |
| queuecall: initiating→waiting | `SetStatusWaitingIfInitiating` | `WHERE status='initiating'` | 1 |
| queuecall: waiting→connecting (진입) | `SetStatusConnectingIfWaiting` | `WHERE status='waiting'` | 1 |
| queuecall: connecting→waiting (dial 실패 롤백) | `SetStatusWaitingIfConnecting` | `WHERE status='connecting'` | 1 |
| queuecall: connecting→service | `SetStatusServiceIfConnecting` | `WHERE status='connecting'` | 2 |
| queuecall: service→done | `SetStatusDoneIfService` | `WHERE status='service'` | 2 |
| queuecall: →abandoned (initiating/waiting/connecting 공용) | `SetStatusAbandonedIfActive` | `WHERE tm_end IS NULL AND status != 'service'` | 2 |

**진입 간선 전수 확인(iter-6·7)**: initiating(Create, 전이 아님), waiting(initiating→, connecting→롤백), connecting(waiting→ 유일 write), service(connecting→), done(service→), abandoned(공유 sink). 6개 상태의 모든 진입 간선 + queue Stop→Run = **7전이 전수 CAS**. 빠진 간선 없음.

**[iter-4 결함1 / iter-5~7 확인] abandon은 공유 sink다.** `UpdateStatusAbandoned`(`db.go:240`)는 waiting abandon(hangup `event.go:34`, wait-timeout `Kick`→`kick.go:45`, customer-delete `kickForce`)과 connecting abandon(신규 타임아웃)을 **공유**한다. 술어를 `status='connecting'`으로 하드코딩하면 waiting abandon이 전부 no-op이 되는 P0 회귀다. 술어는 **"아직 종료 안 됨 AND service 아님"**(`tm_end IS NULL AND status != 'service'`).
- initiating/waiting hangup: tm_end null, status≠service → **통과**.
- connecting hangup/타임아웃: status=connecting → **통과**.
- 이미 abandoned/done: tm_end 세팅(`db.go:376,410`) → **no-op**(이중 abandon 차단).
- **service 상태: status=service → no-op**(라이브 통화 무손상). **[iter-7 확인]** service 콜의 늦은 고객 hangup은 `EventCallCallHangup` 가드(`event.go:28` `status==StatusService`시 return)로 abandon 미호출 → `confbridge_leaved`→**done** 경로. 즉 **service 종료는 done만**, service→abandoned 간선은 존재하지 않는다.

### 불변식 I2 — sleep 후 재확인 (lost-wakeup 봉쇄)
**[iter-4 lost-wakeup / iter-5 초점1: happens-before 코드 증명]** "대기콜 0건" 읽기와 Stop 쓰기 사이 신규 콜 도착 시 그 콜의 wake CAS가 아직 Run이라 no-op → Stop 커밋 → 큐 고착 가능. 봉쇄:
- **모든 sleep 경로**는 `UpdateExecute(Stop)` **커밋 직후 대기콜을 1회 재조회**하고, >0이면 즉시 `SetExecuteRunIfStop` CAS + 트리거로 자가 치유.
- 새 대기콜 진입은 status=waiting **동기 커밋**(`db.go:343`) 뒤 `go` Run-CAS(`db.go:356`).
- **happens-before 증명**: enqueue의 status 커밋 Ts < Run-CAS Tr. "Run-CAS가 run 관측(무트리거)"이면 소유자 Stop 커밋 Tstop > Tr > Ts, 재확인 Trecheck > Tstop > Ts이므로 재확인이 그 waiting 행을 **반드시** 본다. 재확인 '없음' 직후 도착 콜은 Stop이 먼저 커밋돼 Run-CAS가 'stop' 관측 → 자가 트리거. → **잔여 창 없음.**
- **[iter-6 결함2] Get 오류도 sleep 경로다**: 큐 Get 오류를 **`ErrNotFound`(큐 삭제 확정)=순수 return(DB write 없음, 재개 불필요)** vs **그 외 일시 오류=`ExecuteRun(delay)` 재스케줄(자가치유)**로 분기(§5.4). 형제 오류 경로(agent-list/qc-list 오류)와 대칭. **[iter-7 관찰: 폭주 반증]** 일시오류 분기는 오류당 `ExecuteRun(1000ms)` 1회만 예약(1:1 증폭 없음), 영구 DB 장애 시 큐당 초당 1회 bounded, flip CAS가 중복 wake 흡수.
- 60초 안전망(§5.5)은 이 재확인과 별개로 belt-and-suspenders.

### 불변식 I3 — CAS 승자만 부수효과, 정리 순서는 status-확정 우선
CAS 승리 후에만 부수효과를 실행하되, **status를 최종값으로 먼저 확정(CAS 커밋)한 뒤** 정리 부수효과를 수행한다. 진 쪽은 status 가드에서 no-op이라 이중 정리/이중 webhook/통화 절단이 원천 차단된다.
- **[iter-4 minor / iter-5 관찰2]** CAS 커밋~정리 사이 크래시, 또는 hangup-during-dial 고아 confbridge/variables 창은 **현행에도 존재하는 비회귀 결함**(join 실패로 대부분 자가 소멸)이며 본 설계 범위 밖(별도 reaping 이슈). §16·Q9 명시.

## 2. Scope

Phase 1은 **유휴 폴링 제거 + initiating/진입/flip 동시성 안전화 + FIFO head-of-line 방지**, Phase 2는 **connecting 타임아웃 + 탈출 전이(service/done/abandon) CAS + 표시 필드 파생화**. 같은 브랜치 커밋 분리, diff 크기에 따라 별도 PR 가능(Q1).

### In scope
**Phase 1**
- `agent_status_updated` 구독. available 전이 → customer의 잠든 큐 `UpdateExecute(Run)` wake(§5.2).
- 에이전트 0명 → sleep(§5.4) + sleep 후 재확인(I2) + 60초 안전망. Get 오류 분기.
- 쿼리 순서 반전. FIFO 정렬(라우팅 전용, §5.6). **사이클 내 다음 후보 진행(poison head-of-line 방지, §5.4)**. 헬스체크 5→30초.
- **execute flip 원자 CAS**(§5.7) + **flip 시 `queue_updated` 억제**.
- **initiating→waiting CAS**(§5.4b).
- **waiting→connecting 진입 CAS + dial 후 blind write 철거 + 성공前 실패 시 connecting→waiting 롤백**(§5.4a).

**Phase 2**
- `connecting` 타임아웃 → CAS abandon(§5.8).
- connecting/service **탈출** 전이 CAS 대칭화: service/done/abandon(§5.8).
- 표시 필드 파생화(옵션 B, §5.9).

### Out of scope (Phase 3)
- **에이전트 원자적 예약**(cross-call race). Phase 1 진입 CAS로 같은 큐 중복 배정 차단. 실측 신호 도달 시.
- **dial 실패 재라우팅(retry) + poison 콜 능동 제거**: Phase 1은 dial 실패 시 waiting 복귀 + 사이클 내 다음 후보 진행(head-of-line 해소). 지속 실패 poison 콜의 능동 abandon(실패 카운트 상한 등)은 실측 신호 도달 시 Phase 3(§16).
- 큐 단독 삭제 orphan, CAS-크래시/hangup-during-dial 고아 reaping, `StatusKicking` 제거, 라우팅 전략 확장, 경량 이벤트/tag 사전필터.

## 3. Decisions
1. 이벤트 + 저빈도 안전망(60초). 대기콜 0건이면 완전 정지.
2. wake 단위 customer. tag 매칭은 Execute. wake 게이트 배열 B 미의존.
3. available 전이만 wake.
4. **[C1] connecting no-answer는 abandon(재시도 없음)**, CAS. waiting 복귀는 wait 타임아웃 재예약 부재로 무한 루프+우회 유발. 재라우팅은 Phase 3.
5. **[I1] 모든 경합 전이 조건부 CAS(예외 없음, done 진입 포함), 부수효과는 승자만.**
6. **[I2] sleep 후 재확인으로 lost-wakeup 봉쇄. Get 오류도 sleep 경로로 분기.**
7. **[iter-2 #5] 표시 필드 파생(옵션 B).**
8. **[iter-3 #2] execute flip은 `queue_updated` 미발행.**
9. **[iter-5 A-2] dial/flow 성공前 실패 롤백은 waiting 복귀(abandon 아님)** — 현행 재시도 동작 보존. dial **성공後** forwardAction 실패는 롤백 안 함(connecting 정체는 고객 hangup/Phase 2 위임).
10. **[iter-7 결함] FIFO head-of-line 방지는 사이클 내 다음 후보 진행으로 해결** — poison 선두 콜이 뒤 정상 콜을 막지 않게. 스키마 추가 없이, wait_timeout 값과 무관하게 회귀 차단(§5.4).

## 4. Architecture: After
```
새 대기콜 → Create(initiating) → PushStack → SetStatusWaitingIfInitiating(CAS) → AddWaitQueueCallID → UpdateExecute(Run, CAS)
에이전트 available → agent_status_updated(available만) → customer 잠든 큐 UpdateExecute(Run, CAS)  ← NEW
                                      ↓
   ┌───────────────── Execute() 루프 ──────────────────┐
   │ q,err=Get(id):                                      │
   │   ErrNotFound → return(DB write 없음, 재개 불필요)   │  ← 큐 삭제 확정
   │   기타 오류   → ExecuteRun(delay); return           │  ← 일시 오류 자가치유(I2)
   │ if Execute==Stop → return                           │
   │ 1. 에이전트(available) FIRST                        │
   │ 2. 에이전트 0명:                                     │
   │      UpdateExecute(Stop) + [I2] 재확인/자가치유       │
   │      대기콜>0였으면 60초 안전망; return              │
   │ 3. 에이전트 있음 → 대기콜(waiting, ASC, size=N)      │  FIFO 전용
   │ 4. 0건 → UpdateExecute(Stop) + [I2] 재확인; return   │
   │ 5. for qc in qcs:  (FIFO 순회, head-of-line 방지)    │
   │      SetStatusConnectingIfWaiting(CAS)               │  진입 CAS(유일 connecting write)
   │      승리시 flow-create+dial:                        │
   │        성공 → assigned; break                        │
   │        성공前 실패 → SetStatusWaitingIfConnecting(롤백); 다음 후보 │
   │      패배시 → 다음 후보                              │
   │ 6. assigned면 ExecuteRun(100), 아니면 ExecuteRun(1000)│
   └────────────────────────────────────────────────────┘
탈출(P2): service=WHERE status='connecting', done=WHERE status='service', abandon=WHERE tm_end IS NULL AND status!='service'
```

## 5. Detailed Design

### 5.1 이벤트 구독 (Phase 1)
`subscribehandler/main.go`: topicPatterns에 `agent-manager.agent.*.status_updated`(5개), golden 4→5, publisher 상수, switch case → `processEventAMAgentStatusUpdated`.
`subscribehandler/agentmanager.go`(신규): `agent.Agent` unmarshal → `Status!=available`이면 discard 카운터+return, 아니면 available 카운터+`EventAgentAvailable(customerID)`.
**[C4] 볼륨**: `*.status_updated`는 status 필터 불가 → 전 고객 수신 후 available만. 건당 unmarshal+비교+대부분 폐기(DB 미접근). 카운터 관찰, 임계 초과 시 경량 이벤트(§2).

### 5.2 available wake (Phase 1)
`queuehandler/event.go`(신규): customer 큐 List → `q.Execute==Stop`인 큐마다 `UpdateExecute(ctx, q.ID, ExecuteRun)`. bare `QueueV1QueueExecuteRun`은 Stop 가드 no-op이라 금지. **[I1]** flip CAS라 동시 available 이벤트가 같은 큐 다중 wake해도 1회만 승리. 팬아웃 customer 국한, `promQueueWakeTriggerTotal`.

### 5.3 [검증] 이벤트 도달
`agenthandler/db.go:391` `PublishWebhookEvent(CustomerID, EventTypeAgentStatusUpdated, agent.Agent)`, cmd `WithGlobalTopicPublish()` → `bin-manager.event`, key `agent-manager.agent.<id>.status_updated`. 구독 패턴 매칭. 신규 인프라 불필요.

### 5.4 Execute 루프 (Phase 1) [iter-6 Get 분기, iter-7 head-of-line]
```
Execute(id):
  q,err=Get(id)
  if err:
     if errors.Is(err, ErrNotFound): return          // 큐 삭제 확정, DB write 없이 종료
     ExecuteRun(id, defaultExecuteDelay); return      // 일시 오류: 재스케줄 자가치유(I2)
  if q.Execute==Stop → return
  agents=GetAgents(q.ID, StatusAvailable)
  if err → ExecuteRun(id, defaultExecuteDelay); return
  if len(agents)==0:
     had = QueuecallListWaitingFIFO(q.ID,1) 유무
     UpdateExecute(id, Stop)
     if QueuecallListWaitingFIFO(q.ID,1) 있음:                    // [I2] sleep 후 재확인
         if SetExecuteRunIfStop(id) affected: ExecuteRun(id,100); return
     if had: QueueV1QueueExecuteWake(id, defaultExecuteSafetyDelay)
     return
  qcs = QueuecallListWaitingFIFO(q.ID, defaultExecuteBatchSize)   // FIFO 전용, size=N(§5.6)
  if err → ExecuteRun(id, defaultExecuteDelay); return
  if len(qcs)==0:
     UpdateExecute(id, Stop)
     if QueuecallListWaitingFIFO(q.ID,1) 있음:                    // [I2] 재확인
         if SetExecuteRunIfStop(id) affected: ExecuteRun(id,100)
     return
  // [iter-7] FIFO 순회 — poison 선두 콜이 뒤 정상 콜을 막지 않도록 사이클 내 다음 후보 진행
  assigned = false
  for qc in qcs:
     targetAgent,ok = pick(agents, q.RoutingMethod)
     if !ok: break                                                // available 소진(agents 스냅샷 불변이라 사실상 도달 불가; pick 실패 방어)
     res = QueuecallExecute(qc.ID, targetAgent.ID)                // 진입 CAS + flow/dial(§5.4a)
     if res == assigned_ok: assigned = true; break                // 첫 성공에서 배정 완료(페이싱 유지)
     // 진입 CAS 패배(이미 abandoned 등) 또는 dial 성공前 실패(waiting 롤백됨) → 다음 후보
  if assigned: ExecuteRun(id, 100)
  else:        ExecuteRun(id, defaultExecuteDelay)                // 모든 후보 실패/패배 → 1초 재시도
```
- **head-of-line 방지(iter-7 결함 해소)**: poison 선두 콜(dial 지속 실패)이 있어도 순회가 다음 FIFO 후보로 진행해 정상 콜을 같은 사이클에 배정한다. poison 콜 자체는 다음 사이클에 다시 시도되나(무한 재시도) **뒤 콜을 막지 않는다**. `wait_timeout=0` 큐에서도 상한(wait-timeout)에 의존하지 않고 기아를 차단. LIFO 현행 대비 회귀 없음.
- **페이싱 유지**: 첫 성공에서 break해 사이클당 1건 배정(기존 self-pacing 100ms 보존). available 다수라도 다음 available 이벤트/100ms 재호출로 순차 배정.
- **에이전트 중복 배정 없음**: 진입 CAS는 queuecall 상태만 원자화(에이전트 예약은 Phase 3)이나, break 전 배정은 1건이라 같은 사이클 내 동일 에이전트 이중 배정 없음. 순회 중 dial 실패한 콜은 에이전트를 소비하지 않음(진입 CAS 롤백).
- **[iter-7 관찰: busy-loop 반증]** 모든 후보 실패 시 `ExecuteRun(defaultExecuteDelay=1000ms)`라 100ms 폭주 아님. 순회는 size=N bounded.

### 5.4a QueuecallExecute 진입 CAS + blind write 철거 + 성공前 실패 롤백 (Phase 1) [iter-4 결함2, iter-5 A-1·2·4, iter-6]
현행 순서 flow-create(`:35`)→콜 생성(`:48`)→`UpdateStatusConnecting`(`:59`, blind, `db.go:195` connecting webhook). blind write가 waiting 단계 hangup 뒤 resurrection+고아 콜 유발. v8:
1. **진입 CAS 선행**: `SetStatusConnectingIfWaiting(id)`를 flow-create/dial보다 **먼저**. **status→connecting 유일 write**.
2. **[iter-5 A-1] dial 후 blind write 철거**: `UpdateStatusConnecting`(execute.go:59) 재기록 **제거**. connecting webhook은 진입 CAS 승자 경로에서만 발행.
3. **CAS 승리 시에만** flow-create + 콜 생성.
4. **[iter-5 A-2·A-4] 성공前 실패 롤백 = waiting 복귀**: flow-create 또는 dial 성공 이전 모든 실패에서 `SetStatusWaitingIfConnecting(id)` CAS로 되돌린다. 고객 드롭 없이 다음 사이클/후보 재시도. wait-timeout(`db.go:140`) 생존이 (설정 시) 상한. hangup-abandon이 먼저면 no-op. 반환값은 "미배정"이라 Execute 순회가 다음 후보로 진행(§5.4).
5. **[iter-6 B-3] dial 성공後 실패는 롤백 안 함**: dial 성공 후 `forwardAction`(execute.go:66) 실패 시 이미 에이전트 콜이 뜬 상태라 waiting 롤백하면 고아 에이전트 콜+waiting 이중. status를 connecting에 두고 복구를 **고객 hangup(→abandon) 또는 Phase 2 connecting-timeout에 위임**(현행과 동일한 비회귀 stuck, Phase 1 단독 구간 한정). 반환값은 "배정"이라 순회 종료.
6. **CAS 패배** 시 flow-create/dial 스킵 + "미배정" 반환(고아 콜 없음, 순회 다음 후보).
- 진입 CAS는 같은 큐 중복 배정도 차단(하나만 승리). cross-call은 Phase 3.

### 5.4b enqueue initiating→waiting CAS (Phase 1) [iter-5 A-3]
현행 `UpdateStatusWaiting`(`db.go:335`)은 blind. initiating 단계 hangup이 abandon 먼저 승리 뒤 blind write가 waiting 덮어써 resurrection → 죽은 고객 레그로 무용 dial. 봉쇄: `SetStatusWaitingIfInitiating(id)` CAS(`WHERE status='initiating'`). abandon이 이겼으면 no-op. **[iter-5 초점3 / iter-7 확인]** initiating→abandon은 고객 콜/activeflow 자체가 소멸하는 hangup·customer-delete로만 발생 → PushStack된 액션도 고객 레그와 함께 죽어 status-액션 불일치 창 없음. **[I1 부수효과 게이팅]** CAS 패배(abandon 승리) 시 후속 `AddWaitQueueCallID`+`UpdateExecute(Run)`+waiting webhook은 승자 전용 부수효과라 전부 no-op(abandoned 콜이 wait 배열/라우팅에 유입되지 않음).

### 5.5 execute_wake 안전망 (Phase 1)
listenhandler `POST /v1/queues/{id}/execute_wake` → `Wake(id)`=`UpdateExecute(Run)`. self-RPC `QueueV1QueueExecuteWake(id, delay)`(x-delayed-message 재사용). I2 재확인과 별개, 이벤트/RPC 유실 대비. 대기콜 0건이면 예약 없음.

### 5.6 대기콜 정렬 (Phase 1) [C5, iter-7]
dbhandler `QueuecallListWaitingFIFO(queueID, size)`(`ORDER BY tm_create ASC`, `status=waiting`, `queue_id=?`). 기존 `QueuecallList`(DESC, 외부 GET) 불변. Execute는 직접 호출. size=N(`defaultExecuteBatchSize`, 예 10)로 §5.4 순회의 head-of-line 방지 후보 풀 제공. 재확인용 단건 조회는 size=1.

### 5.7 execute-flip CAS + webhook 억제 (Phase 1) [iter-3 #2·#3]
- `SetExecuteRunIfStop(id) (affected bool)` = `UPDATE queues SET execute='run', tm_update=? WHERE id=? AND execute='stop'`. affected=true일 때만 `ExecuteRun(100)` 트리거.
- **webhook 억제**: flip `PublishEvent(EventTypeQueueUpdated)`(`db.go:226`) 중 execute는 `WebhookMessage` 필드에 **없다**(코드 확인) → payload-동일 중복. execute 전용 경로 `PublishEvent` 제거. 설정 변경 발행 유지.
- **[iter-4 결함4]** `AddWaitQueueCallID` 226+254 이중 발행 → 226만 제거, 254 유지(2→1).

### 5.8 connecting/service 탈출 대칭 CAS (Phase 2) [iter-4 결함1, iter-6 수렴]
탈출 전이를 I1로 원자화. **부수효과는 승자만(I3).**
- (1) service: `EventCallConfbridgeJoined`→`UpdateStatusService`가 `SetStatusServiceIfConnecting` 승리 시에만 status=service + `queuecall_serviced` + `AddServiceQueuecallID` + service-timeout 예약. **[iter-4 결함3]** 현행 무조건(`db.go:211-234`)을 CAS 게이팅.
- (2) **[iter-6 수렴] done: `SetStatusDoneIfService`** — service→done writer 2개(`EventCallConfbridgeLeaved` event.go:90 가드 `TMEnd!=nil`, `kickForce` kick.go:107 status==service)가 `go processEvent` 동시 디스패치 → 현행 blind `QueuecallSetStatusDone`(`db.go:298`)로 이중 `queuecall_done`+이중 `RemoveQueuecallID`+이중 `ConfbridgeDelete`+이중 `deleteVariables` 가능. `SetStatusDoneIfService(id)` CAS(`WHERE status='service'`)로 두 writer 게이팅, 부수효과 승자만. **[iter-7 확인]** `TimeoutService→Kick`은 status==service시 flow-stop만 하고 return(`kick.go:38-42`) — done writer 아님. 실제 done writer는 위 2개뿐이라 CAS 직렬화 충분.
- (3) timeout abandon: `TimeoutConnecting`이 `SetStatusAbandonedIfActive` 승리 시에만 정리. **Kick 재사용 금지**(`kick.go:34` 무조건 flow-stop). **[iter-5 초점4]** 예약~발화 사이 service/done 전이 시 no-op이라 stale 발화 안전.
- (4) hangup abandon: `EventCallCallHangup`(initiating/waiting/connecting 공용, service는 가드로 제외)도 `SetStatusAbandonedIfActive` 관통. wait-timeout(`TimeoutWait`→`Kick`), customer-delete(`kickForce`)도 동일 sink.
- 상호배타: 같은 connecting 행에서 service CAS(`status='connecting'`)와 abandon CAS(`status!='service'`)가 정확히 하나 승리; 같은 service 행에서 done CAS(`status='service'`)만 가능하고 abandon CAS는 술어상 no-op(service 종료=done only).

### 5.9 표시 필드 파생화 (Phase 2) [iter-2 #5]
`wait_queuecall_ids`/`service_queuecall_ids` 고객 노출 필드라 제거 불가 → **필드 유지 + status on-read 파생**(옵션 B). `ConvertWebhookMessage`가 status=waiting/service 큐콜 id 조회.
- **[iter-3 #5]** flip이 `queue_updated` 미발행이라 파생 조회는 고객 GET/설정 변경 시에만. `queue_queuecalls(queue_id, status)` 인덱스.
- 배열 컬럼: (a) write/read 중단 → (b) 후속 마이그레이션 drop. 2단계.

## 6. Domain Model
- 상태공간 {initiating, waiting, connecting, service, done, abandoned} 확정. `StatusKicking` 死상태 제외(별도 제거 이슈).
- `connecting→waiting`은 **dial 성공前 실패 롤백에 한해** 존재(§5.4a). connecting 무응답 복귀로서의 waiting 전이는 없음.
- 종료 전이: connecting → service | abandoned. **service → done만**(service→abandoned 간선 없음 — abandon 술어 `status!='service'`가 원천 차단, 늦은 고객 hangup은 `event.go:28` 가드로 abandon 미호출→`confbridge_leaved`→done, §5.0·§5.8(4)).
- done 진입은 service→done CAS 유일(§5.8).
- 옵션 B: webhook 필드 유지(파생), 배열 컬럼만 제거.

## 7. Database Schema
- Phase 1: **DDL 없음**(CAS는 기존 컬럼 조건부 UPDATE, head-of-line 방지는 순회 로직이라 스키마 불변).
- Phase 2: `queue_queuecalls(queue_id, status)` 인덱스 추가, `queue` 배열 컬럼 drop(2단계). Alembic은 `bin-dbscheme-manager` 생성(AI는 파일만, upgrade 금지).

## 8. Handler Interface
`queuehandler`: `+EventAgentAvailable`, `+Wake`(=UpdateExecute(Run)).
`queuecallhandler`: `+TimeoutConnecting`. `QueuecallExecute` 반환에 "배정/미배정" 구분(순회 제어용).
`dbhandler` (전부 affected bool 반환):
- `+QueuecallListWaitingFIFO`(ASC, P1)
- `+SetExecuteRunIfStop`(P1)
- `+SetStatusWaitingIfInitiating`(P1, §5.4b)
- `+SetStatusConnectingIfWaiting`, `+SetStatusWaitingIfConnecting`(P1, §5.4a)
- `+SetStatusServiceIfConnecting`, `+SetStatusDoneIfService`, `+SetStatusAbandonedIfActive`(P2)
`requesthandler`: `+QueueV1QueueExecuteWake`(P1), `+QueueV1QueuecallTimeoutConnecting`(P2).

## 9. RabbitMQ / Routing
- 신규 구독: `agent-manager.agent.*.status_updated`.
- 신규 route: `POST /v1/queues/{id}/execute_wake`(P1), `POST /v1/queuecalls/{id}/timeout_connecting`(P2).
- 신규 self-RPC: `QueueV1QueueExecuteWake`(P1), `QueueV1QueuecallTimeoutConnecting`(P2). x-delayed-message 재사용.

## 10. Observability
- 기존 `promEventProcessTime` 자동.
- 신규(필수): `promAgentStatusEventAvailableTotal`, `promAgentStatusEventDiscardTotal`, `promQueueWakeTriggerTotal`.
- 권고: connecting abandon 카운터, 진입/flip/done CAS 승·패 카운터, **dial 실패 롤백 카운터(poison 콜 관측용, §16)**, **순회 스킵 카운터(head-of-line 빈도)**, **사이클당 dial 시도 수(poison flow churn 증폭률 관측, Phase 3 능동제거 트리거 판단 지표)**.

## 11. Security
- `EventAgentAvailable`는 payload CustomerID로 해당 customer 큐만(격리). 외부 신규 필드 없음. 신규 API 내부 self-RPC 전용.

## 12. Affected Files

| 파일 | 변경 | Phase |
|---|---|---|
| `subscribehandler/main.go` | agent 패턴+case+카운터 | 1 |
| `subscribehandler/agentmanager.go` | 신규 핸들러 | 1 |
| `subscribehandler/binding_golden_test.go` | 4→5 | 1 |
| `queuehandler/event.go` | `EventAgentAvailable` | 1 |
| `queuehandler/execute.go` | 순서 반전+유휴폴링 제거+안전망+I2 재확인+Get 오류 분기+FIFO 순회 | 1 |
| `queuehandler/db.go` | `UpdateExecute` CAS화+`queue_updated` 억제+`Wake` | 1 |
| `queuehandler/main.go` | 신규 상수(batch size 등) | 1 |
| `dbhandler/queue.go` | `SetExecuteRunIfStop` | 1 |
| `dbhandler/queuecall.go` | `QueuecallListWaitingFIFO`+`SetStatusWaitingIfInitiating`+`SetStatusConnectingIfWaiting`+`SetStatusWaitingIfConnecting` | 1 |
| `queuecallhandler/execute.go` | 진입 CAS+blind write 철거+성공前 waiting 롤백+배정/미배정 반환 | 1 |
| `queuecallhandler/db.go` | `UpdateStatusWaiting` CAS화(initiating 가드) | 1 |
| `listenhandler/*`, `requesthandler/queue_queue.go` | execute_wake | 1 |
| `queuecallhandler/{timeout,event,kick}.go` | 탈출 CAS(service/done/abandon) 게이팅 | 2 |
| `dbhandler/queuecall.go` | `SetStatusServiceIfConnecting`/`SetStatusDoneIfService`/`SetStatusAbandonedIfActive` | 2 |
| `listenhandler/*`, `requesthandler/queue_queuecall.go` | timeout_connecting | 2 |
| `queuehandler/db.go`, `models/queue/webhook.go` | 표시 필드 파생 | 2 |
| `docs/{architecture,domain,operations}.md` | 문서 동기화 | 1,2 |

## 13. Docs
- `architecture.md`: Event Subscriptions에 agent-manager, Request Routing에 execute_wake/timeout_connecting.
- `domain.md`: 상태공간 확정, connecting→waiting은 dial 성공前 실패 롤백 한정, service→done만, 표시 필드 파생, execute flip은 queue_updated 미발행.
- `operations.md`: 유휴 폴링 제거+Wake 안전망+I2, Get 오류 분기, FIFO 순회 head-of-line 방지, connecting CAS 실패 모드, dial 성공前/後 실패 처리 차이, poison 콜 관측, execute-flip CAS.

## 14. Implementation Order
1. (P1) subscribehandler 구독+golden+agentmanager+카운터.
2. (P1) `SetExecuteRunIfStop` + `UpdateExecute` CAS화 + `queue_updated` 억제 + `Wake`.
3. (P1) `QueueV1QueueExecuteWake` + execute_wake route.
4. (P1) `EventAgentAvailable` + 상수.
5. (P1) `QueuecallListWaitingFIFO`(ASC) + `SetStatusWaitingIfInitiating` + `SetStatusConnectingIfWaiting` + `SetStatusWaitingIfConnecting`.
6. (P1) execute.go 순서 반전+유휴폴링 제거+I2 재확인+안전망+Get 오류 분기+FIFO 순회; QueuecallExecute 진입 CAS+blind write 철거+성공前 waiting 롤백+배정/미배정 반환; UpdateStatusWaiting initiating 가드.
7. (P1) 검증 + 테스트(동시 flip 1승리/lost-wakeup 재확인/Get 일시오류 재스케줄/wake Stop→Run/진입 CAS 중복배정 차단/dial 성공前 실패 waiting 롤백/initiating hangup resurrection 차단/**poison 선두 콜 뒤 정상 콜 배정(head-of-line)**/FIFO/queue_updated 미발행). **[non-goal 주석]** cross-call(동시 Execute가 서로 다른 qc·동일 에이전트 dial)은 Phase 1 미차단·Phase 3 위임임을 테스트에 명시해 후속 회귀 오인 방지.
8. (P2) 탈출 CAS(service/done/abandon)+hangup/wait-timeout/customer-delete 관통+부수효과 게이팅+TimeoutConnecting+예약.
9. (P2) 표시 필드 파생 + `(queue_id,status)` 인덱스 + 마이그레이션.
10. (P2) 검증 + 테스트(service↔abandon 상호배타/service→done 단일 발행/waiting-abandon 정상/이중 정리·webhook 없음) + 문서.

## 15. Open Questions

| # | 질문 | CPO 권고 | 우선순위 |
|---|---|---|---|
| Q1 | Phase 1+2 한 PR vs 분리 | 커밋 분리, diff 크면 별도 PR | 즉시 |
| Q2 | 안전망(Wake 60초) 유지 vs 이벤트만 | 유지(대기콜 있을 때만) | 즉시 |
| Q3 | 표시 필드 B(파생) vs A(제거) | B | Phase 2 |
| Q4 | ~~wait 예산 차감~~ → abandon 확정 | 결정됨 | — |
| Q5 | cross-call race(Phase 3) | Phase 3. Phase 1 진입 CAS로 같은 큐 중복배정 차단 | 트리거 도달 시 |
| Q6 | connecting 타임아웃 기본값(30초) | 30초 시작 | Phase 2 |
| Q7 | 이벤트/wake 부하 실측 문제 시 경량 이벤트/tag 사전필터 | 계측 먼저 | 트리거 도달 시 |
| Q8 | execute run/stop을 고객 webhook에서 빼는 것 영향 | 없음(WebhookMessage에 없음). 대표님 확인 요망 | 즉시 |
| Q9 | CAS-크래시/hangup-during-dial 고아(현행 존재) reaping | 본 PR 범위 밖, 별도 이슈 | 트리거 도달 시 |
| Q10 | dial 성공前 실패 시 waiting 복귀(재시도) vs 즉시 재라우팅 | waiting 복귀(단순, 현행 보존). 재라우팅은 Phase 3 | 결정됨 |
| Q11 | **[iter-6·7]** poison 선두 콜 head-of-line | Phase 1 **사이클 내 다음 후보 진행으로 해결(§5.4)**. poison 콜 능동 제거(실패 카운트 상한)는 실측 신호 시 Phase 3 | 결정됨(구조) / 완화 Phase 3 |
| Q12 | `defaultExecuteBatchSize` 값 | 10 시작(순회 상한, dial 낭비 vs head-of-line 커버 균형). 실측 후 조정 | 즉시 |

## 16. Rollout / Risk
- Phase 1: 유휴 폴링 제거 + 동시성 안전화. (a) wake 유실 → 60초 안전망. (b) 정렬 누출 → 라우팅 전용 dbhandler. (c) 동시 wake 중복 Execute → flip CAS. (d) lost-wakeup → I2 재확인(Get 오류 분기 포함). (e) resurrection(initiating·waiting) → enqueue/진입 CAS + blind write 철거. (f) dial 성공前 실패 고객 드롭 → waiting 롤백. (g) webhook 증폭 → flip 억제.
- **[iter-7 결함 해소] poison 선두 콜 head-of-line 기아**: FIFO(ASC)가 최고참 waiting 재선택 시, dial 지속 실패하는 선두 콜(손상 confbridge/activeflow)이 뒤 콜을 막는 문제. **§5.4 사이클 내 다음 후보 진행으로 구조적 해결** — 순회가 실패 콜을 건너뛰고 다음 FIFO 후보를 배정하므로 뒤 정상 콜이 기아하지 않는다. **`wait_timeout=0`(유효 설정, create.go/db.go:139 `TimeoutWait>0` 가드로 타임아웃 미예약) 큐에서도** wait-timeout 상한에 의존하지 않고 회귀를 차단한다(iter-7 리뷰어 B 지적의 핵심 전제 해소). poison 콜 자체는 무한 재시도하나 무해(다른 콜 미차단)하고, 진짜 영구 손상 콜은 대개 고객 hangup으로 자연 abandon. **[iter-8 flow churn 증폭 명시]** 순회는 사이클당 최대 N회(size=N) `generateFlowForAgentCall`+dial 시도를 유발할 수 있다(현행 1회/사이클 대비 증폭). 단 해당 flow는 `FlowV1FlowCreate(persist=false)` Redis 임시·자동만료라 영구 DB·maxFlowCount 미계상, size=N bounded, 전량 실패 시 `ExecuteRun(1000ms)`로 초당 상한. 신규 결함 클래스가 아니라 기존 orphan-reaping(Q9) 하위집합. **능동 제거(실패 카운트 상한 후 abandon)**는 dial 실패 롤백 카운터·순회 스킵 카운터·사이클당 dial 시도 수(§10) 실측 신호 도달 시 Phase 3(Q11) — 실측 전 도입 보류(오버엔지니어링 방지).
- Phase 2: 탈출 대칭 CAS(service/done/abandon)로 라이브 통화 절단/이중 정리/이중 webhook(done 이중 발행 포함)/waiting-abandon 회귀 차단. abandon 확정으로 무한 루프 없음. 표시 필드 파생(파괴적 변경 없음).
- **[iter-4·5·6 minor/Q9]** CAS 커밋~정리 사이 크래시, hangup-during-dial 고아, dial 성공後 forwardAction 실패 connecting stuck은 현행에도 존재하는 비회귀 결함(대부분 고객 hangup/join 실패로 자가 소멸)으로 본 PR 범위 밖(별도 reaping/Phase 2 connecting-timeout).

## 17. Review Summary

### iter-1~6 요약
- iter-1: C1 무한루프→abandon, C2 안전망 모순→대기콜 확인, C4 이벤트 볼륨→카운터, C5 FIFO→dbhandler 직접, C6 배열 stale→wake 게이트 배열 제거, C7 Phase 분리.
- iter-2(독립 동일 BLOCKER): #1 wake no-op→`UpdateExecute(Run)`+execute_wake, #2 Kick 통화 절단→CAS abandon, #3 순서반전 고착→진입 CAS+롤백 재도입, #4 팬아웃 정량, #5 배열 제거 파괴적→옵션 B.
- iter-3(독립 수렴): #1 abandon만 CAS→탈출 전이 CAS, #2 flip webhook 증폭→억제, #3 UpdateExecute 비원자→`SetExecuteRunIfStop`, #4 double-assign→CAS 직렬화, #5 파생 부하→억제+인덱스.
- iter-4(독립 수렴): 결함1 공유 sink 파괴→abandon not-terminal 일반화, 결함2 진입 blind→진입 CAS, lost-wakeup→I2, 부수효과 게이팅→I3, double-publish→226만 억제.
- iter-5(A CHANGES / B APPROVED): A-1 진입 CAS 완결성(dial 후 blind 철거), A-2 롤백 abandon→waiting 복귀, A-3 initiating→waiting CAS, A-4 롤백 범위, A-5 StatusKicking 상태공간.
- iter-6(둘 다 CHANGES, "회귀 블로커 없음", 독립 수렴): service→done CAS 누락→`SetStatusDoneIfService`, Get 오류 I2 미커버→분기, poison head-of-line→§16, dial 성공後 비롤백 문서.

### iter-7 (리뷰어 A CHANGES[문서 2] / B CHANGES[실질 1], 핵심 설계 성숙 확인) v7→v8
| ID | 지적 | 반영 |
|---|---|---|
| A-1 (문서 모순) | line 204 "service 종료 전이는 done\|abandoned"가 설계(service→abandoned 금지, done만)와 모순 | §6·§5.0·§5.8(4): "service→done만, service→abandoned 간선 없음(abandon 술어 status!='service' 차단, 늦은 hangup은 event.go:28 가드→done)" 정정 |
| A-2 (문서 clarity) | line 101 다이어그램 "ErrNotFound→Stop"이 pseudocode 순수 return과 불일치(DB write 오독) | §4: "ErrNotFound→return(DB write 없음, 재개 불필요)" 표기 정정 |
| B-1 (실질, 검증) | `wait_timeout=0`(유효 설정) 큐는 wait-timeout 미예약(db.go:139)→poison 상한 부재→FIFO가 선두 poison 영구 재선택→뒤 콜 기아(LIFO 대비 회귀). §16 "wait-timeout이 상한" 전제 무효 | §5.4·§16·Q11·Q12: **사이클 내 다음 후보 진행**(FIFO size=N 순회, 실패 콜 스킵, 첫 성공 배정)으로 head-of-line 구조적 해결. wait_timeout 값과 무관, 스키마 추가 없음. 능동 poison 제거는 실측 신호 시 Phase 3 |
| A/B 공통 | 핵심 설계(7전이 CAS·done 게이팅·Get 분기·Concern 1~4) 성숙·정확, 회귀 블로커 없음 | 확인 |

iter-7 리뷰어 A는 7전이 전수 CAS 그래프 닫힘을 코드로 재확인(문서 정합성 2건만 지적), 리뷰어 B는 concern 1~4 건전 확인 후 `wait_timeout=0` poison 상한 결함 1건 발견. v8은 문서 모순 2건 정정 + poison head-of-line을 순회로 구조 해결(설정 무관, 무스키마).

### iter-8 (리뷰어 A·B 둘 다 APPROVED — 첫 양쪽 승인) v8→v9
| ID | 지적 | 반영 |
|---|---|---|
| A/B 공통 (핵심) | 7전이 전수 CAS·done 게이팅·Get 분기·순회 head-of-line 방지 코드 검증 완료. 순회 로직이 에이전트 이중 소비/페이싱 붕괴/신규 race 미발생, 종료 5경로 전부 done 수렴 확인 | 확인, 변경 없음 |
| A-권고1 (비블로커) | 순회의 사이클당 flow-create 증폭(현행 1→최대 N)을 명시하고 관측 지표 추가 | §16: flow churn 증폭 명시(persist=false Redis 임시·자동만료·bounded), §10: "사이클당 dial 시도 수" 카운터 추가 |
| A-권고2 (비블로커) | `if !ok: break`가 도달 불가 분기임을 주석 | §5.4: agents 스냅샷 불변이라 사실상 도달 불가 주석 |
| A-권고3 (비블로커) | cross-call은 Phase 1 non-goal임을 테스트 주석으로 | §14 step 7: cross-call non-goal 테스트 주석 |

iter-8에서 두 리뷰어가 처음으로 동시 APPROVED. 리뷰어 B는 적대적 4초점(순회 부작용·종료 커버·cross-call·페이싱)을 코드로 전수 검증해 신규 실질 결함 없음 확인. v9는 A의 비블로커 권고 3건(전부 문서/관측 보강, correctness 무변경)만 반영. 설계 리뷰 정책상 2연속 양쪽 APPROVE 필요 → iter-8이 연속 1회, iter-9가 최종 확인 라운드.
