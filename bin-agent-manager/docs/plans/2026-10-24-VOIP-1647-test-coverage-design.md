# VOIP-1647 bin-agent-manager 단위 테스트 커버리지 60% 이상(목표 61%) 설계

## 1. 티켓 검토 (이슈 분석)

판정: 유효. 기준선은 origin/main d68e695b5 에서 재실측했고 기준선 수치와 일치합니다.

- 측정: `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...` (테스트 실패 0건).
  수기 합계는 gens/, `mock_*.go`, `Code generated ... DO NOT EDIT` 헤더 파일을 제외하고 같은 블록 중복은 최대 count 로 합산합니다.
- 수기 58.05% (1737/2992 statement), 원본 51.73% (1737/3358). 60% 는 1796, 61% 는 1826 statement 이므로 최소 +59, 권장 +89 statement 가 필요합니다.
- 아래 수치는 수기 기준이며 `go test` 가 출력하는 패키지 퍼센트(mock 포함)와 다릅니다.
- 패키지별 현재값 (covered/total, %): cmd/agent-control 23/646 (3.6), cmd/agent-manager 1/103 (1.0), internal/config 57/65 (87.7), models/agent 35/36 (97.2), pkg/agenthandler 595/750 (79.3), pkg/cachehandler 0/61 (0), pkg/dbhandler 477/637 (74.9), pkg/listenhandler 473/601 (78.7), pkg/metricshandler 0/1, pkg/subscribehandler 76/92 (82.6).
- 미커버의 대부분은 cmd/ 두 패키지(약 700 statement)입니다. cmd/agent-manager 의 main 과 runDaemon 계열은 RabbitMQ, Redis, DB 연결을 직접 만들어 단위 테스트가 어렵습니다. 제품 코드 변경 금지이므로 제외합니다.
- 달성 가능성: cmd/agent-control 에서 네트워크 없이 도달 가능한 코드만으로 약 +200 statement 가 확보되어 산술적으로 가능합니다(아래 3절).
- 최근 커밋/PR: 이 서비스에 대한 진행 중 PR 이 없음을 `gh pr list` 로 확인합니다. 범위 밖 사항은 이 작업에서 고치지 않습니다.

## 2. 테스트하기 어려운 구조 (확인 결과)

- `cmd/agent-control` 의 run* 함수는 검증 이후 `initHandler()` 로 실제 DB/Redis/RabbitMQ 에 접속합니다. 따라서 검증 실패 경로(입력 검증)까지만 테스트합니다.
- `pkg/cachehandler` 는 `*redis.Client` 를 직접 쓰며 miniredis 가 go.mod 에 없습니다. 새 의존성은 추가하지 않고, 테스트 파일 안의 loopback 최소 RESP 서버(GET/SET/DEL/PING)를 씁니다. 외부 네트워크 의존은 없습니다.

## 3. 계획 (verify 항목, 테스트 코드는 아직 없음)

컨벤션: `Test_` 이름, gomock 컨트롤러 이름 `mc`, testify 금지(`reflect.DeepEqual` + `t.Errorf`), table-driven. viper 전역은 테스트마다 `viper.Reset()` 을 `t.Cleanup` 으로 복구합니다.

1. cmd/agent-control/main_test.go
   - resolveUUID/resolveString: 빈 값, 잘못된 UUID, 정상 값 각각의 오류 메시지와 반환값.
   - printJSON: 직렬화 성공, 직렬화 불가 값(chan)에서 오류.
   - cmd* 생성자와 flag/viper 키 연결: 각 생성자에 대해 `viper.BindPFlags(cmd.Flags())` 후 `cmd.Flags().Set(<flag>, <잘못된 값>)` 을 주고 `cmd.RunE` 를 호출합니다. 빈 값은 viper 키가 어긋나도 `is required` 로 같은 오류가 나 배선을 검증하지 못하므로 쓰지 않습니다. ID 계열(id, customer-id)은 `not-a-uuid` 를 넣고 오류가 `is not a valid UUID` 를 포함하며 `is required` 가 아님을 단언, addresses 와 tag-ids 는 잘못된 JSON 으로 `failed to parse ...JSON` 을 단언합니다(키가 어긋나면 실패). 동작에 쓰이는 기본값만 단언(limit=100, dry-run=true, permission=0). 필수 여부는 단언하지 않음(MarkFlagRequired 없음). PersistentPreRunE 와 LoadGlobalConfig(sync.Once 전역)는 호출하지 않음.
   - run* 검증 실패 경로 (검증 통과 입력은 쓰지 않음). 함수별 실제 순서:
     runCreate(customer-id, username, password 순. customer-id 와 username 이 모두 비면 customer-id 오류가 먼저), runGet/runDelete/runUpdatePermission/runUpdateBasicInfo(agent ID 만), runList(customer ID 만), runUpdatePassword(id, password), runLogin(username, password), runUpdateStatus(id, status, `invalid status`), runUpdateAddresses(id, addresses, `failed to parse addresses JSON`), runUpdateTagIDs(id, tag-ids, `failed to parse tag IDs JSON`). 오류가 각 `invalid ...` 래핑이며 `failed to initialize handlers` 가 아님을 단언.
   - initCommand: 루트 직속 {agent, migrate}, agent 하위 12개 이름 집합(11개 CRUD/조작 + normalize-addresses). 한 테스트에서만 호출.
2. cmd/agent-control/normalize_addresses_scan_test.go (신규 파일, 기존 normalize_addresses_test.go 의 헬퍼 mustNorm 재사용)
   - countNonDeletedAgents: sqlmock 정상, Scan 오류(`could not scan count`).
   - scanAgents (dbhandler mock): 첫 호출 filters 가 `{FieldDeleted:false}`; 한 페이지(`len < pageSize`)이면 AgentList 정확히 1회; 다중 페이지에서 2번째 호출 token 이 리터럴 문자열과 같음(제품 코드와 같은 식으로 기대값을 계산하지 않음. tm_create 에 비UTC 오프셋 시각을 넣어 `UTC()` 제거와 layout 변경을 잡음); nil tm_create 에서 중단; 목록 오류 `could not list agents`; 충돌: 서로 다른 두 agent 가 같은 (customer,type,canonical) 를 가지면 충돌, 같은 agent 가 같은 target 을 두 번 가져도 충돌 아님, 다른 customer 의 같은 target 은 충돌 아님.
   - 페이지가 가득 찬 뒤 빈 페이지가 오는 경로(pageSize=2, agent 2개, 두 번째 호출 빈 목록)로 빈 페이지에서 종료됨을 확인(len==0 분기는 len < pageSize 분기와 결과가 같아 동작 보존 확인 용도).
   - addressesDiffer 길이 불일치, dedupeUUIDs 중복 분기, reportChanges(변경 없음 시 조기 반환, 한 agent 가 변경 없는 주소와 변경된 주소를 함께 가질 때 변경 없는 주소 생략), reportCollisions(충돌 2건 이상으로 정렬 순서와 내용), formatAddresses, dryRunLabel (stdout 캡처).
   - runNormalizeAddresses 는 접속 전 검증이 없어 제외(미검증 잔여 위험으로 PR 본문에 기록).
3. pkg/cachehandler/handler_test.go: 테스트 파일 안의 loopback 최소 RESP 서버(PING/GET/SET/DEL, 받은 명령과 인자를 기록).
   - AgentSet 은 `EX 86400`, PasswordResetTokenSet 은 `password_reset:` 접두와 전달한 ttl, Del 은 정수 응답.
   - 서버는 map 저장소로 동작: Delete 후 Get 이 `redis.Nil`(AgentDel, PasswordResetTokenDelete 모두), DEL 인자에 `password_reset:`+token 이 오는지 단언, 미존재 토큰의 PasswordResetTokenGet 은 `redis.Nil` 반환.
   - AgentGet 왕복, 서로 다른 두 ID 는 서로 다른 키, 미존재 GET 은 `redis.Nil`, 손상 JSON 은 오류와 nil 반환, PasswordResetTokenGet 정상과 잘못된 UUID 오류, Delete.
   - 서버가 `-ERR` 를 돌려줄 때 AgentSet/Get/Del 과 PasswordResetToken Set/Get/Delete 의 오류 전파, PasswordResetTokenDelete 정상. PasswordResetTokenSet 의 ttl 은 24h 와 다른 값(15분, `ex 900`)을 씁니다.
   - RESP 서버 위생: 연결마다 고루틴, 기록은 mutex, 명령 이름은 대소문자 무시 비교, Cleanup 에서 client 와 listener Close.
   - Connect: RESP 서버의 PING 이 +PONG 이면 성공, `-ERR` 이면 실패(포트 경쟁과 재시도 지연 없음). 이 케이스는 `NewHandler(서버 주소, 비밀번호, DB)` 경로로 만들어 NewHandler 도 함께 커버. 서버는 `127.0.0.1:0` 으로 열며 외부 접속은 없고 비밀번호는 빈 문자열(AUTH 불필요). Set/Get/Del 오류 케이스는 `&handler{Cache: redis.NewClient(&redis.Options{Addr:.., MaxRetries:-1})}` 로 구성.
4. pkg/dbhandler withTx(main.go:64-85): sqlmock 으로 BeginTx 오류(`could not begin transaction`, errors.Is 체인), 콜백 오류(원 오류 전파와 Rollback 호출), Commit 오류(`could not commit transaction`), panic(재-panic 과 Rollback). 각 케이스 `ExpectationsWereMet`.
5. Login 은 메트릭을 읽을 방법이 없고 새 의존성도 금지이므로 계획에서 제외합니다. processEventRun 도 제외합니다.

최종 게이트: §1 의 측정 명령(-race 없음)으로 수기 합계 61% 이상(1826 statement 이상), 전체 통과, `go test -count=3 ./cmd/... ./pkg/...` 로 stdout 캡처 안정성 확인, lint, `scripts/check-test-conventions.sh` 통과.
변이 검사(모두 scratch 복사본에서만, 실험 전 DSN/redis 주소를 사용하지 않는 loopback 포트로 주입하여 실제 접속 방지): resolve* 의 `val == ""`, scanAgents 의 `< pageSize` 와 커서 포맷과 `> 1`, withTx 의 Rollback 호출 제거, AgentSet TTL 변경, passwordResetKeyPrefix 제거, AgentDel/PasswordResetTokenGet 의 오류 반환 제거, addressesDiffer 길이 비교 제거.

## 4. 범위와 위험

- 변경 파일은 bin-agent-manager 의 `*_test.go` 와 이 문서뿐입니다. 제품 코드 변경 없음. 테스트 중 범위 밖 발견 사항은 수정하지 않습니다.
- 모든 run* 케이스는 검증 실패 입력만 사용하고(통과 입력은 initHandler 가 실제 DB/Redis 에 접속), 오류가 `invalid ...` 래핑으로 시작하며 `failed to initialize handlers` 가 아님을 단언합니다. runUpdateStatus 의 status 는 빈 값만 씁니다("foo" 는 검증을 통과해 접속으로 흐름).
- viper 전역과 AutomaticEnv(환경변수 USERNAME 등)에 대비해 각 run* 테스트 시작과 Cleanup 에서 `viper.Reset()` 을 호출합니다. config.Bootstrap 은 logrus 전역과 환경 바인딩을 건드리므로 initCommand 는 한 테스트에서만 호출합니다.
- stdout 캡처는 짧은 출력(64K 미만)만 대상으로 하고 `os.Stdout` 교체를 `t.Cleanup` 으로 복구하며 t.Parallel 과 섞지 않습니다.
- 구현 시 주의: addresses/tag-ids 배선 테스트는 유효한 UUID id 를 함께 넣어야 JSON 파싱 오류에 도달합니다. RESP 서버는 `EX` 인자를 대소문자 무시로 파싱합니다. `-count=3` 게이트는 viper 전역 복구 누수 확인도 겸합니다.
- 잔여 위험: username, password, status 같은 문자열 flag 는 유효한 값이 initHandler 로 흐르므로 flag 이름과 viper 키의 어긋남을 이 방식으로 잡지 못합니다.
- config.Bootstrap 은 viper.Reset() 으로 복구되지 않는 logrus 전역(formatter, 레벨)도 바꿉니다. nil tm_create 의 warning 이 stdout 에 찍히므로 캡처 범위에 포함합니다. dry-run 기본값 true 는 안전 기본값이라 단언합니다.
- initCommand 테스트의 Cleanup 에서도 `viper.Reset()` 을 호출합니다.
- 대상 제외: runNormalizeAddresses(접속 전 검증이 없고 재시도, 초과 집계, 충돌 게이트가 접속 뒤에 있어 미검증 잔여 위험으로 PR 본문에 기록), PersistentPreRunE, processEventRun(고루틴), Login(메트릭 읽기 불가), cmd/agent-manager.
- 수치 기준은 수기 합계(gens, mock, DO NOT EDIT 제외)이며 원본 기준 60% 는 별도로 +278 statement 가 필요합니다.
