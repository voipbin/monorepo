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
| 도달 불가 또는 비현실적 | 약 38 | `rand.Read` 실패와 `len(out) < n` (setup.go 31, 43), `maxSanitizedLen < 0`(81, 상수 기반), `generateCredentialSecret` 오류 분기(138), 고정 DTO 의 `json.Marshal` 실패(v1_routes, v1_providercalls, v1_dialroute, telnyxclient 등), dbhandler 의 `PrepareFields`/`ToSql`/`RowsAffected`/`ScanRow` 오류. 일부 marshal 실패는 `Provider.Metadata`(`map[string]interface{}`)에 직렬화 불가 값을 주입하면 도달할 수 있으나 가치가 낮아 제외 |
| 도달 가능(cmd 밖) 약 105 | | dbhandler 68(provider 27, route 27, providercall 14) 중 실제 도달 가능 약 35, providerhandler 19(그중 도달 가능 약 8), routehandler 5, providercallhandler 7(생성자 4, providercall.go 3), config 2, models webhook 3, cachehandler 1 |

따라서 cmd 와 도달 불가를 제외한 도달 가능 상한은 약 2502 - 486 - 38 = 1978 (79%) 이며, 필요 증가분 26 은 도달 가능분(약 105 중 현실적 약 60)으로 충족할 수 있다. 부족하면 cmd/route-control 의 순수 함수와 cmd/route-manager 의 mock 가능 함수를 추가한다.

VOIP-1416(80% 미만 패키지 일반 상향, Backlog)은 저장소 로그와 gh 에서 확인되지 않아 범위를 알 수 없다. 80% 미만 패키지는 cmd/route-control, cmd/route-manager 뿐이며(models 는 90%) 이 PR 이 cmd 를 건드리는 경우에만 PR 본문에 겹침 가능성을 적는다. 흡수하지 않는다.

반증하지 못한 주장: 기준선 수치(독립 리뷰어도 74.98% 재현), 필요량. 틀린 주장: 없음.

## 2. 설계 (계획)

범위: `*_test.go` 추가뿐이다(제품 코드, go.mod, go.sum 변경 없음). 외부 호출과 네트워크 의존 없음. 모든 대상은 기존 mock 과 sqlite 인메모리로 주입된다.

대상과 verify 항목(이 단계는 코드 없음, 구현 시작 후 테스트 작성):

1. `pkg/dbhandler` provider/route/providercall (예상 +35, 상한 약 +38)
   - 별도 `sql.Open("sqlite3", ...)` 로 만든 뒤 `Close` 한 핸들러(공유 `dbTest` 는 건드리지 않는다)로 Create/Update/Delete 의 실행 오류, `providerGetFromDB`/`routeGetFromDB`/`providerCallGetFromDB` 직접 호출의 조회 오류, List 의 조회 오류가 각 함수 문구(`could not execute query`, `exec failed`, `could not query`)를 포함하고 `errors.Is`/메시지로 `sql: database is closed` 원인이 보존된다. 캐시 mock 은 해당 경로에서 호출되지 않는다(EXPECT 없음).
   - `ProviderGet`/`RouteGet`: 캐시가 먼저 호출되므로 `cache.XxxGet` 이 오류를 반환하도록 기대하고, 이어서 닫힌 DB 의 조회 오류가 그대로 전파되며 `Set` 이 호출되지 않는다.
   - provider/route 의 `GetFromDB` 행 없음 `ErrNotFound`, List 의 알 수 없는 필터는 `could not apply filters` 오류(providercall 의 기존 `Test_ProviderCallList_InvalidFilter` 와 같은 방식).
   - List 에서 token 이 빈 문자열이면 mock `utilHandler.TimeGetCurTime()` 값이 쓰이고(기대 호출 인자 고정) `tm_create < token` 인 행만 반환된다. 해당 테스트가 만든 행은 정리해 기존 `Test_ProviderList`/`Test_RouteList` 의 개수 단언을 오염시키지 않는다(providercall 의 `cleanupProviderCalls` 패턴).
   - 라이브 DB 에 행을 만든 뒤 캐시 `ProviderSet`/`RouteSet` 이 오류를 반환하면 `providerUpdateToCache`/`routeUpdateToCache` 가 그 오류를 반환한다. `ProviderCreate`/`ProviderUpdate`/`RouteCreate` 등은 이 오류를 무시하므로 반환값 nil 이면서 `Set` mock 이 정확히 1회 호출됨(gomock 엄격 매칭)을 단언한다.
2. `pkg/providerhandler`
   - Get 이 `dbhandler.ErrNotFound` 를 `PROVIDER_NOT_FOUND` NotFound 도메인 오류(`cerrors` 코드까지 비교)로 바꾸고 다른 오류는 그대로 전파한다.
   - Update 의 현재 provider 조회가 `ErrNotFound` 이면 `PROVIDER_NOT_FOUND` 가 전파되고 `ProviderUpdate` 는 호출되지 않는다(기존 `Test_Update_GetError` 와 다른 오류 값 사용). 잘못된 codecs 값은 `INVALID_CODECS` 인수 오류이며 db 쓰기 호출이 없다(Create 와 Update 각각).
   - `Setup`: 지원하지 않는 carrier 를 주면 네트워크 호출 없이 반환하는 경로(setup.go 91-93)를 확인한다. 잘못된 PSTN 게이트웨이 포트(예 `host:abc`)는 오류를 반환한다. 롤백(`telnyxCleanup`)의 삭제 실패 로그 분기는 기존 setup 테스트의 httptest 서버가 오류 상태를 돌려주도록 구성해 도달시키고, 오류가 호출자에게 원 오류로 전달되는지 단언한다.
3. `pkg/routehandler`
   - Get 의 `ErrNotFound` 를 `ROUTE_NOT_FOUND` 로 변환.
   - `ListByTarget`(route.go)의 target 경로 대 `all` 경로의 provider 중복과 `Dialroute`(dialroute.go)의 customer 경로 대 기본 경로의 provider 중복: 서로 다른 `Route.ID` 에 같은 `ProviderID` 를 주고 우선하는 쪽의 `ID` 가 결과에 남고 다른 쪽은 제외됨을 ID 로 단언한다(개수만 보지 않는다).
4. `pkg/providercallhandler`: `NewProviderCallHandler` 가 지정한 의존성으로 핸들러를 만든다. 그룹콜 목록의 nil 요소를 건너뛰고 나머지 ID 만 모은다. 임시 flow 정리(`FlowV1FlowDelete`) 실패 시에도 원 오류가 반환되고 정리 호출이 정확히 1회임을 mock 호출 횟수로 단언한다.
5. 후속(트리거: 위를 마치고 76% 미만): config 의 미커버 2, models webhook 3, cachehandler 1 을 추가하고, 그래도 부족하면 cmd 의 순수 함수를 추가한다.

네이밍과 충돌: 같은 패키지에 이미 `Test_Get_Error`, `Test_Create_Error`, `Test_Update_GetError`, `Test_Delete_GetError`, `Test_ListByTarget_AllRoutesError`, `Test_Get_NotFoundBecomesTypedError` 등이 있으므로 신규 테스트는 전부 새 이름(`_ErrNotFound`, `_InvalidCodecs`, `_ClosedDB` 등 접미어)을 쓰고 컴파일과 `go vet` 로 충돌이 없음을 확인한다.

검증 방법:

- 위 계산법으로 수기 합계 76% 이상(권장), 75.0% 이상(인수). 측정은 bin-route-manager 에서 위 `go test` 명령으로 하고 PR 본문에 패키지별 전후 표, 수기/원본 합계, 계산법을 적는다.
- bin-route-manager 에서 루트 CLAUDE.md 5단계(`go mod tidy`, `go mod vendor`, `go generate ./...`, `go test ./...`, `golangci-lint run -v --timeout 5m`)를 실행하고 go.mod/go.sum 변경이 없음을 `git diff` 로 확인한다. 저장소 루트에서 `git fetch origin main` 후 `bash scripts/check-test-conventions.sh` 를 실행한다. `-race -count=3` 으로 반복 실행해 안정성을 확인한다.
- 단언은 하드코딩한 기대값과 gomock 엄격 매칭으로 한다. 호출만 하거나 자기 비교인 테스트는 쓰지 않는다.
- 변이 검사(격리 복사본에서만, 기존 테스트는 통과하고 새 테스트만 실패해야 인정): (a) 두 handler 의 `ErrNotFound` 매핑 제거, (b) 두 중복 제거 로직의 `==` 를 `!=` 로, (c) `INVALID_CODECS` 분기 제거, (d) dbhandler 빈 token 분기 제거, (e) 캐시 오류 전파를 `nil` 반환으로 교체, (f) nil 그룹콜 건너뜀 제거.
- 변경 파일은 bin-route-manager/ 안(`*_test.go`, 본 문서)뿐임을 `git diff --stat origin/main...HEAD` 로 확인한다.
- 공개 저장소 위생: 합성 값만 쓰고(호스트는 `example.com` 또는 `203.0.113.x`), 자격증명과 주소를 같은 리터럴에 두지 않으며 `user:pass@host` 형태를 쓰지 않는다. push 전에 신규 `_test.go` 와 본 문서에 대해 키, 토큰, IP 패턴을 grep 으로 검사한다.
