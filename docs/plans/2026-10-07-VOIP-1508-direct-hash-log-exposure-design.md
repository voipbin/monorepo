# VOIP-1508 direct hash 로그 노출 수정 설계

- 상태: DRAFT (디자인 리뷰 대기)
- 근거 문서: `2026-10-07-VOIP-1508-direct-hash-log-exposure-analysis.md` (APPROVED rev5)
- 대표님 결정(2026-10-07): VOIP-1567 의 "유형 4 범위 밖" 결정을 재승인하여 유형 4 를 이번에 처리한다. 범위는 분석 문서 2.1, 2.2 의 전 지점과 direct-manager 요청, 응답 로그다. 유형 3(부모 리소스), 유형 5(call-manager channel)는 범위 밖이며 처리 후에도 direct 통화당 hash 노출은 0이 되지 않는다.

## 1. 원칙

- 구조체 통째 로그는 allow list 필드로 바꾼다(VOIP-1291, VOIP-1567 선례). `Direct` 모델에 `MarshalJSON` 은 넣지 않는다(RPC, 캐시 `direct:hash:*`, 이벤트 직렬화가 깨진다).
- 원문 hash 가 필요한 로그는 앞 12자 + `...` 로 줄인다(api-manager `truncateHash` 와 같은 수준).
- API, 이벤트, DB 계약 변경 없음. 배포 순서 제약 없음.

## 2. 공용 헬퍼

`bin-direct-manager/models/direct/direct.go` 에 추가한다. 모든 대상 서비스가 이미 `dmdirect` 로 이 패키지를 import 하고 `replace` 지시자로 참조하므로 vendor 갱신이 없다.

```go
// MaskHash returns a log-safe form of a direct hash (first 12 chars + "...").
func MaskHash(hash string) string

// LogFields returns the log-safe identity fields of a Direct (no hash).
func (d *Direct) LogFields() logrus.Fields
```

- `LogFields` 키: `direct_id`, `customer_id`, `resource_type`, `resource_id`. nil 수신자는 빈 Fields 를 반환한다.
- 이 패키지에 logrus import 가 추가된다. 모델 패키지가 logrus 에 의존하는 것이 부담이면 `LogFields` 를 두지 않고 호출 지점에서 4개 필드를 직접 쓴다. 호출 지점이 20곳 이상이라 중복 제거 이득이 크므로 헬퍼를 둔다(`bin-common-handler` 는 이미 모든 서비스가 logrus 와 함께 쓴다).
- `api-manager` 의 `truncateHash` 는 그대로 둔다(변경 이득 없음).

## 3. 변경 지점

### 3.1 객체 로그 (분석 2.1, 24곳)

`log.WithField("direct", d)` 를 `log.WithFields(d.LogFields())` 로 바꾼다. 메시지의 `hash: %s` 는 제거한다(9행). 메시지의 `direct_id: %s` 는 이미 있으므로 유지한다. `direct-manager` 의 `db.go:66` 은 base logger 필드 `"direct": d` 를 `d.LogFields()` 로 대체한다.

### 3.2 hash 문자열 로그 (분석 2.2)

- call-manager `start_incoming_domain_type_sip.go:81,83`: `"hash": MaskHash(hash)`, 메시지 `hash: %s` 도 마스킹값.
- direct-manager `handler.go:108-110`, `db.go:49-50`, `v1_directs.go:121-126`: `"hash": MaskHash(hash)`.
- direct-manager `v1_directs.go:62,100,138,182,218,254` 의 `%v` 마샬 실패 로그: `message: %v` 를 제거하고 `direct_id` 를 쓴다(list 인 `:62` 는 건수).

### 3.3 direct-manager 요청, 응답 로그

`listenhandler/main.go:151-155, 211, 222-227` 와 `v1_directs.go:126`.

- 요청 로그: `"request": m` 대신 `method`, `uri`(마스킹), `data_type` 만 기록한다. 요청 본문은 hash 를 담지 않으나(생성은 customer_id, resource_type, resource_id) 일관성을 위해 본문은 기록하지 않는다. 디버깅 단서는 URI 와 method 로 충분하다.
- 응답 로그: `"response": response` 대신 `status_code`, `data_type` 만 기록한다. 응답 본문(`Direct` JSON)은 hash 를 담으므로 기록하지 않는다.
- URI 마스킹: `maskURI(uri string) string` 을 `listenhandler` 에 추가한다. `by-hash/` 뒤 세그먼트를 `MaskHash` 로 대체한다(`regV1DirectsByHashGet` 일치 시). 일치하지 않으면 그대로 반환한다. `main.go:211` 의 Errorf URI 도 이 함수를 쓴다.
- `v1_directs.go:126` 의 `WithField("request", m)` 는 3.3 의 요청 로그 함수로 대체한다.

### 3.4 범위 밖(수용 위험으로 기록)

- 에러 문자열로 전파되는 URI(`requesthandler/send_request.go:45,53,62`). 전송 실패 때만 발생하고, 막으려면 `DirectV1DirectGetByHash` 에 에러 재포장이 필요하다. 트리거 조건: 운영에서 by-hash 전송 실패 로그에 hash 가 실제로 확인되면 별도 설계. 이번에는 하지 않는다.
- 유형 3, 유형 5, 로그 레벨 정책, redaction 훅 신설.

## 4. 테스트

- `models/direct`: `MaskHash`(빈 값, 12자 이하, 초과)와 `LogFields`(hash 미포함, nil 수신자) 단위 테스트.
- direct-manager `listenhandler`: `maskURI` 단위 테스트, 요청 및 응답 로그 훅 테스트(센티널 hash 가 모든 수집 항목의 필드와 메시지에 없고, 수집 건수 1 이상을 요구해 공허한 통과를 막는다).
- 각 서비스(agent, ai aihandler, ai teamhandler, conference, flow, queue, webchat, api-manager `AuthBoot`, call-manager direct 시작, direct-manager directhandler): 로그 훅으로 센티널 hash 가 로그 어디에도 없음을 확인하는 회귀 테스트. 서비스당 해당 함수 하나를 대표로 검증하되, `direct_hash.go` 계열은 hash 가 메시지에 찍히던 재생성 경로를 포함한다.
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
