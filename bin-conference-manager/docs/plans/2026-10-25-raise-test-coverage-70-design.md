# bin-conference-manager unit test coverage to 70% (VOIP-1651)

## 1. 티켓 검토 (분석)

판정: 유효. 기준선은 origin/main 99b59c386 에서 `GOFLAGS=-mod=mod go test -count=1 -coverprofile` 로 재측정했고 카드 수치와 일치합니다.

- 수기 합계 69.38% (1894/2730 statement), 원본 61.45%. 70%(1911) 까지 17개, 71% 권장선(1939)까지 45개를 더 커버해야 합니다.
- 수기 집계 제외 규칙(분모에서 뺌): 경로에 `/gens/` 가 있거나 파일명이 `mock_` 로 시작하거나 파일 헤더(package 줄 이전)에 `Code generated` 와 `DO NOT EDIT` 가 있는 파일. 같은 블록의 중복은 최대 count 로 합칩니다.
- 패키지별 전(수기, 커버/전체): cmd/conference-control 0/510, cmd/conference-manager 0/162, internal/config 132/135, models/conference 34/35, models/conferencecall 18/20, pkg/cachehandler 0/59, pkg/conferencecallhandler 267/271, pkg/conferencehandler 452/467, pkg/dbhandler 382/425, pkg/listenhandler 533/569, pkg/subscribehandler 76/77.
- 이번 대상 선정(분모는 그대로 두고 테스트 대상에서만 제외): cmd/ 중 DB, Redis, RabbitMQ 에 직접 연결하는 `init*`/`run*` 경로는 단위 테스트 대상에서 뺍니다. cmd 에는 순수 함수(`resolveUUID`, `printJSON`, cobra 생성자)가 있어 완전히 불가능하지는 않으므로, 아래 보조 후보로도 모자랄 때의 예비 경로로 둡니다.
- pkg/cachehandler(59)는 이 서비스 go.mod 에 miniredis 가 없어 VOIP-1647 에서 제외했던 대상입니다. miniredis v2.36.1 은 기준 커밋의 `*/go.mod` 12곳 중 11곳(v2.38.0 은 bin-api-manager 1곳. 예: bin-transfer-manager, bin-contact-manager)에 있고 transfer-manager 는 같은 go-redis v8.11.5 구성의 `pkg/cachehandler/handler_test.go` 선례가 있습니다. 이 서비스에는 `go.mod`/`go.sum` 추가와 `go mod tidy`/`go mod vendor` 가 필요합니다.
- 중복 여부: 2026-10-25 에 `gh pr list --state all --search "conference-manager coverage"` 로 확인, 열린 커버리지 PR 없음. 이력은 #1526(VOIP-1647, 머지)과 그 이전의 NOJIRA 커버리지 PR.
- 달성 산술: cachehandler 59 는 `Conference.Data`(`map[string]any`)에 채널 값을 넣어 `json.Marshal` 실패도 만들 수 있어 이 케이스가 있어야 59/59 입니다. cmd 를 분모에 그대로 두고 1894+59=1953/2730=71.5% 이며 71% 권장선(1939) 대비 여유는 14개, 70% 선(1911) 대비 42개입니다. 놓칠 경우를 대비한 보조 후보를 순서대로 둡니다: pkg/conferencehandler/remove.go 미커버 7(고루틴 안 로그 1개는 타이밍 때문에 결정적 검증이 어려워 제외, 확보 가능 6), pkg/dbhandler/conference.go 24, pkg/listenhandler/v1_conferences.go 22, pkg/dbhandler/conferencecall.go 19.

## 2. 설계

범위는 `_test.go` 추가와 go.mod/go.sum 의 miniredis v2.36.1(및 ai-manager, contact-manager 와 같은 전이 의존성 버전) 추가뿐입니다. 제품 코드 무변경. 단언은 `reflect.DeepEqual` 과 `t.Errorf`, 신규 함수는 `Test_` 접두, gomock 컨트롤러 변수는 `mc`. 네트워크는 miniredis 루프백만 씁니다.

- pkg/cachehandler (신규 handler_test.go):
  - Set 후 `mr.Get("conference:conference:<id>")` 와 `conference:conferencecall:<id>` 로 키와 JSON 내용을 직접 단언하고, `mr.TTL(key) == 24*time.Hour` 를 두 Set 각각에 단언합니다. Get 은 왕복으로 `reflect.DeepEqual` 합니다.
  - ConferencecallGetByReferenceID 는 `conference:conferencecall:reference_id:<id>` 를 `mr.Set` 으로 주입해 조회합니다.
  - 왕복 픽스처는 `time.Date(..., time.UTC)` 또는 nil 시각만 쓰고(`time.Now()` 금지, monotonic 값 때문), `Data` 는 문자열 값만, 슬라이스 필드는 nil 또는 비어 있지 않은 값만 씁니다(빈 슬라이스는 왕복 후 nil 과 구분되지 않아 DeepEqual 이 깨질 수 있음)(JSON 왕복 시 숫자가 float64 가 되므로).
  - `ConferenceSet` 에 `Data: map[string]any{"x": make(chan int)}` 를 주면 `*json.UnsupportedTypeError` 와 `mr.Exists(key)` false 를 단언합니다.
  - 키 부재는 `errors.Is(err, redis.Nil)` 과 결과 nil, 저장값이 JSON 이 아닐 때는 `*json.SyntaxError` 와 결과 nil 을 단언합니다.
  - 서버 종료(`mr.Close()`) 후 Set, Get, Connect 는 `handler{Cache: redis.NewClient(&redis.Options{Addr: ..., MaxRetries: -1, DialTimeout: 200ms})}` 를 직접 만들어(transfer-manager 의 `newTestHandler` 선례) 재시도와 대기를 막고 `err != nil` 을 단언합니다. `Connect()` 는 인자가 없고 내부에서 `context.Background()` 를 쓰므로 `context.WithTimeout` 은 쓰지 않습니다.
  - NewHandler 는 DB 인덱스를 0 이 아닌 값으로 만들어 `handler` 타입으로 단언해 Addr, Password, DB 전달을 확인하고, Connect 성공과 Set/Get 왕복을 확인하고, DB 인덱스 n 으로 만든 핸들러가 Set 하면 `mr.DB(n)` 에 키가 생기고 `mr.DB(0)` 에는 없음을 단언합니다(Password 는 `mr.RequireAuth` 로 전달을 검증).
- pkg/conferencehandler (remove_test.go 에 별도 `Test_` 함수로 추가(기존 테이블에는 오류 필드가 없음), 성공 경로 고루틴 케이스는 추가하지 않음): RemoveConferencecallID 의 DB 제거 실패와 재조회 실패(`errors.Is`, 결과 nil, `PublishWebhookEvent` 기대를 걸지 않아 호출 시 실패), removeConferencecallIDTypeConference 의 Destroy 실패(`CallV1ConfbridgeDelete` 는 성공으로 두고 — 이 호출의 오류는 Destroy 가 삼키므로 실패 케이스가 아님 — `ConferenceEnd` 오류, 이후 `ConferenceGet`/webhook 미호출).
- 예상: cachehandler 58 과 remove.go 6 으로 1894+64=1958 (71.7%). 실측이 1939 미만이면 1절의 보조 후보 순서대로 추가하며 기존 sqlmock/gomock 방식을 따릅니다.

## 3. verify 항목

- 수기 합계는 base 측정과 같은 방법(위 제외 규칙을 구현한 스크립트, 방법을 PR 본문에 기재)으로 계산해 1939 이상(71.0%), 수치는 PR 본문에 적습니다.
- 모든 신규 테스트는 반환값 또는 저장 상태(키, TTL)를 단언합니다(단순 호출만 하는 테스트 금지).
- 변이 확인(scratch 복사본에서만 하고 worktree 제품 코드는 건드리지 않음, 원복 확인): `getSerialize` 의 Unmarshal 오류 반환과 `Connect` 의 오류 반환을 nil 로, `NewHandler` 의 Addr/Password/DB 전달 제거, cachehandler 의 키 접두 3종(conference, conferencecall, reference_id)과 두 Set 의 TTL 을 각각 바꾸면, remove.go 의 세 오류 반환(DB 제거 실패, 재조회 실패, Destroy 실패)을 각각 nil 로 바꾸고 NewHandler 의 DB 전달을 빼면 해당 테스트가 실패해야 합니다.
- 루트 CLAUDE.md 5단계(`go mod tidy`, `go mod vendor`, `go generate ./...`, `go test ./...`, `golangci-lint run -v --timeout 5m`)와 `go test -race` 로 remove 패키지 확인.
- `git diff origin/main...HEAD -- go.mod go.sum` 의 추가 줄이 miniredis 와 `yuin/gopher-lua`(go.mod 는 require 2줄, go.sum 은 각 2줄)뿐이고 기존 모듈 버전 변경과 `go` 지시문 변경이 0줄.
- `git diff --name-only origin/main...HEAD` 가 `*_test.go`, go.mod, go.sum, docs/plans 뿐이며 `go generate` 후 mock 차이가 없음. go.mod 의 miniredis 는 직접 require 이고 버전과 전이 의존성이 ai-manager, contact-manager 의 go.mod/go.sum 과 일치.
- VOIP-1416 과 겹칠 수 있는 패키지(cachehandler, conferencehandler, dbhandler)는 PR 본문에 명시만 하고, 착수 시 `git fetch origin main` 으로 충돌을 다시 확인합니다.
