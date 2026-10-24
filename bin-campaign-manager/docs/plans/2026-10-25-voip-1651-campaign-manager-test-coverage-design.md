# VOIP-1651 bin-campaign-manager 단위 테스트 커버리지 상향 (설계)

## 1. 이슈 분석 (티켓 검토)

- 지표 정의: statement 기준 커버리지이며 gens/, mock_*.go, `Code generated ... DO NOT EDIT` 파일을 제외하고 같은 블록은 최대 count 로 합친 값이다. 기준 커밋은 origin/main 99b59c386 이며 PR 직전 `git fetch origin main` 으로 이동 여부를 다시 확인한다.
- 티켓 검토: 유효. 기준선은 origin/main 99b59c386 에서 `GOFLAGS=-mod=mod go test -count=1 -coverprofile` 로 재측정했고, gens/ 와 mock_*.go 와 `DO NOT EDIT` 파일을 제외하고 같은 블록은 최대 count 로 합쳐 수기 66.0% (2313/3503), 원본 56.9% 였다. 테스트 실패 0건.
- 70% 에는 140 statement(2453-2313), 권장 71% 에는 175 statement 를 더 덮어야 한다. 목표는 약 250 statement 이상(수기 73% 안팎)이다.
- 패키지별 전 수치(수기 statement 기준): cmd/campaign-control 0/328, cmd/campaign-manager 0/144, internal/config 93/97, models/campaign 18/26, models/campaigncall 19/27, models/outplan 16/26, pkg/cachehandler 0/67, pkg/campaigncallhandler 200/265, pkg/campaignhandler 638/897, pkg/dbhandler 540/636, pkg/listenhandler 626/782, pkg/outplanhandler 93/121, pkg/subscribehandler 70/87.
- cmd/ 의 main 두 개(472 statement)는 프로세스 기동과 외부 연결(DB, Redis, RabbitMQ)에 묶여 있어 단위 테스트 대상에서 제외한다. 이 둘을 빼도 분모 3503 은 그대로이고 cmd 를 제외한 미커버 statement 가 718개라 필요한 175개의 약 4배이므로 70% 달성은 산술적으로 가능하다.
- 큰 미커버: campaignhandler/execute.go 107 (Execute 오류 분기와 executeCall 전체), campaignhandler/campaign.go 104 (오류 경로와 리소스 검증 분기), listenhandler campaigns/outplans/campaigncalls 126, campaigncallhandler 48, outplanhandler 24, status_stop.go 24, dbhandler 약 96.
- 패키지 전역 상태: 전역 변수는 prometheus 카운터이며 패키지 init 에서 한 번 등록되므로 테스트 간 중복 등록이나 순서 의존은 없다(campaigncallhandler/main.go, campaignhandler/main.go 확인).
- 최근 변경 이력과 열린/머지 PR 조회 결과 이 서비스에 대한 커버리지 PR 은 없다(중복 없음).
- 테스트하기 어려운 구조: 핸들러는 인터페이스 주입(gomock 가능)이라 구조적 장애물은 없다. cachehandler 는 redis 클라이언트 직접 사용으로 별도 도구가 필요해 우선순위를 낮춘다.

## 2. 계획

범위는 `bin-campaign-manager/**/*_test.go` 신규 파일과 이 문서뿐이다. 제품 코드 변경 없음, go.mod 변경 없음. 신규 테스트 파일은 기존 `_test.go` 와 이름이 겹치지 않게 `<소스>_errors_test.go` 형태로 만들고, scripts/check-test-conventions.sh 규칙(Test_ 이름, gomock 컨트롤러 이름 mc, 추가된 줄 기준)을 따른다. 기존 테스트가 이미 덮는 분기는 다시 만들지 않으며, 구현 직전에 `go tool cover -html` 대신 프로파일 블록 목록으로 미커버 분기를 다시 대조한다.

### 2.1 분기 표 (미커버 statement 는 프로파일 집계, 확정 = 71% 에 필요한 175 를 채우는 몫, 여유 = 일부 포기해도 71% 가 유지되도록 하는 몫)

| 구분 | 파일 | 대상 분기 | 예상 statement |
|---|---|---|---|
| 확정 | campaignhandler/execute.go | Execute: 캠페인 조회 실패, 상태 not run, outplan 조회 실패, 대상 조회 실패, 대상 없음 + end handle stop, 대상 없음 + 계속(5000), 발신 불가 + 재예약 실패(5000), 재시도 간격 미도래(5000), 목적지 없음, 실행 오류, 후속 execute(500) 실패. isDialable 의 에이전트 조회 오류와 dialing 조회 오류와 용량 부족 분기(QueueV1QueueGetAgents 와 ListByCampaignIDAndStatus mock 필요), getTarget 오류 분기 | 약 60 |
| 확정 | campaignhandler/execute.go | executeCall 전체(성공, campaigncall 생성 실패, call 생성 실패 시 Done(ResultFail)), Execute 의 TypeCall 경로, executeFlow 의 activeflow 생성 실패와 실행 실패 분기와 Done(ResultFail) 호출 | 약 38 |
| 확정 | campaignhandler/campaign.go | Get 의 NotFound 변환(코드 CAMPAIGN_NOT_FOUND)과 기타 오류 전달, Create 오류 경로, Delete 오류 경로, UpdateBasicInfo 와 UpdateNextCampaignID 와 UpdateServiceLevel 의 단계별 오류, UpdateResourceInfo 와 UpdateActions 의 단계별 오류, createFlowActions 의 queue 추가 분기, updateReferencedResources 오류 | 약 70 |
| 확정 | campaignhandler/campaign.go | validateResources 와 isValidOutdialID, isValidOutplanID, isValidQueueID, isValidNextCampaignID 의 조회 오류, 고객 불일치, 삭제됨, 타 캠페인 사용 중(outdial) 거부 | 약 30 |
| 확정 | campaignhandler/status_stop.go, status_run.go | UpdateStatus 의 unsupported 상태 거부와 stop 위임, campaignStopping 과 campaignStopNow 와 isStoppable 의 오류 분기, campaignRun 의 오류 경로 | 약 35 |
| 확정 | outplanhandler/outplan.go | Get 의 NotFound 변환(OUTPLAN_NOT_FOUND)과 기타 오류, Delete 의 단계별 오류, UpdateBasicInfo 의 단계별 오류 | 약 24 |
| 여유 | campaigncallhandler | Get 의 NotFound 변환(CAMPAIGNCALL_NOT_FOUND), GetByReferenceID, GetByActiveflowID, List 와 ListByCustomerID 와 ListByCampaignID 와 ListByCampaignIDAndStatus 와 ListOngoingByCampaignID 의 성공과 오류, Done 과 Progressing 의 오류 분기 | 약 50 |
| 조건부 | dbhandler | 캐시 오류 분기 약 96. 확정과 여유를 구현한 뒤 측정 합계가 71% 미만이면 기존 dbhandler 테스트 방식(ctx_rows_test.go 의 rows 주입, go-sqlmock)을 따라 추가 | 약 96 |
| 조건부 | listenhandler | 오류 응답 분기(campaigns, outplans, campaigncalls 합 약 126, 패키지 전체 미커버는 156). 위 조건부와 같은 조건 | 약 126 |
| 제외 | cmd/ 두 main(472), cachehandler(67, redis 직접 연결), NewXxxHandler 생성자(약 4 ~ 6 씩), models/*/webhook.go 는 값이 싸므로 여유 시 포함 | | |

확정 항목 합계는 약 60+38+70+30+35+24=257, 여유(campaigncallhandler) 약 50 을 더하면 이론 최대 약 307 이다. 수율 70% 를 가정하면 확정만으로 약 180 이며 71% 에 필요한 175 를 겨우 채우므로 여유 항목까지 구현하는 것을 기본으로 하고, 70% 에 필요한 140 에는 충분한 여유가 있다. 구현 후 측정에서 71% 미만이면 조건부 항목을 추가한다.

### 2.2 Execute 오류 분기의 mock 방침

Execute 의 모든 오류 분기는 `updateExecuteStop` 을 호출한다. 이 체인(`db.CampaignUpdateExecute`, `campaignStop`, `isStoppable` 의 `h.Get` 과 `ListOngoingByCampaignID`, `campaignStopNow` 또는 `campaignStopping` 의 `CampaignGet` 과 `CampaignUpdateStatus` 과 `PublishWebhookEvent`)의 mock 기대는 테스트 파일 안의 헬퍼로 두되, 분기마다 단순화를 위해 `CampaignUpdateExecute` 가 오류를 반환하는 경우(체인 조기 종료)와 정지 체인 전체가 호출되는 경우를 각각 한 개 이상 포함한다. 정지 체인에서 `db.CampaignGet` 은 한 오류 분기에서 최대 4회(Execute, isStoppable, campaignStopNow, 갱신 후 조회) 호출되고 isStoppable 은 캠페인의 Execute 가 ExecuteStop 인 갈래에서만 `util.TimeGetCurTime()` 과 `ListOngoingByCampaignID` 를 호출하므로 이 mock 은 그 갈래에서만 둔다. campaignStopNow 는 상태가 이미 stop 이면, campaignStopping 은 stop 또는 stopping 이면 상태 갱신 없이 조기 반환하므로, 갱신 분기를 타려는 테스트는 캠페인 상태를 stop 이 아닌 값(예: run)으로 잡는다. isStoppable 이 true 가 되려면 조회된 캠페인의 Execute 가 ExecuteStop 이어야 하며 campaignStopNow 갈래와 아니면 campaignStopping 갈래를 모두 쓴다. Execute 의 첫 Get 이 성공한 뒤 정지 체인의 Get 이 실패하는 분기에서는 Get 이 최대 3회(Execute, isStoppable, campaignStopping) 호출되며 실패하도록 Times 와 gomock.InOrder 로 호출 횟수와 순서를 고정한다. Execute 는 반환값이 없으므로 관찰 가능한 결과는 mock 호출 시퀀스뿐이며, 분기마다 구별되는 호출 시퀀스(예: 대상 없음 + 계속은 지연 5000 의 CampaignV1CampaignExecute 한 번, 정상 종료는 지연 500)로 검증한다. "후속 mock 호출 없음" 은 오류 분기에서 정지 체인 외의 후속 호출(발신, campaigncall 생성)에 EXPECT 를 두지 않는 의미로만 사용한다.

### 2.3 verify 항목

- Execute 분기별: CampaignV1CampaignExecute 의 지연값 5000 과 500 구분, Done 의 campaigncall.ResultFail, FlowV1ActiveflowCreate 인자, campaigncall.Create 인자(목적지, 목적지 인덱스, 시도 횟수)를 mock 인자로 정확히 지정한다.
- Get 계열: errors.As 로 cerrors 타입과 코드(CAMPAIGN_NOT_FOUND, OUTPLAN_NOT_FOUND, CAMPAIGNCALL_NOT_FOUND)를 확인하고, 기타 오류는 원본 오류가 그대로 전달됨(errors.Is)을 확인한다.
- 검증 함수: 고객 불일치, 삭제됨, 타 캠페인 사용 중이 각각 false 이고 정상값이 true 임을 같은 표에서 대조한다.
- executeCall 의 UUIDCreate 는 callID, activeflowID 순으로 두 번 호출되므로 서로 다른 UUID 를 순서대로 반환하게 하여 CallV1CallCreateWithID 와 campaigncall.Create 의 인자 위치가 뒤바뀌면 실패하도록 한다.
- 재시도 간격 미도래 테스트는 TryInterval 을 밀리초 값(예: 3600000)으로 크게 잡고 TMUpdate 를 TMCreate 와 다른 값(현재 시각 직전)으로 두어 한 번도 시도하지 않은 분기로 빠지지 않게 한다. 경계에서 충분한 여유를 두어 비결정성을 피한다.
- 모든 신규 테스트는 `-race -count=1` 로 통과해야 한다.

### 2.4 변이 검사 표본(변이 -> 실패해야 하는 테스트)

1. execute.go 의 `c.Status != campaign.StatusRun` 반전 -> Execute 상태 not run 테스트.
2. EndHandleStop 비교 반전 -> 대상 없음 두 테스트.
3. 지연 상수 5000 을 500 으로 변경(사이트별 개별 변이: 대상 없음 + 계속, 발신 불가 + 재예약, 재시도 간격 미도래) -> 각 사이트의 테스트.
4. Done(ResultFail) 호출 제거(executeCall 과 executeFlow 의 호출 사이트 각각) -> call 생성 실패, activeflow 생성 실패, activeflow 실행 실패 테스트.
5. 고객 불일치 비교 제거(isValidOutdialID, isValidOutplanID, isValidQueueID, isValidNextCampaignID 개별) -> 각 고객 불일치 테스트.
6. NotFound 변환 제거(campaign, outplan, campaigncall 개별) -> 각 Get NotFound 테스트.
7. isStoppable 의 Execute 비교 반전, campaignStopNow 와 campaignStopping 의 상태값 교체 -> status_stop 테스트.
8. UpdateStatus 의 unsupported 거부 제거 -> unsupported 테스트.
9. campaigncall List 계열의 필터 인자 변경 -> ListBy 테스트.
10. isValid* 4개 함수의 삭제됨 검사 제거와 isValidOutdialID 의 타 캠페인 사용 중 검사 제거(각각 개별) -> 해당 거부 테스트.
11. campaign.go 의 Create, Delete, Update* 단계별 오류 전파(`return err` 를 `return nil` 로)와 outplan.go 의 Delete, UpdateBasicInfo 오류 전파 -> 해당 오류 테스트.
12. status_run.go 의 오류 경로 제거와 ExecuteRun 값 교체 -> campaignRun 테스트.
13. isDialable 용량 경계 `<=` 를 `<` 로 -> 용량 경계 테스트(동일 값 케이스 포함).
14. Execute 의 각 updateExecuteStop 호출 제거(조회 실패, outplan 조회 실패, 대상 조회 실패, 목적지 없음, 후속 execute 실패 사이트별), isDialable 과 isDialableTarget 의 `!` 반전, TypeCall 과 TypeFlow switch 교체 -> 해당 분기 테스트.
15. isDialable 의 에이전트 조회 오류와 dialing 조회 오류의 `return false` 를 `true` 로 -> 해당 오류 테스트.
16. Get 의 기타 오류 전달(원본 오류 errors.Is), createFlowActions 의 queue 분기, updateReferencedResources 오류, UpdateStatus 의 stop 위임 제거 -> 해당 테스트.
17. campaigncallhandler 의 Done 과 Progressing 오류 전파, GetByReferenceID 와 GetByActiveflowID 오류 전달 -> 해당 테스트.
조건부 행(dbhandler, listenhandler)을 구현하면 각 행에 최소 1개의 변이 표본을 추가한다.
각 변이는 scratch 복사본에서만 적용하고 결과를 PR 본문에 기록한다.

### 2.5 검증 절차

서비스 디렉터리에서 go mod tidy, go mod vendor, go generate ./..., go test -race -count=1 ./..., golangci-lint run -v --timeout 5m, 저장소 루트에서 `git fetch origin main` 후 scripts/check-test-conventions.sh 를 실행하고, 분석 스크립트와 동일한 방법으로 수기 합계를 재측정해 PR 본문 표의 전후 열로 쓴다. 금지 사항(빈 호출 테스트, 자기비교 테스트, 비밀값, 내부 정보, 외부 비용 호출)은 지킨다.
