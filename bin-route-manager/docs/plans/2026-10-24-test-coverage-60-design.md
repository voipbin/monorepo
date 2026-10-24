# bin-route-manager unit test coverage 60% (VOIP-1647)

## 1. 이슈 분석 (티켓 검토)

판정: 유효. 기준선 `origin/main d68e695b5`, 수기 코드(gens/, mock_*.go, `Code generated ... DO NOT EDIT` 제외, 같은 블록 중복은 최대 count) 합계를 직접 재측정했다.

- 수기 statement 2502개 중 1340개 커버, 53.6%. 원본(생성 코드와 mock 포함) 47.8%. 60%까지 161개, 61%까지 약 187개가 더 필요하다. 티켓 수치와 일치한다.
- 관련 PR/브랜치: `gh pr list --search VOIP-1647` 결과 없음. 이 경로의 최근 커밋(VOIP-1642 lint, VOIP-1637, VOIP-1601)은 테스트 범위에 영향을 주지 않는다.
- 제품 코드 변경은 하지 않는다. 테스트 불가 구조는 없고 모든 대상이 기존 mock/인터페이스 또는 sqlite 인메모리로 주입된다.

미커버 상위 파일(미커버 statement / 전체):

| 파일 | 미커버/전체 | 성격 |
|---|---|---|
| cmd/route-control/main.go | 315/315 | CLI, DB/Redis 연결 필요. 제외 |
| cmd/route-manager/main.go | 171/190 | 데몬 기동, 외부 의존. 제외 |
| pkg/listenhandler/v1_providercalls.go | 128/128 | 핸들러 4개 전부 무테스트 |
| pkg/dbhandler/providercall.go | 114/114 | 전부 무테스트, sqlite 스키마에 테이블 없음 |
| pkg/telnyxclient/telnyx.go | 62/186 | 오류 경로, parseRejectedCountries 0% |
| internal/config/config.go | 60/60 | Bootstrap, LoadGlobalConfig |
| pkg/cachehandler | 59/59 (handler.go 46, main.go 13) | 실제 Redis 클라이언트 필요. miniredis 가 이 모듈 go.mod 에 없어 의존성 추가가 필요하므로 후순위 |
| pkg/listenhandler/main.go, v1_routes.go, v1_providers.go | 37, 36, 28 | processRequest 분기, 입력 오류 경로 |
| pkg/dbhandler/provider.go, route.go | 31, 27 | UpdateHealthStatus, 오류 경로 |
| pkg/healthcheckhandler/health.go | 23/90 | Run 루프 |
| pkg/providercallhandler/providercall.go | 21/97 | Get/List/Delete 오류 경로 |
| models/providercall/webhook.go | 9/9 | ConvertWebhookMessage, CreateWebhookEvent |

후순위 대상(cmd/ 전체 statement 505개, cachehandler 59개, 합 564개)의 미커버 약 548개를 빼도 나머지 미커버가 약 600개이므로 필요량(약 187개)은 산술적으로 달성 가능하다. 재측정 시 cmd/route-manager 의 커버 수가 실행마다 3개 내외 달라질 수 있다.

반증하지 못한 주장: 기준선 수치, 60%까지의 필요량. 틀린 주장: 없음. 모호한 부분: "cmd/ 의 main" 은 테스트하기 어려우므로 다른 패키지로 채운다(카드 본문이 허용).

## 2. 설계 (계획)

범위: bin-route-manager 안의 `_test.go` 추가, sqlite 테스트 스키마 fixture 1개(`scripts/database_scripts/table_providercalls.sql`, 마이그레이션 baseline 의 route_providercalls 를 sqlite 문법으로 옮김) 추가. 제품 코드와 시그니처 변경 없음. 외부 호출과 네트워크 의존 없음(telnyx 는 기존 테스트처럼 `httptest.Server`).

대상과 verify 항목(구현 시작 후 테스트 작성, 이 단계에서는 코드 없음):

1. `pkg/dbhandler/providercall_test.go` (추정 +100 stmt)
   - Create 후 Get 왕복에서 nil 슬라이스(Destinations, CallIDs, GroupcallIDs)가 빈 배열로 저장되고 TMCreate 가 주입된 시각이며 TMUpdate/TMDelete 가 nil.
   - Get 미존재 id 는 `ErrNotFound`.
   - List: token 미지정 시 현재 시각 사용, limit 적용, `tm_create DESC` 정렬, customer_id 와 provider_id 필터, soft delete 된 행 제외, 알 수 없는 컬럼 필터는 `could not query` 오류(sqlite 의 no such column, 오류 발생 여부만 확인), 지원하지 않는 필터 값 타입(struct)은 `could not apply filters` 오류.
   - Delete: 대상 행의 tm_delete/tm_update 설정, 미존재 id 는 `ErrNotFound`, 삭제 후 List 에서 제외.

1-b. 기존 `pkg/dbhandler/provider_test.go` 에 `ProviderUpdateHealthStatus` 테스트 추가: health_status 와 health_checked_at 이 갱신되고(ProviderUpdate 가 캐시도 갱신하므로 그 파일의 mockCache 기대 패턴을 따름) 다시 읽은 값이 일치.
2. `pkg/listenhandler/v1_providercalls_test.go` (추정 +110 stmt)
   - Post: 요청 본문이 providerCallHandler.Create 의 인자(customer, provider, flow, actions, source, destinations, anonymous)로 정확히 전달되고 200 응답 본문이 일치, 잘못된 JSON 은 오류, 핸들러 오류 전파.
   - Get: page_size/page_token 파싱, customer_id/provider_id 필터 uuid 변환, 잘못된 uuid 는 핸들러 호출 없이 오류, 핸들러 오류 전파.
   - IDGet/IDDelete: URI 에서 id 추출 후 핸들러 호출, 핸들러 오류 전파.
   - 이미 존재하는 providercalls 4개 라우트(GET 목록, POST, GET id, DELETE id)에 대한 `processRequest` 분기 테스트 추가(제품 코드 변경 아님). `processRequest` 를 거친 잘못된 JSON 본문은 400 응답이고, 핸들러 직접 호출 시에는 오류 반환이므로 구분해 검증한다. IDGet/IDDelete 의 uuid 파싱 오류는 라우트 정규식이 막아 도달 불가이므로 검증 대상에서 제외.
3. `pkg/providercallhandler` (기존 Test_Get/Test_List/Test_Delete 는 happy path 이므로 오류 경로만 신규 추가하고 함수명 충돌을 피한다): Get 의 `dbhandler.ErrNotFound` 가 NotFound 도메인 오류로 변환되는지와 그 외 오류는 그대로 전파, List/Delete 의 db 오류 경로, Delete 후 재조회 실패 경로와 이벤트 미발행 확인.
4. `models/providercall/webhook_test.go`: ConvertWebhookMessage 가 모든 필드를 옮기는지(필드별 비교, 입력과 기대값을 별도 리터럴로 작성), CreateWebhookEvent 가 유효 JSON 이고 json 키(`call_ids`, `groupcall_ids` 등)가 맞는지.
5. `internal/config`: 환경변수로 값을 주입한 뒤 Bootstrap + LoadGlobalConfig 가 모든 필드에 반영되고, 환경변수가 없으면 flag 기본값이 쓰이는지. `sync.Once` 와 viper 전역 때문에 `sync.Once` 때문에 한 테스트에서 LoadGlobalConfig 를 두 번 검증할 수 없으므로 테스트 시작에서 같은 패키지의 `once = sync.Once{}` 로 초기화하고, 환경변수 주입 검증(`t.Setenv`)과 기본값 검증(Bootstrap 후 viper.Get* 와 flag 기본값)을 분리한다. 기본값 검증은 CI 환경변수 오염에 취약하므로 관련 변수를 `t.Setenv("")` 가 아닌 `os.Unsetenv` 후 복구(t.Cleanup)로 비운다. 실패하면 확인이 어려운 전역 상태를 건드리지 않도록 이 항목은 후순위로 두고 필요량이 부족할 때만 쓴다.
6. `pkg/telnyxclient`: `parseRejectedCountries`(marker 없음, 2글자 코드만 선택, 공백 처리), `CreateOutboundVoiceProfile` 의 미테스트 응답 분기를 httptest 로 검증.
7. 필요량이 부족하면 healthcheckhandler.Run(컨텍스트 취소로 종료), providerhandler.Setup 오류 경로를 추가한다. 부족하지 않으면 추가하지 않는다.

제외: cmd/, cachehandler(후순위, 1절 참조).

금지 사항: 빈 호출 테스트, 입력과 기대값을 같은 변수로 만드는 자기비교 테스트. 모든 기대값은 독립 리터럴로 쓴다. 테스트 값은 합성값만 쓴다.

검증 절차:
- 서비스 디렉터리에서 `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`, `bash scripts/check-test-conventions.sh`(Test_ 네이밍, gomock 컨트롤러 이름 mc) 통과.
- 수기 합계를 `GOFLAGS=-mod=mod go test -count=1 -coverprofile` 와 집계 스크립트로 측정해 61% 이상(최소 60.0%). 패키지별 전후 표를 PR 본문에 쓴다.
- 변이 검사(표본, scratch 사본에서만): (a) providercall Delete 의 RowsAffected==0 분기 제거, (b) List 의 tm_delete 필터 제거, (c) listenhandler 필터 uuid 변환 오류를 무시, (d) providercallhandler.Get 의 NotFound 변환 제거. 각 변이에서 새 테스트가 실패해야 한다.
- 변경 파일은 `git diff --stat origin/main...HEAD` 로 bin-route-manager/ 안에만 있음을 확인.

롤백: 테스트와 fixture 와 문서만 추가하므로 PR revert 로 충분하다. 운영과 DB 스키마 영향 없음.

## 3. 구현 결과

항목 1, 1-b, 2, 3, 4, 6 을 구현했다. 수기 합계가 65.3% 로 목표를 넘어 항목 5(config)와 항목 7(healthcheck, providerhandler 추가)은 구현하지 않았다.
