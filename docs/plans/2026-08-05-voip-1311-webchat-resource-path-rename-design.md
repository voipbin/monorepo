# VOIP-1311: webchat_widgets / webchat_sessions / webchat_messages 리소스 경로 하이픈 개명 설계

- Jira: [VOIP-1311](https://voipbin.atlassian.net/browse/VOIP-1311)
- 작성일: 2026-08-05 (Round 1 리뷰 반영으로 개정)
- CPO 결정: 옵션3 (리소스 경로 자체를 하이픈으로 전면 개명, deprecation을 통한 breaking change)

## 배경

`bin-openapi-manager/openapi/openapi.yaml`의 최상위 리소스 경로 중 `webchat_widgets`, `webchat_sessions`, `webchat_messages` 3개만 언더스코어 표기이고, 다른 모든 리소스는 하이픈 컨벤션을 따름. 서브액션(`direct_hash_regenerate`)도 다른 리소스의 `direct-hash-regenerate`와 표기가 다름.

이 문서는 **REST API 경로**(`/webchat_widgets` → `/webchat-widgets` 등)만을 하이픈으로 통일하는 마이그레이션을 설계한다. **admin SPA 라우트 경로(`/resources/webchat_widgets/...`)와 `src/views/webchat_widgets/` 디렉터리명은 이 작업의 스코프가 아니다** (아래 "스코프 아님" 참조).

## 현황 조사 결과

### 1. 영향 범위 — 내부 RPC 계층은 스코프 아님

`bin-webchat-manager/pkg/listenhandler/main.go`의 내부 RabbitMQ RPC 경로는 애초에 `/v1/widgets`, `/v1/sessions` (언더스코어 아님)이고, 서브액션도 이미 `/v1/widgets/{id}/direct-hash-regenerate` (하이픈)로 되어 있음. 언더스코어 문제는 `bin-api-manager`의 외부 REST 계층에만 존재.

### 2. 공개 임베드 위젯 — 캐시 리스크 재평가 (Round 1에서 정정)

최초 초안은 임베드 스니펫이 직접 번들 URL(`webchat-widget-runtime.bundle.js`)을 가리킨다고 잘못 기술했다. 실제로는:

- `square-admin/src/views/webchat_widgets/detail.js:50`, `create.js:94`가 생성하는 고객 임베드 스니펫은 `https://admin.voipbin.net/webchat/embed.js`.
- `square-admin/nginx.conf:28-31`가 `/webchat/embed.js`를 번들의 alias로 서빙하며 `Cache-Control: public, max-age=300`(5분) — **정상 임베드 경로의 캐시 지연은 수 분 단위**.
- 그러나 `nginx.conf:20-23`의 `*.js` 전체에 걸린 `expires 1y; Cache-Control: public, immutable` 규칙이 번들 파일 자체에도 적용되어, **직접 번들 URL로 접근하면 1년간 불변 캐시**된다. admin 자체 테스트 하네스(`public/webchat-widget-test.html:211`)가 이 직접 URL을 사용 중.
- 번들은 `.gitignore` 처리되고 `scripts/build-widget.js`가 prebuild 시점에 생성, 버전 고정/SRI 해시 없음 — 고객 셀프호스팅 가능성은 낮음.

**리스크 재평가**: 정상 임베드 경로(`/webchat/embed.js`)의 캐시 지연은 5분 수준으로 매우 짧다. 진짜 롱테일은 (a) 오래 열려 있는 브라우저 탭이 계속 구버전 JS를 실행 중인 경우, (b) 직접 번들 URL(1년 불변 캐시)에 접근하는 소수 케이스다. 이번 작업에서 **직접 번들 URL의 `immutable` 캐시 규칙도 함께 수정**하여 (b)를 제거한다(예: 번들 URL에 콘텐츠 해시를 붙이거나, `/webchat/embed.js` alias와 동일한 5분 캐시 정책 적용).

## 스코프 아님 (명시적 제외)

- **admin SPA 라우트 경로**(`/resources/webchat_widgets/list`, `/detail` 등)와 `src/views/webchat_widgets/` 디렉터리명 — 이들은 API 경로가 아니라 프론트엔드 내부 라우팅이며, `resourceLinks.js`, `routes.js`, `_nav.js`, `navHelpManifest.js`, `Breadcrumb.jsx`가 이 문자열을 폴더 프리픽스/북마크/딥링크 매칭에 사용 중. 여기를 건드리면 기존 북마크·딥링크·헬프 매뉴얼 링크가 깨진다. 이번 작업은 API 호출 URL만 바꾼다.
- 직접 토큰(direct hash) 권한 스코프는 URL 경로가 아니라 클레임 문자열 `"webchat_session"`으로 판별(`bin-api-manager/pkg/servicehandler/boot.go`, `webchat_session.go`, `webchat_message.go`) — 변경 불필요.
- WebSocket 토픽 문자열(`customer_id:<id>:webchat_session:<id>`)과 웹훅 이벤트 타입명(`webchat_message_created`, `webchat_session_ended`)은 REST 경로와 독립적 — 변경 불필요.
- DB 테이블명/Alembic 마이그레이션은 API 계약이 아니므로 변경 불필요.
- square-talk, square-meet, square-dev, square-main, sandbox, api-validator/SDK/CLI 어디에도 이 경로에 대한 참조 없음(확인 완료) — 영향 없음.

## 코드 생성 충돌 해결 방침 (Round 1/2 blocking finding 반영)

oapi-codegen은 경로의 `-`와 `_`를 동일하게 취급해 operationId를 생성하므로, 스펙에 `/webchat_widgets`와 `/webchat-widgets`를 동시에 선언하면 `ServerInterface`에 동일 이름 메서드가 중복 생성되어 컴파일이 깨진다.

**채택 방식**: OpenAPI 스펙은 **신규 하이픈 경로만** 선언한다(구 언더스코어 경로는 스펙에서 제거).

구 경로는 `bin-api-manager/cmd/api-manager/main.go`에서 **기존 `v1` 라우터 그룹**(258-260행, `RateLimit`+`Authenticate` 미들웨어가 이미 적용된 그룹)에 수동 등록한다. **주의: `server/webchat_*.go`의 `ServerInterface` 메서드(예: `GetWebchatWidgets(c *gin.Context, params GetWebchatWidgetsParams)`)는 `gin.HandlerFunc` 타입이 아니라 직접 라우팅에 바인딩할 수 없다.** 파라미터 바인딩(쿼리/경로 파라미터 검증, `BindingErrorHandler` 연결)은 생성된 `ServerInterfaceWrapper`가 담당하므로, 이를 직접 재사용한다:

```go
// main.go, RegisterHandlersWithOptions(v1, appServer, ...) 호출부 근처
legacyWrapper := openapi_server.ServerInterfaceWrapper{
    Handler:      appServer,
    ErrorHandler: server.BindingErrorHandler,
}
// v1 그룹에 등록 -- RateLimit/Authenticate 미들웨어를 그대로 상속받아야 함.
// app이나 별도 그룹에 등록하면 인증이 빠져 인증 우회가 됨.
v1.GET("/webchat_widgets", webchatDeprecationMarker(), legacyWrapper.GetWebchatWidgets)
v1.GET("/webchat_widgets/:id", webchatDeprecationMarker(), legacyWrapper.GetWebchatWidgetsId)
// ... 나머지 webchat_widgets/webchat_sessions/webchat_messages 전체 경로 + 서브액션 동일 패턴
```

이렇게 하면 파라미터 바인딩/에러 처리가 신규 경로와 완전히 동일하게 유지되고, 수동 바인딩을 직접 구현하지 않는다. 핸들러 로직 중복 없음, 공개 문서(redoc/swagger)에는 신규 경로만 노출.

이 패턴은 `bin-api-manager`에 현재 선례가 없는 신규 패턴이다(`/ping`, `/auth/*`, 문서 엔드포인트만 수동 등록되어 있고 나머지는 전부 생성된 라우터를 통함). 구현 시 `main.go`에서 이 레거시 별칭 등록부를 `RegisterHandlersWithOptions` 호출 바로 아래 명확히 구분된 블록으로 묶고, `// TODO(Phase 3): remove after <sunset date>` 주석으로 표시한다.

## 변경 지점

| 계층 | 파일/위치 | 변경 내용 |
|---|---|---|
| OpenAPI 스펙 | `bin-openapi-manager/openapi/openapi.yaml` (8350-8367행), `openapi/paths/webchat_widgets/` → `webchat-widgets/` 등 3개 리소스 디렉터리 | 경로를 하이픈으로 변경 (언더스코어 경로는 스펙에서 완전히 제거, 구 경로는 수동 라우팅으로만 유지) |
| 생성 코드 | `bin-openapi-manager/gens/models/gen.go` (먼저), `bin-api-manager/gens/openapi_server/gen.go` (다음) | `go generate ./...`를 bin-openapi-manager → bin-api-manager 순서로 재실행 |
| 공개 API 문서 | `bin-api-manager/gens/openapi_redoc/openapi.json`, `api.html` | 재생성 시 자동 반영 (api.voipbin.net/docs에 신규 경로만 노출) |
| 레거시 라우팅 | `bin-api-manager/cmd/api-manager/main.go` (RegisterHandlersWithOptions 호출부 근처) | 구 언더스코어 경로 3종(+서브액션)을 Gin에 수동 등록, 신규 경로와 동일 핸들러 바인딩. 등록 지점에 deprecation 카운터/헤더 부착 |
| REST 핸들러 | `bin-api-manager/server/webchat_widgets.go`, `webchat_sessions.go`, `webchat_messages.go` | 변경 없음 (신규+구 라우팅 모두 동일 핸들러 재사용) |
| RST 문서 | `webchat_overview.rst`, `webchat_struct_session.rst`, `webchat_struct_message.rst`, `webchat_struct_widget.rst`, `direct_hash_overview.rst`, `auth_overview.rst` | 신규 경로로 갱신 + 구 경로 deprecated 안내 추가, 클린 Sphinx 리빌드 + `git add -f docsdev/build/` |
| square-admin API 호출부 | `views/webchat_widgets/{list,create,detail,sessions_list,sessions_list_global,sessions_detail,useWebchatSessionMessages}.js`, `src/provider.js`(347-399행, RBAC 리스트 3곳) | 신규 하이픈 경로로 전환 |
| square-admin 캐시 키 | `src/provider.js`의 `LoadResource()`(347-367행), `src/utils/resourceCache.js:25` | **API 리소스 문자열을 바꾸지 않고 캐시 키와 분리한다** (Round 2에서 문자열을 맞추는 방식은 이미 로그인된 세션의 localStorage가 갱신 시점까지 구 키만 갖고 있어 문제를 없애지 못하고 옮기기만 함을 지적받음). `LoadResource(resource, cacheKey)`처럼 캐시 키를 별도 인자로 받게 하여, API 호출 경로는 `webchat-widgets`로 바뀌어도 localStorage 키는 계속 `webchat_widgets`로 유지 — 기존 세션 마이그레이션 자체가 불필요해짐. `resourceCache.js`는 변경 없음. |
| square-admin 위젯 런타임 | `src/webchat-widget-runtime/client.js` | 신규 경로로 전환, 재빌드. `webchat-widget-runtime.esm.js`는 `WidgetPreview.js`가 webpack import로만 사용하고 URL로 fetch되지 않으므로 캐시 정책 변경 불필요 |
| nginx 캐시 정책 | `square-admin/nginx.conf` (28-31행의 `/webchat/embed.js` exact-match 블록과 동일한 패턴 추가) | 번들 직접 URL(`webchat-widget-runtime.bundle.js`)에 대해서도 `location = /webchat-widget-runtime.bundle.js { add_header Cache-Control "public, max-age=300"; }` 형태의 exact-match 블록을 추가해, 20-23행의 `*.js` 전역 1년 불변 캐시 규칙보다 우선 적용되게 한다(콘텐츠 해시 방식은 `build-widget.js`가 고정 파일명을 생성하고 `webchat-widget-test.html`이 그 고정 URL을 참조하므로 채택하지 않음). |
| 공개 문서 스키마 설명 | `bin-openapi-manager/openapi/openapi.yaml:2465,2481`, `openapi/paths/webchat_sessions/main.yaml:15`, `openapi/paths/webchat_messages/main.yaml:15` | 스키마/경로 description 내 구 경로 언급(예: "POST /webchat_sessions", "direct_hash_regenerate") 전부 신규 경로로 갱신 (redoc/openapi.json, `gens/models/gen.go`에 그대로 노출되므로) |
| 테스트 | `views/webchat_widgets/__tests__/*.test.js`, `webchat-widget-runtime/__tests__/client.test.js`, `resourceCache.test.js` | 경로 리터럴 문자열 갱신 |

## 마이그레이션 전략

### Phase 1 — 신규 하이픈 경로 추가 + 구 경로 수동 라우팅 유지 (이번 구현 스코프)
1. OpenAPI 스펙을 하이픈 경로로 전환 (구 경로는 스펙에서 제거).
2. `bin-api-manager`에서 구 경로를 수동 라우팅으로 등록, 신규 경로와 동일 핸들러 사용.
3. square-admin 관리 콘솔의 API 호출부 + 위젯 런타임을 신규 경로로 전환·재빌드.
4. `resourceCache.js` 캐시 키를 `provider.js`와 일치시킴.
5. nginx의 번들 직접 URL 캐시 정책 완화.
6. RST 문서 갱신.
7. 구 경로 요청에 대해 Prometheus 카운터(경로/메서드 라벨) + `Deprecation: true` / `Sunset: <date>` 응답 헤더(RFC 8594) 추가. 기능은 정상 동작 유지(에러 응답 아님).

### Phase 2 — 사용량 모니터링 (Phase 1 배포 후)
- 구 경로 Prometheus 카운터를 관측. 종료 조건: **주간 요청 수가 2주 연속 N건 미만**(N은 배포 후 실측 베이스라인 대비 정함, 예: 초기 트래픽의 5% 미만)일 때 Phase 3 착수 가능.
- 담당: 다음 정기 인프라 리뷰에서 재확인 (구체적 날짜는 Phase 1 배포일 기준 +6주 후 1차 점검).

### Phase 3 — 구 경로 제거 (별도 결정 시점, 이번 스코프 아님)
- Phase 2 종료 조건 충족 시 별도 티켓/PR로 구 경로 제거. 이 문서는 Phase 1까지만 다룬다.
- Phase 2에서 종료 조건이 충족되지 않고 장기간(예: 6개월 이상) 무의미한 트래픽이 지속되면, "영구 이중 경로 유지"를 공식 결정으로 전환할지 이 시점에 재논의한다.

## AC (이번 구현 스코프: Phase 1)

- [ ] OpenAPI 스펙: 신규 하이픈 경로 3종(+서브액션)만 선언, 구 언더스코어 경로는 스펙에서 제거. 스키마 description 내 구 경로 언급도 갱신
- [ ] `bin-api-manager/cmd/api-manager/main.go`에 구 경로 **15개**(widgets 6 + sessions 5 + messages 4, `direct_hash_regenerate`/`end` 서브액션 포함)를 **`v1` 그룹**(RateLimit+Authenticate 상속)에 `ServerInterfaceWrapper` 경유로 등록 — 수동 파라미터 바인딩 구현 금지, 생성된 wrapper 재사용. 등록 개수를 테스트/코드 리뷰에서 15개로 명시 검증(누락 방지)
- [ ] `webchatDeprecationMarker` 미들웨어는 `bin-api-manager/lib/middleware/`(기존 `RateLimit`/`Authenticate`가 있는 패키지)에 위치
- [ ] 회귀 테스트: 구 경로에 토큰 없이 요청 시 401 (인증 미들웨어 상속 확인), rate limit도 신규 경로와 동일하게 적용됨을 확인
- [ ] `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m` (bin-openapi-manager, bin-api-manager 순서) 전부 통과
- [ ] 구 경로 요청에 Prometheus 카운터 부착. **경로 라벨은 반드시 `c.FullPath()`(라우트 템플릿, 예: `/webchat_widgets/:id`) 사용 — `c.Request.URL.Path`는 UUID가 그대로 라벨에 들어가 카디널리티 폭발을 일으키므로 금지.** + `Deprecation: true` / `Sunset: <RFC 8594 HTTP-date>` 응답 헤더 부착 (Sunset 값은 배포 설정의 상수/플래그로 관리, 코드에 명시). 기능은 정상 동작(에러 아님) 확인하는 테스트
- [ ] 신규 경로도 동일 기능 정상 동작 확인 (회귀 테스트)
- [ ] square-admin 관리 콘솔 API 호출부 전체를 신규 경로로 전환
- [ ] square-admin 위젯 런타임(`client.js`) 신규 경로로 전환, 재빌드
- [ ] `provider.js`의 `LoadResource(resource, cacheKey = resource)` 형태로 캐시 키를 API 리소스 문자열과 분리(기본값은 기존 동작과 동일하게 `resource` 재사용, webchat 호출부만 `cacheKey: 'webchat_widgets'` 명시 전달). `resourceCache.js:25`는 변경 없음. 캐시 키가 API 경로와 다른 이유를 `provider.js`/`resourceCache.js` 양쪽에 인라인 주석으로 남겨 향후 "일치시키는" 리팩터로 되돌아가지 않게 함. 이미 로그인된 세션에서도 위젯 이름이 UUID로 폴백되지 않음을 확인하는 테스트 추가
- [ ] nginx에 번들 직접 URL 전용 exact-match 캐시 블록 추가(5분), 기존 `*.js` 전역 1년 불변 규칙보다 우선 적용됨을 확인
- [ ] RST 문서 갱신 + 클린 Sphinx 리빌드 + `git add -f docsdev/build/`
- [ ] 관련 프론트엔드 테스트(`__tests__/*.test.js`) 경로 리터럴 갱신, **main 베이스라인 대비 신규 실패 0건** + `npm run build` 성공 (모노레포 CLAUDE.md의 baseline-비교 절차 준수)
- [ ] PR 머지 시점에 Phase 2 후속 Jira 티켓을 생성 — 담당자 지정, 1차 점검일(배포일 +6주)을 이슈에 명시

## 스코프 아님 (재확인)

- admin SPA 라우트 경로, `src/views/webchat_widgets/` 디렉터리명
- `bin-webchat-manager` 내부 RPC 경로 (이미 하이픈/무관 컨벤션)
- DB 스키마, 테이블명
- Phase 3 (구 경로 완전 제거)
