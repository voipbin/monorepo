# VOIP-1651 bin-number-manager 단위 테스트 커버리지 70% 이상 설계

## 1. 티켓 검토 (이슈 분석)

- 판정: 유효. origin/main 99b59c386 에서 같은 명령으로 실측한 수기 코드 65.8% (1502/2282), 원본 57.5% (1502/2612). 카드 수치와 일치.
- 관련 PR/커밋: 이 경로의 최근 테스트/문서 관련 커밋은 VOIP-1647(60% 목표, 48.2%에서 65.8%로 상향, 머지됨), VOIP-1648(문서 정리, 머지됨)이며 진행 중인 PR 은 없음. 중복 없음.
- 필요량: 70.0% 는 1598 stmt (+96, 0.7x2282=1597.4 의 올림), 권장 71% 는 1621 stmt (+119, 0.71x2282=1620.2 의 올림). 목표는 71% 이상.
- 로컬 `vendor/` 가 go.mod 와 어긋나 있어 `GOFLAGS=-mod=mod` 로 측정. 실행 간 변동은 없었음.
- 테스트하기 어려운 구조: `cmd/number-control` (323 stmt, 0%) 과 `cmd/number-manager` (116 미커버) 는 DB, Redis, RabbitMQ, cobra 에 묶인 process wiring 이라 계획에서 제외. 제외하면 나머지 미커버는 341 stmt(780-439)이며 필요량 96~119 는 그 28~35%. 그중 requestexternal/telnyx.go 의 2 stmt 는 도달 불가로 제외.
- 카드 본문이 적은 VOIP-1416(80% 미만 패키지 일반 상향, 카드 기준이며 저장소에서 별도 확인 불가)과 겹치는 패키지: numberhandler 76.7%, models/number 52.6%, pkg/cachehandler 0% (이 PR 이 올리지만 흡수 관계는 아님).

## 2. 대상과 산정 (패키지별 기준선, 수기 기준)

| 패키지 | 전 (cov/stmt) | 미커버 | 이번 대상 |
|---|---|---|---|
| cmd/number-control | 0/323 | 323 | 제외 |
| cmd/number-manager | 2/118 | 116 | 제외 |
| pkg/cachehandler | 0/70 | 70 | 전부 (miniredis) |
| pkg/numberhandler | 395/515 | 120 | available_number 44, number.go 31, event.go 16, db.go 14, renew.go 11 (116). main.go NewNumberHandler 4 는 대상 아님 |
| pkg/dbhandler | 180/225 | 45 | 오류 경로 (닫힌 DB, 캐시 오류) |
| pkg/listenhandler | 342/418 | 76 | 요청 파싱, 하위 오류 응답 |
| pkg/subscribehandler | 61/71 | 10 | processEventRun |
| models/number | 10/19 | 9 | webhook 변환 |
| models/availablenumber | 8/9 | 1 | 대상 아님 |
| pkg/requestexternal/models/telnyx | 23/28 | 5 | phone_number.go 4 (ConvertProviderNumber, 대상). available_number.go 1 은 대상 아님 |
| internal/config, pkg/requestexternal | - | 3 + 2 | 대상 아님 (requestexternal 2 는 도달 불가) |
| 합계 | 1502/2282 | 780 | 323+116+70+120+45+76+10+9+1+5+3+2 = 780 |

예상 증가: cachehandler 약 60 + numberhandler 약 90 + dbhandler 약 30 + models 약 13 = 약 193 stmt (예상 수기 합계 약 74%). 부족하면 listenhandler 와 subscribehandler 를 추가.

## 3. Verify 항목 (설계 단계, 코드 없음)

1. 수기 합계 71% 이상. gens/, mock_*.go, DO NOT EDIT 제외, 중복 블록은 최대 count. PR 본문에 패키지별 전후 표.
2. `GOFLAGS=-mod=mod go test -race -count=1 ./...` 전부 통과, golangci-lint 통과, `scripts/check-test-conventions.sh` 통과(Test_ 네이밍, gomock 컨트롤러명 mc).
3. 비테스트 .go 변경 0. `git diff --stat origin/main...HEAD` 가 bin-number-manager 의 _test.go, go.mod/go.sum(miniredis 추가 시), 이 문서뿐.
4. go.mod 변경은 miniredis v2.36.1(다른 서비스와 동일 버전)만. 테스트에서만 import, `go mod tidy` 후 추가 diff 없음.
5. 기대값은 리터럴 또는 독립 계산. 제품 함수 출력으로 기대값을 만드는 자기비교 금지. 빈 호출 금지.
6. 변이 표본 최소 4개(원복 확인): 가상번호 후보 필터링 조건, Delete 의 provider switch, cachehandler 키 접두사, 웹훅 변환 필드 하나.
7. 합성값만. 외부 호출과 네트워크 의존 없음(miniredis 는 로컬 루프백 리스너).
8. 가상번호 후보는 난수이므로 후보 집합에 의존하는 단언은 gomock DoAndReturn 으로 실제 전달된 후보를 받아 부분 집합으로 단언하고 고정 시드나 난수값 비교는 하지 않음.

## 4. 테스트 계획 (파일과 케이스)

- pkg/cachehandler/handler_test.go (miniredis, 선례: bin-message-manager/pkg/cachehandler/handler_test.go): NewHandler 가 주소/DB 를 보존하고 Connect 가 성공, 서버 종료 후 Connect 오류. Number/NumberByNumber 의 Set 후 Get 왕복에서 필드 값을 리터럴과 비교하고 miniredis 에서 키 `number:<id>`, `number-number:<번호>` 가 모두 존재하며 TTL 이 24시간인지 확인(NumberSet 이 두 키를 쓰는지). 키 없음 오류, `mr.Set` 으로 넣은 손상된 JSON 의 Unmarshal 오류, Del 후 Get 오류, 서버 종료 후 Set/Get/Del 오류. setSerialize 의 Marshal 오류 분기는 Number 가 항상 직렬화 가능해 도달 불가라 제외.
- pkg/numberhandler/available_number_test.go: GetAvailableVirtualNumbers. (a) limit=0 은 10 으로 처리되어 NumberGetExistingNumbers 에 전달된 후보가 30개, 모두 `+899` 접두사와 총 길이 13, 중복 없음(후보는 DoAndReturn 인자로 캡처). (b) 점유 번호를 후보의 일부로 반환하면 결과에 없음. (c) 후보보다 점유가 많아 limit 에 못 미치면 남은 수만 반환. (d) limit 절단. (e) db 오류는 wrap 되어 전파되고 결과 nil. 결과 필드(Provider none, 국가 코드 상수, 음성 기능)는 리터럴 상수가 아닌 모델 상수와 비교하되 상수 값 자체의 독립 확인(접두사 리터럴)을 병행.
- pkg/numberhandler number/db/event/renew 오류 경로 (gomock): Create 의 예약 접두사 거부(provider 와 billing 호출 없음), 잔액 조회 오류, 잔액 부족(구매 미호출), 구매 실패. CreateVirtual 의 db 오류들. Delete 의 Get 실패, provider 해제 실패(Telnyx, Twilio 각각), 미지원 provider 이름, dbDelete 실패. Get 의 NotFound 변환(cerrors)과 일반 오류 wrap. List/Count/Update 오류 wrap. db.go 의 dbCreate 의 NumberCreate/NumberGet 실패, dbDelete/dbUpdate 실패. EventCustomerDeleted 의 목록 실패 오류 반환과 삭제 실패 후 계속. EventFlowDeleted 의 목록 실패 중단, 업데이트 실패 후 계속, 1000건 가득 찬 페이지 후 다음 페이지 조회. RenewNumbers 의 days, hours, tmRenew 모두 비었을 때 오류.
- pkg/dbhandler (기존 sqlite TestMain 재사용): 별도로 열었다 닫은 sql.DB 로 만든 handler 에서 각 메서드(Create/Get/List/Update/Delete/GetExisting/Count/GetsByTMRenew)의 쿼리 오류 전파, 캐시 mock 의 오류 시 DB 폴백(NumberGet), 캐시 갱신 오류 전파(numberUpdateToCache), NumberGetFromCacheByNumber.
- models/number/webhook_test.go: ConvertWebhookMessage 가 모든 공개 필드를 옮기고 내부 필드(ProviderName, ProviderReferenceID)를 노출하지 않음을 리터럴 구조로 확인, CreateWebhookEvent 의 JSON 키 집합과 값(리터럴 기대 JSON).
- pkg/requestexternal/models/telnyx: ConvertProviderNumber 의 필드 매핑을 리터럴 기대값으로, 알 수 없는 상태 값의 처리.
- 부족할 때만 listenhandler(요청 본문 파싱 오류, 하위 오류 응답), subscribehandler(processEventRun) 오류 분기.
- 파일은 기존 `_test.go` 에 케이스를 추가하거나, 같은 이름 규칙의 새 `_test.go`(handler_test.go, webhook_test.go 등)를 만든다.
