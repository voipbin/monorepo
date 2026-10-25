# VOIP-1652 bin-campaign-manager 단위 테스트 커버리지 상향 설계

## 1. 이슈 분석 (착수 전 티켓 검토)

- 기준: origin/main accc03e (서비스 경로의 마지막 커밋은 VOIP-1651 #1572, 492de9e4f). 수기 코드(gens/, mock_*.go, DO NOT EDIT 헤더 파일 제외, 같은 블록 중복은 최대 count) 74.45% (2608/3503), 75.0% 이상은 2628, 권장 76%는 2663 statement 이상 필요. cmd 472개는 분모에 포함된 채 계산한다.
- 티켓 수치는 재현됨. 최근 경로 커밋은 VOIP-1651(#1572, 머지)로 campaignhandler/campaigncallhandler/outplanhandler 를 이미 올렸으며 중복 작업 PR 은 열려 있지 않음.
- 패키지별 미커버(statement): cmd/campaign-control 328, cmd/campaign-manager 144 (main, 외부 연결 의존으로 제외 대상), pkg/listenhandler 156, pkg/dbhandler 96, pkg/cachehandler 67 (테스트 0건), campaignhandler 30, campaigncallhandler 23, subscribehandler 17, models/*/webhook.go 24.
- 구조 확인: cachehandler 는 go-redis/redis/v8 클라이언트(v9 아님)를 필드로 가진 구조체이며 miniredis 로 네트워크 없이 검증 가능(같은 버전 v2.36.1 이 main 의 다른 서비스에 있음). webhook.go 의 ConvertWebhookMessage 는 기존 webhook_test.go 로 이미 커버되어 있고 미커버 24 statement 는 모두 CreateWebhookEvent 이다(json.Marshal 오류 분기의 도달 가능성은 모델마다 다르므로 억지로 목표에 넣지 않는다). listenhandler 미커버는 대부분 JSON 파싱 실패와 하위 handler 오류 반환 경로.
- listenhandler/main.go 미커버: errorResponse 의 ToResponse 실패 분기(L116-117), NewListenHandler 와 Run(L135-164, sockhandler mock 필요), processRequest 의 default 404 분기(L290-297)와 error 분기 일부(L314, L316).
- 측정 방법은 scratch 의 cov.py 와 동일하다(mode set, 같은 블록은 최대 count, 헤더와 파일명 필터, cmd 472개는 분모에 포함).
- 위험: miniredis 추가에 따른 go.mod/go.sum 변경(gopher-lua 등 간접 의존 포함, tidy 후 추가 diff 없어야 함), cachehandler Connect 실패 시험은 닫은 miniredis 주소만 쓰고 고정 외부 주소를 쓰지 않는다.
- 달성 가능성: 확정 경로는 cachehandler 도달 가능분 66(67 중 setSerialize 의 json.Marshal 오류 반환 1 제외) + CreateWebhookEvent 성공 경로 21(cp.out 기준 파일당 미커버 블록 3+3+1+1 중 json.Marshal 오류 반환 블록 1 statement 를 뺀 7, 3개 파일) = 87 이고 76% 필요량 55(75% 하한은 20)를 넘는다. 이 경로만으로 listenhandler 와 dbhandler 없이도 목표에 도달한다. listenhandler(파일별 미커버 campaigns 68, outplans 39, main 26, campaigncalls 19)는 JSON 파싱 실패와 하위 오류 반환 분기를 추가해 여유를 확보하는 용도이며 얼마나 회수될지는 구현 후 측정한다.
- 결론: cmd 472개를 제외해도 위 확정 경로(cachehandler, CreateWebhookEvent)만으로 필요량(55 이상)을 채울 수 있어 목표 달성은 가능. 제품 코드 변경 불필요.

## 2. 계획

변경은 bin-campaign-manager 안의 신규 `*_test.go` 와 test-only 모듈(miniredis) 반영을 위한 go.mod/go.sum 뿐이다.

1. pkg/cachehandler/handler_test.go: miniredis 로 Outplan/Campaign/Campaigncall Set 후 Get 왕복(저장된 JSON 키와 TTL 확인), 없는 키 Get 오류, 손상된 JSON Get 오류, 서버 종료 후 Set 오류, Connect 성공과 실패, NewHandler 필드.
2. models/{campaign,campaigncall,outplan}/webhook_test.go 에 CreateWebhookEvent 시험 추가: 결과 JSON 을 파싱해 공개 키와 값이 있고 내부 전용 필드(Campaign 의 execute, flow_id 등)가 없는지 단언한다. ConvertWebhookMessage 는 기존 시험이 있어 중복하지 않는다.
3. pkg/listenhandler: 기존 test 스타일을 따라 campaigns/outplans/campaigncalls 의 잘못된 JSON 본문과 하위 handler 오류에 대해 오류를 반환하고 하위 mock 이 호출되지 않거나 오류가 전파됨을 검증하는 표 주도 테스트. main.go 의 errorResponse ToResponse 실패, Run/NewListenHandler(sockhandler mock), processRequest default 404 분기.
4. 부족하면 dbhandler(기존 테스트와 같은 sqlite 인메모리 스키마 방식)의 오류 경로와 subscribehandler 를 추가.

세부: errorResponse 의 ToResponse 실패 분기는 (*cerrors.VoipbinError)(nil) 을 error 로 넘겨 500 을 기대하는 방식으로 시도한다(어려우면 제외). Run 은 ConsumeRPC 가 goroutine 에서 호출되므로 채널로 동기화한다. processRequest 경유 시험의 기대값은 하위 일반 오류 400, ErrNotFound 와 typed error 는 errorResponse 매핑(404 등), 알 수 없는 URI 는 404 로 명시한다. Connect 실패와 서버 종료 후 Set 오류 시험은 닫은 miniredis 주소와 MaxRetries 를 줄인 클라이언트를 쓴다. 하위 mock 미호출 단언은 EXPECT() 를 등록하지 않는 방식으로 한다. CreateWebhookEvent 의 Marshal 오류는 잘못된 json.RawMessage 로 시도할 수 있으나 목표에 의존하지 않는다.

verify 항목: (a) 수기 합계 76% 이상, (b) 새 테스트는 값을 만들어 같은 값을 비교하는 방식이 아니라 외부 입력(JSON 문자열, redis 상태)에 대한 관측 결과를 단언, (c) 변이 표본(수동 1회씩, 어떤 변이로 어떤 테스트가 실패했는지 PR 본문에 한 줄씩 기록): cachehandler 키 접두어 변경, webhook 필드 누락, listenhandler 오류 전파 제거 시 해당 테스트가 실패, (d) 측정은 cov.py 기준(cmd 472 분모 포함)을 PR 본문에 적고, go.mod/go.sum 는 miniredis 와 그 간접 의존만 변경되고 `go mod tidy` 와 `go mod vendor` 후 추가 diff 없음, (e) `go test ./...`, golangci-lint 통과, PR 파일은 bin-campaign-manager/ 안에만 존재.
