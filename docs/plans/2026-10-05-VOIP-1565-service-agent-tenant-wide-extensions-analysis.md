# 이슈 분석: VOIP-1565 / SQUARE-89 (Agent tenant-wide extension 조회 + 타인 비밀번호 마스킹)

작성일 2026-10-05. 기준 코드: monorepo origin/main 440b71f9d, monorepo-javascript origin/main a604bb12.
워크트리: `~/gitvoipbin/monorepo/.worktrees/VOIP-1565-Expose-tenant-wide-extensions-with-password-masking`, `~/gitvoipbin/monorepo-javascript/.worktrees/SQUARE-89-Load-own-extensions-via-me-addresses`.

## 1. 요구사항 (대표님 확정)
- Agent(상담사)가 같은 고객사의 다른 상담사 extension 목록/단건을 조회할 수 있어야 한다.
- 타인 extension 은 `password` 를 숨긴다. 본인 소유 extension 만 비밀번호 평문.
- `GET /service_agents/extensions` 를 고객사 전체 조회로 전환 (scope 선택값 방식, 별도 /me/extensions 경로 모두 불채택).
- 서비스 단절(하위 호환)은 고려하지 않는다.
- `direct_hash` 타인 마스킹: 분석 단계 권고 (대표님 이의 없음, 설계에서 확정).

## 2. 이슈 유효성 (현재 코드 재확인)

### 2.1 monorepo (VOIP-1565)
| 사실 | 근거 코드 |
|---|---|
| Agent 목록은 본인 `a.Agent.Addresses` 의 `type=extension` 만 순회하여 `extensionGet(id)` 반복 후 `ConvertWebhookMessage()` 반환 | `bin-api-manager/pkg/servicehandler/serviceagent_extension.go` `ServiceAgentExtensionList` |
| Agent 단건은 요청 id 가 본인 Addresses 에 있을 때만 반환, 아니면 `ErrNotFound` | 같은 파일 `ServiceAgentExtensionGet` |
| 서버 핸들러는 OpenAPI 에 `PageSize`/`PageToken` 이 정의되어 있음에도 `ServiceAgentExtensionList(ctx, a)` 로 호출하며 params 미사용 | `bin-api-manager/server/service_agents_extensions.go`, `bin-openapi-manager/openapi/paths/service_agents/extensions.yaml` |
| `WebhookMessage` 에 `Password`(평문), `DirectHash` 포함, `ConvertWebhookMessage()` 가 그대로 복사 | `bin-registrar-manager/models/extension/webhook.go` |
| 상위 admin 경로 `/extensions` 는 `PermissionCustomerAdmin\|PermissionCustomerManager` 로 보호, customer_id 필터 + `deleted=false` + 페이지네이션 | `bin-api-manager/pkg/servicehandler/extension.go` `ExtensionList`, `ExtensionGet` |
| 이미 존재하는 tenant-wide Agent 경로 선례: `ServiceAgentAgentList` 가 `PermissionAll` 게이트, customer_id/deleted 필터, size/token | `bin-api-manager/pkg/servicehandler/serviceagent_agent.go`, `server/service_agents_agents.go` (page size 기본 100, 최대 100) |
| `extensionGet` 내부 헬퍼는 권한 검사가 없는 RPC 래퍼 | `bin-api-manager/pkg/servicehandler/extension.go:40` |
| extension `direct_hash` 는 direct-manager 가 발급(`DirectV1DirectCreate(..., ResourceTypeExtension)`)하는 SIP 직접 호출 주소(`sip:<hash>@sip.voipbin.net`) 식별자. api-manager auth 경로에서 `ResourceTypeExtension` 으로 JWT 를 발급하는 코드는 발견되지 않음 | `bin-registrar-manager/pkg/extensionhandler/extension.go:110`, `bin-api-manager/pkg/servicehandler/auth*.go` grep |
| 문서 반영 대상: `bin-api-manager/docsdev/source/service_agent_overview.rst`, `restful_api_errors.rst`, `docs/routing.md`, OpenAPI `paths/service_agents/extensions*.yaml` | grep |

판정: 이슈 유효. 현재 Agent 는 본인 extension 만 조회 가능(요구와 불일치). 비밀번호 마스킹 대상(타인 extension)은 현재 조회 자체가 불가하므로, 본 변경이 노출 범위를 넓히면서 동시에 마스킹을 도입하는 구조이다.

### 2.2 square-talk (SQUARE-89)
이전 대화(및 티켓 초안)에서 "square-talk 은 목록을 내 extension 으로 쓰고 softphone 등록이 목록에 의존한다" 고 가정했으나, 코드 재확인 결과 부분적으로 틀렸다. 정정:

| 사실 | 근거 코드 |
|---|---|
| softphone 등록용 `webrtcExtension` 은 목록이 아니라 `InitMe()` 가 `/me` 의 addresses 에서 첫 `type=extension` 의 `target` 을 얻어 `FetchExtension(id)`(단건 GET)로 설정한다 | `square-talk/src/store/slices/meSlice.js:87-107`, `extensionsSlice.js` `FetchExtension.fulfilled` |
| 목록 `FetchExtensions` 는 DashboardLayout, CallsPage, ProfilePage 에서 dispatch 되어 `state.extensions.extensions` 에 저장되고, 사용처는 통화 상대 식별(`getExtensionCallerInfo`, `resolveCallerDisplay`, `identifyExtensionParty`, CallerContextPanel)과 ProfilePage 의 extension 주소 보강뿐이다 | `layouts/DashboardLayout.jsx`, `features/calls/CallsPage.jsx`, `components/call/CallerContextPanel.jsx`, `features/profile/ProfilePage.jsx`, `utils/callHelpers.js` |
| `callHelpers.js` 주석이 이 목록을 "legacy self-scoped extensions match, kept as a fallback" 으로 명시하고, 타인 상담사 이름은 customer-wide agents 목록으로 우회 해결하고 있다 | `utils/callHelpers.js` `getAgentOwnerByExtensionId`, `resolveCallerDisplay` |
| `FetchExtensions.fulfilled` 에 `ext.type === 'webrtc'` 자동 선택 로직이 있으나 extension `WebhookMessage` 에는 `type` 필드가 없어 사실상 도달 불가(dead). 서버가 tenant-wide 로 바뀐 후 `type` 이 생기는 일이 있으면 타인 extension 이 `webrtcExtension` 으로 설정될 위험이 잠복 | `extensionsSlice.js` |
| `getExtensions()` 는 page_size 를 보내지 않는다(서버 기본값 사용). tenant-wide 전환 시 서버가 기본 100 으로 자르면 잘림 가능성. agents.js 는 `page_size: 100` 명시 | `api/services/extensions.js`, `api/services/agents.js` |

판정: SQUARE-89 의 원 가정(softphone 이 목록에 의존)은 틀렸다. 서버가 tenant-wide 로 바뀌어도 softphone 등록은 영향이 없다 **[7.1 로 정정: 소유 판정이 /me 와 불일치하면 마스킹된 본인 extension 이 webrtcExtension 에 들어가는 회귀 경로가 있으며 7.2/7.3 으로 해소]**. 그러나 다음 이유로 square-talk 변경은 여전히 타당하다.
1. tenant-wide 목록은 오히려 통화 상대 식별(이름/번호)을 개선한다(현재는 agents 목록 + SIP display name 우회).
2. dead `type==='webrtc'` 자동 선택은 서버 의미 변화 후 잠재 위험이므로 제거한다.
3. page_size 명시 및(필요 시) 다음 페이지 로딩 정책 결정이 필요하다.
4. 주석(callHelpers)이 사실과 달라지므로 갱신해야 한다.
5. 의미(Redux 상태 이름 `extensions`)가 "내 것" 에서 "고객사 전체" 로 바뀌므로, 소비처가 "내 것" 가정을 하는지 전수 확인이 필요하다(ProfilePage 가 목록 존재 여부로 재조회 조건을 쓰는 점 포함).

## 3. 진행 타당성
- 우선순위/리스크: 신규 정보 노출 확대이므로 마스킹 누락이 곧 보안 사고. 핵심 위험은 (a) 교차 고객사 노출, (b) 마스킹 필드 누락(password, direct_hash), (c) 공유 모델 수정으로 admin/webhook 영향.
- 의존성: monorepo 변경과 square-talk 변경은 저장소가 다르므로 PR 2개(저장소당 1). 대표님이 서비스 단절 비고려를 확정했으므로 배포 순서 제약 없음. square-talk 은 서버 변경에 기능적으로 의존하지 않는다(위 2.2) **[7.1 로 정정: 소유 판정 재조회(7.2)와 클라이언트 가드(7.3)를 전제로 한다]**.
- 대안: (a) scope=all 선택값(불채택, 대표님 결정), (b) `/me/extensions` 신설(불채택), (c) 서버 변경 없이 agents 목록만 활용(요구 미충족). 진행이 타당하다.
- 오버엔지니어링 점검: 새 필드(`is_owner`) 미도입(공유 WebhookMessage 정합성 비용), 전역 플래그 미도입, 마스킹은 servicehandler 한 곳. 클라이언트의 "내 것" 판별은 `/me` addresses 로 가능.

## 4. 열린 질문 (설계 단계에서 확정)
1. 타인 extension 마스킹 시 JSON 상 `password`/`direct_hash` 를 `""` 로 둘지 키를 제거할지. (권고: 빈 문자열, WebhookMessage 에 omitempty 가 없어 구조 변경 없이 복사본에서 치환 가능)
2. 목록 정렬/토큰: 기존 `ExtensionList`(tm_create 내림차순 추정) 동작을 그대로 따른다. nextToken 은 마지막 항목 TMCreate.
3. `PermissionAll` 게이트(Agent 선례)와 `a.IsAgent()` 유지 여부. 권고: 유지.
4. 목록에서 개별 항목 customer_id 불일치 방어(서버 필터가 있어도 per-item 확인) 여부. 권고: Get 은 필수, List 는 필터 + 방어적 재검사.
5. square-talk 은 목록 1페이지(page_size=100)만 로딩할지, 다음 페이지를 순회할지. (권고: agents 와 동일하게 100 고정, 초과 시 수용된 한계로 명시, 설계에서 확정)

## 5. 리뷰 1회차 반영 (R1-A: APPROVED, R1-B: CHANGES_REQUESTED, 모든 지적 코드 재검증 완료)

### 5.1 인증 주체와 권한 (R1-B 보충, 코드 확인)
- `ServiceAgentExtensionList/Get` 는 `a.IsAgent()` 만 통과시킨다. direct, accesskey, delegate 신원은 `ErrAuthenticationRequired`. 신규 코드도 동일 가드를 유지한다.
- `hasPermission`(`servicehandler/etc.go:13`)은 `ProjectSuperAdmin` 이면 customerID 와 무관하게 true 를 반환한다. 따라서 신규 Get/List 는 `hasPermission` 외에 `ext.CustomerID == a.CustomerID` 를 **명시적으로 강제**한다(superadmin 이어도 타 고객사 extension 은 NotFound). 근거: 이 엔드포인트는 Agent 용 tenant 경계 API 이며 superadmin 의 cross-tenant 조회는 admin 경로(`/extensions`)의 책임이다.
- `Agent.HasPermission(PermissionAll)` 은 항상 true 이므로 `PermissionAll` 게이트는 사실상 "같은 고객사" 검사만 한다(`ServiceAgentAgentList` 선례와 동일). 유지한다.

### 5.2 오류 의미 (R1-B #4)
- 타 고객사 extension id 또는 존재하지 않는 id: `ErrNotFound`(존재 여부 비노출).
- 같은 고객사의 타인 extension: 200 + 마스킹.
- 삭제된 extension(`tm_delete` 비어있지 않음) Get 처리는 기존 `extensionGet` 동작을 따르며 설계에서 확정한다.

### 5.3 노출 필드 표 (R1-B #2)
| 필드 | 본인 소유 | 타인 소유 |
|---|---|---|
| id, customer_id, name, detail, extension, domain_name, username, tm_* | 노출 | 노출 |
| password | 평문 | 빈 문자열 |
| direct_hash | 노출 | 빈 문자열 |

`username` 은 extension 번호와 동일(`extension.go` 주석)하고 `domain_name` 은 고객사 공개 realm 이므로 노출을 유지한다. `WebhookMessage` 에는 `DirectID` 가 없다(코드 확인). `direct_hash` 마스킹 근거: `sip:<direct_hash>@sip.voipbin.net` 로 외부에서 해당 상담사 extension 으로 직접 발신 가능한 공개 링크형 식별자이며 재발급(`ExtensionDirectHashRegenerate`)이 폐기 수단이다. 타인 hash 를 알면 스팸/사칭 발신 위험이 있다. `/auth/boot` 의 `directResourceMapping` 에는 `ResourceTypeExtension` 이 없어 JWT 발급 경로는 아니다(R1-A 확인).

### 5.4 소유 판정 기준과 기존 위험 (R1-B #1, #6, 코드 확인)
- 소유 판정: 요청 Agent 의 `a.Agent.Addresses` 중 `type=extension` 의 `target`(extension UUID)과 id 가 일치하면 본인 소유. 기존 own-scope 구현과 같은 기준이며 새 소유 개념을 만들지 않는다.
- **[7.2 로 대체됨, 아래 서술은 이력 보존용이며 구현 기준 아님]** 출처: `a.Agent` 는 JWT 클레임(`lib/middleware/authenticate.go:272-290`)에서 복원되므로 토큰 발급 시점의 Addresses 이다. 즉 (a) 방금 `PUT /me/addresses` 로 등록한 extension 은 재로그인 전까지 본인 소유로 보이지 않고(마스킹), (b) 관리자가 할당을 해제한 직후에도 기존 토큰이 만료되기 전에는 본인 소유로 보인다(평문). (b) 는 기존 own-scope 구현에도 동일하게 존재하는 창이며 신규 위험이 아니다. 결정: 기존 컨벤션과 일관성을 위해 JWT 값을 그대로 사용하고(재조회 RPC 미도입), 이 창은 수용된 한계로 설계 문서에 명시한다. 재조회 도입은 실측 신호가 생길 때 별도 이슈로 다룬다.
- **기존 위험(본 변경 범위 밖, 대표님 보고 대상)**: `PUT /service_agents/me/addresses` → `agenthandler.UpdateAddresses`(`bin-agent-manager/pkg/agenthandler/agent.go:371-445`)는 extension 이 같은 customer 이고 다른 agent 에 할당되지 않았다면 일반 Agent 의 자기 등록을 허용한다(권한 검사 없음). 따라서 미할당 extension 은 어느 Agent 든 스스로 등록해 평문 password 를 볼 수 있다. 이는 현재도 가능한 동작이며 본 변경으로 새로 생기지 않는다. 다만 본 변경 후 타인/미할당 extension 의 id 가 목록에 드러나 발견 비용이 낮아진다. 본 PR 에서는 변경하지 않고 별도 후속 이슈로 대표님께 보고한다(범위 확장 금지 원칙, 의도된 셀프 등록 UX 일 수 있어 정책 결정이 필요).
- 클라이언트는 `password` 가 빈 값인지로 소유 여부를 추정하지 말고 `/me` addresses 로 판단한다(square-talk 은 이미 그렇게 동작).

### 5.5 이벤트 경로 (R1-B 확인)
웹소켓 `validateTopics` 는 customer 단위/4-part 리소스 topic 모두 `CustomerAdmin|CustomerManager` 를 요구하므로 일반 Agent 는 extension 이벤트(평문 password 포함)를 구독할 수 없다. 본 변경은 이 경로를 넓히지 않는다.

### 5.6 소비자 조사 보강 (R1-B #5, R1-A #1)
- 이 엔드포인트의 소비자: square-talk(`src/api/services/extensions.js`)와 `tmp/android` 설계 문서(단건 GET 으로 본인 SIP 자격증명 수신, 본인 평문 필요, 영향 없음). square-admin 은 미사용(검색 결과 square-talk 사본만). 외부 SDK/문서 외 소비자는 발견되지 않음.
- 문서 갱신 대상 추가: `bin-api-manager/docsdev/source/extension_overview.rst`, `extension_struct_extension.rst`, `extension_tutorial.rst`(응답 필드/Agent 조회 설명), `service_agent_overview.rst`, `routing.md`, OpenAPI `paths/service_agents/extensions*.yaml`. `docsdev/build` 산출물 재생성 여부는 설계에서 저장소 규칙을 확인해 확정.

### 5.7 동작 변화 명시 (R1-A #2)
- 현재 목록은 항목별 `extensionGet` 실패를 `continue` 로 건너뛰나, 신규 목록은 단일 RPC(`RegistrarV1ExtensionList`) 이므로 실패는 전체 오류가 된다. 대신 부분 누락 없이 일관되며 `ServiceAgentAgentList` 와 동일하다.
- 신규 목록의 페이지네이션은 `ExtensionList`(admin) 와 동일하게 `token == ""` 이면 현재 시각, nextToken 은 마지막 항목 `TMCreate`.

### 5.8 테스트 요건 (R1-A #3)
목록/단건 × 본인/타인 × (password, direct_hash 마스킹) × 소유 판정은 `agentGet` 재조회 결과 기준(JWT Addresses 와 DB 값이 다른 경우 DB 를 따름) × `agentGet` 실패 시 오류 반환(fail closed) × 삭제된 agent(빈 Addresses)는 전부 마스킹 × List 에서 `agentGet` 호출이 요청당 1회 × 교차 고객사 Get NotFound × superadmin 교차 고객사 NotFound × 비-Agent 신원 거부 × 페이지네이션 전달.

## 6. 리뷰 2회차 반영 (R2-A: APPROVED, R2-B: APPROVED, 비차단 권고 코드 재검증 후 반영)

- 6.1 (5.2 확정) 삭제된 extension: `extensionGet` → `RegistrarV1ExtensionGet` → `extensionHandler.Get` 은 `deleted` 필터를 적용하지 않는다(코드 확인: `extensionhandler/extension.go:172`). 따라서 신규 Get 은 `TMDelete != nil` 이면 `ErrNotFound` 로 확정한다(삭제된 extension 의 평문 password 반환 방지). 목록은 `deleted=false` 필터를 서버가 건다.
- 6.2 마스킹은 `ConvertWebhookMessage()` 가 반환한 새 `WebhookMessage` 복사본에서만 수행한다. 원본 `rmextension.Extension` 과 공유 모델은 변경하지 않는다.
- 6.3 `extensionGet`(`servicehandler/extension.go:40`)은 `log.WithField("tag", res).Debug(...)` 로 password 를 포함한 객체를 Debug 로그에 남긴다(기존 동작, Debug 레벨). 신규 경로도 이 헬퍼를 호출하므로 Debug 로그 범위가 같은 고객사 타인 extension 으로 넓어진다. 결정: 본 PR 은 이 기존 로그를 변경하지 않는다(범위 밖, admin 경로 동작 변경 회피)하고, 후속 보고 항목에 포함한다. 신규 코드는 password 를 로그에 출력하지 않는다.
- 6.4 목록 마스킹 판정은 항목별로 수행하며 테스트는 "목록 본인/타인 혼합 응답에서 타인 항목만 마스킹" 과 "목록 항목 customer_id 불일치 방어" 를 포함한다.
- 6.5 수정된 라인 참조: `authenticate.go` agent 분기는 271행부터(문서 5.4 의 272-290 은 근사치).

## 7. 리뷰 3회차 반영 (R3-A: CHANGES_REQUESTED, R3-B: APPROVED. R3-A 지적 코드 재검증 완료: 사실)

- 7.1 회귀 확인: `ServiceAgentMeGet` 은 `agentGet(a.AgentID())` 로 DB 최신 addresses 를 반환(`serviceagent_me.go:25`). 반면 5.4 는 JWT 시점 `a.Agent.Addresses` 로 소유 판정. 로그인 후 할당/자기등록된 extension 은 `InitMe`(최신 /me)가 id 를 얻고 단건 GET 이 비소유로 판정하여 password "" 200 반환 → `FetchExtension.fulfilled` 가 `webrtcExtension` 에 마스킹 객체 설정 → StatusBar `isRegistered=true`, CallsPage `showDialpad=true` 이나 `lib/webphone.js` 가 빈 password 로 등록 거부. 현재 구현은 같은 상황에서 404 rejected 였으므로 회귀다. 따라서 2.2절 "softphone 영향 없음"과 3절 "기능적 비의존"은 이 경로에서 정정한다.
- 7.2 **5.4 결정 변경(실측 근거 확보)**: 소유 판정은 JWT 값이 아니라 호출 시점 `h.agentGet(ctx, a.AgentID())` 의 Addresses 를 사용한다(`/me` 와 동일 소스). Get 은 RPC 1회, List 는 요청당 1회 추가. 이로써 `/me` 와 단건/목록의 소유 판정이 일치하고, 해제 직후 평문이 남는 창(5.4 b)도 사라진다. 재조회 실패 시 안전 방향으로 전체 마스킹이 아니라 오류 반환(fail closed)한다. 호출 순서: `IsAgent()` → (Get) `extensionGet` → `ext.CustomerID == a.CustomerID` 및 `TMDelete` 검사 → `hasPermission(PermissionAll)` → `agentGet(a.AgentID())` 순이며, 타 고객사 id 는 `agentGet` 호출 전에 NotFound 로 종료한다. List 는 `hasPermission` 후 `agentGet` 1회 + `RegistrarV1ExtensionList` 1회.
- 7.2-b 비차단 반영: `agentGet` 은 목록/단건 모두 요청당 1회만 호출한다. 서버 단위 테스트에 "JWT Addresses 와 agentGet 결과가 다르면 agentGet 을 따른다" 케이스를 필수로 포함한다(3회차 회귀 고정). 삭제된 agent 는 Addresses 가 하드 삭제되어 전부 마스킹(안전 방향)이며 존재하지 않는 id 는 fail closed.
- 7.3 SQUARE-89 방어 추가: `FetchExtension.fulfilled` 는 `password` 가 비어 있으면 `webrtcExtension` 을 설정하지 않고 error 처리(softphone 자격증명 유효성 검사이며 소유 추정이 아님). 단위 테스트로 고정(password 가 `""` 인 경우와 필드 누락(undefined) 인 경우 모두).
- 7.4 ProfilePage 재조회 조건(`!extensions || length===0`, deps `[extensions]`): `fulfilled` 가 새 배열을 저장하므로 목록이 `[]` 일 때 재dispatch 반복 가능성이 있다. 기존 위험이며 전환 후 고객사에 extension 이 0개일 때로 좁아진다. 설계에서 실제 동작을 확인하고 필요 시 1회 로딩 가드를 추가한다.
- 7.5 `meSlice.js` 라인 참조 정정: 약 79-108행.
- 7.6 변경 파일 목록(R3-B 메모): `servicehandler/main.go` 인터페이스(`ServiceAgentExtensionList(ctx, a, size, token)`), mock 재생성, `serviceagent_extension.go`, `server/service_agents_extensions.go`(pagesize 100 클램프), 두 테스트 파일, OpenAPI description + 생성물, RST/routing 문서.

## 8. 리뷰 5회차 반영 (R5-A: APPROVED, R5-B: APPROVED, 비차단 메모 반영)
- 8.1 5.8 테스트 목록에 추가: 삭제된 extension(`TMDelete != nil`) Get 은 `ErrNotFound`(6.1 확정).
- 8.2 square-talk: `getExtensionCallerInfo` 는 `ext.id === rawId || ext.extension === rawId` 로 매칭한다. 목록이 고객사 전체가 되면 발신번호가 우연히 내선번호와 같은 외부 착신이 내부 extension 으로 표시될 수 있다(보안 영향 없음, 표시 오류). 설계 단계에서 `rawId` 가 UUID 형식이면 id 매칭만, 아니면(발신 다이얼 번호) 번호 매칭을 쓰도록 구분할지 결정한다. 기존 own-scope 에서도 동일한 매칭이었으나 대상 집합이 커지므로 위험이 커진다.

## 9. 디자인 리뷰 1회차 반영 메모
- 9.1 5.2 / 6.1 의 `ErrNotFound` 는 `serviceerrors.ErrNotFound`(→ reason `RESOURCE_NOT_FOUND`)가 아니라, registrar 의 typed 오류와 동일한 `EXTENSION_NOT_FOUND` 로 확정한다(reason 차이가 타 고객사 id 존재를 노출하므로). 상세는 설계 문서 D7.
