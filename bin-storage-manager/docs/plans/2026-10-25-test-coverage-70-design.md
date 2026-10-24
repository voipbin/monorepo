# bin-storage-manager 단위 테스트 커버리지 70% 상향 (VOIP-1651)

## 1. 티켓 검토와 이슈 분석

### 1.1 티켓 검토 (결과: 유효)

- 기준선 재측정(origin/main 99b59c386, `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...`, 테스트 실패 0건): 수기 합계 68.37% (1738/2542 statement), 원본 60.0%. 카드 수치와 일치한다.
- 목표 70% 에는 42개, 권장 71% 에는 67개 statement 를 더 커버해야 한다.
- 이전 작업(VOIP-1647 이 60% 달성, VOIP-1648 이 오류 처리 보강)은 모두 머지되어 기준선에 포함돼 있다. `gh pr list --state all --search 'storage-manager in:title'` 와 `VOIP-1651` 검색(2026-10-25 시점) 결과 열린 PR 은 없다.
- 수기 합계는 gens/, mock_*.go, `Code generated ... DO NOT EDIT` 파일을 제외하고 같은 블록의 중복은 최대 count 로 합쳐 계산한다. 서비스에는 gens/ 가 없고 제외 대상은 6개 패키지(cachehandler, filehandler, accounthandler, dbhandler, subscribehandler, storagehandler)의 mock_main.go 이며 `Code generated` 헤더를 가진 다른 파일은 없다.

### 1.2 기준선 (수기 코드)

| 패키지 | 커버/전체 | 비율 |
|---|---|---|
| cmd/storage-control | 0/490 | 0.0% |
| cmd/storage-manager | 0/180 | 0.0% |
| internal/config | 66/73 | 90.4% |
| models/account | 15/17 | 88.2% |
| models/file | 14/16 | 87.5% |
| pkg/accounthandler | 158/158 | 100.0% |
| pkg/cachehandler | 87/87 | 100.0% |
| pkg/dbhandler | 316/346 | 91.3% |
| pkg/filehandler | 465/522 | 89.1% |
| pkg/listenhandler | 368/401 | 91.8% |
| pkg/storagehandler | 179/180 | 99.4% |
| pkg/subscribehandler | 70/72 | 97.2% |
| 합계 | 1738/2542 | 68.37% |

### 1.3 미커버 분석 (함수 단위 statement 수, 기준선 프로파일에서 직접 집계)

미커버 804개 중 670개가 cmd/ 에 있고 134개가 pkg/internal/models 에 있다. 134개를 전부 덮으면 (1738+134)/2542 = 73.6% 이므로 산술상 pkg 만으로도 가능하지만, 아래 분류에서 pkg 의 도달 불가가 46개이고 도달 가능 88개 중 29개(GCS 본문 읽기, zip 쓰기)는 fake 확장이 필요해 단순 구간만으로는 59개로 71% 에 필요한 67개에 못 미친다. 반면 cmd 에서 외부 연결 없이 닿는 구간은 약 380개로 필요한 67개의 몇 배이며 fake 도 필요 없다. 따라서 우선순위는 cmd 의 도달 가능 구간(주 경로), pkg 의 오류 경계와 입력 검증(품질 경로) 순이다.

cmd/storage-control (미커버 490):

| 구간 | statement | 도달 |
|---|---|---|
| `cmd*` 생성자 10개 (File 4: 36/12/12/24, Account 4: 12/12/12/18, Recording 2: 12/12) | 162 | 가능(플래그 정의) |
| `initCommand` (루트, 그룹 3개, `migrate` 등록) | 94 | 가능. `migrate` 클로저는 실행해야 닿으며 `migrationhandler.New` 가 DSN 파싱을 가장 먼저 하므로 형식 오류 DSN 이면 연결 없이 오류 반환 |
| `resolveUUID` 7, `printJSON` 5 | 12 | 가능. `printJSON` 의 마샬 오류는 json 불가 값(채널)으로 도달 |
| `runFileCreate` 의 입력 검증(UUID 3회, reference-type, name, filename, bucket-name, filepath) | 약 18 | 가능(연결 이전에 return) |
| `runAccountCreate` 의 UUID 검증 | 약 5 | 가능 |
| 나머지 `run*` 8개(FileGet, FileList, FileDelete, AccountGet, AccountList, AccountDelete, RecordingGet, RecordingDelete)의 `init*Handler` 호출과 그 오류 return | 약 25 | 가능(UUID 검증은 init 뒤라 도달 불가) |
| `initStorageHandler`/`initAccountHandler` 의 연결 실패 return | 약 6 | 가능 |
| `initCache`, `initHandler`, 위 함수들의 연결 이후 구간, 각 `run*` 의 성공 경로, `main` | 약 168 | 불가(Redis, RabbitMQ, DB 연결 필요, 주입 지점 없음) |

DSN 주의: `commondatabasehandler.Connect` 는 `sql.Open` 다음 `PingContext` 이다. go-sql-driver 는 `sql.Open` 에서 `ParseDSN` 으로 DSN 을 파싱하며, 슬래시가 없는 비어 있지 않은 문자열(예: `invalid`)만 네트워크 없이 실패한다. 빈 문자열은 파싱을 통과해 로컬 3306 으로 연결을 시도하므로 쓰지 않고, 형식이 맞는 DSN 도 쓰지 않는다. 설정 주입은 두 경로를 섞지 않는다: `run*` 와 `init*Handler` 직접 호출 테스트는 `config.Get().DatabaseDSN` 에 위 값을 대입하고, 루트 명령을 `Execute` 하는 테스트(`migrate` 클로저)는 `PersistentPreRunE` 의 `LoadGlobalConfig`(sync.Once)가 `globalConfig` 전체를 덮어쓰므로 `--database_dsn` 플래그로 같은 값을 준다. `Execute` 테스트는 첫 실행의 값이 고정되므로 같은 DSN 만 쓴다.

cmd/storage-manager (미커버 180):

| 구간 | statement | 도달 |
|---|---|---|
| `init()` (플래그 정의, `signal.Notify`, 고루틴 시작) | 42 | 테스트 패키지가 생기면 자동 실행되어 따라옴 |
| `runListen` 6, `runSubscribe` 6 | 12 | 가능(sockhandler mock) |
| `createDBHandler` 의 DSN 파싱 실패 return | 약 5 | 가능 |
| `signalHandler` 3 | 3 | 가능하나 `init()` 의 고루틴과 경합하므로 시그널을 보내지 않고 제외 |
| `run` | 27 | 불가. `Bootstrap` 오류 return 은 정상 입력에서 닿지 않고, `initProm`(전역 `http.Handle` 과 포트 바인딩)을 먼저 부르므로 호출 자체를 제외 |
| `initProm` 13, `runService` 61, `createDBHandler` 연결 이후, `main` 3 | 약 90 | 불가(연결, 포트 바인딩) |

pkg/internal/models (미커버 134) 분류. 미커버 블록 목록을 하나씩 분류한 값이며 도달 가능성은 구현 단계에서 실측으로 확인한다.

| 분류 | statement | 내용 |
|---|---|---|
| 도달 불가 | 46 | listenhandler 마샬 실패 분기 23, dbhandler `PrepareFields`/`ToSql` 실패 16, rows affected 오류 2, `bindConfig`/`Bootstrap` 바인딩 오류 3, 고루틴 안 `ConsumeMessage`/`ConsumeRPC` 오류 로그 2 (빈 `Init` 은 0 statement 라 집계 밖) |
| 도달 가능, 단순 | 59 | `ApplyFields` 오류 2(지원하지 않는 필터 타입), dbhandler 행 스캔과 변환 오류 6, 캐시 쓰기 실패 4, listenhandler `url.Parse` 오류 4, `errorResponse` 2, `v1_recordings` 퍼센트 디코딩 오류 3, `logDownloadURIFailure` 1, `processEvent` 오류 로그 1, `compress.go` 2, `file.go` `Get` 3, `storagehandler` 1, models 변환 오류 4, `LoadGlobalConfig` 4, `NewFileHandler` 인증 파일 분기 7, `decodeWavPeaks`/`readWavHeader` 경계 15 |
| 도달 가능, fake 확장 필요 | 29 | `PeaksComputeByFile` 9(GCS 객체 본문 읽기), `bucketfile` 의 이동, 압축, 서명 URL 오류 10, `file.go` `Create` 10 |

세 분류의 합은 46+59+29 = 134 이다. 산술 근거: 도달 가능한 cmd 약 380 + pkg 단순 59 = 약 440 으로 필요한 67 의 6배 이상이다. 위 수치 중 "약 N" 은 소스 줄 수로 환산한 추정이며(프로파일 블록은 소스 줄 수와 일치하지 않는다) 구현 단계에서 항상 프로파일로 실측해 확인한다. fake 확장 구간은 필요할 때만 추가한다.

### 1.4 위험과 제약

- 전역 상태: viper, `config.Get()`, `http.DefaultServeMux`, `STORAGE_EMULATOR_HOST` 환경변수. 이를 바꾸는 테스트는 `t.Parallel` 을 쓰지 않고 `t.Cleanup` 으로 복원한다.
- `config` 의 `sync.Once` 는 패키지 밖에서 초기화할 수 없다. 설정 주입 방식은 1.3 의 DSN 주의를 따른다.
- cmd/storage-manager 의 `init()` 은 테스트 패키지가 생기면 자동 실행되어 42 statement 가 따라온다.
- 전역 상태 추가: `run*` 는 `viper` 전역 값을 읽으므로 `viper.Reset()` 으로 복원한다. `printJSON` 은 stdout 에 쓰므로 표준출력을 파이프로 바꿔 검사하고 복원한다. `initCommand` 는 바인딩 오류 시 `cobra.CheckErr` 로 프로세스를 종료하므로 정상 경로만 호출한다. cmd/storage-manager 의 `init()` 은 `signal.Notify` 와 `go signalHandler()` 를 이미 실행하므로 테스트에서 시그널을 보내지 않는다.
- 결함 발견 시 고치지 않고 카드 코멘트에만 기록한다.
- VOIP-1416(80% 미만 패키지 일반 상향)과 겹침은 PR 본문에 명시만 한다. 수기 기준으로는 pkg 패키지가 모두 80% 이상(최저 87.5%)이고 cmd 만 0% 이다. 원본 Go 출력 기준(mock 포함)으로는 cachehandler 61.3%, accounthandler 70.9%, dbhandler 74.2% 가 80% 미만이므로 어느 기준인지 PR 본문에 같이 적는다.

## 2. 설계

### 2.1 원칙

- 비테스트 `.go` 를 바꾸지 않는다. `go.mod`/`go.sum` 도 바꾸지 않는다(새 의존성 없음, sqlmock 없음).
- 새 테스트는 `Test_<대상>_<시나리오>` 이름, gomock 컨트롤러 변수명 `mc`, 합성값만, 외부 호출과 네트워크 없음(`scripts/check-test-conventions.sh` 통과). DSN 은 `invalid`(슬래시 없는 비어 있지 않은 값)만 쓴다.
- 기대값은 입력에서 계산하지 않고 리터럴로 적는다. 빈 호출과 자기비교는 금지한다.
- 전역 상태: 각 테스트는 시작할 때 자기 값을 직접 대입(`viper.Reset()`, `config.Get().DatabaseDSN`)하고 `t.Cleanup` 으로 이전 값을 복원해 순서 의존을 없앤다. 루트 명령 `Execute` 테스트는 `viper.Reset()` 뒤 `initCommand()` 를 새로 만들고 반드시 `SetArgs(...)` 를 지정하며(안 하면 `-test.*` 플래그를 파싱한다), `migrate` 오류의 stderr 출력은 검사하지 않는다. `Execute` 가 `once` 로 `globalConfig` 를 덮어쓰며 `-count=3` 의 두 번째 반복부터는 `once` 가 소진되어 플래그를 다시 읽지 않으므로(다른 테스트의 복원으로 DSN 이 빈 문자열이면 빈 DSN 은 `ParseDSN` 을 통과해 로컬 3306 연결을 시도한다), `Execute` 직전에 `config.Get().DatabaseDSN = "invalid"` 도 직접 대입하고 플래그 `--database_dsn=invalid` 와 함께 쓴다. `Execute` 테스트는 `globalConfig` 를 복원하지 않는다. `bindConfig` 가 전역 viper 에 `AutomaticEnv` 를 켜므로 `name`, `filename`, `filepath`, `token` 같은 키는 환경변수를 읽을 수 있다. 따라서 필수값을 비우는 케이스는 부재에 기대지 않고 `viper.Set(key, "")` 로 명시한다. 순서(`-shuffle`, `-run` 단독 실행)와 상관없이 통과해야 하며 검증 절에서 확인한다.
- 범위: 필수는 cmd 단계(2.2 의 cmd 항목)이며 단독으로 70% 와 71% 를 넘는다(임시 실측으로 control 약 52~60%, manager 약 29%, 합쳐 +300 statement 이상). pkg 단순 구간은 오류 경계와 입력 검증을 지키기 위한 선택 항목으로, 기대값이 코드로 하나로 정해지는 것만 추가한다. fake 확장 구간은 cmd 단계 실측이 71% 에 못 미칠 때만 한다. 70%, 71% 의 결정은 cmd 단계 완료 후 프로파일 실측으로 한다.

### 2.2 테스트 항목

cmd/storage-control (`main_test.go`, 전역 상태 `viper`, `config.Get()`, stdout 은 `t.Cleanup` 으로 복원, `t.Parallel` 없음):
- `resolveUUID`: 빈 값은 `<label> is required`, 형식 오류 값은 `invalid format for <label>: '<값>' is not a valid UUID`, 올바른 값은 그 UUID 반환.
- `printJSON`: 구조체 출력이 들여쓰기 2칸 JSON 이고 개행으로 끝남, json 불가 값(채널)은 `failed to marshal JSON` 오류.
- `cmd*` 생성자 10개: `Use` 와 `RunE` 비어 있지 않음, 플래그 이름과 기본값을 리터럴 목록으로 단언(예: file list 의 `limit` 기본 100, `reference-type`, account list 의 플래그).
- `initCommand`: 루트 `Use`, 하위 명령 트리(`file`: create/get/list/delete, `account`: create/get/list/delete, `recording`: get/delete, `migrate`), `--database_dsn=invalid` 로 `migrate status` 실행 시 연결 없이 오류 반환.
- `runFileCreate` 입력 검증: 필수값을 하나씩 비운 표(customer, owner, reference id 형식 오류, reference-type, name, filename, bucket-name, filepath)별 오류 문구 단언과 그 시점까지 `init` 이 호출되지 않음(DSN 을 `invalid` 로 두었는데 오류 문구가 검증 문구이면 init 전에 return 한 것).
- `runAccountCreate`: customer-id 비었음과 형식 오류.
- 나머지 `run*` 8개와 `initStorageHandler`/`initAccountHandler`: `config.Get().DatabaseDSN = "invalid"` 일 때 `failed to initialize handlers` 로 래핑된 오류(내부 `could not connect to the database` 포함) 반환.

cmd/storage-manager (`main_test.go`, `init()` 자동 실행):
- `rootCmd` 플래그 이름과 기본값을 리터럴로 단언(`database_dsn`, `prometheus_endpoint`, `redis_database` 기본 1 등)하되 기본값에 계정 문자열이 있는 `database_dsn` 과 `rabbitmq_address` 는 단언하지 않는다.
- `runListen`/`runSubscribe`: `sockhandler` mock 으로 `QueueCreate` 실패는 오류 반환과 이후 호출 없음, 구독은 `TopicCreateWithKind`, `QueueBind` 실패도 각각, 성공은 `ConsumeRPC`/`ConsumeMessage` 고루틴 시작(`DoAndReturn` 의 return 직전에 채널로 신호를 보내고 `nil` 을 반환해 고루틴이 끝나게 하며, 테스트가 그 신호를 기다린 뒤 종료해 종료 후 mock 호출을 막는다). `printJSON` 의 stdout 은 파이프로 바꿔 쓰기 종료 뒤 읽고 출력이 파이프 버퍼보다 작은 값만 쓴다. `createDBHandler` 는 `invalid` DSN 으로 오류 반환.
- 제외: `run`, `runService`, `initProm`, `main`, `signalHandler` 의 시그널 전송(시그널을 보내지 않아도 `init()` 의 고루틴 시작으로 `signalHandler` 진입 statement 는 따라올 수 있다).

pkg 단순 구간:
- `decodeWavPeaks`/`readWavHeader`: 다채널 최대 크기, 버킷 수가 프레임 수보다 큼, 잘린 PCM, 홀수 크기 미지 청크 건너뛰기, 잘린 청크 헤더, 크기 0 데이터. 기대값은 손계산 리터럴.
- `LoadGlobalConfig`(기존 `TestLoadGlobalConfig` 는 `once` 를 소진시켜 본문을 실행하지 않으므로, 같은 패키지에서 `once = sync.Once{}` 후 `viper.Set` 값이 `Get()` 의 필드별로 반영되는지 단언하고 `globalConfig`, `once`, viper 를 복원), models 의 `ConvertStringMapToFieldMap` 과 `CreateWebhookEvent` 오류(json 불가 값).
- listenhandler: `url.Parse` 오류 4, 퍼센트 디코딩 오류 3, `errorResponse` 의 typed nil `*VoipbinError`(500 반환).
- dbhandler: `ApplyFields` 지원하지 않는 필터 타입 오류 2, `AccountUpdate`/`FileUpdate` 의 json 불가 값 오류 2(1.3 은 `PrepareFields` 실패 16 을 도달 불가로 분류했으나 `Update` 계열 2개는 호출자가 넘긴 map 이라 실측으로 도달 가능하므로 이 항목이 우선하고 도달 불가는 14 로 정정), 캐시 쓰기 실패 4. 캐시 쓰기 실패의 계약은 코드 기준으로 확정: 쓰기 함수들이 `_ = h.<entity>UpdateToCache(...)` 로 오류를 버리므로 `AccountSet`/`FileSet` 이 실패해도 쓰기 함수는 nil 을 반환하고 DB 에는 반영되어 있음(재조회로 확인)을 단언한다.
- filehandler: `NewFileHandler` 인증 파일 분기(파싱 불가 파일과 유효 키만. 없는 경로는 기존 `Test_NewFileHandler_credentials` 가 이미 다룬다. `JWTConfigFromJSON` 은 `private_key` 를 파싱하지 않으므로 `private_key` 는 PEM 형태가 아닌 명백한 가짜 문자열, 이메일은 `example.invalid` 류로 쓰고 시크릿 탐지기에 걸릴 형태를 만들지 않는다. 기존 테스트처럼 `STORAGE_EMULATOR_HOST` 를 `t.Setenv` 로 지정하고 `t.Parallel` 을 쓰지 않는다), `Get` 의 NotFound 변환과 일반 오류 래핑, `Create` 의 계정 검증 실패, DB 생성 실패, 재조회 실패, 계정 증가 실패(기존 mock 으로 가능한 것만), `compress.go`, `logDownloadURIFailure`.
- storagehandler `CompressfileCreate` 의 파일 조회 오류, subscribehandler `processEvent` 오류 로그 분기.
- fake 확장 구간(`PeaksComputeByFile` 등)은 위 단계 실측이 71% 에 못 미칠 때만 기존 `fake_gcs_test.go` 를 확장해 추가한다.

### 2.3 검증

- 수기 합계 70.0% 이상, 목표 71% 이상(CI 방식 측정, 집계는 `hw.py` 와 같은 규칙).
- `go test -count=1 ./...` 와 신규 테스트 `-race -count=3` 통과, `golangci-lint run -v --timeout 5m`, `go mod tidy` 후 `go.mod`/`go.sum` 불변, `-shuffle=on` 과 테스트 단독 `-run` 실행에서도 통과(`Execute` 와 `once` 의존 확인), `git diff --stat origin/main...HEAD` 가 bin-storage-manager/ 안에만 있음, `scripts/check-test-conventions.sh` 통과, 공개 저장소 위생 검사(내부 주소, 비밀값 없음).
- 변이 검사(별도 scratch 복사본에서만): `resolveUUID` 의 빈 값 분기 제거, `runFileCreate` 의 검증 순서 변경(init 이전 return 파괴), `printJSON` 들여쓰기 변경, `runListen` 의 오류 무시, 플래그 기본값 변경(limit 100), `errorResponse` 상태 코드 변경, `readWavHeader` 의 홀수 청크 정렬 제거, `Get` 의 NotFound 변환 제거, `LoadGlobalConfig` 의 필드 매핑 하나 변경, `runSubscribe` 의 `QueueBind` 오류 무시, `initCommand` 의 `PersistentPreRunE` 제거 또는 `migrate` 의 DSN 인자 변경, `initStorageHandler`/`initAccountHandler` 의 오류 래핑 문구 변경, `decodeWavPeaks` 의 다채널 처리 변경에서 해당 신규 테스트가 실패하는지 확인한다.
- 롤백: 테스트 추가뿐이므로 PR 되돌림만으로 복구된다.
