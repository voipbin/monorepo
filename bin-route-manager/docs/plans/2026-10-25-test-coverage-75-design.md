# bin-route-manager unit test coverage 75% (VOIP-1652)

## 1. 이슈 분석 (티켓 검토)

판정: 유효. 기준선 `origin/main accc03e00` 을 `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...` 로 재측정했다(브랜치 base 는 accc03e00 이다).

- 합격 판정 계산법(이 작업 고유 정의, PR 본문에도 적는다): cp.out 의 블록을 (파일, 범위) 키로 합쳐 count 최대값을 취하고, `/gens/` 경로, 파일명 `mock_*`, `package` 앞 헤더에 `Code generated` 와 `DO NOT EDIT` 가 있는 파일을 제외한 뒤 커버 statement / 전체 statement 로 계산한다. CI 와 소수점 차이가 날 수 있다.
- 기준선: 수기 2502 statement 중 1876 커버, 74.98%. 원본(`go tool cover -func` total, 생성 코드와 mock 포함) 66.5%. 75.0% 는 1877(+1), 76% 는 1902(+26, 0.76 x 2502 = 1901.52), 77% 는 1927(+51)이다. 목표는 76% 이상.
- 관련 PR: 선행 작업 VOIP-1647(#1508), VOIP-1651(#1575)은 머지되었고 같은 경로를 바꾸는 열린 PR 은 없다(`gh pr list --state all --search "route-manager coverage"`).

미커버 분류(statement):

| 구분 | 수 | 처리 |
|---|---|---|
| cmd/route-control (cobra CLI, 대부분 DB/Redis/RabbitMQ 연결 후 실행되는 `RunE`) | 315 | 후순위. 순수 함수(`resolveUUID`, `printJSON`, 명령 생성자)는 도달 가능하나 목표 달성에 필요하지 않으면 건드리지 않는다 |
| cmd/route-manager (`main`, 연결 초기화, 시그널 대기) | 171 | 후순위. `runRequestListen` 등 mock 가능한 함수는 여유분으로만 검토 |
| cmd 밖 미커버 합계 | 140 | listenhandler 32, telnyxclient 3, dbhandler 68, providerhandler 19, routehandler 5, providercallhandler 7, config 2, models webhook 3, cachehandler 1 |
| 그중 이번 작업에서 채택하지 않는 것(보수적 추정) | 약 78 | listenhandler 32, telnyxclient 3, models webhook 3 (`json.Marshal` 실패. route 와 providercall 모델은 고정 필드라 도달 불가이고 provider 계열은 `Metadata`(`map[string]interface{}`)에 직렬화 불가 값을 넣으면 도달할 수 있으나 가치가 낮아 채택하지 않는다), config `BindEnv` 오류 1, providerhandler/setup.go 6 (`rand.Read` 실패와 `len(out) < n` 31, 43, `maxSanitizedLen < 0` 81 상수 기반, `generateCredentialSecret` 오류 분기 138), dbhandler 약 33 (`PrepareFields`, `ToSql`, `ScanRow`, `RowsAffected` 오류와 기타. 이 테스트들은 sqlite 인메모리로 돌며 스키마 변경 등으로 일부 유도할 수 있을 가능성은 있으나 실측 전이라 채택하지 않는다) |
| 그중 채택 대상 | 약 62 | dbhandler 약 35, providerhandler 약 13, routehandler 5, providercallhandler 7, config `BindPFlag` 오류 1(빈 FlagSet 으로 유도 가능하면), cachehandler 1 |

위 분류의 숫자는 블록 단위로 센 계획용 추정치이며(합 140 = 78 + 62) 구현 중 실측으로 갱신한다. 78 에는 실제로 도달 가능한 블록이 섞여 있을 수 있으므로 이론 상한이 아니라 보수적인 목표선이다. 이 분류로 계산한 천장은 2502 - 486 - 78 = 1938 (약 77.5%) 이고, 필요 증가분 26 은 채택 대상 약 62 로 충족할 수 있다(여유 약 35). dbhandler 를 0 으로 쳐도 providerhandler 13 + routehandler 5 + providercallhandler 7 + cachehandler 1 = 26 으로 필요량과 같아 하한은 여유가 없으므로 dbhandler 항목을 먼저 구현한다. providerhandler 13 에는 `Setup` 래퍼 2(비-telnyx carrier 로 호출하며 `NewTelnyxClient` 는 네트워크 호출이 없는 구조체 생성이다)가 포함된다. providercallhandler 7 은 `NewProviderCallHandler` 생성자 4 와 providercall.go 3 이다. 부족하면 cmd/route-control 의 순수 함수와 cmd/route-manager 의 mock 가능 함수, 그리고 위 미채택 항목 중 실측으로 도달이 확인된 것을 추가한다.

VOIP-1416(80% 미만 패키지 일반 상향, Backlog)은 저장소 로그와 gh 에서 확인되지 않아 범위를 알 수 없다. 수기 코드 기준 80% 미만은 cmd/route-control, cmd/route-manager 뿐이지만 `go test` 패키지 출력(mock 포함) 기준으로는 cachehandler, dbhandler, providercallhandler, routehandler, telnyxclient, providerhandler 도 80% 미만이다. 지표가 달라 범위를 알 수 없으므로 PR 본문에 겹침 가능성만 적고 흡수하지 않는다.

반증하지 못한 주장: 기준선 수치(독립 리뷰어도 74.98% 재현), 필요량. 틀린 주장: 없음.

## 2. 설계 (계획)

범위: `*_test.go` 추가뿐이다(제품 코드, go.mod, go.sum 변경 없음). 외부 호출과 네트워크 의존 없음. 모든 대상은 기존 mock 과 sqlite 인메모리로 주입된다.

대상과 verify 항목(이 단계는 코드 없음, 구현 시작 후 테스트 작성):

1. `pkg/dbhandler` provider/route/providercall (예상 +35, 상한 약 +38)
   - 닫힌 DB 의 `ProviderCreate`/`RouteCreate`/`ProviderUpdate`/`ProviderDelete`/`RouteUpdate`/`RouteDelete` 는 SQL 실행 전에 `utilHandler.TimeNow()` 를 호출하므로 mock 기대를 둔다. 닫힌 DB 의 List 조회 오류 테스트에서 token 이 비어 있으면 `TimeGetCurTime` mock 도 둔다. (j) 를 위해 구현 시 먼저 sqlite 드라이버의 `tm_create` 저장 형식을 실측해 mock 토큰 문자열을 맞춘다.
   - 별도 `sql.Open("sqlite3", ":memory:")`(공유 DSN 과 다른 고유 DSN. 스키마는 필요 없다. 오류가 쿼리 실행 전에 난다)로 만든 뒤 `Close` 한 핸들러(공유 `dbTest` 는 건드리지 않는다)로 Create/Update/Delete 의 실행 오류, `providerGetFromDB`/`routeGetFromDB`/`providerCallGetFromDB` 직접 호출의 조회 오류, List 의 조회 오류가 각 함수 문구(`could not execute query`, `exec failed`, `could not query`)를 포함하고 오류 메시지에 `sql: database is closed` 원인 문구가 포함되고 `errors.Unwrap` 이 nil 이 아니다(`%w` 보존. `database/sql` 의 비공개 오류 값이라 `errors.Is` 는 쓰지 않는다). 캐시 mock 은 해당 경로에서 호출되지 않는다(EXPECT 없음).
   - 라이브 DB 에서 행이 없는 id 로 `providerUpdateToCache`/`routeUpdateToCache` 를 호출하면 `GetFromDB` 의 `ErrNotFound` 가 반환된다(캐시 `Set` 미호출, +2).
   - `ProviderGet`/`RouteGet`: 캐시가 먼저 호출되므로 `cache.XxxGet` 이 오류를 반환하도록 기대하고, 이어서 닫힌 DB 의 조회 오류가 그대로 전파되며 `Set` 이 호출되지 않는다.
   - provider/route 의 `GetFromDB` 행 없음 `ErrNotFound`, List 의 알 수 없는 필터는 `could not apply filters` 오류(providercall 의 기존 `Test_ProviderCallList_InvalidFilter` 와 같은 방식).
   - provider/route 의 List 에서 token 이 빈 문자열이면(providercall 은 기존 `Test_ProviderCallList_DefaultToken` 가 있어 제외) mock `utilHandler.TimeGetCurTime()` 값이 쓰이고(기대 호출 인자 고정) `tm_create < token` 인 행만 반환된다. 기존 `Test_ProviderList`/`Test_RouteList` 가 정리하지 않는 잔여 행(고정 ID, 2018년 `tm_create`)이 공유 DB 에 있으므로 개수가 아니라 이 테스트가 만든 고유 ID 로 걸러 단언하고(`limit` 은 잔여 행보다 충분히 크게 잡는다. 새 행은 `tm_create` 가 token 이전인 것과 이후인 것을 모두 만들어 `<` 조건을 검증하며, 시각은 행 삽입 직후 직접 UPDATE 하거나 `TimeNow` mock 의 고정값으로 정한다), 기존 고정 UUID 와 겹치지 않는 ID 를 쓰며, provider/route 전용 정리 헬퍼(사전 삭제와 `t.Cleanup`)를 테스트 파일 안에 둔다.
   - 라이브 DB 에 행을 만든 뒤 캐시 `ProviderSet`/`RouteSet` 이 오류를 반환하면 `providerUpdateToCache`/`routeUpdateToCache` 가 그 오류를 반환한다. `ProviderCreate`/`ProviderUpdate`/`RouteCreate` 등은 이 오류를 무시하므로 반환값 nil 이면서 `Set` mock 이 정확히 1회 호출됨(gomock 엄격 매칭)을 단언한다.
2. `pkg/providerhandler`
   - Get 이 `dbhandler.ErrNotFound` 를 `PROVIDER_NOT_FOUND` NotFound 도메인 오류(`cerrors` 코드까지 비교)로 바꾸고 다른 오류는 그대로 전파한다.
   - 오류 코드(`cerrors`) 비교는 `pkg/errors` 래핑을 통과하도록 `errors.As` 로 한다(기존 `providercall_errors_test.go` 와 같은 방식).
   - Update 의 사전 현재 provider 조회(첫 번째 `Get`)가 `ErrNotFound` 이면 `PROVIDER_NOT_FOUND` 가 전파되고 `ProviderUpdate` 는 호출되지 않는다(기존 `Test_Update_GetError` 는 갱신 뒤 두 번째 `Get` 실패를 다루므로 호출 위치가 다르다). 잘못된 codecs 값은 `INVALID_CODECS` 인수 오류이며 db 쓰기 호출이 없다(Create 와 Update 각각).
   - 공개 `Setup` 래퍼(setup.go 91-93, 기존 `Test_Setup_UnknownCarrier` 는 `setupWithClient` 를 호출하므로 미커버)를 `telnyx` 가 아닌 carrier(`vonage`)로 직접 호출한다. `NewTelnyxClient` 는 네트워크 호출이 없고 `telnyx` carrier 는 외부 호출이 생기므로 쓰지 않는다. 지원하지 않는 carrier 오류를 단언한다.
   - 잘못된 PSTN 게이트웨이 포트(`host:abc`, setup.go 159-162. 기존 `Test_Setup_InvalidFQDNFormat_*` 의 SplitHostPort 분기와 다른 분기): `MockTelnyxClient` 로 오류에 포트 문구가 포함되고 `DeleteFQDNConnection` 과 `DeleteOutboundVoiceProfile` 이 정확히 1회씩 호출됨을 단언한다.
   - `telnyxCleanup` 삭제 실패(setup.go 215, 223, 231): Step 5(`h.Create` 의 `db.ProviderCreate` mock 오류)로 `fqdnID` 가 채워진 상태에서 `MockTelnyxClient` 의 `DeleteFQDN`, `DeleteFQDNConnection`, `DeleteOutboundVoiceProfile` 이 오류를 반환하게 하고(기존 setup 테스트는 httptest 가 아니라 mock 과 `setupWithClient` 를 쓴다) `provider create failed: %w` 래핑 메시지 접두를 확인하고 `errors.Is` 로 원 오류를 확인하며(FlowDelete 실패 테스트도 `could not create calls` 래핑을 전제로 한다) 각 Delete 호출이 gomock 으로 정확히 1회임을 단언한다.
3. `pkg/routehandler`
   - Get 의 `ErrNotFound` 를 `ROUTE_NOT_FOUND` 로 변환.
   - `ListByTarget`(route.go)의 target 경로 대 `all` 경로의 provider 중복과 `Dialroute`(dialroute.go)의 customer 경로 대 기본 경로의 provider 중복: 서로 다른 `Route.ID` 에 같은 `ProviderID` 를 주고 우선하는 쪽의 `ID` 가 결과에 남고 다른 쪽은 제외됨을 ID 로 단언한다(개수만 보지 않는다).
4. `pkg/providercallhandler`: `NewProviderCallHandler` 가 지정한 의존성으로 핸들러를 만든다. 그룹콜 목록 `[nil, g]` 입력에서 nil 을 건너뛰고(루프 본문 전체가 현재 미커버) 비-nil 의 ID 가 `ProviderCallCreate` 인자의 `GroupcallIDs` 에 정확히 들어간다(gomock 인자 일치). 임시 flow 정리(`FlowV1FlowDelete`) 실패 시에도 원 오류가 반환되고 정리 호출이 정확히 1회임을 mock 호출 횟수로 단언한다. 추가로 `flowID` 가 주어지고 actions 가 없어 임시 flow 를 만들지 않은 경로에서 `CallV1CallsCreate` 가 오류를 내면 `FlowV1FlowDelete` 에 EXPECT 를 두지 않아 미호출을 gomock 으로 강제한다(기존 테스트에 없는 경로).
증가분 추정(statement): dbhandler 약 35, providerhandler 약 13(Get NotFound 1, INVALID_CODECS 2, Update 사전 조회 1, Setup 래퍼 2, 포트 오류 3, 정리 실패 3 중 도달분), routehandler 5, providercallhandler 7 으로 합계 약 +60 이며 필요량 26 대비 여유가 있다.

5. 후속(트리거: 위를 마치고 76% 미만): config `BindPFlag` 오류 1, cachehandler 1 을 추가하고, 그래도 부족하면 cmd 의 순수 함수를 추가한다.

네이밍과 충돌: 같은 패키지에 이미 `Test_Get_Error`, `Test_Create_Error`, `Test_Update_GetError`, `Test_Delete_GetError`, `Test_ListByTarget_AllRoutesError`, `Test_Get_NotFoundBecomesTypedError` 등이 있으므로 신규 테스트는 전부 새 이름(`_ErrNotFound`, `_InvalidCodecs`, `_ClosedDB` 등 접미어)을 쓰고 컴파일과 `go vet` 로 충돌이 없음을 확인한다.

검증 방법:

- 위 계산법으로 수기 합계 76% 이상(권장), 75.0% 이상(인수). 측정은 bin-route-manager 에서 위 `go test` 명령으로 하고 PR 본문에 패키지별 전후 표, 수기/원본 합계, 계산법을 적는다.
- bin-route-manager 에서 루트 CLAUDE.md 5단계(`go mod tidy`, `go mod vendor`, `go generate ./...`, `go test ./...`, `golangci-lint run -v --timeout 5m`)를 실행하고 go.mod/go.sum 변경이 없음을 `git diff` 로 확인한다. 저장소 루트에서 `git fetch origin main` 후 `bash scripts/check-test-conventions.sh` 를 실행한다. `-race -count=3` 으로 반복 실행해 안정성을 확인한다.
- 단언은 하드코딩한 기대값과 gomock 엄격 매칭으로 한다. 호출만 하거나 자기 비교인 테스트는 쓰지 않는다.
- 변이 검사(격리 복사본에서만, 기존 테스트는 통과하고 새 테스트만 실패해야 인정): (a) 두 handler 의 `ErrNotFound` 매핑 제거, (b) 두 중복 제거 로직(route.go, dialroute.go)에서 `exist = true` 를 제거(같은 provider 가 중복으로 남는지 새 테스트가 잡아야 한다. `==` 를 `!=` 로 바꾸는 변이는 기존 테스트가 먼저 실패하므로 쓰지 않는다), (c) `INVALID_CODECS` 분기 제거, (d) provider/route dbhandler List 의 빈 token 분기 제거(providercall 에는 적용하지 않는다), (e) `providerSetToCache`/`routeSetToCache` 의 `return err` 를 `return nil` 로, (f) nil 그룹콜 건너뜀 제거(nil 역참조 panic 으로 패키지 바이너리 전체가 죽으므로 제외할 새 테스트 이름 목록을 PR 본문에 적고 기존 테스트는 `-run` 으로 따로 돌려 판정한다), (g) 닫힌 DB 경로의 `%w` 사이트(PR 본문에 목록 기재)를 `%v` 로, (j) 빈 token List 의 `tm_create <` 조건을 `<=` 로(경계 행 `tm_create == token` 을 저장 형식과 같은 문자열 토큰으로 만들어 잡는다. 저장 형식은 sqlite 드라이버 형식이고 `TimeGetCurTime` 은 ISO 형식이라 문자열 비교임을 고려해 mock 토큰을 저장 형식과 동일한 문자열로 둔다), (k) `Setup` 래퍼가 carrier 인자를 빈 문자열로 넘기도록 변경(오류 메시지에 `vonage` 가 포함되는지 단언한다), (m) `NewProviderCallHandler` 가 db 필드를 연결하지 않도록 변경(생성된 핸들러의 필드 값을 단언), (n) FlowDelete 정리 조건 `tempFlowCreated &&` 제거(임시 flow 를 만들지 않은 경로의 미호출 단언이 잡는다), (l) 포트 오류 분기의 cleanup 호출 제거, (h) `telnyxCleanup` 에서 `DeleteFQDN` 실패 시 early return 하도록 변경(기존 setup 테스트는 통과하고 정리 실패 새 테스트만 실패해야 한다), (o) `ProviderGet`/`RouteGet` 이 DB 오류 뒤에도 캐시 `Set` 을 호출하도록 변경, (i) `providerGetFromDB` 의 `ErrNotFound` 반환을 `nil, nil` 로(`routeGetFromDB` 도 동일).
- 변경 파일은 bin-route-manager/ 안(`*_test.go`, 본 문서)뿐임을 `git diff --stat origin/main...HEAD` 로 확인한다.
- 공개 저장소 위생: 합성 값만 쓰고(호스트는 `example.com` 또는 `203.0.113.x`. 기존 `testSipGatewayFQDNForPSTN` 상수는 재사용하지 않고 새 상수를 `example.com` 으로 둔다), 자격증명과 주소를 같은 리터럴에 두지 않으며 `user:pass@host` 형태를 쓰지 않는다. push 전에 신규 `_test.go` 와 본 문서에 대해 키, 토큰, IP 패턴을 grep 으로 검사한다.
