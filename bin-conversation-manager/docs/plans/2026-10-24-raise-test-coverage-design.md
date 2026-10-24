# bin-conversation-manager 단위 테스트 커버리지 상향 (VOIP-1647)

## 목표와 측정
- 수기 코드(gens/, mock_*.go, `Code generated ... DO NOT EDIT` 파일 제외, 동일 블록은 최대 count) 커버리지를 56.47%(2035/3604)에서 61% 이상(2199 이상, 목표 2230 안팎)으로 올린다.
- 측정은 패키지별 프로파일(`GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...`) 후 블록 단위 집계로 한다. `-coverpkg` 는 쓰지 않는다. CI 의 go tool cover total 은 mock 과 main 을 포함해 약 54% 로 다르다.
- 범위: 테스트 파일 추가만. 제품 코드(비테스트 .go)는 바꾸지 않는다. 네트워크, 외부 비용 호출 없음. 값은 합성값만 쓴다.
- 제외: cmd/ 두 개의 main, pkg/cachehandler(Redis 직접 생성), linehandler 의 Send, Setup, Teardown, GetPeer 와 follow 후반부(linebot Do 호출).

## 테스트 추가 계획 (파일 단위, 기존 파일에 추가하거나 새 `_test.go`)
1. models/account, conversation, message: ConvertStringMapToFieldMap(UUID 문자열, time 문자열, 알 수 없는 키 오류 입력 확인), conversation ConvertWebhookMessage/CreateWebhookEvent, message/account CreateWebhookEvent. verify: 변환된 맵의 값 타입이 uuid.UUID 와 *time.Time 이고 웹훅 JSON 에 내부 필드(Owner 등)의 노출 범위가 구조체 정의와 일치.
2. pkg/linehandler/hook_test.go: Hook(잘못된 JSON, 빈 events, 미지원 이벤트 타입이 결과에서 제외, 텍스트 메시지 성공, image JSON 비텍스트 성공), hookEventTypeMessage(dialogID 빈 값, List 오류, List 0건, MessageCreate 오류, 성공 시 source=peer destination=self 인자 확인), getDialogID(user, group, room, 기타), follow 의 dialogID 빈 값 조기 반환. verify: gomock 인자 매칭이 구체 값이며 mock 이 호출되지 않아야 하는 경로는 mock 기대를 두지 않음. e.Message nil 이벤트와 follow 정상 경로는 쓰지 않는다.
3. pkg/conversationhandler: MessageEventSent(신규 생성, 기존 메시지 UpdateStatus, GetOrCreate 오류, Create 오류, UpdateStatus 오류, 타깃 2개), MessageSend 오류 경로, hook/event/event_webchat 의 미커버 오류 분기. verify: 타깃 2개에서 각 Create 인자의 destination 이 타깃별로 달라야 함.
4. pkg/listenhandler: processV1ConversationsCreateAndExecuteFlowPost(성공, 400, 핸들러 오류), v1_conversations/v1_accounts/v1_messages 의 400, 핸들러 오류, 성공 분기 중 미커버 부분. verify: 상태 코드와 응답 본문 비교.
5. pkg/subscribehandler: webchat 이벤트, email created/updated(JSON 오류, 핸들러 오류, 성공), message created 오류 경로, processEvent 분기. verify: 핸들러 오류가 그대로 반환.
6. pkg/messagehandler/send.go: sendLine, sendWhatsApp, sendWebchat, sendSMS 의 오류 분기와 미지원 분기.
7. pkg/dbhandler: AccountUpdate(sqlite shared DB, 새 UUID, 새 customer_id, 케이스마다 새 fields map). 캐시와 TimeNow mock 기대 정확히.
8. 우선순위는 위험이 낮은 순(models, listenhandler, linehandler, subscribehandler, conversationhandler, messagehandler, dbhandler). 61% 도달 시 중단하고 불필요한 추가는 하지 않는다.

## 검증과 변이 확인
- 각 단계 후 `agg.py` 실측. 최종: `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run`, `scripts/check-test-conventions.sh`(Test_<Method>, testify 금지, gomock controller 이름 mc), `git diff --stat origin/main...HEAD` 가 bin-conversation-manager/ 안만 포함.
- 변이 표본(scratch 복사본): MessageEventSent 의 UpdateStatus 분기 제거, hookEventTypeMessage 의 source/destination 교환, getDialogID 의 case 제거, AccountUpdate 의 TMUpdate 설정 제거에서 새 테스트가 실패해야 한다.
- 자기비교식과 빈 호출 테스트 금지. 제품 결함 발견 시 고치지 않고 완료 요약에 적는다.
- 롤백: 테스트 파일 추가뿐이므로 PR revert.
