# VOIP-1567 extension 자격 증명 로그 노출 설계

- 상태: Draft (설계 리뷰 2회차 대기, 1회차 반영)
- 티켓: VOIP-1567
- 선행 문서: docs/plans/2026-10-07-VOIP-1567-extension-credential-log-exposure-analysis.md (이슈 분석, 범위와 결정 확정)
- 범위: 분석 문서 4절의 유형 1(extension, trunk 객체 5곳)과 유형 2(`AuthIdentity` 단일 지점). 유형 3, 4 는 대표님 결정으로 처리하지 않는다.

## 1. 목표와 비목표

목표.

- extension, trunk 의 `password`, `direct_hash` 가 로그에 나오지 않게 한다(유형 1).
- 인증된 요청 로그에 agent `direct_hash` 와 direct scope `hash_fingerprint` 가 나오지 않게 한다(유형 2).
- 로그의 키 구조는 가능한 한 그대로 유지해, 로그를 읽는 쪽(수동 조회, 외부 대시보드)의 변화를 최소화한다.

비목표.

- 유형 3, 4 처리, 일반 redaction 훅이나 CI 게이트, 로그 레벨 정책 변경, 이미 기록된 로그의 정리(분석 문서 6절).
- API 응답, DB, RPC 계약 변경. 이 변경은 로그 출력에만 영향을 준다.

## 2. 결정 사항

### D1. 유형 1: 객체 통째 로그를 명시 필드 로그로 교체 (5곳)

객체를 `WithField` 로 넘기지 않고, 비밀이 없는 필드만 `WithFields` 로 명시한다. 메시지 문자열은 바꾸지 않는다.

| 위치 | 현재 | 변경 |
|---|---|---|
| bin-api-manager/pkg/servicehandler/extension.go:52 (`extensionGet`) | `log.WithField("tag", res).Debug("Received result.")` | `log.WithFields(logrus.Fields{"customer_id": res.CustomerID, "extension": res.Extension}).Debug("Received result.")` (이 함수의 `log` 에는 이미 요청 `id` 로 `extension_id` 키가 있으므로 다시 넣지 않는다) |
| bin-call-manager/pkg/callhandler/start_incoming_domain_type_sip.go:134 | `log.WithField("extension", ext).Debugf(...)` | extension 은 위와 같은 세 필드 |
| bin-call-manager/pkg/callhandler/start_incoming_domain_type_registrar.go:355 (`tmp`), :383 (`ext`) | `log.WithField("extension", ...).Debugf(...)` | 위와 같은 세 필드 |
| bin-call-manager/pkg/callhandler/start_incoming_domain_type_trunk.go:44 | `log.WithField("trunk", trunk).Debugf(...)` | `trunk_id`, `customer_id`, `domain_name` |

- `extension.go:52` 의 키 `"tag"` 는 tag 핸들러에서 복사된 잔재다. 객체를 더 이상 넘기지 않으므로 키를 사용하지 않고 위 필드로 대체한다.
- 5곳 모두 오류 반환 처리 뒤에서만 객체를 읽는다. 기존 코드도 같은 위치에서 객체를 로그에 넘기거나 `.ID` 를 참조하므로 nil 위험이 늘지 않는다. `extension.go:52` 의 메시지는 `.ID` 를 참조하지 않지만, 오류가 없으면 RPC 응답은 non-nil 이다.
- 디버깅 손실: 메시지에 이미 id 가 있고, extension 번호와 도메인을 남기므로 손실이 작다. realm, username 등 비밀이 아닌 필드는 필요하면 DB 와 registrar 조회로 얻는다.
- 로그 호출 제거(방안 B)는 채택하지 않는다.

### D2. 유형 2: `AuthIdentity` 에 `MarshalJSON` 구현 (한 지점)

`bin-api-manager/models/auth/auth.go` 의 `AuthIdentity` 에 `MarshalJSON` 을 구현해 비밀 필드를 뺀 출력만 만든다. 호출 지점은 바꾸지 않는다.

- 수신자는 값 수신자(`func (a AuthIdentity) MarshalJSON()`)로 한다. 포인터와 값 모두에 적용되고, nil 포인터는 `encoding/json` 이 `null` 로 처리한다.
- 출력은 기존 구조를 유지하고 비밀 필드만 뺀다. 최상위 키는 기존과 같은 `Type`, `CustomerID`, `Agent`, `Accesskey`, `DirectScope`, `DelegateScope` 이고, 값이 nil 인 멤버는 기존 출력과 같이 `null` 로 둔다.
- 멤버별 출력 필드.
  - `Agent`: `id`, `customer_id`, `username`, `name`, `status`, `permission`. 제외: `direct_hash`, `direct_id`, `addresses`, `tag_ids`, `ring_method`, `detail`, 시각 필드. 로그에 필요한 식별과 권한 판단 정보만 남긴다.
  - `Accesskey`: `id`, `customer_id`, `token_prefix`. 제외: `raw_token`(생성 시에만 채워짐), 이름과 상세, 시각.
  - `DirectScope`: `customer_id`, `resource_type`, `resource_id`, `allowed_resource_types`, `allowed_resource_id`, `direct_id`, `boot_expire`, `scope_version`. 제외: `hash_fingerprint`(원문 hash 는 아니지만 방어적으로 제외).
  - `DelegateScope`: 전체(`customer_id`, `issued_by`, `jti`). 비밀 값이 아니라 토큰 식별자다.
- 구현은 이 용도의 비공개 보조 구조체를 쓰는 방식이 단순하다. 기존 모델을 복제하지 않고, 필요한 필드만 옮겨 담는다.
- 이 타입은 JSON 직렬화, 캐시, gob, 테스트 비교 어디에도 쓰이지 않고(분석 문서 2절), 다른 서비스가 import 하지 않으므로 부작용이 없다. JWT 에 들어가는 것은 `AuthIdentity` 가 아니라 `Agent` 와 `DirectScope` 이므로 영향이 없다.
- 이 Marshaler 는 `json.Marshal` 과 로그 포매터에만 영향을 준다. `%v`, `%+v` 로 `AuthIdentity` 를 직접 찍는 경우는 적용되지 않는다. 구현 단계에서 그런 호출을 실코드로 한 번 더 조회해 결과를 PR 에 적는다. 있다면 같은 변경으로 `%v` 인자를 바꾼다.

### D3. 테스트

- `models/auth/auth_test.go`: 마샬 결과 검증.
  - 각 타입(agent, accesskey, direct, delegate)별로 비밀 값(`direct_hash`, `raw_token`, `hash_fingerprint`)이 문자열로 출력에 없다.
  - 허용 필드는 출력에 있다. 최상위 키 구조가 기존과 같다.
  - nil 멤버는 `null`, nil 포인터는 `null`, 값 복사본도 같은 결과.
  - `password_hash` 가 계속 출력에 없다.
  - logrus 로그 포매터(joonix)로 `WithField("agent", identity)` 를 기록한 결과에도 같은 속성이 성립한다.
- `extension_test.go`: `extensionGet` 에 대해 logrus test hook(`logrus/hooks/test`)으로 항목을 수집한다. 항목의 `Data` 값 중 `*Extension` 이 없고, JSON 포매팅한 결과에 비밀 값이 없다.
- call-manager 4곳: 각 함수의 기존 테스트에 같은 hook 단언을 추가한다. 기존 테스트가 해당 로그 줄에 도달하지 않으면 도달하는 케이스를 추가한다. 케이스를 새로 만들 정도로 무거우면 그 곳의 단언은 생략하고 PR 본문에 사유를 적는다(변경이 로그 호출 한 줄이라 리뷰와 변이 시험 결과로 갈음한다).
- 변이 시험: 수정 전 코드로 되돌렸을 때 위 테스트가 모두 실패해야 한다. 구현 단계에서 5곳과 Marshaler 각각을 되돌려 확인하고 PR 에 결과를 적는다.

### D4. 배포와 호환

- 배포 순서 제약이 없다. api-manager 와 call-manager 를 각각 배포하면 된다.
- 외부 계약(API, 이벤트, DB) 변경이 없다.
- 로그 출력 형태 변경: 유형 1 은 `tag`, `extension`, `trunk` 필드의 내용이 객체에서 id, 번호 요약으로 줄어든다. 유형 2 는 `agent`, `auth`, `auth_identity` 필드 안에서 위 제외 필드가 사라진다. 저장소 안에는 이 필드를 파싱하는 알림이나 대시보드가 없다(확인함). 저장소 밖의 Grafana 쿼리에 대해서는 확인하지 못했으므로 위험으로 기록한다.

### D5. 이미 기록된 값과 수용 위험

- 이미 로그에 남은 password 와 direct_hash 는 확인하지 않고, 교체 여부는 대표님이 별도로 판단한다(분석 문서 6절). 이번 PR 에 포함하지 않는다.
- 유형 3, 4 는 처리하지 않으며, 로그인, agent 조회, 통화 경로의 agent `direct_hash` 와 direct 통화의 `hash` 가 계속 기록된다(분석 문서 4절의 수용 위험). PR 본문에 같은 내용을 적는다.

## 3. 변경 파일 목록

- bin-api-manager/models/auth/auth.go (Marshaler), auth_test.go
- bin-api-manager/pkg/servicehandler/extension.go, extension_test.go
- bin-call-manager/pkg/callhandler/start_incoming_domain_type_sip.go, start_incoming_domain_type_registrar.go, start_incoming_domain_type_trunk.go 와 각 `_test.go`
- docs/plans 의 분석 문서와 이 설계 문서
- 사용자 가시 문서(`docsdev`), OpenAPI, 생성물: 변경 없음(API 계약 변경 없음)

## 4. 위험

- Marshaler 가 향후 `AuthIdentity` 를 다른 목적으로 직렬화하려는 코드와 충돌할 수 있다. 완화: 타입 주석에 "JSON 출력은 로그용 요약이며 역직렬화용이 아니다"를 명시한다. 현재 그런 코드는 없다.
- logrus `TextFormatter` 는 필드 값에 `MarshalJSON` 을 호출하지 않고 `%v` 로 출력한다. 운영은 JSON 포매터(joonix, `internal/config/main.go` 의 `initLog`)이므로 영향이 없다. 로컬에서 텍스트 포매터로 실행하면 구조체가 기존처럼 출력될 수 있다.
- 새 필드가 `AuthIdentity` 에 추가될 때 Marshaler 를 갱신하지 않으면 새 필드는 로그에 나오지 않는다. 비밀을 가리는 방향으로 안전한 실패(허용 목록 방식)이므로 수용한다.
- 같은 종류의 실수가 다른 곳에서 반복될 위험은 유형 3, 4 와 함께 수용 위험으로 남는다. 새 도구는 만들지 않는다.
