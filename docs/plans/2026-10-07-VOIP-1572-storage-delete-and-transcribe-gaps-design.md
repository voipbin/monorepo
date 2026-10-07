# VOIP-1572 설계: storage 삭제 에러 반환, transcribes API 문서와 테스트 공백, -count 반복 테스트 안정화

- 상태: Draft (설계 리뷰 2-3회차 연속 승인, 구현 시작)
- 티켓: VOIP-1572
- 선행 문서: docs/plans/2026-10-07-VOIP-1572-storage-delete-and-transcribe-gaps-analysis.md (이슈 분석 리뷰 2, 3회차 연속 승인)
- 기준: origin/main 754296f72
- 범위: monorepo 한 PR. 네 건과 추가 항목 4b 를 모두 포함한다. 프로덕션 코드 변경은 D1 한 줄뿐이고 나머지는 OpenAPI 문서, 생성 산출물, 테스트다.

## 1. 결정 요약

| 항목 | 결정 |
|---|---|
| D1 | `StorageAccountDelete` 가 삭제 실패 시 `return nil, err` 로 반환한다. 실패 경로 테스트를 새 함수로 추가한다. |
| D2 | `/transcribes`, `/service_agents/transcribes` 의 POST 에 403, 404 를, GET 목록에 403 을 문서화하고 생성 산출물을 재생성한다. |
| D3 | `Test_TranscribeStart`, `Test_ServiceAgentTranscribeStart` 계열에 call 조회 실패와 권한 거부 테스트를 새 함수로 추가한다. |
| D4 | `TestRateLimit_MetricsIncrement`, `TestPostBootstrap` 이 실행마다 고유한 이름을 쓰게 해 `-count>1` 에서 통과시킨다. |
| 비목표 | 다른 엔드포인트의 403, 404 문서 공백, `ReferenceTypeConfbridge` 라벨 변이 커버리지, 다른 서비스의 `-count` 안정성은 이번 범위가 아니다. |

## 2. 상세 결정

### D1. StorageAccountDelete 에러 반환

`bin-api-manager/pkg/servicehandler/storage_accounts.go` 의 `StorageAccountDelete`:

```go
res, err := h.reqHandler.StorageV1AccountDelete(ctx, storageAccountID, 60000)
if err != nil {
    log.Errorf("Could not delete storage account. err: %v", err)
    return nil, err
}

return res, nil
```

- 로그 문구와 레벨은 바꾸지 않는다. 같은 파일의 다른 함수와 같이 에러를 그대로 반환하고 래핑하지 않는다. 서버 핸들러(`server/storage_accounts.go`)가 `abortWithServiceError` 로 변환한다(typed 에러는 그대로 통과, `ErrNotFound` 는 404, 그 외는 500).
- 호출자 영향은 분석 문서 3절에서 확인했고(프런트엔드 호출 없음, CLI 는 에러를 올바르게 보고, OpenAPI DELETE 응답에 400, 401, 403, 404, 500 이 이미 있음) 추가 대응은 필요 없다.
- 테스트: `storage_account_test.go` 에 새 함수 `Test_StorageAccountDelete_deleteFailure` 를 추가한다. 기존 `Test_StorageAccountDelete` 는 건드리지 않는다. 케이스 둘이다.
  - 삭제 RPC 일반 에러: `errors.Is(err, errFake)` 가 참이고 결과가 nil 이다.
  - 삭제 RPC typed 에러(`cerrors.NotFound(commonoutline.ServiceNameStorageManager, ...)`, 상수는 `bin-common-handler/models/outline` 에 존재함): `errors.As` 로 `*cerrors.VoipbinError` 가 유지된다.
  - 두 케이스 모두 선행 `StorageV1AccountGet` 은 정상 응답(삭제 전 조회, 권한 검사 통과 가능한 고객)으로 설정한다. 테이블 필드는 `name`, `responseErr`, `expectTyped`(bool) 정도로 최소화한다.
- 변이 시험: `return nil, err` 를 `return res, nil` 로 되돌리면 두 케이스가 모두 실패해야 한다.

### D2. OpenAPI 응답 추가와 재생성

- `bin-openapi-manager/openapi/paths/transcribes/main.yaml` 과 `paths/service_agents/transcribes.yaml` 에서
  - POST: 응답 코드 오름차순 정렬에 맞춰 401 과 409 사이에 `'403': $ref: '#/components/responses/PermissionDenied'` 와 `'404': $ref: '#/components/responses/NotFound'` 를 추가한다.
  - GET 목록: `'403': $ref: '#/components/responses/PermissionDenied'` 를 추가한다(404 는 근거가 없어 추가하지 않는다).
- 다른 필드, 설명, 파라미터는 바꾸지 않는다.
- 재생성 절차(저장소 관례)
  1. `cd bin-openapi-manager && go mod tidy && go mod vendor && go generate ./... && go mod tidy && go mod vendor`
  2. `cd ../bin-api-manager && go generate ./...`
  3. `go.mod`, `go.sum` 변경이 생기면 되돌린다(`oapi-codegen` 도구 의존 잡음).
- 기대 diff(분석 문서 2.2절 실험 기준): yaml 2개, `bin-api-manager/gens/openapi_server/gen.go`(strict-server 응답 타입 순수 추가), `gens/openapi_redoc/{openapi.json,api.html}`. `bin-openapi-manager/gens/models/gen.go` 는 변하지 않아야 한다. 이 외 파일이 바뀌면 원인을 확인한다. redoc 산출물은 이미 추적 파일이면 그대로 add 하고, 추적되지 않으면 `git add -f` 한다.
- redoc 재생성은 `npx` 로 `@redocly/cli` 를 실행한다. 도구 버전 차이로 `api.html` 에 이번 변경과 무관한 대량 diff 가 생기면 원인을 확인하고, 응답 코드 추가분 외의 변경이 크면 보고한다.
- `config_redoc/generate.go` 는 `npx` 가 없으면 조용히 건너뛰므로, 재생성 후 redoc 두 파일이 실제로 바뀌었는지 `git diff --stat` 으로 확인한다. `bin-api-manager` 의 `go generate` 는 mock 도 재생성하므로 mock 파일에 불필요한 diff 가 생기면 원인을 확인하고 이번 변경과 무관하면 되돌린다.
- 검증: `go build ./...` 와 `go test ./...` 를 `bin-api-manager` 에서 실행하고, `gen.go` diff 에 기존 타입 수정이 없고 추가만 있는지 확인한다. 응답 타입 이름이 분석 문서의 목록과 같은지 grep 한다.
- RST 문서(`bin-api-manager/docsdev/source/transcribe*.rst`)에는 transcribes 의 상태코드 표가 없어(403, 404 언급 없음) 변경하지 않는다. 구현 시 한 번 더 grep 으로 확인한다.
- 충돌 대응: main 이 이동해 생성 산출물이 충돌하면 손으로 병합하지 않고 위 절차로 재생성한다.

### D3. Test_TranscribeStart 커버리지

새 함수만 추가하고 기존 테스트 함수는 수정하지 않는다. 모두 `Test_` 이름, `reflect.DeepEqual` 또는 `errors.Is`, 컨트롤러 이름 `mc` 를 따른다(`scripts/check-test-conventions.sh`, CI 무조건 게이트 `check-test-conventions`).

`transcribe_test.go`

- `Test_TranscribeStart_callLookupFailure`: `referenceType: "call"`, `CallV1CallGet` 이 `(nil, errFake)` 를 반환한다. 기대: `errors.Is(err, errFake)`, 결과 nil. `TranscribeV1TranscribeStart` 기대를 설정하지 않아 호출되면 gomock 이 실패시킨다. 신원은 `PermissionCustomerAdmin`.
- `Test_TranscribeStart_permissionDenied`: 두 케이스.
  - 조회 전 권한 거부: 신원이 `PermissionCustomerAgent`(관리자, 매니저 아님, `Admin|Manager` 비트와 겹치지 않음), 어떤 `reqHandler` 기대도 설정하지 않는다. 기대: `errors.Is(err, serviceerrors.ErrPermissionDenied)`, 결과 nil. 조회 전 `hasPermission`(`transcribe.go:153`)을 제거하면 `CallV1CallGet` 이 예상 밖 호출이 되어 실패한다.
  - 조회 후 소유 고객 불일치: 신원은 고객 A 의 `PermissionCustomerAdmin`, `CallV1CallGet` 이 고객 B 의 call 을 반환한다. 기대: `errors.Is(err, serviceerrors.ErrPermissionDenied)`, 결과 nil, `TranscribeV1TranscribeStart` 미호출. 소유 검사(`transcribe.go:243`)를 제거하면 예상 밖 `TranscribeV1TranscribeStart` 호출로 실패한다. (`hasPermission` 은 `ProjectSuperAdmin` 이거나 고객이 같아야 통과하므로 두 고객이 달라야 한다.)

`serviceagent_transcribe_test.go`

- `Test_ServiceAgentTranscribeStart_callLookupFailure`: 위와 같은 구성, 신원은 `PermissionCustomerAgent`(에이전트 신원).
- `Test_ServiceAgentTranscribeStart_permissionDenied`: 두 케이스.
  - 에이전트가 아닌 신원(`auth.NewAccesskeyIdentity(...)`): 기대 `ErrAuthenticationRequired`, `reqHandler` 기대 없음. `a.IsAgent()` 검사(`serviceagent_transcribe.go:95`) 제거 시 실패한다.
  - 조회 후 소유 고객 불일치: 고객 A 의 `PermissionCustomerAgent` 에이전트, 고객 B 의 call. 기대 `ErrPermissionDenied`, `TranscribeV1TranscribeStart` 미호출(`:207`). 소유 검사 제거 시 실패한다.
  - 조회 전 `PermissionAll` 권한 검사(`:111`)는 테스트하지 않는다. `Agent.HasPermission` 이 `PermissionAll` 요청에 대해 무조건 `true` 를 반환하고(`bin-agent-manager/models/agent/agent.go:46-48`) 고객 비교 대상이 `a.CustomerID` 자기 자신이라, 에이전트 신원으로는 이 검사가 거부할 수 없다(프로덕션 경로상 도달 불가인 방어 코드이며, 이 검사를 제거해도 `NewAgentIdentity` 로 만든 모든 신원에서 관측 가능한 동작이 바뀌지 않는 동치 변이). 이 검사를 제거해도 관측 가능한 동작이 바뀌지 않으므로 테스트 대상에서 제외한다. 코드 정리는 이번 범위가 아니다.

공통 구현 결정
- 추가 import: `serviceagent_transcribe_test.go` 는 `NewAccesskeyIdentity` 용으로 `csaccesskey "monorepo/bin-customer-manager/models/accesskey"` 가 필요하다. `storage_account_test.go`(D1)는 `errors`, `cerrors`(`monorepo/bin-common-handler/models/errors`), `commonoutline` 이 필요하다. 컴파일러가 알려주므로 구현 시 맞춘다.
- VOIP-1569 에서 추가한 `*_referenceLookupFailure` 와 같은 구조(함수 지역 `errFake`, 테이블 구조체, 파일마다 독립 정의, 공용 헬퍼 없음)를 쓴다. 기존 import 외에 필요한 것은 `errors`, `serviceerrors`, `amagent` 로, 해당 파일에 이미 있으면 중복 추가하지 않는다.
- call 응답 객체는 기존 정상 케이스처럼 `cmcall.Call{Identity: ..., Status: cmcall.StatusProgressing}` 를 쓴다.
- 변이 시험(구현 단계에서 수행하고 PR 본문에 기록)
  - call 분기 `break` 제거(`transcribe.go:208`, `serviceagent_transcribe.go:173`)가 각 `*_callLookupFailure` 에 잡힌다. 패닉은 테스트 바이너리를 중단시키므로 케이스별 `-run` 으로 개별 확인한다.
  - `TranscribeStart` 의 조회 전 권한 검사 제거와 소유 검사 제거, `ServiceAgentTranscribeStart` 의 `IsAgent` 검사 제거와 소유 검사 제거가 각각 `*_permissionDenied` 의 해당 케이스에 잡힌다. `ServiceAgentTranscribeStart` 의 조회 전 `PermissionAll` 검사 제거는 위 이유로 동치 변이라 매핑에서 제외한다.

### D4. -count 반복 안정화

두 테스트 모두 `func` 선언 줄은 바꾸지 않고 본문만 바꾼다(컨벤션 게이트 Rule 1 이 추가된 줄의 `func Test…` 이름을 검사하므로, 이름이 `Test_` 로 시작하지 않는 기존 두 함수를 건드리면 기존 위반이 새로 걸린다).

- `lib/middleware/ratelimit_test.go` `TestRateLimit_MetricsIncrement`: 고정 문자열 `"test_metrics_increment"`(`RateLimit(...)` 인자 1곳과 `WithLabelValues(...)` 6곳, 총 7곳)를 함수 상단의 지역 변수 `tier` 로 바꾼다. 값은 파일 상단에 추가하는 패키지 변수 `metricsIncrementRuns atomic.Int64` 의 `Add(1)` 결과로 만든 `fmt.Sprintf("test_metrics_increment_%d", n)` 이다. 실행마다 새 레이블이라 전역 카운터가 0 에서 시작한다. `fmt`, `sync/atomic` import 가 없으면 추가하고 `gofmt -w` 로 정렬한다. 단언 로직과 기대값은 바꾸지 않는다.
- `internal/config/main_test.go` `TestPostBootstrap`: `globalConfig.PrometheusEndpoint = "/test-metrics"` 를 실행마다 고유한 경로로 바꾼다. 같은 방식으로 파일 상단의 패키지 변수 `postBootstrapRuns atomic.Int64` 와 `fmt.Sprintf("/test-metrics-%d", n)` 를 쓴다. 기본 `http.DefaultServeMux` 의 같은 패턴 중복 등록 패닉이 사라진다. `PrometheusListenAddress` 는 기존 `:0` 을 그대로 둔다.
- 검증: `go test ./lib/middleware/ ./internal/config/ -count=5 -race -shuffle=on` 이 통과해야 한다. 변이 시험: 고정 문자열로 되돌리면 `-count=2` 에서 다시 실패해야 한다.
- 프로덕션 코드(`ratelimit.go`, `internal/config/main.go`)는 바꾸지 않는다.

## 3. 전체 검증 계획 (구현 단계)

- `gofmt -l`, `go vet`, `go build ./...`, `go test ./... -count=1` 과 `-count=2`(`bin-api-manager` 전체). `-count=2` 에서 실패하는 테스트가 없어야 한다.
- `bash scripts/check-test-conventions.sh`(커밋된 추가 줄 기준, CI 무조건 게이트)를 로컬에서 실행한다. `make` lint 타깃(`scripts/check-docs.sh`, `scripts/check-error-envelope.sh`)도 실행한다(CI 설정이 아니라 `Makefile` 에 정의). OpenAPI 를 바꾸므로 `bin-openapi-manager-validate`(생성 모델 재생성 diff 검사, `bin-openapi-manager` 경로 변경 시 CI 조건부 실행)에 상응하는 확인으로 재생성 후 `bin-openapi-manager/gens/models/gen.go` 불변을 확인한다.
- `git fetch origin main` 후 충돌 확인, `gh pr view --json mergeable`.

## 4. 위험

- D1 은 운영 동작 변경이다(삭제 실패가 200 과 null 에서 에러 응답으로). 호출자 영향이 없음을 분석 문서에서 확인했다. 정상 삭제 경로는 바뀌지 않는다.
- D2 의 생성 산출물이 main 이동으로 충돌하면 재생성한다.
- D4 는 테스트 전용 변경이라 운영 영향이 없다. 레이블, 경로가 실행마다 늘어나 전역 레지스트리와 mux 에 항목이 쌓이지만 테스트 프로세스 안에서만 존재한다.
- 한 PR 에 네 건이 섞여 리뷰 부담이 있으나 대표님 지시이며, 항목 간 파일 겹침이 없어 서로 영향을 주지 않는다.

## 5. PR 계획

- 브랜치 `VOIP-1572-Fix-storage-delete-error-and-close-transcribe-gaps`, 제목은 브랜치명과 같게, 본문은 서술 한 문단과 `project:` 접두 불릿(헤더, Test plan, AI 속성 없음).
- 본문에 포함할 것: 각 항목의 변경, 변이 시험 결과, 4b 가 지시 밖 추가 항목임, 처리하지 않은 항목(위 비목표), 별도 후속 후보.
- 머지는 대표님 지시를 기다린다.
