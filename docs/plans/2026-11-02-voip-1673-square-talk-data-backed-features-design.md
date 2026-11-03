# VOIP-1673: square-talk 워크스페이스 데이터 기반 기능 백엔드 설계 (A, B, C)

- 상태: 설계, 개정 1 반영 (코드 변경 없음, 설계 문서만 포함하는 PR)
- 기준: monorepo `77b29166d`, monorepo-javascript `c39bbacb` (개정 1 의 1.5절과 3.10 의 call-manager 확인은 monorepo `242fa584e` 기준)
- 범위: 클라이언트만으로 만들 수 없는 square-talk 워크스페이스 요소의 백엔드 설계. 구현은 이 설계가 확정된 뒤 별도 작업으로 진행합니다.
- 제품 전제: 통화 중 Insight 는 케이스에 묶인 통화에서만 동작합니다. 이 전제를 넘는 범위 확대는 이 문서에 넣지 않았습니다.
- 제품 전제 (개정 1 추가): square-talk 사용자는 회사 또는 그룹의 소속원이며 상식적인 조직 규율을 이해한다고 봅니다. 따라서 통화 중 녹음을 누를 때마다 녹음 동의를 요청하는 팝업이 필요하지 않습니다. 이 전제는 에이전트의 규율 준수에 대한 신뢰만 뜻하며, 테넌트 격리, 통화 소유자 검증, 고객 일치 검증, 외부 PSTN 비용 안전은 유지합니다.

## 개정 이력

개정 1: 위 전제 추가에 따라 C2 녹음 정책을 정리하고, 관련 주장을 코드로 다시 확인해 반영했습니다. (1) P1(녹음 고지)을 "조직 규율, 서버 강제 없음" 으로 결정하고 클라이언트 REC 표시를 의무에서 UX 권고로 낮췄으며, RST 한 줄 명시를 구현 범위에 추가했습니다. (2) P7(녹음 중지)에서 시작 주체 표식 논의와 "중지는 제공하지 않음" 선택지를 삭제하고 소유 에이전트의 start/stop 을 모두 허용하되 감사 로그만 남기게 했습니다. (3) `duration` 상한을 실제로 작동하게 고쳤고(0 은 무제한이라 `minimum: 1` 과 기본값 7200), `end_of_silence` 상한은 제거했습니다. (4) P2/P3 의 주소 유형 제한을 "보안 경계" 가 아닌 "단계적 출시 선택" 으로 다시 서술하고 (a) `extension` 만, (b) `tel` 포함 으로 정리해 남은 결정 사항으로 두었습니다. (5) 플로우 `recording_start` 와의 충돌을 새 위험으로 문서화했습니다(통화가 끊기는 것이 아니라 액션이 건너뛰어짐). (6) call-manager 기존 결함이 한 줄 수정 범위를 넘는 것을 확인해 3.10 의 수정 범위(B, C, D)와 규모를 재산정했습니다(C2 3일 -> 5일, C1 4일 -> 3일, C3 10일 -> 9일, 합계 동일). 영향받는 곳: 1.2, 1.5(신설), 2.1, 2.3, 2.4, 3.9, 3.10, 3.11, 3.13, 5, 6. 3.1~3.8(A1, A2, A4, A5, B1~B3)은 변경하지 않았습니다.

개정 2 (C3 구현 착수 시, 스파이크 실측 반영): 스파이크 S1~S5, S7 을 실제 Asterisk 환경에서 실측한 결과 설계 Y안(confbridge 재사용)은 유지되고, 3.11 에 다음 다섯 가지를 정정했습니다. (a) 에이전트가 직접 건 발신 통화(토폴로지 B)에서 에이전트가 끊으면 제3자까지 전원 종료됩니다(이전 서술 "제3자 유지" 정정, 회피책은 구현하지 않음(트리거: 제품 요구가 생기면 별도 설계)). (b) extension 목적지의 응답은 call 이 아니라 groupcall 이므로 응답을 `CallManagerGroupcall` 로 정정. (c) 발신 주소는 `target_name` 에 내선 번호. (d) 참가자 수 상한 검증을 api-manager 와 call-manager 에 신규 구현(409, reason 구분). (e) 같은 레그 녹음 재시작은 C2 의 사전 409 가 이미 처리. 영향받는 곳: 3.11, 6. 나머지 절은 변경하지 않았습니다.

## 1. 이슈 확인과 분석

### 1.1 티켓 검토 (착수 전 적대적 검토)

판정: **부분 유효**. 모든 항목은 기준 커밋(`git fetch` 후 확인한 main)의 실제 코드로 다시 확인했고, 요청 시점에 전제로 주어진 주장 중 일부는 틀리거나 보정이 필요했습니다.

반증하지 못한 주장 (코드 근거):

| 주장 | 근거 |
|---|---|
| 메시지 멘션 집계/조회 API 가 없다 | `bin-talk-manager/pkg/messagehandler/message.go:127` 이 본문 `text` 를 가공 없이 저장. 손으로 작성한 서버 코드(talk-manager, api-manager 의 비테스트, 비생성 Go 파일을 `mention` 으로 grep 한 결과 0건)에 멘션 처리 코드 없음 |
| Pinned 데이터가 없다 | talk 3개 테이블(`talk_chats`, `talk_messages`, `talk_participants`, `migrations/000001_baseline.up.sql`)에 pin 류 컬럼 없음 |
| Starred 데이터가 없다 | `contact_contacts` 에 해당 컬럼 없음. 태그(`/service_agents/contacts/{id}/tags`)는 customer 전체 공유 상태라 에이전트별 상태가 아님 |
| Transfer, Record 는 에이전트 경로가 없다 | `bin-api-manager/pkg/servicehandler/transfer.go:39`, `call.go:749`(`CallRecordingStart`) 가 Admin\|Manager 만 허용. OpenAPI `/transfers` 는 POST 만 있고 `/service_agents/calls` 목록과 `/service_agents/calls/{id}` 상세에는 조회(GET)만 노출돼 있음(`ServiceAgentCallDelete` 는 서비스 핸들러에만 있고 미라우팅) |
| Disposition 입력이 없다 | `bin-contact-manager/pkg/casehandler/lifecycle.go` 의 `Close` 가 `kase.ClosedReasonAgentClosed` 를 고정 전달. close 요청에 body 없음 |
| 큐 이름을 에이전트가 읽을 수 없다 | `/queues`, `/queuecalls` 는 Admin\|Manager 전용(`bin-api-manager/pkg/servicehandler/queue.go`, `queuecall.go`). queuecall 의 외부 응답(`WebhookMessage`)에 `queue_id` 없음 |
| SLA, 우선순위 필드가 없다 | `kase.Case`, `queuecall.Queuecall` 에 해당 필드 없음. queue 에는 `wait_timeout`, `service_timeout` 만 있음 |

틀렸거나 보정이 필요한 주장:

1. "통화 중 Insight 는 신규 백엔드가 필요 없을 수 있다": listen 이 청취하는 통화는 `case.ReferenceID`(케이스를 만든 통화 id)입니다(`bin-ai-manager/pkg/aicallhandler/listen_trigger.go` 의 `checkListenEligible`). 그런데 통화에서 케이스로 가는 서버 경로가 없습니다. `GET /service_agents/contact_cases` 는 `reference_id` 필터를 노출하지 않고 api-manager 가 빈 문자열로 고정합니다(`bin-api-manager/pkg/servicehandler/serviceagent_case.go` 의 `ServiceAgentCaseList`). contact-manager 는 `reference_id` 필터와 인덱스(`idx_case_customer_reference_id`)를 이미 가지고 있어, api-manager 와 OpenAPI 의 배선만 부족합니다.
2. "webphone 통화로 queuecall 을 찾을 수 있다": webphone 의 `currentCall.id` 는 클라이언트가 만든 타임스탬프(`monorepo-javascript/square-talk/src/lib/webphone.js`)이고 서버 call id 가 아닙니다. 서버 통화는 `useLiveCallMapping` 이 `GET /service_agents/calls` 를 폴링해 하나로 좁혀질 때만 얻습니다(0건 또는 2건 이상이면 실패).
3. "연락처 Notes 는 클라이언트 N+1 로 충분하다": 케이스 노트 목록(`bin-contact-manager/pkg/dbhandler/casenote.go` 의 `CaseNoteListByCase`)은 페이지네이션이 없고 `tm_create ASC` 전체를 돌려줍니다. 연락처 단위 시간순 합산 페이지를 클라이언트에서 만들 수 없습니다(3절 A1).
4. "통화가 케이스에 묶인다": `case_create` 플로우 액션이 `af.ReferenceID`(그 플로우가 실행되는 통화 id)를 `reference_id` 로 저장합니다(`bin-flow-manager/pkg/activeflowhandler/actionhandle.go` 의 `actionHandleCaseCreate`). 열린 케이스의 유일 키는 `customer_id + peer_type + peer_target + reference_type` 이고(`bin-contact-manager/migrations/000001_baseline.up.sql` 의 `open_peer_uk`, `status='open'` 일 때만), 위반하면 `AlreadyExists`(`bin-contact-manager/pkg/casehandler/create.go:57`)가 되며 플로우는 로그만 남기고 계속합니다. 따라서 같은 peer 에 같은 reference_type(call) 의 열린 케이스가 있으면 두 번째 통화는 어느 케이스에도 묶이지 않습니다. 제품 전제("케이스에 묶인 통화에서만")와 일치하는 현행 동작이며 한계로 명시합니다.

### 1.2 항목별 사실 확인 요약

경로는 monorepo 루트 기준이며 서비스 디렉터리 접두를 생략한 것은 바로 앞에서 명시한 서비스의 `pkg/` 아래입니다.

| 항목 | 확인한 사실 | 판정 |
|---|---|---|
| A1 | 케이스 목록은 `contact_id` 필터 가능(`server/service_agents_contact_cases.go`, page_size 상한 100). 케이스 노트 목록은 케이스 단위, 무페이지 | 집계 엔드포인트 필요 |
| A2 | 청취 대상 = 케이스의 `reference_id`. 케이스 조회에 `reference_id` 필터 배선 없음(`reference_type` 필터도 없음). 에이전트가 보는 통화(`/service_agents/calls`, `CallManagerCall` 스키마)는 `id` 와 `master_call_id` 를 노출(`bin-call-manager/models/call/webhook.go`). 체인 전체와 listen 성공 조건은 1.2.1 | 케이스 조회 배선 필요, 단 listen 성공 조건은 별도(1.2.1) |
| A3 | 케이스 노트 쓰기/읽기 이미 존재. 케이스가 없는 통화에는 노트 대상이 없음 | 통화 단위 노트 신설 불필요 |
| A4 | `contact_cases.closed_reason` 은 `agent_closed` / `merged`(예약)만. 케이스 태그는 contact-manager 에 이미 구현돼 있음(`kase.go` 의 `TagIDs`, `casehandler/case_tag.go` 의 `CaseTagAdd/Remove/List`, 내부 라우트 `/v1/cases/{id}/tags`, RPC `ContactV1CaseTagAdd/List`)이지만 api-manager/OpenAPI 에는 admin, service_agents 어느 쪽에도 노출이 없음(`bin-openapi-manager/openapi/paths/contact_cases/` 에 tags 파일 없음). `service_agents` 경로에 태그가 있는 것은 연락처 태그뿐 | 확장 필요, 3절에서 비교 |
| A5 | 아래 1.3 참조 | 매핑 가능, 읽기 API 신설 필요 |
| B1~B3 | 1.2.2 참조. talk, contact 의 최신 마이그레이션 `000001`, talk 에 트랜잭션 헬퍼 없음 | 신규 테이블 필요 |
| C1 | `bin-transfer-manager` 는 호출자/소유자/고객을 검증하지 않음(`pkg/transferhandler/transfer.go` 의 `transferInit` 은 call 만 조회). api-manager 검증이 유일한 방어선. 전제: transferer 통화에 `ConfbridgeID` 가 있어야 함(`transfer.go:63~66`, 없으면 실패) | 에이전트 경로 신설 |
| C2 | `call-manager` 의 `RecordingStart/Stop` 존재. 녹음 고지/동의를 강제하는 서버 로직 없음(`CallRecordingStart` 부터 `callhandler.RecordingStart`, `recordinghandler` 까지 consent/announce/disclosure/notice 를 다루는 코드가 없음을 grep 으로 확인. 고지는 플로우의 `talk` 액션에 맡겨져 있고 `docsdev/source/recording_overview.rst` 는 권고만 함) | 에이전트 경로 신설. 고지 정책은 개정 1 에서 결정됨(P1). call-manager 기존 결함은 1.5 |
| C3 | 전용 API 없음. 2자 통화는 `connect` 액션으로 처음부터 confbridge 안에서 만들어짐. Insight listen 은 참가자 정확히 2명일 때만 시작(1.2.1) | 설계안 3개 비교 |
| C4, C5 | 필드와 제품 정의 모두 없음 | 정의 초안 + 질문 목록 |

#### 1.2.1 A2: 통화 -> 케이스 -> Insight -> listen 체인 (코드로 확인)

1. 통화에서 케이스 찾기(서버에 없음, 신설 대상): `case_create` 는 그 플로우가 실행되는 통화의 id 를 `reference_id` 로 저장합니다(`actionHandleCaseCreate`). 큐 경유 통화라면 이 통화는 고객 레그이고, 에이전트가 webphone 에서 보는 서버 통화(에이전트 레그)에서는 `master_call_id` 로 닿습니다. 에이전트가 직접 건 발신이면 에이전트 레그 자체가 `reference_id` 일 수 있습니다. 따라서 조회 키 후보는 에이전트 레그의 `id` 와 `master_call_id` 두 개이며, `reference_type` 조건 없이 `reference_id` 만으로 조회됩니다(`ServiceAgentCaseList` 가 넘기는 필터 구조). `master_call_id` 는 1.3 처럼 비어 있을 수 있습니다.
2. 케이스에서 Insight 세션 만들기(이미 존재): `ServiceAgentAIcallCreate`(`bin-api-manager/pkg/servicehandler/serviceagent_aicall.go`)가 `reference_type=contact_case`, `reference_id=<case id>` 를 받고 `assistance_type=ai` 이면 `assistance_id` 를 생략할 수 있습니다(고객에게 Insight AI 가 없으면 `ErrNotFound`, `resolveInsightAIID`). 이전 aicall 이 끝났다면 `AIcallContactCaseRecreateRateLimitMinutes` 동안 재생성이 제한됩니다(`bin-ai-manager/pkg/aicallhandler/start.go` 의 `checkContactCaseRecreateRateLimit`).
3. 청취 시작(이미 존재): `POST /service_agents/aicalls/{id}/listen`. `checkListenEligible` 는 AI `Type=Insight`, aicall `Status=Progressing`, aicall `ReferenceType=contact_case`, 케이스 `ReferenceType=call`, 케이스 `reference_id` 의 통화가 같은 고객이고 아직 살아 있을 것(`isListenableCallStatus`)을 모두 요구하며, 하나라도 어긋나면 조용히 스킵합니다(`listen_trigger.go`). 그 뒤 `runListenStart` 의 `waitForConfbridgeReady` 는 통화의 confbridge 가 `Progressing` 이고 참가자 수(`ChannelCallIDs`)가 **정확히 2** 일 때만 ready 로 보며, 그렇지 않으면 대기 시간 후 `skipped_confbridge_not_ready` 로 끝납니다(`listen_trigger.go:829~` 의 `lastPartyCount == 2`). 따라서 3자 이상 통화에서는 Insight 청취가 시작되지 않습니다(C3 설계 전제).
4. 한계: 통화가 케이스에 묶이지 않는 경우는 같은 peer 의 열린 케이스 중복(1.1 의 4번) 외에도 있습니다. 플로우에 `case_create` 액션이 없는 경우, CRM 대상이 아닌 peer(`isCRMEligiblePeer`), 같은 activeflow 의 `contact_case_id` 변수가 이미 있는 경우, 통화 또는 대화 조회 실패(이 경우 `af.ReferenceID` 의 `uuid.Nil` 분기에 닿기 전에 종료), activeflow reference 타입이 call/conversation 이 아닌 경우입니다(`actionHandleCaseCreate`, `actionhandle.go:1328~1351`; 어느 경우든 플로우는 계속 진행). 따라서 조회 결과가 비어 있는 것은 정상 경로이며 Insight 를 노출하지 않는 것이 올바른 동작입니다. 같은 peer 의 두 번째 통화처럼 케이스가 있어도 그 `reference_id` 통화는 이미 끝났을 수 있어 listen 이 스킵됩니다. 제품 전제("케이스에 묶인 통화에서만")에 맞는 동작이지만, "케이스가 조회된다"가 "listen 이 성공한다"를 보장하지 않으므로 클라이언트는 케이스 조회 결과의 `reference_id` 가 현재 통화(또는 그 master)와 일치할 때만 Insight 를 노출해야 합니다.

#### 1.2.2 B1~B3: 대상 정의와 talk-manager 접근 경로

- B1 Pinned: 에이전트가 자기 화면에서 채팅(chat) 또는 메시지(message)를 고정하는 개인 상태. B2 Starred: 에이전트가 연락처를 즐겨찾기하는 개인 상태. B3 Mentions: 다른 사람이 본문에서 나(에이전트)를 멘션한 메시지 목록.
- 에이전트의 talk 접근 규칙(`bin-api-manager/pkg/servicehandler/serviceagent_talk.go`): 읽기는 `canAccessChat`(`talk` 타입 채팅은 같은 고객의 모든 에이전트, direct/group 은 참여자만), 쓰기는 `isParticipantOfTalk`, 삭제는 작성자 본인(`OwnerID == a.AgentID()`). 메시지 목록은 `chat_id` 필수의 채팅 단위입니다(`ServiceAgentTalkMessageList`, OpenAPI `talk_messages.yaml`). `talkMessageListByChatIDs` 는 여러 chat id 를 받는 이름이지만 첫 번째 id 하나로만 필터하는 임시 구현입니다(`serviceagent_talk.go:488~502`, TODO 주석). 따라서 채팅을 가로지르는 멘션 집계는 현재 경로로 불가능합니다.
- talk 모델: 소유자는 `owner_type + owner_id` 복합 키(`talk_participants.unique_participant`, `talk_messages.idx_owner`), 메시지에는 `text`, `medias`, `metadata`(reactions 만) 가 있고 `text` 컬럼에는 인덱스가 없습니다. 멘션을 컬럼/별도 테이블로 둘지 `metadata` 에 둘지는 3절 B3 에서 결정합니다.
- talk-manager 에는 트랜잭션 헬퍼가 없고(`pkg/dbhandler` 에 `Begin`/`Commit` 사용 0건), 목록 `page_size` 는 내부 RPC 에서 상한이 없으며 api-manager 에서 기본/상한 100 입니다.

### 1.3 A5: queuecall 과 에이전트 레그 매핑 (코드로 확인)

- 큐 가입 시(`queue_join` 액션은 `af.ReferenceType` 을 확인하지 않고 `ReferenceTypeCall` 로 전달하므로, 통화 플로우에서 실행될 때만 "고객 레그" 서술이 성립합니다) `queue_join` 액션이 `QueueV1ServiceTypeQueuecallStart(opt.QueueID, af.ID, ReferenceTypeCall, af.ReferenceID)` 를 호출(`bin-flow-manager/pkg/activeflowhandler/actionhandle.go:775`)하고, `queuecallhandler/db.go` 의 `Create` 가 그 값을 `queuecall.reference_id` 로 저장합니다. 큐에 들어온 통화(고객 레그)의 call id 입니다.
- 에이전트가 선택되면 `bin-queue-manager/pkg/queuecallhandler/execute.go` 가 `CallV1CallsCreate(customerID, flowID, masterCallID=qc.ReferenceID, source, [{type: agent, target: agentID}])` 를 호출하고, 반환된 groupcall id 를 `UpdateStatusConnecting` 으로 `queuecall.groupcall_id` 와 `service_agent_id` 에 저장합니다(execute.go:68~94).
- call-manager 는 에이전트 레그를 만든 뒤 `ChainedCallIDAdd` 로 `master_call_id` 를 별도 쓰기로 설정합니다(`bin-call-manager/pkg/callhandler/outgoing_call.go:375~383` → `chained_call.go:45` 의 `CallSetMasterCallID`; 생성 시점의 `db.go:103` 은 `uuid.Nil`). 마스터 통화가 `terminating/canceling/hangup` 이면 이 쓰기는 실패하고 호출부는 로그만 남기고 진행하므로(`outgoing_call.go:377~380`), 레그의 `master_call_id` 가 비어 있을 수 있습니다. `owner_id` 는 주소로 찾은 에이전트(또는 상위 groupcall)에서 채우지만, 조회 실패는 로그만 남기고 진행하고 상위 groupcall 의 owner 가 None 이면 상속도 일어나지 않으므로(`outgoing_call.go:303~331`) 비어 있을 수 있습니다. 비어 있는 레그는 `ServiceAgentCallList` 의 `owner_id` 필터와 `ServiceAgentCallGet` 의 소유자 비교(`serviceagent_call.go:41, 77`)에서 에이전트에게 보이지 않습니다. 이 문서의 모든 "에이전트 소유 통화" 전제는 이 경우 fail-closed 로 동작합니다.
- 따라서 `master_call_id` 가 채워진 에이전트 레그에서는 **`master_call_id` == `queuecall.reference_id`** 이고, 서버는 `QueueV1QueuecallGetByReferenceID` 로 queuecall 을 찾을 수 있습니다(`dbhandler/queuecall.go` `QueuecallGetByReferenceID`, `tm_create DESC` 의 최신 1건).
- 에이전트 레그 call id 를 queuecall 에 저장하는 필드는 없습니다. ring_all 이면 레그가 여러 개 생기며 어느 레그가 답했는지는 queuecall 에 없습니다. 그래서 키는 "에이전트 소유의 서버 통화 하나(소유자 검증 통과)"이고, 서버는 `queuecall.service_agent_id == 호출한 에이전트` 를 추가로 확인해야 합니다. `queuecall.groupcall_id` 에 저장되는 값은 바깥 에이전트 groupcall(`createCallsOutgoingGroupcall` 이 `RingMethodRingAll` 로 고정해 만든 `groupcalls[0]`, `outgoing_call.go:547`, `execute.go:90`)이고, 에이전트의 링 방식(`ring_all`/`linear`)은 안쪽 groupcall 에서만 적용됩니다(`groupcallhandler/dial.go:255~258`). 링 방식과 무관하게 `master_call_id` 는 전달되므로 키로 유효합니다.
- `service_agent_id` 는 통화 종료 경로에서 지워지지 않고 예약만 해제됩니다(`bin-queue-manager/pkg/queuecallhandler/db.go:270, 358, 550`). 지우는 곳은 `connecting -> waiting` 롤백(`dbhandler/queuecall.go:698`, `groupcall_id` 도 함께 지움)뿐입니다. 다만 `connecting` 상태에서는 에이전트가 아직 응답하지 않았을 수 있으므로 응답에 `status` 를 포함하고, 서버는 `service_agent_id` 일치만으로 노출합니다(소유한 통화의 큐 이름이라 민감하지 않음).

### 1.4 확인하지 못한 것

- 각 테이블의 실제 규모: 이 문서는 코드와 마이그레이션만 근거로 하며, 구현 단계의 락/소요 시간 분석에서 측정 근거를 붙여야 합니다.
- confbridge `connect` 타입에서 3번째 레그가 합류했을 때의 자동 종료, 링/응답 처리, 녹음 유지 동작(C3 스파이크 항목).
- 에이전트가 webphone 으로 직접 건 발신 통화가 항상 confbridge 안에 들어가는지(C1 의 전제 조건이기도 함). registrar 경로가 `connect` 를 만드는 것까지만 확인했습니다(`start_incoming_domain_type_registrar.go`).

### 1.5 개정 1: C2 기존 결함 확인과 전제 변경 영향 (코드로 확인)

기준 커밋은 개정 시점의 origin/main(`242fa584e`)입니다. 아래는 모두 코드 읽기로 판정했고 실행 재현은 하지 않았습니다(Asterisk 필요).

제품 전제의 추가: square-talk 사용자는 회사 또는 그룹의 소속원이며 상식적인 조직 규율을 이해한다고 봅니다. 이 전제는 에이전트의 규율 준수에 대한 신뢰만 뜻합니다. 테넌트 격리, 통화 소유자 검증, 고객 일치 검증, 외부 PSTN 비용 안전(잔액, 발신 검증)은 유지합니다.

| 항목 | 확인한 사실 | 판정 |
|---|---|---|
| 고지 (P1) | `callhandler/recording.go:16-76`, `recordinghandler/recording.go:18-110` 에 고지/동의 로직 없음. `recording_overview.rst:446-452` 는 플로우 작성자 대상 권고. 에이전트 API 경로에서 서버가 고지하지 않는다는 문장이 RST 에 없음 | 서버 강제 없음으로 결정, RST 한 줄 추가 |
| 주소 제한 (P2/P3) | 에이전트는 자기 `tel`/`sip` 주소를 바꿀 수 있음(`me_addresses.yaml`). 그러나 외부 발신은 `v1_calls.go:197` -> `outgoing_call.go:125` `CreateCallOutgoing` 을 거치며 `validate.go:121-151`(고객 활성, `tel` 은 본인확인)과 `outgoing_call.go:195`(잔액)가 걸림. `groupcallhandler/dial.go:172-173` 은 `tel`/`sip` 를 그대로 확장(가드는 `CreateCallOutgoing`). attended 의 발신자 번호는 transferer 통화의 Source(`transferhandler/attended.go:137`) | 제한은 보안 경계가 아니라 단계적 출시 선택 |
| A: Answer 후 상태 가드 | `callhandler/recording.go:46-51` 이 `progressing` 이 아니면 `channelHandler.Answer` 후 곧바로 `recordinghandler.Start` 호출. `recordinghandler/recording.go:39,54-56` 은 `CallV1CallGet` 으로 읽은 상태가 `progressing` 이 아니면 "invalid status". 상태 전이는 ARI `ChannelStateChange` 로 비동기(`docs/plans/2026-09-15-voip-1525-recording-start-auto-answer-design.md`). 그 설계는 `callhandler` 의 가드만 풀고 `recordinghandler` 의 가드(2023 년부터 존재)를 남김 | 유효(타이밍 의존). ringing 통화의 start 는 응답만 되고 녹음 실패. 닿는 곳은 flow 액션과 admin `recording_start`, 에이전트 경로는 비해당(P8) |
| B: 자연 종료 시 `call.RecordingID` | `UpdateRecordingID` 호출처는 `callhandler/recording.go:70`(set), `:104`(clear), `listenhandler/v1_calls.go:667`(PUT) 뿐. `arieventhandler/ari_recording.go:85` -> `recordinghandler/stop.go:17-42` `Stopped` 는 녹음 상태만 `ended` 로 바꾸고 call 을 갱신하지 않음. `call.recording_id` 는 외부 응답에 노출(`models/call/webhook.go:28`). 종료된 통화의 stale 값은 영향 없음(snoop 채널이 사라지고 Finished 가 와서 녹음은 정상 종료) | 유효. 자연 종료 뒤 start 가 "already progressing" 으로 막히고 REC 표시 기준이 부정확. confbridge 도 같은 결함(`confbridgehandler/recording.go:61,100`) |
| C: 상태 덮어쓰기 | `stop.go:138` 의 `RecordingSetStatus(Stopping)` 은 현재 상태를 보지 않는 UPDATE(`dbhandler/recording.go:321-325`). (i) B 의 stale id 로 stop 호출 시 `ended` -> `stopping` 퇴행(Finished 는 이미 지나 영구 `stopping`). (ii) 정상 stop 에서도 hangup(`stop.go:162`)한 채널의 Finished 가 138행보다 먼저 처리되면 같은 퇴행 | 유효. RST 의 "상태는 앞으로만 간다" 위반 |
| D: `(nil, nil)` | `callhandler/recording.go:91-92` 는 `err` 가 nil 인 채 `errors.Wrapf(err, ...)` 호출 -> `(nil, nil)` | 유효 |
| 동시 start 경합 | `callhandler/recording.go:36`(검사)과 `:70`(설정)이 원자적이지 않고 `CallSetRecordingID` 는 무조건 덮어씀(`callhandler/db.go:340` 호출부, 덮어쓰는 쪽은 `dbhandler/call.go:457`) | 알려진 한계(3.10) |
| Stop 의 hangup 실패 | `recordinghandler/stop.go:159-165` 가 snoop 채널 hangup 실패를 로그만 남기고 성공 처리, 이후 `callhandler/recording.go:104` 가 id 를 비움 | 알려진 한계(3.10) |
| 플로우 `recording_start` 충돌 | 에이전트가 먼저 켠 레그에서 나중에 실행되면 `callhandler/recording.go:36-38` 오류. `action.go:147-148` 은 액션 오류 시 다음 액션으로 넘어감, 끊김은 `action.go:203-206`(다음 액션 요청 자체 실패) | 개정 전의 "통화가 끊길 수 있음" 추정은 틀림. 액션이 건너뛰어지는 위험(3.10) |

`ari_recording.go` 는 `_in` 이름의 녹음만 처리하고 `_out` 은 무시(`ari_recording.go:21,62`)하므로 두 snoop 채널을 hangup 해도 `Stopped` 는 한 번만 실행됩니다.

## 2. 디자인 (결정 요약과 선택지 비교)

### 2.1 결정 요약

| 항목 | 권고 | 소유 서비스 | 스키마 변경 | 규모 |
|---|---|---|---|---|
| A1 연락처 Notes | 연락처 단위 케이스 노트 집계 엔드포인트 신설 (클라이언트 N+1 불가) | bin-contact-manager, bin-api-manager, bin-openapi-manager | 없음 | S |
| A2 통화 중 Insight | `GET /service_agents/contact_cases` 에 `reference_id` 필터 노출 (contact-manager 무변경) | bin-api-manager, bin-openapi-manager | 없음 | XS |
| A3 Call notes | 신설하지 않음. 케이스 노트 재사용 | - | 없음 | 0 |
| A4 Disposition | close 요청에 선택 필드 `disposition` 추가 + `contact_cases.disposition` 컬럼 (태그 활용안, `closed_reason` 값 확장안 기각) | bin-contact-manager, bin-api-manager, bin-openapi-manager | contact-manager 다음 번호 (컬럼 1개) | S |
| A5 큐 이름 | `GET /service_agents/calls/{id}/queuecall` (소유한 통화에서 서버가 큐를 풀어 좁은 응답만 반환) | bin-api-manager, bin-openapi-manager | 없음 | S |
| B1 Pinned | 에이전트별 pin 테이블 + `service_agents/talk_pins` | bin-talk-manager, bin-api-manager, bin-openapi-manager | talk-manager 다음 번호 (테이블 1개) | M |
| B2 Starred | 에이전트별 star 테이블 + 연락처 별표 API + 목록 `starred` 필터 | bin-contact-manager, bin-api-manager, bin-openapi-manager | contact-manager 다음 번호 (테이블 1개) | M |
| B3 Mentions | 저장 시 추출(서버 파싱) + 멘션 테이블 + `service_agents/talk_mentions` | bin-talk-manager, bin-api-manager, bin-openapi-manager | talk-manager 다음 번호 (테이블 1개) | M |
| C1 Transfer | `POST /service_agents/transfers` (소유자, 상태, confbridge 검증은 api-manager. 주소는 P2 (a) 권고에 따라 `extension` 만) | bin-api-manager, bin-openapi-manager | 없음 | S~M (약 3일, (b) 는 4일) |
| C2 Record | `POST /service_agents/calls/{id}/recording_start`, `recording_stop`. 고지/중지 정책은 결정됨(P1, P7: 서버 강제 없음, 감사 로그만). call-manager 기존 결함 B, C, D 수정 포함 | bin-api-manager, bin-openapi-manager, bin-call-manager(기존 결함 수정, 3.10) | 없음 | M (약 5일) |
| C3 Add | 설계안 Y: call-manager 에 "참가자 추가" RPC (기존 confbridge 재사용). 스파이크 선행 | bin-call-manager, bin-api-manager, bin-openapi-manager | 없음 | L |
| C4 SLA, C5 우선순위 | 제품 정의 초안과 질문 목록. 데이터 모델 초안은 정의 확정 뒤 구현 | bin-contact-manager (케이스 필드) | 정의 확정 뒤 | 정의 필요 |

마이그레이션 번호는 서비스마다 현재 `000001` 만 있고, 같은 서비스에 들어가는 변경(contact-manager 의 A4, B2 / talk-manager 의 B1, B3)은 먼저 머지되는 쪽이 `000002`, 다음이 `000003` 을 씁니다. 번호는 파일명 규약상 구현 PR 에서 확정하며 머지 전 `git fetch origin main` 으로 충돌을 확인합니다. 같은 서비스의 두 PR 이 같은 번호를 쓰면 뒤 PR 이 번호를 바꿉니다(`scripts/check-service-migrations.sh` 가 검출).

규모는 XS < 1일, S 1~3일, M 3~5일, L 8일 이상(리뷰 루프, 테스트, RST 문서 갱신 포함 추정). 근거는 3절의 항목별 비용과 5절의 표입니다.

### 2.2 선택지 비교

#### A1 연락처 Notes 탭

| 안 | 호출 수 | 페이지네이션 | 판단 |
|---|---|---|---|
| (a) 백엔드 무변경, 클라이언트 N+1 | 케이스 목록 1회(상한 100) + 케이스마다 노트 목록 1회, 케이스가 100개를 넘으면 목록 페이지 추가 | 노트 목록은 무페이지 전체 반환이라 연락처 전체의 시간순 합산 페이지를 만들려면 모든 케이스의 모든 노트를 받아 합쳐야 함 | 기각 |
| (b) 연락처 단위 집계 엔드포인트 (권고) | 1회 | `tm_create` 토큰 페이지네이션(기존 목록 규약) | 채택 |
| (c) 연락처 단위 독립 노트 신설 | 1회 | 가능 | 기각: 케이스 노트와 이중 저장, 모델 신설, 요구 근거 없음 |

#### A4 Disposition

| 안 | 내용 | 판단 |
|---|---|---|
| (1) close 확장 + `disposition` 컬럼 (권고) | close 에 선택 body `{disposition}` (고정 enum, 생략 가능). 컬럼 1개 추가: `ALTER TABLE contact_cases ADD COLUMN disposition varchar(32) NULL, ALGORITHM=INPLACE, LOCK=NONE` (두 엔진 공통 문법, 선례 `bin-ai-manager/migrations/000002_...up.sql`; `ALGORITHM=INSTANT` 와 `IF NOT EXISTS` 는 엔진 전용이라 쓰지 않음). `contact_cases` 는 STORED 생성 컬럼 5개와 UNIQUE 키를 가진 테이블이라 INPLACE/NONE 이 두 엔진에서 거부되지 않는지를 구현 PR 이 scratch MariaDB 12.3 과 MySQL 8.0 에서 upgrade, downgrade, upgrade 로 실측해야 하며(이 설계에서는 실행하지 않음), 거부되면 마이그레이션이 실패하므로 그 경우 설계를 다시 엽니다. `closed_reason` 은 시스템 사유(`agent_closed`, 예약 `merged`)로 유지 | 채택. 단일 값, 종료와 원자적 기록, 목록/통계 조회 가능, body 생략 시 현행과 동일 |
| (2) `closed_reason` 값 확장 | 컬럼 추가 없이 `closed_reason` 에 `resolved` 등을 추가 | 기각. 현재 `closed_reason` 은 "누가/왜 시스템적으로 닫았는가"(에이전트 종료, 병합)이고, 처분 결과와 의미가 섞여 병합 같은 시스템 종료와 구분이 흐려짐. `varchar(32)` 라 스키마 변경은 불필요하다는 장점만 있음 |
| (3) 태그 활용 | 케이스 태그(`TagIDs`)에 disposition 태그를 붙임 | 기각. 태그는 contact-manager 내부에만 구현돼 있고(`CaseTagAdd/Remove/List`) 외부 API 가 없어 api-manager/OpenAPI 에 엔드포인트 3개를 새로 노출해야 하며, 고객이 태그를 미리 만들어야 하고, 복수 선택이 가능해 "처분은 하나"를 강제할 수 없고, close 와 원자적이지 않음. 스키마 변경 회피 이점이 노출 비용으로 상쇄됨 |

#### B3 멘션 추출 위치

| 안 | 내용 | 판단 |
|---|---|---|
| (a) 저장 시 서버 추출 + 별도 테이블 (권고) | 메시지 생성 시 `talk-manager` 가 본문의 에이전트 멘션 `[@표시이름](mention:<agent-id>)` 만 추출(연락처 멘션 `mention:contact:<id>` 는 제외, 소비자가 없으므로 저장하지 않음)해 `(message, mentioned agent)` 행으로 기록. 메시지 행과 멘션 행은 **하나의 DB 트랜잭션**으로 저장(트랜잭션 헬퍼가 없으므로 `dbhandler` 에 `MessageCreateWithMentions` 한 개를 신설, 선례: `bin-ai-manager` 의 `AIAcceptProposal`). 실패하면 메시지 생성도 실패로 처리해 "메시지는 있는데 멘션이 없는" 상태를 만들지 않음. 조회는 수신자 기준 인덱스 | 채택. 메시지는 수정 API 가 없고 삭제는 soft delete 라 조회 시 메시지 삭제 여부와 채팅 접근 권한을 조인/재확인하면 정합 유지 |
| (b) 조회 시 본문 파싱 | `text` 를 LIKE/정규식으로 훑음 | 기각. `text` 인덱스 없음, 채팅 단위 목록 구조(1.2.2)에서 가로질러 훑으려면 접근 가능한 모든 채팅의 전체 메시지 스캔 |
| (c) `metadata` JSON 에 멘션 배열 저장 | 메시지 행의 `metadata` 에 보관 | 기각. 두 엔진(MariaDB, MySQL 8.0)에서 공통으로 쓸 수 있는 JSON 배열 인덱스가 없어 수신자 기준 조회가 전체 스캔이 됨 |

#### C3 Add (3자 통화)

| 안 | 내용 | 비용/위험 | 판단 |
|---|---|---|---|
| X. attended transfer 변형 | `POST /transfers`(attended) 로 제3자를 합류시키고 별도 해제 호출로 MOH, mute 를 풀어 줌 | 합류까지는 기존 구현. 그러나 `transferer` 가 끊을 때의 분기가 transfer 의미(완료)에 묶여 있고(`transferer.go`), 해제는 `calls/{id}/moh`, `mute` 조합을 클라이언트가 직접 맞춰야 하며 transfer 레코드가 의미 없이 남음 | 기각 |
| Y. call-manager "참가자 추가" (권고) | 진행 중 통화의 confbridge 에 제3자 발신 레그를 만들어 합류시킴(`confbridge_join` 하나짜리 flow + 발신). 기존 레그는 건드리지 않음 | `attendedExecute` 와 같은 구성에서 `attendedBlock` 과 transfer 레코드를 뺀 형태. L(스파이크 포함) | 채택, 스파이크 선행 |
| Z. webphone 에서 SIP 로 직접 | 두 번째 INVITE, 로컬 믹싱 | webphone 은 모듈 전역 단일 세션(`webphone.js`)이고 두 번째 INVITE 는 486 으로 거절. 서버 측 녹음, 과금, 이벤트와 불일치 | 기각 |

### 2.3 제품/정책 결정이 필요한 사항 (구현 착수 전 확정; P1, P7 은 개정 1 에서 결정됨)

| # | 사항 | 기본값 제안 | 근거 |
|---|---|---|---|
| P1 | C2 에이전트 수동 녹음의 고지/동의 | **결정됨(개정 1): 고지는 조직 규율이며 서버 강제 없음.** 에이전트가 소유한 통화에 한해 허용하고, 시작 때마다의 동의 팝업이나 서버 강제 고지(녹음 시작 전 고객 레그에 안내 재생)를 두지 않음. 클라이언트의 통화 중 REC 표시는 UX 권고(의무 아님). 구현 범위에 `recording_overview.rst` 에 한 줄 명시를 포함(3.10) | 에이전트는 회사/그룹 소속원으로 상식적 규율을 이해한다는 제품 전제. 서버에는 고지를 강제하는 코드가 없고(`callhandler/recording.go`, `recordinghandler/recording.go`) 현행 flow `recording_start` 액션도 동일. 플로우 대상의 법적 권고(`recording_overview.rst` 446~452행)는 유지 |
| P2 | C1 transferee 주소 유형 v1 | 선택 필요. (a) v1 은 `extension` 만(권고): `agent`, `tel`, `sip` 는 400. (b) `tel` 까지 허용: `extension` 과 `tel`(E.164 검증)을 받고 `agent` 는 필요 시 포함. 이 제한은 보안 경계가 아니라 단계적 출시 선택입니다 | 에이전트는 자기 `tel`/`sip` 주소를 바꿀 수 있고(`bin-openapi-manager/openapi/paths/service_agents/me_addresses.yaml`) call-manager 는 `agent` 대상의 `Addresses` 를 그대로 발신 대상으로 확장하지만(`groupcallhandler/dial.go`), 외부 발신은 어떤 경로든 `CreateCallOutgoing`(`callhandler/outgoing_call.go`)의 고객 상태, 본인확인(`validateOutgoingCallPermission`), 잔액(`ValidateCustomerBalance`) 검사를 거치므로 이 제한 없이도 비용 안전의 건당 가드는 유지됩니다. 따라서 (a) 를 권고하는 이유는 가장 단순하기 때문입니다: `agent` 유형이 없으면 에이전트 조회, `Addresses` 검사, TOCTOU 처리, C3 의 "extension 정확히 1개" 제한이 모두 불필요합니다(3.9, 3.11). 단 square-talk 가 동료 에이전트를 고르려면 클라이언트가 `GET /service_agents/agents` 의 `addresses` 에서 extension 을 찾아야 합니다. (b) 는 `tel` 처리(형식 검증)와 PSTN 비용 누적 검토가 추가됩니다. 구현 검증은 어느 쪽이든 실제 PSTN 발신 없이 mock 과 내부 extension 으로만 함 |
| P3 | C3 제3자 주소 유형 v1 | P2 와 같은 선택. (a) `extension` 만(권고), (b) `tel` 포함 | (b) 에서 `tel` 을 허용하면 외부 PSTN 비용이 3자 통화 시간 동안 누적됩니다. 발신 가드는 건당 검사라 통화 시간은 제한하지 않으므로 별도 위험으로 유지하며, (b) 를 고르면 통화 시간 상한이나 경고를 함께 설계해야 합니다 |
| P4 | A4 disposition 값 집합 | `resolved`, `unresolved`, `follow_up_needed`, `spam`, `other` | 고객별 사용자 정의 목록은 별도 모델이 필요해 v1 에서 제외(트리거: 고객이 목록 자체를 바꾸고 싶다는 요구 발생) |
| P5 | B3 기존 메시지 멘션 소급 | 소급하지 않음. 배포 이후 메시지부터 | 소급은 전체 `talk_messages.text` 스캔이 필요. 필요하면 일회성 도구로 별도 결정 |
| P6 | C4, C5 | 3.12절의 정의 질문 목록에 답이 있기 전에는 구현하지 않음 | 필드도 정의도 없음 |
| P7 | C2 에이전트의 녹음 중지 | **결정됨(개정 1): 통화를 소유한 에이전트의 start/stop 모두 허용, 감사 로그(에이전트 id, 통화 id, 동작)만 남김.** 시작 주체 표식과 "중지는 제공하지 않음" 선택지는 두지 않음 | 에이전트의 상식적 규율 준수를 신뢰한다는 제품 전제. 현행 플로우 녹음에도 중지 권한 구분이 없음 |
| P8 | call-manager 의 기존 결함 A: `RecordingStart` 가 `progressing` 이 아닌 통화를 Answer 한 뒤 곧바로 녹음을 시작하나 `recordinghandler/recording.go:54-56` 의 상태 가드가 비동기 상태 전이 전의 값을 읽어 "invalid status" 로 실패(응답만 되고 녹음 실패. 1.5) | 선택 필요. (A-1) `callhandler` 의 Answer 분기를 없애고 `progressing` 이 아니면 거부: 가장 단순하나 울리는 통화에서 플로우 `recording_start` 가 거부되어 이미 병합된 자동 응답 시도의 의도가 되돌려짐. (A-2) Answer 분기를 두고 `recordinghandler` 의 가드를 `ringing` 도 허용하도록 완화하거나 `progressing` 대기: 의도를 지키나 응답 전 snoop 채널이 동작하는지 sandbox 확인이 필요(약 +1.5일). 에이전트 경로는 api-manager 가 `progressing` 을 강제하므로 어느 쪽이든 C-2 의 필수 범위가 아님 | A 는 flow 액션과 admin 의 `recording_start` 에서만 닿는 기존 결함이며 이 설계가 만든 것이 아님. 권고: 별도 판단. (A-1) 약 +0.5일 |

### 2.4 구현 체인 분할과 의존성

원칙: 서비스 하나의 변경과 그 변경을 노출하는 api-manager/OpenAPI 변경은 한 작업이므로 한 PR 입니다(이 모노레포는 OpenAPI 생성 코드와 소비 서비스가 같은 저장소). PR 은 항목 단위이며 아래 그룹은 체인 분할 단위입니다.

| 그룹 | 항목 | 선행 의존 |
|---|---|---|
| A (소형, 독립) | A1, A2, A4, A5 | 서로 독립. 병렬 가능. A3 는 변경 없음 |
| B (사용자별 상태) | B1, B2, B3 | 서로 독립. 마이그레이션은 서비스별(talk-manager 에 B1, B3 가 함께 들어가므로 번호는 먼저 머지되는 쪽이 `000002`, 다음이 `000003`, 3.6절과 3.8절). 그룹 A 와 독립 |
| C (통화 제어) | C1, C2, C3 | C1, C2 는 독립(C1 은 스파이크 S7 의 extension 주소 필드 확정에 의존. C2 는 call-manager 의 기존 결함 수정 B, C, D 를 같은 PR 에 포함하므로 call-manager 를 먼저 배포할 필요 없이 한 PR 로 나감). C3 는 C1 과 같은 소유자 검증 헬퍼를 쓰므로 C1 뒤(순서 고정). C3 는 스파이크(3.11절) 결과에 의존 |
| D (제품 정의 필요) | C4, C5 | 제품 정의 확정 뒤 별도 설계 |

스키마 롤링 호환: 서비스는 시작 시 리스닝 전에 자기 마이그레이션을 적용하므로 신 스키마와 구버전 pod 가 공존하는 구간이 있습니다. A4 는 nullable 컬럼 추가라 구 코드가 영향받지 않는지(INSERT/SELECT 가 컬럼 목록을 명시하는지)를 구현 PR 이 확인하고, B1~B3 는 신규 테이블이라 구 코드는 참조하지 않습니다. 항목별 전체 분석은 4절입니다.

롤링 배포(서비스별로 독립 배포): api-manager 가 새 선택 필드를 먼저 보내고 contact-manager 가 아직 구버전이면 그 필드는 무시됩니다(A4). 대응: (1) 응답이 처분을 되돌려주므로 클라이언트가 저장된 `disposition` 을 표시하고, (2) PR 설명에 배포 순서(contact-manager 먼저)를 적습니다. 반대 순서(새 contact-manager, 구 api-manager)는 필드를 받지 않을 뿐 안전합니다.

일반 규칙: 새 RPC 경로나 새 필터를 쓰는 항목(A1, B1~B3, C3)은 **하위 서비스를 먼저 배포**합니다. 새 api-manager 가 먼저 나가면 구 하위 서비스가 모르는 경로에 404 를 돌려주어 그 항목의 엔드포인트가 일시적으로 5xx/404 로 보이고, B2 의 별표 목록은 필터가 아니라 전용 내부 경로(3.7)를 쓰므로 구 contact-manager 는 조용히 필터를 무시하지 않고 404 를 돌려주어 실패가 드러납니다. 기존 RPC 만 쓰는 항목(A2, A5 의 기존 RPC 조합, C1, C2)은 순서 제약이 없습니다.

## 3. 설계 (항목별)

공통 규약 (모든 신규 `service_agents` 엔드포인트):

- 인증/권한: `!a.IsAgent()` 이면 `ErrAuthenticationRequired`, 그 뒤 `hasPermission(ctx, a, a.CustomerID, amagent.PermissionAll)`. Admin/Manager 엔드포인트와 그 권한 비트마스크는 변경하지 않습니다. `bin-openapi-manager/CLAUDE.md` 의 체크리스트(새 path 파일, `ServiceAgent*` 함수, 인터페이스와 mock 재생성, `server/service_agents_*.go`, 서비스 핸들러와 HTTP 테스트)를 따릅니다.
- 응답은 외부 노출 필드만(`WebhookMessage` 변환). 목록은 `page_size` 기본/상한 100, `page_token` 은 마지막 항목의 `tm_create` 입니다(기존 규약).
- 각 항목의 구현 PR 은 RST 문서(`bin-api-manager/docsdev/source/`)와 clean 빌드한 `build/` 를 함께 갱신하고, 해당 서비스의 `docs/*.md`(라우팅 표, 도메인, 설정)를 같은 커밋에서 갱신합니다.
- 테스트 verify 항목은 이 문서의 계획이며 테스트 코드는 구현 단계에서 씁니다.
- 비용 호출 금지: 모든 verify 는 mock 또는 내부 번호/extension 으로만 하고 외부 PSTN 발신, 번호 구매, SMS/Email 발송은 하지 않습니다.

### 3.1 A1 연락처 Notes 탭: 연락처 단위 케이스 노트 집계

- 소유: bin-contact-manager(조회), bin-api-manager, bin-openapi-manager.
- API (신규 `openapi/paths/service_agents/contacts_id_case_notes.yaml`):

```yaml
get:
  summary: Get list of case notes of a contact
  description: Notes written on the cases attributed to this contact, newest first, across all of its cases.
  tags: [Service Agent]
  parameters:
    - name: id            # path, uuid. "The contact ID returned from GET /service_agents/contacts."
    - $ref: '#/components/parameters/PageSize'
    - $ref: '#/components/parameters/PageToken'
  responses:
    '200': { allOf: [CommonPagination, { result: [ContactManagerCaseNote] }] }   # case_id 가 이미 스키마에 있음
    '401', '404', '500'
```

- 내부 RPC: contact-manager `GET /v1/contacts/{id}/case_notes` (body 에 `customer_id`), 핸들러 `CaseNoteListByContact(ctx, customerID, contactID, size, token)`.
- 쿼리: `contact_case_notes n JOIN contact_cases c ON c.id = n.case_id AND c.customer_id = n.customer_id WHERE n.customer_id = ? AND c.contact_id = ? AND n.tm_delete IS NULL AND n.tm_create < ? ORDER BY n.tm_create DESC LIMIT size+1`. 노트 측은 기존 `idx_contact_case_notes_case_id (case_id, tm_delete)`, 케이스 측은 `customer_id` 접두 인덱스를 씁니다. `contact_id` 만으로 좁히는 인덱스는 없으므로(`idx_case_unresolved` 는 `customer_id, status, contact_id` 순) 고객의 케이스 수만큼 스캔합니다. 기존 `GET /service_agents/contact_cases?contact_id=` 가 이미 같은 조건이므로 새 부담이 아니며, 인덱스 `(customer_id, contact_id)` 추가는 측정에서 느린 쿼리가 확인될 때의 트리거로만 둡니다.
- 권한: PermissionAll. 연락처가 호출자 고객의 것인지 확인(다른 고객이면 404, 존재 누설 금지)한 뒤 조회합니다. 노트 열람 범위는 기존 `ServiceAgentCaseNoteList`(같은 고객의 모든 에이전트)와 같습니다.
- 데이터 모델: 변경 없음. 롤링: 새 RPC 경로이므로 contact-manager 먼저 배포.
- Verify: (1) 케이스 3개, 노트 여러 개인 연락처에서 케이스를 가로질러 `tm_create` 내림차순이고 페이지 경계(size+1 방식)에서 누락/중복이 없음. (2) 다른 고객의 연락처 id 는 404이고 교차 고객 행이 섞이지 않음. (3) 삭제된 노트 제외, 케이스가 없으면 빈 배열. (4) 케이스의 `contact_id` 재귀속 후에는 현재 귀속 기준. (5) 비에이전트 401, `page_size` 상한 100. (6) 케이스 노트 `case_id` 가 응답에 포함됨.
- 비용: S (약 2일).

### 3.2 A2 통화 중 Insight 노출: 케이스 조회에 `reference_id` 필터 노출

- 소유: bin-api-manager, bin-openapi-manager. contact-manager 무변경(`v1_cases.go:68` 이 이미 `reference_id` 처리, 인덱스 `idx_case_customer_reference_id` 존재, admin OpenAPI `contact_cases/main.yaml` 는 이미 노출).
- API: `GET /service_agents/contact_cases` 에 선택 쿼리 `reference_id` (string, 설명은 admin 문서와 동일: "the internal VoIPBin resource ID `reference_type` points at, e.g. a call ID"). `ServiceAgentCaseList` 의 시그니처에 `referenceID string` 을 추가하고 `ContactV1CaseList` 마지막 인자로 전달합니다(현재 `""` 고정). 노출 범위는 기존 `contact_id` 필터와 같은 테넌트 스코프(같은 고객의 케이스, 열람 제한 없음)라 권한이 넓어지지 않습니다. `ServiceAgentCaseList` 의 함수 주석("reference_id filters remain unsupported")도 갱신합니다. 이 함수에는 `!a.IsAgent()` 가드가 없고(기존 케이스 핸들러 전체가 `hasPermission(PermissionAll)` 만 사용) 이 PR 은 기존 패턴을 유지하며 가드를 새로 넣지 않습니다.
- 클라이언트 사용 순서(규범 아님): 서버 통화(`useLiveCallMapping` 의 `mappedCall`, 하나로 좁혀진 경우만)의 `id` 와 비어 있지 않은 `master_call_id` 로 최대 2회 조회 -> `reference_type=call` 이고 `status=open` 인 케이스가 있으면 기존 `POST /service_agents/aicalls`(`contact_case`, `assistance_id` 생략) -> `POST /service_agents/aicalls/{id}/listen`. 결과가 비는 것은 정상 경로이며(1.2.1) 그때 Insight 를 노출하지 않는 것이 제품 전제와 일치합니다. 서버는 새 상태를 노출하지 않으며 listen 응답은 기존처럼 청취 상태 필드가 없습니다. 3자 이상 통화(C3)에서는 listen 이 시작되지 않습니다(1.2.1).
- 데이터 모델 변경 없음. 롤링: 순서 제약 없음(contact-manager 는 이미 지원).
- Verify: (1) `reference_id` 지정 시 해당 값의 케이스만 반환, 미지정 시 기존 동작과 동일. (2) `contact_id` 와 함께 주면 AND. (3) 다른 고객의 케이스는 반환되지 않음. (4) 어떤 케이스에도 묶이지 않은 id 는 빈 배열.
- 비용: XS (약 1일, OpenAPI 재생성과 mock 재생성 포함).

### 3.3 A3 Call notes

- 결정: 통화 단위 노트를 신설하지 않습니다. 통화가 케이스에 묶인 경우(A2 조회가 성공) 케이스 노트(`POST /service_agents/contact_cases/{id}/notes`)를 그대로 씁니다. 케이스에 묶이지 않은 통화에서는 노트 대상이 없으며, 클라이언트는 노트 입력을 비활성화합니다.
- 근거: 통화 단위 노트의 유일한 추가 가치는 "케이스가 없는 통화"에 대한 기록인데, 그 통화가 CRM 대상이 아니거나(`isCRMEligiblePeer`) 플로우가 케이스를 만들지 않도록 구성된 경우이므로 요구가 실사용 신호로 확인되지 않았습니다.
- 트리거(그때 설계): 케이스가 없는 통화에서 노트가 필요하다는 요구가 실제로 반복해서 나타날 때.
- 비용: 0.

### 3.4 A4 Disposition

- 소유: bin-contact-manager, bin-api-manager, bin-openapi-manager. 선택 근거는 2.2.
- API: `POST /service_agents/contact_cases/{id}/close` 에 선택 요청 본문.

```yaml
requestBody:
  required: false
  content:
    application/json:
      schema:
        type: object
        properties:
          disposition:
            type: string
            enum: [resolved, unresolved, follow_up_needed, spam, other]
            description: The outcome the agent records when closing the case. Omit to close without recording one.
```

  `ContactManagerCase` 에 `disposition` 을 `type: string` 으로 추가하고 값 목록은 설명에 적습니다(`closed_reason` 과 같은 방식: 열린 케이스와 처분 없이 닫힌 케이스는 빈 문자열이라 enum 으로 선언하면 계약이 어긋남). enum 은 요청 본문에만 둡니다. 본문이 없거나 `{}` 이면 현행과 동일하게 닫힙니다. enum 밖의 값은 400.
- 내부: `ServiceAgentCaseClose(ctx, a, id, disposition)` -> `ContactV1CaseClose(..., disposition)` -> `V1DataCasesIDClose` 에 `Disposition string json:"disposition,omitempty"` 추가 -> `casehandler.Close(..., disposition)` 가 enum 을 다시 검증하고(`InvalidArgument`) `CaseUpdateStatusClosed(..., disposition)` 가 `status='open'` 조건의 같은 UPDATE 에서 `disposition` 을 함께 씁니다. 이미 닫힌 케이스(`AlreadyClosed`)는 기존 값을 유지하고 응답이 실제 저장된 값을 돌려줍니다. `kase.Case` 에 `Disposition string json:"disposition" db:"disposition"` 추가(`closed_reason` 과 같은 nullable 문자열 처리 방식).
- 데이터 모델: contact-manager 마이그레이션 `ALTER TABLE contact_cases ADD COLUMN disposition varchar(32) NULL, ALGORITHM=INPLACE, LOCK=NONE;` (파일당 DDL 한 문장, 컬럼은 테이블 끝에 추가, down 은 `ALTER TABLE contact_cases DROP COLUMN disposition, ALGORITHM=INPLACE, LOCK=NONE;` 에 `-- DOWN-LOSSY: drops the recorded dispositions` 표기, 선례 `bin-ai-manager/migrations/000002_*.down.sql`). sqlite 테스트 스키마 `scripts/database_scripts_test/contacts.sql` 을 수동 갱신.
- Admin 의 `POST /contact_cases/{id}/close` 는 변경하지 않습니다(스키마 `ContactManagerCase` 는 공유라 응답에 `disposition` 이 보임). 트리거: admin 화면에서 입력이 필요해질 때.
- 이벤트: 현재 `casehandler.Close` 는 이벤트를 발행하지 않고(`lifecycle.go`, `kase/event.go` 에 종료 이벤트 타입이 없음) 이 설계도 새 이벤트를 만들지 않습니다.
- 롤링: 4절. 배포 순서는 contact-manager 먼저(2.4).
- Verify: (1) 각 enum 값으로 닫으면 저장되고 응답에 반영됨. (2) 본문 없음/`{}` 는 `disposition` 비어 있음, `closed_reason=agent_closed` 유지. (3) enum 밖의 값은 400이고 상태 변경 없음. (4) 두 번째 close 는 `AlreadyClosed` 이고 처음 값이 유지됨. (5) 동시 close 경합에서 한 쪽만 기록. (6) scratch MariaDB 12.3 과 MySQL 8.0 에서 upgrade, downgrade, upgrade 및 기존 데이터가 있는 DB 에서 적용(INPLACE/NONE 이 생성 컬럼 테이블에서 거부되지 않는지 포함). (7) 구 바이너리 호환: 기존 쿼리의 컬럼 목록이 명시적이어서 새 컬럼이 읽기에 영향 없음.
- 비용: S (약 3일).

### 3.5 A5 큐 이름 조회

- 소유: bin-api-manager, bin-openapi-manager. queue-manager 무변경(기존 RPC `QueueV1QueuecallGetByReferenceID`, `QueueV1QueueGet` 사용).
- API (신규 `openapi/paths/service_agents/calls_id_queuecall.yaml`):

```yaml
get:
  summary: Get the queue of a call
  description: Returns the queue this call was routed through, for a call the agent owns.
  tags: [Service Agent]
  parameters: [ { name: id, in: path, schema: { type: string, format: uuid }, description: "The call ID returned from GET /service_agents/calls." } ]
  responses:
    '200': { schema: QueueManagerQueuecallSummary }   # { id (queuecall), queue_id, queue_name, status, tm_create }
    '401', '403', '404', '500'
```

- 서버 절차: (1) `IsAgent`, PermissionAll. (2) `callGet(id)` 후 `c.CustomerID == a.CustomerID` 와 `c.OwnerID == a.AgentID()` 확인, 아니면 403(소유자가 비어 있어도 거부: fail-closed, 1.3). (3) 후보 키 `[c.MasterCallID(비어 있지 않으면), c.ID]` 순으로 `QueueV1QueuecallGetByReferenceID`, NotFound 면 다음 후보(에이전트가 직접 큐에 들어간 통화는 `c.ID` 가 reference). (4) 찾은 queuecall 이 `CustomerID == a.CustomerID` 이고 `ServiceAgentID == a.AgentID()` 인지 확인, 아니면 404(존재 누설 금지). (5) `QueueV1QueueGet(qc.QueueID)` 로 이름을 읽고 고객 일치를 확인(삭제된 큐도 이름은 반환). 응답은 queuecall 의 `id`, `queue_id`, `queue_name`, `status`, `tm_create` 만 담습니다(source, tag_ids, timeout, reference 관련 필드와 groupcall id 는 제외).
- 왜 이 형태인가: webphone `currentCall.id` 는 서버 id 가 아니라서 키가 될 수 없고(1.2), 소유자 검증이 끝난 서버 통화에서 출발하면 기존 `ServiceAgentCall*` 패턴을 그대로 재사용합니다. `queuecall.WebhookMessage` 에 `queue_id` 를 추가하거나 `/queues` 를 에이전트에게 열면 Admin 표면을 건드리거나 필요 이상으로 노출합니다. 큐 목록 페이지가 필요해지는 시점에 `service_agents/queues` 를 따로 설계합니다(트리거).
- 호출 빈도: 통화 연결 시 1회(폴링 아님). RPC 3~4회/요청.
- Verify: (1) 소유자 불일치/소유자 비어 있음 403. (2) `master_call_id` 로 찾는 경우와 `c.ID` 로 찾는 경우(에이전트가 직접 큐에 건 통화) 모두. (3) 다른 에이전트의 queuecall 은 404. (4) 다른 고객의 queuecall/큐는 404. (5) `connecting` 과 `service` 상태가 `status` 로 구분되어 반환. (6) `master_call_id` 가 비어 있는 레그에서 404(fail-closed). (7) 응답에 제외 필드가 없음.
- 비용: S (약 3일).

### 3.6 B1 Pinned (채팅/메시지)

- 소유: bin-talk-manager(저장), bin-api-manager, bin-openapi-manager.
- 데이터 모델 (talk-manager 신규 마이그레이션, 파일당 DDL 하나, 베이스라인과 같은 테이블 옵션):

```sql
CREATE TABLE talk_pins (
  id binary(16) NOT NULL,
  customer_id binary(16) NOT NULL,
  owner_type varchar(255) NOT NULL,
  owner_id binary(16) NOT NULL,
  chat_id binary(16) NOT NULL,
  target_type varchar(16) NOT NULL,   -- 'chat' | 'message'
  target_id binary(16) NOT NULL,      -- chat id or message id
  tm_create datetime(6) NOT NULL DEFAULT current_timestamp(6),
  PRIMARY KEY (id),
  UNIQUE KEY uq_talk_pins_owner_target (owner_type, owner_id, target_type, target_id),
  KEY idx_talk_pins_owner_create (customer_id, owner_type, owner_id, tm_create),
  KEY idx_talk_pins_chat_id (chat_id),
  CONSTRAINT talk_pins_ibfk_1 FOREIGN KEY (chat_id) REFERENCES talk_chats (id) ON DELETE CASCADE
) DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
```

  `chat_id` 의 선두 인덱스를 명시해 두 엔진이 암묵 인덱스를 따로 만들지 않게 하고, 외래키 이름은 베이스라인과 같은 `<table>_ibfk_N` 형식으로 명시합니다(엔진 간 자동 이름 차이를 피하기 위함). 메시지는 soft delete 이므로 `target_id` 에 메시지 FK 를 걸지 않고 조회 시 삭제 여부를 조인으로 걸러냅니다. down 은 `DROP TABLE talk_pins` 에 `-- DOWN-LOSSY: drops the agents' pins`.
- API (신규 `service_agents/talk_pins*.yaml`):
  - `POST /service_agents/talk_pins` body `{target_type: chat|message, target_id}` -> 200 `TalkManagerPin`. 멱등(이미 있으면 기존 행). `target_type=chat` 은 `canAccessChat`, `message` 는 메시지 조회 후 삭제 여부와 `canAccessChat(message.chat_id)` 를 확인하고 `chat_id` 는 서버가 채웁니다. 저장은 `INSERT` 후 유일 키 중복 에러(UNIQUE/1062)면 기존 행을 재조회하는 방식입니다(선례: `bin-talk-manager/pkg/dbhandler/participant.go` 의 중복 처리. 이 서비스에는 `ON DUPLICATE`/`ON CONFLICT` 사용이 없음).
  - `DELETE /service_agents/talk_pins/{id}` -> 200 삭제된 pin. 본인 pin 이 아니면 404.
  - `GET /service_agents/talk_pins?target_type=&chat_id=&page_size&page_token` -> 본인 pin 만, `tm_create` 내림차순. 각 항목 `TalkManagerPin {id, target_type, target_id, chat_id, tm_create, message?}`(message pin 은 `TalkManagerMessage` 를 함께 반환해 클라이언트 N+1 을 막음).
- 접근 규칙 R (B1, B3 공통): pin 은 본인에게만 보입니다. 조회 시 talk-manager 쿼리가 다음을 조인으로 확인합니다: 채팅 `tm_delete IS NULL` 이고 `chat.customer_id = 호출 고객` 이며, (`chat.type = 'talk'` 이거나 호출 에이전트가 `talk_participants` 의 참가자), 메시지 pin 이면 메시지 `tm_delete IS NULL`. 이는 api-manager 의 `canAccessChat`(`serviceagent_talk.go`)보다 **의도적으로 더 엄격**합니다. `canAccessChat` 은 `talk` 타입일 때만 고객 일치를 확인하고 direct/group 은 참가자(`OwnerType == "agent"` 와 id)만 보며, `ChatGet` 이 `tm_delete` 를 거르지 않아 삭제된 채팅도 접근 허용으로 판정합니다. 규칙 R 은 모든 타입에 고객 일치와 삭제 제외를 요구하고 참가자 조건에 `owner_type = 'agent'` 를 포함합니다. 따라서 두 구현은 (a) 채팅 삭제, (b) 모든 타입의 고객 불일치(`talk` 타입도 고객이 다르면 `canAccessChat` 은 참가자 검사로 넘어가 참가자이면 true 이지만 R 은 false) 두 축에서만 다르며 그 밖에는 같은 판정이어야 합니다. 채팅에서 나가면 pin 행은 남지만 보이지 않고 다시 참여하면 다시 보입니다.
- 이벤트/웹훅: 없음(개인 상태, 클라이언트가 재조회). 개수 상한 없음(페이지네이션이 상한 역할), 트리거: 개인 pin 수 폭증이 측정될 때.
- 롤링: 신규 테이블/RPC. talk-manager 먼저.
- Verify: (1) chat/message pin 생성, 중복 생성은 같은 행. (2) 접근 불가 채팅/타인 채팅의 pin 생성 거부. (3) 삭제된 메시지 pin 은 목록에서 빠짐. (4) 참가자 제거 후 해당 direct/group 의 pin 이 보이지 않음, 재참여 시 복구. (5) 타인의 pin id 로 DELETE 는 404. (6) 페이지 경계. (7) 규칙 R 과 `canAccessChat` 의 판정 표(talk/direct/group x 참가자/비참가자 x 채팅 삭제 여부 x 고객 일치 여부)를 테스트하고 위 (a), (b) 두 축에서만 차이가 나는지 확인. (8) 마이그레이션 upgrade/downgrade/upgrade 두 엔진.
- 비용: M (약 4일).

### 3.7 B2 Starred (연락처)

- 소유: bin-contact-manager(저장), bin-api-manager, bin-openapi-manager. agent id 는 이미 contact-manager 가 케이스 `owner_id`, 노트 `author_id` 로 다루므로 새 의존이 아닙니다.
- 데이터 모델 (contact-manager 신규 마이그레이션):

```sql
CREATE TABLE contact_stars (
  id binary(16) NOT NULL,
  customer_id binary(16) NOT NULL,
  agent_id binary(16) NOT NULL,
  contact_id binary(16) NOT NULL,
  tm_create datetime(6) DEFAULT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_contact_stars_agent_contact (agent_id, contact_id),
  KEY idx_contact_stars_agent_create (customer_id, agent_id, tm_create)
) DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci ROW_FORMAT=DYNAMIC;
```

  테이블 옵션은 contact-manager 베이스라인과 같게 맞춥니다. 연락처는 soft delete(`tm_delete`)라 FK 를 두지 않고 조회에서 삭제된 연락처를 걸러냅니다. 고객 삭제 이벤트(`contacthandler.EventCustomerDeleted`)의 정리 대상에 `contact_stars` 를 추가합니다(contact-manager 는 이미 이 이벤트로 연락처를 정리하고 별표는 연락처에 종속된 데이터이므로 같은 수명). 반대로 talk-manager 는 고객 삭제 이벤트를 구독하지 않고 기존 `talk_*` 테이블에도 정리 정책이 없으므로 B1, B3 의 새 테이블은 기존 테이블과 같은 수명 정책을 따르며 별도 정리를 만들지 않습니다(의도된 일관성).
- API:
  - `PUT /service_agents/contacts/{id}/star` -> 200 `{contact_id, starred: true, tm_create}`. 멱등.
  - `DELETE /service_agents/contacts/{id}/star` -> 200 `{contact_id, starred: false}`. 멱등(별표가 없어도 200).
  - `GET /service_agents/contacts?starred=true` -> 호출 에이전트가 별표한 연락처(기존 목록과 같은 스키마). 정렬과 `page_token` 은 기존 연락처 목록과 같은 연락처 `tm_create` 내림차순입니다(별표한 시각 순이 아님, 문서화). 서버가 만드는 다음 토큰이 응답 연락처 마지막 항목의 `TMCreate` 이므로(`server/service_agents_contacts.go`) 정렬 키와 토큰 키를 같게 맞춘 것입니다.
  - `GET /service_agents/contact_stars?page_size&page_token` -> `ContactManagerContactStar[]`(`starred` 는 항상 `true`, 같은 스키마 재사용; 토큰은 별표 행의 `tm_create`). 두 경로는 다른 질문에 답합니다: 앞은 별표 탭이 한 번의 호출로 연락처 객체를 받기 위한 것이고(없으면 id 목록 뒤에 연락처 조회 N+1 이 생김), 뒤는 일반 목록 행과 상세 화면의 별표 표시를 위한 id 집합입니다. 서버가 목록 응답에 `starred` 를 덧붙이는 대안은 Admin 과 공유하는 연락처 스키마와 서비스 핸들러의 변환 경로를 바꾸게 되어 채택하지 않았습니다.
  - 연락처가 호출자 고객의 것이고 삭제되지 않았는지 확인(아니면 404). 내부 RPC: `POST/DELETE /v1/contacts/{id}/stars/{agent_id}`, 전용 경로 `GET /v1/contacts/starred`, `GET /v1/contact_stars`(요청 본문에 `customer_id`, `agent_id`, `page_size`/`page_token` 은 쿼리). 별표 목록은 이 서비스의 목록 필터 규약(`ParseFiltersFromRequestBody` + `contact.FieldStruct`)에 억지로 끼우지 않고 별도 dbhandler 함수(`ContactListStarred`)로 구현합니다: `contact_stars s JOIN contact_contacts c`, 정렬/토큰은 `c.tm_create`(기존 `ContactList` 와 같음)이며, `ApplyFields` 가 붙이는 비한정 컬럼(`customer_id`, `tm_create`, `tm_delete`)이 JOIN 에서 모호하므로 컬럼을 테이블로 한정합니다. 에이전트 id 는 호출자 인증에서 오며 contact-manager 는 값으로만 받습니다.
- 롤링: contact-manager 먼저(전용 경로라 구 contact-manager 는 404 를 돌려주어 실패가 드러남, 2.4).
- Verify: (1) 별표/해제 멱등. (2) 에이전트 A 의 별표가 B 에게 보이지 않음. (3) 다른 고객/삭제된 연락처 404. (4) `starred=true` 가 삭제된 연락처를 제외하고 토큰 키와 정렬 키가 일치해 페이지 경계에서 누락/중복이 없음. (5) 고객 삭제 이벤트가 별표 행도 지움. (6) 마이그레이션 upgrade/downgrade/upgrade 두 엔진과 sqlite 테스트 스키마 갱신.
- 비용: M (약 4일).

### 3.8 B3 Mentions

- 소유: bin-talk-manager(추출과 저장), bin-api-manager, bin-openapi-manager.
- 추출 위치: `messagehandler.MessageCreate`(`pkg/messagehandler/message.go`)의 저장 직전. `talk-control` 의 메시지 생성도 같은 핸들러를 쓰므로 거기서도 동작하며, `type` 이 `system` 인 메시지는 추출하지 않습니다. 정규식은 클라이언트와 같은 형태이되 id 를 UUID 로 제한합니다: `\[@([^\]]+)\]\(mention:([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})\)`. 연락처 멘션 `mention:contact:<id>` 는 UUID 형태가 아니므로 자연히 제외되고, 이름에 `]` 가 있으면 클라이언트도 멘션으로 인식하지 않으므로 서버도 같은 결과입니다. 한 메시지에서 서로 다른 에이전트 최대 50명까지만 기록하고(저장 증폭 상한), 작성자 본인 멘션은 제외합니다. 멘션 id 가 같은 고객의 에이전트인지는 확인하지 않습니다(메시지 전송마다 agent-manager RPC 가 필요해지고, 조회가 호출자 본인의 id 와 고객 id 로만 걸려 노출 위험이 없음).
- 저장: 메시지 행과 멘션 행을 **하나의 트랜잭션**으로 기록하는 `dbhandler.MessageCreateWithMentions` 를 신설합니다(트랜잭션 헬퍼가 없고 선례는 `bin-ai-manager` 의 `AIAcceptProposal`). 실패하면 메시지 생성도 실패합니다.
- 데이터 모델 (talk-manager 신규 마이그레이션):

```sql
CREATE TABLE talk_mentions (
  id binary(16) NOT NULL,
  customer_id binary(16) NOT NULL,
  chat_id binary(16) NOT NULL,
  message_id binary(16) NOT NULL,
  agent_id binary(16) NOT NULL,
  tm_create datetime(6) NOT NULL DEFAULT current_timestamp(6),   -- the INSERT sets the message's tm_create explicitly
  PRIMARY KEY (id),
  UNIQUE KEY uq_talk_mentions_message_agent (message_id, agent_id),
  KEY idx_talk_mentions_agent_create (customer_id, agent_id, tm_create),
  KEY idx_talk_mentions_chat_id (chat_id),
  CONSTRAINT talk_mentions_ibfk_1 FOREIGN KEY (chat_id) REFERENCES talk_chats (id) ON DELETE CASCADE,
  CONSTRAINT talk_mentions_ibfk_2 FOREIGN KEY (message_id) REFERENCES talk_messages (id) ON DELETE CASCADE
) DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
```

- API: `GET /service_agents/talk_mentions?page_size&page_token` -> 호출 에이전트를 멘션한 메시지(`TalkManagerMessage[]`, 스키마 재사용), 메시지 `tm_create` 내림차순. 내부 RPC `GET /v1/mentions` (body 에 `customer_id`, `agent_id`). 쿼리는 멘션 행을 메시지와 채팅에 조인해 3.6 의 접근 규칙 R(삭제 제외, 고객 일치, `talk` 타입은 같은 고객 전체, direct/group 은 참가자)을 SQL 로 적용합니다. 읽음/미읽음 상태는 이 설계 범위가 아닙니다(질문 목록).
- 소급 없음(P5). 롤링 구간에 구 talk-manager pod 가 만든 메시지는 멘션 행이 없습니다(배포 시간만큼의 공백, 허용).
- 롤링: talk-manager 먼저. 구 코드는 새 테이블을 참조하지 않고, 신 코드는 시작 시 마이그레이션이 끝난 뒤에만 리스닝합니다.
- Verify: (1) 추출 단위: 에이전트 멘션만 추출, 연락처 멘션 제외, 중복/자기 멘션 제외, 50명 상한, `]` 가 들어간 이름, 잘못된 UUID. (2) 메시지+멘션 원자성: 멘션 INSERT 실패 시 메시지가 남지 않음. (3) 삭제된 메시지/채팅이 목록에서 빠짐. (4) 접근 불가 direct/group 채팅의 멘션이 목록에 나오지 않음(규칙 R 과 `canAccessChat` 의 판정 표, 삭제와 고객 불일치 축에서만 차이). (5) 타 에이전트의 멘션이 보이지 않음. (6) 페이지 경계. (7) 마이그레이션 upgrade/downgrade/upgrade 두 엔진.
- 비용: M (약 4~5일).

### 3.9 C1 Transfer (attended, blind)

- 소유: bin-api-manager, bin-openapi-manager. transfer-manager 무변경(그러나 호출자/소유자/고객을 검증하지 않으므로 api-manager 검증이 유일한 방어선: 이 PR 이 책임).
- API: `POST /service_agents/transfers`. 요청 필드는 admin `/transfers` 와 같은 이름(`transfer_type`, `transferer_call_id`, `transferee_addresses`)이나 `transferee_addresses` 는 3.13 의 `CallManagerParticipantAddress`(P2 (a): `extension` 한 개, `maxItems: 1`)로 좁혀집니다, 응답 `TransferManagerTransfer`. 새 path 파일 `service_agents/transfers.yaml`, Admin `/transfers` 는 변경하지 않습니다.
- 서버 검증 (순서): (1) `IsAgent`, PermissionAll. (2) `callGet(transferer_call_id)`, `CustomerID == a.CustomerID`, `OwnerID == a.AgentID()`(에이전트 자신의 레그만 transferer 가 될 수 있음, 소유자가 비어 있으면 거부). (3) 통화가 `progressing` 이고 `ConfbridgeID != uuid.Nil`(없으면 400 "the call is not transferable", transfer-manager 가 500 으로 뭉뚱그리는 에러를 앞에서 구분). (4) `transferee_addresses` 는 정확히 1개(v1)이고 P2 (a) 의 주소 규칙을 서버가 검증한다: `extension` 만 받으며 레지스트라에서 (고객, 내선 번호)로 조회해 같은 고객의 내선인지 확인하고, 그 밖의 유형(`agent`, `tel`, `sip`)은 400 입니다. 본인 내선으로의 전환은 막지 않습니다(자기 전화가 울릴 뿐 무해). (a) 에서는 `agent` 유형이 없으므로 에이전트 조회(`AgentV1AgentGet`), 에이전트 `Addresses` 검사, 발신 시점 주소 변경(TOCTOU) 처리가 모두 필요하지 않습니다. (b) 를 고르면 2.3 P2/P3 의 (b) 항목을 따릅니다. (5) 검증한 값으로 주소를 **서버가 새로 조립**해 전달한다(`type` 과 검증된 식별자만 쓰고 클라이언트가 보낸 `target_name`, `name`, `detail` 은 버림; extension 의 식별 필드는 문서(`target` = 번호)와 발신 해석(`dial.go` 가 `TargetName` 으로 레지스트라 조회)이 어긋나 보여 구현 PR 이 코드를 따라가 확정, 스파이크 S7). (6) `TransferV1TransferStart` 호출.
- attended 의 완료/취소: 완료는 transferer 가 통화를 끊을 때의 기존 의미(`transferer.go`)이며 새 API 가 없습니다. transferee 가 응답하지 않거나 먼저 끊는 경우는 transfer-manager 가 `attendedUnblock` 으로 자동 복구합니다(`transferee.go`). **attended 취소(transferee 를 끊고 고객에게 돌아가기)는 transfer-manager 에 취소 RPC 가 없고 transferee 레그 소유자가 transferee 라 에이전트가 직접 끊을 수 없어** 이 설계에서는 범위 밖으로 두며, `transferer.go`/`transferee.go` 의 전체 분기를 읽는 스파이크(3.11 의 S6) 후 필요하면 `POST /service_agents/transfers/{id}/cancel` 과 transfer-manager 신규 RPC(약 +1.5일)를 같은 항목의 후속 설계로 추가합니다.
- 데이터 모델 변경 없음. 롤링: 순서 제약 없음(기존 RPC).
- Verify: (1) 타인 소유 통화/다른 고객 통화 거부. (2) 소유자 비어 있는 레그 거부. (3) `progressing` 이 아닌 통화, confbridge 없는 통화 400. (4) 주소 규칙(P2 (a)): `extension` 외 유형(`agent`, `tel`, `sip`) 400, 다른 고객 extension 거부, `transferee_addresses` 가 2개 이상이거나 비어 있으면 400, 본인 내선 요청은 막지 않음(동작 고정), 클라이언트가 보낸 `target_name`, `name`, `detail` 이 전달되지 않고 서버가 조립한 `extension` 주소만 전달됨. (5) blind/attended 정상 경로는 mock transfer-manager 로 호출 인자 확인. (6) 직접 접근(`IsDirect`) 거부. (7) Admin `/transfers` 의 권한 검사가 변하지 않음(회귀).
- 비용: S~M (약 3일, 취소 제외. 개정 1 이전 견적 4일에서 에이전트 조회, 주소 검사, TOCTOU 처리가 빠져 1일 감소. (b) 를 고르면 약 4일).

### 3.10 C2 Record (start/stop)

- 소유: bin-api-manager, bin-openapi-manager, bin-call-manager(기존 결함 B, C, D 수정, 아래). A 는 에이전트 경로가 닿지 않는 기존 결함이라 필수 범위 밖이며 P8 에서 정합니다.
- 전제: 에이전트는 조직의 규율(녹음 고지 포함)을 이해하고 지키는 소속원으로 봅니다. 따라서 에이전트 녹음에는 서버 강제 고지, 시작 때마다의 동의 확인, 시작 주체 표식, 중지 제한을 두지 않습니다(P1, P7 결정됨). 테넌트 격리, 소유자 검증, 고객 일치 검증은 유지합니다.
- API: `POST /service_agents/calls/{id}/recording_start`, `POST /service_agents/calls/{id}/recording_stop`. start 본문은 `{format?, end_of_silence?, end_of_key?, duration?}` 이며 admin 의 `on_end_flow_id` 는 **제외**합니다(녹음 종료 시 임의 플로우 실행 권한을 에이전트에게 주지 않기 위해 `uuid.Nil` 로 고정). 응답은 `CallManagerCall`(WebhookMessage).
- 서버 검증: `IsAgent`, PermissionAll, `callGet`, 고객 일치와 `OwnerID == a.AgentID()`(서버가 에이전트 레그 한 개만 녹음, 소유자 비어 있으면 거부). **start 도 `progressing` 상태를 api-manager 가 강제합니다**(400): call-manager 의 `RecordingStart` 는 `progressing` 이 아닌 통화를 먼저 Answer 하므로(`callhandler/recording.go:46-51`) 울리는 에이전트 레그가 녹음 호출만으로 자동 응답되는 것을 막기 위함이며, 이 때문에 에이전트 경로는 아래 결함 A 에 닿지 않습니다. 이미 `call.recording_id` 가 있으면 api-manager 가 call-manager 로 보내기 전에 409 를 돌려줍니다(이미 읽은 통화 객체로 판정하는 best-effort 사전 검사이며, 원자적 방어가 아님은 아래 "알려진 한계"). 요청 값: `duration` 만 상한이 있습니다. call-manager 의 `duration` 은 초 단위이고 0 이 무제한이므로(`bin-flow-manager/models/action/option.go` 의 `OptionRecordingStart.Duration`, `bin-common-handler/pkg/requesthandler/ast_channel.go` 의 `AstChannelRecord` 가 쓰는 `maxDurationSeconds`) 에이전트 경로는 생략을 7200 으로 채우고 1~7200 만 받습니다(녹음은 과금되고 동시 녹음 제한이 없어 무제한 녹음의 비용을 막기 위함). `end_of_silence` 는 상한을 두지 않습니다(0 이상, 0 은 무제한 유지: 비용, 안전 가치가 낮아 개정 1 에서 제거). 감사: 시작/중지마다 에이전트 id, 통화 id, 동작을 구조화 로그로 남깁니다(기존 call-manager 녹음 이벤트는 그대로). 별도의 표식이나 알림은 두지 않습니다.
- 클라이언트 권고(의무 아님): 통화 중 녹음 상태 표시(REC)를 `call.recording_id` 로 보여 주면 에이전트가 녹음 여부를 알 수 있습니다. 이 값은 아래 결함 B 가 수정되어야 정확합니다.
- call-manager 수정 범위 (코드로 확인, 1.5): 한 줄 수정이 아닙니다.
  - B. 자연 종료(`end_of_silence`, `duration`, `end_of_key`) 때 `call.RecordingID` 해제. `recordinghandler` 는 `callhandler` 보다 먼저 초기화되므로(`cmd/call-manager/main.go`) `recordinghandler.Stopped` 가 `callhandler.UpdateRecordingID` 계열을 부르면 순환 의존이 됩니다(`recordinghandler` 가 가진 dbhandler 를 직접 부르는 대안도 있으나 call 갱신은 `callhandler` 에 두는 기존 패턴을 따름). 해제는 `callHandler` 를 가진 `arieventhandler` 의 `ari_recording.go` 에서 `Stopped` 직후, `r.ReferenceType == call` 일 때만 합니다. `Stopped` 는 녹음 상태를 `ended` 로 바꾼 뒤 파일 저장 등 `stopped()` 가 실패하면 오류를 반환하므로(`stop.go:17-42`) 그 오류 때문에 해제가 건너뛰어지지 않도록, 반환 오류를 잡아 둔 채 해제를 수행한 뒤 그 오류를 그대로 반환하고, 해제 자체의 실패는 로그만 남깁니다(이벤트 재처리를 유발하지 않음). `Stopped` 가 첫 단계(상태를 `ended` 로 기록)에서 실패한 경우에도 해제할지는 구현 PR 이 정하고 결과를 PR 본문에 적습니다(해제하면 녹음 상태는 `ended` 가 아닌데 `recording_id` 만 비는 불일치가 생김). 해제 조건은 `call.RecordingID == r.ID` 로 한정한 원자적 조건부 UPDATE(`WHERE id=? AND recording_id=?`)입니다. 필요한 변경은 이 조건부 갱신을 하는 새 dbhandler 메서드(`CallAddExternalMediaID` 처럼 직접 UPDATE 뒤 `callUpdateToCache` 로 Redis 를 갱신), 이를 부르는 새 callhandler 메서드, 두 인터페이스의 mock 재생성이며 5일 견적 안에 들어갑니다. 일치 조건이 필요한 이유: 명시적 stop 은 `callhandler/recording.go:104` 에서 먼저 해제하므로 뒤에 오는 Finished 시점의 `RecordingID` 는 비었거나 새 녹음의 id 일 수 있어, 무조건 해제하면 새 녹음 id 를 지워 동시 녹음이 생깁니다.
  - C. `recordinghandler/stop.go:138` 의 `RecordingSetStatus(Stopping)` 은 현재 상태를 보지 않는 UPDATE 라 이미 `ended` 인 녹음을 `stopping` 으로 되돌립니다(RST 의 "상태는 앞으로만 간다"와 충돌). 수정: `RecordingSetStatus(Stopping)` 을 조건부 UPDATE(`WHERE status IN (initiating, recording)`)로 바꾸고(현재 `RecordingSetStatus` 는 오류만 반환하고 `RecordingUpdate` 는 영향 행 수를 돌려주지 않으므로 이를 알려 주는 새 dbhandler 메서드가 필요), 갱신된 행이 없으면(이미 `ended`) 상태를 바꾸지 않고 현재 녹음을 그대로 반환합니다(읽은 뒤 확인하는 방식은 Finished 선착 경합을 막지 못하므로 쓰지 않음). B 가 stale id 를 없애면 자연 종료 뒤 stop 경로는 사라지지만, 정상 stop 중 hangup 한 채널의 Finished 가 138행보다 먼저 처리되는 경합은 남으므로 C 는 B 와 별개로 필요합니다.
  - D. `RecordingStop` 의 `(nil, nil)` 반환(`callhandler/recording.go:91-92`, `err` 가 nil 인 채 `errors.Wrapf(err, ...)` 호출). 구현 PR 이 `errors.Errorf` 로 바로잡고 api-manager 에서 nil 응답을 방어합니다.
  - 범위 밖으로 둔 것: confbridge 의 같은 자연 종료 stale(`confbridgehandler/recording.go:61,100`)은 에이전트 경로와 무관한 기존 결함이라 이 항목에서 고치지 않고 기록만 합니다.
- 알려진 한계와 위험:
  - 에이전트가 먼저 녹음을 켠 레그에서 플로우의 `recording_start` 가 나중에 실행되면 `callhandler/recording.go:36-38` 의 "already progressing" 오류가 납니다. 이 충돌 시나리오는 실행으로 검증하지 못했고(Asterisk 필요) 코드 읽기 근거입니다. 이 오류는 통화를 끊지 않습니다: 액션 오류는 다음 액션으로 넘어갈 뿐이며(`callhandler/action.go:147-148`) 끊김은 다음 액션 요청 자체가 실패할 때만 일어납니다(`action.go:203-206`). 실제 영향은 그 플로우 녹음이 건너뛰어져 플로우가 지정한 파라미터(예: `on_end_flow_id`)가 적용되지 않는 것입니다. 에이전트 녹음과 플로우 녹음은 대부분 서로 다른 레그(통화 id)에 걸려 충돌하지 않고, 같은 레그에서만 일어납니다. 완화: 에이전트 쪽은 위 409 와 B 수정으로 stale 로 인한 오탐을 줄이고, 플로우 녹음이 건너뛰어지는 것은 조직 규율(컴플라이언스 녹음이 필요한 플로우에서는 에이전트 수동 녹음을 켜지 않음)과 문서에 맡기며 서버 조정은 하지 않습니다(트리거: 건너뜀이 실제로 보고될 때 플로우 액션의 "이미 녹음 중이면 성공 처리" 여부를 별도 판단).
  - 자연 종료가 `RecordingStart` 의 id 설정(`callhandler/recording.go:70`)보다 먼저 처리되는 경우(아주 짧은 `duration` 이나 즉시 `end_of_key`)에는 B 의 조건부 해제가 아직 없는 id 를 지우려다 아무것도 하지 못하고, 뒤이어 id 가 설정되어 남습니다. 발생 조건이 좁아 v1 에서는 고치지 않습니다(트리거: 실제 stale 이 보고될 때 `Start` 가 id 설정 뒤 녹음 상태를 다시 확인). 이 stale 은 `recording_stop` 으로 풀 수 있습니다(C 적용 후 `ended` 녹음에 stop 을 호출해도 `ended` 가 유지되고 이어서 `callhandler/recording.go:104` 가 id 를 비움). `UpdateRecordingID` 는 call 이벤트를 발행하지 않으므로 클라이언트의 REC 표시는 `call.recording_id` 를 통화 조회로 읽는 방식이어야 하며 변경을 푸시로 받지 못합니다(녹음 이벤트 `recording_started`/`recording_finished` 웹훅을 쓰는 방안은 클라이언트 설계 몫).
  - 동시 start 경합: `RecordingStart` 의 `RecordingID` 검사(`callhandler/recording.go:36`)와 설정(`:70`)은 원자적이지 않고 `CallSetRecordingID` 는 무조건 덮어씁니다. 소유 에이전트는 통화당 1명이라 에이전트 사이의 경합은 없고, 에이전트와 플로우 사이에서만 이론상 가능합니다. 이중 녹음 2건이 생기고 이전 id 를 잃을 수 있으나 api-manager 사전 409 는 best-effort 일 뿐입니다. 발생 조건이 좁아 v1 에서는 고치지 않습니다(트리거: 실제 이중 녹음이 확인될 때 조건부 설정).
  - `Stop` 은 snoop 채널 hangup 실패를 로그만 남기고 성공으로 넘기며(`recordinghandler/stop.go:159-165`) 그 뒤 `callhandler/recording.go:104` 가 `RecordingID` 를 지웁니다. 채널이 살아 있는데 id 만 비어 두 번째 녹음이 시작될 수 있습니다. 에이전트에게 stop 을 열면 의미가 커지나 hangup 실패는 드문 경로라 이 항목의 필수 수정에는 넣지 않고 구현 PR 이 오류 반환 여부를 확인합니다.
  - 녹음 파일 조회/재생은 현재 Admin|Manager 전용이며 이 항목 범위가 아닙니다(에이전트가 시작한 녹음의 재생 경로는 6절 "추가로 정할 사항").
- 문서(RST): `bin-api-manager/docsdev/source/recording_overview.rst` 에 다음 취지의 한 줄을 추가합니다: "에이전트 API(`/service_agents/calls/{id}/recording_start`)로 시작한 녹음에서는 플랫폼이 통화 상대에게 녹음 사실을 알리지 않습니다. 고지는 조직의 운영 규율이며, 플로우에서는 `talk` 액션으로 고지를 재생하십시오." 문구는 Best Practices 의 "Legal Compliance"(`recording_overview.rst` 446행 부근) 아래에 두고 기존 법적 권고(플로우 대상, 446~452행)는 유지합니다. 같은 구현 PR 에서 "플랫폼 전체 최대 길이 없음, `duration` 미지정이면 무제한" 이라고 적은 두 곳(121행 부근 Key Behaviors, 307행 부근 Storage Details 표)에 "에이전트 API 는 `duration` 생략 시 7200초로 제한" 이라는 예외를 덧붙이고, 문서를 clean 재빌드해 `docsdev/build/` 와 함께 커밋합니다.
- 데이터 모델 변경 없음. 롤링: 순서 제약 없음(call-manager 의 B, C, D 는 기존 RPC 의 내부 동작 수정이라 api-manager 와 배포 순서가 무관하며, B 가 배포되기 전에는 자연 종료 뒤 start 가 계속 막히는 현행 동작).
- Verify: (1) 소유자/고객 불일치 거부, 소유자 비어 있음 거부. (2) `on_end_flow_id` 가 요청에 있어도 전달되지 않음(스키마에 없음). (3) 이미 녹음 중인 통화(`call.recording_id` 존재)에 start 는 409. (4) 녹음이 없는 통화에 stop 은 에러(수정 전 `(nil, nil)` 반환 재현 후 통과). (5) `progressing` 이 아닌 통화(울리는 레그)에 start 는 400이고 Answer 되지 않음. (6) `duration` 생략은 7200 으로 채워지고 명시적 0, 음수, 7201 은 400, `end_of_silence` 음수는 400 이고 상한은 없음(검증은 스펙 기반 요청 검증기가 api-manager 에 없으므로 서버 코드가 직접 하며, 생략과 명시적 0 을 구분하려고 요청 필드를 포인터로 받음). (7) Admin 엔드포인트 권한 불변. (8) 에이전트가 소유한 레그의 녹음은 시작 주체와 무관하게 stop 이 성공하고, start 와 stop 모두 감사 로그에 에이전트 id, 통화 id, 동작이 남음. (9) B: `Stopped` 가 오류를 반환해도 `call.recording_id` 가 해제되고 그 오류는 그대로 전달됨, 자연 종료(Finished) 뒤 `call.recording_id` 가 비고 다시 start 가능, 명시적 stop 뒤 새 녹음이 시작된 상태에서 늦게 도착한 이전 녹음의 Finished 가 새 id 를 지우지 않음(조건부 UPDATE), confbridge 녹음은 영향 없음. (10) C: `ended` 녹음에 stop 을 호출해도 `ended` 유지, 정상 stop 에서 Finished 가 먼저 와도 `ended` 유지. (11) 감사 로그에 통화 상대 정보나 개인정보가 들어가지 않음. (12) B 의 해제 실패는 로그만 남기고 이벤트 처리 결과에 영향이 없음. (13) RST 3곳(안내 한 줄, 121행 부근, 307행 부근)이 반영되고 `docsdev/build/` 가 clean 재빌드됨.
- 비용: S 에서 M 으로 상향, 약 5일 (api-manager/openapi 2일 + call-manager B 1.5일 + C 0.5일 + D 0.25일 + RST/테스트 0.75일. P8 의 A 는 제외).

### 3.11 C3 Add (3자 통화)

- 소유: bin-call-manager(신규 RPC), bin-api-manager, bin-openapi-manager. 선택 근거는 2.2(설계안 Y).
- API: `POST /service_agents/calls/{id}/participants` body `{address: {type: extension, target}}` (P3 (a)) -> 200 `CallManagerGroupcall`(새 참가자 레그의 groupcall, 개정 2 정정 (b): `call_ids` 에 레그 id. call 행은 응답 뒤에 비동기로 생성되고 contact 가 여럿이면 레그도 여럿이라 call 하나를 결정적으로 돌려줄 수 없음).
- api-manager 검증: C1 과 같은 소유자/상태 검증과 주소 규칙(재사용 헬퍼: P2/P3 (a) 의 `extension` 만, 서버가 주소를 새로 조립; (b) 를 고르면 2.3 의 차이를 따름), 통화가 `progressing` 이고 `ConfbridgeID != Nil`. 본인 내선을 대상으로 한 요청은 C1 과 같이 막지 않습니다(본인 전화가 울릴 뿐 다른 사람에게 영향 없음). 통화가 이미 3자이면 409(v1 은 2자 -> 3자까지, 그 이상 확장은 범위 밖). 제3자가 응답하기 전에는 `ChannelCallIDs` 가 늘지 않으므로 이 검사만으로는 연속 요청 둘이 모두 통과할 수 있어, call-manager 가 confbridge id 기준의 SET NX 점유(선례: `bin-call-manager/pkg/cachehandler/handler.go` 의 `CallRecoveryClaim`)를 잡고 점유 중이면 409 를 돌려줍니다. 점유는 모든 검증을 통과한 직후, 발신 생성 직전에 잡고(검증 실패로 점유가 남지 않게), 캐시 오류가 나면 점유를 못 잡은 것으로 보고 거부(fail-closed)합니다. 새 `cachehandler` 메서드와 mock 은 call-manager 4일 견적에 포함합니다. TTL 은 제3자 발신의 응답 대기 상한과 같게 하고, 발신 생성이 동기적으로 실패하면 바로 해제합니다. 제3자가 거절한 뒤 재시도는 TTL 이 지나야 할 수 있으며 이는 v1 의 알려진 제약입니다(트리거: 사용성 문제가 보고될 때 참가자 레그 hangup 시 해제 추가).
- call-manager `POST /v1/calls/{id}/participants` (`callHandler.AddParticipant`): (1) 통화와 confbridge 조회(`connect` 타입, `Progressing`, `ChannelCallIDs == 2`), (2) `confbridge_join(confbridge_id)` 하나짜리 임시 flow 생성(transfer-manager 의 `createFlow` 와 같은 구성), (3) 대상 주소로 발신 레그 생성(`source` 는 원 통화의 source, `masterCallID` 는 **원 통화의 `MasterCallID`**, 비어 있으면 `uuid.Nil`; attended transfer 가 `transfererCall.MasterCallID` 를 쓰는 선례와 같음. 원 통화 자체를 master 로 삼으면 에이전트 레그가 끊길 때 chained call 로 제3자도 끊기므로 피함, `callhandler/hangup.go` 의 chained call hangup). 대상이 응답한 뒤 flow 가 confbridge 에 합류시킵니다. 응답은 groupcall 입니다(개정 2 정정 (b): `startWithDestination` 이 레지스트라 contact 마다 call id 를 동기로 정해 `groupcall.call_ids` 에 넣고, call 행은 고루틴에서 `CallV1CallCreateWithID` 로 비동기 생성합니다. 발신 주소는 정정 (c) 에 따라 `target` 에 extension id, `target_name` 에 내선 번호를 넣어 api-manager 가 조립합니다). (a) 에서는 `extension` 만 받으므로 "에이전트는 extension 이 정확히 1개" 라는 제한이 필요 없습니다(`agent` 유형 자체가 없음). 단 `extension` 한 개도 등록된 contact 가 여럿이면 레그가 여럿이 되고(ring-all, 응답한 하나만 합류), 레그의 owner 는 내선 소유 동료 에이전트가 되어 동료 UI 에 보이고 동료가 제어할 수 있습니다. (b) 에서 `agent` 유형을 허용하면 groupcall 이 여러 레그를 만들 수 있어 이 제한이 다시 필요하고, 그 경우 검증된 `extension` 주소가 정확히 1개인 에이전트로 한정합니다. 기존 레그에는 MOH/mute 를 걸지 않습니다(attended transfer 와의 차이). 외부 PSTN 비용: (a) 에서는 내부 extension 만 허용하므로 발생하지 않고 검증도 내부 extension 만 사용합니다. (b) 에서 `tel` 을 허용하면 3자 통화 시간 동안 PSTN 비용이 누적됩니다. 이 누적은 발신 가드(`CreateCallOutgoing` 의 고객 상태, 본인확인, 잔액 검사)가 건당 통제하지만 통화 시간 전체를 제한하지는 않으므로 별도 위험으로 유지합니다(트리거: (b) 선택 시 통화 시간 상한 또는 경고를 설계).
- 검증 순서와 reason (개정 2, 정정 (d)): api-manager 는 상담사 신원과 소유자, 주소 형식(`INVALID_PARTICIPANT_ADDRESS` 400), 통화 상태(`CALL_NOT_JOINABLE` 409), confbridge 의 `ChannelCallIDs != 2`(`PARTICIPANT_COUNT_INVALID` 409, 빠른 실패용), extension 조회(없음, 타 고객, 삭제는 동일 400) 순서로 검사한 뒤 call-manager 를 호출합니다. call-manager 는 주소, 통화, confbridge(`connect`, `progressing`, `ChannelCallIDs == 2`)를 검사하고 그 뒤에 SET NX 점유(`confbridge:participant_add:<id>`, TTL 90초 = 발신 대기 60초 + join 여유 30초)를 잡습니다. 점유가 없거나 캐시 오류이면 `PARTICIPANT_ADD_IN_PROGRESS`(409, fail-closed), extension 에 등록된 contact 가 없으면 기존 `EXTENSION_NOT_REGISTERED`(409). 점유와 임시 flow 는 flow 생성 또는 groupcall 시작이 동기로 실패할 때만 해제(요청 ctx 취소와 무관)하고, 발신이 시작된 뒤에는 TTL 까지 유지합니다.
- 종료 의미 (개정 2: 스파이크 S5 실측으로 확정. 아래 "코드 확인과 추정" 서술 중 토폴로지 B 는 정정됨): `connect` 타입 confbridge 는 한 명만 남으면 자동 종료됩니다(`confbridgehandler/leaved.go:55`). 에이전트 레그와 제3자 레그가 같은 master(고객 레그)에 chained 되면 master 가 끊길 때(고객 이탈) chained 레그가 모두 끊기고(`hangup.go` 의 chained call hangup), 에이전트 이탈 시에는 master 가 남아 고객과 제3자가 통화를 이어 갑니다. 즉 고객이 이탈하면 통화 전체가 끝나고 에이전트가 이탈하면 고객+제3자는 유지됩니다. 이는 **고객 레그가 master 인 토폴로지(큐 경유 인입, `masterCallID = qc.ReferenceID`)** 기준입니다. 에이전트가 직접 건 발신 통화는 에이전트 레그의 `MasterCallID` 가 비어 있고 상대 레그가 에이전트 레그에 chained 되므로, 이 토폴로지에서는 상대가 끊어도 에이전트와 제3자가 유지되지만 **에이전트가 끊으면 상대 레그가 chained 로 끊긴 뒤 제3자 혼자 남은 confbridge 가 자동 종료되어 제3자까지 전원 종료됩니다**(실측 B2, B4, 종료 순서 에이전트, 상대, 제3자). 이 동작은 수용하며 `FlagNoAutoLeave` 등 회피책은 구현하지 않습니다(트리거: 제품 요구가 생기면 별도 설계). 실측 요약: 토폴로지 A(고객이 master): 고객 이탈은 전원 종료, 에이전트 이탈은 고객과 제3자 유지, 제3자 이탈은 나머지 유지, 두 명이 이탈하면 남은 한 명도 종료. 제3자의 master 를 에이전트 레그로 하면(기각한 변형) 에이전트 이탈 시 제3자가 즉시 종료되어 "원 통화의 `MasterCallID`" 선택의 근거가 확인되었습니다. API 문서(RST, OpenAPI)와 응답 설명에 이 동작을 명시합니다.
- 스파이크 실측 결과 (개정 2, 실제 Asterisk 환경에서 내부 extension 으로 실행, 이미지는 install 고정본): S1 3번째 레그 합류 시 기존 레그의 재INVITE, BYE, ring 없이 `ChannelCallIDs` 가 2에서 3이 되고 상태 유지, 제3자 무응답은 약 30초 뒤 제3자 레그만 hangup 이며 기존 두 레그와 confbridge 는 영향 없음, call-manager 는 4번째 참가자를 막지 않으므로 상한은 정정 (d) 로 신규 구현. S2 에이전트 레그의 call 녹음은 제3자 합류 뒤에도 유지되고 3자 상태에서 레그별 call 녹음 3개와 confbridge 녹음 1개가 동시 시작 성공, 같은 레그의 재시작은 500(C2 의 사전 409 가 처리, 정정 (e)), 자연 종료 뒤 `recording_id` 가 stale 로 남는 결함은 C2 의 범위. S3 Insight listen 은 2자에서만 시작되고 3자에서는 약 45초 대기 뒤 시작하지 않으며 제3자 이탈로 2자가 되면 다시 시작됨(이미 청취 중인 세션이 3자에서 어떻게 되는지는 STT 미설정으로 미검증). S5 는 위 종료 의미. S7 은 C1 에서 확정(`target` = extension id, 발신은 `target_name` 만 해석, `target` 만 주면 409 `EXTENSION_NOT_REGISTERED`). S4(UI)와 S6 은 이 PR 범위 밖. 미검증: 등록 UA 레그의 음성 믹싱 청취, 이미 청취 중인 Insight 세션의 3자 동작. 증거는 별도 증거 문서에 있습니다.
- 영향과 스파이크 항목 정의(구현 착수 전 sandbox 에서 내부 extension 으로 실행): S1 3번째 레그 합류 시 `joinedTypeConnect`(`joined.go:58~`)가 이미 응답한 레그에 미치는 영향(대상이 응답한 뒤 합류하므로 `Ring` 분기를 피하는 설계이지만 미검증). S2 3자 통화 중 call 단위 녹음(`recording.go`, 시작 시 `RecordingID` 존재 검사)과 confbridge 녹음의 충돌/유지(3.10 의 에이전트 녹음은 call 단위이며, confbridge 의 `RecordingID` 도 자연 종료 때 지워지지 않는 같은 결함이 있음: 1.5). S3 Insight listen: 시작 조건이 정확히 2명이라(`listen_trigger.go` `lastPartyCount == 2`) 3자 이후에는 시작되지 않고, 이미 청취 중이던 세션이 3자에서 어떻게 되는지(화자 매핑이 2자 구조에 기대고 있음)는 미확인. S4 에이전트 webphone 의 UI 가 제3자 레그를 보여주는 방법(클라이언트 설계). S5 이탈 주체별(고객, 에이전트, 제3자) x 토폴로지별 연쇄 종료 의미와 외부 이벤트(`call_*`, `confbridge_*`)의 3자 처리. S6 `transferer.go`/`transferee.go` 전체 분기(C1 취소 판단). S7 extension 주소의 식별 필드(문서의 `target` = 번호와 발신 해석의 `TargetName` 불일치, 그리고 `bin-common-handler/models/address/validate.go:33` 의 검증과 `validate_test.go:58-59` 는 extension 의 target 으로 UUID 만 유효하다고 봄. C1 과 공통이며 P2 (a) 에서는 `extension` 이 유일한 유형이라 S7 을 C1, C3 구현의 첫 작업으로 확정해야 함).
- 데이터 모델 변경 없음. 롤링: call-manager 먼저(새 RPC 경로).
- 알려진 한계 (개정 2): 토폴로지 B 에서 에이전트 이탈 시 전원 종료. 점유 TTL(90초) 동안은 제3자 거절이나 무응답 뒤에도 재시도가 거부될 수 있음. 응답한 뒤 join 완료 전에 점유가 만료되고 `ChannelCallIDs` 가 아직 2 이면 두 번째 요청이 통과할 수 있는 좁은 창(60초 대기 뒤 30초 이상의 join 지연이 겹칠 때). 비동기 레그 생성 실패는 동기 해제가 아니라 TTL 대기. 통화 검사 통과 직후 master 가 끊기면 chain 에 묶이지 않은 제3 레그가 울리고 응답하면 종료 중인 confbridge 에 join 을 시도함(attended transfer 와 같은 구조). 블라인드 전환 진행 중인 통화는 transfer 상태를 조회하지 않으므로 거부되지 않음. 점유 해제는 소유 토큰이 없는 DEL 이라 동기 구간에서 남의 점유를 지울 수 있음(트리거: 관측 시 값 비교 삭제). 실측한 두 토폴로지(큐 경유 인입, 상담사 발신) 밖의 통화(예: 상담사 내선으로 걸려 온 직접 통화)의 종료 의미는 미실측. 제3 레그는 owner 가 동료이므로 요청한 상담사의 `GET /service_agents/calls` 에는 나타나지 않음.
- Verify: (1) 소유자/상태/참가자 수(2 -> 3) 검증과 409. (2) 제3자 응답 전에는 합류하지 않음, 응답 후 confbridge `ChannelCallIDs` 가 3. (3) 제3자 거절/타임아웃 시 기존 두 레그가 영향받지 않음. (4) 토폴로지별(큐 경유 인입, 에이전트 발신) x 이탈 주체별(고객, 에이전트, 제3자) 기대 동작을 따로 검증하고 두 명 이탈 시 종료. (4-1) 제3자 응답 전 연속 요청 두 번째는 409, 발신 생성이 동기 실패하면 점유가 즉시 해제되어 재시도 가능, TTL 경과 뒤 재시도 허용, 캐시 오류 시 거부. (5) 유형 allowlist, 다른 고객 대상 거부. (6) 스파이크 S1~S7 의 결과 기록.
- 비용: L (스파이크 2일 + call-manager 4일 + api-manager 2일 + 테스트/문서 2일, 약 10일. (a) 는 `agent` 유형 처리와 'extension 정확히 1개' 검증이 없어 약 9일).

### 3.12 C4 SLA, C5 우선순위: 제품 정의 초안과 질문 목록

현재는 필드도 정의도 없으므로 구현하지 않고 정의 초안과 질문을 둡니다(P6).

**C5 우선순위 초안**
- 값 집합: `low`, `normal`(기본), `high`, `urgent`.
- 부여 주체: (1) 플로우 작성자가 `case_create` 액션 옵션으로 정적으로 지정, (2) 에이전트가 케이스 수정에서 변경. 자동 산정은 v1 에 두지 않음.
- 효과: 표시와 정렬만. queue 라우팅은 `routing_method` 와 `tag_ids` 만 쓰므로 큐 대기 순서에 영향을 주지 않음.
- 데이터 모델 초안: `contact_cases.priority varchar(16) NOT NULL DEFAULT 'normal'`(기존 케이스는 `normal`).

**C4 SLA 후보 정의** (택일 필요)
| 후보 | 측정 대상 | 필요한 데이터 |
|---|---|---|
| S-A 큐 대기 SLA | 큐 호출 중 N초 이내에 에이전트에게 연결된 비율 | 이미 있음: `queuecall.duration_waiting`. 큐별 목표 초 필드 필요 |
| S-B 케이스 첫 응답 SLA | 케이스 생성부터 담당 에이전트의 첫 응답까지 | "첫 응답" 사건 정의와 기록 필드가 없음 |
| S-C 케이스 해결 SLA | 케이스 생성부터 종료까지 마감 시각 | 마감 시각 필드(`sla_due_at`)와 산정 규칙 |
square-talk 시안의 "SLA 정렬과 마감 시간" 은 S-C 에 가깝습니다. 데이터 모델 초안(S-C 기준): `contact_cases.sla_due_at datetime(6) NULL`(생성 시 `opened_at + 목표 초`로 계산, 목표 초는 `case_create` 옵션 `sla_seconds` 로 전달).

**정의 필요 질문 목록**
1. SLA 가 측정하는 것은 S-A, S-B, S-C 중 무엇인가(복수 가능한가).
2. 목표 시간은 누가 어디서 설정하는가(고객 전역, 큐별, 우선순위별, 플로우 옵션).
3. 영업시간(business hours)을 제외하고 계산하는가.
4. 위반 시 동작은 표시만인가, 알림/웹훅/에스컬레이션을 포함하는가.
5. 메신저 대화 케이스(`conversation_message`)에도 같은 SLA 를 적용하는가.
6. 우선순위의 값 집합과 기본값은 위 초안으로 충분한가, 부여는 수동/플로우 중 무엇을 허용하는가.
7. 우선순위가 큐 라우팅이나 대기 순서에 영향을 주어야 하는가(주어야 한다면 별도 큐 설계 필요).
8. 기존 케이스의 기본값(우선순위 `normal`, SLA 없음)과 목록 정렬 규칙은 무엇인가.
9. 보고서/통계/웹훅에 노출해야 하는가.
- 비용: 정의 확정 전에는 산정하지 않음. 확정 뒤 스키마 변경 규모는 케이스 컬럼 1~2개와 `case_create` 옵션, 목록 정렬이므로 M 으로 예상(참고용).

### 3.13 신규 OpenAPI 스키마 초안

`bin-openapi-manager/CLAUDE.md` 의 규칙(유형별 `oneOf`, 구조화 문자열의 `format`, ID 설명에 출처 엔드포인트, 모든 leaf 의 현실적인 example, 필수 배열의 `minItems`)과 새 스키마의 서비스명 접두 규칙을 따른 초안입니다. 구현 PR 은 이 초안을 `components/schemas` 에 옮기고 `go generate` 합니다.

```yaml
TalkManagerPin:
  type: object
  properties:
    id:          { type: string, format: uuid, description: "The pin ID returned from POST /service_agents/talk_pins.", example: "6f1c1a52-8d3b-4c1e-9a5e-0b7f2d9a1c10" }
    target_type: { type: string, enum: [chat, message], description: "What is pinned.", example: "message" }
    target_id:   { type: string, format: uuid, description: "The chat ID returned from GET /service_agents/talk_chats, or the message ID returned from GET /service_agents/talk_messages, depending on target_type.", example: "0d4b7a3e-2f5c-4b8e-8a21-9c3e6f1d7a42" }
    chat_id:     { type: string, format: uuid, description: "The chat the pinned chat or message belongs to. The ID is returned from GET /service_agents/talk_chats.", example: "9a2b5c7d-1e3f-4a6b-8c9d-0e1f2a3b4c5d" }
    message:     { allOf: [ { $ref: '#/components/schemas/TalkManagerMessage' } ], description: "Present only when target_type is message." }
    tm_create:   { type: string, format: date-time, description: "When the pin was created.", example: "2026-11-02T09:15:30.123456Z" }

ContactManagerContactStar:
  type: object
  required: [contact_id, starred]
  properties:
    contact_id: { type: string, format: uuid, description: "The contact ID returned from GET /service_agents/contacts.", example: "b2c3d4e5-f6a7-8901-bcde-f12345678901" }
    starred:    { type: boolean, description: "Whether the calling agent has starred the contact.", example: true }
    tm_create:  { type: string, format: date-time, description: "When the contact was starred. Absent when starred is false.", example: "2026-11-02T09:15:30.123456Z" }

QueueManagerQueuecallSummary:
  type: object
  properties:
    id:         { type: string, format: uuid, description: "The queuecall ID of the queue entry this call belongs to.", example: "3c8e1f20-7a4b-4d6e-9b1a-5f2c8d0e7a31" }
    queue_id:   { type: string, format: uuid, description: "The queue ID. Returned as queue_id of GET /queues for administrators.", example: "7d2a9f10-3b5c-4e8a-a6d1-1c4e7b9f2a05" }
    queue_name: { type: string, description: "The queue name.", example: "Support" }
    status:     { allOf: [ { $ref: '#/components/schemas/QueueManagerQueuecallStatus' } ], description: "The queue entry status." }
    tm_create:  { type: string, format: date-time, description: "When the call entered the queue.", example: "2026-11-02T09:10:00.000000Z" }

CallManagerParticipantAddress:        # used by POST /service_agents/transfers and POST /service_agents/calls/{id}/participants (P2/P3 (a): extension only)
  type: object
  required: [type, target]
  properties:
    type:   { type: string, enum: [extension], example: "extension" }
    target: { type: string, description: "The extension of the colleague to reach, taken from the addresses of GET /service_agents/agents (type extension). Whether the value is the extension number or the extension ID is fixed in the implementation after spike S7.", example: "2001" }
# P2/P3 (b) would turn this into a oneOf over extension, tel (E.164 pattern) and agent; see 2.3.
```

- C1 요청(`service_agents/transfers.yaml`): `transfer_type`(`TransferManagerTransferType`), `transferer_call_id`(`type: string, format: uuid`, 설명 "The call ID returned from GET /service_agents/calls. It must be a call the calling agent owns.", example), `transferee_addresses`(`type: array, items: $ref CallManagerParticipantAddress, minItems: 1, maxItems: 1`). 세 필드 모두 `required`.
- C3 요청: `{ address: $ref CallManagerParticipantAddress }`, `required: [address]`.
- C2 start 요청: `format`(`CallManagerRecordingFormat`), `end_of_silence`(`integer, minimum: 0`, 0 은 무제한 유지), `end_of_key`(`string`), `duration`(`integer, minimum: 1, maximum: 7200`, 설명에 "생략하면 7200" 을 적고 스키마의 `default` 키는 쓰지 않음: 0 은 call-manager 에서 무제한이므로 에이전트 경로는 0 을 받지 않음. 생략 시 7200 채움과 범위 검사는 서버 코드가 직접 함, 3.10). admin 의 `on_end_flow_id` 는 없음.
- A4 close 요청: 3.4 의 `disposition` enum 본문(선택). 응답 스키마 `ContactManagerCase` 의 `disposition` 은 `type: string` 에 값 목록을 설명으로 기재.
- B3 응답은 기존 `TalkManagerMessage` 를 재사용하고 새 스키마가 없습니다.

## 4. 스키마 변경 영향도 분석

대상: A4(`contact_cases.disposition`), B1(`talk_pins`), B2(`contact_stars`), B3(`talk_mentions`). 모두 서비스 소유 마이그레이션(각 서비스 `migrations/`)이며 `bin-dbscheme-manager/bin-manager/**` 의 Alembic 스트림은 동결 상태라 건드리지 않습니다. C4, C5 의 컬럼은 정의가 확정된 뒤 같은 형식으로 별도 분석합니다.

공통 사항: 파일당 DDL 한 문장, up/down 쌍, 번호는 서비스 안에서 공백 없이 이어짐(현재 `000001` 만 있음), 머지된 파일은 수정하지 않음. 새 테이블은 `schema-ownership.md` 의 해당 서비스 소유 목록과 서비스 `docs/domain.md` 에 추가하고 sqlite 테스트 스키마를 수동 갱신합니다. 구현 PR 본문에는 이 절을 그대로 옮겨 실측으로 보강한 '영향도 분석' 섹션을 쓰며, 파괴적 연산이 없으므로 `[DESTRUCTIVE]` 표시 대상이 아닙니다(아래 a-1).

| 항목 | A4 `disposition` | B1 `talk_pins` | B2 `contact_stars` | B3 `talk_mentions` |
|---|---|---|---|---|
| a-1 파괴적 연산 | 없음. up 은 nullable 컬럼 추가 1건(DROP, TRUNCATE, 타입 축소/변경, 기본값 제거/변경, 이름 변경, 기존 데이터에 대한 NOT NULL/UNIQUE/FK/CHECK 추가, enum 제거, 기본값 없는 NOT NULL 컬럼 추가 모두 해당 없음). down 의 `DROP COLUMN` 은 롤백 평가에만 해당(`-- DOWN-LOSSY`) | 없음. up 은 `CREATE TABLE` 1건. 새 테이블 안의 UNIQUE/FK 는 기존 데이터가 없어 위반 불가. down 의 `DROP TABLE` 은 롤백 평가에만 해당(`-- DOWN-LOSSY`) | 없음 (B1 과 같은 이유) | 없음 (B1 과 같은 이유) |
| a-2 롤링 배포 호환 (main 의 사용처) | `contact_cases` 를 읽고 쓰는 구 코드의 쿼리는 `kase.Case` 의 db 태그 목록으로 컬럼을 명시하므로 새 컬럼을 읽지도 쓰지도 않음. 신규 `Disposition` 은 nullable 문자열이라 NULL 도 구 응답과 호환 | 테이블명 `talk_pins` 의 main 사용처 0건 | `contact_stars` 0건 | `talk_mentions` 0건 |
| a-3 락, 소요 시간 | `ALGORITHM=INPLACE, LOCK=NONE` 명시(거부되면 마이그레이션이 실패하므로 두 엔진 scratch 실측 필수, 2.2). 대상 `contact_cases` 의 행 수는 이 문서 작성 시 확인하지 못함: 구현 PR 은 대상 테이블 규모의 측정 근거(읽기 전용 조회 결과)를 PR 본문에 적어야 하며 '모름'이나 확인 방법만 적는 것은 허용되지 않음 | 새 테이블이라 데이터가 없어 소요 시간은 DDL 1건. 부모 테이블(`talk_chats`, `talk_messages`)을 참조하는 FK 를 가진 `CREATE TABLE` 은 부모에 대한 메타데이터 락 대기가 생길 수 있으므로 구현 PR 이 이 가능성을 scratch 에서 확인하고 적는다 | 새 테이블, FK 없음. 데이터 없음 | B1 과 같음 (FK 2개) |
| a-4 롤백 | 서비스 소유 마이그레이터는 전진 호환이라 구 이미지가 신 스키마(추가 컬럼)에서도 동작함. 실제 롤백은 코드 되돌림 배포이며 컬럼은 남겨도 무해. down 은 로컬 일회용 DB 에서만 실행 | 코드 되돌림 배포. 테이블은 남겨도 무해 | 같음 | 같음 |
| a-5 데이터 손실 | up: 없음. down: 기록된 처분 값이 사라짐(재입력 필요, 케이스 자체는 유지) | up: 없음. down: 에이전트의 개인 pin 이 사라짐(개인 상태) | up: 없음. down: 에이전트의 별표가 사라짐 | up: 없음. down: 멘션 인덱스가 사라지며 메시지는 유지(재생성 가능: 본문에서 다시 추출) |
| (d) 구 코드 / 신 스키마 | 동작함 (a-2) | 구 코드는 참조하지 않음 | 같음 | 같음 |
| (d) 신 코드 / 구 스키마 | 발생하지 않음: 서비스는 리스닝 전에 마이그레이션을 끝냄 | 같음 | 같음 | 같음 |

검증 계획(구현 PR): 결과는 구현 PR 본문에 남깁니다. 각 서비스에서 `go test ./migrations/`, `scripts/check-service-migrations.sh <service>` 와, scratch MariaDB 12.3 및 MySQL 8.0 에서 `ALEMBIC_VENV=<venv> scripts/verify-service-migrations.sh <service>` 로 upgrade, downgrade, upgrade, 기존 데이터가 있는 DB 에서의 적용, 'Alembic head + 서비스 마이그레이션 전체' 와 '신규 빈 DB + 서비스 마이그레이션 전체' 의 덤프 diff 0 을 확인해 결과를 PR 본문에 남깁니다. 위 실행은 로컬 일회용 데이터베이스에서만 합니다. contact-manager 와 talk-manager 는 서비스 소유 마이그레이션이라 같은 PR 에 코드와 마이그레이션을 함께 담을 수 있습니다(`bin-dbscheme-manager` 변경이 없으므로 별도 선행 PR 이 필요 없음).

## 5. 비용 추정과 PR 분할안

규모는 설계 리뷰, 구현, 코드 리뷰 루프, 테스트, RST 와 서비스 docs 갱신을 포함한 인일 추정입니다(실측이 아니라 코드 구조에 근거한 추정이며 C3 는 스파이크 결과로 바뀔 수 있습니다).

| PR | 항목 | 변경 서비스 | 마이그레이션 | 인일 | 선행 |
|---|---|---|---|---|---|
| A-1 | A1 | contact-manager, api-manager, openapi-manager | 없음 | 2 | 없음 |
| A-2 | A2 | api-manager, openapi-manager | 없음 | 1 | 없음 |
| A-3 | A4 | contact-manager, api-manager, openapi-manager | contact `+1` | 3 | 없음 |
| A-4 | A5 | api-manager, openapi-manager | 없음 | 3 | 없음 |
| B-1 | B1 | talk-manager, api-manager, openapi-manager | talk `+1` | 4 | 없음 |
| B-2 | B2 | contact-manager, api-manager, openapi-manager | contact `+1` | 4 | 없음 |
| B-3 | B3 | talk-manager, api-manager, openapi-manager | talk `+1` | 4.5 | B-1 과 번호만 조율 |
| C-1 | C1 | api-manager, openapi-manager | 없음 | 3 ((b) 는 4) | 스파이크 S7 (취소 제외) |
| C-2 | C2 | api-manager, openapi-manager, call-manager(기존 결함 B, C, D 수정: 3.10) | 없음 | 5 | 없음 (P8 의 A 는 별도 선택) |
| C-3 | C3 | call-manager, api-manager, openapi-manager | 없음 | 9 ((b) 는 10) | C-1 의 소유자 검증 헬퍼, 스파이크 S1~S7 |
| D | C4, C5 | contact-manager 등 | 정의 확정 뒤 | 미산정 | 제품 정의 |

합계(D 제외): 그룹 A 9일, 그룹 B 12.5일, 그룹 C 17일(P2/P3 (a) 기준; (b) 는 19일), 총 약 38.5일((b) 는 40.5일). 여기에 병렬 PR 의 공통 파일 충돌 비용을 더해야 합니다: 10개 PR 이 모두 `bin-openapi-manager/gens/models/gen.go`, `servicehandler/main.go` 인터페이스와 `mock_main.go`, 추적되는 `bin-api-manager/docsdev/build/` 전체를 바꾸므로 머지 직전마다 main 을 가져와 재생성하고 충돌을 푸는 작업이 PR 당 약 0.5일(합 5일)이라 총 약 43.5일((b) 는 45.5일)로 봅니다. 개정 1 이전 견적은 C-1 4, C-2 3, C-3 10 이었고, (a) 에서는 C-1 이 1일 줄고(에이전트 조회, 주소 검사, TOCTOU 처리가 사라짐) C-2 가 2일 늘어(call-manager 기존 결함 B, C, D 와 RST) 합계가 같습니다. P8 의 선택(A-1 또는 A-2)은 위 합계에 포함하지 않았습니다(A-1 약 +0.5일, A-2 약 +1.5일). 완화책: 머지를 한 번에 하나씩 순차로 하고(앞 PR 머지 뒤 다음 PR 은 rebase 후 `go generate` 와 문서 clean 재빌드), 생성 파일 충돌은 손으로 합치지 않고 재생성으로만 해소합니다.

PR 분할 원칙: 한 항목은 한 PR 입니다. 모노레포이므로 서비스 변경과 그것을 노출하는 OpenAPI/api-manager 변경이 같은 PR 에 들어가며(생성 코드가 서로 의존), 그 안의 배포 순서는 2.4 의 "하위 서비스 먼저" 규칙을 각 PR 설명에 적어 지킵니다. 같은 서비스에 마이그레이션이 둘 이상인 경우(contact: A-3, B-2 / talk: B-1, B-3)는 먼저 머지되는 PR 이 `000002` 를 씁니다. 그룹 A, B 는 서로 독립이라 구현은 병렬로 할 수 있지만 머지는 위 규칙대로 순차이고, 그룹 C 는 C-1 -> C-3 순서만 고정입니다.

검증 메모: 각 항목은 api-validator 시나리오가 필요하며 외부 PSTN 비용이 드는 경로는 시나리오에 넣지 않습니다. C1, C3 는 내부 extension 간 통화만 사용합니다.

## 6. 열린 사항과 확인하지 못한 것

제품/정책 결정 대기: 아래 "남은 결정 사항" 목록(P2/P3, P4, P5, P6, P8)과 3.12 의 질문 1~9 입니다.

남은 결정 사항 (개정 1 이후):
- P2/P3: 주소 유형. (a) v1 은 `extension` 만(권고, 가장 단순), (b) `tel` 까지 허용(`tel` 형식 검증과 3자 통화의 PSTN 비용 누적 대책 추가).
- P4: A4 disposition 값 집합.
- P5: B3 기존 메시지 멘션 소급 여부.
- P6: C4, C5 제품 정의(3.12 질문 1~9).
- P8: call-manager 기존 결함 A(ringing 통화의 `recording_start`)를 고칠지와 방식(A-1 또는 A-2). C-2 의 필수 범위가 아님.
결정됨: P1(고지는 조직 규율, 서버 강제 없음), P7(소유 에이전트의 start/stop 허용, 감사 로그만).

추가로 정할 사항:
- B3 멘션 목록의 읽음/미읽음 상태와 알림이 필요한지.
- 에이전트가 시작한 녹음의 재생/다운로드 경로(현재 recordings API 는 Admin|Manager 전용).
- C1 attended 취소 의미(3.9, 스파이크 S6 이후).

확인하지 못한 것(코드 근거가 없거나 실행이 필요한 것):
- 각 대상 테이블의 실제 규모와 `ALGORITHM=INPLACE, LOCK=NONE` 이 `contact_cases`(STORED 생성 컬럼 보유)에서 두 엔진에 모두 허용되는지: 구현 PR 의 scratch 실측 항목.
- 3.11 의 스파이크 S1~S7: S1, S2, S3, S5, S7 은 개정 2 에서 실측 반영, S4(UI)와 S6(attended 취소)은 미수행. 이미 청취 중인 Insight 세션의 3자 동작과 등록 UA 레그의 음성 믹싱은 미검증.
- 에이전트 webphone 으로 직접 건 발신이 항상 confbridge 안에 들어가는지(C1, C3 의 전제 조건).
- 종료된 queuecall 의 `service_agent_id` 보존(설계는 진행 중 통화에만 의존).
- 클라이언트(monorepo-javascript) 쪽 UI 변경은 이 문서 범위가 아님.
