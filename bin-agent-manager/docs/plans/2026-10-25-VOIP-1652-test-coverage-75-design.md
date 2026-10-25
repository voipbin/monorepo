# VOIP-1652 bin-agent-manager 단위 테스트 커버리지 75% 이상(목표 76%) 설계

## 1. 티켓 검토 (이슈 분석)

판정: 유효. origin/main accc03e00 에서 재실측했고 카드의 수치와 일치합니다.

- 측정: `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...` (실패 0건, go.mod/go.sum 변경 없음). 수기 합계는 `gens/`, `mock_*.go`, `Code generated ... DO NOT EDIT` 헤더 파일을 제외하고 같은 블록 중복은 최대 count 로 합산합니다.
- 수기 74.23% (2221/2992 statement), 원본 66.14% (go test 가 출력하는 패키지별 %는 mock 과 생성 코드를 분모에 포함해 이 표와 다릅니다). 75% 는 2244, 76% 는 2274 statement 이므로 최소 +23, 권장(76%) +53 statement 입니다.
- 패키지별 현재값 (covered/total, %): cmd/agent-control 441/646 (68.3), cmd/agent-manager 1/103 (1.0), internal/config 57/65 (87.7), models/agent 35/36 (97.2), pkg/agenthandler 595/750 (79.3), pkg/cachehandler 60/61 (98.4), pkg/dbhandler 483/637 (75.8), pkg/listenhandler 473/601 (78.7), pkg/metricshandler 0/1, pkg/subscribehandler 76/92 (82.6).
- 최근 변경: VOIP-1647 이 cmd/agent-control 과 cachehandler 를 올렸고 이 경로의 열린 PR 은 없습니다(`gh pr list` 로 확인). 티켓의 전제(수기 합계가 75% 미만)는 유효합니다.
- 테스트하기 어려운 구조: cmd/agent-manager 의 main 과 runDaemon 은 DB, Redis, RabbitMQ 를 직접 연결하여 제외합니다(102 statement). 이 서비스는 cmd 제외 시에도 산술적으로 도달 가능합니다.
- 가장 큰 미커버: listenhandler/v1_agents.go 81 (요청 검증과 핸들러 오류 전파의 오류 경로, 블록 44개 81 statement), dbhandler/agent.go 79, cmd/agent-control 205, agenthandler/agent.go 62 와 db.go 59.
- 사실 확인: v1_agents.go 의 `len(uriItems) < N` 분기는 processRequest 의 정규식 라우터가 먼저 걸러 라우터를 통해서는 도달할 수 없고 processV1AgentsXxx 를 직접 호출해야 닿습니다. 일부 핸들러(IDPut, StatusPut, PasswordPut, TagIDsPut, AddressesPut, Login)는 핸들러 오류를 typed 오류 매핑 없이 400 으로, Reserve 계열은 500 으로 고정합니다. 이 동작은 현재 코드 그대로 고정 대상으로 삼고 변경하지 않습니다.

## 2. 계획 (verify 항목, 테스트 코드는 아직 없음)

컨벤션: `Test_` 이름, gomock 컨트롤러 이름 `mc`, testify 금지(`reflect.DeepEqual` + `t.Errorf`), table-driven, 명백한 합성 테스트 값(기존 `f1ba04b0-...` UUID 형식, `test-password` 류). 신규 파일을 쓰고 기존 테스트 파일은 고치지 않습니다(기존 케이스와 겹치지 않게 아래 표의 미커버 블록만 대상으로 합니다).

도달성: v1_agents.go 의 미커버 44블록(81 statement) 중 `json.Marshal` 실패 13블록(26 statement; 줄 66, 106, 162, 202, 246, 287, 328, 379, 420, 461, 502, 543, 589)은 정상적인 반환값은 항상 직렬화 가능하여 제품 코드 변경 없이는 도달 불가(비정상적인 시각 값을 주입해야만 닿는 경로를 포함)이므로 제외합니다. 도달 가능은 31블록 55 statement 이며 +53 목표 대비 마진이 2 뿐이므로 Part B 를 처음부터 필수로 합니다. 목록의 합은 31블록이어야 하며 구현 전 프로파일 대조로 확인합니다.

### Part A. pkg/listenhandler/v1_agents_validation_test.go (신규, 도달 가능 55 statement)

핸들러 14개를 `h.processV1AgentsXxx` 로 직접 호출합니다(라우터는 URI 형태를 먼저 걸러 이 분기에 닿지 않음). mock 에 EXPECT 를 두지 않은 케이스는 호출되면 gomock 이 실패하므로 호출 부재를 검증합니다. 호출이 있는 케이스는 `gomock.Any()` 를 ctx 외에는 쓰지 않고 id 와 인자를 정확히 일치시킵니다. 아래 블록은 줄 번호 기준(미커버 프로파일 값)입니다.

| 분류 | 블록(줄) | 케이스 |
|---|---|---|
| List 입력 | 30, 47, 54 | path 에 `%zz` 를 둔 URI(예: `/v1/agents%zz`, query 의 `%zz` 는 파싱 오류를 내지 않음)이면 error 반환(30). 본문 `{bad` 이면 400(47). 본문 `{"customer_id":"zzz"}` 처럼 UUID 로 변환 불가한 값이면 400(54). 모두 List 호출 없음. 54 의 입력은 RED 단계에서 실제 오류를 내는지 먼저 확인(`deleted` 에 문자열은 변환을 통과하므로 쓰지 않음) |
| 비 UUID 400 | 94(IDGet), 190(IDDelete) | `/v1/agents/not-a-uuid` 이면 400, 호출 없음 |
| URI 항목 수 부족 | 83, 179, 396, 519, 563, 607 | 해당 핸들러가 요구하는 항목 수보다 1개 적은 경계 URI(IDGet/IDDelete 는 `/v1/agents`, 나머지는 `/v1/agents/<유효 UUID>`)와 유효한 JSON 본문을 사용(본문이 잘못되면 unmarshal 400 과 구별되지 않아 `< N` 변이를 못 잡음). 호출 없음 |
| 본문 unmarshal 400 | 224, 275, 316, 357, 408, 449, 490, 575, 619 | 유효한 id URI 와 `{bad` 본문, 400, 호출 없음. 224 는 GetByCustomerIDAddressPost 이며 Post 의 기존 unmarshal 테스트와 겹치지 않음 |
| 핸들러 오류 전파 | 60(List), 100(IDGet) | typed NotFound 를 주면 404 이며 기대 응답을 `cerrors.ToResponse(ve)` 로 계산해 `reflect.DeepEqual` 로 비교. 일반 error 는 500, `dbhandler.ErrNotFound` 는 404 |
| | 240(GetByCustomerIDAddress) | 위와 같음 |
| | 281, 322, 373, 414, 455, 496 (Login, Addresses, IDPut, Status, Password, TagIDs) | 같은 typed NotFound 를 줘도 400 이고 DataType 이 `DataTypeVoipbinError` 가 아님(errorResponse 로 바뀌는 변이를 잡음) |
| | | 581(Reserve), 624(ReserveRelease) | 오류를 줘도 500 이며 Data 가 비어 있음. 두 핸들러는 UUID 검증이 없으므로 유효한 UUID 로 인자를 정확 일치시킴 |

기존 `Test_processV1AgentsIDDelete_error`, `Test_processV1AgentsIDPermissionPut_error` 가 IDDelete 와 IDPermissionPut 의 오류 매핑을 이미 검증하므로 중복 케이스는 넣지 않습니다. 줄 83 은 processV1AgentsIDGet 의 `len < 4` 블록입니다. 구현 시 프로파일 줄 번호와 대조해 케이스를 맞춥니다.

핸들러별 요청 형태: IDGet/IDDelete 는 `/v1/agents/<유효 UUID>`(4항목, `len < 4` 경계는 `/v1/agents`), IDPut 도 `/v1/agents/<유효 UUID>`(4항목), Status/Password/TagIDs/Addresses/Permission/Reserve/ReserveRelease 는 `/v1/agents/<유효 UUID>/<suffix>`(5항목, 경계는 `/v1/agents/<유효 UUID>`), Login 은 `/v1/agents/<username>/login`, Post 는 `/v1/agents`, GetByCustomerIDAddressPost 는 `/v1/agents/get_by_customer_id_address`. 형태가 틀리면 `len < N` 분기로 먼저 빠져 의도와 다른 블록을 커버합니다. List 케이스는 page_size 와 page_token 이 URI 에 있고 빈 본문이면 filters 는 nil 이 아닌 빈 map 이므로 `map[agent.Field]any{}` 로 `Eq` 를 맞춥니다(오류 전파 케이스의 본문은 유효한 필터 JSON 을 줍니다). 새 테스트 이름에는 접미사 `_validation` 을 붙여 기존 이름(`Test_processV1AgentsIDPut_nameLengthValidation` 등)과 겹치지 않게 합니다.

### Part B. pkg/agenthandler 오류 경로 (필수, 도달 가능 최대 82 statement)

실측 미커버: db.go 59, direct_hash.go 10, event.go 15 (합 84). event.go 104-106 의 `rand.Read` 실패 2 statement 는 도달 불가로 제외하여 82 statement 가 상한입니다. dbCreate 의 mock 호출 순서는 HashGenerate(password, defaultPasswordHashCost), UUIDCreate, DirectV1DirectCreate(ctx, customerID, ResourceTypeAgent, id), AgentCreate 이며 UUIDCreate 가 반환한 agent id 를 AgentCreate 의 에이전트와 AgentGet 에, DirectV1DirectCreate 가 반환한 direct id 를 DirectV1DirectDelete 에 일관되게 씁니다(두 id 는 서로 다른 UUID 로 둡니다)(ring method 오류 케이스만 HashGenerate 와 UUIDCreate 를 호출하지 않음). 기존 db_test.go 처럼 `agentHandler{utilHandler, reqHandler, db, notifyHandler}` 를 직접 구성하고 신규 파일 `db_error_test.go`, `direct_hash_error_test.go`, `event_error_test.go` 에 씁니다. 테스트 이름은 `Test_dbCreate_*`, `Test_EventGroupcallProgressing_*` 처럼 기존 이름(`Test_dbCreate_DoesNotLogDirectHash`, `Test_EventGroupcallAnswered` 등)과 겹치지 않게 합니다. metricshandler 전역 카운터는 값을 단언하지 않습니다.

함수별 케이스(호출하는 db 메서드명 기준; 구현 전에 미커버 프로파일 줄 번호와 대조해 표를 확정):

| 함수 | 케이스 |
|---|---|
| dbCreate (db.go 69-75, 80, 93, 122-147) | 지원하지 않는 ring method 이면 typed INVALID_RING_METHOD(`errors.As` 로 코드 단언), HashGenerate 와 DB 호출 없음. HashGenerate 실패(원 오류 전파). DirectV1DirectCreate 실패(`could not create direct hash` 래핑, AgentCreate 호출 없음). AgentCreate 는 어떤 오류로 실패해도 ErrAlreadyExists 분기보다 먼저 DirectV1DirectDelete 를 호출하므로 AgentCreate 실패 케이스 전부(orphan 삭제, ErrAlreadyExists, 그 밖의 오류)에 생성된 direct id 로 DirectV1DirectDelete EXPECT 를 둠(삭제 자체가 실패해도 원 오류 유지). ErrAlreadyExists 케이스는 `fmt.Errorf("...: %w", dbhandler.ErrAlreadyExists)` 처럼 래핑해서 주고(`errors.Is` 가 `==` 로 바뀌는 변이를 잡기 위함) 반환은 `errors.As` 로 VoipbinError 와 코드 ADDRESS_ALREADY_ASSIGNED 를 단언. 그 밖의 오류는 그대로 전파 생성 후 AgentGet 실패이면 오류 전파, 웹훅 발행 없음 |
| dbDelete (166-186) | 사전 AgentGet 실패(AgentDelete 호출 없음). direct 삭제 실패는 best-effort 이므로 AgentDelete 가 호출되고 최종 성공이 반환됨(DirectID 가 Nil 이면 삭제 호출 없음도 한 케이스). AgentDelete 실패. 삭제 후 AgentGet 실패 |
| dbLogin (202-208) | 사용자 없음 이면 `no agent info`, 비밀번호 불일치이면 `wrong password`(두 메시지가 서로 다름을 단언) |
| dbUpdateInfo (240-255) | 변경 필드 없음 + AgentGet 실패, AgentUpdate 실패, 갱신 후 AgentGet 실패(각각 이벤트 발행 없음) |
| dbUpdatePassword (271-284) | HashGenerate 실패, AgentSetPasswordHash 실패, 재조회 실패 |
| dbUpdatePermission, dbUpdateTagIDs (300-332) | AgentSetPermission/AgentSetTagIDs 실패와 재조회 실패 |
| dbUpdateAddresses (347-366) | AgentSetAddresses 가 래핑된 ErrAlreadyExists 이면 ADDRESS_ALREADY_ASSIGNED(`errors.As`), 일반 오류 전파, 재조회 실패 |
| dbUpdateStatus (382-390) | AgentSetStatus 실패, 재조회 실패 |
| DirectHashRegenerate | 시작 AgentGet 실패, Regenerate 실패(DirectID 있음), Create 실패(DirectID 없음), AgentUpdate 실패, 갱신 후 AgentGet 실패(65-67). 단계별 래핑 메시지(`could not get agent`, `could not regenerate direct hash`, `could not create direct hash`, `could not update agent`)와 `errors.Is`. 갱신 후 AgentGet 실패(65-67)만 래핑 없이 원 오류를 그대로 반환하므로 `==` 로 단언 |
| EventGroupcallProgressing | destination 이 agent 가 아닌 것, 잘못된 target, Get 실패, UpdateStatus 실패를 각각 포함하되 destination 을 2개 이상 두고 첫째가 실패하며 둘째가 정상 처리됨(h.Get 은 dbGet 의 AgentGet, h.UpdateStatus 는 dbUpdateStatus 의 AgentSetStatus(ctx, id, StatusBusy), 성공 경로의 재조회 AgentGet, PublishWebhookEvent(ctx, customerID, EventTypeAgentStatusUpdated, res) 를 정확한 id 로 호출하며 같은 id 의 AgentGet 이 2회이므로 `.Times(2)` 또는 `gomock.InOrder`. "UpdateStatus 실패" 는 첫 destination 에서 AgentSetStatus 가 실패하고 이후 재조회와 웹훅이 호출되지 않음)을 확인하고, 함수는 항상 nil 을 반환하므로 반환 오류가 nil 임도 단언(`continue` 를 `return nil` 로 바꾸는 변이를 잡기 위함). Get 실패는 일반 오류로 모킹 |
| EventCustomerDeleted | List 실패(래핑 오류, `utilHandler.TimeGetCurTime()` mock 필요), AgentList(ctx, 1000, TimeGetCurTime 반환값, {FieldCustomerID: cu.ID, FieldDeleted: false}) 를 인자 정확 일치로 EXPECT 하고, deleteForce(dbDelete 의 첫 AgentGet) 실패가 첫 agent 에서 나도(그 agent 에는 AgentDelete 호출 없음) 둘째 agent 의 AgentGet, DirectID 가 있으면 DirectV1DirectDelete, AgentDelete, 재조회 AgentGet(같은 id 에 AgentGet 이 2회이므로 `.Times(2)` 또는 `gomock.InOrder`), PublishWebhookEvent 를 모두 EXPECT 하고 계속 삭제됨을 확인 |
| EventCustomerCreated (128-130) | AgentGetByUsername 은 오류를 반환(신규 고객)하고 DirectV1DirectCreate 실패로 dbCreate 를 실패시킴(ring method 는 코드에서 고정이라 INVALID_RING_METHOD 로는 실패시킬 수 없음). `could not create basic customer admin agent` 래핑을 단언. 비밀번호는 rand 이므로 HashGenerate 의 password 인자만 `gomock.Any()` 로 허용하고 cost 는 `defaultPasswordHashCost` 로 정확히 일치시키며 UUIDCreate EXPECT 도 둠(Part A 의 "ctx 외 gomock.Any 금지" 규칙의 명시적 예외). 기존 EmailFails/AlreadyExists/Headless 케이스와 겹치지 않음 |

### Part C. 예비 (트리거: A+B 구현 뒤 §1 의 측정 명령과 수기 합계 절차로 2274 statement 미만일 때만)

A(55)+B(최대 82)가 모두 들어오면 137 이므로 트리거 가능성은 낮습니다. 수기 합계 절차는 scratch 의 `run.py`(gens/, `mock_*.go`, `Code generated ... DO NOT EDIT` 헤더 파일을 제외하고 같은 블록은 최대 count 로 합산)이며 PR 본문에 절차를 적습니다. 후보는 pkg/dbhandler/agent_reserve.go(미커버 32)와 agent.go 오류 경로(sqlmock; dbhandler 의 기존 `agent_reserve_test.go`, `withtx_test.go` 가 sqlmock 을 쓰는지 구현 시 먼저 확인), pkg/subscribehandler/main.go(미커버 10). 발동하면 이 문서에 결과를 갱신한 뒤 진행합니다.

### 변이 검사 (scratch 복사본에서만, 새 테스트 파일만 `-run` 으로 실행해 새 테스트가 죽이는지 확인)

- Part A: `len(uriItems) < 5` 를 `< 4` 로, IDGet 의 `uuid.Nil` 검사 제거, Reserve 의 500 을 400 으로, ReserveRelease 오류 분기를 200 으로, IDPut/Status/Login/Addresses/Password/TagIDs 각각의 400 을 errorResponse 로, Get/IDGet 의 errorResponse 를 400 으로, 필터 변환 400 을 500 으로, 본문 unmarshal 400 제거.
- Part B: dbCreate 의 orphan direct 삭제 호출 제거, dbCreate 와 dbUpdateAddresses 의 `ErrAlreadyExists` 변환 제거, dbLogin 의 두 오류 메시지 교환(비밀번호 반전은 기존 `Test_dbLogin` 도 죽이므로 쓰지 않음), dbDelete 에서 direct 삭제 오류 시 중단으로 변경 및 `DirectID != uuid.Nil` 가드 제거, dbUpdatePassword 와 dbUpdateStatus 의 오류 반환을 nil 로(함수별 1개씩), EventGroupcallProgressing 의 `continue` 4곳(agent 아님, uuid.Nil, Get 실패, UpdateStatus 실패)을 각각 `return` 으로(각 실패 사유마다 둘째 destination 이 처리되는 케이스가 있어야 함), dbUpdateInfo, dbUpdatePermission, dbUpdateTagIDs 의 오류 반환을 nil 로(함수별 1개씩), EventCustomerDeleted 의 deleteForce 실패 시 `continue` 를 `return` 으로.
- RED 확인: 구현 중 각 Part 의 테스트 중 하나는 기대값을 일부러 틀리게 해 실패함을 한 번 확인합니다.

최종 게이트: §1 의 측정 명령으로 수기 합계 76% 이상(2274 statement 이상), 전체 통과, lint, `scripts/check-test-conventions.sh` 통과, `git diff --stat origin/main...HEAD` 가 bin-agent-manager/ 안에만 있음, go.mod/go.sum 변경 없음. 합격 기준은 75% 이상이며 76% 이상을 목표로 합니다.

## 3. 범위와 위험

- 변경 파일은 bin-agent-manager 의 `*_test.go` 와 이 문서뿐입니다. 제품 코드 변경 없음.
- 잔여 위험: 핸들러 오류 매핑 불일치(400 고정과 typed 매핑)는 현재 동작의 고정이며 변경 제안은 카드 요약에만 적습니다.
- VOIP-1416 과 겹침: pkg/listenhandler, pkg/dbhandler 등이 80% 미만이면 겹칠 수 있어 PR 본문에 명시만 합니다.
