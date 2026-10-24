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
4. go.mod/go.sum 변경은 miniredis v2.36.1 과 그 간접 의존성 gopher-lua v1.1.1 (bin-message-manager, bin-ai-manager 와 같은 버전)만. 테스트에서만 import, `go mod tidy` 후 추가 diff 없음.
5. 기대값은 리터럴 또는 독립 계산. 제품 함수 출력으로 기대값을 만드는 자기비교 금지. 빈 호출 금지.
6. 변이 표본 최소 4개(원복 확인): 가상번호 필터의 existingSet 제거 무력화, Delete switch 의 Twilio 분기 변경, cachehandler 키 접두사(`number:`/`number-number:`) 변경, 웹훅 변환에서 필드 하나 누락. 추가로 Create 롤백 호출 제거.
7. 합성값만. 외부 호출과 네트워크 의존 없음(miniredis 는 로컬 루프백 리스너).
8. 가상번호 후보는 난수이므로 후보 집합에 의존하는 단언은 gomock DoAndReturn 으로 실제 전달된 후보를 받아 부분 집합으로 단언하고 고정 시드나 난수값 비교는 하지 않음.

## 4. 테스트 계획 (파일과 케이스, 미커버 줄 대조)

### 4.1 pkg/cachehandler/handler_test.go (miniredis, 선례: bin-message-manager)
- `NewHandler(...).(*handler)` 타입 단언으로 비공개/비인터페이스 메서드(NumberSetByNumber, NumberDelByNumber)도 호출. NewHandler 가 주소와 DB 번호를 필드에 보존, Connect 성공, 서버 종료 후 Connect 오류. 서버 종료 후 오류 케이스는 go-redis v8 재시도/백오프로 느릴 수 있어 `MaxRetries` 를 0 으로 둔 클라이언트(테스트에서 h.Cache 교체)와 짧은 ctx 타임아웃을 사용.
- Set 후 Get 왕복에서 필드를 리터럴과 비교. NumberSet 이 miniredis 에 `number:<id>` 와 `number-number:<번호>` 두 키를 모두 쓰고 TTL 이 24시간인지 확인. SetByNumber/GetByNumber/DelByNumber 왕복, Del 후 Get 오류, 키 없음 오류, `mr.Set` 으로 넣은 손상된 JSON 의 Unmarshal 오류, 서버 종료 후 Set/Get/Del 오류.
- 도달 불가로 제외: setSerialize 의 Marshal 오류(Number 는 항상 직렬화 가능), NumberSet 의 두 번째 setSerialize 실패(첫 쓰기만 성공시키는 주입이 어려움). 예상 도달 약 66 of 70.

### 4.2 pkg/numberhandler/available_number_test.go (GetAvailableVirtualNumbers, 44 stmt)
- (a) limit=0 은 10 으로 처리되어 NumberGetExistingNumbers 에 전달된 후보가 30개(DoAndReturn 으로 인자 캡처), 모두 리터럴 접두사 `+899` 로 시작하고 길이 13, 중복 없음. (b) 점유 번호를 후보 일부로 반환하면 결과에 없음(캡처한 후보 중 지정한 2개를 점유로 반환). (c) 점유가 21개 이상(후보 30개 중)이면 limit=10 에 못 미쳐 남은 9개 이하만 반환. (d) 점유가 없으면 limit 으로 절단되어 정확히 limit 개이고 순서는 후보 순서. (e) db 오류는 wrap 되어 전파, 결과 nil. 결과 필드는 모델 상수가 아니라 리터럴(`"899"`, `"none"` 은 구현 단계에서 실제 상수 값을 확인해 사용)과 비교.

### 4.3 pkg/numberhandler 오류 경로 (gomock), 미커버 줄 기준
- number.go 미커버(34-35, 40-42, 45-47, 52-54, 80-81, 90-91, 139-141, 201-203, 219-221, 230, 237, 241-243, 247-249, 263-270, 289-291, 305-307, 324-326): Create 의 예약 접두사 거부(provider 와 billing 미호출), 잔액 조회 오류, 잔액 부족(구매 미호출), 구매 실패, Register 실패 시 NumberRelease 롤백 호출 확인과 롤백 자체 실패(오류는 여전히 반환), NumberUpdateTags 실패(결과는 성공). Register 의 목록 조회 오류, 이미 존재하는 번호, dbCreate 실패. Delete 의 Get 실패, Telnyx/Twilio 해제 실패, 미지원 provider 이름, dbDelete 실패. Get 의 NotFound 변환(cerrors 타입과 코드)과 일반 오류 wrap. List/Count/Update/UpdateMetadata 오류 wrap.
- db.go 미커버(68-70, 75-77, 93-95, 100-102, 117-119, 160-162, 188-190): dbCreate 의 NumberCreate 실패와 NumberGet 실패, dbDelete 의 NumberDelete 실패와 삭제 후 Get 실패, dbGet/dbList/dbUpdate/dbListByTMRenew 실패 전파.
- event.go 미커버(31-33, 42-43, 68, 77-78, 99-111): EventCustomerDeleted 의 목록 실패(오류 반환)와 삭제 실패 후 다음 번호 계속. EventFlowDeleted 는 call_flow 루프와 message_flow 루프 각각에 대해 목록 실패 중단, NumberUpdate 실패 후 계속, 1000건 가득 찬 첫 페이지 뒤 다음 페이지 조회(두 번째 조회가 빈 목록이면 종료), 1000건 미만이면 종료.
- renew.go 미커버(35-36, 40-42, 59-61, 81-82, 125-127, 144-146): RenewNumbers 의 days/hours/tmRenew 모두 빔, renewNumbersByTMRenew 의 목록 조회 오류, 잔액 부족 시 Delete 실패해도 계속, dbUpdate 실패 후 continue 로 processed==0 이 되어 반복 종료, renewNumbersByDays/ByHours 의 하위 오류 wrap.

### 4.4 pkg/dbhandler (기존 sqlite TestMain 재사용)
- 닫힌 sql.DB 로 만든 handler 에서 각 메서드(Create/Get/List/Update/Delete/GetExistingNumbers/CountVirtual/GetsByTMRenew)의 쿼리 또는 Exec 오류 전파(약 8-9 stmt).
- ApplyFields 에 잘못된 필터를 넣는 NumberList/NumberGetsByTMRenew 오류(190, 378), 캐시 mock 오류 분기: numberGetFromCache 오류 시 DB 폴백(NumberGet), NumberGetFromCacheByNumber 오류 전달, numberSetToCache/numberUpdateToCache 오류는 비공개 함수를 같은 패키지에서 직접 호출해 전파를 확인하고 공개 메서드에서는 캐시 오류가 나도 DB 결과는 성공(`_ =` 로 버려짐)임을 확인.
- 도달 불가로 제외: squirrel ToSql 오류, PrepareFields/ScanRow 오류(열 구성이 고정). main.go NewHandler 4 stmt 는 필드 주입(db, cache, util)이 보존되는지 단언하는 1개 케이스만(빈 호출 아님). 예상 도달 약 25 of 45.

### 4.5 기타
- models/number/webhook_test.go: ConvertWebhookMessage 가 공개 필드를 모두(서로 다른 리터럴 값으로) 옮기고, 내부 필드(ProviderName, ProviderReferenceID)가 WebhookMessage 에 없음을 JSON 키 집합으로 확인. CreateWebhookEvent 의 JSON 키 집합과 값(리터럴 기대 JSON).
- pkg/requestexternal/models/telnyx: ConvertProviderNumber 의 필드 매핑(ID, Status 캐스팅, T38 와 Emergency 가 서로 바뀌지 않음)을 값이 서로 다른 리터럴 기대값으로.
- 부족할 때만 listenhandler(요청 본문 파싱 오류, 하위 오류 응답), subscribehandler(processEventRun) 오류 분기.
- 새 파일은 같은 이름 규칙의 `_test.go`(handler_test.go, webhook_test.go 등), 기존 파일에는 케이스 추가. Test_ 네이밍과 gomock 컨트롤러명 mc 준수.
