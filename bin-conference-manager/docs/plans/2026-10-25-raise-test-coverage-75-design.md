# bin-conference-manager unit test coverage to 75% (VOIP-1652)

## 1. 분석

기준선은 origin/main accc03e00 입니다.

- 수기 합계 71.76% (1959/2730 statement), 원본 63.56%. 75.0% 는 2048, 76% 는 2075 이므로 각각 89개, 116개를 더 커버해야 합니다.
- 제외 방법: 경로에 `/gens/` 가 있거나 파일명이 `mock_` 로 시작하거나 파일 헤더(package 줄 이전)에 `Code generated` 와 `DO NOT EDIT` 가 있는 파일. 같은 블록의 중복은 최대 count 로 합칩니다.
- 패키지별 수기 커버리지(전): cmd/conference-control 0.0 (510), cmd/conference-manager 0.0 (162), internal/config 97.8 (135), models/conference 97.1 (35), models/conferencecall 90.0 (20), pkg/cachehandler 100.0 (59), pkg/conferencecallhandler 98.5 (271), pkg/conferencehandler 98.1 (467), pkg/dbhandler 89.9 (425), pkg/listenhandler 93.7 (569), pkg/subscribehandler 98.7 (77).
- cmd 672개를 전부 미커버로 두면 분모 2730 기준 최대 2058/2730 = 75.38% 이고, 도달 가능한 cmd 외 미커버는 약 75개라 89개에 모자랍니다. 따라서 cmd/conference-control 의 일부를 테스트로 확보합니다.
- cmd/conference-control 에서 제품 코드 변경 없이 테스트할 수 있는 것: 12개 `cmd*` 생성자(flag 정의), `initCommand`, `resolveUUID`, `printJSON`(직렬화 불가 값으로 오류 경로 도달), `runCreate` 의 입력 검증(핸들러 초기화 이전), 각 `run*` 의 `initHandler` 실패 전파. `initHandler` 가 성공해야 도달하는 본문(DB, Redis, RabbitMQ 연결)은 단위 테스트 대상이 아닙니다.
- cmd/conference-manager(162)는 `runMain` 이 외부 연결을 직접 맺는 구조라 제외합니다.
- 함정: `config.Get()` 은 `LoadGlobalConfig()` 이전에 호출하면 panic 이고 `cfg` 는 비공개이며 `sync.Once` 라 프로세스당 한 번만 값이 고정됩니다. 형식이 잘못된 DSN 은 mysql 드라이버가 `sql.Open` 에서 파싱 오류로 끝내 접속을 시도하지 않지만, 빈 DSN 과 기본 DSN 은 로컬 포트로 Ping 하므로 사용하지 않습니다.
- 도달 가능한 cmd 외 미커버: dbhandler 의 `PrepareFields` 실패(맵 값 안에 chan 을 넣어 직렬화 불가로 만듦)와 `NotEq{Value: <Value() 가 오류인 값>}` 로 만드는 `ToSql` 실패, listenhandler 의 `json.Marshal` 실패(`Conference.Data` 에 chan). 인자가 고정된 `ToSql` 실패와 conferencecalls, count 의 Marshal 실패는 도달 불가입니다.

## 2. 설계

범위는 테스트 파일 추가뿐입니다(제품 코드, go.mod, go.sum 무변경). 단언은 `t.Errorf`, 신규 함수는 `Test_` 접두, 테스트 값은 하드코딩된 UUID 문자열을 씁니다. 실제 네트워크 접속은 만들지 않습니다. 목표는 수기 합계 76% 이상(최소 75.0%)입니다.

### 2.1 cmd/conference-control (`main_test.go`)

- `TestMain`: `viper.Set("database_dsn", "not-a-valid-dsn")` 후 `config.LoadGlobalConfig()` 한 번.
- 각 테스트는 쓰는 viper 키를 `old := viper.Get(k)` 로 저장하고 `t.Cleanup` 에서 `viper.Set(k, old)` 로 되돌립니다. `viper.Reset` 은 DSN 도 지우므로 쓰지 않습니다. 각 케이스가 읽히는 키를 모두 명시 설정하므로 `AutomaticEnv` 가 읽는 환경변수에 의존하지 않습니다.
- `root.Execute()` 는 쓰지 않고 `PersistentPreRunE` 를 직접 호출합니다. `Execute` 를 쓰면 `BindPFlags` 가 sub 명령의 flag 를 viper 에 묶어 이후 테스트가 오염됩니다.
- `Test_resolveUUID`: 빈 값, 잘못된 UUID, 정상 UUID 와 오류 메시지.
- `Test_printJSON`: 정상 값은 nil, 직렬화 불가 값(chan)은 `failed to marshal JSON`.
- `Test_command_constructors`: 12개 `cmd*` 의 `Use`, `RunE` 존재, flag 이름과 기본값(`type=conference`, `format=wav`, `lang=en`, `limit=100` 등).
- `Test_initCommand`: 루트 persistent flag, `conference` 아래 12개 명령과 `migrate` 존재, `PersistentPreRunE` 성공과 DSN 유지.
- `Test_runCreate_validation`: customer-id 누락과 형식 오류, id 형식 오류, type 오류, pre/post flow 형식 오류 각각의 오류 메시지. 모두 `initHandler` 이전에 끝납니다.
- `Test_run_initHandlerFailure`: 12개 `run*` 이 잘못된 DSN 때문에 `failed to initialize handlers` 로 끝남.

### 2.2 pkg/dbhandler (`query_build_errors_test.go`)

쿼리 구성 실패가 DB 호출 전에 오류 문자열과 함께 반환되는지 확인합니다: conference Create/Update 와 conferencecall Update 의 `PrepareFields` 실패, 두 List 의 `could not build query`. sqlmock 에 기대 구문이 없으므로 DB 로 전달되면 실패합니다.

### 2.3 pkg/listenhandler (`response_serialization_test.go`)

`Conference.Data` 에 chan 을 가진 응답을 gomock 핸들러가 반환하게 해 conferences 11개 라우트와 direct-hash 1개의 `json.Marshal` 실패 응답 코드를 라우트별로 단언합니다. 현재 응답 코드는 500(list, post, put, direct-hash)과 400(그 외)이며 현재 동작을 그대로 기록하는 특성 테스트입니다.

### 2.4 verify

- 측정: `GOFLAGS=-mod=mod go test -count=1 -coverprofile=<f> ./...` 후 위 제외 방법으로 수기 합계를 계산. 전 패키지 `-race -shuffle=on` 통과, `golangci-lint run -v --timeout 5m` 0 issues, `scripts/check-test-conventions.sh` 통과, go.mod/go.sum 무변경.
- 변이 표본(컴파일되는 변이만): `resolveUUID` 의 빈 값 검사와 nil 비교 각각 무력화, `runCreate` 의 type 검증 무력화, `printJSON` 오류 분기 무력화, type flag 기본값 변경, dbhandler `PrepareFields` 오류 반환 제거, listenhandler Marshal 오류 분기 제거가 새 테스트를 실패시켜야 합니다.
