# bin-transfer-manager 단위 테스트 커버리지 상향 설계 (VOIP-1647)

## 1. 티켓 검토
유효. origin/main d68e695b5 에서 수기 코드 516/949 = 54.37%, 원본 516/1076 = 47.96% 를 재측정해 카드 수치와 일치합니다.
수기 합계는 gens/, mock_*.go, `Code generated ... DO NOT EDIT` 헤더 파일을 제외하고 같은 블록 중복은 최대 count 로 합칩니다.

## 2. 기준선 (수기, covered/stmts)
- cmd/transfer-control 0/145, cmd/transfer-manager 1/122 (main 과 외부 연결, 이번 범위 제외)
- internal/config 58/61, models/transfer 8/9, pkg/dbhandler 42/42
- pkg/cachehandler 0/59, pkg/listenhandler 86/103, pkg/subscribehandler 71/88, pkg/transferhandler 250/320

## 3. 변경 계획 (테스트 파일과 go.mod/go.sum 만, 제품 코드 변경 없음)
목표: 수기 합계 61% 이상(579/949 이상, 인수 조건). 기준선은 base d68e695b5 에서 3회 연속 516/949 로 측정했으며 비동기 테스트 때문에 환경에 따라 ±2 편차가 있을 수 있습니다. 미커버 433 중 cmd 를 뺀 미커버는 167(cachehandler 59, listenhandler 17, subscribehandler 17, transferhandler 70, 기타 4)이고 필요 증가분은 +63 입니다. 아래 표의 기대치 합은 약 100(약 616, 65%)이며 61% 에 여유 약 37 을 둡니다.

| 대상 | 기대 증가 stmt | 내용 |
|---|---|---|
| cachehandler/handler_test.go (신규) | 약 55 | miniredis v2.36.1(타 서비스 다수 버전). 리터럴 키 `mr.Get("transfer:transfer:<id>")`, `transfer:transferer_call_id:<id>`, `transfer:groupcall_id:<id>` 로 직접 조회해 세 키의 본문이 같은 JSON 이고 `mr.TTL` 이 24h 임을 단언, 각 Get 의 필드 단위 복원 단언(합성 입력과 다른 리터럴 JSON 을 심어 복원), 없는 키는 redis.Nil, 손상 JSON 은 unmarshal 오류, NewHandler(mr.Addr(), "", 0) 정상 경로와 Connect 성공, 서버 종료 뒤 Connect/Get/Set 오류(컨텍스트 타임아웃으로 시간 제한, 같은 패키지이므로 MaxRetries 0 의 redis.Client 를 직접 구성). 두 번째와 세 번째 Set 만 실패하는 분기는 miniredis 의 명령 훅으로 N번째 SET 만 실패시켜 침. 불가하면 도달 불가로 명시 |
| listenhandler | 약 10 | 현재 유일한 경로 POST /v1/transfers 에서 processRequest 로: 잘못된 JSON 은 400(이미 커버된 케이스는 반복하지 않음), ServiceStart 실패는 processV1TransfersPost 가 `(simpleResponse(500), nil)` 로 오류를 삼키므로 500 과 transferHandler 호출 인자 단언(v1_transfers.go:36-38). Run: QueueCreate 실패 오류 반환, 성공 시 ConsumeRPC 고루틴을 채널로 동기화해 종료 전 호출 완료 보장(125-140). 도달 불가로 제외(제품 코드 변경 없이는 입력 불가): processRequest 의 VoipbinError 와 ErrNotFound 분기(main.go:191, 193), errorResponse 의 ToResponse 실패(95-96), Marshal 실패(v1_transfers.go:30-32 아님, 36 이후 Marshal 분기). errorResponse 자체는 기존 error_response_test.go 가 커버하므로 중복하지 않음 |
| subscribehandler | 약 14 | 세 callmanager 핸들러: 잘못된 JSON 은 오류 반환(23-25, 51-53, 79-81), GetBy 조회 실패는 nil 반환이며 Answer/Hangup 미호출(gomock 컨트롤러에 t 전달, 기대 미설정), 하위 핸들러 오류도 nil(32-37 등). processEvent 의 오류 로그 분기(173-174)는 오류를 내는 핸들러 이벤트로 침. Run 의 QueueCreate, TopicCreateWithKind 실패와 QueueBind 를 두 번째 패턴에서 실패시켜 이후 호출이 없음을 확인, 성공 시 ConsumeMessage 고루틴 동기화 |
| transferhandler | 약 25 이상 | attended/blind: confbridge 조회 실패, groupcall 생성 실패, Create 실패 시 rollback 이 올바른 confbridge ID 인자로 호출되고 rollback 자체가 실패해도 원 오류가 래핑되어 반환됨, moh/mute 실패는 이후 mock 호출이 진행되고 오류 없음. transferee/transferer: 지원하지 않는 타입, CallGet, ConfbridgeTerminate, updateTransfereeCallID, attendedUnblock 실패. transfer.go 와 db.go: GetByGroupcallID, GetByTransfererCallID, Get 각각 ErrNotFound 가 cerrors NotFound 타입이며 errors.Is(dbhandler.ErrNotFound) 가 보존됨, 그 외 오류는 래핑, updateTransfereeCallID 의 Get/Update/재Get 실패(141-143 포함), createFlow 와 calls 조회 실패 |

## 4. 품질 규칙
빈 호출과 자기비교 금지: 단언은 반환 오류, 상태코드, 호출 인자, 리터럴 키와 TTL 로 합니다. 고루틴 테스트는 채널로 호출 완료를 기다려 -race 에서 종료 뒤 mock 호출이 없게 합니다. 합성 값만 씁니다(203.0.113.x, 가짜 UUID, 주소는 mr.Addr()). 네트워크와 비용 호출은 없습니다. docs/conventions/testing.md 와 scripts/check-test-conventions.sh 규칙(Test_ 이름, mc 컨트롤러)을 따릅니다. prometheus 전역 메트릭 값에 의존하지 않습니다.

## 5. verify 항목
- `go mod tidy && go mod vendor && go generate ./... && go test -count=1 -race ./... && golangci-lint run` 통과, 수기 합계 61% 이상(전후 표는 agg 스크립트). vendor 는 커밋하지 않음.
- 변이 표본(제품 코드를 scratch 복사본에서만 바꿈, 변이 전후 go test 출력을 카드 코멘트에 남김): rollback 호출 제거, errors.Wrap 을 nil 반환으로, err != nil 뒤집기, 캐시 키 접두사 변경, 캐시 TTL 변경, QueueBind 루프를 첫 패턴 뒤 중단, errors.Is(ErrNotFound) 를 cerrors NotFound 로 바꾸는 db.go 의 매핑 제거. 각 변이를 잡는 테스트: rollback 제거는 attended/blind rollback 인자 단언, Wrap 제거와 err 뒤집기는 오류 경로 단언, 키 접두사와 TTL 은 cachehandler 리터럴 키와 mr.TTL 단언, QueueBind 중단은 두 번째 패턴에서 실패시켜 호출 수를 보는 subscribehandler Run 테스트, NotFound 매핑은 transferhandler 의 cerrors 타입 단언. 각 변이에서 해당 테스트가 실패해야 함.
- `git diff --name-only origin/main...HEAD` 가 bin-transfer-manager/ 안의 테스트 파일, go.mod, go.sum, 설계 문서뿐임.

## 6. 롤백
테스트와 go.mod/go.sum 변경뿐이므로 PR revert 로 함께 되돌립니다. 운영 영향 없음.
