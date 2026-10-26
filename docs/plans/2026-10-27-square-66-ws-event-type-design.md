# SQUARE-66: WS event_type key and agent topic for conversation messages

## 1. 티켓 검토 (요약)

유효. 근거: `bin-api-manager/pkg/pubsubhandler/run.go` 는 `m.data` 만 WS 에 쓰고, `subscribehandler/webhookmanager.go` `processEventWebhookManagerRoutingKeyedEvent` 는 `m.Type` 을 알면서도 raw 리소스만 발행한다. `conversation_message_created` 의 `message.WebhookMessage` 는 `Owner` 가 없어 `createTopics` (api-manager) 와 `createRoutingKeys` (webhook-manager) 의 owner 게이트에서 agent 키가 만들어지지 않는다.

## 2. 설계

### 2.1 event_type 주입 (bin-api-manager)

- 위치: `processEventWebhookManagerRoutingKeyedEvent` 에서 topic 마다가 아니라 이벤트당 1회 payload 에 `"event_type": m.Type` 을 주입한 뒤 모든 topic 에 같은 문자열을 발행한다.
- 방식: JSON 주입(`map[string]json.RawMessage` 로 decode, 키 설정, encode, HTML escape 끔). `Publish` 시그니처, `message{topic,data}`, mock 은 바꾸지 않는다(시그니처 변경은 모든 호출자와 mock 을 건드리고 이득이 없음).
- payload 가 JSON object 가 아니면 원문을 그대로 발행한다(기존 동작 유지). decode 실패뿐 아니라 `null` 처럼 에러 없이 nil map 이 되는 경우도 같은 degrade 로 처리한다(nil map 대입 panic 방지). 테스트: `{}`, `null`, 배열, 스칼라, 빈 입력.
- `Encoder.SetEscapeHTML(false)` 를 쓰므로 `Encode` 가 붙이는 후행 개행을 제거해 WS 프레임에 개행이 실리지 않게 한다. 키 순서는 알파벳 정렬로 바뀌며(의미 동일) `event_type` 을 맨 앞에 두려는 의도는 없다. 테스트는 바이트가 아니라 JSON 의미 비교로 쓴다.
- 기존 키는 변경하지 않는다. 같은 이름의 기존 키가 있으면 덮어쓴다(현재 WS 리소스 페이로드에는 없음).
- 레거시 wrapped 경로(`processEventWebhookManagerWebhookPublished`)는 도달 불가이고 이미 top-level `type` 을 가지므로 건드리지 않는다.

### 2.2 conversation message 의 agent 키 (bin-webhook-manager)

owner 소스: webhook-manager 의 `publishRoutingKeyedEvent` 에서 resource 가 `conversation` 이고 eventType 이 `conversation_message_` 로 시작하는 이벤트에 한해(`conversation_message_created/updated/deleted` 3종 모두 `messagehandler` 의 Create/UpdateStatus/Delete 에서 발행되며 3종 모두 대상. 비교는 `strings.HasPrefix(eventType, "conversation_message_")`. `conversation_created/updated/deleted` 는 이미 owner 를 가지므로 RPC 없이 기존 `createRoutingKeys` 를 그대로 타며 회귀 테스트로 고정) `ConversationV1ConversationGet(conversation_id)` 로 조회한다.

- RPC 는 requesthandler 기본 timeout(3초)이 이미 걸려 있으므로 별도 context timeout 은 두지 않는다. timeout 도 조회 실패와 같이 degrade 로 처리한다(publish 경로가 `sendWebhookToActiveflow` 보다 먼저 동기로 돌기 때문).
- payload 의 `conversation_id` 가 Nil 이면 RPC 없이 degrade.
- 조건: 조회 성공, `conversation.CustomerID == payload.customer_id`(교차 고객 방어), `OwnerType == agent`, `OwnerID != Nil` 일 때만 agent 키를 만든다.
- 조회 실패, 불일치, owner 없음, owner_type 이 agent 아님: customer 키만 발행(degrade). 조회 실패는 Error 로그(`errors.Is(err, requesthandler.ErrNotFound)` 이면 삭제된 conversation 일 수 있어 Warn).
- 주입은 owner 검증을 통과한 경우에만 하고, 이벤트당 1회 만든 같은 payload 를 customer 키와 agent 키 모두에 발행한다. degrade 시에는 원문 payload 를 그대로 발행한다. 주입도 `map[string]json.RawMessage`(nil map 방어 포함)로 한다.
- topic exchange 로 나가는 payload 에만 `owner_type`, `owner_id` 를 주입한다. 이유: api-manager 는 라우팅 키를 보지 못하고 payload 만 받으며 `createTopics` 가 `CommonData.OwnerID` 로 agent 토픽을 만든다. 주입하면 api-manager 의 `createTopics` 는 변경 없이 `agent_id:<owner>:conversation:<msgID>` 와 `agent_id:<owner>:conversation:conversation_message_created:<msgID>` 를 만든다.
- 고객용 webhook HTTP 본문(`SendWebhookToCustomer` 가 보내는 `data`)과 `message.WebhookMessage`/`ConvertWebhookMessage` 는 변경하지 않는다. 주입은 topic exchange publish 직전에만 한다.
- 주입 값: `owner_id` 는 조회한 conversation 의 OwnerID, `owner_type` 은 `agent`.
- 구독 계약: square-talk 은 prefix `agent_id:<me>:conversation` 로 구독한다(pubsub 은 prefix 매칭). 신규 토픽은 `agent_id:<owner>:conversation:<msgID>` 와 `agent_id:<owner>:conversation:conversation_message_created:<msgID>` 이므로 이 prefix 에 매칭된다(테스트로 고정).
- 키: `agent_id.<owner>.conversation.<event_type>.<messageID>` (기존 `createRoutingKeys` 형식). 구독 binding 은 `agent_id.<me>.#` 라 변경 불필요.
- 구현: `webhookOwnerData` 에 `ConversationID` 추가, `resolveConversationMessageOwner`/`injectOwner` 신설(키 생성은 기존 `createRoutingKeys` 재사용). 다른 리소스의 키와 payload 는 byte 동일.

### 2.3 문서

`websocket_overview.rst` 의 Event Message Format 절(dual-shape 서술)을 `event_type` 기준 단일 shape 으로 교체. `websocket_struct.rst`, `websocket_tutorial.rst` 의 예제 payload 를 같은 기준으로 갱신(`type`/`data` wrapped 예제 제거). 서버가 보내지 않는 `event_type/topic/timestamp/data` 래퍼를 보여 주는 `quickstart_realtime.rst`(~118-150) 와 `data["data"]` 를 읽는 `quickstart_events.rst`(~66-74) 예제는 평탄 shape + `event_type` 으로 고치고, `transcribe_tutorial.rst`(~284, WS 부분만. ~387 은 고객 HTTP webhook 이라 제외), `websocket_tutorial.rst`(~93 `data.get('type')` 를 `event_type` 으로)도 맞춘다. `architecture_communication.rst`, `flow_execution_internals.rst`, `service_agent_*` 는 WS 프레임 예제가 없으므로 개별 확인 후 필요한 줄만 고친다. 고객 HTTP webhook 예제(`conversation_*`, `architecture_dataflow/sequences`)는 제외. 재검증 grep 은 `docsdev/source` 전체를 대상으로 하되 WS 맥락(구독 후 수신 예제)만 고친다. `quickstart_transcribe.rst`, `call_scenarios.rst` 의 WS 예제도 포함한다. message 프레임의 `owner_id` 는 conversation 의 owner 임을 문서에 명시한다. WS 문서에 `conversation` 토픽과 `conversation_message_created` 이벤트, message 프레임의 `owner_id`/`owner_type` 을 추가한다. OpenAPI 는 변경하지 않는다(`paths/ws/main.yaml`, `paths/service_agents/ws.yaml` 는 설명 한 줄뿐이며 프레임 shape 를 정의하지 않는다. description 만 바꾸면 생성물 diff 가 커지므로 RST 문서로 충분하다고 판단). `routingkey.go` 의 `publishRoutingKeyedEvent` doc 주석(RPC 는 chat 계열만이라는 전제)과 `bin-webhook-manager/docs/architecture.md`, `domain.md` 에 conversation message 라우팅 키와 owner 주입을 반영한다. Sphinx clean 재빌드 후 build/ 를 force-add 한다.

## 3. 영향도

- 가산 키만 추가. 리소스 기존 키 불변. 엄격 스키마 검증 클라이언트 존재 여부는 미검증(가산 키라 영향 낮다고 판단).
- 구/신 버전 혼재(롤링 배포): api-manager 만 먼저 배포되면 event_type 만 생김. webhook-manager 만 먼저 배포되면 agent 키는 생기지만 payload 에 owner 가 있어 구 api-manager 도 agent 토픽을 만든다. 순서 무관.
- 부하: conversation message 이벤트(created/updated/deleted)당 RPC 1회(webhook-manager 에서 1회, pod 수와 무관하나 이벤트 수에 비례).
- DB 스키마 변경 없음.
- 동기 RPC: `publishRoutingKeyedEvent` 는 동기이므로 conversation-manager 지연이 topic publish 와 `sendWebhookToActiveflow` 시작을 지연시킨다. 고객 HTTP 전송은 별도 goroutine 이라 영향 없음. chat 의 `TalkV1ParticipantList` 와 같은 선례라 수용한다.
- 같은 topic exchange 의 다른 consumer: timeline-manager 는 `#` 로 바인딩해(`bin-timeline-manager/pkg/subscribehandler/main.go:116`) 모든 키 복사본을 적재하므로 owner 가 있는 conversation message 이벤트가 1행 더 쌓이고 payload 에 `owner_id`/`owner_type` 이 들어간다(owner 가 있는 다른 리소스와 같은 기존 동작의 연장, 수용). peer_events 는 publisher 게이트로 해당 없음. agent-manager 는 default 에서 무시.
- 중복 프레임: customer 범위와 agent 범위를 같은 pod 에서 구독하면 같은 프레임이 두 번 나갈 수 있다(owner 가 있는 다른 리소스에 이미 있는 동작). 서버 중복 제거는 하지 않고 frontend reducer 멱등에 의존한다(범위 밖).
- WS 로 나가는 message payload 에 `owner_id`/`owner_type` 이 추가된다. customer 토픽은 같은 고객 관리자만, agent 토픽은 본인만 구독하며 `conversation.WebhookMessage` 가 이미 Owner 를 노출하므로 새 노출 대상은 없다. agent 용과 customer 용 payload 를 분리하는 대안은, 고객 관리자가 `conversation` 이벤트로 이미 Owner 를 보며 분리하면 키마다 payload 가 달라 복잡도만 늘어 채택하지 않는다. conversation-manager 가 message 이벤트에 owner 를 직접 싣는 대안은 customer webhook 계약 변경이라 기각.
- 조회 시점과 발행 시점 사이에 owner 가 바뀌면 이전 owner 에게 갈 수 있다(범위 밖의 재할당 항목과 같은 계열, 수용).
- frontend(square-talk `handleMessage`, reducer 멱등, 테스트 재작성)는 별도 frontend 카드 소관이며 서버 계약만으로 SQUARE-66 이 닫히지는 않는다.

## 4. 검증 기준

- api-manager: 주입 단위 테스트(object, 비 object, 기존 키 덮어쓰기, 비 ASCII 와 HTML 문자 보존), routing-keyed 이벤트 테스트에서 모든 topic 에 같은 event_type 이 실림, `createTopics` 가 owner 가 있는 `conversation_message_created` 에 agent 토픽 2개를 만듦.
- webhook-manager: agent owner, owner 없음, owner_type 불일치, 조회 실패, 고객 불일치 케이스의 키와 payload, 고객 HTTP 본문 불변, 다른 리소스 payload 불변.
- `json:"event_type"` grep 으로 WS 대상 리소스 충돌 없음을 확인(구현 시 재실행).
- `Test_ConvertWebhookMessage_NeverCopiesCaseID` 류 테스트는 그대로 통과.

## 5. 범위 밖 (보고만)

이전 owner 에게 재할당 이벤트 미전달, `SendWebhookToCustomer` account 조회 실패 시 routing 발행 생략, chat 이벤트 중복 프레임(서버 중복 제거 안 함, frontend reducer 멱등으로 처리), 구독 prefix 변경.
