# bin-conference-manager unit test coverage to 61% (VOIP-1647)

## 1. 티켓 검토 (분석)

판정: 유효. 기준선은 origin/main d68e695b5 에서 재측정했고 카드 수치와 일치합니다.

- 수기 합계 53.75% (1468/2731 statement), 원본 47.6%. 60% 까지 약 170개, 61% 권장선까지 약 200개를 더 커버해야 합니다.
- 제외 방법: 경로에 `/gens/` 가 있거나 파일명이 `mock_` 로 시작하거나 파일 헤더(package 줄 이전)에 `Code generated` 와 `DO NOT EDIT` 가 있는 파일. 같은 블록의 중복은 최대 count 로 합칩니다.
- 패키지별 전(수기): cmd/conference-control 0.0 (510), cmd/conference-manager 0.0 (162), internal/config 43.7 (135), models/conference 97.1, models/conferencecall 90.0, pkg/cachehandler 0.0 (62), pkg/conferencecallhandler 67.5 (271), pkg/conferencehandler 75.3 (465), pkg/dbhandler 79.8 (425), pkg/listenhandler 74.3 (569), pkg/subscribehandler 80.5 (77).
- 구조 확인: cmd/ 두 main 은 672 statement 이며 cobra 와 외부 연결(DB, Redis, RabbitMQ)을 직접 엮는 main 이라 단위 테스트 대상에서 제외합니다. pkg/cachehandler 는 go-redis 클라이언트를 직접 생성하며 이 모듈에는 miniredis 가 없어 go.mod 변경 없이는 테스트할 수 없으므로 제외합니다. 둘을 빼도 나머지 오류 경로 미커버가 충분합니다(산술 근거는 3절).
- 중복 여부: 겹치는 열린 PR 없음(과거 커버리지 PR #501 이 현재 수준의 기반).
- 미커버의 성격: 제외분(cmd 672, cachehandler 62)을 뺀 미커버 529 중 약 2/3(휴리스틱 분류 기준 67~70%) 는 `uriItems` 길이 검사, JSON unmarshal 실패, 하위 핸들러/DB/RPC 오류 전파 같은 오류 경로이고, 나머지는 `conferencecallhandler.ServiceStart`(38, 정상 경로 포함 전체 0%)와 config 의 `LoadGlobalConfig`/`Init` 정상 경로 등입니다. 오류 경로의 응답(400, 500, `nil, err`)은 호출자(api-manager) 계약이므로 테스트 가치가 있습니다.
- 기존 `service_test.go` 의 `Test_ServiceStart` 는 이름과 달리 `Terminate` 를 호출합니다(이름 오류, 수정 대상 아님. 신규는 `Test_ServiceStart_*` 로 구분).
- 목표 판정: 61%(1666) 로 통일. 필요 증분 198, 제외 후 도달 가능한 미커버 529 의 37.4%.

## 2. 설계

범위는 기존 `_test.go` 에 케이스를 추가하거나 새 `_test.go` 를 만드는 것뿐입니다(제품 코드 무변경, go.mod/go.sum 무변경). 단언은 `reflect.DeepEqual` 과 `t.Errorf`(testify 금지), 신규 함수는 `Test_` 접두, gomock 컨트롤러 변수는 `mc`. 하위 호출은 gomock 만 쓰고 실제 네트워크 연결을 만들지 않습니다. 기존 테스트의 setup 을 재사용하려고 해당 영역의 기존 `_test.go` 에 케이스를 추가하는 쪽을 우선합니다.

### 2.1 확보 목표 (수기 기준, 필요 증분 198)

패키지 전체 미커버에서 `json.Marshal` 실패 블록(도메인 구조체라 실패시킬 수 없음)과 `ToResponse` 실패 분기 등 도달 불가 블록을 뺀 값을 "도달 가능"으로 씁니다.

| 대상 | 패키지 미커버 | 도달 가능(추정) | 목표 확보 |
|---|---|---|---|
| pkg/listenhandler (v1_conferences 55, v1_conferencecalls 19, v1_count 4, v1_services 4, v1_conferences_direct_hash 4, main.go 24: 404 분기 4, 오류 3분기 7, NewListenHandler 4, Run 9) | 146 | 약 110 (Marshal 실패 34 와 ToResponse 실패 2 제외) | 90 |
| pkg/conferencecallhandler (service.go 38, db.go 17, health.go 13, terminate.go 12, joined/main 8) | 88 | 약 85 | 55 |
| pkg/conferencehandler (conference.go 56, transcribe 16, recording 12, direct_hash 10, terminate 8, 기타 13) | 115 | 약 108 | 55 |
| pkg/dbhandler (`NewHandler` 4, `ApplyFields` unsupported type 2, `Add*` 성공 경로 8 은 sqlmock 으로, 캐시 `return err` 경로는 `cachehandler` gomock 으로 관찰 가능한 것) | 86 | 약 40 (squirrel `ToSql`/`PrepareFields` 실패 약 17 은 도달 불가) | 30 |
| internal/config (`once.Do` 클로저 블록 약 65 포함, `bindConfig` 오류 3 제외) | 76 | 약 73 | 45 |
| pkg/subscribehandler (callmanager.go 10, main.go 5) | 15 | 약 15 | 12 |
| 합계 | 526 | 약 438 | 287 (필요 198 대비 여유 89) |

도달 불가(제외): cmd/ 672, pkg/cachehandler 62, `bindConfig` 의 BindPFlag/BindEnv 오류 분기(플래그를 직접 등록하므로 발생 불가), `json.Marshal` 실패 블록 약 34, `errorResponse` 의 `ToResponse` 실패 2. `NewListenHandler` 와 `listenHandler.Run` 은 `sockHandler` 가 gomock 이므로 `QueueCreate` 실패와 `ConsumeRPC` 호출 구성(`QueueSubscribe` 는 호출하지 않음), 고루틴 동기화(채널)로 단위 테스트 대상에 포함합니다. 확보 목표가 모자라면 먼저 conferencehandler 와 dbhandler 의 목표를 늘립니다.

### 2.2 listenhandler 단언 표

`processRequest` 오류 처리는 3분기입니다: 핸들러가 반환한 `err` 가 `*cerrors.VoipbinError` 이면 그 상태, `dbhandler.ErrNotFound` 이면 404, 그 외는 400. 알 수 없는 URI 또는 메서드는 404(`err=nil`). 핸들러 내부가 직접 `simpleResponse(400)`/`errorResponse(err)`(일반 오류 500, 타입 오류와 `ErrNotFound` 는 해당 상태)를 반환하는 경로와 `nil, err` 를 반환하는 경로가 섞여 있으므로 케이스는 `processRequest` 로 호출해 외부 응답 코드를 단언합니다.

입력 구성: 정규식에 `^` 가 없어 `/v1/conferences/<uuid>` 는 `uriItems` 가 4개입니다. 길이 검사에 걸리는 짧은 URI 는 앞 슬래시를 뺀 `v1/conferences/<uuid>`(3개, 길이 4 미만) 또는 `v1/conferences/<uuid>/stop`(4개, 5 미만)처럼 구성해 `processRequest` 가 아닌 개별 핸들러(`h.processV1...`)를 직접 호출합니다. 라우팅이 필요한 케이스는 정상 URI 를 씁니다.

| 입력 | 대상 | 기대 | 후속 호출 |
|---|---|---|---|
| 알 수 없는 URI 또는 메서드 | processRequest | 404 | 없음 |
| `ErrNotFound` 를 반환하는 하위 핸들러 (List/Create/CountByCustomerID/DirectHashRegenerate/ServiceStart) | `errorResponse` 경로 | 404 | - |
| 타입 오류 반환 (`cerrors` NotFound 등) | errorResponse 경로와 `nil, err` 경로 모두 | 해당 상태 | - |
| 일반 오류 반환 (List/Create/Count/DirectHash/ServiceStart/conferencecalls List) | errorResponse 경로 | 500 | - |
| 일반 `errors.New` 반환 (conferences Get/Delete/Put 과 conferencecalls Get/Delete) | `errorResponse` 경로 | 500 (타입 오류와 `ErrNotFound` 는 해당 상태, 404) | - |
| 일반 오류 반환 (UpdateRecordingID/Recording*/Transcribe*/Stop) | `nil, err` | `processRequest` 가 400 | - |
| `dbhandler.ErrNotFound` 를 감싼 오류를 `nil, err` 경로(예: RecordingStop)에서 반환 | `processRequest` 의 `case Is(ErrNotFound)` | 404 | - |
| 짧은 URI (모든 ID 계열 핸들러) | 개별 핸들러 직접 호출 | 400 | 하위 핸들러 호출 없음 |
| 본문 JSON 깨짐: conferences Post, count, 그 외 | Post/count 는 직접 400, Put/Recording*/Transcribe*/HealthCheck/services 는 `nil, err` 후 400 | 400 | 하위 호출 없음 |
| `url.Parse` 실패(`/x%zz/v1/conferences?page_size=1`(정규식에 `^` 가 없어 라우팅되고 경로의 잘못된 퍼센트 인코딩으로 `url.Parse` 가 실패. `?%zz` 와 `/v1/conferences%zz` 는 각각 파싱 통과, 라우팅 불일치로 쓰지 않음. conferencecalls List 도 같은 방식)) | List 핸들러 | `nil, err` 후 400 | List 호출 없음 |
| 필터 파싱 실패(본문이 객체가 아님)와 변환 실패(uuid 필드에 잘못된 문자열, 예: `{"customer_id":"not-a-uuid"}`. 알 수 없는 키는 조용히 무시되어 실패하지 않으므로 쓰지 않음)를 서로 다른 입력으로 | conferences List, conferencecalls List | 400 | List 호출 없음 |
| `DirectHashRegenerate` 성공 경로 | `v1_conferences_direct_hash.go:33-34,39-43` | 200 과 응답 본문 | 하위 호출 1회 |
| HealthCheck 정상 | 기존 `v1_conferencecalls_test.go` 에 이미 있음 | 새 커버리지 없음. 고루틴 동기화(채널)로 기대 소진 단언만 보강(선택) | |

기존 `error_response_test.go` 가 `errorResponse` 단위를 이미 다루므로 중복하지 않습니다. `promReceivedRequestProcessTime` 는 기존 테스트가 `processRequest` 로 간접 사용 중이므로 신규 테스트도 같은 경로를 쓰며 `-count=2 -shuffle=on` 으로 충돌을 확인합니다.

### 2.3 그 외 패키지

- conferencecallhandler: `HealthCheck` 는 반환값 없는 비동기 함수이므로 고루틴이 `ConferenceV1ConferencecallHealthCheck(retryCount+1)` 를 호출하는지를 채널로 동기화해 단언합니다(오류 전파가 아님). `ServiceStart` 정상 경로(PreFlowID 유무 두 갈래, 생성된 액션 목록과 `Service` 반환값 단언), `conferenceHandler.Get` 실패, `Create` 실패. db.go/health.go/terminate.go 는 dbhandler/RPC 오류 시 오류 전파와 후속 호출 부재를 단언합니다.
- conferencehandler: Create/Get/Delete/Update, recording, transcribe, direct_hash, terminate 의 DB/RPC 오류 전파, 상태별 분기(이미 종료, 진행 중 아님 등)를 현재 코드의 반환 경로를 읽고 케이스로 만듭니다.
- dbhandler: 쿼리 오류 경로는 `sqlmock`(기존 `ctx_rows_test.go` 가 이미 사용)으로 만들고, 캐시 오류 경로는 `cachehandler` gomock 으로 만듭니다. 공유 인메모리 `dbTest` 를 닫거나 drop 하지 않습니다. `_ = h.conferenceUpdateToCache(...)` 처럼 캐시 오류를 버리는 호출의 실패는 관찰 불가라 대상이 아니며, 캐시 오류는 오류를 반환하는 `SetToCache`/`UpdateToCache` 경로로 한정합니다.
- subscribehandler: `callmanager.go:30-31,59-60` 의 조회 실패 시 의도적 `return nil` 분기(후속 호출 없음)를 포함하고, 이벤트 데이터 unmarshal 실패와 하위 핸들러 오류에서 후속 호출이 없음을 단언합니다.
- config: 전역 `cfg`/`once`/viper 때문에 테스트마다 `cfg=nil`, `once=sync.Once{}`, `viper.Reset()` 을 초기화하고 `t.Cleanup` 으로 복원하며(`once` 는 값 복사 시 copylocks 경고가 나므로 `cfg` 포인터만 저장해 복원하고 `once = sync.Once{}` 로 리셋하며, `Bootstrap` 이 바꾸는 전역 logrus 포맷터와 레벨도 복원), `t.Setenv` 로 환경변수를 격리합니다(`t.Parallel` 금지). 대상은 `Get()` 미초기화 panic, `Init`(플래그와 환경변수 반영, 가장 큰 덩어리), `LoadGlobalConfig`(자체로는 바인딩을 하지 않으므로 `Bootstrap`/`bindConfig` 를 먼저 거쳐 환경변수를 반영)(두 함수가 `once` 를 공유하므로 각각 리셋 후 실행). 케이스마다 새 `cobra.Command` 를 만들고 `Init` 전에 `RegisterFlags` 를 호출하며(격리를 위해), 환경변수 경로와 함께 `cmd.Flags().Set` 으로 Changed 상태를 만든 플래그 값 케이스를 하나 포함해 `BindPFlag` 제거 변이를 죽입니다. 기본값이 아닌 서로 다른 값을 환경변수로 주어 Config 필드 매핑이 바뀌면 실패하게 합니다.

금지: 반환값을 단언하지 않는 빈 호출, 입력에서 기대값을 그대로 복제하는 자기비교.

## 3. verify 항목

1. 수기 합계 61% 이상(1666/2731 이상). 파일별 수치는 `agg.py` 수기 기준으로 뽑고 위 2.1 표의 목표 대비 실제를 PR 본문 표와 카드에 기록하고, 구현 중 파일별 `agg.py` 중간 측정을 남깁니다. 2.1 표의 파일별 괄호 수치는 도달 가능 추정이며 패키지 미커버 실측과 다를 수 있습니다. dbhandler 는 실측상 도달 가능이 약 69 로 표의 40 은 보수적입니다.
2. 실행: `GOFLAGS=-mod=mod go test -count=1 -race ./...`, `go test -count=2 -shuffle=on ./...`, `golangci-lint run -v --timeout 5m`, `git fetch origin main` 후 저장소 루트에서 `bash scripts/check-test-conventions.sh`(Test_ 접두, testify 금지, gomock 컨트롤러 `mc` 세 규칙). 실행 전후 `git diff --stat -- go.mod go.sum` 이 비어 있어야 함.
3. 모든 신규 gomock 컨트롤러는 `mc` 이며 `defer mc.Finish()` 로 기대가 소진됨을 확인합니다. 고루틴을 띄우는 경로(HealthCheck, listenhandler Run, db.go:69-71 의 Create 후처리(`ServiceStart` 성공 케이스도 해당하며 `ConferenceV1ConferencecallHealthCheck(..., 0, ...)` 기대를 채널로 동기화), conferencehandler/remove.go:34-39, subscribehandler/main.go Run)는 `time.Sleep` 대신 채널/WaitGroup 으로 동기화해 flaky 를 막습니다. HealthCheck 정상 경로는 `retryCount+1` 이 아니라 `0` 으로 호출하는 점을 별도로 단언합니다. subscribehandler.Run 의 `QueueCreate`/`TopicCreateWithKind`/`QueueBind`/`ConsumeMessage` 오류 3분기와 `topicPatterns` 순회도 케이스에 포함합니다.
4. 변이 검사(scratch 복사본, 패키지당 1곳 이상): listenhandler 의 `uriItems` 길이 비교(`<` 를 `<=`), `processRequest` 의 `case Is(ErrNotFound)` 블록 제거(순서 교체는 두 case 가 같은 `errorResponse` 를 호출해 동치 변이라 쓰지 않음)와 `default` 의 `simpleResponse(400)` 을 500 으로 변경, 짧은 URI 케이스가 잡는 `< 4` 를 `< 3` 으로 변경, conferencehandler/conferencecallhandler 의 오류 전파를 `nil` 로 교체, dbhandler 의 `ConferenceUpdate` 등 `%w` 사이트를 `%v` 로 교체(신규 테스트가 `errors.Is` 로 단언해야 죽음. 기존 `%v` getter 는 대상 아님), config 의 필드 매핑 교체(`LoadGlobalConfig` 와 `Init` 양쪽에서 각각 하나), subscribehandler 의 unmarshal 오류 분기 제거. 각 변이에서 신규 테스트가 실패해야 하며 결과를 카드에 표로 기록.
5. `git diff --stat origin/main...HEAD` 가 bin-conference-manager/ 안의 `_test.go` 와 이 문서만 포함.

## 4. 위험

- 오류를 주입하기 전에 해당 오류 경로가 반환값을 검사 없이 사용하지 않는지 코드를 읽어 확인합니다.
- 핸들러마다 오류 반환 방식(`errorResponse` 또는 `nil, err`)이 다르므로 테스트는 `processRequest` 가 돌려주는 외부 응답 코드를 단언합니다.
- 전역 상태(config 의 once/cfg/viper, prometheus 메트릭)는 `-shuffle=on -count=2` 로 순서 의존을 확인합니다.
- `GOFLAGS=-mod=mod` 가 go.mod/go.sum 을 바꾸지 않았음을 확인합니다.
