# VOIP-1567 extension 자격 증명 로그 노출 이슈 분석

- 상태: Draft (이슈 분석 리뷰 대기)
- 티켓: VOIP-1567 (원 제목의 [registrar-manager] 는 오기, 실제 대상은 api-manager 와 call-manager)
- 기준 코드: origin/main 163952de1

## 1. 결론 요약

- 이슈는 유효하다. 가능성 확인이 아니라 실재하는 노출이다. extension 과 trunk 객체를 통째로 로그에 남기는 곳이 5곳 있고, 이 객체의 `password` 와 `direct_hash`(trunk 는 `password`)가 평문으로 JSON 로그에 출력된다.
- 운영에서 Debug 로그가 켜져 있다. api-manager(`internal/config/main.go:253`)와 call-manager(`internal/config/main.go:124`)가 로그 레벨을 조건 없이 `DebugLevel` 로 고정한다.
- 지금 진행하는 것이 타당하다. 수정이 작고 위험이 낮으며, 코드 변경 없이는 노출이 계속 쌓인다. 특히 call-manager 는 해당 extension 으로 오는 통화마다 기록한다.
- 우선순위는 티켓의 "중간"에서 P2 로 상향한다. 티켓이 정한 상향 조건(password 가 로그에 남는 경우)을 충족한다.

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

## 3. 영향 평가

- 노출 자원: extension 의 SIP password 와 direct_hash, trunk 의 SIP password.
- 노출 위치: 로그 수집 경로(Loki 등). 접근 주체는 로그 열람 권한이 있는 내부 운영자다. 외부 공격자가 이 경로로 직접 읽는 것은 아니다.
- 규모: 코드상 api-manager 는 extension 을 조회, 수정, 삭제할 때마다, call-manager 는 extension 또는 trunk 로 들어오는 통화마다 기록한다. 이미 쌓인 값이 있을 것으로 추정한다.
- 이미 기록된 값의 잔존 여부는 측정하지 못했다. 로그 보존 기간과 실제 잔존 건수는 확인하지 않았다. 확인에는 인프라 시크릿 복호화가 필요해 이번 분석에서는 접근하지 않았다. 코드상 추정과 측정은 구분해 둔다.

## 4. 범위

수정 대상(위 5곳, 같은 종류의 결함).

- 5곳 모두 같은 방식(객체 통째 로그)이므로 한 PR 에서 처리한다.

범위 밖(이번에 하지 않음).

- agent 객체 통째 로그(`WithField("agent", ...)` 약 14곳). agent 의 `direct_hash` 가 노출되지만 SIP password 와 달리 영향이 작고 산발적으로 퍼져 있다. 실측 신호가 생기면 별도로 다룬다.
- 일반적 로그 redaction 훅이나 CI 게이트. 같은 실수가 반복된다는 신호가 생기면 그때 설계한다(과잉 설계 방지).
- 로그의 `Debug` 고정 자체(운영 로그 레벨 정책). 별개 주제다.

## 5. 해결 방향 (설계 단계에서 확정할 사항)

- 방안 A (권장): 5곳의 객체 통째 로그를 비밀이 없는 필드만 명시한 로그로 바꾼다. 메시지에는 이미 id 가 있으므로, extension 은 `extension_id`, `customer_id`, extension 번호, trunk 는 `trunk_id`, `domain_name` 정도로 한정한다. 새 코드나 공용 도구가 필요 없다.
- 방안 B: 로그 호출 자체를 제거한다. 가장 단순하지만 디버깅 단서가 줄어든다.
- 방안 C: 공용 redaction 도구 신설. 이번에는 기각한다(위 범위 밖 사유와 같다).
- 재발 방지 테스트: 대상이 로그 호출이라 단위 테스트가 어렵다. 로그 출력에 비밀 값이 없는지 확인하는 테스트를 5곳에 두는 것이 가능한지, 비용 대비 가치가 있는지는 설계 단계에서 판단한다.

## 6. 대표님 결정이 필요한 사항 (코드 범위 밖)

- 이미 로그에 남은 password 의 처리. 코드 수정으로는 과거 로그가 사라지지 않는다. 로그 보존 기간을 확인하고, 필요하면 extension 과 trunk password 교체 여부를 정해야 한다. 이번 PR 에 포함하지 않는다.

## 7. 진행 판단

- 진행하는 것이 타당하다. 유효성은 재현으로 확인됐고, 범위는 5곳으로 작으며, 방안 A 는 되돌리기 쉽다. 의존성이 없다.
