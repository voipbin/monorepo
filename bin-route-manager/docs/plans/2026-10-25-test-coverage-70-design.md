# bin-route-manager unit test coverage 70% (VOIP-1651)

## 1. 이슈 분석 (티켓 검토)

판정: 유효. 기준선 `origin/main 99b59c386` 을 `GOFLAGS=-mod=mod go test -count=1 -coverprofile` 로 직접 재측정했다. 수기 코드(gens/, mock_*.go, `Code generated ... DO NOT EDIT` 제외, 같은 블록 중복은 최대 count) 합계는 2502 statement 중 1632개 커버, 65.2% 이다. 원본(생성 코드와 mock 포함)은 58.0% 이다. 70% 까지 120개(1752), 71% 까지 145개(1777)가 더 필요하다.

- 관련 PR: VOIP-1647(#1508)이 `gh pr list --state all --search "route-manager coverage"` 결과 MERGED 이며 선행이다. 같은 검색에서 열린 중복 PR 은 없다. 제품 코드 변경은 하지 않는다.
- 테스트 불가 구조: cmd/route-control(315)은 DB, Redis, RabbitMQ 연결이 필요해 제외한다. cmd/route-manager 는 대부분 데몬 기동 코드라 제외하되(커버 수가 실행마다 16~19개로 달라질 수 있다), 일부 함수는 mock 으로 가능하면 여유분으로만 쓴다. 분모 2502 에는 cmd/ 505 statement(미커버 약 489)가 그대로 남는다. cmd/ 밖 미커버는 381개이며, 이 중 +120(70%), +145(71%), +150(목표)을 회수해야 하므로 회수율은 각각 31%, 38%, 39% 이다. 산술적으로 달성 가능하다. 여유를 위해 목표는 약 +150 statement 로 잡는다.
- 수기 합계는 cp.out 후처리 스크립트(제외 필터 적용)로 산출한 값이며 go test 가 출력하는 값과 다르다. 이 방법은 PR 본문에 적는다. cachehandler(미커버 59, miniredis)와 config(미커버 60, viper 전역 상태)는 합 119개로 가장 큰 단일 몫이며 먼저 시도한다. 이 둘을 제외하면 cmd/ 밖 나머지 미커버는 262개(listenhandler 106, dbhandler 68, telnyxclient 31, healthcheckhandler 23, providerhandler 19, providercallhandler 7, routehandler 5, models 3)이고 +120~150 을 회수하려면 46~57% 가 필요해 현실적이지 않다. miniredis 는 bin-route-manager/go.mod 에 아직 없어 go.mod/go.sum 변경이 생기며(테스트 전용 의존성, 비테스트 .go 변경 없음), `go mod tidy` 후 추가 diff 가 없어야 한다. 둘 중 하나라도 구조적으로 불가능하면 목표를 낮추지 않고 사유와 달성값을 첨부해 작업을 중단하고 오케스트레이터에게 질의한다.

패키지별 기준선(수기 기준, 커버/전체): cmd/route-control 0/315, cmd/route-manager 16/190, internal/config 0/60, models/* 9/10 각각, pkg/cachehandler 0/59, pkg/dbhandler 372/440, pkg/healthcheckhandler 70/93, pkg/listenhandler 468/574, pkg/providercallhandler 94/101, pkg/providerhandler 238/257, pkg/routehandler 190/195, pkg/telnyxclient 157/188.

VOIP-1416 은 카드 본문에 따르면 80% 미만 패키지의 일반 상향 티켓(Backlog)이며 저장소 PR 이력에서는 확인되지 않는다(미확인). 80% 미만인 cachehandler, config, dbhandler, healthcheckhandler, telnyxclient 와 범위가 겹칠 수 있어 PR 본문에 명시만 하고 흡수하지 않는다.

## 2. 계획

대상과 예상 증가분(statement):

| 대상(미커버) | 내용 | 예상 |
|---|---|---|
| internal/config (60) | Bootstrap 이 모든 flag 를 등록하고 환경변수가 flag 기본값을 덮어쓰는지, LoadGlobalConfig 가 viper 값을 Config 로 옮기는지, Get 이 같은 전역을 반환하는지 | 약 55 |
| pkg/cachehandler (59) | miniredis 로 Provider/Route 의 Set/Get, RouteDelete, 존재하지 않는 키의 오류, 손상된 JSON 의 오류, Connect 성공과 연결 실패 | 약 55 |
| pkg/listenhandler (106) | v1_routes, v1_providers, v1_providercalls, v1_dialroute, v1_providers_setup 의 잘못된 JSON, 잘못된 UUID(400), 하위 핸들러 오류 전파(errorResponse 변환 또는 nil, error)와 processRequest 의 미지원 경로/메서드 분기 | 약 70 |
| pkg/telnyxclient (31) | httptest 서버로 HTTP 오류 상태, 잘못된 응답 본문, 전송 실패 경로 | 약 25 |
| pkg/healthcheckhandler (23) | Run 이 ctx 취소로 종료하고 tick 마다 runOnce 를 호출함 | 약 10 |
| dbhandler, providerhandler 등 | 위로 71% 에 못 미칠 때만 오류 경로를 추가 | 필요 시 |

miniredis 는 테스트에서만 import 하며 main 의 다른 서비스(v2.36.1 은 11개 서비스가 사용하고 v2.38.0 은 1개이며, 다수가 쓰는 v2.36.1 을 택한다)와 같은 버전을 쓴다. go.mod 와 go.sum 변경 외에 비테스트 .go 변경은 없다.

상세 케이스(단언은 모두 하드코딩한 기대값):

- config: (1) 환경변수 9개를 합성값으로 설정하고 `Bootstrap` 후 `LoadGlobalConfig` 가 `Get()` 에 그 값을 채운다(포트 `:2112` 기본값 포함 flag 기본값은 환경변수를 지운 별도 `cobra.Command` 에서 확인). `sync.Once` 때문에 LoadGlobalConfig 는 프로세스당 1회이므로 한 테스트 함수에서만 호출하고 viper.Reset 으로 격리한다. 값 변경을 못 잡는 자기 비교를 피하려고 기대값은 리터럴이다.
- cachehandler: ProviderSet 후 ProviderGet 이 필드 전체를 되돌리고 miniredis 에 `provider:<id>` 키와 24시간 TTL 이 생김, RouteSet/RouteGet/RouteDelete 후 키 소멸, 없는 키 Get 은 redis.Nil, 손상된 JSON 이 들어 있으면 오류, 서버 종료 후 Set/Get/Delete/Connect 는 오류, 기동 중 Connect 는 성공, NewHandler 가 주소, 비밀번호, DB 를 보관.
- listenhandler: v1_routes, v1_providers, v1_providercalls, v1_dialroute, v1_providers_setup 각 핸들러에 대해 (a) 본문이 깨진 JSON(오류 반환, 하위 핸들러 미호출), (b) 경로 ID 가 UUID 가 아님(400 응답, 하위 핸들러 미호출), (c) 하위 핸들러 오류(오류 반환 또는 errorResponse 매핑 상태 코드), (d) 잘못된 URI 파싱 오류가 해당하면. processRequest 는 미지원 경로/메서드 404 분기. NewListenHandler 는 sockHandler 주입만 확인.
- telnyxclient: httptest 로 각 호출의 비정상 상태 코드, 잘못된 JSON 본문, 연결 끊김(서버 종료 후 호출)에서 오류 반환과 오류 메시지의 일부를 단언.
- healthcheckhandler: Run 은 짧은 interval 과 mock DB 로 runOnce 가 호출됨을 확인하고 ctx 취소로 반환됨을 단언(타임아웃 가드 포함).

## 3. verify 항목

- 수기 합계 71% 이상, `go test ./...` 와 golangci-lint 통과, `bash scripts/check-test-conventions.sh` 통과.
- 모든 테스트는 기대값을 하드코딩한 상태를 비교하고 mock 의 호출 인자(gomock 엄격 매칭)로 검증한다. 호출만 하는 테스트와 자기 비교 테스트는 쓰지 않는다.
- 오류 경로 테스트는 오류 응답의 상태 코드와 하위 핸들러 미호출(EXPECT 없음)을 단언한다.
- 변이 검사: listenhandler 의 UUID 검증, cachehandler 의 키 접두어, config 의 flag 기본값을 각각 바꿔 새 테스트가 실패함을 확인한다.
- 변경 파일은 bin-route-manager/ 안(`*_test.go`, go.mod, go.sum, 본 문서)뿐이다.

측정 재현: cp.out 의 블록을 (파일, 범위) 키로 합쳐 count 최대값을 취하고, 경로에 `/gens/` 가 있거나 파일명이 `mock_` 로 시작하거나 `package` 앞 헤더에 `Code generated` 와 `DO NOT EDIT` 가 있는 파일을 제외한 뒤 커버된 statement 수 / 전체 statement 수를 계산한다. cmd/route-manager 변동(16~19개, 필요량은 70% 에 117~120, 71% 에 142~145) 때문에 최종 목표는 71% 이상으로 둔다.

달성 기준은 위 후처리 스크립트 방식의 수기 합계이며 `go test` 가 출력하는 패키지별 값이 아니다. PR 본문에 두 값과 스크립트 방식을 함께 적는다.
`go test` 패키지별 출력(예: dbhandler 68.9%)은 생성 코드를 포함하므로 수기 값(372/440, 84.5%)과 다르다. PR 본문에 둘을 함께 적는다.
