# VOIP-1572 이슈 분석: storage 삭제 에러 삼킴, transcribes API 문서·테스트 공백, -count 반복 테스트 실패

- 상태: Draft (이슈 분석 리뷰 3회차 대기, 2회차 비차단 반영)
- 티켓: VOIP-1572 (Bug, 2026-10-07 생성, In Progress)
- 기준: origin/main 754296f72
- 출처: VOIP-1569 코드 리뷰 중 발견한 네 건을 대표님이 한 티켓, 한 PR 로 묶어 처리하도록 지시하셨다.

## 1. 결론 요약

| # | 항목 | 현재 main 에서 유효한가 | 처리 |
|---|---|---|---|
| 1 | `StorageAccountDelete` 가 삭제 실패를 삼킴 | 유효 | 에러 반환 + 테스트 |
| 2 | `/transcribes` OpenAPI 에 403, 404 누락 | 유효 | 응답 추가 + 코드 생성 |
| 3 | `Test_TranscribeStart` call 조회 실패, 권한 거부 커버리지 없음 | 유효 | 테스트 추가 |
| 4 | `TestRateLimit_MetricsIncrement` 가 `-count>1` 에서 실패 | 유효, 재현 | 테스트의 전역 상태 의존 제거 |
| 4b | `TestPostBootstrap` 도 `-count>1` 에서 실패 (추가 발견) | 유효, 재현 | 같은 부류라 함께 처리 |

네 건은 서로 독립이라 한 PR 에 묶어도 서로 영향이 없다. 1번만 운영 동작이 바뀐다(실패한 삭제 요청이 200 과 null 대신 에러 응답을 받는다). 나머지는 문서와 테스트다.

## 2. 항목별 사실 확인 (모두 origin/main 754296f72 기준)

### 2.1 StorageAccountDelete 에러 삼킴

`bin-api-manager/pkg/servicehandler/storage_accounts.go` 의 `StorageAccountDelete`(105행 부근)는 다음과 같다.

```go
res, err := h.reqHandler.StorageV1AccountDelete(ctx, storageAccountID, 60000)
if err != nil {
    log.Errorf("Could not delete storage account. err: %v", err)
}

return res, nil
```

- `StorageV1AccountDelete`(`bin-common-handler/pkg/requesthandler/storage_accounts.go:90`)는 실패 시 `nil, err` 를 반환한다. 따라서 삭제가 실패하면 이 함수는 `nil, nil` 을 반환한다.
- 유일한 호출자 `server/storage_accounts.go:152` 는 `err == nil` 이면 `c.JSON(200, res)` 를 호출한다. 삭제 실패 시 HTTP 200 과 본문 `null` 이 내려간다. 호출자는 삭제가 성공했다고 오해한다.
- 패닉은 없다. 같은 파일의 다른 함수들은 실패 시 `return nil, err` 를 쓴다(예: 같은 함수 안의 `storageAccountGet` 실패 처리).
- 기존 테스트 `Test_StorageAccountDelete`(`storage_account_test.go:78`)는 정상 케이스 하나뿐이다.

### 2.2 `/transcribes` OpenAPI 의 403, 404 누락

- `bin-openapi-manager/openapi/paths/transcribes/main.yaml` 의 POST 응답은 200, 400, 401, 409, 500 이다. GET 목록 응답은 200, 400, 401, 500 이다.
- `paths/service_agents/transcribes.yaml` 의 POST, GET 도 같은 상태코드만 정의한다.
- 같은 디렉터리의 `paths/transcribes/id.yaml` 은 이미 403(`PermissionDenied`), 404(`NotFound`)를 정의한다. 공통 컴포넌트 `PermissionDenied`, `NotFound` 가 `openapi.yaml` 의 `components/responses` 에 있다.
- 실제 동작: POST 는 권한 거부 시 `ErrPermissionDenied`(403), 참조 리소스가 없으면 404(VOIP-1569 수정 이후 recording, conference 조회 실패가 404 로 내려간다)를 반환한다. 서버 코드 근거는 `servicehandler/transcribe.go` 의 `hasPermission` 검사와 `transcribeGetResourceInfo`, `serviceagent_transcribe.go` 의 동일 구조다. GET 목록(`TranscribeList`)은 참조 리소스를 조회하지 않으므로 404 근거가 없고 권한 거부 시 403 만 반환한다. 따라서 응답 추가 대상은 POST 의 403, 404 와 GET 목록의 403 이다.
- OpenAPI 를 바꾸면 `bin-openapi-manager/gens/models/gen.go` 와 `bin-api-manager/gens/openapi_server/gen.go`, `gens/openapi_redoc/{openapi.json,api.html}` 이 재생성 대상이다. 생성 범위는 실험으로 확인했다(임시 worktree, 두 yaml 의 GET, POST 에 403, 404 를 모두 추가한 경우로 `go generate`). `bin-openapi-manager/gens/models/gen.go` 는 변하지 않는다. `bin-api-manager` 의 서버 생성 설정이 `strict-server: true`(`openapi/config_server/config.generate.yaml`)라서 `gens/openapi_server/gen.go` 에 상태코드별 응답 타입(`GetTranscribes403JSONResponse`, `PostTranscribes403JSONResponse`, `PostTranscribes404JSONResponse`, 같은 이름의 `ServiceAgentsTranscribes` 계열)이 추가된다. 1회차 실험은 GET 에도 404 를 넣은 경우였고 그때 약 112줄이었으므로, GET 404 를 제외한 실제 추가량은 이보다 약간 적다. 기존 타입은 바뀌지 않는 순수 추가이며 호환성 문제는 없다. redoc 산출물(`gens/openapi_redoc/{openapi.json,api.html}`)은 구현 단계에서 재생성해 확인한다.

### 2.3 Test_TranscribeStart 커버리지 공백

- `Test_TranscribeStart`(`transcribe_test.go:225`)와 `Test_ServiceAgentTranscribeStart`(`serviceagent_transcribe_test.go:148`)는 각각 정상 케이스 하나다. VOIP-1569 에서 추가한 `*_referenceLookupFailure` 는 conference, recording 조회 실패만 다룬다.
- VOIP-1569 코드 리뷰 2회차 변이 시험에서 다음 변이가 기존·신규 테스트에 잡히지 않고 생존했다.
  - call 분기의 `break` 제거(call 조회 실패 후 nil 역참조 패닉이 다시 생겨도 테스트가 통과한다).
  - 조회 전 권한 검사(`hasPermission`) 제거 또는 순서 변경.
- 서버 코드의 권한 거부 경로는 두 곳이다. (1) 조회 전 호출자 권한 검사: `TranscribeStart` 는 관리자, 매니저 권한이 아니면 `ErrPermissionDenied`(`transcribe.go:155`), `ServiceAgentTranscribeStart` 는 에이전트가 아니면 `ErrAuthenticationRequired`, `PermissionAll` 권한 검사 실패 시 `ErrPermissionDenied`(`serviceagent_transcribe.go:113`). (2) 조회 후 참조 리소스의 소유 고객 검사(`transcribe.go:245`, `serviceagent_transcribe.go:209`).

### 2.4 `-count>1` 반복 실행 실패

재현(worktree, origin/main 754296f72):

```
go test ./lib/middleware/ -run TestRateLimit_MetricsIncrement -count=2
--- FAIL: TestRateLimit_MetricsIncrement
    ratelimit_test.go:234: expected pre-initialized allowed series at 0, got 1
    ratelimit_test.go:237: expected pre-initialized rejected series at 0, got 1

go test ./internal/config/ -run TestPostBootstrap -count=2
--- FAIL: TestPostBootstrap
panic: pattern "/test-metrics" ... conflicts with pattern "/test-metrics"
```

- `TestRateLimit_MetricsIncrement`(`lib/middleware/ratelimit_test.go:224`): `RateLimit("test_metrics_increment", 1, 1)` 의 prometheus 카운터는 전역(`promRateLimitAllowedTotal`, `promRateLimitRejectedTotal`, `ratelimit.go:22-44`)이고 레이블이 고정 문자열이라, 같은 프로세스의 두 번째 실행에서 시작값이 0 이라는 단언이 깨진다.
- `TestPostBootstrap`(`internal/config/main_test.go:206`): `PostBootstrap` 이 `globalConfig.PrometheusEndpoint`("/test-metrics")를 기본 `http.DefaultServeMux` 에 등록(`internal/config/main.go:263`)하므로 두 번째 호출이 같은 패턴 충돌로 패닉한다.
- 두 건 모두 프로세스 전역 상태를 쓰는 테스트가 반복 실행에 대응하지 못하는 같은 부류이며 프로덕션 동작 결함이 아니다. `-count=2` 로 `bin-api-manager` 전체를 돌렸을 때 실패하는 테스트는 이 둘뿐이다(`pkg/servicehandler`, `server` 는 통과).
- CI 는 `-count` 를 쓰지 않아 가려져 있었다. VOIP-1569 PR 리뷰에서 `-shuffle`, `-count` 반복 안정성을 검증 수단으로 쓰고 있어 이 불안정은 리뷰 신뢰도에 직접 영향을 준다.

## 3. 진행 타당성

- 네 건 모두 현재 main 에서 유효하다. 이미 해결된 것은 없다.
- 위험: 1번은 운영 동작 변경(삭제 실패가 에러 응답으로 바뀜)이다. 호출자 영향은 `~/gitvoipbin` 하위 저장소에서 확인했다. `monorepo-javascript`(square-admin 등)에는 이 엔드포인트를 호출하는 코드가 없고 타입 정의뿐이다. `cli`(`internal/commands/storage_accounts.go:143`)는 비 2xx 응답을 에러로 처리하며, 현재는 삭제가 실패해도 "deleted" 를 출력하지만 수정 후에는 `could not delete storage account` 로 올바르게 실패를 알린다. OpenAPI `paths/storage_accounts/id.yaml` 의 DELETE 는 이미 400, 401, 403, 404, 500 을 정의하므로 스펙 변경이나 코드 생성은 필요 없다. 따라서 호환성 영향은 없다.
- 2번은 코드 생성 산출물 변경을 동반하므로 main 이 이동하면 생성 파일 충돌이 날 수 있다. 충돌 시 손으로 병합하지 않고 재생성한다.
- 의존성: 네 건은 서로 의존하지 않는다. VOIP-1569 PR(#1370, #158)은 모두 머지되어 이 작업과 파일 충돌이 없다.
- 대안: 네 건을 별개 PR 로 나누는 안은 대표님이 한 티켓, 한 PR 로 처리하도록 명시하셨으므로 채택하지 않는다.
- 4b 는 대표님이 지정한 네 건 밖의 추가 항목이다. 4번과 같은 부류(프로세스 전역 상태 때문에 `-count>1` 에서 실패)이고 수정이 테스트 파일에 한정되어 함께 처리하며, PR 본문과 보고에 추가 항목임을 명시한다.
- 범위 밖으로 남기는 것: `Test_TranscribeStart` 의 `ReferenceTypeConfbridge` 라벨 변이 생존(정상 케이스 추가가 필요하며 이번 항목의 call 조회 실패와 권한 거부와는 별개), `/transcribes` 외 다른 엔드포인트의 403, 404 문서 공백, `TestRateLimit_MetricsIncrement` 외 다른 전역 prometheus 의존 테스트(`bin-api-manager` 범위에서 `-count=2` 로 전수 확인했고 위 둘 외에는 실패하지 않았다. 다른 서비스의 같은 문제는 확인 범위 밖이다).
