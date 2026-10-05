# VOIP-1565 Service Agent: 고객사 전체 extension 조회와 타인 extension 비밀번호 마스킹 (설계 + 구현 계획)

- 상태: Draft (Design Review 대기)
- 작성일: 2026-10-05
- 티켓: VOIP-1565 (본 저장소), 동반 티켓 SQUARE-89 (monorepo-javascript, 별도 PR)
- 기준 코드: origin/main 440b71f9d
- 이슈 분석(리뷰 루프 6회, 5-6회차 연속 승인 종료): `2026-10-05-VOIP-1565-service-agent-tenant-wide-extensions-analysis.md` (본 디렉터리). 본 문서의 절 번호 참조는 분석 문서의 절을 가리킨다.

## 1. 문제

`GET /service_agents/extensions`, `GET /service_agents/extensions/{id}` 는 로그인 Agent 본인 `Addresses` 의 extension 만 반환한다(분석 2.1). 상담사는 같은 고객사 다른 상담사의 extension(번호, 이름)을 볼 수 없다. 대표님 요구는 고객사 전체 extension 을 조회하되 SIP 비밀번호는 본인 소유만 볼 수 있게 하는 것이다.

## 2. 목표

1. `GET /service_agents/extensions` 가 같은 고객사(`customer_id == a.CustomerID`)의 삭제되지 않은 extension 전체를 `page_size`/`page_token` 으로 페이지네이션하여 반환한다.
2. `GET /service_agents/extensions/{id}` 가 같은 고객사의 삭제되지 않은 모든 extension 을 반환한다.
3. 본인 소유가 아닌 extension 은 `password`, `direct_hash` 를 빈 문자열로 마스킹한다. 본인 소유는 평문.
4. 소유 판정은 호출 시점 `agentGet(a.AgentID())` 의 Addresses 로 한다(`/me` 와 동일 소스, 분석 7.2).
5. 마스킹은 servicehandler 에서 복사본에 수행하며 공유 모델 `ConvertWebhookMessage()`, admin 경로, webhook payload 는 변경하지 않는다.

## 3. Non-goals

- 별도 `scope` 파라미터, `/me/extensions` 경로, `is_owner` 필드 신설 (대표님 결정 및 분석 3절 오버엔지니어링 점검).
- 하위 호환 유지 (대표님: 서비스 단절 비고려).
- `PUT /service_agents/me/addresses` 자기 등록 정책 변경 (분석 5.4, 후속 이슈로 대표님께 별도 보고).
- `extensionGet` 의 기존 Debug 로그 변경 (분석 6.3, 후속 보고).
- admin `/extensions` 경로 변경.

## 4. 결정 사항 (확정)

| # | 결정 | 근거 |
|---|---|---|
| D1 | 고객사 전체 조회로 전환, 별도 모드 없음 | 대표님 확정 |
| D2 | 마스킹 필드는 `password`, `direct_hash`. `username`, `domain_name`, `extension`, `name`, `detail` 은 노출 | 분석 5.3 |
| D3 | 마스킹 값은 빈 문자열(키 유지). `WebhookMessage` 에 omitempty 없음, 구조 변경 없음 | 분석 4절 1번 |
| D4 | 소유 판정 = `agentGet` 재조회 Addresses 의 `type=extension` target 일치. 재조회 실패 시 오류(fail closed) | 분석 7.2 |
| D5 | 신원 가드 `a.IsAgent()` 유지(direct/accesskey/delegate 거부), `hasPermission(PermissionAll)` 유지 | 분석 5.1 |
| D6 | superadmin 이어도 `ext.CustomerID != a.CustomerID` 면 NotFound | 분석 5.1 |
| D7 | 타 고객사 id, 존재하지 않는 id, 삭제된 extension(`TMDelete != nil`)은 **바이트 단위로 동일한 404 응답**(status NOT_FOUND, domain `registrar-manager`, reason `EXTENSION_NOT_FOUND`, message `The extension was not found.`)을 낸다. 존재하지 않는 id 는 registrar 가 이미 이 typed 오류를 반환하므로(`extensionhandler/extension.go:172`) 그대로 통과시키고, 타 고객사/삭제 경우는 서비스핸들러가 같은 값으로 `cerrors.NotFound(commonoutline.ServiceNameRegistrarManager, "EXTENSION_NOT_FOUND", "The extension was not found.")` 를 생성한다. `serviceerrors.ErrNotFound`(→ `RESOURCE_NOT_FOUND`)는 사용하지 않는다(reason 차이로 타 고객사 id 존재가 노출되므로, 리뷰 D1-1) | 분석 5.2, 6.1, 디자인 리뷰 1회차 |
| D8 | 목록은 `deleted=false`, `customer_id` 필터를 서버 호출에 포함하고 항목별 `CustomerID` 방어 재검사(불일치 항목 제외 + 에러 로그) | 분석 6.4 |
| D9 | 서버 핸들러 page size 기본/상한 100 (`ServiceAgentAgentList` 와 동일) | 분석 2.1 |

## 5. 설계

### 5.1 servicehandler (`bin-api-manager/pkg/servicehandler/serviceagent_extension.go`)

시그니처 변경:

```go
ServiceAgentExtensionList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*rmextension.WebhookMessage, error)
ServiceAgentExtensionGet(ctx context.Context, a *auth.AuthIdentity, extensionID uuid.UUID) (*rmextension.WebhookMessage, error)   // 시그니처 동일, 동작 변경
```

`newExtensionNotFound()` 는 `cerrors.NotFound(commonoutline.ServiceNameRegistrarManager, "EXTENSION_NOT_FOUND", "The extension was not found.")` 를 반환하는 같은 파일의 비공개 함수이다. 같은 reason 문자열이 registrar 에도 있으므로 값 일치를 테스트로 고정한다(7절).

`ServiceAgentExtensionList` 순서:
1. `!a.IsAgent()` → `ErrAuthenticationRequired`.
2. `token == ""` → `h.utilHandler.TimeGetCurTime()`.
3. `!h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionAll)` → `ErrPermissionDenied`.
4. `owned, err := h.serviceAgentOwnedExtensionIDs(ctx, a)` (내부에서 `h.agentGet(ctx, a.AgentID())`, 실패 시 error 반환).
5. filters `{"customer_id": a.CustomerID.String(), "deleted": "false"}` → `h.convertExtensionFilters` → `h.reqHandler.RegistrarV1ExtensionList(ctx, token, size, typedFilters)`. 실패는 admin `ExtensionList` 와 동일하게 `fmt.Errorf("%w: could not find extensions info", err)` 로 감싼다(typed 오류는 통과).
6. 항목별: `ext.CustomerID != a.CustomerID` 이면 에러 로그 후 제외. 아니면 `maskExtensionForAgent(ext.ConvertWebhookMessage(), owned[ext.ID])` 결과를 append.

`ServiceAgentExtensionGet` 순서:
1. `!a.IsAgent()` → `ErrAuthenticationRequired`.
2. `tmp := h.extensionGet(ctx, extensionID)` (오류는 그대로 반환).
3. `tmp.CustomerID != a.CustomerID || tmp.TMDelete != nil` → D7 의 동일 typed NotFound(`newExtensionNotFound()` 비공개 헬퍼가 생성) 반환 (`agentGet` 호출 전 종료, D6/D7).
4. `!h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionAll)` → `ErrPermissionDenied`.
5. `owned, err := h.serviceAgentOwnedExtensionIDs(ctx, a)`; err 이면 반환(fail closed). 마스킹 후 반환.

헬퍼(같은 파일, 비공개):

```go
// serviceAgentOwnedExtensionIDs returns the set of extension ids assigned to the calling agent,
// resolved from the up-to-date agent record (same source as GET /service_agents/me).
func (h *serviceHandler) serviceAgentOwnedExtensionIDs(ctx context.Context, a *auth.AuthIdentity) (map[uuid.UUID]bool, error)

// maskExtensionForAgent blanks the credential-like fields of a copy when the extension is not owned.
func maskExtensionForAgent(ws *rmextension.WebhookMessage, owned bool) *rmextension.WebhookMessage
```

`maskExtensionForAgent` 는 `ws` 가 `ConvertWebhookMessage()` 가 만든 새 객체이므로 직접 수정해도 원본 모델/캐시에 영향이 없다(분석 6.2). `owned` 이면 그대로 반환. password 를 로그에 출력하지 않는다.

### 5.2 server (`bin-api-manager/server/service_agents_extensions.go`)

`GetServiceAgentsExtensions` 가 `params.PageSize`/`params.PageToken` 을 사용한다. page size 는 기본 100, `<=0 || >100` 이면 100 (`service_agents_agents.go` 패턴). nextToken 은 마지막 항목 `TMCreate` UTC (`2006-01-02T15:04:05.000000Z`), 기존 로직 유지. `ServiceAgentExtensionList(ctx, a, pageSize, pageToken)` 호출. `GetServiceAgentsExtensionsId` 변경 없음.

### 5.3 인터페이스, mock, OpenAPI, 생성물, 문서

| 파일 | 변경 |
|---|---|
| `bin-api-manager/pkg/servicehandler/main.go` | `ServiceAgentExtensionList` 시그니처 |
| `bin-api-manager/pkg/servicehandler/mock_main.go` | `go generate` 재생성 |
| `bin-api-manager/pkg/servicehandler/serviceagent_extension.go` | 5.1 |
| `bin-api-manager/server/service_agents_extensions.go` | 5.2 |
| `bin-openapi-manager/openapi/paths/service_agents/extensions.yaml`, `extensions_id.yaml` | description 갱신(고객사 전체, 타인 password/direct_hash 빈 문자열, 404 의미), `extensions.yaml` 에 `'403': $ref: '#/components/responses/PermissionDenied'` 추가(목록에 `hasPermission` 실패 경로가 생김), `extensions_id.yaml` 은 403/404 가 이미 있으므로 404 설명만 보강 |
| 생성물 (`bin-openapi-manager/gens/...`, `bin-api-manager/gens/...`, redoc) | 저장소 규칙대로 `go generate` |
| `bin-api-manager/docsdev/source/extension_overview.rst`, `extension_struct_extension.rst`, `extension_tutorial.rst`, `service_agent_overview.rst` | Agent 조회 범위와 마스킹 설명 갱신 |
| `bin-api-manager/docsdev/source/restful_api_errors.rst` (약 596-598행 `EXTENSION_NOT_FOUND`) | 설명을 "Extension ID does not exist, was deleted, or belongs to another customer" 로 갱신하고 발생 엔드포인트에 `GET /service_agents/extensions/{id}` 추가(현재 설명은 admin 경로 전용으로 서술되어 사실과 어긋남) |
| `bin-api-manager/docsdev/build/` | 루트와 `bin-api-manager` `CLAUDE.md` 규칙(CRITICAL): `cd bin-api-manager/docsdev && rm -rf build && python3 -m sphinx -M html source build` 후 `git add -f bin-api-manager/docsdev/build/`. 최근 docs 변경 커밋도 build 를 함께 커밋하며 835개 파일이 추적 중(리뷰 확인) |
| `bin-api-manager/docs/routing.md` | 설명 문구만 갱신(필요 시) |

### 5.4 동작 변화 (분석 5.7)

- 목록은 단일 RPC 이므로 실패가 전체 오류가 된다(기존: 항목별 `continue`).
- 요청당 `AgentV1AgentGet` RPC 1회 추가(목록/단건 공통).

## 6. 영향/리스크

| 리스크 | 대응 |
|---|---|
| 교차 고객사 노출 | RPC 필터 + 항목별 방어 재검사 + Get 명시 `CustomerID` 비교 + 테스트 |
| 마스킹 누락 | 신규 경로 하나(`maskExtensionForAgent`)만 거친다. 일반 Agent 가 쓸 수 있는 평문 경로는 이 파일 둘뿐(분석 6회차 B) |
| superadmin 교차 조회 | D6 명시 비교 + 테스트 |
| 요청당 RPC 증가 | `/me` 와 같은 agent-manager 캐시 경로. 목록도 요청당 1회(N+1 아님) |
| square-talk 회귀 | SQUARE-89 에서 방어(별도 PR). 서버 배포가 먼저여도 마스킹 객체가 softphone 에 들어가지 않도록 클라이언트 가드가 있음 |

## 7. 검증 계획

1. `bin-openapi-manager`: `GOFLAGS=-mod=mod go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`
2. `bin-api-manager`: 동일 순서(openapi 생성 후). 단, 생성물에 무관한 go.mod/go.sum 변동은 커밋하지 않는다(skill 의 vendor 잔재 항목).
3. `go test -count=1 ./pkg/servicehandler/... ./server/...` (전체 `go test ./...` 에서 비관련 패키지가 환경 요인으로 실패하면 단독 재실행으로 격리).
3-b. 문서: 위 sphinx 빌드가 경고/오류 없이 성공하는지 확인(기존 경고 대비 신규 경고 없음).
4. 테스트 케이스(분석 5.8, 6.1, 6.4, 7.2-b, 8.1):
   - List: 본인/타인 혼합에서 타인만 password, direct_hash 마스킹, 본인은 평문.
   - List: 항목 `CustomerID` 불일치 제외, page_size/token 전달, filters(customer_id, deleted=false).
   - Get: 본인 평문 / 같은 고객사 타인 마스킹 200 / 타 고객사 NotFound / superadmin 타 고객사 NotFound / 삭제된(`TMDelete`) NotFound.
   - **404 응답 동일성**: 타 고객사, 삭제된, 존재하지 않는(registrar 가 typed `EXTENSION_NOT_FOUND` 반환 mock) 세 경우의 `cerrors.VoipbinError`(Status, Domain, Reason, Message)가 모두 같음을 단일 테스트가 비교. 서버 핸들러 테스트에서는 세 경우의 HTTP 응답 본문이 동일한지도 확인.
   - 소유 판정은 `agentGet` 결과 기준(JWT `a.Agent.Addresses` 와 다를 때 `agentGet` 을 따름) — 3회차 회귀 고정.
   - `agentGet` 실패 → 오류(fail closed), 삭제된 agent(빈 Addresses) → 전부 마스킹.
   - 비-Agent 신원(direct/accesskey/delegate) → `ErrAuthenticationRequired`.
   - List 에서 `agentGet` 호출이 요청당 1회(gomock `Times(1)`).
   - 서버 핸들러: page_size 기본/상한 클램프, nextToken, params 전달. 기존 `Test_extensionsGET`(`server/service_agents_extensions_test.go`)의 mock 기대가 `ServiceAgentExtensionList(req.Context(), tt.agent)` 형태이므로 새 시그니처(size, token)로 수정한다.
5. 테스트 변이 확인: 마스킹 호출을 제거하면 실패하는지, 소유 판정 소스를 JWT 로 바꾸면 회귀 테스트가 실패하는지 확인한다.

## 8. 구현 계획 (Plan, 순서)

1. 브랜치/워크트리 준비 완료(`VOIP-1565-Expose-tenant-wide-extensions-with-password-masking`).
2. TDD: `serviceagent_extension_test.go` 를 먼저 새 요구 기준으로 수정/추가(실패 확인).
3. `serviceagent_extension.go` 구현(5.1), `main.go` 시그니처, mock 재생성.
4. `server/service_agents_extensions.go` + 서버 테스트.
5. OpenAPI description 수정 → 생성물 재생성.
6. 문서(RST, routing) 갱신.
7. 검증 계획 7절 전체 실행.
7-b. 문서 빌드 산출물 갱신(5.3, CLAUDE.md 규칙): sphinx 재빌드 후 `git add -f bin-api-manager/docsdev/build/`.
8. 커밋(제목=브랜치명, 본문 narrative + `bin-api-manager:`/`bin-openapi-manager:` 불릿, AI 표기 없음, author 확인 `pchero21@gmail.com`), 푸시, `origin/main` 충돌 확인, PR 생성.
9. PR 코드 리뷰 루프(최소 3회, 연속 2회 승인).
10. 후속 보고: `PUT /me/addresses` 미할당 extension 자기 등록(분석 5.4), `extensionGet` Debug 로그(분석 6.3).

## 9. 구현 시 유의 (디자인 리뷰 3회차 메모)
- 기존 `Test_ServiceAgentExtensionList` 는 `utilHandler` 목이 없고 응답 extension 의 `CustomerID` 가 zero 라 D8 방어에서 전부 제외된다. 신규 테스트는 `CustomerID` 를 agent 와 일치시키고 `pageToken` 을 명시하거나 `utilHandler.TimeGetCurTime` 기대를 추가한다. admin `Test_ExtensionList` 의 `RegistrarV1ExtensionList(ctx, token, size, filters)` 기대 작성 방식을 선례로 따른다.

## 10. 승인 상태

APPROVED (Design Review 3회, 2-3회차 연속 승인으로 종료. 1회차 CHANGES_REQUESTED 2건 반영).
