# bin-conversation-manager 단위 테스트 커버리지 상향 (VOIP-1652)

## 1. 티켓 검토

- 판정: 유효. 기준선 origin/main accc03e 에서 수기 코드 커버리지는 2725/3622 = 75.23% 로 실측되어 카드 수치와 일치합니다. 원본(생성 코드와 mock 포함) 66.8% 도 카드와 같은 방법입니다.
- 제외 방법: gens/, 파일명 mock_*, 파일 머리(package 이전)에 `Code generated` 와 `DO NOT EDIT` 이 있는 파일을 뺍니다. 같은 블록의 중복은 최대 count 로 합칩니다.
- 75.0% 는 이미 넘었으므로 인수 조건의 실질은 76% 이상 확보(CI 와의 소수점 차이 대비)입니다. 최소 2753/3622, 즉 28개 이상의 statement 를 더 커버해야 합니다.
- 선행 PR: 같은 서비스에서 VOIP-1647(#1524, 56.47%에서 61% 이상)과 VOIP-1649(#1549, 65% 설계 `2026-10-24-test-coverage-65-design.md`)가 머지되었고 이번 카드가 그 결과를 기준선으로 이어받습니다. 두 PR 모두 linehandler 의 Send, Setup, Teardown, GetPeer 를 외부 호출로 보아 제외했습니다.
- 분모 차이(3604, 3622) 설명: 1647 은 3604 기준이었고, 그 뒤 VOIP-1648(cb3c753ca)이 비테스트 코드를 바꾸었습니다(`git diff --stat 5a3496e0c accc03e -- bin-conversation-manager ':!*_test.go'`: pkg/linehandler/hook.go +31 줄, pkg/conversationhandler/message.go +12 줄. 그 밖에 go.mod, go.sum 과 문서 변경은 분모와 무관). 이번 기준선 3622 는 accc03e 의 실측이며 이후 PR 본문의 '전' 열도 이 값을 씁니다.
- cmd/ 두 패키지(1649 에서 이미 일부 테스트됨)는 분모에서 빼지 않습니다. 합계는 cmd 를 포함한 3622 기준이고, 신규 테스트 대상에서만 제외합니다(cmd main 은 외부 연결 의존).
- 공개 저장소에 등록된 다른 열린 PR 중 이 서비스와 겹치는 것은 없습니다(`gh pr list --state all --search 'conversation-manager coverage'` 결과는 위 두 PR 과 오래된 무관 PR 뿐).

## 2. 기준선 (수기 코드, statement)

| 패키지 | 커버/전체 | 비율 |
|---|---|---|
| cmd/conversation-control | 311/528 | 58.9% |
| cmd/conversation-manager | 25/203 | 12.3% |
| pkg/listenhandler | 460/586 | 78.5% |
| pkg/linehandler | 112/202 | 55.4% |
| pkg/dbhandler | 483/585 | 82.6% |
| pkg/conversationhandler | 598/688 | 86.9% |
| pkg/accounthandler | 108/134 | 80.6% |
| pkg/messagehandler | 155/183 | 84.7% |
| 그 외 | 85~99% | |

cmd 의 두 main 은 외부 연결 의존이라 신규 테스트 대상에서 제외합니다(분모에는 포함). linehandler 의 Send(텍스트 있음, `send.go:45` 의 `Do()` 는 ctx 를 받지 않음)와 GetPeer(`hook.go` 의 `profile.Do()`)는 실제 LINE API 호출이므로 제외합니다. Setup, Teardown 은 `WithContext(ctx)` 를 쓰므로 이미 취소된 ctx 로 호출하면 연결 시도 없이 `context canceled` 로 끝남을 scratch 에서 확인했습니다(네트워크 호출 없음). 이 오류 경로와 텍스트 없는 Send 는 대상으로 삼습니다.

필요량: 76% 는 2753(+28), 77% 목표는 2789(+64) 입니다.

## 3. 변경 계획 (테스트 파일만)

1. pkg/listenhandler (v1_conversations, v1_accounts, v1_messages, v1_hooks). 핸들러마다 반환 형태가 다르므로 코드에 맞춰 각각 단언합니다.
   - 잘못된 JSON 본문에서 400 응답을 반환: ConversationsPost, GetOrCreateBySelfAndPeerPost, CreateAndExecuteFlowPost, IDMetadataPut, AccountsPost, MessagesPost, MessagesCreatePost, HooksGet, HooksPost. 이때 하위 handler 는 호출되지 않아야 합니다(mock 에 기대를 걸지 않음).
   - 잘못된 JSON 본문 또는 URI 파싱 오류에서 `(nil, err)` 를 반환: ConversationsGet, AccountsGet, MessagesGet(JSON), AccountsIDPut 와 ConversationsIDPut 의 필터 분기.
   - 필터 변환 실패에서 400 응답: ConversationsGet, AccountsGet, MessagesGet.
   - URI 항목 부족에서 400 응답: ConversationsIDGet, ConversationsIDPut, ConversationsIDMetadataPut, AccountsIDGet, AccountsIDPut, AccountsIDDelete.
   - 하위 handler 오류에서 errorResponse 매핑(타입 오류 상태, ErrNotFound 404, 그 외 500): ConversationsGet, Post, IDGet, GetOrCreate, CreateAndExecuteFlow, IDMetadataPut, AccountsGet, IDGet, IDPut, IDDelete, MessagesPost, MessagesCreatePost. 반면 AccountsPost 와 MessagesGet 은 하위 오류를 `(nil, err)` 로 전달하므로 그렇게 단언합니다.
   - 허용 필드가 없는 PUT 은 200 이고 Update 가 호출되지 않음: ConversationsIDPut, AccountsIDPut.
   - HooksGet 의 잘못된 ReceivedURI 는 400.
   - 반환값 직렬화 실패(500) 분기는 정상 타입에서 만들 수 없으므로 대상에서 뺍니다.
2. pkg/accounthandler: Update, Delete, Create 의 DB 오류와 setup 오류 전파. Delete 는 조회 실패 시 teardown 과 삭제를 호출하지 않음, 삭제 실패 시 알림을 발행하지 않음을 단언합니다.
3. pkg/messagehandler (send.go): 계정 조회 실패, 생성 실패, 전송 실패 시 상태를 failed 로 갱신(sendWhatsApp, SMS), 상태 갱신 실패 전파. sendWebchat 은 세션 ID 파싱 실패가 이미 커버되어 제외하고, `Create` 실패 후 `Duplicate entry` 와 `UNIQUE constraint failed` 각각의 재조회 성공, 재조회도 실패하면 원래 오류 반환을 다룹니다.
4. pkg/conversationhandler (event.go): 알 수 없는 type 오류, 잘못된 JSON, direction 불명, MessageEventReceived 와 MessageEventSent 의 오류 래핑 분기(하위 mock 설정 필요). 예비 후보로 event_webchat.go 의 미커버 분기.
5. pkg/linehandler: 합성 secret 과 token 을 가진 계정으로 실제 `NewLineHandler` 핸들러의 Send 를 호출하되 텍스트를 비우고 media 를 한 개 넣어 media 루프를 지나게 하며, 외부 호출(PushMessage)이 일어나지 않고 nil 을 반환함을 단언합니다. 이미 취소된 ctx 로 Setup, Teardown 을 호출하면 `errors.Is(err, context.Canceled)` 임을 단언합니다(요청이 전송되지 않음).

예상 증가량(statement, 대상 파일의 미커버 합에서 제외 분기를 뺀 어림): listenhandler 약 45(대상 약 75), accounthandler 약 12(대상 19), messagehandler 약 15(대상 21), conversationhandler 약 8(대상 13), linehandler 약 20(대상 53 중 Send 와 Setup/Teardown 의 가능분). 합계 약 100 이면 77% 이상(필요 +64). 미달 시 dbhandler 오류 경로를 추가합니다.

제품 코드, 시그니처, go.mod 는 바꾸지 않습니다. 테스트 규약: 함수명 Test_, gomock 컨트롤러 이름 mc, 합성 값만 사용.

## 4. 검증 항목 (verify)

- 수기 합계 76% 이상 (목표 77%), 패키지별 전후 표를 PR 본문에 첨부.
- `go test ./...`, `golangci-lint run`, `scripts/check-test-conventions.sh` 통과, `git diff --stat origin/main...HEAD` 가 bin-conversation-manager/ 안에만 있음.
- 변이 검사: 패키지별 최소 1곳, 최소 5곳을 scratch 복사본에서 확인하며 새 테스트가 실패해야 합니다. 예: listenhandler 의 400 을 500 으로, messagehandler 의 failed 상태 갱신 제거, 같은 곳의 재조회 제거, accounthandler 의 teardown 호출 제거, linehandler 의 Canceled 오류 삼킴.
- 결함을 발견하면 고치지 않고 카드 코멘트에만 적음.

## 5. VOIP-1416

카드 본문은 VOIP-1416(80% 미만 패키지 일반 상향)과 패키지가 겹치면 PR 본문에 명시만 하라고 합니다. 이 서비스에서 80% 미만인 패키지는 pkg/linehandler(55.4%), pkg/listenhandler(78.5%) 와 cmd 두 개이며, 티켓 내용은 열람할 수 없어 겹침 여부는 확인 불가로만 적고 흡수하지 않습니다.
