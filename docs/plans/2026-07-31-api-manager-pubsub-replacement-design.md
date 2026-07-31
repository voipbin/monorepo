# api-manager ZeroMQ 대체 분석 (디자인 문서)

날짜: 2026-07-31
작성: Claude (CPO)
상태: 디자인 리뷰 루프 완료 (R1 Request Changes → R2 Approve → R3 Approve, 2연속 Approval)

## 1. 배경 및 현재 상태 (코드 확인 기반 사실)

`bin-api-manager`는 내부 pub/sub으로 ZeroMQ(`github.com/pebbe/zmq4`, cgo 바인딩)를 사용한다.

**사용 실태:**
- 전송 방식은 `inproc://api-manager` 단 하나. 네트워크 통신 없음. 같은 프로세스 내 goroutine 간 전달 전용.
- 데이터 흐름: `subscribehandler`가 RabbitMQ에서 웹훅/이벤트를 수신 → `zmqpubhandler.Publish(topic, data)`로 발행 (PUB 소켓 1개, bind) → WebSocket 연결마다 생성되는 `zmqsubhandler`(SUB 소켓, connect)가 topic 필터링 후 해당 WebSocket 클라이언트로 전달.
- 구조: PUB 소켓 1개, 구독자 N(WebSocket 연결 수만큼)의 topic 기반 fan-out. 단, **`Publish` 호출은 동시 다발적이다**. `subscribehandler.processEventRun`이 이벤트마다 `go h.processEvent(m)`을 띄우므로 다수 goroutine이 동시에 Publish를 호출한다.
- 모노레포 전체(bin-* 서비스군 + voip-* 프록시)에서 ZeroMQ 사용처는 `bin-api-manager` 단 한 곳 (전체 go.mod 전수 조사, `pebbe/zmq4` 기준).
- pub/sub 랑데부는 libzmq의 전역 기본 컨텍스트 + `inproc://api-manager` 주소 문자열을 통해 암묵적으로 이뤄진다. `NewZMQSubHandler()`는 인자를 받지 않고, `websockhandler`는 pub 핸들러 참조를 갖지 않는다.
- 관련 패키지: `pkg/zmq` (소켓 래퍼), `pkg/zmqpubhandler`, `pkg/zmqsubhandler`. 모두 인터페이스로 추상화되어 있고 mockgen mock 존재.

**보존해야 할 동작 (behavioral contract):**
1. **Prefix 매칭 구독**: ZMQ SUB의 topic 필터는 prefix 매칭이다. 실제로 클라이언트는 `customer_id:<id>:call` 같은 prefix로 구독하며, `subscribehandler.createTopics()`가 더 긴 구체 topic을 발행한다 (`processEventWebhookManagerRoutingKeyedEvent`의 주석에 명시: "ZMQ SUB clients (which subscribe using the OLD-FORMAT prefix)"). 대체 구현은 prefix 매칭을 반드시 유지해야 한다.
2. **느린 구독자 격리**: ZMQ PUB/SUB은 HWM(high water mark) 도달 시 해당 구독자에 대한 메시지를 드롭한다. 즉 느린 WebSocket 연결 하나가 발행자나 다른 구독자를 블록하지 않는다. 이벤트 스트림 특성상 드롭은 허용된다(웹훅이 신뢰 전달 경로, WebSocket은 실시간 알림 경로). **현재 유효 용량**: `pkg/zmq`는 SNDHWM/RCVHWM을 설정하지 않으므로 libzmq 기본값(측당 1000, inproc은 양측 합산으로 실효 ~2000 메시지)이 적용된다. 대체 구현의 버퍼는 이 기준 이상이어야 한다.
3. **구독자 생명주기**: WebSocket 연결마다 subscriber 생성/종료(`Terminate`). Subscribe/Unsubscribe는 연결 도중 동적으로 발생.
4. **발행자 비블로킹**: `Publish`는 구독자 상태와 무관하게 즉시 반환.
5. **와이어 형식**: 2-frame multipart (`SendMessage(topic, m)`, 수신 측은 `len(m) == 2` 강제). prefix 필터는 첫 frame(topic)에만 적용된다. 대체 구현의 `Message{Topic, Data}` 구조체는 이와 동등하다.
6. **구독 전파 타이밍**: ZMQ의 구독 등록은 비동기라 `Subscribe` 직후 짧은 구간 동안 이벤트를 놓칠 수 있다. 대체 구현의 동기식 구독은 이보다 엄격히 낫다(놓치는 구간 없음). 즉 slow-joiner 동작은 개선이지 회귀가 아니다.

## 2. 문제 정의 (왜 재검토하는가)

1. **cgo 의존**: `pebbe/zmq4`는 cgo 기반. libzmq 네이티브 라이브러리 필요 (`apt install pkg-config libzmq5 libzmq3-dev libczmq4 libczmq-dev`). 이로 인해:
   - 37개 서비스 중 유일하게 빌드 사전 준비가 다른 서비스. Docker 이미지, CI, 로컬 개발 환경 모두 특별 취급.
   - `CGO_ENABLED=0` 정적 빌드 불가, distroless/scratch 최소 이미지 채택 제약.
   - cgo 경계는 크래시/디버깅 난이도, race detector 사각지대 등 운영 리스크 소폭 증가.
   - sandbox 셀프호스팅, AI 주도 설치를 추진 중인 제품 방향에서 "빌드 요구사항이 다른 서비스 하나"는 지속적 마찰 비용.
2. **기능 대비 과한 도구**: 실제 필요 기능은 "프로세스 내부, topic prefix 기반, drop 허용 fan-out" 뿐. 범용 메시징 라이브러리의 나머지 기능(네트워크 전송, 다양한 소켓 패턴)은 전혀 사용하지 않는다.
3. **부수 비용**: `zmqsubhandler.receiveMessage`는 `ReceiveNoBlock` + 1초 sleep 폴링 루프로 구현되어 있다(블로킹 Receive가 소켓 종료를 인지하지 못하는 zmq 특성의 우회). 이는 이벤트 전달에 최대 1초 지연을 추가하고, 과거 flaky test의 원인이기도 했다(코드 주석에 기록). Go channel 기반이면 폴링 없이 select로 즉시 전달/즉시 종료가 된다.
4. **기존 잠재 data race**: libzmq 소켓은 thread-safe가 아니다. 그런데 현재 코드는 (a) PUB 측에서 이벤트마다 goroutine을 띄워 단일 PUB 소켓에 동시 `Publish`하고, (b) SUB 측에서 websock reader goroutine의 `Subscribe`/`Unsubscribe`, Run goroutine의 `ReceiveNoBlock`, 부모 defer의 `Terminate()`가 한 소켓을 3개 goroutine에서 만진다. `run.go`의 flaky test 주석은 이 증상 중 하나를 기록한 것이다. 즉 현행 구조는 이미 동시성 계약을 위반하고 있으며, 이는 교체의 추가 근거다.

단, 현재 구조가 "동작하지 않는다"는 문제는 없다. 이는 긴급 장애 대응이 아니라 부채 상환 성격의 결정이다.

## 3. 대안 비교

| 기준 | A. 현상 유지 (pebbe/zmq4) | B. 순수 Go 내부 pub/sub (자체 구현) | C. go-zeromq/zmq4 (pure Go ZMQ) | D. RabbitMQ로 통합 | E. Redis pub/sub |
|---|---|---|---|---|---|
| cgo 제거 | ✗ | ✓ | ✓ | ✓ | ✓ |
| 외부 의존성 | libzmq 네이티브 | 없음 (stdlib) | Go 모듈 1개 | 기존 인프라 | 기존 인프라 |
| prefix 매칭 | 기본 제공 | 직접 구현 (단순) | 기본 제공 | topic exchange 와일드카드 (`#`/`*`, prefix와 의미 다름 → 변환 계층 필요) | `PSUBSCRIBE` glob (prefix 표현 가능) |
| 느린 구독자 드롭 | HWM | buffered chan + drop 정책 직접 구현 | HWM | per-connection queue TTL/limit 설정 필요 | Redis가 client output buffer로 처리 |
| 지연/홉 | inproc | 프로세스 내 (폴링 제거로 오히려 개선) | inproc 상당 | 네트워크 왕복 추가 (이미 RabbitMQ에서 온 이벤트를 다시 브로커로) | 네트워크 왕복 추가 + Redis 부하 |
| 성숙도/유지보수 | 성숙하나 바인딩 유지보수 부담 | 코드 소유권 100%, 표면적 작음 (~200-300줄 예상) | v4 프로토콜 구현이나 커뮤니티 규모 작음, pebbe 대비 검증 부족 | 성숙 | 성숙 |
| 장애 결합도 | 프로세스 내 | 프로세스 내 | 프로세스 내 | 브로커 장애 시 WebSocket 전달도 중단 (현재는 무관) | Redis 장애 시 동일 |
| 마이그레이션 비용 | 0 | 소 (인터페이스 유지, 구현체 교체) | 소~중 (API 차이 흡수) | 중 (연결당 queue/binding 관리 재설계) | 중 |

**탈락 사유 요약:**
- **D (RabbitMQ)**: 이벤트가 이미 RabbitMQ에서 오는데 pod 내부 fan-out을 위해 다시 브로커를 왕복하는 것은 불필요한 네트워크 비용과 브로커 부하. WebSocket 연결마다 queue/binding 생성 시 연결 수에 비례해 브로커 객체가 늘어남. 프로세스 내부 문제를 외부 인프라로 푸는 방향 자체가 역행.
- **E (Redis)**: D와 같은 이유. 공유 인프라(Redis)에 pod-로컬 트래픽을 태울 이유가 없음. Redis pub/sub은 클러스터 전환 시 제약도 있음.
- **C (go-zeromq/zmq4)**: cgo는 제거되지만 ZMQ 와이어 프로토콜 자체가 불필요한 상황(원격 통신 없음). 검증이 덜 된 구현체를 들여오는 리스크만 남는다.
- **A (현상 유지)**: 기능적 문제는 없으나 §2의 비용이 계속 누적. 특히 셀프호스팅/설치 단순화 로드맵과 충돌.

## 4. 권장안: B. 순수 Go 내부 pub/sub 자체 구현

### 4.1 설계 스케치

새 패키지 `pkg/pubsubhandler` (또는 기존 `zmqpubhandler`/`zmqsubhandler` 인터페이스를 유지한 채 내부 구현만 교체):

```go
// Broker: 프로세스당 1개. 현재의 inproc PUB 소켓 역할.
type Broker struct {
    mu   sync.RWMutex
    subs map[*Subscriber]struct{}
}

// Publish: 모든 구독자에 대해 prefix 매칭 검사 후 non-blocking send.
// 버퍼가 가득 찬 구독자는 드롭(+ 드롭 카운터 메트릭).
// 동시 호출 안전(RLock). subscribehandler가 이벤트마다 goroutine으로 호출하는 현실을 전제.
func (b *Broker) Publish(topic, data string)

// Subscriber: WebSocket 연결당 1개. 현재의 SUB 소켓 역할.
type Subscriber struct {
    mu     sync.Mutex
    topics []string          // prefix 목록. Publish 경로도 s.mu로 보호하며 읽는다.
    ch     chan Message      // buffered, 기본 2000 (현행 inproc 실효 HWM ~2000과 동등)
    done   chan struct{}     // 종료 신호. 데이터 채널(ch)은 절대 close하지 않는다.
}

func (s *Subscriber) Subscribe(prefix string)
func (s *Subscriber) Unsubscribe(prefix string)
func (s *Subscriber) Chan() <-chan Message   // Run 루프에서 done과 함께 select로 소비
func (s *Subscriber) Close()                 // 내부적으로 broker.unregister(s) 호출 후 done close.
                                             // Subscriber는 생성 시 Broker 역참조를 보관한다.
```

- **Prefix 매칭**: `strings.HasPrefix(topic, subscribedPrefix)`. ZMQ SUB 의미론과 동일. 매칭 검사 시 `Subscriber.topics`는 `s.mu`로 보호해 읽는다 (Subscribe/Unsubscribe가 동시 변경하므로). **중복 prefix 시 1회만 전달(deliver-once)**: 구독 prefix 여러 개가 같은 topic에 매칭돼도(예: `customer_id:x:`와 `customer_id:x:call`) 메시지는 한 번만 전달한다. 첫 매칭에서 break. ZMQ SUB 동작과 동일하며, 테이블 테스트로 커버한다.
- **드롭 정책**: `select { case s.ch <- m: default: dropCount++ }`. ZMQ HWM 드롭과 동등, 대신 관측 가능(메트릭). 버퍼 크기는 현행 inproc 실효 용량(~2000)과 동등한 **기본 2000**.
- **종료 안전성 (핵심 설계 결정)**: 데이터 채널 `ch`는 **절대 close하지 않는다**. `Close()`는 (1) Broker의 write lock 아래에서 구독자를 `subs`에서 제거하고 (2) `done` 채널을 close한다. `Run` 루프는 `select { case m := <-s.ch: ... case <-s.done: return }`으로 종료를 감지한다. 이 순서 덕분에 동시 `Publish`(RLock 보유)와 `Close()`(Lock 대기)가 경합해도 send-on-closed-channel panic이 원천적으로 불가능하다. 현재의 1초 폴링 및 EAGAIN 처리 코드는 삭제된다. 참고: `close(done)` 직후 Run 루프의 select가 남은 버퍼 메시지 몇 건을 종료 중인 WebSocket에 더 흘릴 수 있으나, write 오류는 기존 경로에서도 허용되는 동작이므로 무해하다(현행과 동등).
- **동시성 요약**: Publish는 다중 goroutine 동시 호출 안전(RLock + per-subscriber non-blocking send), 구독자 등록/해제는 Broker write lock. 이는 현행 zmq 소켓이 지키지 못하던 계약(§2-4)을 처음으로 올바르게 지키는 것이기도 하다.

### 4.2 마이그레이션 계획

1. `pkg/pubsubhandler` 신규 구현 + 단위 테스트 (prefix 매칭, 중복 prefix deliver-once, 동일 prefix 재구독 멱등성(현행 `slices.Contains` 동작 보존), 드롭, 동시 subscribe/unsubscribe, Publish/Close 경합. `-race` 필수).
2. **Broker 배선 (명시적 결정 필요)**: 현재는 pub/sub이 libzmq 전역 컨텍스트로 암묵적으로 만나므로, 순수 Go 대체 시 랑데부 지점을 명시적으로 만들어야 한다. 선택지는 (a) 패키지 전역 Broker 싱글턴(현행 전역 컨텍스트와 대칭, diff 최소) 또는 (b) `main.go`에서 Broker를 생성해 `subscribehandler`와 `websockhandler` 양쪽에 생성자 주입(명시적, 테스트 용이, churn 소폭 증가). **권장: (b) 생성자 주입.** 모노레포의 기존 DI 관례(생성자 주입)와 일치하고, 전역 상태는 이 리팩터링에서 굳이 새로 도입할 이유가 없다. `ZMQPubHandler`/`ZMQSubHandler` 인터페이스 시그니처는 유지하되, `NewZMQSubHandler()`가 Broker 인자를 받는 형태로 변경된다 (`websockhandler`에 Broker 참조 전달 필요).
3. `pkg/zmq`, `pkg/zmqpubhandler`, `pkg/zmqsubhandler`, `pebbe/zmq4` 의존성 제거.
4. Dockerfile/CI에서 libzmq 설치 단계 제거, `CGO_ENABLED=0` 전환 검토.
5. sandbox 환경에서 WebSocket 이벤트 수신 E2E 확인 (기존 `test_call.py`/talk 앱 시나리오 활용).

### 4.3 리스크 및 완화

| 리스크 | 완화 |
|---|---|
| prefix 매칭 의미 차이로 이벤트 누락 | ZMQ와 동일한 `HasPrefix` 의미론 사용 + 기존 topic 생성/구독 케이스를 테이블 테스트로 커버 |
| 드롭 정책 차이로 체감 동작 변화 | 버퍼 크기를 현행 inproc 실효 용량과 동등(기본 2000)으로 설정, 드롭 카운터 메트릭 추가로 관측 |
| 자체 구현의 동시성 버그 | Publish/Close/Subscribe 경합 시나리오를 `-race`로 커버하는 스트레스 테스트 필수. 종료 설계(§4.1)로 panic 클래스는 구조적으로 제거 |
| 숨은 zmq 의존 (다른 사용처) | go.mod 전수 조사 완료: api-manager 외 사용처 없음. `docsdev` 문서 및 CLAUDE.md의 libzmq 언급 동시 정리 |
| WebSocket write 경로 회귀 | `RunWithMutex` 의미(공유 writeMu) 그대로 유지, 기존 테스트 이식 |

### 4.4 트레이드오프 명시

- **얻는 것**: cgo/네이티브 의존 제거, 빌드/이미지/CI 단순화, 폴링 지연 제거, 기존 잠재 data race(§2-4) 해소, 코드 소유권과 관측성.
- **잃는 것**: 검증된 라이브러리의 성숙도 대신 자체 코드 유지보수 책임. 단, 표면적이 작고(수백 줄) 도메인이 단순해 감내 가능한 수준.
- **하지 않는 것**: 멀티 pod 간 이벤트 공유 구조 변경 없음(현재도 pod별 RabbitMQ queue로 전 이벤트 수신, pod-로컬 fan-out). 이 설계 결정은 본 문서 범위 밖.

## 5. 결론

ZeroMQ는 "프로세스 내부 fan-out"이라는 실제 용도에 비해 과한 도구이며, cgo 의존이 유일하게 남는 실질 비용이다. **순수 Go 자체 구현(B)으로 교체를 권장한다.** 긴급하지 않으므로 일반 우선순위 티켓으로 등록 후 진행한다.
