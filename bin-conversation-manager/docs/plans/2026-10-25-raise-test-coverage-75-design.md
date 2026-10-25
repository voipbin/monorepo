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
| 그 외 | 90% 안팎 | |

cmd 의 두 main 은 외부 연결 의존이라 신규 테스트 대상에서 제외합니다(분모에는 포함). linehandler 의 Send(텍스트 있음, `send.go:45` 의 `Do()` 는 ctx 를 받지 않음)와 GetPeer(`hook.go` 의 `profile.Do()`)는 실제 LINE API 호출이므로 제외합니다. Setup, Teardown 은 `WithContext(ctx)` 를 쓰므로 이미 취소된 ctx 로 호출하면 연결 시도 없이 `context canceled` 로 끝남을 scratch 에서 확인했습니다(네트워크 호출 없음). 이 오류 경로와 텍스트 없는 Send 는 대상으로 삼습니다.

필요량: 76% 는 2753(+28), 77% 목표는 2789(+64) 입니다.

## 3. 변경 계획 (테스트 파일만)

1. pkg/listenhandler: 입력 검증과 오류 경로. 잘못된 JSON 본문은 400, URI 항목 부족은 400, 하위 handler 오류는 errorResponse 매핑(타입 오류 상태, ErrNotFound 404, 그 외 500), 허용 필드가 없는 PUT 은 200, 잘못된 필터는 400. 대상: v1_conversations, v1_accounts, v1_messages, v1_hooks.
2. pkg/accounthandler: Update, Delete, Create 의 DB 오류와 setup 오류 전파, Delete 시 하위 호출 순서.
3. pkg/messagehandler: sendWhatsApp, sendWebchat, SMS 전송의 오류 경로(계정 조회 실패, 생성 실패, 전송 실패 시 failed 로 상태 갱신, webchat 세션 ID 파싱 실패, 중복 키 경합 시 재조회).
4. pkg/conversationhandler: Event 의 분기(알 수 없는 type 오류, 잘못된 JSON, direction 불명).
5. pkg/linehandler: 텍스트와 media 가 없는 Send 가 클라이언트 생성 후 네트워크 호출 없이 성공함.

제품 코드, 시그니처, go.mod 는 바꾸지 않습니다. 테스트 규약: 함수명 Test_, gomock 컨트롤러 이름 mc, 합성 값만 사용.

## 4. 검증 항목 (verify)

- 수기 합계 76% 이상 (목표 77%), 패키지별 전후 표를 PR 본문에 첨부.
- `go test ./...`, `golangci-lint run`, `scripts/check-test-conventions.sh` 통과, `git diff --stat origin/main...HEAD` 가 bin-conversation-manager/ 안에만 있음.
- 변이 검사: 표본 3곳 이상(예: 400 반환을 500 으로, 실패 상태 갱신 호출 제거, 중복 키 재조회 제거)에서 새 테스트가 실패함을 scratch 복사본에서 확인.
- 결함을 발견하면 고치지 않고 카드 코멘트에만 적음.

## 5. VOIP-1416 중복

pkg/linehandler, pkg/accounthandler, pkg/messagehandler, pkg/dbhandler 등 80% 미만 패키지중 VOIP-1416(80% 미만 패키지 상향, Backlog)이 적은 패키지와 겹치는 것이 있습니다(카드 본문 기준, 해당 티켓 내용은 열람 불가). 흡수하지 않고 명시만 합니다.
