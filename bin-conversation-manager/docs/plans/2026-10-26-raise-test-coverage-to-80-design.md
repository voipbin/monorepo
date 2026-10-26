# bin-conversation-manager 단위 테스트 커버리지 80% 달성 (VOIP-1657)

## 1. 티켓 검토와 이슈 분석
- 검토 결과: 유효. origin/main(161e9748) 에서 같은 명령으로 재측정한 수기 코드 커버리지는 79.65%(2885/3622)로 카드의 기준선과 같다. 열린 PR 과 머지된 PR 중 VOIP-1657 로 이 서비스를 다루는 것은 없다. 이전 상향(VOIP-1647, 1649, 1652)은 이미 머지되어 있고 이번 변경은 그 뒤의 잔여분이다.
- 측정: `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...` 로 패키지별 프로파일을 만들고, `gens/`, `mock_*.go`, `Code generated ... DO NOT EDIT` 파일을 빼고 같은 블록은 최대 count 로 합산한다(`-coverpkg` 는 쓰지 않는다).
- 목표: 80.0% 이상, 권장 80.5% 이상(수기 statement 3622개 기준 80% 는 2898개 이상이라 +13, 80.5% 는 2916개 이상이라 +31).
- 도달 가능 상한: 시그니처 변경 없이 도달 가능한 미커버 분기만으로 80.5% 를 넘길 수 있어 상한은 80% 이상이다(conversationhandler 미커버 76개 중 도달 불가는 아래 호출 지점 7개이고 도달 가능한 약 68개(statement 기준) 대비 필요한 것은 +31 이다). 상한 미달이 아니므로 block 하지 않는다.
- 테스트 중 제품 코드의 결함 후보를 발견해도 이 문서와 코드에는 쓰지 않고 고치지 않는다.

## 2. 범위와 제외
- 범위: bin-conversation-manager 안의 `_test.go` 추가뿐이다. 제품 코드, 시그니처, 노출 범위, go.mod 는 바꾸지 않는다.
- 제외(도달 불가): json.Marshal 실패 응답, dbhandler 의 squirrel ToSql 와 PrepareFields 오류, 설정 바인딩 오류(internal/config. 착수 후 실제 코드에서 도달 여부를 다시 확인하고 도달 가능하면 테스트한다), 항상 nil 을 반환하는 runExecuteModeAgent 의 호출 지점 오류 분기(hook.go 두 곳, event_webchat.go, message.go 각 한 곳), getExecuteMode 가 만들지 않는 ExecuteModeNone 과 알 수 없는 execute mode 분기(hook.go 두 곳, message.go), 서버 Run 의 고루틴, DB, Redis, RabbitMQ 연결이 필요한 생성자와 main(mock 만으로 호출되는 생성자는 제외하지 않는다).
- VOIP-1416 이 적은 패키지와의 겹침은 PR 본문에 명시만 하고 흡수하지 않는다.

## 3. 추가할 테스트와 verify 기준
1. conversationhandler(미커버가 가장 큼): Hook 의 URL 파싱 오류, 경로 길이 부족, 계정 조회 실패, line 과 whatsapp hook 실패, 지원하지 않는 계정 타입. HookVerify 의 URL 오류와 계정 조회 실패. line 과 whatsapp 결과의 agent 소유 대화는 flow 를 시작하지 않고 flow 소유 대화의 실패는 `could not run flow mode` 로 반환되며 불완전한 결과는 건너뛴다. message, whatsapp, webchat 의 flow 소스별 activeflow 생성 실패 전파. setVariables 의 메시지 변수 설정 실패 전파(대화 변수 실패 시 메시지 변수를 시도하지 않는 경로는 기존 테스트가 다룬다). List, Create, Update, UpdateMetadata, CreateAndExecuteFlow 의 db 오류와 읽기 실패 경로(실행 실패는 생성된 대화를 유지). EmailEventSent 의 중복 조회 실패와 대화 조회 실패(수신자 2명 이상에서 메시지 생성 실패를 포함해 첫째가 실패해도 둘째가 처리되고, 두 수신자가 모두 실패하면 반환값은 첫 오류임을 단언), EmailEventUpdated 의 조회 실패. MessageEventReceived, MessageEventSent, eventSMS, eventWebchat 의 대화 생성 실패, 메시지 생성 실패, 상태 갱신 성공과 실패(outbound 에서 기존 메시지가 있는 경우), agent 소유 시 flow 미실행.
2. (1항만으로 80.5% 에 도달하면 2, 3항은 하지 않는다) dbhandler 는 ExecContext, QueryContext, scan 오류, 캐시 갱신 오류, ErrNotFound, 영향 행 0건, 빈 토큰 분기처럼 기존 sqlmock 패턴(ctx_rows_test.go)으로 닿는 분기를 예비 풀로 둔다. whatsapphandler, messagehandler, smshandler, subscribehandler, linehandler: 계정과 대화 조회 실패, 전송 실패, 입력 오류 등 외부 호출 없이 mock 으로 도달하는 오류 분기. 착수 후 미커버 블록 중 위 제외에 들지 않는 것만 고른다.
3. (1항만으로 80.5% 에 도달하면 2, 3항은 하지 않고, 부족할 때만 필요한 만큼 추가한다) listenhandler 와 cmd: Run 은 큐 생성 실패만(성공 경로는 고루틴을 띄우므로 제외), processRequest 의 미지원 URI, 오류 응답 매핑처럼 mock 으로 닿는 분기, cmd 의 DB 와 Redis 연결 없이 실행되는 함수만 선택한다.
- listenhandler 와 subscribehandler 에는 파싱된 요청 payload 에 `gomock.Any()` 를 쓰지 않는다(docs/conventions/testing.md). 이미 커버된 시나리오(hookLine 의 flow 오류와 불완전 결과 건너뛰기 등. message.go 의 대화 조회 실패와 메시지 생성 실패는 미커버이므로 대상이다)는 statement 가 늘지 않으므로 중복해서 쓰지 않는다. NewConversationHandler 는 mock 만으로 호출 가능하다.
- 모든 테스트는 합성값, gomock, 외부 상태와 시간에 의존하지 않는 입력만 쓰고, 호출 여부는 mock 의 기대 호출 수로 검증한다(호출이 없어야 하는 경우는 기대를 설정하지 않아 호출 시 실패하게 한다). 오류 경로는 bool 이 아니라 오류 메시지 부분 문자열(예: `could not run flow mode`)이나 `errors.Is` 로 원인을 단언하고, 성공 경로는 반환값을 단언한다. 오류 메시지에 map 이 들어가는 경우는 부분 문자열만 비교하고, sqlmock 의 시간 인자는 `sqlmock.AnyArg` 로 받고 sqlmock 테스트는 `ExpectationsWereMet` 로 기대 미충족을 실패로 만든다.
- verify: 측정 직전 origin/main 이 161e9748 에서 움직였으면 기준선을 재측정한다. `-race` 는 CGO 가 필요하다. 측정은 1절과 같은 명령과 제외 규칙으로 한다. `git fetch origin main` 을 먼저 실행한 뒤 `go mod tidy`, `go mod vendor`, `go generate ./...`(mock 파일 diff 없음 확인), `go test -count=1 -race`, 추가한 패키지 전체에 `-count=3 -shuffle=on` 반복이 통과하고, 수기 합계 80.5% 이상(PR 본문에 제외 규칙, 같은 블록은 최대 count 로 합산한다는 방식, 재현 가능한 계산 명령을 적는다), `go vet`, `golangci-lint run`, `scripts/check-test-conventions.sh` 통과, go.mod 와 go.sum 불변, `git diff --stat origin/main...HEAD` 가 이 서비스 안의 `_test.go` 와 이 문서뿐임.
- 변이 표본(scratch 복사본에서 수행): 변이마다 먼저 기존 테스트만으로 통과함을 확인하고, 신규 테스트가 실패해야 하며 실패한 신규 테스트 이름을 기록한다. 대상: hookWhatsApp 의 agent 분기 제거(hook.go:89-92), eventWebchat 의 agent 분기를 flow 호출로 대체(agent 소유 webchat 테스트에는 위젯 조회와 activeflow 기대가 없어 gomock 이 잡는다), flow 오류 전파 제거(hook.go:94-96 의 hookWhatsApp, event_webchat.go:140-142, execute_mode.go 의 106, 122, 147 줄의 activeflow 오류 래핑 반환을 nil 반환으로. hookLine 의 hook.go:164-166 은 기존 테스트가 잡으므로 대상이 아니다), CreateAndExecuteFlow 의 실행 실패(create_and_execute_flow.go:127-129)를 오류로 반환(생성 실패와 변수 설정 실패도 생성된 대화를 유지하고 nil 오류임을 단언), EmailEventSent 의 continue(email.go:66, :87, :114)와 EmailEventUpdated 의 continue(email.go:157)를 각각 return 으로 변경(첫째 수신자만 실패하고 둘째는 정상 처리되며 두 수신자의 transaction id 가 달라 기대를 구분하는 테스트), EmailEventSent 의 `firstErr == nil` 가드 제거(두 수신자 모두 실패하는 테스트), EmailEventSent 의 중복 조회 오류 무시, setVariablesMessage 의 오류 무시.

## 4. 롤백
테스트 전용 PR 이므로 squash 커밋의 `git revert` 로 충분하다. 런타임, DB, 배포에 영향이 없고, 신규 테스트가 CI 에서 흔들리면 그 테스트만 수정하거나 제거하되, 제거로 합계가 80% 아래로 내려가면 대체 테스트를 추가하거나 PR 전체를 되돌린다.
