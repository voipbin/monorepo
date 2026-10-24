# bin-route-manager unit test coverage 70% (VOIP-1651)

## 1. 이슈 분석 (티켓 검토)

판정: 유효. 기준선 `origin/main 99b59c386` 을 `GOFLAGS=-mod=mod go test -count=1 -coverprofile` 로 직접 재측정했다. 수기 코드(gens/, mock_*.go, `Code generated ... DO NOT EDIT` 제외, 같은 블록 중복은 최대 count) 합계는 2502 statement 중 1632개 커버, 65.2% 이다. 원본(생성 코드와 mock 포함)은 58.0% 이다. 70% 까지 120개(1752), 71% 까지 145개(1777)가 더 필요하다.

- 관련 PR: VOIP-1647(#1508)이 `gh pr list --state all --search "route-manager coverage"` 결과 MERGED 이며 선행이다. 같은 검색에서 열린 중복 PR 은 없다. 제품 코드 변경은 하지 않는다.
- 테스트 불가 구조: cmd/route-control(315)은 DB, Redis, RabbitMQ 연결이 필요해 제외한다. cmd/route-manager 는 대부분 데몬 기동 코드라 제외하되(커버 수가 실행마다 16~19개로 달라질 수 있다), 일부 함수는 mock 으로 가능하면 여유분으로만 쓴다. 분모 2502 에는 cmd/ 505 statement(미커버 약 489)가 그대로 남는다. cmd/ 밖 미커버는 381개이며, 이 중 +120(70%), +145(71%), +150(목표)을 회수해야 하므로 회수율은 각각 31%, 38%, 39% 이다. 산술적으로 달성 가능하다. 여유를 위해 목표는 약 +150 statement 로 잡는다.
- 수기 합계는 cp.out 후처리 스크립트(제외 필터 적용)로 산출한 값이며 go test 가 출력하는 값과 다르다. 이 방법은 PR 본문에 적는다. cachehandler(미커버 59, miniredis)와 config(미커버 60, viper 전역 상태)는 합 119개로 가장 큰 단일 몫이며 먼저 시도한다. 이 둘을 제외하면 cmd/ 밖 나머지 미커버는 262개(listenhandler 106, dbhandler 68, telnyxclient 31, healthcheckhandler 23, providerhandler 19, providercallhandler 7, routehandler 5, models 3)이고 +120~150 을 회수하려면 46~57% 가 필요해 현실적이지 않다. miniredis 는 bin-route-manager/go.mod 에 아직 없어 go.mod/go.sum 변경이 생기며(테스트 전용 의존성 miniredis 와 간접 의존 gopher-lua, 비테스트 .go 변경 없음), `go mod tidy` 후 그 외 추가 diff 가 없어야 한다. 둘 중 하나라도 구조적으로 불가능하면 목표를 낮추지 않고 사유와 달성값을 첨부해 작업을 중단하고 오케스트레이터에게 질의한다.

패키지별 기준선(수기 기준, 커버/전체): cmd/route-control 0/315, cmd/route-manager 16/190, internal/config 0/60, models/* 9/10 각각, pkg/cachehandler 0/59, pkg/dbhandler 372/440, pkg/healthcheckhandler 70/93, pkg/listenhandler 468/574, pkg/providercallhandler 94/101, pkg/providerhandler 238/257, pkg/routehandler 190/195, pkg/telnyxclient 157/188.

VOIP-1416 은 카드 본문에 따르면 80% 미만 패키지의 일반 상향 티켓(Backlog)이며 저장소 PR 이력에서는 확인되지 않는다(미확인). 80% 미만인 cachehandler, config, dbhandler, healthcheckhandler, telnyxclient 와 범위가 겹칠 수 있어 PR 본문에 명시만 하고 흡수하지 않는다.

## 2. 계획

대상과 예상 증가분(statement):

| 대상(미커버) | 내용 | 예상 |
|---|---|---|
| internal/config (60) | Bootstrap 이 모든 flag 를 등록하고 환경변수가 flag 기본값을 덮어쓰는지, LoadGlobalConfig 가 viper 값을 Config 로 옮기는지, Get 이 같은 전역을 반환하는지 | 약 55 |
| pkg/cachehandler (59) | miniredis 로 Provider/Route 의 Set/Get, RouteDelete, 존재하지 않는 키의 오류, 손상된 JSON 의 오류, Connect 성공과 연결 실패 | 약 55 |
| pkg/listenhandler (106, 도달 가능 상한 약 74) | v1_routes, v1_providers, v1_providercalls, v1_dialroute, v1_providers_setup 의 잘못된 JSON, 잘못된 UUID(400), 하위 핸들러 오류 전파(errorResponse 변환 또는 nil, error)와 processRequest 의 미지원 경로/메서드 분기 | 약 70 |
| pkg/telnyxclient (31) | httptest 서버로 HTTP 오류 상태, 잘못된 응답 본문, 전송 실패 경로 | 약 25 |
| pkg/healthcheckhandler (23, 모두 도달 가능) | Run 이 ctx 취소로 종료하고 tick 마다 runOnce 를 호출함 | 약 20 |
| dbhandler, providerhandler 등 | 위로 71% 에 못 미칠 때만 오류 경로를 추가 | 필요 시 |

miniredis 는 테스트에서만 import 하며 main 의 다른 서비스(v2.36.1 은 11개 서비스가 사용하고 v2.38.0 은 1개이며, 다수가 쓰는 v2.36.1 을 택한다)와 같은 버전을 쓴다. go.mod 와 go.sum 변경 외에 비테스트 .go 변경은 없다.

상세 케이스(단언은 모두 하드코딩한 기대값, 도달 불가 분기는 계산에서 제외):

- config (예상 +55, 상한 58, `BindPFlag`/`BindEnv` 오류 반환 2블록은 도달 불가라 제외): 각 테스트 시작에 `viper.Reset()` 과 `once = sync.Once{}`, `globalConfig = Config{}` 를 되돌려 `-count=N` 에서도 독립으로 만든다. `t.Setenv` 를 쓰므로 `t.Parallel` 은 쓰지 않는다. 환경변수 합성값은 호스트를 `example.invalid` 또는 `203.0.113.x` 로 하고 DSN 과 AMQP 주소에 자격증명을 넣지 않으며(예 `tcp(db.example.invalid:3306)/testdb`, `amqp://mq.example.invalid:5672`), 비밀번호는 주소와 분리된 명백한 가짜 값을 별도 문장에서 대입한다. flag 기본값 단언은 자격증명이 없는 항목(`/metrics`, `:2112`, redis db `1`, interval `30s`)으로 한정하고 `database_dsn` 과 `rabbitmq_address` 기본값 문자열은 테스트에 옮기지 않는다. Load 전 zero 값 단언처럼 실행 순서에 의존하는 단언은 쓰지 않는다.
- cachehandler (예상 +55, 상한 58, `setSerialize` 의 marshal 실패 1블록은 도달 불가): `redis.NewClient` 를 miniredis 주소에 붙인 핸들러로 ProviderSet 후 ProviderGet 이 필드 전체를 되돌리고 `mr.Keys` 에 `provider:<id>`, `mr.TTL` 이 24시간임을 단언, RouteSet/RouteGet/RouteDelete 후 키 소멸, 없는 키 Get 은 redis.Nil, 손상된 JSON 이 들어 있으면 오류, miniredis `RequireAuth` 로 `NewHandler` 의 비밀번호가 실제로 쓰이는지(맞는 값이면 Connect 성공, 틀린 값이면 실패), 서버를 닫은 뒤 Set/Get/Delete/Connect 오류(`context.WithTimeout` 으로 상한). 비밀번호는 주소와 분리된 가짜 값. NewHandler 의 필드 보관 자체를 비교하는 자기 비교 단언은 쓰지 않는다.
- listenhandler (예상 +70, 상한 약 74. `json.Marshal` 실패 블록 32 statement 는 도달 불가라 제외): 핸들러별 실제 미커버 블록과 1:1 로 둔다. 경로 ID 400 분기와 url.Parse 오류는 processRequest 의 정규식이 먼저 걸러내므로 핸들러 함수를 직접 호출해 검증한다.
  - v1_routes(36), v1_providers(28), v1_providercalls(8): (a) 깨진 JSON 본문은 오류 반환이고 하위 핸들러 미호출, (b) `Invalid route ID` 400 분기는 v1_routes 에만 있으므로 v1_routes 의 ID 핸들러(Get/Put/Delete)만 UUID 가 아닌 경로로 직접 호출해 400 과 하위 핸들러 미호출을 단언(v1_providers 와 v1_providercalls 는 `uuid.FromStringOrNil` 만 쓰고 400 분기가 없으므로 이 케이스를 두지 않는다), (c) 하위 핸들러 오류는 errorResponse 매핑 상태 코드 또는 오류 반환(핸들러별 기존 코드를 따름), (d) 잘못된 URI(`%zz`)로 url.Parse 오류.
  - v1_dialroute(6)과 v1_providers_setup(4)에는 경로 UUID 가 없으므로 (b)는 제외하고, 깨진 JSON, 하위 오류, setup 의 `ErrInvalidKey`(422)와 일반 오류(400)를 단언.
  - main.go(24): processRequest 의 `VoipbinError`/`ErrNotFound` 오류 매핑과 미지원 경로/메서드 404, errorResponse 의 `ToResponse` 실패 분기(typed-nil `(*cerrors.VoipbinError)(nil)` 을 `error` 로 넘겨 `errors.As` 는 성공하고 값이 nil 인 경우에만 도달하며 기대 상태 코드는 500), `/providers/setup` 디스패치와 요청 본문 로그가 `[redacted]` 로 대체됨, `Run` 에서 sockHandler mock 의 `QueueCreate` 오류 반환(`ConsumeRPC` 는 고루틴에서 호출되므로 mock 의 `DoAndReturn` 에서 채널로 동기화해 실행마다 커버가 달라지지 않게 하고, 동기화가 어려우면 이 분기는 제외한다). NewListenHandler 의 필드 보관만 비교하는 단언은 쓰지 않는다.
- telnyxclient (예상 +25, `json.Marshal` 실패 3블록은 도달 불가): `NewTelnyxClient` 생성, httptest 로 각 호출의 비정상 상태 코드, 잘못된 JSON 본문, 서버 종료 후 연결 오류, 제어문자가 든 baseURL 로 `NewRequestWithContext` 오류에서 오류 반환과 메시지 일부를 단언. password 인자는 가짜 값을 별도 변수로.
- healthcheckhandler (예상 +20): `Run` 은 기존 `newAlwaysSucceedsLocker`/`Mocklocker`, requesthandler mock, dbhandler mock 을 쓰고 sleep 대신 mock 의 `DoAndReturn` 에서 채널로 tick 호출을 동기화하며 `select` 타임아웃 가드를 둔다. ctx 취소로 반환됨, `ProviderList` 오류 후에도 루프가 계속됨, `UnlockContext` 오류, `last.TMCreate == nil` 종료 분기, `ProviderUpdateHealthStatus` 오류 로그 분기.
- 추가 대상(트리거): 위를 마친 뒤 수기 합계가 71% 미만이면 미커버가 큰 순서로 dbhandler(오류 경로), providerhandler/setup.go, providercallhandler 를 추가한다.

## 3. verify 항목

- 수기 합계 71% 이상(`-count=3` 안팎으로 재측정해 하한값을 보고). 측정 절차: bin-route-manager 에서 `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...` 를 실행하고, cp.out 의 블록을 (파일, 범위) 키로 합쳐 count 최대값을 취한 뒤 `/gens/` 경로, `mock_` 파일명, `package` 앞 헤더에 `Code generated` 와 `DO NOT EDIT` 가 있는 파일을 제외하고 커버 statement / 전체 statement 를 계산한다. 이 계산용 짧은 python 코드는 PR 본문에 첨부하고 카드 handoff 에도 둔다.
- 저장소 루트 CLAUDE.md 의 5단계(`go mod tidy`, `go mod vendor`, `go generate ./...`, `go test ./...`, `golangci-lint run -v --timeout 5m`)를 bin-route-manager 에서 모두 실행하고, tidy 후 go.mod/go.sum 의 diff 가 miniredis 와 그 간접 의존(gopher-lua)뿐이며 generate 후 mock diff 가 없음을 확인한다.
- 저장소 루트에서 `git fetch origin main` 후 `bash scripts/check-test-conventions.sh` 통과(추가된 줄만 검사: `Test_` 접두어, `reflect.DeepEqual` 과 `t.Errorf`, gomock 컨트롤러 변수명 `mc`). 신규 테스트가 `docs/conventions/testing.md` 에 맞는지도 확인한다.
- 단언은 하드코딩한 기대값과 gomock 엄격 매칭으로 하고 호출만 하는 테스트와 자기 비교 테스트는 쓰지 않는다. 오류 경로 테스트는 상태 코드와 하위 핸들러 미호출(EXPECT 없음)을 단언한다.
- 변이 검사: 격리된 복사본에서만 적용하고(원 worktree 에서는 `git diff --stat` 로 `_test.go`, go.mod, go.sum, 본 문서 외 변경이 없음을 최종 확인) 기존 테스트는 통과하고 새 테스트만 실패해야 인정한다. 대상은 listenhandler UUID 검증, cachehandler 키 접두어, cachehandler TTL 24h, RouteDelete, processRequest default 404, config 의 flag 기본값(`:2112`), `LoadGlobalConfig` 필드 매핑, telnyx 의 `ErrInvalidKey`(401/403) 분기, healthcheck `Run` 의 ctx.Done 분기이다. PR 본문에는 요약만 쓰고 출력은 붙이지 않는다.
- 변경 파일은 bin-route-manager/ 안(`*_test.go`, go.mod, go.sum, 본 문서)뿐이다.
- 공개 저장소 위생: 테스트 값은 합성값이며 자격증명과 주소를 같은 리터럴에 두지 않고, GitGuardian 체크가 녹색이어야 한다.
