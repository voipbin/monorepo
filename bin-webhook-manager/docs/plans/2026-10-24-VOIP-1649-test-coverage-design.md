# VOIP-1649 bin-webhook-manager 단위 테스트 커버리지 65% 이상 설계

## 1. 티켓 검토 (이슈 분석)

- 판정: 유효. origin/main 1dd36136d (VOIP-1647 #1515 머지 포함, 이전 측정 bffdd69e3 과 수치 동일) 기준 수기 코드 64.39% (669/1039), 원본 57.18%.
- 수기 65% 에는 676 stmt(+7), 권장 66% 에는 686 stmt(+17)가 필요.
- 최근 변경: 이 경로의 마지막 커버리지 작업은 VOIP-1647 (#1515, 머지됨). 열린 중복 PR 없음.
- 패키지별 미커버 수기 stmt: cmd/webhook-control 164, cmd/webhook-manager 137, listenhandler 22, subscribehandler 17, activeflowhandler 8, webhookhandler 17, config 3, 그 외 각 1.
- 이번 범위에서 제외: 두 cmd 의 main() 은 RabbitMQ, DB, 시그널 연결이 섞여 있어 비용 대비 효과가 낮고(합계 301 stmt 미커버), 나머지 도달 가능분만으로 목표를 넘기므로 이번 범위에서 제외함. cmd 안의 작은 함수(resolveUUID, printJSON, initCommand, cmdSendTo*, runListen, runSubscribe 등)는 main 패키지 테스트가 필요하며 부족할 때에만 쓰는 여유분 후보로 둠.
- 도달 불가(근거: 코드 읽기): listenhandler.processRequest 의 오류 매핑 블록(194~204줄, 7 stmt)은 processV1Webhooks* 의 모든 return 이 nil 오류라 라우트로 도달할 수 없으므로 시도하지 않음. listenhandler 미커버 22 = Run 9 + 오류 매핑 블록 7(제외) + errorResponse 의 ToResponse 실패 분기 2 + v1_webhooks.go 의 json.Marshal(req.Data) 오류 분기 4(37~39줄, 76~78줄). 마지막 항목은 req.Data 가 방금 Unmarshal 에 성공한 값이라 Marshal 이 실패할 수 없어 도달 불가이므로 제외. ToResponse 실패는 typed nil *VoipbinError 를 직접 넘기면 닿는 방어용 폴백이며 2 stmt 한 케이스로만 확인함.
- 이미 커버됨: v1_webhooks.go 의 len(uriItems) < 3 분기는 기존 InvalidURI 테스트가 커버하므로 대상 아님.
- webhookhandler 나머지 9 stmt(main.go 1, message.go 1, urlvalidator.go 1, webhook.go 6)는 이번 계획에서 제외하고, 목표에 부족할 때에만 추가하는 여유분으로 둠.
- 가정 정정: 남은 미커버가 main 뿐인 것은 아님. listenhandler 의 오류 매핑 7 stmt 와 Marshal 오류 4 stmt 는 도달 불가이고, cmd 안에도 작은 함수가 있음. 확인된 사실: 수기 합계와 기준선 수치, 65% 달성 가능성.

## 2. 대상 (미커버 stmt 와 방법)

| 대상 | stmt | 방법 |
|---|---|---|
| activeflowhandler/get.go | 8 | 캐시 읽기 오류 후 RPC 폴백, 캐시 쓰기 오류가 결과에 영향 없음(NotFound, 삭제됨, 빈 URI, 양성), fallbackTimestamp 우선순위(update > create > now) |
| subscribehandler/flowmanager.go | 11 | 잘못된 JSON, 캐시 쓰기 오류 전파(양성, 음성, 삭제), activeflowEventTimestamp 우선순위(delete > update > create > now) |
| subscribehandler/main.go | 6 (고루틴 로그 1 stmt 를 뺀 확정분 5) | Run 의 QueueCreate, TopicCreateWithKind, QueueBind 오류 반환과 후속 호출 미실행, ConsumeMessage 오류 시에도 Run 은 nil 반환(고루틴 안 오류 로그 1 stmt 는 산정 제외), processEvent 의 flow-manager 세 이벤트 라우팅 |
| webhookhandler/routingkey.go | 8 | createRoutingKeysForChat(chat id 없음, id 대체, 고객/소유자 키, 참가자 RPC 오류, 소유자 중복 제거), publishRoutingKeyedEvent(형식 오류 이벤트 타입은 발행 없음, chat 리소스는 chat 키로 발행) |
| listenhandler/main.go Run | 9 | QueueCreate 오류 반환(ConsumeRPC 호출 없음), 성공 시 mock 안에서 채널로 신호하여 ConsumeRPC 호출과 큐 이름, 소비자 이름 검증, ConsumeRPC 가 오류를 돌려줘도 Run 은 nil 반환. 고루틴 안의 오류 로그 1 stmt 는 mock 반환 뒤 실행이라 실행 시점이 보장되지 않으므로 목표 산정에서 제외 |
| listenhandler/main.go errorResponse | 2 | typed nil *VoipbinError 는 ToResponse 가 실패하여 500 으로 폴백(방어 폴백 확인, 이 한 케이스만) |

예상 증가 약 44 stmt (713/1039), 수기 합계 약 68.6%. 실행 시점이 보장되지 않는 고루틴 로그 2 stmt 를 제외한 확정분은 42 stmt (711/1039, 68.4%). 추가 여유: config, accounthandler, cachehandler 단일 분기(필요 시).

기준선 주의: subscribehandler.Run 의 ConsumeMessage 고루틴 호출 stmt 는 기존 테스트에서 실행 시점이 비결정적이라 측정마다 1 stmt(668~669)가 흔들릴 수 있다. 새 테스트는 이 호출을 채널 신호로 결정적으로 커버한다.

## 3. Verify 항목 (설계 단계에서는 코드 없음)

1. 수기 합계: 필수 기준 65% 이상, 권장 기준(여유) 66% 이상을 목표로 함 (gens/, mock_*.go, DO NOT EDIT 제외, 중복 블록 최대 count).
2. `go test -race -count=1 ./...` 통과, golangci-lint 통과, `scripts/check-test-conventions.sh` 통과(이 스크립트는 merge-base 이후 커밋된 diff 만 검사하므로 커밋 후, origin/main 을 fetch 한 상태에서 실행).
3. 커밋 후 `git diff --stat origin/main...HEAD` 가 bin-webhook-manager 의 _test.go 와 이 문서뿐이고 비테스트 .go 변경이 0 (커밋 전에는 `git status --short` 로 확인).
4. 변이 표본은 5절. 새 테스트가 실제 결함을 잡아야 함.
5. 자기비교 금지: 기대값은 리터럴 또는 독립 계산. 시각은 구체적 `time.Time` 리터럴과 `Equal` 로 비교하고 `gomock.Any()` 로 받지 않음. 소비자 이름 등도 상수가 아니라 리터럴(`"webhook-manager"`)로 검증.
6. 동시성: listenhandler.Run 의 ConsumeRPC 와 subscribehandler.Run 의 ConsumeMessage 는 고루틴이다. 호출 여부는 mock 의 `DoAndReturn` 안에서 채널로 신호하고 테스트는 `select` 와 `time.After` 타임아웃(무한 대기 금지)으로 기다리며 정확히 1회만 기대한다(AnyTimes 금지). 고루틴 안 오류 로그 stmt 의 실행 시점은 보장되지 않으므로 산정에서 제외. `time.Sleep` 에 의존하지 않음. `-race -count=20` 으로 안정성 확인.
7. 합성값만 사용. 네트워크, 외부 비용 호출 없음. UUID 리터럴은 hex 문자만 쓰고(`FromStringOrNil` 이 비 hex 를 조용히 `uuid.Nil` 로 만듦) 필요한 곳에서 `!= uuid.Nil` 을 단언.
8. 전역 상태: prometheus 카운터 값을 단언하지 않고, logrus 전역 `SetOutput`/`AddHook` 을 쓰지 않음.
9. 컨벤션: 새 테스트는 `Test_` 접두와 gomock controller 이름 `mc`, 테이블 기반. 기존 헬퍼(`containsString` 등)와 기존 테스트 이름을 재선언하지 않음. `publishRoutingKeyedEvent` 와 `createRoutingKeysForChat` 는 직접 동기 호출하므로 sleep 이 필요 없음.

## 4. 테스트 계획 (파일과 케이스)

- pkg/activeflowhandler/get_test.go 추가
  - 캐시 읽기 오류: `ActiveflowWebhookGet` 이 `(nil, false, errors.New(...))` 를 돌려주면 `FlowV1ActiveflowGet` 이 정확히 1회 호출되고, `ActiveflowWebhookSet` 이 구체적 URI 와 Tm 으로 정확히 1회, `SetNegative` 는 0회, 반환 Destination 이 리터럴과 같음.
  - 쓰기 오류 표(NotFound 의 transient negative, 삭제됨, 빈 URI, 양성; NotFound 행은 제품 코드가 `time.Now()` 를 넘기므로 `SetNegative` 의 시각 인자를 `DoAndReturn` 으로 받아 before/after 범위로 검증하는 이 행만의 예외이고 나머지 인자는 구체 값): 캐시 쓰기가 오류를 돌려줘도 Get 이 오류 없이 기대 Destination(양성) 또는 nil(그 외)을 돌려줌.
  - fallbackTimestamp: `Get` 경유로 TMUpdate 와 TMCreate 가 모두 있고 서로 다른 값일 때 update 가 쓰인 것을 `ActiveflowWebhookSet` 의 Tm 구체 값으로 검증(`get.go` 130~131 도달). TMCreate 만 있는 경우는 create. 둘 다 없는 경우는 `before := time.Now()` 와 `after` 사이임을 `!tm.Before(before)`, `!tm.After(after)` 로 검증(`DoAndReturn` 으로 받아 비교).
- pkg/subscribehandler/flowmanager_test.go 추가
  - 두 핸들러에 잘못된 JSON: 오류 반환, 캐시 호출 0회.
  - 캐시 쓰기 오류 전파 표(양성 Set, 음성 SetNegative, 삭제 SetNegative): 반환 오류가 mock 이 돌려준 오류.
  - activeflowEventTimestamp: 함수를 직접 호출해 tm_delete, tm_update, tm_create 가 모두 서로 다른 값일 때 delete, update 만, create 만 순으로 우선함을 리터럴 `time.Time` 과 `Equal` 로 검증. 셋 다 없으면 호출 전후 시각 사이. 이벤트 JSON 시각은 `Z` 표기.
- pkg/subscribehandler/main_test.go 추가
  - Run 오류 3종 표: QueueCreate 오류면 TopicCreateWithKind 이하 호출 없음, TopicCreateWithKind 오류면 QueueBind 호출 없음, QueueBind 오류면 ConsumeMessage 호출 없음. 반환 오류에 해당 단계 문구 포함. (변이 시 미기대 호출이 고루틴에서 일어나 패닉으로 보일 수 있으나 반환값 단언이 먼저 실패함.)
  - ConsumeMessage 가 오류를 돌려줘도 Run 은 nil 반환(반환값 확인용이며 변이 검출력은 없음), 호출은 채널 신호로 정확히 1회 확인(큐 이름과 소비자 이름 리터럴 검증).
  - processEvent 라우팅 표: flow-manager 의 created 와 updated 는 webhook_uri 유무로 `ActiveflowWebhookSet` 또는 `SetNegative(..., nil, ...)`; deleted 는 webhook_uri 가 비어 있고 tm_delete 가 있는 payload 로 `SetNegative` 의 4번째 인자가 `&tmDelete` 와 일치(created/updated 핸들러로 잘못 가면 nil 이라 실패). 발행자와 타입이 어긋난 이벤트(customer-manager 발행자에 activeflow 타입)는 캐시 호출 0회.
- pkg/webhookhandler/routingkey_test.go 추가 (직접 동기 호출)
  - createRoutingKeysForChat: chat id 와 d.ID 가 모두 nil 이면 비 nil 빈 슬라이스(`[]string{}`, nil 슬라이스와 DeepEqual 이 다름), ChatID 없으면 d.ID 로 대체되어 키에 d.ID 가 쓰임, 고객 키와 소유자 키 포함 순서 검증(참가자 RPC 오류 시에도 이 키들이 유지되고 슬라이스를 반환함을 확인), 참가자 중 d.OwnerID 와 같은 사람은 중복 키가 생기지 않도록 `reflect.DeepEqual` 로 슬라이스 전체(순서: 고객, 소유자, 참가자)를 리터럴과 비교.
  - publishRoutingKeyedEvent: 비어 있지 않지만 `_` 가 없는 type 이면 `PublishEventWithRoutingKey` 0회(mock 미기대); 이 케이스의 Data 는 유효한 JSON 객체(예: `{"type":"call","data":{}}`)여야 하며 비어 있으면 앞단에서 먼저 반환해 167줄 블록에 닿지 않으므로 coverprofile 로 해당 블록 커버를 확인. chat 케이스의 Data 에는 customer_id, owner_id, id, chat_id 를 명시, chat 리소스(`chatmessage_created`)면 `reqHandler`(`TalkV1ParticipantList`)와 `topicNotifyHandler` 를 모두 주입하고 `gomock.InOrder` 로 키 3개(고객, 소유자, 참가자)의 발행 순서까지 고정하여 정확히 발행됨(`PublishEventWithRoutingKey` 의 데이터 인자는 gomock.Any 가 아니라 `json.RawMessage` 리터럴로 검증).
- pkg/listenhandler/main_test.go (새 파일): Run 3케이스(QueueCreate 오류는 ConsumeRPC 호출 없이 오류 반환, 성공은 ConsumeRPC 호출을 채널로 확인하고 큐 이름과 소비자 이름 리터럴 `"webhook-manager"` 검증, ConsumeRPC 오류 케이스는 반환값 nil 확인용이며, mock 은 신호 후 오류를 돌려주고 오류 로그는 그 뒤 테스트 종료 후에 실행될 수 있으므로 오류 로그 stmt 커버와 변이 검출력은 없음). errorResponse typed nil 폴백 1케이스는 error_response_test.go 끝에 추가(응답이 nil 이 아니고, StatusCode 가 500 이며, DataType 이 cerrors.DataTypeVoipbinError 가 아님을 함께 단언).
- 새 테스트 함수 이름(기존 `Test_Run_TopicMigrationSequencing`, `TestCreateRoutingKeysForChat`, `Test_Get_*`, `TestProcessEvent*` 등과 겹치지 않음): `Test_Get_cacheReadError`, `Test_Get_cacheWriteErrorsIgnored`, `Test_Get_fallbackTimestamp`, `Test_processEventFMActiveflowCreatedUpdated_invalidJSON`, `Test_processEventFMActiveflow_cacheErrors`, `Test_activeflowEventTimestamp`, `Test_Run_errors`, `Test_Run_consumeError`, `Test_processEvent_flowManagerRouting`, `Test_createRoutingKeysForChat_cases`, `Test_publishRoutingKeyedEvent_cases`, `Test_Run`(listenhandler), `Test_errorResponse_typedNilFallback`. 오류 비교는 errors.Is 를 쓰고 golangci-lint 를 통과(standard 린터에 더해 errorlint, nilerr, wastedassign, predeclared, nolintlint 가 테스트에도 적용되고 그 밖의 추가 린터는 _test.go 에서 제외됨).
- 제품 코드 변경 없음.

## 5. 변이 표본 계획과 합격 기준

- 변이는 worktree 가 아니라 scratch 사본(저장소 밖 임시 디렉터리, 새 테스트 파일을 복사해 적용)에서 제품 파일을 바꿔 실행하여 해당 패키지 테스트가 실패하는 것을 확인한다. scratch 사본은 go.mod 의 `replace monorepo/bin-* => ../bin-*` 때문에 같은 상대 배치가 필요하다: scratch 디렉터리 안에 bin-webhook-manager 만 복사하고 나머지 `bin-*` 형제 모듈은 worktree 의 해당 디렉터리로 가는 심볼릭 링크로 둔다(vendor 는 쓰지 않고 `GOFLAGS=-mod=mod`). worktree 의 제품 파일은 건드리지 않으므로 실수로 커밋될 위험이 없고, 최종 diff 에 제품 파일이 없음을 확인한다. 변이 적용 시 미기대 mock 호출이 고루틴에서 일어나 깔끔한 FAIL 대신 패닉으로 보일 수 있으며 이 경우도 '실패'로 판정하되 귀속을 명확히 하기 위해 `-run` 으로 대상 테스트 하나만 단독 실행해 실패를 확인한다.
- 변이 목록과 잡아야 하는 새 테스트: (1) get.go fallbackTimestamp 에서 TMUpdate 와 TMCreate 의 우선순위 교환, (2) flowmanager.go activeflowEventTimestamp 에서 TMDelete 와 TMUpdate 우선순위 교환(기존 delete 테스트는 update 가 없어 통과하므로 새 테스트가 잡아야 함), (3) subscribehandler Run 에서 QueueBind 오류를 무시, (4) routingkey.go 참가자 소유자 중복 제거 삭제, (5) get.go 에서 캐시 읽기 오류 시 `return nil, nil` 로 폴백 생략, (6) listenhandler Run 에서 QueueCreate 오류를 무시, (7) subscribehandler processEvent 에서 deleted 이벤트를 created/updated 핸들러로 라우팅.
- 변이 표본에서 의도적으로 뺀 항목: routingkey.go chat 분기, ChatID 를 d.ID 로 대체, flowmanager.go 의 return err 는 해당 테스트가 직접 단언하며 표본 수를 제한하기 위해 제외. 참가자 RPC 오류 분기는 participants 가 nil 이라 return 제거와 동등 변이이므로 이 케이스는 변이 검출력이 없고 키 유지 확인용으로만 둔다.
- 합격: 수기 합계 필수 65% 이상(권장 66% 이상, 예상 68.4%), 변이 7개 모두 새 테스트가 실패, `-race -count=20` 안정.

## 6. 롤백 계획

테스트와 문서 추가뿐이며 제품 코드와 운영 환경에 영향이 없다. 문제 시 PR 을 닫거나 되돌림 커밋으로 충분.
