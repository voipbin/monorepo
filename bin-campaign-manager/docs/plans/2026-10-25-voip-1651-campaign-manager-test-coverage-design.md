# VOIP-1651 bin-campaign-manager 단위 테스트 커버리지 상향 (설계)

## 1. 이슈 분석 (티켓 검토)

- 지표 정의: statement 기준 커버리지이며 gens/, mock_*.go, `Code generated ... DO NOT EDIT` 파일을 제외하고 같은 블록은 최대 count 로 합친 값이다. 기준 커밋은 origin/main 99b59c386 이며 PR 직전 `git fetch origin main` 으로 이동 여부를 다시 확인한다.
- 티켓 검토: 유효. 기준선은 origin/main 99b59c386 에서 `GOFLAGS=-mod=mod go test -count=1 -coverprofile` 로 재측정했고, gens/ 와 mock_*.go 와 `DO NOT EDIT` 파일을 제외하고 같은 블록은 최대 count 로 합쳐 수기 66.0% (2313/3503), 원본 56.9% 로 카드 수치와 일치했다. 테스트 실패 0건.
- 70% 에는 140 statement(2453-2313), 권장 71% 에는 175 statement 를 더 덮어야 한다. 목표는 약 250 statement 이상(수기 73% 안팎)이다.
- 패키지별 전 수치(수기 statement 기준): cmd/campaign-control 0/328, cmd/campaign-manager 0/144, internal/config 93/97, models/campaign 18/26, models/campaigncall 19/27, models/outplan 16/26, pkg/cachehandler 0/67, pkg/campaigncallhandler 200/265, pkg/campaignhandler 638/897, pkg/dbhandler 540/636, pkg/listenhandler 626/782, pkg/outplanhandler 93/121, pkg/subscribehandler 70/87.
- cmd/ 의 main 두 개(472 statement)는 프로세스 기동과 외부 연결(DB, Redis, RabbitMQ)에 묶여 있어 단위 테스트 대상에서 제외한다. 이 둘을 빼도 분모 3503 은 그대로이고 cmd 를 제외한 미커버 statement 가 718개라 필요한 175개의 약 4배이므로 70% 달성은 산술적으로 가능하다.
- 큰 미커버: campaignhandler/execute.go 107 (Execute 오류 분기와 executeCall 전체), campaignhandler/campaign.go 104 (오류 경로와 리소스 검증 분기), listenhandler campaigns/outplans/campaigncalls 126, campaigncallhandler 48, outplanhandler 24, status_stop.go 24, dbhandler 약 96.
- 패키지 전역 상태: 전역 변수는 prometheus 카운터이며 패키지 init 에서 한 번 등록되므로 테스트 간 중복 등록이나 순서 의존은 없다(campaigncallhandler/main.go, campaignhandler/main.go 확인).
- 최근 변경 이력과 열린/머지 PR 조회 결과 이 서비스에 대한 커버리지 PR 은 없다(중복 없음).
- 테스트하기 어려운 구조: 핸들러는 인터페이스 주입(gomock 가능)이라 구조적 장애물은 없다. cachehandler 는 redis 클라이언트 직접 사용으로 별도 도구가 필요해 우선순위를 낮춘다.
- VOIP-1416 겹침: campaignhandler, campaigncallhandler, outplanhandler, dbhandler 는 80% 미만 패키지로 겹칠 수 있어 PR 본문에 명시만 한다(흡수하지 않음).

## 2. 계획

범위는 `bin-campaign-manager/**/*_test.go` 신규 파일과 이 문서뿐이다. 제품 코드 변경 없음, go.mod 변경 없음(미사용 시).

우선순위와 verify 항목(테스트는 구현 단계에서 작성):
1. campaignhandler/execute: Execute 의 각 종료 분기(캠페인 조회 실패, 상태 not run, outplan 조회 실패, 대상 조회 실패, 대상 없음 + end handle stop/계속, 발신 불가 + 재예약 실패, 대상 재시도 간격 미도래, 목적지 없음, 실행 오류, 후속 execute 요청 실패), executeCall 성공과 campaigncall 생성 실패와 call 생성 실패 시 Done(fail) 호출, executeFlow 의 activeflow 생성 실패와 실행 실패 시 Done(fail) 호출.
   verify: 각 분기에서 gomock 의 호출 순서와 인자가 맞고, 호출되지 않아야 할 후속 mock 은 EXPECT 가 없어 호출 시 실패한다.
2. campaignhandler/campaign, status_run, status_stop: Get 의 NotFound 변환(cerrors 타입 확인)과 기타 오류 전달, UpdateResourceInfo 와 UpdateActions 의 단계별 오류, isValid* 의 오류와 고객 불일치와 삭제됨과 타 캠페인 사용 중 거부, UpdateStatus 의 지원하지 않는 상태 거부와 stop 위임, campaignRun 의 오류 경로.
3. outplanhandler / campaigncallhandler: Get 의 NotFound 변환, Delete 와 UpdateBasicInfo 의 각 단계 오류, campaigncall List 의 성공과 오류.
4. 부족 시 listenhandler 오류 응답과 dbhandler 캐시 오류 분기, GetCurTimeAdd.
5. 제외: cmd/ main, cachehandler(외부 연결), NewXxxHandler 생성자 중 의미 없는 호출 테스트.

금지 사항 준수: 빈 호출 테스트와 자기비교 테스트를 만들지 않고, 값 검증은 기대값을 독립적으로 적는다. 테스트 값은 합성값이다. 결함 발견 시 고치지 않고 카드 코멘트에만 남긴다.

검증 절차: 서비스 디렉터리에서 go mod tidy, go generate, go test, golangci-lint 실행, 커버리지 재측정(분석 스크립트 동일 방법), scripts/check-test-conventions.sh 확인, 변이 검사 표본(Execute 분기 조건 반전, Get 의 NotFound 변환 제거, Done(fail) 호출 제거)으로 새 테스트가 실패하는지 확인.
