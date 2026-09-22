# Q&A 분석 주장 검증 대상 (queue-manager 이벤트 드리븐 설계 관련)

이 문서는 설계 문서 v9와 별개로, 세션 중 사용자(대표님) 질문에 답하며 제시한 **분석 주장**들을 코드로 검증받기 위한 목록이다. 각 주장이 실제 monorepo 코드와 일치하는지, 과장/오류가 없는지 검증한다.

> **[구현됨 vs 설계 단계 구분 — iter-1 공통 보완]** 아래 주장 중 현행 코드에 **이미 존재**하는 사실(주장 1·2·3, 그리고 5의 "self-RPC 에러 폐기"·방어3)과, redesign.md에만 있는 **미구현 설계**(주장 4·6, 5의 방어1·2, 즉 `SetExecuteRunIfStop`·I2 재확인·60초 `execute_wake` 안전망)를 구분한다. 미구현 설계 주장은 코드 대조가 아니라 "설계 추론의 타당성"으로 검증한다.

## 주장 1. 최초 request 메시지 생성 주체는 스케줄러가 아니라 "일이 생긴 트랜잭션" [현행 코드]

- (A) 새 대기콜 진입: `queuecallhandler/db.go:358` `AddWaitQueueCallID` → `queuehandler/db.go:249` `UpdateExecute(Run)` → `db.go:228-230` `q.Execute==Stop`이면 `QueueV1QueueExecuteRun(ctx, id, 100)` self-RPC 발행.
- (B) 새 설계: agent available 이벤트 수신 → 같은 `UpdateExecute(Run)` 경로.
- Execute 자기 재생성: `queuehandler/execute.go:99` 매 사이클 끝에 `QueueV1QueueExecuteRun(id, 100)` 재발행. 배정할 게 없으면 Stop 찍고 멈춤.
- **[iter-1 각주]** 성공 경로(execute.go:99) 외에도 재발행(재시도) 지점 다수 존재: `execute.go:46/66/72/94`(각종 오류 시 `defaultExecuteDelay`로 재스케줄). 체인 자체는 정확.
- **검증 포인트**: 별도 주기 스케줄러/틱이 라우팅 request를 만드는 경로가 정말 없는가? 위 체인이 정확한가?

## 주장 2. queue-manager는 competing consumer라 이벤트 1개는 인스턴스 1개만 수신 [현행 코드]

- 모든 인스턴스가 같은 이름 큐 공유: 이벤트=`QueueNameQueueSubscribe`(`cmd/queue-manager/main.go:149`), 요청=`QueueNameQueueRequest`(`main.go:171`). 큐명은 고정 상수(`queuename.go:113-114`).
- 큐 선언: `queue.go:56-68` `queueCreateNormal`→`QueueDeclare(durable=true, autoDelete=false, exclusive=false)`. 인스턴스별 고유 큐 아님.
- 소비: `subscribehandler/main.go:129` `ConsumeMessage(..., exclusive=false, ...)`, `listenhandler/main.go:161` `ConsumeRPC(..., exclusive=false, ...)`, consumerName 고정 "queue-manager".
- 공유 큐 + 비배타 소비 = competing consumer → RabbitMQ 라운드로빈 → 이벤트/요청 1개는 1인스턴스만 수신.
- **검증 포인트**: 큐 이름이 인스턴스마다 고정(공유)인가, 아니면 인스턴스별 고유(fanout)인가? (이게 틀리면 N배 중복 주장 전체가 바뀜)

## 주장 3. 현재 `UpdateExecute`는 read-then-write라 동시 wake 시 중복 발생 가능(잠재 버그) [현행 코드]

- `queuehandler/db.go:201-231`: `QueueGet`(execute 읽기) → `if q.Execute==execute return`(207) → `QueueUpdate`(216) → `PublishEvent`(226) → `QueueV1QueueExecuteRun`(230). 원자 아님.
- **검증 포인트**: 동시 다중 wake가 모두 Stop을 읽고 모두 run을 쓰고 모두 메시지를 발행할 수 있는가? 즉 현재 코드에 중복 execute_run/이중 배정 race가 실재하는가?

## 주장 4. flip CAS가 인스턴스 경계를 넘어 중복 wake를 dedup [미구현 설계]

- 새 설계 `SetExecuteRunIfStop`(redesign.md §5.5, **현행 코드 미존재**) = `UPDATE queues SET execute='run' WHERE id=? AND execute='stop'`. affected=1(승자)만 `QueueV1QueueExecuteRun` 발행, affected=0은 no-op.
- 공유 MySQL 행 잠금(InnoDB, PK=queue id)이라 인스턴스 N개여도 승자 1개 → 메시지 1개.
- **[iter-1 단서]** "인스턴스 수 무관 큐당 1개"는 **동시 wake 한정** 주장이다. flip→완료→재flip 순차 사이클은 정상적으로 복수 메시지를 만든다(버그 아님, 의도된 자기영속 체인).
- **검증 포인트**: 단일 조건부 UPDATE가 정말 원자적 dedup을 보장하는가? MySQL 격리수준/행잠금 전제가 맞는가?

## 주장 5. 메시지 유실 3중 방어 [혼합: 유실 사실=현행 / 방어1·2=설계]

- self-RPC는 `_ = h.reqHandler.QueueV1QueueExecuteRun(...)`로 에러를 버림(`db.go:230`, `execute.go:99`). [현행]
- **[iter-1 결정적 근거]** execute_run/execute_wake은 RPC 요청 경로(`QueueNameQueueRequest`)이고 `consumeRPCWorker`는 **ack-before-process, 재시도 없음**(`consume.go:263-269`). 이벤트 경로의 `maxEventRetries=3` 백오프 재시도(`consume.go:143-190`)와 달리 RPC self-메시지는 전송계층 재시도가 없어 유실이 실제로 가능 → 안전망 필요성의 코드적 근거.
- 방어1 (I2, 경합 유실) [설계]: sleep 후 대기콜 재확인 + 자가 Run-CAS.
- 방어2 (60초 안전망, 메시지 유실) [설계]: 대기콜 있는 채 잠드는 모든 경로에서 `execute_wake` 지연 메시지 예약. 유실돼도 최대 60초 후 자가복구. 대기콜 0건이면 예약 안 함.
  - **[iter-1 적대적 관찰: 하드닝]** 안전망 `execute_wake`는 `RequestPublishWithDelay`→`publish.go:147` **transient** 발행. 재시도 경로(`consume.go:169`)는 브로커 재시작 생존 위해 의도적 `amqp.Persistent` 사용. 60초 안전망을 **persistent로 승격**하면 방어2가 브로커 재시작에도 견고해짐. (설계 하드닝 항목, redesign.md에 반영 권고)
- 방어3 (헬스체크, 콜 유실) [현행]: `pkg/queuecallhandler/health.go`. 참조 콜 생존 확인 후 유실 시 force-kick. "콜 유실" 백스톱이지 라우팅 메시지 복구는 아님(라벨 정확).
- **검증 포인트**: 60초 안전망이 정말 "self-RPC 유실 시 영구 고착"을 막는가? 안전망도 유실되면?(→ persistent 승격으로 완화) 안전망이 이벤트 wake와 겹칠 때 최악이 "헛도는 1사이클"이 맞는가(중복 배정 없음)?

## 주장 6. 동시 다중 wake + 60초 안전망 겹침에도 execute_run은 큐당 1개 [미구현 설계, 주장4 CAS 의존]

- 시나리오: 큐 잠듦(60초 예약) → 30초에 agent available wake(CAS 승리, 배정, 다시 Stop) → 60초 안전망 도착 → CAS가 상태 확인(Stop이면 승리 후 Execute 1회→대기콜0→즉시 Stop=헛도는 1사이클 / Run이면 no-op).
- **검증 포인트**: 안전망도 같은 flip CAS 게이트를 지나므로 중복 배정/메시지 폭주가 없는가? 최악이 무해한 헛사이클인가? (주장4 CAS 미구현에 의존 → 설계 추론으로 검증)

## 주장 7. reconcile(표시 필드 스냅샷 갱신)은 "틱"이면 N배 중복, 그래서 Lazy(조회 시 TTL) 권고 [설계 추론]

- 각 프로세스 독립 타이머로 전 큐 스캔 → 인스턴스 N개면 N배 중복 작업. **competing consumer는 공유 큐의 push 메시지에만 적용, 로컬 타이머엔 미적용**이라 틱은 안 막힘.
- Lazy: GET/webhook 시 stale(TTL 경과)이면 1회 재계산+저장. 아무도 안 보면 0. 새 폴링 없음.
- 지연 메시지 self-RPC 대안: 활성 큐가 자기 reconcile 예약 → competing consumer로 1인스턴스만 처리(중복 없음)지만 유휴에도 약간 헛일.
- **[iter-1 관찰]** Lazy도 동시 GET 다수면 중복 재계산 가능하나 멱등·경계적이라 틱 대비 무해.
- **검증 포인트**: "프로세스별 틱은 N배 중복"이 맞는가? Lazy가 중복/유휴폴링을 모두 회피하는가? 현재 코드에 reconcile 틱 자체가 없어 회귀 대상은 아님.

---

## iter-1 리뷰 결과 (리뷰어 A·B 둘 다 APPROVED)
- 주장 1·2·3: CONFIRMED(현행 코드 일치). 주장 4·5·6·7: 미구현 설계에 대한 타당한 추론, 실질 오류·과장 없음.
- 적대적 반증 시도 전부 실패: competing consumer(exclusive/autoDelete fanout 가설 반증), flip CAS dedup vs 유실 모순(별개 축, 은폐 없음), 안전망 자체 유실(공개 질문으로 노출됨), Lazy 중복(멱등·경계적).
- 반영한 보완 5건: (1) 구현됨/설계 단계 구분 명시, (2) RPC ack-before-process·무재시도 근거 추가 + persistent 승격 권고, (3) health.go 경로 정정, (4) 주장4 "동시 wake 한정" 단서, (5) 주장1 재발행 지점 각주.
- iter-2에서 fresh 재검토(2연속 Approve 확인).
