# VOIP-1391: Extension QR Provisioning — Implementation Plan

- Date: 2026-08-24
- Ticket: VOIP-1391
- Design: docs/plans/2026-08-24-voip-1391-extension-qr-provisioning-design.md (리뷰 2연속 Approval 통과)
- Status: plan review loop 대상

## 실행 순서 개요

Phase 1 (backend, 이 워크트리) → Phase 2 (frontend, monorepo-javascript 워크트리) 순.
백엔드 PR이 머지·배포되기 전에는 프론트가 호출할 API가 없으므로, 프론트 구현은 백엔드 구현 완료
(로컬 검증 통과) 후 착수하되 PR은 각 레포에 1개씩 생성한다.

## Phase 1: Backend (monorepo, 브랜치 VOIP-1391-extension-qr-provisioning)

### Task 1.1 — bin-openapi-manager 스펙

1. `openapi/paths/extensions/id_provisioning_token.yaml` 신규:
   - `POST /extensions/{id}/provisioning-token`. 전례: `id_direct_hash_regenerate.yaml`의 구조
     (parameters/responses/401/403/404). 응답 200: `ApiManagerExtensionProvisioningToken` 참조.
2. `openapi/paths/provisioning/extension.yaml` 신규:
   - `GET /provisioning/extension?token=`. 전례: `auth/password-reset.yaml`
     (unauthenticated 프로즈, token query param `pattern: "^[0-9a-f]{64}$"`, 400).
   - 응답 200: `application/xml`, schema type string. 설명에 Linphone remote provisioning XML 명시,
     실제 경로는 `/provisioning/extension`(no /v1.0)임을 프로즈로 명시 (auth 전례와 동일).
3. `openapi/openapi.yaml`: paths 2건 `$ref` 추가 + `components/schemas`에
   `ApiManagerExtensionProvisioningToken` (token: 64hex 예시 / url: https 실값 예시 / expire: date-time).
   스펙 규칙 준수: 리프마다 실값 예시, format 지정, ID 설명에 provenance 문장.
4. 검증: `cd bin-openapi-manager && go generate ./... && go test ./...` 후 `gens/models/gen.go` 커밋 대상 포함.

### Task 1.2 — bin-api-manager config

`internal/config/main.go`:
- `rate_limit_provisioning_public_rps` (float64, 기본 5) / `rate_limit_provisioning_public_burst` (int, 기본 10),
  env `RATE_LIMIT_PROVISIONING_PUBLIC_RPS/BURST` 바인딩. 전례: `rate_limit_auth_public_*`.
- `public_base_url` (string, 기본 `https://api.voipbin.net`), env `API_PUBLIC_BASE_URL`.
  주의: bin-api-manager의 CLI 플래그는 `LoadGlobalConfig()`가 cobra 파싱 전에 실행되어 **비활성**이므로
  실질 env-var-only. 플래그 설명에 기존 rate-limit 플래그처럼 "Env var only, see API_PUBLIC_BASE_URL"
  문구를 달고, operations.md 행에도 "(env var only)" 표기 (전례: 기존 표기 방식).
- Config 구조체 필드 + viper 읽기 + 소비 지점 배선.
- **주의(빌드 순서)**: Task 1.1이 스펙을 추가한 뒤 bin-api-manager에서 `go generate ./...`를 돌리면
  `gens/openapi_server/gen.go`가 재생성되어 ServerInterface에 신규 메서드 2개가 생기고, Task 1.5의
  구현 전까지 서비스 전체 컴파일이 깨진다. 따라서 Task 1.2~1.4에서는 전체 `go generate ./...`를 돌리지
  말고 패키지 스코프(`go generate ./pkg/cachehandler/... ./pkg/servicehandler/...`,
  `go test ./pkg/...`)로 검증하며, 전체 5단계 검증은 Task 1.5 완료 후 Task 1.7에서 수행한다.
- docs 동기화: `bin-api-manager/docs/operations.md` config 표에 3키 추가.

### Task 1.3 — cachehandler

`pkg/cachehandler/main.go` + `handler.go`:
- `ProvisioningTokenSet(ctx, token string, extensionID uuid.UUID, ttl time.Duration) error`
- `ProvisioningTokenGet(ctx, token string) (uuid.UUID, error)`
- Redis key: `api-manager.provisioning_token.<token>` (설계 확정값).
- `go generate`로 mock 재생성. 전례: bin-agent-manager PasswordResetToken.
- 테스트: Set/Get 왕복, TTL 만료, 미존재 키 에러. miniredis v2가 이미 bin-api-manager go.mod에
  있으므로 miniredis 기반으로 작성한다 (기존 cachehandler 테스트 파일은 없음. 신규 작성).

### Task 1.4 — servicehandler

`pkg/servicehandler/main.go`:
- `NewServiceHandler`에 `cache cachehandler.CacheHandler` 파라미터 추가, 구조체 필드 추가.
  `cmd/api-manager/main.go` 호출부 갱신. mock 재생성.
- 상수: `ProvisioningTokenTTL = 10 * time.Minute`, `provisioningTokenLen = 32`.

`pkg/servicehandler/extension.go`:
- `ExtensionProvisioningTokenCreate(ctx, a *auth.AuthIdentity, id uuid.UUID) (*ExtensionProvisioningToken, error)`
  (기존 extension 메서드들과 동일하게 `*auth.AuthIdentity` 사용. amagent 아님):
  1) `a.IsDirect()` → `ErrDirectAccessNotSupported`
  2) `extensionGet(ctx, id)` → 404 계열 전파
  3) `hasPermission(CustomerAdmin|CustomerManager)` → 403
  4) `crypto/rand` 32B → hex 64자
  5) `cache.ProvisioningTokenSet(token, id, TTL)`
  6) 반환: token, url(`<public_base_url>/provisioning/extension?token=<token>`), expire(now+TTL, RFC3339)
- 반환 타입은 servicehandler 패키지 도메인 구조체(json 태그 포함, style A).
- `ExtensionProvisioningXMLGet(ctx, token string) ([]byte, error)` (공개 경로용):
  1) `cache.ProvisioningTokenGet(token)` → 미존재/만료 에러
  2) `reqHandler.RegistrarV1ExtensionGet(extensionID)`
  3) `tm_delete` 설정 시 에러
  4) `ConvertWebhookMessage()` 값으로 lpconfig XML 렌더링(encoding/xml 단일 경로, §3.4 형태)
- XML 렌더러는 `pkg/servicehandler/extension_provisioning_xml.go` 등 별도 파일로 분리, 골든 테스트.
- 테스트: 발급(정상/IsDirect/권한없음/타고객/미존재), XML(매핑/이스케이프/tm_delete/토큰미존재).

### Task 1.5 — HTTP 계층

- bin-api-manager 전체 `go generate ./...` (여기서 gens/openapi_server/gen.go 재생성) 후 즉시 구현:
- `server/extensions.go`: 생성된 인터페이스 메서드 `PostExtensionsIdProvisioningToken` 구현
  (전례: `PostExtensionsIdDirectHashRegenerate`. `getAuthIdentity(c)` → servicehandler 호출).
  server 계층 테스트(정상 200 / 인증 없음 401)는 기존 `server/extensions_test.go`에 추가.
- `server/provisioning.go`: `GetProvisioningExtension` 404 스텁 (전례: auth_boot.go. 실 경로 안내 문구).
  스텁 테스트는 기존 `server/auth_stubs_test.go` 테이블에 추가.
- `lib/service/provisioning.go`: 공개 핸들러 `GetProvisioningExtension(c *gin.Context)`:
  - 패키지 레벨 `validProvisioningToken = regexp.MustCompile("^[0-9a-f]{64}$")` 사전 검증 → bare 400
  - `c.MustGet(common.OBJServiceHandler)`로 servicehandler 획득 (전례: PostLogin)
  - `ExtensionProvisioningXMLGet` 호출, 모든 에러 → bare 400 (본문 동일)
  - 성공: `c.Data(200, "application/xml; charset=utf-8", xml)`
  - 자체 구조화 로그 1줄 (extension_id, 결과, ClientIP. 토큰 미포함)
  - `X-Linphone-Provisioning` 헤더는 요구하지 않되, 값이 있으면 위 구조화 로그에 포함(설계 §3.3)
- `cmd/api-manager/main.go` `runListenHTTP()`:
  - `gin.Default()` → `gin.New()` + `gin.LoggerWithConfig(SkipPaths: ["/provisioning/extension",
    "/v1.0/provisioning/extension"])` + `gin.Recovery()`
  - servicehandler 주입은 전역 `app.Use(...)`가 이미 담당하므로(그룹별 미들웨어 아님) `/provisioning`
    그룹을 그 주입 라인 **이후에** 등록하기만 하면 됨. 별도 주입 미들웨어 추가 금지(중복).
  - `provisioning := app.Group("/provisioning")` + RateLimit("provisioning_public", cfg...) + GET 라우트 등록
  - **로거 동등성 검증**: 교체 후 스킵 대상이 아닌 경로(예: /ping) 요청의 로그 라인이 기존
    gin.Default() 형식과 동일한지 확인하고 결과를 기록 (설계 §6의 광역 변경 완화책 절반).
- docs 동기화 (root CLAUDE.md 소스→문서 표):
  - `bin-api-manager/docs/routing.md`: `## Extensions` 표에 인증 POST 행 추가 + 공개 라우트용
    `## Provisioning` 섹션 신설(무인증 명시)
  - `bin-api-manager/docs/architecture.md` Middleware Stack 섹션의 rate-limit 티어 목록에
    `provisioning_public` 추가 + 같은 파일의 "Public endpoints (no authentication required)" 목록에
    신규 라우트 추가 (cmd/*/main.go 변경 → architecture.md 규칙)
  - `bin-api-manager/docs/operations.md`: Prometheus `api_manager_rate_limit_allowed_total` 라벨 목록 +
    "Rate Limiting" 티어 표에 `provisioning_public` 행 추가

### Task 1.6 — RST 문서

- `bin-api-manager/docsdev/source/extension_overview.rst`: QR provisioning 섹션 (개념, Linphone 전용,
  10분 만료, 계정 0 대체 동작).
- `extension_tutorial.rst`: 토큰 발급 curl + QR 스캔 흐름 튜토리얼.
- `extension_struct_extension.rst`: provisioning token 응답 구조(token/url/expire) 문서화 (설계 §5 대상).
- `quickstart_extension.rst`: 해당 시 간단 언급.
- 클린 빌드: `cd bin-api-manager/docsdev && rm -rf build && python3 -m sphinx -M html source build`
- `git add -f bin-api-manager/docsdev/build/`

### Task 1.7 — 검증 및 PR

- `bin-openapi-manager`, `bin-api-manager` 각각: `go mod tidy && go mod vendor && go generate ./... &&
  go test ./... && golangci-lint run -v --timeout 5m`
- 코드 리뷰 루프 (최소 3회, 2연속 Approval, CLAUDE.md 규칙)
- main 충돌 확인 (git fetch + merge-tree) 후 push, PR 생성 (제목 = 브랜치명, 본문 형식 규칙 준수)
- **머지는 대표님 명시 승인 후에만.**

## Phase 2: Frontend (monorepo-javascript, 브랜치 VOIP-1391-extension-qr-provisioning)

### Task 2.1 — 워크트리 준비

- `cd ~/gitvoipbin/monorepo-javascript && git worktree add .worktrees/VOIP-1391-extension-qr-provisioning
  -b VOIP-1391-extension-qr-provisioning origin/main`
- `square-admin`에서 `npm install` (워크트리는 node_modules 없음), 테스트 기준선 확보:
  `npm test -- --watchAll=false --forceExit 2>&1 | grep "Tests:"` (main 기준선과 비교용)

### Task 2.2 — ProvisioningQrSection 컴포넌트

- `qrcode.react` 의존성 추가 (`npm install qrcode.react`).
- `src/components/ProvisioningQrSection.js` 신규 (DirectHashSection 구조 준용):
  - props: `resourcePath`, `resourceId` (DirectHashSection 관례 준용)
  - "Generate QR" 버튼 → `Post('extensions/{id}/provisioning-token')` → `{token, url, expire}`
  - QRCodeSVG로 url 렌더링(크기 ~200px), expire 기반 카운트다운(mm:ss), 만료 시 QR 영역 흐림 +
    "Expired" 오버레이 + Regenerate 버튼
  - 실패 시 ActionFeedback (alert 금지 규칙), 로딩 스피너
  - 문구: "Scan with the Linphone mobile app. The QR code expires in 10 minutes." +
    "Scanning on a Linphone app that already has an account will replace account 0." +
    구버전 안내: "Older Linphone versions may show a one-time provisioning warning after app restart.
    Your account keeps working." (설계 §3.2 transient_provisioning 미지원 클라이언트 안내)
  - 언마운트 시 인터벌 정리
- 테스트 `src/components/__tests__/ProvisioningQrSection.test.js`:
  생성 호출/URL 렌더/카운트다운/만료 상태/에러 상태/재생성.

### Task 2.3 — extensions_detail 통합

- `src/views/extensions/extensions_detail.js`: Extension Configuration 카드의 DirectHashSection 아래에
  `<ProvisioningQrSection resourcePath="extensions" resourceId={detailData.id} />` 추가.
- 기존 `__tests__/extensions_detail.test.js` 갱신(섹션 렌더 확인 추가).

### Task 2.4 — 검증 및 PR

- 테스트 게이트: main 기준선 대비 신규 실패 0 확인 (monorepo-javascript CLAUDE.md 규칙).
- `npm run build` 성공 확인.
- 코드 리뷰 루프 (최소 3회, 2연속 Approval)
- main 충돌 확인 (`git fetch origin main` + merge-tree, 워크트리에서. 필수 규칙) 후
  push + PR 생성. **머지는 대표님 승인 후.**

## Phase 3: 배포 후 실기기 검증 (머지·배포 이후, 별도 보고)

- admin.voipbin.net에서 실제 extension QR 생성 → Linphone 최신(Android 또는 iOS) 스캔 →
  REGISTER 성공 (kamailio 로그 확인), 프로비저닝 계정이 활성 계정인지 확인.
- TTL 만료 후 URL 접속 400 확인. api-manager stdout에 토큰 미노출 확인.
- transient_provisioning 동작(앱 재시작 시 재접속 경고 없음) 확인.
- 결과를 VOIP-1391에 코멘트로 기록.

## 수용 기준 (DoD)

- [ ] 관리자 인증으로 토큰 발급 API가 token/url/expire를 반환한다
- [ ] 공개 URL이 유효 토큰에 대해 올바른 lpconfig XML을 반환하고, 그 외 모든 경우 동일한 400을 반환한다
- [ ] 토큰은 10분 후 만료된다
- [ ] api-manager 액세스 로그에 토큰이 남지 않는다
- [ ] square-admin extension 상세에서 QR 생성/표시/만료/재생성이 동작한다
- [ ] 두 레포 모두 검증 워크플로 통과 + 코드 리뷰 루프 통과
- [ ] RST/routing.md/operations.md 문서 동기화 완료
- [ ] (배포 후) Linphone 실기기 스캔으로 REGISTER 성공

## 구현 방식

- 실행은 Subagent-Driven (기본값). Task 단위로 executor 서브에이전트에 위임하되,
  각 Task 완료 시 검증 증거(테스트 출력, git diff)를 확인한 뒤 다음 Task로 진행.
- Phase 1과 Phase 2는 순차 (API 계약 확정 후 프론트 착수).
