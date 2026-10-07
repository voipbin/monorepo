# VOIP-1508 direct hash 로그 노출 이슈 분석

- 상태: DRAFT rev3 (리뷰 1, 2회차 Request Changes 반영, 재리뷰 대기). 진행은 4절의 대표님 확인을 조건으로 한다
- 티켓: VOIP-1508 (원 제목은 api-manager AuthBoot 과 agent-manager 두 곳이나, 실제 범위는 그보다 넓다)
- 기준 코드: origin/main 49ec4a06d
- 관련: VOIP-1567 (같은 날 머지). VOIP-1567 분석 문서의 "유형 4"(Direct 객체와 hash 문자열 로그, 약 27곳, 9개 서비스)가 이 티켓과 동일한 대상이며, 그 PR 에서 대표님 결정으로 범위 밖 처리되었다("처리하지 않는다, 별도 티켓도 만들지 않는다, 재론하지 않는다").

## 1. 결론 요약

- 이슈는 유효하다. 티켓 설명이 가리킨 코드는 현재 main 에서도 그대로 있다. 다만 줄 번호는 이동했다(`boot.go:146` 은 현재 `boot.go:144`).
- 티켓이 지목한 2곳(api-manager `AuthBoot`, agent-manager `direct_hash.go`)은 일부일 뿐이다. `*direct.Direct`(필드 `Hash`, 태그 `json:"hash"`)를 `WithField("direct", d)` 로 통째로 기록하는 곳이 9개 서비스에 걸쳐 있다.
- 티켓의 "다른 bin-*-manager 감사" 항목이 바로 이 확장이다. 감사 결과 범위는 아래 2절이다.
- 운영에서 Debug 로그가 켜져 있다는 전제는 VOIP-1567 분석에서 확인되었다(api-manager `internal/config/main.go:253`, call-manager `internal/config/main.go:124` 의 무조건 DebugLevel).
- 진행하는 것이 타당하다. 수정은 로그 호출 지점의 필드 축소이고 되돌리기 쉽다. API, 이벤트, DB 계약 변경이 없다.

## 2. 코드 근거 (origin/main 기준 직접 확인)

`Direct` 모델(`bin-direct-manager/models/direct/direct.go`)은 `Hash string json:"hash" db:"hash"` 를 가지며 redaction(`String()`, `MarshalJSON`)이 없다. 티켓 코멘트 확인대로 Hash 외의 추가 민감 필드는 없다(`Identity`, `ResourceType`, `ResourceID`, `Hash`, `TMCreate`, `TMUpdate`). 이 필드는 API 응답으로 나가야 하므로 `json:"-"` 는 불가하다.

### 2.1 Direct 객체 전체 로그 (WithField("direct", ...))

| 서비스 | 위치 |
|---|---|
| bin-api-manager | pkg/servicehandler/boot.go:144 (티켓 지목) |
| bin-agent-manager | pkg/agenthandler/direct_hash.go:38,47 (hash 문자열도 함께), db.go:98 |
| bin-ai-manager | pkg/aihandler/direct_hash.go:45, db.go:57, pkg/teamhandler/direct_hash.go:45, handler.go:50 |
| bin-conference-manager | pkg/conferencehandler/direct_hash.go:45, conference.go:93 |
| bin-flow-manager | pkg/flowhandler/direct_hash.go:37,46, db.go:118 |
| bin-queue-manager | pkg/queuehandler/direct_hash.go:38,47, create.go:67 |
| bin-webchat-manager | pkg/widgethandler/create.go:50, db.go:133 (객체 로그만, hash: %s 메시지 없음) |
| bin-call-manager | pkg/callhandler/start_incoming_domain_type_sip.go:94 |
| bin-direct-manager | pkg/directhandler/handler.go:76,116,137,240,271, db.go:66 (`"direct": d` 가 base logger 필드로 묶임) |

위 중 `direct_hash.go` 계열 9행(6개 파일: agent 38,47, aihandler 45, teamhandler 45, conference 45, flow 37,46, queue 38,47)은 메시지에 `hash: %s` 로 원문 hash 를 한 번 더 찍는다(티켓이 말한 "더 직접적인" 경우가 agent-manager 뿐이 아니라 ai, team, conference, flow, queue 모두에 있다).

### 2.2 hash 가 필드 값, 메시지, URI, 응답 본문으로 찍히는 곳 (객체 로그가 아님)

- bin-call-manager `pkg/callhandler/start_incoming_domain_type_sip.go:81,83`: base logger 에 `"hash": hash` 를 묶고 `Debugf("Starting direct call handler. hash: %s", hash)` 로 원문을 찍는다. direct 통화마다 발생하는 가장 빈번한 경로다(VOIP-1567 분석 문서가 유형 4 의 대표 사례로 든 지점).
- bin-direct-manager `pkg/directhandler/handler.go:109-110`(`GetByHash`), `db.go:49-50`(`dbGetByHash`): base logger 에 `"hash": hash` 를 묶어 해당 함수의 모든 로그(Error 포함)에 원문이 실린다.
- bin-direct-manager `pkg/listenhandler/v1_directs.go:120-126`(`processV1DirectsByHashGet`): `"hash": hash` 필드와 `WithField("request", m)`(URI `/v1/directs/by-hash/<hash>` 포함).
- bin-direct-manager `pkg/listenhandler/main.go:153-155`: 모든 요청에 대해 `"request": m` 필드와 `Received request. ... uri: %s` 를 기록한다. by-hash 조회의 URI 에 hash 가 남는다.
- bin-direct-manager `pkg/listenhandler/main.go:222-227`: `"response": response` 필드를 `Sending response` 와 함께 기록한다. `sock.Response.Data` 는 `json.RawMessage`(`bin-common-handler/models/sock/message.go:25`)이고 create, get, by-hash, regenerate, list 응답 본문에 `Direct` JSON(hash 포함)이 들어간다. 따라서 URI 마스킹만으로는 부족하고 요청과 응답 로그 필드까지 다뤄야 한다.
- bin-direct-manager `pkg/listenhandler/v1_directs.go:62,100,138`: 마샬 실패 경로의 `Debugf("... message: %v", tmp)` 가 `*direct.Direct` 를 `%v` 로 찍는다(드문 경로). `:38,88,126,157` 의 `WithField("request", m)` 중 hash 를 담는 것은 by-hash 인 `:126` 뿐이고, 나머지는 ID URI 다.
- bin-api-manager 는 이미 잘라서 쓴다(`boot.go:130` 사용, `truncateHash` 정의 `boot.go:242-247`, `lib/service/boot.go:36-41`). 변경 불필요. 단 `truncateHash` 는 앞 12자 + `...` 이고 hash 는 `direct.` + hex 이므로 hex 는 일부(5자 안팎)만 남는다. 이 수준을 허용 기준으로 삼는다.

### 2.2.1 hash 가 다른 객체에 내장되어 나가는 경로 (call-manager channel, 유형 5)

- direct 통화에서는 `cn.DestinationNumber` 가 `direct.<hash>` 다(`start_incoming_domain_type_sip.go:31-32`). `channel.Channel.DestinationNumber` 는 `json:"destination_number"`(`bin-call-manager/models/channel/main.go:62`)이고 redaction 이 없다.
- 따라서 `WithField("channel", cn)` 류의 객체 로그가 같은 통화마다 hash 를 남긴다. 확인된 지점: `channelhandler/db.go:69`, `arieventhandler/ari_channel.go:44`, `arieventhandler/ari_stasis.go:29,36`, `callhandler/hangup.go:192`. call-manager 전체의 `WithField("channel", ...)` 는 14건이며 direct 통화가 아닌 경우 `DestinationNumber` 는 일반 번호다. `ari_channel.go:166` 의 ARI 이벤트(`Dialplan.Exten` 포함 가능)는 미검증이다.
- 이 경로는 hash 를 담은 값이 일반 전화번호 필드와 같은 칸에 들어 있어 "hash 필드를 제거" 방식으로 고칠 수 없고, channel 로그 전체 재설계가 필요하다. 이번 범위에서 제외한다(4절).

### 2.3 이미 안전한 곳 (음성 확인 포함)

- `bin-common-handler` 의 requesthandler(`direct_directs.go`, `send_request.go`), sockhandler, notifyhandler(`publish.go`, event data 미기록)에는 hash 로그가 없다. direct-manager 이벤트(`direct_created` 등)를 구독하는 서비스는 없는 것으로 보이나 별도 grep 은 하지 않았다. registrar-manager 는 `WithField("direct_id", d.ID)` 만 쓴다.

- `DirectScope.HashFingerprint` 는 JWT 서명 키로 키잉한 HMAC 파생값이라 원문이 아니다(VOIP-1501, `boot.go:183` 의 `"direct": scope`).

## 3. 영향 평가

- direct hash 는 위젯 페이지 JS 에 노출되는 공개 링크 값이라 그 자체가 비밀은 아니다. 그러나 `POST /auth/boot` 에 hash 하나를 주면 해당 리소스 범위 토큰이 발급되므로(`AuthBoot`), 로그 열람 권한자가 전 고객사의 hash 를 한 곳에서 모을 수 있다는 비대칭이 실제 위험이다. 티켓 판단과 동일하다.
- 노출 빈도: `/auth/boot` 호출마다(2.1 api-manager), direct 통화마다(call-manager), direct-manager 의 모든 생성, 조회, 재생성, 삭제, by-hash 조회마다, 각 리소스의 direct hash 생성과 재생성마다.
- 이미 기록된 값의 잔존 여부는 측정하지 않았다. 코드 추정이며 측정이 아니다.

## 4. 범위 판단 (대표님 확인 필요 사항 포함)

결정 이력과의 관계. VOIP-1567 분석 문서 4절은 "유형 3과 4는 처리하지 않는다. 별도 티켓도 만들지 않는다. 확정이며 재론하지 않는다"고 기록했다. 유형 4 가 바로 이 티켓의 대상이다. VOIP-1508 은 그보다 이전부터 있던 티켓이라 결정 당시 함께 고려되었는지는 불확실하다(1567 문서에 1508 언급 없음). 따라서 이번 PR 은 사실상 그 결정을 다시 여는 것이며, **대표님의 명시적 재승인이 필요하다.** 승인 전에는 설계 이후 단계로 진행하지 않는다. 유형 3 은 1567 결정 그대로 범위 밖이다.

실효 평가. 유형 3(부모 리소스 객체 로그)이 남으면 `direct_hash` 는 로그인과 통화 경로에서 계속 쌓인다. 그러므로 이번 수정은 "hash 가 로그에서 완전히 사라진다"를 보장하지 않으며, 로그 열람자가 hash 를 수집할 가능성(3절의 위험)도 해소하지 못한다. 이번 PR 의 가치는 direct 모델과 hash 문자열 경로의 직접 노출 제거에 한정된다. 유형 5(call-manager channel 객체, 2.2.1)도 같은 이유로 남는다. 대표님 판단 재료로 제시한다. 이번 수정이 닫는 것은 direct 모델과 hash 문자열 경로(요청마다, 통화마다, direct-manager 전 경로)이고, 이 중 call-manager `:81,83` 과 direct-manager 요청, 응답 로그를 포함하지 않으면 실효가 더 낮아지므로 반드시 함께 닫는다. 단 같은 direct 통화의 channel 객체 로그(유형 5)는 남는다.

이번 수정 대상(제안).

- 2.1 의 전 지점: `*direct.Direct` 통째 로그를 비밀이 없는 필드(`direct_id`, `resource_type`, `resource_id`, `customer_id`)로 교체. 메시지의 `hash: %s` 는 제거한다.
- 2.2 전 지점: call-manager `:81,83` 은 hash 를 앞 12자 + `...` 로 줄인다(api-manager `truncateHash` 패턴). direct-manager 의 base logger 필드는 동일하게 줄이고, `listenhandler/main.go` 의 요청과 응답 로그(`request`, `response` 필드, URI)는 hash 가 포함되지 않도록 설계 단계에서 방식을 확정한다(예: by-hash URI 만 마스킹하고 응답 본문은 로그에서 제외).
- 재발 방지: 지점별 로그 훅 회귀 테스트(VOIP-1567 과 같은 방식, 센티널 값 + 수집 건수 1 이상 요구).

범위 밖(제안, 대표님 확인 필요).

- 부모 리소스(agent, ai, team, flow, queue, conference, extension, widget)가 `DirectHash` 필드를 가진 채 통째로 로그되는 경우. VOIP-1567 분석의 "유형 3" 과 같은 계열이며 해당 PR 에서 대표님이 처리하지 않기로 결정한 항목이다. 이번 티켓이 닫혀도 이 경로로 hash 가 계속 남는다는 점을 수용 위험으로 기록한다. 규모는 서비스별 `WithField("<리소스>", <구조체>)` 형태가 conference 12, ai 7, agent 6, queue 6, flow 3, webchat 1 이상(개략치, 서비스별 `WithField("<리소스명>", <변수>)` grep). api-manager 는 단일 문자 변수 패턴이 `AuthIdentity` 와 섞여 정확한 수를 내지 못해 제외했고, 호출 형태가 다양해 정밀 감사는 하지 않았다이다. 처리하려면 별도 설계가 필요하다.
- 일반 로그 redaction 훅이나 CI 게이트 신설. 오버엔지니어링 회피 원칙에 따라 하지 않는다.
- 로그 레벨 Debug 고정 정책.
- 이미 로그에 남은 hash 의 재생성(rotate) 여부. 대표님이 판단할 사안이며 이번 PR 에 포함하지 않는다.

## 5. 해결 방향 (설계 단계에서 확정)

- 각 지점을 개별 수정한다. `Direct` 에 `MarshalJSON` 을 넣는 안은 기각한다. 이 모델은 RPC, 캐시(`direct:hash:*` 키 값), 이벤트로 직렬화되므로 redaction 을 넣으면 기능이 깨진다.
- 필드는 allow list 로 한정한다(`d.ID`, `d.CustomerID`, `d.ResourceType`, `d.ResourceID`).
- 필드 이름은 VOIP-1291, VOIP-1567 의 ID-only 선례와 맞춘다.

## 6. 진행 판단

- 진행하는 것이 타당하다. 유효성은 코드로 확인되었고, 수정은 소규모 기계적 변경이며 외부 의존성이 없다. 티켓의 범위(구조체 전체 로그 제거, 다른 서비스 감사)에 정확히 부합한다.
