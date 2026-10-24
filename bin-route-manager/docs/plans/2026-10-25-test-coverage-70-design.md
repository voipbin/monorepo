# bin-route-manager unit test coverage 70% (VOIP-1651)

## 1. 이슈 분석 (티켓 검토)

판정: 유효. 기준선 `origin/main 99b59c386` 을 `GOFLAGS=-mod=mod go test -count=1 -coverprofile` 로 직접 재측정했다. 수기 코드(gens/, mock_*.go, `Code generated ... DO NOT EDIT` 제외, 같은 블록 중복은 최대 count) 합계는 2502 statement 중 1632개 커버, 65.2% 이다. 원본(생성 코드와 mock 포함)은 58.0% 이다. 70% 까지 120개(1752), 71% 까지 145개(1777)가 더 필요하다. 카드 수치와 일치한다.

- 관련 PR: VOIP-1647(#1508, 60% 상향)이 `gh pr list --state all --search "route-manager coverage"` 결과 MERGED 이며 선행이다. 같은 검색에서 열린 중복 PR 은 없다. 제품 코드 변경은 하지 않는다.
- 테스트 불가 구조: cmd/route-control(315)은 DB, Redis, RabbitMQ 연결이 필요해 제외한다. cmd/route-manager 는 대부분 데몬 기동 코드라 제외하되(커버 수가 실행마다 16~19개로 달라질 수 있다), 일부 함수는 mock 으로 가능하면 여유분으로만 쓴다. 분모 2502 에는 cmd/ 505 statement(미커버 약 489)가 그대로 남는다. cmd/ 밖 미커버는 381개이며, 이 중 +120(70%), +145(71%), +150(목표)을 회수해야 하므로 회수율은 각각 31%, 38%, 39% 이다. 산술적으로 달성 가능하다. 여유를 위해 목표는 약 +150 statement 로 잡는다.
- 수기 합계는 cp.out 후처리 스크립트(제외 필터 적용)로 산출한 값이며 go test 가 출력하는 값과 다르다. 이 방법은 PR 본문에 적는다. cachehandler(miniredis)와 config(viper 전역 상태)는 위험이 있어 먼저 시도하되, , telnyxclient, dbhandler, healthcheckhandler 오류 경로만으로 70% 는 가능하다.

패키지별 기준선(수기 기준, 커버/전체): cmd/route-control 0/315, cmd/route-manager 16/190, internal/config 0/60, models/* 9/10 각각, pkg/cachehandler 0/59, pkg/dbhandler 372/440, pkg/healthcheckhandler 70/93, pkg/listenhandler 468/574, pkg/providercallhandler 94/101, pkg/providerhandler 238/257, pkg/routehandler 190/195, pkg/telnyxclient 157/188.

VOIP-1416 과의 관계: 80% 미만 패키지(cachehandler, config, dbhandler, healthcheckhandler 등)와 겹치는 부분은 PR 본문에 명시만 하고 흡수하지 않는다.

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

## 3. verify 항목

- 수기 합계 71% 이상, `go test ./...` 와 golangci-lint 통과, `bash scripts/check-test-conventions.sh` 통과.
- 모든 테스트는 기대값을 하드코딩한 상태를 비교하고 mock 의 호출 인자(gomock 엄격 매칭)로 검증한다. 호출만 하는 테스트와 자기 비교 테스트는 쓰지 않는다.
- 오류 경로 테스트는 오류 응답의 상태 코드와 하위 핸들러 미호출(EXPECT 없음)을 단언한다.
- 변이 검사: listenhandler 의 UUID 검증, cachehandler 의 키 접두어, config 의 flag 기본값을 각각 바꿔 새 테스트가 실패함을 확인한다.
- 변경 파일은 bin-route-manager/ 안(`*_test.go`, go.mod, go.sum, 본 문서)뿐이다.

측정 재현: cp.out 의 블록을 (파일, 범위) 키로 합쳐 count 최대값을 취하고, 경로에 `/gens/` 가 있거나 파일명이 `mock_` 로 시작하거나 `package` 앞 헤더에 `Code generated` 와 `DO NOT EDIT` 가 있는 파일을 제외한 뒤 커버된 statement 수 / 전체 statement 수를 계산한다. cmd/route-manager 변동(±3개, 약 0.1%p) 때문에 최종 목표는 71% 이상으로 둔다.
