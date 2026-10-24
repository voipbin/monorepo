# bin-storage-manager 단위 테스트 커버리지 70% 상향 (VOIP-1651)

## 1. 티켓 검토와 이슈 분석

### 1.1 티켓 검토 (결과: 유효)

- 기준선 재측정(origin/main 99b59c386, `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...`, 테스트 실패 0건): 수기 합계 68.37% (1738/2542 statement), 원본 60.0%. 카드 수치와 일치한다.
- 목표 70% 에는 42개, 권장 71% 에는 67개 statement 를 더 커버해야 한다.
- 이전 작업(VOIP-1647 이 60% 달성, VOIP-1648 이 오류 처리 보강)은 모두 머지되어 기준선에 포함돼 있다. 열린 PR 중 같은 서비스를 건드리는 것은 없다.
- 수기 합계는 gens/, mock_*.go, `Code generated ... DO NOT EDIT` 파일을 제외하고 같은 블록의 중복은 최대 count 로 합쳐 계산한다. 서비스에는 gens/ 가 없고 제외 대상은 mock_main.go 뿐이다.

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

### 1.3 미커버 분석

미커버 804개 중 670개가 cmd/ 에 있고 나머지 134개가 pkg/internal/models 에 있다. 134개를 전부 덮으면 (1738+134)/2542 = 73.6% 이므로 산술상 cmd 없이 pkg 만으로도 70%(42개, 31%)와 71%(67개, 50%)가 가능하다. 다만 아래 도달 불가 분류(약 40개: 마샬 실패 분기, 로그만 있는 분기, 빈 `Init`)를 빼면 도달 가능한 pkg 구간은 약 90개여서 71% 에 필요한 67개는 그 70% 를 넘는 비율이고 여유가 작다. 따라서 cmd 의 연결 없이 도달 가능한 구간은 필수가 아닌 안전 여유분으로 쓴다.

cmd/storage-control (490):
- `run*` 함수는 모두 `initStorageHandler`/`initAccountHandler` 를 직접 호출하며 DB, Redis, RabbitMQ 에 연결한다. 주입 지점이 없어 성공 경로는 단위 테스트가 불가능하다(제품 코드 변경 금지).
- 연결 없이 도달 가능한 구간: `resolveUUID`, `printJSON`, `cmd*` 생성자 10개(file 4, account 4, recording 2. 그룹 명령 3개는 `initCommand` 안의 리터럴), `initCommand`(루트와 하위 명령 등록), `runFileCreate`/`runAccountCreate` 의 입력 검증(연결 이전에 return), 나머지 `run*`(`runFileGet`, `runFileList`, `runFileDelete`, `runAccountGet`, `runAccountList`, `runAccountDelete`, `runRecordingGet`, `runRecordingDelete`) 의 `init*Handler` 호출까지와 그 오류 return(DB 연결이 먼저인 함수는 입력 검증 이전에 실패하므로 UUID 검증 오류는 `init` 가 성공해야 닿고 도달 불가). `config.Get()` 은 전역 구조체 포인터를 돌려주므로 테스트에서 DSN 을 형식 오류 값으로 두면 `sql.Open` 의 DSN 파싱 단계에서 네트워크 없이 실패한다. 즉 DSN 파싱 실패 return 까지를 닫는다.
- 도달 불가: DSN 파싱 이후의 `initCache`/`initHandler`, 호출과 출력 구간(성공 경로).

cmd/storage-manager (180):
- `run`, `createDBHandler`, `runService` 는 연결이 필요하다. `createDBHandler` 의 DSN 파싱 실패 경로는 위와 같은 방식으로 도달 가능하다.
- `runListen`, `runSubscribe` 는 sockhandler 인터페이스를 받으므로 기존 mock 으로 성공과 실패 경로를 모두 닫을 수 있다.
- `initProm` 은 전역 `http.Handle` 과 포트 바인딩을 쓰므로 제외한다.

pkg 와 internal (134):
- 도달 가능: `decodeWavPeaks`/`readWavHeader` 의 경계(다채널, 버킷 수가 프레임 수보다 큼, 잘린 스트림, 홀수 크기의 미지 청크, 잘린 청크 헤더), `PeaksComputeByFile`(기존 GCS fake 확장), `NewFileHandler` 의 인증 파일 분기, `file.go` 의 `Get` 오류 분기(NotFound 변환), 목록 조회 오류, `compress.go`, `v1_recordings` 의 퍼센트 디코딩 오류, `internal/config` 의 `LoadGlobalConfig`/바인딩 오류, models 의 변환 오류, dbhandler 의 오류 분기(닫힌 DB, 스캔 실패 등).
- 도달 불가 또는 가치 낮음: json.Marshal 실패 분기(입력으로 만들 수 없음), 로그만 있는 분기, `fileHandler.Init`(본문이 비어 있어 1 statement 이며 의미 있는 검증이 없으므로 제외).

### 1.4 위험과 제약

- 전역 상태: viper, `config.Get()`, `http.DefaultServeMux`, `STORAGE_EMULATOR_HOST` 환경변수. 이를 바꾸는 테스트는 `t.Parallel` 을 쓰지 않고 `t.Cleanup` 으로 복원한다.
- `config` 의 `sync.Once` 는 패키지 밖에서 초기화할 수 없다. cmd 테스트는 `LoadGlobalConfig` 에 의존하지 않고 `config.Get()` 포인터를 직접 설정한다.
- 전역 상태 추가: `run*` 는 `viper` 전역 값을 읽으므로 `viper.Reset()` 으로 복원한다. `printJSON` 은 stdout 에 쓰므로 표준출력을 파이프로 바꿔 검사하고 복원한다. `initCommand` 는 바인딩 오류 시 `cobra.CheckErr` 로 프로세스를 종료하므로 정상 경로만 호출한다. cmd/storage-manager 의 `init()` 은 `signal.Notify` 와 `go signalHandler()` 를 이미 실행하므로 테스트에서 시그널을 보내지 않는다.
- 결함 발견 시 고치지 않고 카드 코멘트에만 기록한다.
- VOIP-1416(80% 미만 패키지 일반 상향)과 겹치는 패키지는 없다(현재 모든 pkg 패키지가 80% 이상이고 cmd 만 0%). 겹치는 경우 PR 본문에 명시만 한다.
