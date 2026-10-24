# VOIP-1647 bin-webhook-manager 단위 테스트 커버리지 61% 이상 설계

## 1. 티켓 검토 (이슈 분석)

- 판정: 유효. origin/main 기준 수기 코드 48.6% (505/1039), 원본 43.2%로 실측하여 이전 측정(48.5%, 43.1%)와 소수점 차이만 있음.
- 관련 PR/커밋: 이 경로의 최근 커밋은 lint, 의존성, 로그 변경뿐이며 커버리지 작업은 없음. VOIP-1647 열린 PR 없음. 과거 PR 493(NOJIRA-Test-coverage-webhook-manager)은 머지됨.
- 수기 합계 60%에 필요한 추가 statement: 약 120개 (61% 목표는 약 130개).
- 테스트하기 어려운 구조: cmd/webhook-manager, cmd/webhook-control 의 main 은 RabbitMQ, DB, 시그널에 묶여 있어 제외(합계 302 stmt). 나머지로 달성 가능한지 아래에서 산정.
- 리뷰 확인: 실행마다 수기 합계가 1 stmt 흔들림(504~505).

## 2. 대상과 산정 (미커버 statement)

| 대상 | 미커버 | 방법 |
|---|---|---|
| internal/config | 61 | cobra 명령에 bindConfig/Bootstrap, 환경변수 주입 후 LoadGlobalConfig. 패키지 내부에서 sync.Once 를 재설정 |
| pkg/cachehandler (handler.go, main.go) | 39 | miniredis(기존 테스트가 이미 사용)로 AccountSet/Get 왕복, 키 없음, 손상된 JSON, 서버 중단 오류, NewHandler/Connect |
| models/activeflow | 17 | IsPositive 경계(nil, Deleted, 빈 URI), Marshal/Unmarshal 왕복과 Tm 의 unix-nano 표현, 잘못된 JSON 오류 |
| pkg/webhookhandler message.go | 33 | h.httpClient 에 RoundTripper 스텁 주입(네트워크 없음, 공인 IP 리터럴 URL 사용). 헤더(서명, Content-Type), 5xx 외 응답 비재시도, 5xx 3회 재시도 후 오류, 전송 오류 재시도, 잘못된 method, 사설 IP 거부 |
| pkg/webhookhandler main.go | 14 | newSafeHTTPClient 의 DialContext 가 사설 IP 리터럴과 잘못된 주소를 거부, CheckRedirect 가 len(via)=2 는 통과, 3 은 거부(main.go 의 `>= 3` 경계)하고 사설 대상 리다이렉트를 거부. 공인 IP 로 dial 하는 마지막 줄 1개는 실제 연결이 필요해 제외(최대 14) |
| pkg/cachehandler activeflow.go | 4 | 서버 중단 오류 경로, nil 항목 거부, 손상된 JSON (Marshal 오류 경로는 도달 불가라 제외) |

표의 합계는 약 170 statement 이나 아래 10항의 도달 불가를 빼면 약 162 statement, 예상 수기 합계 약 64%(필요 129, 여유 약 30). config 케이스는 필수. 하한 61%에 여유가 있음. 부족하면 subscribehandler, listenhandler 의 오류 분기를 추가.

## 3. Verify 항목 (설계 단계에서는 코드 없음)

1. 수기 합계 61% 이상 (agg 스크립트: gens/, mock_*.go, DO NOT EDIT 제외, 중복 블록 최대 count).
2. `go test -race -count=1 ./...` 전부 통과, golangci-lint 통과, scripts/check-test-conventions.sh 통과.
3. 비테스트 .go 파일 변경 0 (`git diff --stat origin/main...HEAD` 가 bin-webhook-manager 의 _test.go 와 이 문서뿐).
4. 변이 표본: 제품 코드를 임시로 바꿔(예: 5xx 재시도 횟수, 서명 헤더 생략, IsPositive 조건, 사설 IP 차단) 새 테스트가 실패함을 확인하고 원복.
5. 자기비교 금지: 기대값은 리터럴 또는 독립 계산(예: 알려진 HMAC 벡터)으로 쓰고 제품 함수 출력으로 만들지 않음.
6. 5xx/전송오류 재시도는 time.Sleep 1초 3회이므로 해당 케이스는 최소화하고 t.Parallel 사용.
7. 합성값만 사용. 사설/문서용 대역(TEST-NET 포함)은 차단 대상이므로 성공 경로에는 공인 IP 리터럴(예: 8.8.8.8)을 쓰되 RoundTripper 스텁이라 실제 연결은 하지 않음.
8. 전역 상태: config 테스트는 t.Setenv, viper.Reset(), once 재설정을 쓰고 t.Parallel 을 쓰지 않으며 종료 시 원복(logrus 포매터/레벨 포함). sendMessage nil httpClient 분기(1 stmt)와 서버 중단 지연은 실제 연결/지연 위험이 있어 포기하거나 제한 시간 내에서만 시험.
9. 5xx 외 응답 비재시도 확인, 재시도 케이스마다 약 3초이므로 병렬 실행하고 케이스마다 별도 RoundTripper 스텁을 사용.
10. 도달 불가 제외: config 의 BindPFlag/BindEnv 오류 분기와 Bootstrap 의 errBind 분기, cachehandler setSerialize/활성플로우 Marshal 오류 분기. 도달 가능 합계는 약 160~166 stmt(예상 약 64%).
11. 네트워크 없는 resolve 실패는 호스트가 빈 주소(`:80`)로 시험하며 외부 DNS 를 타는 호스트명(.invalid 등)은 쓰지 않음.

## 4. 테스트 계획 (파일과 케이스)

- internal/config/main_test.go: (a) 환경변수 7개를 t.Setenv 로 주입하고 cobra 명령에 bindConfig 후 LoadGlobalConfig 가 각 필드를 리터럴 기대값으로 채움(RedisDatabase 정수 변환 포함). (b) 플래그 값이 환경변수보다 우선하는지. (c) LoadGlobalConfig 는 once 로 첫 값을 고정하므로 두 번째 호출이 새 환경변수를 무시함. (d) Bootstrap 이 플래그 7개를 등록하고 logrus 레벨을 Debug 로 설정(종료 시 원복). 각 테스트 시작에 viper.Reset() 과 once 재설정, 직렬 실행.
- models/activeflow/activeflow_test.go: IsPositive 표(nil, 정상, Deleted, 빈 URI). MarshalJSON 이 tm 을 unix-nano 정수로 쓰는 리터럴 JSON 검증, UnmarshalJSON 왕복(UTC, TMDelete 포함/미포함), 잘못된 JSON 오류.
- pkg/cachehandler/handler_test.go: AccountSet/Get 왕복(키 형식 `webhook.account:<id>`을 miniredis 에서 리터럴로 확인, TTL 24h), 없는 키 오류, 손상 JSON 오류, 서버 종료 후 Set 오류. NewHandler 가 인자를 보존하고 Connect 가 miniredis 에 성공/종료 후 실패. activeflow.go: ActiveflowWebhookGet/Set 서버 종료 오류, Set(nil) 거부, 손상 JSON 의 Get 오류.
- pkg/webhookhandler/message_test.go 추가: RoundTripper 스텁이 요청을 기록. 서명 헤더(알려진 HMAC 벡터 리터럴 사용)와 Content-Type 유무(secret 빈 값/data nil), 5xx 외 응답(400, 404, 410, 429 표본)은 호출 1회로 nil, 5xx 는 호출 3회 후 오류 메시지, 전송 오류 3회 후 마지막 오류, 첫 시도 실패 후 성공(재시도 성공), 잘못된 method 오류, 사설 IP/빈 URL 거부 시 호출 0회.
- pkg/webhookhandler/main_test.go: newSafeHTTPClient 의 DialContext 가 사설 IP, 잘못된 host:port, 빈 호스트를 거부(연결 시도 없음), CheckRedirect len(via) 2 통과/3 거부/사설 대상 거부.
- 컨벤션: Test_ 접두, gomock controller 이름 mc, 테이블 기반, 합성값. 프로덕션 코드 변경 없음.

## 5. 구현 시 주의와 변이 목록

- RoundTripper 스텁은 응답마다 `Body: io.NopCloser(...)` 를 채운다(message.go 가 resp.Body 를 읽음).
- 빈 body 의 HMAC 기대값은 제품 함수가 아닌 독립 도구(openssl)로 구한 리터럴을 쓴다.
- config 테스트 정리 시 globalConfig 도 초기화한다.
- 변이 표본: 5xx 기준 `>= 500` 을 `>= 400`, 재시도 횟수 3 을 2, 서명 헤더 생략, 캐시 키 접두사, Set TTL 24h, Tm 의 UnixNano 변환, IsPositive 조건, 사설 IP 목록 항목 제거, CheckRedirect 경계 `>= 3`.
12. 변이마다 대응 입력을 명시: 사설 IP 항목 제거 변이는 해당 대역 대표 IP(10.x, 127.x, 169.254.x, ::1 등)를 각각 쓰는 표 테스트가 잡음. config 변이: 플래그 우선순위 반전, once 제거. 집계 스크립트는 저장소에 넣지 않고 scratch 에서 사용하며 PR 본문에 제외 규칙을 서술.
