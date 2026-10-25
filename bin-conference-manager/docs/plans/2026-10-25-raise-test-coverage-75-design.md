# bin-conference-manager unit test coverage to 75% (VOIP-1652)

## 1. 티켓 검토 (분석)

판정: 유효. origin/main accc03e00 에서 재측정했고 카드 수치와 일치합니다.

- 수기 합계 71.76% (1959/2730 statement), 원본 63.56%. 75.0% 는 2048, 76% 는 2075 이므로 각각 89개, 116개를 더 커버해야 합니다.
- 제외 방법: 경로에 `/gens/` 가 있거나 파일명이 `mock_` 로 시작하거나 파일 헤더(package 줄 이전)에 `Code generated` 와 `DO NOT EDIT` 가 있는 파일. 같은 블록의 중복은 최대 count 로 합칩니다.
- 패키지별 전(수기): cmd/conference-control 0.0 (510), cmd/conference-manager 0.0 (162), internal/config 97.8 (135), models/conference 97.1 (35), models/conferencecall 90.0 (20), pkg/cachehandler 100.0 (59), pkg/conferencecallhandler 98.5 (271), pkg/conferencehandler 98.1 (467), pkg/dbhandler 89.9 (425), pkg/listenhandler 93.7 (569), pkg/subscribehandler 98.7 (77).
- 구조 확인: cmd 를 제외한 statement 는 2058개이고 그중 1959 가 커버되어 미커버는 99개뿐입니다. cmd 672개를 전부 미커버로 두면 분모 2730 기준 최대 2058/2730 = 75.38% 라서 75.0% 는 99 중 89 를 채워야 하고 76% 는 불가능합니다(이 가정은 cmd/conference-control 테스트를 추가하면 풀립니다). 99 중 도달 가능분은 리뷰어 실측으로 약 75개입니다: dbhandler 43 중 인자가 고정된 `ToSql`/`PrepareFields` 약 13개만 도달 불가이고 나머지는 sqlmock 과 cachehandler mock, `Data` 필드에 직렬화 불가 값(chan)을 넣어 `PrepareFields` 를 실패시키는 방식, `NotEq{Value: <Value() 가 오류인 값>}` 로 `ToSql` 을 실패시키는 방식으로 도달합니다. listenhandler 36 중 `json.Marshal` 실패 34개는 `Conference.Data`(map) 에 chan 을 넣으면 conferences 쪽 약 24개가 도달하고, conferencecalls 6개와 count 2개 정도만 도달 불가입니다. 도달 가능분만으로는 약 75/89 라 부족하므로 cmd/conference-control 의 일부가 필요합니다.
- cmd/conference-control (510 statement): 12개 `cmd*` 생성자(flag 등록, 약 69), `initCommand`, `resolveUUID`, `printJSON`(chan 입력으로 오류 경로 도달), `runCreate` 의 입력 검증(핸들러 초기화 이전), 각 `run*` 의 `initHandler` 실패 전파는 제품 코드 변경 없이 테스트할 수 있습니다. 리뷰어 실측으로 실패 전파만으로 이 패키지가 약 56% 가 됩니다. `initHandler` 가 성공해야 도달하는 본문(DB, Redis, RabbitMQ 연결)은 단위 테스트 대상이 아닙니다.
- 함정: `config.Get()` 은 `LoadGlobalConfig()` 이전에 호출하면 panic 이고 `cfg` 는 비공개이며 `sync.Once` 라 프로세스당 한 번만 값이 고정됩니다. 따라서 `TestMain` 에서 `viper.Set("database_dsn", <형식이 잘못된 DSN>)` 후 `config.LoadGlobalConfig()` 를 한 번 호출합니다. viper 는 전역이므로 각 테스트가 설정한 키를 정리합니다. `runGet` 등은 `initHandler` 가 `resolveUUID` 보다 먼저라 입력 검증이 `initHandler` 실패에 가려지고, 입력 검증을 칠 수 있는 것은 `runCreate` 뿐입니다.
- 네트워크 금지: 형식이 잘못된 DSN(슬래시 없음 등)은 mysql 드라이버가 `sql.Open` 에서 파싱 오류로 끝내 접속을 시도하지 않습니다. 빈 DSN 과 기본 DSN 은 로컬 포트로 Ping 하므로 사용하지 않습니다.
- cmd/conference-manager (162): `runMain` 이 외부 연결을 직접 맺는 구조이며 이번 범위에서 제외합니다. 산술 부족분이 생길 때만 `run`/`runSubscribe`/`runListen` 을 재검토합니다.
- 중복 여부: 겹치는 열린 PR 없음(VOIP-1647, 1648, 1651 모두 머지됨). VOIP-1416 이 다루는 패키지(dbhandler 등)와 겹침은 PR 본문에 명시만 합니다.
- 틀린 주장: 없음. 카드의 수치는 모두 재현됩니다(작업 worktree 가 오래된 base 여서 처음 69.4% 로 측정됐으나 origin/main 으로 갱신 후 일치).

## 2. 설계

범위는 테스트 파일 추가와 기존 `_test.go` 에 케이스 추가뿐입니다(제품 코드, go.mod, go.sum 무변경). 단언은 `reflect.DeepEqual` 과 `t.Errorf`, 신규 함수는 `Test_` 접두, gomock 컨트롤러 변수는 `mc`. 실제 네트워크 접속은 만들지 않습니다. 목표는 수기 합계 76% 이상(2075, 필요 증분 116)이고 최소 75.0% 입니다.

### 2.1 cmd/conference-control (신규 `main_test.go`, 필수, 이것만으로 목표 충족: 리뷰어 프로토타입 기준 패키지 약 64% 이며 합계 약 84%)

- `TestMain`: `viper.Set("database_dsn", "not-a-valid-dsn")` 후 `config.LoadGlobalConfig()` 한 번. 이후 `os.Exit(m.Run())`.
- 각 테스트는 쓰는 viper 키를 `old := viper.Get(k)` 로 저장하고 `t.Cleanup` 에서 `viper.Set(k, old)` 로 복원합니다(viper 에 `Unset` 이 없고, 이전 값이 nil 이면 키가 해제된 것으로 동작하며 빈 문자열로 복원하면 `IsSet` 이 참이 되므로 사용하지 않습니다). `viper.Reset` 은 DSN 도 지우므로 사용하지 않습니다.
- `PersistentPreRunE` 는 직접 호출로만 검증하고 `root.Execute()` 는 쓰지 않습니다. `Execute` 를 쓰면 `BindPFlags` 가 sub 명령의 flag(`id` 등)를 viper 에 묶어 `viper.Set(k, nil)` 로도 해제되지 않고, 복원이 그 값을 override 로 고정해 이후 테스트가 오염되기 때문입니다. 직접 호출은 루트 flag 만 묶으므로 `id`, `type` 같은 키에는 영향이 없고 `-shuffle`, `-count>1` 에서도 안전합니다. 이 호출의 성공(오류 nil)만 단언하며, `TestMain` 이 이미 `LoadGlobalConfig` 를 호출했으므로 DSN 유지 여부는 이 호출의 회귀를 잡지 못하는 약한 단언이라 쓰지 않습니다.
- `initCommand` 가 `viper.AutomaticEnv()` 를 켜므로 `id`, `type`, `name`, `lang` 같은 키는 환경변수 `ID`, `TYPE`, `NAME`, `LANG` 의 영향을 받습니다. 각 케이스는 읽히는 모든 키(`customer-id`, `id`, `type`, `pre-flow-id`, `post-flow-id`)를 `viper.Set` 으로 명시하고(미사용은 빈 문자열) 환경변수에 의존하지 않습니다.
- `Test_resolveUUID`: 빈 값(`... is required`), 잘못된 UUID(`invalid format`), 정상 UUID, 오류 메시지에 라벨이 들어가는지.
- `Test_printJSON`: 정상 값은 nil, 직렬화 불가 값(chan)은 `failed to marshal JSON` 오류.
- `Test_cmd_constructors`: 12개 `cmd*` 의 `Use`, `RunE` 비-nil, 필수 flag 이름과 기본값(`type=conference`, `format=wav`, `lang=en`, `limit=100`)을 표로 단언.
- `Test_initCommand`: 하위 명령 `conference` 아래 12개와 `migrate` 존재, 루트 persistent flag(`database_dsn` 등) 존재, `PersistentPreRunE` 실행 성공.
- `Test_runCreate_validation`: customer-id 누락, id 형식 오류, type 오류, pre/post flow 형식 오류 각각의 오류 메시지. 모두 `initHandler` 이전에 끝나므로 DSN 과 무관. type 오류는 `invalid conference type` 문자열을 단언합니다(검증이 없으면 `failed to initialize handlers` 로 바뀌므로 문자열 단언이 변이를 잡습니다).
- `Test_run_initHandlerFailure`: 11개 `run*` 이 잘못된 DSN 때문에 `failed to initialize handlers` 로 끝남(표 구동). `runCreate` 는 검증을 통과하는 입력으로 같은 실패.

### 2.2 pkg/dbhandler (선택 추가, 기존 `error_paths_test.go` 에 추가, 목표 +25)

도달 가능한 분기만 다룹니다. conference 에만 해당: `Data` 필드의 맵 안에 chan 을 넣어 `PrepareFields` 실패(Create 는 `could not prepare fields`, Update 는 `prepare fields failed` 문자열 단언. `FieldData` 값 자체를 chan 으로 주면 다른 위치에서 실패하므로 맵 안에 넣음). conferencecall 은 Create 에 JSON 필드가 없어 이 분기가 도달 불가이고, Update 는 `FieldStatus: []any{make(chan int)}` 처럼 맵 값 안에 chan 을 넣으면 `prepare fields failed` 에 도달합니다(Delete 는 인자 고정이라 불가). conference 와 conferencecall 공통: `NotEq{Value: <Value() 가 오류를 반환하는 값>}` 로 List 의 `could not build query` 실패, sqlmock 으로 캐시 적중 반환, `SetToCache` 실패, `ScanRow` 실패, NULL 컬럼의 빈 slice/map 초기화, `rows.Err()`. 오류 단언은 `err != nil` 이 아니라 오류 문자열 포함 여부로 합니다. 인자가 고정된 `ToSql` 실패(약 13)는 도달 불가라 제외합니다.

### 2.3 pkg/listenhandler (선택 추가, 기존 `error_paths_test.go` 에 추가, 목표 +20)

`Conference.Data` 에 chan 을 가진 응답을 gomock 핸들러가 반환하게 해 `json.Marshal` 실패에 도달시킵니다. Marshal 실패는 모두 `simpleResponse` 이고 응답 코드는 라우트마다 500 과 400 이 섞여 있으므로 구현 시 라우트별 코드를 `v1_conferences.go`/`v1_conferences_direct_hash.go` 에서 읽어 표로 만들고 그 값을 단언합니다. 확인된 도달 가능 라우트만 다룹니다(conferences 의 Marshal 실패 지점 11개: list, post, id get, id put, id delete, recording_id put, recording_start/stop, transcribe_start/stop, stop 와 direct-hash 1개). conferencecalls 와 count 의 Marshal 실패는 도달 불가라 제외합니다.

### 2.4 그 밖

`conferencehandler`, `conferencecallhandler`, `config`, `models/*` 의 개별 미커버 분기 중 한 줄 오류 전파는 같은 방식으로 필요할 때만 추가합니다.

### 2.5 verify 항목과 변이 검사

- 측정은 `GOFLAGS=-mod=mod go test -count=1 -coverprofile=<f> ./...` 후 `hw.py <f>` (카드의 측정 방법과 같은 기준)로 하며 76% 이상(최소 75.0%), 전 패키지 `go test -count=1 -race ./...` 통과, `golangci-lint run -v --timeout 5m` 통과, `scripts/check-test-conventions.sh` 통과, go.mod/go.sum 무변경, `git diff --stat origin/main...HEAD` 가 bin-conference-manager/ 안에만 있음.
- 변이 표본(컴파일되는 변이만 인정하며 블록 삭제로 `err` 미사용 컴파일 실패가 나는 경우는 `if false {`, `&& false` 형태로 바꿉니다): `resolveUUID` 의 `val == ""` 검사와 `res == uuid.Nil` 검사 각각 무력화, `runCreate` 의 type 검증 무력화, `printJSON` 오류 분기 무력화, type flag 기본값 변경. `PersistentPreRunE` 단언은 약해 변이 표본에 넣지 않습니다. 선택 추가는 다음을 포함합니다: (선택 추가를 구현한 경우) dbhandler `PrepareFields` 오류 반환 제거, listenhandler Marshal 오류 분기 제거를 각각 scratch 복사본에서 적용했을 때 새 테스트가 실패해야 합니다. 오류 문자열 단언이 있어야 변이가 생존하지 않습니다.
- 테스트 값은 `docs/conventions/testing.md` 의 규칙대로 하드코딩된 UUID 문자열을 쓰고 `uuid.Nil`/`uuid.NewV4()` 로 만든 값을 기대값으로 쓰지 않습니다. `scripts/check-test-conventions.sh` 는 `git fetch origin main` 후 실행합니다.
- 금지: 빈 호출 테스트, 자기비교식 테스트, 비밀값 리터럴과 `user:pass@host` 형태 DSN(잘못된 DSN 은 `not-a-valid-dsn` 만 사용).
- 결함 발견 시 고치지 않고 카드 코멘트와 완료 요약에만 적습니다.
