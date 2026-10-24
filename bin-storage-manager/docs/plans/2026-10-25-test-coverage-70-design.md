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

미커버 804개 중 670개가 cmd/ 에 있고 134개가 pkg/internal/models 에 있다. 134개를 전부 덮으면 (1738+134)/2542 = 73.6% 이므로 산술상 pkg 만으로도 가능하지만, 아래 분류에서 pkg 의 도달 불가가 약 50개이고 도달 가능 약 85개 중 일부(GCS 본문 읽기, zip 쓰기)는 fake 확장이 필요해 여유가 작다. 반면 cmd 에서 외부 연결 없이 닿는 구간은 약 380개로 필요한 67개의 몇 배이며 fake 도 필요 없다. 따라서 우선순위는 cmd 의 도달 가능 구간(주 경로), pkg 의 오류 경계와 입력 검증(품질 경로) 순이다.

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

DSN 주의: `commondatabasehandler.Connect` 는 `sql.Open` 다음 `PingContext` 이다. go-sql-driver 는 `sql.Open` 에서 `ParseDSN` 으로 DSN 을 파싱하므로 `ParseDSN` 이 거부하는 값(슬래시가 없는 문자열, 빈 문자열)만 네트워크 없이 실패한다. 형식은 맞지만 닿지 않는 호스트는 Ping 에서 네트워크를 타므로 쓰지 않는다. 설정은 `config.Get()` 포인터에 직접 대입하거나 `--database_dsn` 플래그로 준다.

cmd/storage-manager (미커버 180):

| 구간 | statement | 도달 |
|---|---|---|
| `init()` (플래그 정의, `signal.Notify`, 고루틴 시작) | 42 | 테스트 패키지가 생기면 자동 실행되어 따라옴 |
| `runListen` 6, `runSubscribe` 6 | 12 | 가능(sockhandler mock) |
| `createDBHandler` 의 DSN 파싱 실패 return | 약 5 | 가능 |
| `signalHandler` 3 | 3 | 가능하나 `init()` 의 고루틴과 경합하므로 시그널을 보내지 않고 제외 |
| `run` | 27 | 앞부분(Bootstrap 오류 return)만 가능. `initProm`(전역 `http.Handle` 과 포트 바인딩)을 먼저 부르므로 `run` 호출은 제외 |
| `initProm` 13, `runService` 61, `createDBHandler` 연결 이후, `main` 3 | 약 90 | 불가(연결, 포트 바인딩) |

pkg/internal/models (미커버 134) 함수 단위 분류:

| 분류 | statement | 비고 |
|---|---|---|
| 도달 불가 | 약 50 | listenhandler 의 마샬 실패 분기 23, dbhandler 의 `PrepareFields`/`ToSql` 실패 18, rows affected 오류 2, `bindConfig`/`Bootstrap` 바인딩 오류 3, 고루틴 안 `ConsumeMessage`/`ConsumeRPC` 오류 로그 2, 빈 `Init` 1 등 |
| 도달 가능, 단순 | 약 40 | `decodeWavPeaks`/`readWavHeader` 경계 15(다채널, 버킷 수가 프레임 수보다 큼, 잘린 스트림, 홀수 크기 미지 청크, 잘린 청크 헤더, 빈 결과), `LoadGlobalConfig` 4, models 변환 오류 4, `v1_recordings` 의 퍼센트 디코딩 오류 3, `compress.go` 2, `file.go` `Get` 3, `storagehandler` 1, `NewFileHandler` 인증 파일 분기 5, dbhandler 캐시 쓰기 실패 4 |
| 도달 가능, fake 확장 필요 | 약 40 | `PeaksComputeByFile` 9(GCS 객체 본문 읽기), `bucketfile` 의 이동, 압축, 서명 URL 오류 10, `file.go` `Create` 10 |

산술 근거: 도달 가능한 cmd 약 380 + pkg 단순 약 40 = 약 420 로 필요한 67 의 6배 이상이고 fake 확장 구간은 필요하면 추가한다.

### 1.4 위험과 제약

- 전역 상태: viper, `config.Get()`, `http.DefaultServeMux`, `STORAGE_EMULATOR_HOST` 환경변수. 이를 바꾸는 테스트는 `t.Parallel` 을 쓰지 않고 `t.Cleanup` 으로 복원한다.
- `config` 의 `sync.Once` 는 패키지 밖에서 초기화할 수 없다. cmd 테스트는 `LoadGlobalConfig` 에 의존하지 않고 `config.Get()` 포인터를 직접 설정한다.
- cmd/storage-manager 의 `init()` 은 테스트 패키지가 생기면 자동 실행되어 42 statement 가 따라온다.
- 전역 상태 추가: `run*` 는 `viper` 전역 값을 읽으므로 `viper.Reset()` 으로 복원한다. `printJSON` 은 stdout 에 쓰므로 표준출력을 파이프로 바꿔 검사하고 복원한다. `initCommand` 는 바인딩 오류 시 `cobra.CheckErr` 로 프로세스를 종료하므로 정상 경로만 호출한다. cmd/storage-manager 의 `init()` 은 `signal.Notify` 와 `go signalHandler()` 를 이미 실행하므로 테스트에서 시그널을 보내지 않는다.
- 결함 발견 시 고치지 않고 카드 코멘트에만 기록한다.
- VOIP-1416(80% 미만 패키지 일반 상향)과 겹치는 패키지는 없다(현재 모든 pkg 패키지가 80% 이상이고 cmd 만 0%). 겹치는 경우 PR 본문에 명시만 한다.
