# VOIP-1567 extension 자격 증명 로그 노출 이슈 분석

- 상태: Draft (이슈 분석 리뷰 4회차 대기, 3회차 반영)
- 티켓: VOIP-1567 (원 제목의 [registrar-manager] 는 오기, 실제 대상은 api-manager 와 call-manager)
- 기준 코드: origin/main 163952de1

## 1. 결론 요약

- 이슈는 유효하다. 가능성 확인이 아니라 실재하는 노출이다. 자격 증명 필드를 가진 객체를 통째로 로그에 남기는 곳이 있고, 평문으로 JSON 로그에 출력된다. 확인된 노출은 네 유형이다.
  - 유형 1: extension 과 trunk 객체(`password`, `direct_hash`). 5곳. 이번 범위.
  - 유형 2: 인증된 요청의 `*auth.AuthIdentity`(안의 agent `direct_hash`). api-manager 에서 약 420회, 거의 모든 인증된 API 핸들러가 시작부에서 기록한다. 이번 범위.
  - 유형 3: `*amagent.Agent` 직접 로그(`direct_hash`). 14곳(api-manager 3, agent-manager 7, call-manager 4). 범위 밖(대표님 결정).
  - 유형 4: `Direct` 객체와 hash 문자열 로그(`hash`, 공개 링크 비밀 값). 약 27곳, 9개 서비스. 범위 밖(대표님 결정).
- 운영에서 Debug 로그가 켜져 있다. api-manager(`internal/config/main.go:253`)와 call-manager(`internal/config/main.go:124`)가 로그 레벨을 조건 없이 `DebugLevel` 로 고정한다.
- 지금 진행하는 것이 타당하다. 수정이 작고 위험이 낮으며, 코드 변경 없이는 노출이 계속 쌓인다. 유형 1 은 call-manager 가 해당 extension 이나 trunk 로 오는 통화마다, 유형 2 는 api-manager 가 인증된 요청마다 기록한다.
- 우선순위는 티켓의 "중간"에서 P2 로 상향한다. 티켓이 정한 상향 조건(password 가 로그에 남는 경우)을 충족한다. `direct_hash` 는 VOIP-1566 에서 `password` 와 같은 급의 자격 증명 노출로 다룬 값이다.
- 정정: 이 문서의 이전 판은 agent 객체 로그를 "약 14곳, 영향 작음, 산발적" 으로 적었으나 사실과 달랐다. 그 수치는 유형 3 에 해당하고, 더 큰 유형 2 와 유형 4 를 놓쳤다. 리뷰의 지적으로 정정했다.

## 2. 코드 근거 (origin/main 기준 직접 확인)

노출 지점 5곳.

| 위치 | 로그 호출 | 객체 | 노출 필드 |
|---|---|---|---|
| bin-api-manager/pkg/servicehandler/extension.go:52 (`extensionGet`) | `log.WithField("tag", res).Debug(...)` | `*rmextension.Extension` | password, direct_hash |
| bin-call-manager/pkg/callhandler/start_incoming_domain_type_sip.go:134 | `log.WithField("extension", ext).Debugf(...)` | extension | password, direct_hash |
| bin-call-manager/pkg/callhandler/start_incoming_domain_type_registrar.go:355 | `log.WithField("extension", tmp).Debugf(...)` | extension | password, direct_hash |
| bin-call-manager/pkg/callhandler/start_incoming_domain_type_registrar.go:383 | `log.WithField("extension", ext).Debugf(...)` | extension | password, direct_hash |
| bin-call-manager/pkg/callhandler/start_incoming_domain_type_trunk.go:44 | `log.WithField("trunk", trunk).Debugf(...)` | trunk | password |

- 모델의 JSON 태그에 비밀 필드가 그대로 있다. `Extension.Password` 는 `json:"password"`, `Extension.DirectHash` 는 `json:"direct_hash"`, `Trunk.Password` 는 `json:"password"` 이고, 어느 모델에도 로그용 redaction(`String()`, `MarshalJSON`)이 없다.
- 재현으로 확인했다. api-manager 의 joonix 포매터로 `Extension{Password:"SECRET-PASSWORD-123", DirectHash:"direct.SECRETHASH456"}` 를 위와 같은 방식으로 기록하자, 출력 JSON 에 `"password":"SECRET-PASSWORD-123"` 과 `"direct_hash":"direct.SECRETHASH456"` 가 그대로 나왔다(임시 테스트, 확인 후 삭제).
- `extensionGet` 의 호출자는 `extension.go` 의 6곳과 `serviceagent_extension.go` 의 1곳이다. 즉 extension 조회, 삭제, 수정, direct hash 재생성, provisioning token 생성이 모두 이 로그를 거친다.
- registrar-manager 자체는 이미 막혀 있다. `pkg/redacthandler`(민감 키 password, secret, token, credential 과 `*hash` 접미사)와 listenhandler 의 요청, 응답 로그 redaction 이 있다. 따라서 구멍은 RPC 응답을 역직렬화해 통째로 기록하는 호출자 쪽에만 있다.
- `redacthandler` 는 registrar-manager 의 `pkg/` 안에 있어 다른 서비스가 쓰는 공용 도구가 아니다. 키 기반 맵 redaction 이라 구조체를 그대로 받는 이 5곳에는 맞지 않는다.

유형 2, 3, 4 의 근거(리뷰 지적 후 실코드로 확인).

- 유형 2: api-manager 의 인증된 핸들러 시작부가 `log = log.WithField("agent", a)` 로 `*auth.AuthIdentity` 를 기록한다(예: `server/extensions.go:26`). 로그 키는 `agent`, `auth`, `auth_identity` 로 나뉘고, `WithField` 형태와 `WithFields` 맵 형태를 합쳐 약 420회다. `AuthIdentity`(`models/auth/auth.go:24`)의 필드에는 json 태그가 없고 `Agent`, `Accesskey`, `DirectScope`, `DelegateScope` 를 포인터로 품는다. 예외로 `pkg/servicehandler/auth.go:29` 의 `a` 는 `*amagent.Agent`(유형 3), `auth.go:82` 는 로그가 아니라 JWT 클레임 맵이다.
- 재현으로 확인했다. `NewAgentIdentity(&Agent{DirectHash:"direct.AGENTHASH789", PasswordHash:"..."})` 를 `l.WithField("agent", a).Debug(...)` 로 기록하자 출력에 `"direct_hash":"direct.AGENTHASH789"` 가 그대로 나왔고 `password_hash` 는 나오지 않았다(임시 테스트, 확인 후 삭제).
- `AuthIdentity` 는 어디에서도 JSON 직렬화, 캐시, gob 에 쓰이지 않고, 테스트에도 JSON 스냅샷 비교가 없으며, 다른 서비스가 이 타입을 import 하지 않는다(실코드 확인). 따라서 `json.Marshaler` 구현이 안전하다. `lib/service/unregister.go:110,190` 은 키 이름이 `auth_identity` 이고 같은 타입이라 함께 닫힌다.
- 유형 3: `WithField("agent", <*amagent.Agent>)` 직접 호출. 14곳. 범위 밖이므로 목록은 설계 문서에 두지 않는다.
- 유형 4: `*direct.Direct`(`bin-direct-manager/models/direct/direct.go:30` 의 `Hash string json:"hash"`) 객체와 `hash: %s` 메시지. agent-manager, ai-manager, api-manager, call-manager, conference-manager, direct-manager, flow-manager, queue-manager, webchat-manager 에 약 27곳. 범위 밖.
- `HashFingerprint`(DirectScope)는 `boot.go:29-32` 에서 키 기반 MAC 으로 파생된 값이라 원문 hash 는 아니다.

## 3. 영향 평가

- 노출 자원: extension 의 SIP password 와 direct_hash, trunk 의 SIP password, agent 의 direct_hash. direct scope 의 `hash_fingerprint` 는 원문 hash 가 아니라 키 기반 MAC 으로 파생된 값이지만, 5절에서 방어적으로 출력에서 제외한다.
- 노출 위치: 로그 수집 경로(Loki 등). 접근 주체는 로그 열람 권한이 있는 내부 운영자다. 외부 공격자가 이 경로로 직접 읽는 것은 아니다.
- 규모: 유형 2 는 인증된 API 요청마다 발생해 양이 가장 많다. 유형 1 은 extension 조회, 수정, 삭제와 extension, trunk 로 오는 통화마다 발생한다. 이미 쌓인 값이 있을 것으로 추정한다.
- 이미 기록된 값의 잔존 여부는 측정하지 못했다. 로그 보존 기간과 실제 잔존 건수는 확인하지 않았다. 확인에는 인프라 시크릿 복호화가 필요해 이번 분석에서는 접근하지 않았다. 코드상 추정과 측정은 구분해 둔다.
- `password_hash` 는 `json:"-"` 라 `WithField` 객체 로그(JSON 경로)에는 나오지 않는다(재현으로 확인). 노출되는 것은 `direct_hash` 계열이다. `pkg/servicehandler/auth.go:96` 의 `log.Debugf("... data: %v", data)` 는 `data` 가 `map[string]interface{}{"agent": a}`(`a` 는 `*amagent.Agent`)라서 맵 안의 포인터가 구조체 내용이 아니라 주소로 출력된다. 이 역시 재현으로 확인했고, 이 경로로는 `password_hash` 도 `direct_hash` 도 나오지 않는다.

## 4. 범위

대표님 결정(2026-10-07): 유형 1 과 유형 2 만 이번 PR 에서 처리하고, 유형 3 과 4 는 처리하지 않는다. 별도 티켓도 만들지 않는다. 이 결정은 확정이며 리뷰에서 재론하지 않는다.

수정 대상.

- 유형 1 (5곳): 객체 통째 로그를 비밀이 없는 필드만 명시한 로그로 바꾼다.
- 유형 2 (약 420회): 호출 지점을 하나씩 고치지 않고 `AuthIdentity` 의 JSON 출력을 한 곳에서 비밀 필드 없이 만든다. 호출 지점 변경은 0 이다.

범위 밖(이번에 하지 않음).

- 유형 3(Agent 직접 로그 14곳)과 유형 4(Direct 객체와 hash 문자열 로그 약 27곳). 대표님 결정으로 처리하지 않는다. 이로 인해 유형 3, 4 의 `direct_hash` 와 공개 링크 `hash` 가 계속 남는다는 점을 수용 위험으로 기록한다. 빈도는 낮지 않다.

  - 유형 3 은 로그인마다(`pkg/servicehandler/auth.go:29`), agent 조회마다(`pkg/servicehandler/agent.go:32,73`), agent 로 연결되는 통화마다(call-manager `start_incoming_domain_type_sip.go:313`, `start_incoming_domain_type_registrar.go:101`, `outgoing_call.go:493`, `groupcallhandler/dial.go:237`) 기록한다.
  - 유형 4 는 direct 통화마다 hash 문자열을 기록한다(call-manager `start_incoming_domain_type_sip.go:83`). 생성, 재생성 이벤트 로그가 나머지다.
  - 따라서 유형 2 를 닫아도 agent 의 `direct_hash` 는 로그인과 통화 경로에서 계속 쌓인다. 이후 처리 여부는 대표님 몫이다.
- 일반적 로그 redaction 훅이나 CI 게이트, 공용 redaction 도구 신설. 이번 수정이 가장 위험한 유형(SIP password)과 가장 빈번한 유형(요청 로그)을 닫으므로 새 도구는 만들지 않는다.
- 로그의 `Debug` 고정 자체(운영 로그 레벨 정책). 별개 주제다.

## 5. 해결 방향 (설계 단계에서 확정할 사항)

- 유형 1: 방안 A. 객체 통째 로그를 비밀이 없는 필드만 명시한 로그로 바꾼다. 메시지에는 이미 id 가 있으므로, extension 은 `extension_id`, `customer_id`, extension 번호, trunk 는 `trunk_id`, `domain_name` 정도로 한정한다. 로그 호출 제거는 디버깅 단서가 줄어 기각한다.
- 유형 2: `*auth.AuthIdentity` 에 `json.Marshaler` 를 구현해 비밀 필드를 뺀 출력을 만든다. 출력은 Type, CustomerID, agent 는 id 등 비밀이 아닌 식별 필드, accesskey 는 id 와 prefix, direct scope 는 `hash_fingerprint` 를 뺀 필드로 한정한다. 상세 필드 목록은 설계 단계에서 확정한다.
- 재발 방지 테스트: 유형 2 는 마샬 결과에 비밀 값이 없고 허용 필드가 유지되는지 단위 테스트로 검증할 수 있다. 유형 1 은 로그 호출이라 단위 테스트가 어렵다. 로그 출력을 가로채 비밀 값 유무를 확인하는 테스트가 비용 대비 가치가 있는지는 설계 단계에서 판단한다.

## 6. 대표님 결정 사항 (기록)

- 범위: 유형 1 과 2 만 처리, 유형 3 과 4 는 처리하지 않음(위 4절).
- 이미 로그에 남은 password 와 direct_hash: 코드 수정으로 과거 로그는 사라지지 않는다. 로그 보존 기간과 잔존 건수는 확인하지 않기로 했고, extension 과 trunk password 교체, direct_hash 재생성 여부는 대표님이 별도로 판단한다. 이번 PR 에 포함하지 않는다.

## 7. 진행 판단

- 진행하는 것이 타당하다. 유효성은 유형 1, 2 모두 재현 또는 코드로 확인됐다. 수정은 호출 지점 5곳(유형 1)과 `AuthIdentity` 한 지점(유형 2)으로 작고, 되돌리기 쉽다. 외부 의존성이 없다.
