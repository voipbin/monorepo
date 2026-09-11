# VOIP-1514: Admin/Manager Case Owner Assign 엔드포인트 설계

- JIRA: https://voipbin.atlassian.net/browse/VOIP-1514
- 브랜치/워크트리: `VOIP-1514-admin-case-assign-endpoint`
  (`~/gitvoipbin/monorepo/.worktrees/VOIP-1514-admin-case-assign-endpoint`)
- 범위: 설계 문서 작성까지만. 구현 코드는 이 태스크에서 작성하지 않는다.

## 1. 배경 및 목적

square-admin(Admin/Manager 권한 프런트)의 Case Detail 화면에서 관리자가 담당
상담원(owner)을 직접 지정/변경할 수 있어야 한다. 현재 owner 배정 API는
`POST /service_agents/contact_cases/{id}/assign` (Agent 전용, `PermissionAll`
게이트) 하나뿐이며, top-level(Admin/Manager) 경로가 없다. Admin/Manager가
Case를 재배정하려면 대상 상담원 본인 세션으로 로그인해야 하는 상황이므로,
관리자 권한으로 직접 호출 가능한 `POST /contact_cases/{id}/assign`을 신설한다.

이미 CaseClose/CaseContinue/CaseUpdateContact 등 top-level Case 작업 함수들이
동일한 권한 체크 패턴(`PermissionCustomerAdmin|PermissionCustomerManager`)을
따르고 있으므로, 이번 신설 엔드포인트도 그 패턴을 그대로 재사용한다.

## 2. 재사용 대상 (기존 구현, 변경 없음)

| 계층 | 대상 | 비고 |
|---|---|---|
| bin-contact-manager | `pkg/casehandler/assign.go` `caseHandler.Assign(ctx, customerID, id, ownerType, ownerID)` | 이미 완성. 테넌트 체크(`c.CustomerID != customerID` → `dbhandler.ErrNotFound`)만 수행하고, 권한(Authorization) 판단은 하지 않음. 이번 기능도 이 규칙을 그대로 따른다. |
| bin-contact-manager | `pkg/dbhandler` `CaseUpdateOwner` | 이미 완성. |
| bin-common-handler | `pkg/requesthandler/contact_cases.go` `ContactV1CaseAssign(ctx, customerID, id, ownerID uuid.UUID) (*cmkase.Case, error)` | 이미 완성. RPC로 `/v1/cases/{id}/assign`을 호출하며, `OwnerType`을 wire상에서 `string(commonidentity.OwnerTypeAgent)`로 하드코딩한다(호출자는 owner_type을 지정할 수 없음). 신규 top-level `CaseAssign`도 이 동일한 requesthandler 메서드를 그대로 호출한다(신규 RPC 클라이언트 불필요). |
| bin-api-manager | `pkg/servicehandler/serviceagent_case.go:101` `ServiceAgentCaseAssign` | Agent 전용 버전. 로직(케이스 조회 → closed 체크 → owner agent 검증 → RPC 호출)을 그대로 참고해 권한 체크만 Admin/Manager로 교체한 top-level 버전을 만든다. |
| bin-openapi-manager | `openapi/paths/service_agents/contact_cases_id_assign.yaml` | 신규 yaml의 스키마 구조(request/response) 참고용. |
| bin-api-manager | `server/service_agents_contact_cases.go:124` `PostServiceAgentsContactCasesIdAssign` | 핸들러 구조(인증 → path id 파싱 → body 파싱 → owner_id 파싱 → servicehandler 호출 → 200 JSON) 참고용. |

## 3. 신설 대상

### 3.1 `bin-api-manager/pkg/servicehandler/case.go` — `CaseAssign`

기존 `CaseClose`/`CaseUpdateContact`와 동일한 함수 배치 규칙(top-level Case
함수는 `case.go`에 위치)에 따라 이 파일에 추가한다.

```go
// CaseAssign assigns the case of the given id to the given owner agent
// for an Admin/Manager caller (VOIP-1514). Mirrors
// ServiceAgentCaseAssign's logic (case lookup, closed-case guard, owner
// agent tenant validation, RPC call) but gates on
// PermissionCustomerAdmin|PermissionCustomerManager instead of
// PermissionAll, matching CaseClose/CaseUpdateContact's top-level
// permission pattern.
func (h *serviceHandler) CaseAssign(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID, ownerID uuid.UUID) (*cmkase.Case, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CaseAssign",
		"customer_id": a.CustomerID,
		"case_id":     id,
		"owner_id":    ownerID,
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	c, err := h.caseGet(ctx, a.CustomerID, id)
	if err != nil {
		log.Errorf("Could not get the case info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	if c.Status == cmkase.StatusClosed {
		log.Infof("Case is closed, status: %s", c.Status)
		return nil, serviceerrors.ErrCaseClosed
	}

	owner, err := h.reqHandler.AgentV1AgentGet(ctx, ownerID)
	if err != nil || owner.CustomerID != a.CustomerID {
		log.Infof("Could not validate the owner agent. err: %v", err)
		return nil, serviceerrors.ErrNotFound
	}

	res, err := h.reqHandler.ContactV1CaseAssign(ctx, a.CustomerID, id, ownerID)
	if err != nil {
		log.Errorf("Could not assign case. err: %v", err)
		return nil, err
	}

	return res, nil
}
```

주의: `CaseGet`은 `a.IsDirect()` 체크를 하지 않지만 `CaseClose`/`CaseContinue`/
`CaseUpdateContact`는 모두 함수 최상단에서 `a.IsDirect()`를 체크한다(direct
access 미지원). `CaseAssign`은 상태 변경 작업이므로 이 그룹(Close/Continue/
UpdateContact)의 패턴을 따라 `IsDirect()` 체크를 포함한다. 또한 이 그룹은
공통적으로 "case를 먼저 조회 → 조회 결과의 `CustomerID`로 권한 체크"
순서를 따르므로 `CaseAssign`도 동일 순서(caseGet 먼저, 그 다음
hasPermission)로 구현한다. `ServiceAgentCaseAssign`은 (자기 customer_id로
필터되는 Agent 컨텍스트라) 순서가 permission 체크 → caseGet이지만, top-level
그룹의 관례를 우선한다.

권한 체크 함수/상수:
- 함수: `h.hasPermission(ctx, a, customerID, permissionBitmask)` (기존
  `bin-api-manager/pkg/servicehandler` 공통 헬퍼, 신규 작성 불필요)
- 비트마스크: `amagent.PermissionCustomerAdmin | amagent.PermissionCustomerManager`
  (CaseClose/CaseContinue/CaseUpdateContact와 완전히 동일한 상수 조합)

### 3.2 `ServiceHandler` 인터페이스 (`bin-api-manager/pkg/servicehandler/main.go`)

`CaseClose` 옆에 시그니처 1줄 추가:

```go
CaseAssign(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID, ownerID uuid.UUID) (*cmkase.Case, error)
```

추가 후 `go generate ./...`(mockgen)로 `mock_main.go`를 재생성해
`MockServiceHandler.CaseAssign` / `MockServiceHandlerMockRecorder.CaseAssign`을
생성한다(`ServiceAgentCaseAssign` mock과 동일한 패턴이 자동 생성됨). 수동 편집
금지, 반드시 재생성.

### 3.3 `bin-api-manager/server/contact_cases.go` — `PostContactCasesIdAssign`

`PostContactCasesIdClose`(단순 케이스)와 `PutContactCasesId`(request body
파싱 케이스)의 패턴을 결합한다. `id`는 이 파일의 기존 컨벤션대로
`openapi_types.UUID` 파라미터 + `uuid.UUID(id)` 직접 캐스팅을 사용한다
(service_agents 쪽처럼 `uuid.FromString(id.String())` 에러 분기를 쓰지 않음 —
이 파일 내 기존 핸들러들의 일관된 방식을 따름).

```go
// PostContactCasesIdAssign handles POST /contact_cases/{id}/assign (VOIP-1514):
// Admin/Manager assigns a case's owner agent.
func (h *server) PostContactCasesIdAssign(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostContactCasesIdAssign",
		"request_address": c.ClientIP(),
		"id":              id,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithField("customer_id", a.CustomerID)

	caseID := uuid.UUID(id)

	var req openapi_server.PostContactCasesIdAssignJSONRequestBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not bind request body. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON.").Wrap(err))
		return
	}

	ownerID, err := uuid.FromString(req.OwnerId.String())
	if err != nil {
		log.Errorf("Invalid owner ID format. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_OWNER_ID", "The provided owner_id is not a valid UUID.").Wrap(err))
		return
	}

	res, err := h.serviceHandler.CaseAssign(c.Request.Context(), a, caseID, ownerID)
	if err != nil {
		log.Errorf("Could not assign case. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}
```

생성된 openapi 코드에서 실제 필드/타입 이름(`OwnerId` 및 JSONRequestBody
타입명)은 `go generate` 실행 후 `bin-api-manager/openapi_server`(또는 해당
생성 패키지)를 확인해 정확히 맞춘다. `service_agents_contact_cases.go:124`의
`PostServiceAgentsContactCasesIdAssignJSONBody` / `req.OwnerId` 네이밍
컨벤션을 그대로 따른다.

## 4. 요청/응답 스키마

`POST /contact_cases/{id}/assign`

- Path parameter: `id` (uuid, required) — Case ID.
- Request body (application/json):
  ```json
  { "owner_id": "2a2ec0ba-8004-11ec-aea5-439829c92a7c" }
  ```
  - `owner_id` (string, uuid, required): 배정할 agent의 ID.
  - `owner_type`은 요청 바디에 **포함하지 않는다**. 기존
    `service_agents/contact_cases/{id}/assign`과 동일 정책: 현재 플랫폼의
    Case/Conversation owner 타입은 `agent`만 존재하므로, 서버(정확히는
    `bin-common-handler`의 `ContactV1CaseAssign`)가
    `string(commonidentity.OwnerTypeAgent)`를 wire 상에 하드코딩해 전달한다.
    클라이언트가 owner_type을 지정할 수 없다. **신규 엔드포인트도 이
    정책을 그대로 따르며, 별도 처리를 추가하지 않는다**(기존
    `ContactV1CaseAssign`을 그대로 재사용하므로 자동으로 동일 동작).
- 응답 (200): Case 리소스 전체. 스키마는 기존
  `#/components/schemas/ContactManagerCase`를 그대로 재사용한다
  (ServiceAgent 버전과 완전히 동일 — 별도 응답 스키마 신설 불필요).
- 오류 응답: `400 BadRequest`(잘못된 uuid 형식/JSON), `401 Unauthenticated`,
  `403 PermissionDenied`, `404 NotFound`(case 없음 또는 owner agent
  검증 실패 — 두 경우 모두 동일한 404로 수렴시켜 존재 여부를 노출하지
  않는 anti-enumeration 원칙을 ServiceAgent 버전과 동일하게 유지),
  `500 InternalError`. 추가로 case가 이미 `closed` 상태이면
  `serviceerrors.ErrCaseClosed`가 반환되는데, 이는 `abortWithServiceError`가
  매핑하는 기존 오류 코드 체계를 그대로 사용한다(신규 오류 코드 불필요 —
  `CaseClose`/`ServiceAgentCaseAssign`와 동일 방식으로 이미 매핑되어 있는지
  `abortWithServiceError` 구현을 구현 단계에서 재확인).

## 5. 권한 체크 상세

- 상수: `amagent.PermissionCustomerAdmin | amagent.PermissionCustomerManager`
  (`bin-agent-manager/models/agent` 패키지의 비트마스크 상수).
- 체크 함수: `h.hasPermission(ctx, a, customerID, bitmask)` — top-level
  `CaseClose`/`CaseContinue`/`CaseUpdateContact`/`CaseGet`이 사용하는 것과
  완전히 동일한 헬퍼, 신규 작성 없음.
- 체크 대상 customerID는 **caller의 customerID가 아니라 조회된 case의
  `c.CustomerID`**를 사용한다(top-level 그룹의 공통 패턴 — 크로스 테넌트
  케이스에 대한 우회 방지).
- `a.IsDirect()`인 경우 `serviceerrors.ErrDirectAccessNotSupported` 반환
  (Close/Continue/UpdateContact와 동일, direct-access 토큰으로는 호출 불가).
- Agent 전용 경로(`ServiceAgentCaseAssign`)는 `amagent.PermissionAll`을
  체크하는 반면, 이번 top-level 경로는 Admin/Manager 비트만 허용한다 — 일반
  agent 권한으로는 top-level `/contact_cases/{id}/assign`을 호출할 수 없다.

## 6. Auth/Authorization 경계 원칙 준수

- `bin-contact-manager`의 `caseHandler.Assign`은 (주석에 명시된 대로)
  **권한 판단을 전혀 하지 않는다** — 테넌트 일치 여부만 확인
  (`c.CustomerID != customerID` → `dbhandler.ErrNotFound`). 이 계층에는
  손대지 않으며, 신규 기능에서도 이 원칙(Auth/Authorization은
  bin-api-manager에서만 수행)을 그대로 준수한다.
- `bin-common-handler`의 `ContactV1CaseAssign` RPC 클라이언트도 권한 판단
  없이 단순 wire 변환/전달만 수행 — 변경하지 않는다.
- 신규 `CaseAssign` 서비스 함수(§3.1)가 유일한 권한 판단 지점이다.

## 7. 영향받는 파일 목록

### bin-openapi-manager
- 신규: `openapi/paths/contact_cases/id_assign.yaml`
- 수정: `openapi/openapi.yaml` — `/contact_cases/{id}/assign` 경로 등록
  (`/contact_cases/{id}/close` 등록 블록 바로 아래 위치, §7.1 예시 참고)
- `go generate`로 재생성되는 산출물(예: `bin-api-manager`가 참조하는
  `openapi_server`/`openapi_types` 생성 코드)은 별도 항목이 아니라
  generate 결과물로 함께 반영됨.

### bin-api-manager
- 수정: `pkg/servicehandler/case.go` — `CaseAssign` 추가
- 수정: `pkg/servicehandler/main.go` — `ServiceHandler` 인터페이스에
  `CaseAssign` 시그니처 추가
- 재생성: `pkg/servicehandler/mock_main.go` (`go generate ./...`)
- 신규: `pkg/servicehandler/case_test.go`에 `Test_CaseAssign` 추가 (기존
  `case_test.go`에 이미 `CaseClose` 등의 테스트가 있으므로 같은 파일에
  추가, `serviceagent_case_test.go`의 `Test_ServiceAgentCaseAssign` 케이스
  구성을 참고)
- 수정: `server/contact_cases.go` — `PostContactCasesIdAssign` 핸들러 추가
- 수정: `server/contact_cases_test.go` — `Test_PostContactCasesIdAssign` 추가
  (`Test_PostContactCasesIdClose` 패턴 참고, `service_agents_contact_cases_test.go`의
  assign 테스트도 함께 참고)
- openapi 코드 생성 설정 파일(라우터 등록 등)이 있다면 재생성 과정에서
  자동 반영되는지 구현 단계에서 확인 (`go generate ./...` 범위 내인지 점검)

### 문서 (RST, `bin-api-manager/docsdev/source/`)
- `contact_case_overview.rst`:
  - 138번째 줄 근처 문장 "there is no equivalent top-level admin endpoint
    for assignment"를 정정 — 이제 top-level에도 assign 엔드포인트가
    생겼음을 반영.
  - 엔드포인트 목록 표(`service_agent_case_overview.rst`의 82~83번 줄과
    대칭되는 top-level 표가 있다면 거기)에 `POST /contact_cases/{id}/assign`
    행 추가.
  - "Assigning and Closing" 유사 섹션이 top-level 문서에도 있다면 curl
    예시 추가(`service_agent_case_overview.rst`의 124~128번 줄 예시를
    top-level 경로로 변형).
- `service_agent_case_overview.rst`:
  - 82~83번 줄 및 202번 줄 문장("there is no top-level equivalent")을
    정정 — top-level assign 엔드포인트 존재를 반영.
- `contact_case_struct.rst`: 응답 스키마가 기존 `ContactManagerCase`
  그대로이므로 구조체 문서 변경은 불필요(필드 추가 없음). 다만 assign
  관련 서술이 있다면 교차 확인.
- 빌드: `cd bin-api-manager/docsdev && rm -rf build && python3 -m sphinx -M html source build`
  후 `git add -f bin-api-manager/docsdev/build/`로 커밋에 포함(구현
  단계에서 수행).

## 8. 테스트 계획

### 8.1 servicehandler 레벨 (`pkg/servicehandler/case_test.go`, 신규 `Test_CaseAssign`)

`Test_ServiceAgentCaseAssign`의 테이블 구조를 그대로 차용하되, agent 대신
Admin/Manager identity로 교체:

- 정상 케이스: `PermissionCustomerAdmin`(또는 `PermissionCustomerManager`)
  보유 identity, 유효한 same-customer owner agent → `AgentV1AgentGet` +
  `ContactV1CaseAssign` 모두 호출, 200 상당 결과 반환.
- 권한 없음: `PermissionCustomerAgent`만 보유 → `ErrPermissionDenied`,
  하위 RPC(`caseGet` 이후) 호출 안 됨 검증.
- direct access identity: `a.IsDirect() == true` → `ErrDirectAccessNotSupported`,
  케이스 조회조차 발생하지 않음 검증.
- case가 closed 상태: `ErrCaseClosed` 반환, `AgentV1AgentGet`/`ContactV1CaseAssign`
  미호출 검증.
- owner agent가 다른 customer 소속이거나 조회 실패: `ErrNotFound` 반환
  (case 존재 여부와 owner 존재 여부를 구분하지 못하게 하는 anti-enumeration
  검증 포함).
- case 자체가 존재하지 않음(`caseGet` 에러): 에러 그대로 propagate 검증.
- 크로스 테넌트 case(caller customerID ≠ case customerID): `caseGet`이
  이미 `a.CustomerID`로 RPC 조회를 수행하므로(`case.go:30-44`), 크로스
  테넌트 case ID는 `hasPermission` 체크에 도달하기 전에 `caseGet` 단계에서
  `ErrNotFound`로 귀결될 가능성이 높다(디자인 리뷰 3라운드 반영 — `ErrPermissionDenied`가
  아닐 확률이 높음). 구현 단계에서 `ContactV1CaseGet`의 실제 테넌트
  필터링 동작을 확인해 기대값을 `ErrNotFound`로 확정한다.

### 8.2 server HTTP 레벨 (`server/contact_cases_test.go`, 신규 `Test_PostContactCasesIdAssign`)

`Test_PostContactCasesIdClose` + `service_agents_contact_cases_test.go`의
assign 테스트 두 패턴을 결합:

- 정상: Admin/Manager 토큰, 유효 JSON body(`{"owner_id": "..."}`) →
  `mockSvc.EXPECT().CaseAssign(...)` 호출, `200 OK` + Case JSON 검증.
- 인증 없음: `getAuthIdentity`가 false → `401`.
- 잘못된 JSON body: `BindJSON` 실패 → `400`, `INVALID_JSON_BODY`.
- `owner_id` 필드 누락/잘못된 uuid 형식 → `400`, `INVALID_OWNER_ID`.
- servicehandler가 `ErrPermissionDenied` 반환 → `403`.
- servicehandler가 `ErrNotFound` 반환 → `404`.
- servicehandler가 `ErrCaseClosed` 반환 → `abortWithServiceError`가 매핑하는
  상태 코드(구현 시 `CaseClose` 테스트에서 사용하는 매핑값과 동일한지 확인
  후 반영).

### 8.3 회귀 검증
- 기존 `ServiceAgentCaseAssign`/`PostServiceAgentsContactCasesIdAssign` 관련
  테스트는 변경하지 않음(재사용만 하고 로직 수정 없음) — 회귀 없음을
  전체 `go test ./...` 실행으로 확인.
- `bin-openapi-manager` 빌드/lint(스펙 검증)와 `bin-api-manager`의
  `go mod tidy && go mod vendor && go generate ./... && go test ./... &&
  golangci-lint run -v --timeout 5m` 전체 검증 워크플로우를 구현 단계에서
  수행.

## 9. 미해결/구현 단계에서 재확인할 사항

- `abortWithServiceError`가 `serviceerrors.ErrCaseClosed`를 어떤 HTTP
  상태 코드로 매핑하는지 (`CaseClose` 관련 기존 테스트에서 확인 후
  8.2 테스트 케이스의 기대값 확정).
- openapi 코드 생성 후 실제 생성되는 Go 타입명
  (`PostContactCasesIdAssignJSONRequestBody`, 필드명 `OwnerId` 등)이
  본 설계서의 가정과 정확히 일치하는지 `go generate ./...` 실행 후 확인.
- `case.go` 내 `CaseAssign`의 `caseGet` → `hasPermission` 순서가 §3.1에서
  선택한 대로(top-level 그룹 우선) 실제 코드 컨벤션과 100% 일치하는지,
  구현 시작 직전 `case.go` 최신 상태를 재열람해 재확인.
- **[디자인 리뷰 2라운드 반영] 동시 재배정 충돌(알려진 제약, 이번 설계로
  신규 도입하지 않음)**: `bin-contact-manager/pkg/dbhandler/kase.go:299-315`
  `CaseUpdateOwner`는 단순 `UPDATE ... WHERE id, customer_id`만 수행하며
  낙관적 락/버전 체크가 없고, `pkg/casehandler/assign.go:22-37`의 `Assign`도
  이전 owner에 대한 알림/이벤트 발행이 없다. 즉 상담원 A가 self-assign한
  case를 관리자가 동시에 B로 재배정하면 A에게 알림 없이 소유권이 조용히
  덮어써진다. 이는 기존 `service_agents` 경로에도 이미 있는 동작이며 이번
  설계가 신규로 만드는 문제는 아니지만, 이번 기능은 "관리자가 이미 작업
  중인 case를 강제로 재배정"하는 유스케이스를 명시적으로 활성화하므로
  (§1 배경) 이 제약을 알려진 한계로 명시해 둔다. 낙관적 락/변경 알림 도입은
  이번 스코프 밖(별도 티켓 필요 시 후속 논의).
- **[디자인 리뷰 2라운드 반영] owner_type 확장 시 두 엔드포인트 동시 영향**:
  owner_type 하드코딩은 엔드포인트별 코드가 아니라 두 엔드포인트가 공유하는
  `bin-common-handler/pkg/requesthandler/contact_cases.go:247-256`의
  `ContactV1CaseAssign` 내부에 있다. 따라서 향후 team/queue owner 등으로
  owner_type을 확장하려면 이 공유 지점 한 곳만 수정하면 되어 락인 자체는
  작지만, 그 수정은 Agent용(`service_agents/contact_cases/{id}/assign`)과
  Admin용(`contact_cases/{id}/assign`) 두 엔드포인트 모두에 동시 적용되며
  독립적으로 한쪽만 확장할 수 없다는 점을 인지하고 있어야 한다(구현 단계
  변경 없음, 향후 확장 논의 시 참고용 기록).
- **[디자인 리뷰 3라운드 권고, 필수 아님]** 이번 기능이 관리자에 의한
  강제 재배정 유스케이스를 새로 활성화하는 만큼, 낙관적 락 또는 재배정
  알림(이전 owner에게 통지) 도입 여부를 VOIP 백로그에 별도 후속 티켓으로
  등록해 두는 것을 권고한다. 이번 티켓의 구현 스코프에는 포함하지 않는다.
