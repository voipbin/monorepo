# VOIP-1566 extension 자기 등록 차단과 소유 판정 강화 (설계 + 구현 계획)

- 상태: Draft (Design Review 대기)
- 작성일: 2026-10-06
- 티켓: VOIP-1566
- 기준 코드: origin/main bc68e3c32
- 이슈 분석: Jira VOIP-1566 코멘트(분석 확정 사항), 이슈 분석 리뷰 10회차, 9-10회차 연속 승인 종료.

## 1. 문제

VOIP-1565 로 Agent 가 같은 고객사 extension 을 조회하되 타인 extension 의 `password`, `direct_hash` 는 마스킹된다. 소유 판정은 `agentGet` 으로 재조회한 `Addresses` 의 `type=extension` target 을 `uuid.FromStringOrNil` 로 파싱해 한다. 다음 두 결함이 이 보호를 우회한다.

1. `PUT /service_agents/me/addresses` 는 `IsAgent()` 만 확인한다(VOIP-956 부터, 회귀 아님). 일반 상담사가 extension 주소를 자기 주소로 등록할 수 있다.
2. extension target 은 `NormalizeTarget` 에서 identity 이고 중복 검사와 DB `UNIQUE(customer_id,type,target)` 는 문자열 exact 비교다. 하이픈 없는 32hex, `{...}`, `urn:uuid:...` 형식은 타 agent 소유 extension 에 대해서도 중복 검사를 통과하고 `FromStringOrNil` 은 같은 UUID 로 해석하므로 소유 판정이 true 가 된다. 미할당 extension 은 정상 형식으로도 등록된다. (대문자 형식은 운영 DB 기본 `utf8_general_ci` 로 막힐 가능성이 크고 조건부.)

결과: 같은 고객사 내부자가 임의 extension 의 SIP password, direct_hash 를 평문으로 열람(toll fraud 가능, P2). 또한 canonical 형식으로 등록된 extension 은 해당 extension 발착신 call 의 소유자가 된다.

## 2. 대표님 결정 (확정, 번복 금지)

- (a) extension 주소는 Admin/Manager 권한의 관리자 경로(`PUT /agents/{id}/addresses`, square-admin 사용)로만 등록한다. `PUT /service_agents/me/addresses` 에서는 역할과 무관하게 extension 주소의 추가와 변경을 거부하고 기존 유지와 제거만 허용한다.
- (b) tel, sip 주소는 비관리자 자유 등록을 유지한다(VOIP-956 self-service). 위험 수용 범위는 12절.

## 3. 목표

1. (방어선 1) 소유 판정은 저장 target 이 canonical UUID 문자열과 일치할 때만 인정한다.
2. (방어선 2) `ServiceAgentMeUpdateAddresses` 는 extension 주소의 추가와 변경을 거부한다.
3. (방어선 3) agent-manager 가 저장 전 extension target 을 canonical 화하고 기존 데이터를 정리할 수 있게 한다.
4. (방어선 4) tel 주소는 숫자가 없는 값을 거부한다.

## 4. Non-goals

- `normalize.go` 의 `NormalizeTarget` 전역 변경(호출처 다수, 영향 큼).
- extension 쪽 소유권 기록(옵션 C, 장기 과제).
- transcribe, transcript, aicall 의 고객사 전체 노출 및 `transcribeGetResourceInfoForAgent` break 누락(이슈 분석상 별도 독립 문제, 티켓 미등록, 대표님 결정).
- tel, sip 소유 선점 방지, `groupcallhandler getAddressOwner` 목적지 비정규화.
- 관리자 경로 권한 변경.

## 5. 결정 사항 (D1 ~ D8)

| # | 결정 | 근거 |
|---|---|---|
| D1 | 방어선 2 비교 규칙. 요청의 각 extension 주소는 호출 시점 `agentGet` 으로 재조회한 저장 주소 중 `(Type, Target, TargetName)` 이 문자열 exact 로 일치하는 항목이 있고, 그 저장 `Target` 이 canonical(`uuid.FromStringOrNil(t).String()==t` 이고 Nil 아님)일 때만 허용한다. 일치 항목이 없으면(추가, `Target` 또는 `TargetName` 변경 포함) 거부한다. 저장 target 이 비-canonical 인 row 는 재제출해도 거부된다(제거만 가능). `Name`, `Detail` 은 비교하지 않는다(라우팅, 소유와 무관). extension 항목이 하나도 없는 요청은 기존 extension 을 제거하는 것으로 해석한다(PUT 전체 교체 의미 유지). 거부 오류는 `cerrors.PermissionDenied(ServiceNameAPIManager, "EXTENSION_ADDRESS_ADMIN_ONLY", "...")`(HTTP 403). 역할(관리자 포함)은 판정하지 않는다 | 대표님 결정 (a), 분석 8회차 canonical 재제출 승격 방지 |
| D2 | 방어선 2 의 read-then-update 경쟁은 수용한다. `AgentSetAddresses` 는 행 잠금 후 전체 교체(last-writer-wins)이며, 관리자가 extension 을 회수한 직후 구버전 목록을 제출하면 회수분이 되살아날 수 있는 좁은 P3 창이 있다. 같은 agent 의 자기 요청이고 서버 측 CAS 는 agent-manager 변경 범위를 키우므로 도입하지 않고 문서화한다 | 오버엔지니어링 지양, 실측 신호 없음 |
| D3 | 방어선 3 구현. agent-manager `agenthandler` 에 `canonicalExtensionTarget(target) (string, error)` 헬퍼(`uuid.FromString` 후 `.String()`, 실패 시 오류)를 둔다. `UpdateAddresses` 는 extension 분기에서 검증 직후 target 을 canonical 로 교체한 뒤 중복 검사와 저장을 한다. `Create` 는 extension 주소가 UUID 로 파싱되지 않으면 오류, 파싱되면 canonical 로 저장한다(registrar 존재 검증은 추가하지 않음, 관리자 경로이고 RPC 비용 회피). `GetByCustomerIDAndAddress` 의 조회 키도 extension 이면 canonical 화한다. `normalize.go` 는 변경하지 않는다 | 영향 최소화 |
| D4 | 기존 DB 정리. `cmd/agent-control normalize-addresses` 의 `normalizeAddresses` 에 extension UUID canonical 화를 추가한다(도구 내부 로컬 함수, 공유 `NormalizeTarget` 불변). 기존 도구의 충돌 hard-fail 은 유지한다. 변형 row 가 타 agent 의 canonical row 와 충돌하면 우회 흔적이므로 hard-fail 이 탐지 신호가 된다. 자동 변환은 취약 기간에 자기 등록된 canonical row 를 정당화하지 않으며(DB 에 등록 주체 기록이 없음), 실행 전 관리자가 extension 보유 agent 전수 목록을 검토하는 절차를 runbook 으로 둔다(11절). 롤백은 dry-run 우선, 변환 전 `agent_addresses` 백업 | 분석 8회차 |
| D5 | 운영 DB collation 확인은 배포 선행 항목이다(`SHOW FULL COLUMNS FROM agent_addresses`). 코드 결정에는 영향이 없다(canonical 이 소문자 하이픈형이라 저장과 조회가 일치) | 대문자 조건부 우회 |
| D6 | 방어선 1 구현. `serviceAgentOwnedExtensionIDs` 에서 `id != uuid.Nil && id.String() == address.Target` 일 때만 owned. 호출처(List, Get)는 마스킹 전용이라 정상 canonical row 영향 없음. 비-canonical row 소유자는 마스킹된다(fail-closed) | 분석 방어선 1 |
| D7 | 방어선 4. agent-manager `UpdateAddresses` 와 `Create` 의 tel 처리에서 `NormalizeTarget(TypeTel, target)` 이 `ErrNotNormalizable`(숫자 없음)을 반환하면 거부한다. 기준은 숫자 포함 여부이며 E.164 엄격 검증은 사용하지 않는다(내선형 짧은 번호 등 기존 입력 보호). sip 는 변경 없음 | 소형, anonymous 귀속 차단 |
| D8 | 문서 갱신. `service_agent_overview.rst` 의 me/addresses 설명에 extension 추가, 변경 불가와 에러 사유를 명시하고 OpenAPI `me_addresses.yaml` description 과 403 응답을 갱신, redoc, docsdev/build 재생성. 외부 클라이언트 영향은 PR 본문에 고지 | 영향 고지 |

## 6. 배포 순서

1. 운영 DB collation 확인(D5).
2. agent-manager 배포(D3, D7 포함).
3. 정리 실행(D4 runbook): 관리자 검토, dry-run, 백업, 소비자 정지, apply.
4. api-manager 배포(D1, D6). 정리 전에 방어선 1 이 나가면 비-canonical 로 저장된 정당한 소유자의 password 열람이 끊기므로 순서를 지킨다.

같은 PR 에 포함하되 PR 본문에 위 순서를 명시한다.

## 7. 구현 계획

### 7.1 api-manager
- `pkg/servicehandler/serviceagent_extension.go`: D6.
- `pkg/servicehandler/serviceagent_me.go`: `ServiceAgentMeUpdateAddresses` 에서 `IsAgent()` 후 요청에 extension 주소가 있으면 `agentGet` 재조회, D1 규칙 검사, 거부 시 PermissionDenied 오류. extension 주소가 요청에 없으면 `agentGet` 호출을 생략한다(불필요한 RPC 회피). 위반 시 `agentUpdateAddresses` 호출 안 함.
- `server/service_agents_me.go`: 오류가 `abortWithServiceError` 로 403 이 되는지 확인.

### 7.2 agent-manager
- `pkg/agenthandler/agent.go`: D3, D7. `Create`, `UpdateAddresses`, `GetByCustomerIDAndAddress`.
- `cmd/agent-control/normalize_addresses.go`: D4.

### 7.3 문서와 생성물
- RST, OpenAPI yaml, redoc, `docsdev/build` 의 실제 변경 파일만 커밋.

### 7.4 테스트
- D1 표 전수: 추가, Target 변경, TargetName 변경, 변형 재제출, 정상 유지, 제거, extension 없는 요청, 다건 혼합, tel 만 요청(재조회 생략), 재조회 실패(fail closed).
- D6: canonical, 변형 형식(32hex, 중괄호, urn, 대문자), Nil.
- D3: 변형 target 이 canonical 로 저장되고 타 agent canonical 소유 시 중복 거부, `Create` 변형 canonical 저장, 비-UUID 거부.
- D7: tel `anonymous` 거부, 숫자 포함 허용, sip 영향 없음.
- D4: extension 변형 row 가 dry-run 에서 변경으로 집계, 충돌 시 hard-fail.
- 변이 시험으로 각 방어선 제거 시 테스트가 실패함을 확인.

## 8. 위험과 트레이드오프

- 비-canonical row 보유 agent 는 방어선 2 로 재제출이 거부되고 방어선 1 로 마스킹된다. 정리(D4) 선행으로 완화.
- 관리자가 `me/addresses` 로 extension 을 자기 등록하는 기능은 사라진다. 호출처 전수 확인 결과 square-talk 는 thunk 정의만 있고 UI 호출이 없으며 square-admin 은 관리자 경로를 쓴다.
- 외부 API 직접 호출 클라이언트가 extension 을 me 경로로 등록하던 경우 403 이 된다. PR 본문과 문서에 고지.

## 9. 범위 밖 후속

- transcribe, transcript, aicall 고객사 전체 노출 및 break 누락(티켓 미등록).
- 옵션 C(extension 쪽 소유권 기록).

## 10. 문서 영향

- RST, OpenAPI, redoc, docsdev/build (D8).

## 11. 정리 runbook (D4)

1. `SELECT agent_id, type, target, target_name FROM agent_addresses WHERE type='extension'` 으로 전수 목록을 추출해 관리자가 각 보유 관계가 의도된 것인지 검토한다(취약 기간의 자기 등록 의심 건 제거).
2. `agent-control normalize-addresses --dry-run` 으로 변경과 충돌을 확인한다. 충돌은 조사 후 수동 해결.
3. `agent_addresses` 백업 후 소비자를 멈추고 `--dry-run=false` 로 적용한다.

## 12. 수용한 위험 (대표님 결정)

tel, sip 자유 등록을 유지한다. 아무도 등록하지 않은 번호를 선점하면 그 번호의 수신, 발신 call 의 소유자가 되어 call 메타데이터(`recording_ids` 포함)와 owner 토픽 실시간 이벤트를 볼 수 있다(이미 다른 agent 가 소유한 번호는 중복 검사로 막힘). 자기 agent 주소로 임의 tel, sip 를 두어 자기에게 오는 호의 착신 대상을 정하면 고객사 PSTN 비용이 발생할 수 있다(VOIP-956 본래 성질). 녹취 본체는 Admin|Manager 전용이다. 별개의 기존 노출로 `ServiceAgentTranscribeStart` 는 call 참조를 고객사 일치만 확인하므로 call 소유와 무관하게 고객사 내 어느 agent 든 call id 로 transcript 를 읽을 수 있다(범위 밖).
