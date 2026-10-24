# VOIP-1651 bin-number-manager 단위 테스트 커버리지 70% 이상 설계

## 1. 티켓 검토 (이슈 분석)

- 판정: 유효. origin/main 99b59c386 에서 같은 명령으로 실측한 수기 코드 65.8% (1502/2282), 원본 57.5% (1502/2612). 카드 수치와 일치.
- 관련 PR/커밋: 이 경로의 최근 커밋은 VOIP-1647(60% 목표, 48.2%에서 65.8%로 상향, 머지됨), VOIP-1648(문서 정리, 머지됨)이며 진행 중인 PR 은 없음. 중복 없음.
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
| pkg/numberhandler | 395/515 | 120 | available_number 44, number.go 31, event.go 16, db.go 14, renew.go 11 |
| pkg/dbhandler | 180/225 | 45 | 오류 경로 (닫힌 DB, 캐시 오류) |
| pkg/listenhandler | 342/418 | 76 | 요청 파싱, 하위 오류 응답 |
| pkg/subscribehandler | 61/71 | 10 | processEventRun |
| models/number | 10/19 | 9 | webhook 변환 |
| 기타 (internal/config, numberhandler/main.go NewNumberHandler, requestexternal) | - | 9 | 대상 아님 (config 3, NewNumberHandler 4, requestexternal 2) |
| 기타 (internal/config, numberhandler/main.go NewNumberHandler, requestexternal) | - | 9 | 대상 아님 (config 3, NewNumberHandler 4, requestexternal 2) |
| requestexternal/models/telnyx | 23/28 | 5 | ConvertProviderNumber |

예상 증가: cachehandler 약 60 + numberhandler 약 90 + dbhandler 약 30 + models 약 13 = 약 190 stmt (예상 수기 합계 약 74%). 부족하면 listenhandler 와 subscribehandler 를 추가.

## 3. Verify 항목 (설계 단계, 코드 없음)

1. 수기 합계 71% 이상. gens/, mock_*.go, DO NOT EDIT 제외, 중복 블록은 최대 count. PR 본문에 패키지별 전후 표.
2. `GOFLAGS=-mod=mod go test -race -count=1 ./...` 전부 통과, golangci-lint 통과, `scripts/check-test-conventions.sh` 통과(Test_ 네이밍, gomock 컨트롤러명 mc).
3. 비테스트 .go 변경 0. `git diff --stat origin/main...HEAD` 가 bin-number-manager 의 _test.go, go.mod/go.sum(miniredis 추가 시), 이 문서뿐.
4. go.mod 변경은 miniredis v2.36.1(다른 서비스와 동일 버전)만. 테스트에서만 import, `go mod tidy` 후 추가 diff 없음.
5. 기대값은 리터럴 또는 독립 계산. 제품 함수 출력으로 기대값을 만드는 자기비교 금지. 빈 호출 금지.
6. 변이 표본 최소 4개(원복 확인): 가상번호 후보 필터링 조건, Delete 의 provider switch, cachehandler 키 접두사, 웹훅 변환 필드 하나.
7. 합성값만. 외부 호출과 네트워크 의존 없음(miniredis 는 로컬 루프백 리스너).
8. 가상번호 후보는 난수이므로 후보 집합에 의존하는 단언은 gomock DoAndReturn 으로 실제 전달된 후보를 받아 부분 집합으로 단언하고 고정 시드나 난수값 비교는 하지 않음.

## 4. 테스트 계획

- pkg/cachehandler/handler_test.go: NewHandler/Connect(정상, 서버 중단 오류), Number/NumberByNumber 의 Set/Get/Del 왕복, NumberSet 이 두 키를 쓰는지, 키 없음 오류, 손상된 JSON 오류, 서버 중단 오류, TTL 24시간.
- pkg/numberhandler/available_number_test.go: GetAvailableVirtualNumbers 의 limit=0 기본 10, 후보 3배수 요청, 점유 번호 제외, limit 절단, 점유로 limit 미달 시 부족한 수 반환, db 오류 전파, 결과 필드(Provider none, 국가, 음성 기능).
- pkg/numberhandler 의 number/db/event/renew 오류 경로: 예약 접두사 거부, 잔액 조회 오류, 잔액 부족, 구매 실패, Delete(조회 실패, provider 해제 실패, 미지원 provider, dbDelete 실패), List/Count/Update 오류, db.go 의 NumberCreate/Get/Delete 실패, EventCustomerDeleted 목록 실패와 삭제 실패 계속, EventFlowDeleted 의 목록 실패와 1000건 페이지 반복, RenewNumbers 의 기간 미지정 오류.
- pkg/dbhandler: 닫힌 sql.DB 로 각 메서드의 쿼리 오류, 캐시 mock 오류에서의 폴백.
- models/number: ConvertWebhookMessage 필드 복사, CreateWebhookEvent 의 JSON 키 집합.
- requestexternal/models/telnyx: ConvertProviderNumber.
- 부족할 때만 listenhandler, subscribehandler 오류 분기.
