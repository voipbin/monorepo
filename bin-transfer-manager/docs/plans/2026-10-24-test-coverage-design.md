# bin-transfer-manager 단위 테스트 커버리지 상향 설계 (VOIP-1647)

## 1. 티켓 검토
유효. origin/main d68e695b5 에서 수기 코드 516/949 = 54.37%, 원본 516/1076 = 47.96% 를 재측정해 카드 수치와 일치합니다.
수기 합계는 gens/, mock_*.go, `Code generated ... DO NOT EDIT` 헤더 파일을 제외하고 같은 블록 중복은 최대 count 로 합칩니다.

## 2. 기준선 (수기, covered/stmts)
- cmd/transfer-control 0/145, cmd/transfer-manager 1/122 (main 과 외부 연결, 이번 범위 제외)
- internal/config 58/61, models/transfer 8/9, pkg/dbhandler 42/42
- pkg/cachehandler 0/59, pkg/listenhandler 86/103, pkg/subscribehandler 71/88, pkg/transferhandler 250/320

## 3. 변경 계획 (테스트 파일만, 제품 코드 변경 없음)
목표 수기 합계 62% 이상 (약 590/949 이상).
1. cachehandler/handler_test.go: miniredis 로 TransferSet 이 3개 키(id, transferer_call_id, groupcall_id)를 모두 쓰는지, 각 Get 이 값을 복원하는지, 없는 키는 redis.Nil 오류, 손상된 JSON 은 unmarshal 오류, 서버 종료 후 Set/Get/Connect 오류를 확인합니다. go.mod 에 miniredis 테스트 의존성 추가(타 서비스 사용 버전과 동일).
2. subscribehandler: 세 callmanager 핸들러의 잘못된 JSON 은 오류 반환, 조회 실패는 nil 반환이며 transferHandler 의 Answer/Hangup 이 호출되지 않음, 하위 핸들러 오류도 nil 반환. Run 의 QueueCreate, TopicCreate, QueueBind 실패 전파와 processEvent 의 알 수 없는 이벤트 무시.
3. listenhandler: V1TransfersPost 의 잘못된 JSON 은 400, ServiceStart 실패는 500, errorResponse 의 nil, VoipbinError, ErrNotFound, 기타 분기 상태코드, Run 의 QueueCreate 실패.
4. transferhandler: attended/blind 의 confbridge 조회 실패, groupcall 생성 실패, Create 실패, rollback(unblock) 호출 여부, moh/mute 실패는 계속 진행, transferee 의 지원하지 않는 타입과 각 오류 분기, transferer, transfer, db 의 오류 분기.

## 4. 금지와 품질 기준
빈 호출과 자기비교 테스트 금지. 각 테스트는 mock 호출 기대(gomock 엄격 모드)와 반환 오류 또는 상태코드를 단언합니다. 합성 값만 사용하고 네트워크와 외부 비용 호출은 없습니다.

## 5. verify 항목
- `go test -count=1 -race ./...` 통과, 수기 합계 61% 이상 (agg 스크립트로 전후 표 작성)
- 변이 표본: transferHandler 의 rollback 호출 제거, cachehandler 의 키 접두사 변경, listenhandler 상태코드 변경 시 새 테스트가 실패해야 합니다.
- golangci-lint, scripts/check-test-conventions.sh 통과. PR 파일은 bin-transfer-manager/ 안에만 있어야 합니다.

## 6. 롤백
테스트 전용 변경이므로 PR revert 로 되돌립니다. 운영 영향 없음.
