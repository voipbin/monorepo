# bin-conversation-manager 단위 테스트 커버리지 상향 (VOIP-1647)

## 목표와 측정
- 수기 코드(gens/, mock_*.go, `Code generated ... DO NOT EDIT` 파일 제외, 동일 블록은 최대 count 로 합산) statement 커버리지를 56.47%(2035/3604)에서 61% 이상으로 올린다.
- 측정: 패키지별 프로파일(`go test -count=1 -coverprofile=cp.out ./...`)에서 위 제외를 적용해 블록 단위로 집계한다. `-coverpkg` 는 다른 패키지 테스트가 models 를 간접 실행해 값이 부풀려지므로 쓰지 않는다. `go tool cover -func` 의 total 은 mock 과 main 을 포함하므로 약 56% 로 수기 기준과 다르다.
- 범위: 테스트 파일 추가만. 제품 코드(비테스트 .go)는 바꾸지 않고, 네트워크와 외부 비용 호출 테스트를 만들지 않으며, 값은 합성값만 쓴다.

## 제외
- cmd/ 의 두 main, pkg/cachehandler(Redis 클라이언트를 직접 생성하고 주입점이 없음).
- linehandler 의 Send, Setup, Teardown, GetPeer 와 follow 후반부(LINE 클라이언트의 외부 호출).
- 도달 불가 코드: models 의 `json.Marshal` 오류 반환 한 줄씩, getExecuteMode 가 agent 와 flow 만 반환하고 runExecuteModeAgent 가 항상 nil 을 반환하므로 agent 오류 반환과 unknown execute mode 분기, listenhandler 의 json.Marshal 실패 응답, AccountUpdate 의 ToSql 오류.

## 추가한 테스트와 verify 기준
1. models: ConvertStringMapToFieldMap 3개. convertValue 는 uuid, bool, string-kind, int, float64 만 변환하고 같은 타입 값은 그대로 통과시키며 모르는 키는 원값으로 통과시킨다. 정상 경로는 UUID 문자열과 `*time.Time` 값 통과, 오류 경로는 잘못된 UUID 문자열과 time 필드에 문자열. conversation 의 ConvertWebhookMessage 와 CreateWebhookEvent, message 의 CreateWebhookEvent 는 모든 필드가 비영값인 픽스처와 리터럴 기대 JSON 으로 비교한다(내부 필드 case_id, metadata 가 노출되지 않음 포함).
2. linehandler Hook 계열: 잘못된 JSON, 빈 events, 미지원 이벤트, 텍스트와 비텍스트(image) 메시지 성공, dialog id 가 없는 follow 이벤트, hookEventTypeMessage 의 목록 오류와 0건과 생성 오류, getDialogID 의 user, group, room, 그 외. 성공 경로는 source 가 cv.Peer, destination 이 cv.Self 이고(두 값이 서로 다른 픽스처) medias 가 빈 슬라이스임을 인자 매칭으로 검증한다. source 가 없는 이벤트와 message 가 없는 message 이벤트는 제품 코드가 nil 역참조를 하므로 쓰지 않는다.
3. conversationhandler: MessageEventSent(신규 생성과 case 힌트 전달, 기존 메시지 상태 갱신, 타깃 0개와 2개, 대화 생성 실패, 메시지 생성 실패, 상태 갱신 실패), MessageSend 의 조회 실패와 전송 실패. 타깃 2개에서는 타깃별로 다른 대화를 반환해 생성 인자의 destination 이 타깃별로 달라짐을 검증한다.
4. listenhandler: create_and_execute_flow POST 의 성공, JSON 오류 400, 일반 오류 500, typed 오류의 상태 코드와 envelope.
5. subscribehandler: email updated 의 정상, JSON 오류, 핸들러 오류, email created 와 message created 의 오류 경로, webchat 이벤트, processEvent 의 분기별 핸들러 호출 여부(processEvent 는 반환값이 없고 오류를 로그로만 남기므로 호출 여부로 검증한다). processEventRun 과 Run 은 고루틴을 쓰므로 대상에서 제외한다.
6. messagehandler: sendLine 의 계정 조회 실패, 메시지 생성 실패, 전송 실패 시 failed 상태 갱신, 전송 후 상태 갱신 실패, 미지원 대화 타입.
7. dbhandler: AccountUpdate(sqlite 인메모리 DB, 다른 테스트와 겹치지 않는 ID 와 customer id, 케이스마다 새 fields map, tm_update 와 캐시 갱신 인자 검증, 빈 fields 와 존재하지 않는 컬럼).

## 검증
- `go mod tidy`, `go mod vendor`, `go generate ./...`, `go test ./...`, `golangci-lint run`, `scripts/check-test-conventions.sh` 를 모두 실행하고 go.mod 와 go.sum 이 바뀌지 않음을 확인한다.
- 변이 표본(제품 코드를 임시로 바꿔 새 테스트가 실패하는지 확인): MessageEventSent 의 상태 갱신 인자, hook 의 source 와 destination 교환, getDialogID 의 room 분기, AccountUpdate 의 tm_update 설정, create_and_execute_flow 의 400 응답, email created 의 오류 반환, sendLine 의 실패 상태 갱신, conversation 웹훅의 필드 누락, errorResponse 의 typed 오류 분기.
- 롤백: 테스트 파일 추가뿐이므로 PR 되돌림으로 충분하다.
