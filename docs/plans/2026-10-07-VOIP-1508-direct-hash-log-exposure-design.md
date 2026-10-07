# VOIP-1508 direct hash 로그 노출 수정 설계

- 상태: DRAFT rev2 (디자인 리뷰 1회차: Approve 1, Request Changes 1 반영, 재리뷰 대기)
- 근거 문서: `2026-10-07-VOIP-1508-direct-hash-log-exposure-analysis.md` (APPROVED rev5)
- 대표님 결정(2026-10-07): VOIP-1567 의 "유형 4 범위 밖" 결정을 재승인하여 유형 4 를 이번에 처리한다. 범위는 분석 문서 2.1, 2.2 의 전 지점과 direct-manager 요청, 응답 로그다. 유형 3(부모 리소스), 유형 5(call-manager channel)는 범위 밖이며 처리 후에도 direct 통화당 hash 노출은 0이 되지 않는다.

## 1. 원칙

- 구조체 통째 로그는 allow list 필드로 바꾼다(VOIP-1291, VOIP-1567 선례). `Direct` 모델에 `MarshalJSON` 은 넣지 않는다(RPC, 캐시 `direct:hash:*`, 이벤트 직렬화가 깨진다).
- 원문 hash 가 필요한 로그는 앞 12자 + `...` 로 줄인다(api-manager `truncateHash` 와 같은 수준).
- API, 이벤트, DB 계약 변경 없음. 배포 순서 제약 없음.

## 2. 공용 헬퍼

`bin-direct-manager/models/direct/direct.go` 에 추가한다. 대상 서비스가 `replace` 지시자로 이 모듈을 참조하고 vendor 디렉터리가 없어 vendor 갱신이 없다. `LogFields` 는 메서드라 `dmdirect` 를 import 하지 않는 서비스(flow-manager)도 호출할 수 있다. `MaskHash` 를 쓰는 call-manager 와 direct-manager 는 이미 import 한다.

```go
// MaskHash returns a log-safe form of a direct hash (first 12 chars + "...").
// Values of 12 chars or fewer are returned unchanged, same as api-manager truncateHash.
func MaskHash(hash string) string

// LogFields returns the log-safe identity fields of a Direct (no hash).
func (d *Direct) LogFields() logrus.Fields
```

- `LogFields` 키: `direct_id`, `customer_id`, `resource_type`, `resource_id`. nil 수신자는 빈 Fields 를 반환한다.
- 이 패키지에 logrus import 가 추가된다. 모델 패키지가 logrus 에 의존하는 것이 부담이면 `LogFields` 를 두지 않고 호출 지점에서 4개 필드를 직접 쓴다. 호출 지점이 20곳 이상이라 중복 제거 이득이 크므로 헬퍼를 둔다(`bin-common-handler` 는 이미 모든 서비스가 logrus 와 함께 쓴다).
- `api-manager` 의 `truncateHash` 는 그대로 둔다(변경 이득 없음).

## 3. 변경 지점

### 3.1 객체 로그 (분석 2.1, `WithField` 24곳 + direct-manager `db.go:66` base logger = 25곳)

`log.WithField("direct", d)` 를 `log.WithFields(d.LogFields())` 로 바꾼다. 메시지의 `hash: %s` 는 제거한다(9행). 메시지의 `direct_id: %s` 는 이미 있으므로 유지한다. `direct-manager` 의 `db.go:66` 은 base logger 필드 `"direct": d` 를 `d.LogFields()` 로 대체한다.

### 3.2 hash 문자열 로그 (분석 2.2)

- call-manager `start_incoming_domain_type_sip.go:81,83`: `"hash": MaskHash(hash)`, 메시지 `hash: %s` 도 마스킹값.
- direct-manager `handler.go:108-110`, `db.go:49-50`, `v1_directs.go:121-126`: `"hash": MaskHash(hash)`.
- direct-manager `directhandler/db.go:94-101`(`dbUpdate`): base logger 의 `"fields": fields` 는 재생성 경로(`handler.go:212-214`)에서 `{FieldHash: <새 hash 원문>}` 이므로 update 실패 시 새 hash 가 Error 줄마다 찍힌다(충돌 재시도 시 최대 3회). `"fields"` 를 키 이름 목록(`field_keys`)으로 바꾼다.
- direct-manager `directhandler/db.go` `dbList` 의 `"filters": filters`: `FieldStruct`(`models/direct/filters.go`)가 `Hash` 필터를 허용하므로 이 필터에 hash 가 실릴 수 있다. 기록 전에 `FieldHash` 값을 `MaskHash` 로 바꾼 사본을 쓴다. 현재 저장소의 호출자가 hash 필터를 쓰는지는 확인하지 않았고, 허용되는 입력이므로 닫는다. `processV1DirectsGet` 의 `WithField("request", req)`(요청 본문에 필터가 들어 있음)도 3.3 의 요청 로그 함수로 대체한다.
- direct-manager `v1_directs.go:62,100,138,182,218,254` 의 `%v` 마샬 실패 로그: `message: %v` 를 제거하고 `direct_id` 를 쓴다(list 인 `:62` 는 건수).

### 3.3 direct-manager 요청, 응답 로그

`listenhandler/main.go:151-155, 211, 222-227` 와 `v1_directs.go:126`.

- `processRequest` 의 base logger 필드 `"request": m`(`main.go:151-153`)을 `method`, `uri`(마스킹)로 교체한다. 이 logger 가 이후 모든 줄(404, 에러 포함)에 실리므로 근본 교체다. `maskURI` 는 `processRequest` 의 모든 `m.URI` 출력에 적용한다: `main.go:155`, `:200`(`Could not find corresponded message handler`, 정규식이 method 와 무관해 GET 이 아닌 by-hash 요청도 이 분기로 와서 URI 를 남긴다), `:211`, `:227`.
- 요청 로그(그 외 핸들러의 `WithField("request", m)`): `"request": m` 대신 `method`, `uri`(마스킹), `data_type` 만 기록한다. 요청 본문은 hash 를 담지 않으나(생성은 customer_id, resource_type, resource_id) 일관성을 위해 본문은 기록하지 않는다. 디버깅 단서는 URI 와 method 로 충분하다.
- 응답 로그: `"response": response` 대신 `status_code`, `data_type` 만 기록한다. 응답 본문(`Direct` JSON)은 hash 를 담으므로 기록하지 않는다.
- URI 마스킹: `maskURI(uri string) string` 을 `listenhandler` 에 추가한다. `by-hash/` 뒤 세그먼트를 `MaskHash` 로 대체한다(`regV1DirectsByHashGet` 일치 시). 일치하지 않으면 그대로 반환한다. `main.go:211` 의 Errorf URI 도 이 함수를 쓴다.
- `v1_directs.go:126`, `:38`(`processV1DirectsGet`), 그 외 `:88,157` 의 `WithField("request", m)` 는 모두 같은 요청 로그 함수로 대체한다(`:88,157` 은 ID URI 라 hash 가 없으나 일관성을 위해 함께 교체한다).

### 3.4 범위 밖(수용 위험으로 기록)

- 에러 문자열로 전파되는 URI(`requesthandler/send_request.go:45,53,62`). 전송 실패 때만 발생하고, 막으려면 `DirectV1DirectGetByHash` 에 에러 재포장이 필요하다. 트리거 조건: 운영에서 by-hash 전송 실패 로그에 hash 가 실제로 확인되면 별도 설계. 이번에는 하지 않는다.
- 유형 3, 유형 5, 로그 레벨 정책, redaction 훅 신설.

## 4. 테스트

- `models/direct`: `MaskHash`(빈 값, 12자 이하, 초과)와 `LogFields`(hash 미포함, nil 수신자) 단위 테스트.
- direct-manager `listenhandler`: `maskURI` 단위 테스트, 요청 및 응답 로그 훅 테스트(센티널 hash 가 모든 수집 항목의 필드와 메시지에 없고, 수집 건수 1 이상을 요구해 공허한 통과를 막는다).
- 서비스별 회귀 테스트(로그 훅으로 센티널 hash 가 어디에도 없음을 확인). 대표 함수를 특정한다.

| 서비스 | 대표 함수 |
|---|---|
| agent, flow, queue | `direct_hash.go` 의 재생성과 생성 경로 양쪽 |
| ai aihandler, ai teamhandler, conference | `direct_hash.go` 의 재생성 함수 |
| webchat | `widgethandler/create.go` 와 `db.go` 재생성 |
| api-manager | `AuthBoot` |
| call-manager | direct 시작(`startIncomingDomainTypeSIPDirect`) |
| direct-manager | `GetByHash`, `Regenerate`(update 실패 경로 포함), `dbUpdate`, `dbList`(hash 필터), 요청과 응답 로그 |

- 훅 검사는 `fmt.Sprint(v)` 가 아니라 JSON 포매터 출력 등 직렬화 결과 전체에서 센티널을 찾는다(구조체가 `Data` 맵 값으로 들어가면 문자열 타입 검사만으로는 통과하기 때문). 훅 헬퍼는 서비스마다 복제한다(VOIP-1567 의 `logsecret_test.go` 패턴, 공용화는 오버엔지니어링).
- 변이 확인: 각 대표 지점을 원래 로그로 되돌리면 대응 테스트가 실패함을 PR 본문에 기록한다.
- 사이트별 잔존 확인용 grep 을 PR 에서 수행한다: `WithField("direct",` 와 `hash: %s` + `d.Hash` 가 대상 서비스에 남지 않아야 한다.

## 5. 로그 형태 변경과 영향

- 키 `direct`(객체)가 사라지고 `direct_id`, `customer_id`, `resource_type`, `resource_id` 가 추가된다.
- direct-manager 의 `request`, `response` 필드가 사라지고 `method`, `uri`, `status_code`, `data_type` 로 바뀐다.
- 이 저장소의 알림과 대시보드가 이 필드를 파싱하는지 확인한다(VOIP-1567 과 동일). 외부 Grafana 쿼리는 확인하지 못한다는 점을 PR 본문에 기록한다.

## 6. 수용 위험 (PR 본문 기록 항목)

- 유형 3(부모 리소스 객체의 `direct_hash`), 유형 5(call-manager channel 의 `destination_number`)로 direct 통화당 hash 노출이 남는다.
- 에러 문자열 URI 경로.
- 이미 로그에 남은 hash 의 재생성 여부는 대표님 판단 사항이며 이번 PR 에 포함하지 않는다.
