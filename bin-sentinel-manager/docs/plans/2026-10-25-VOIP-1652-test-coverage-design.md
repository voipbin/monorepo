# VOIP-1652 bin-sentinel-manager 단위 테스트 커버리지 75% 이상

이 문서의 모든 수치는 수기 코드 기준입니다(gens/, mock_*.go, DO NOT EDIT 헤더 파일 제외, 같은 블록은 최대 count 로 합침). 통과 기준은 수기 합계 75.0% 이상(794/1058)이며, 권장 목표는 76% 이상(805/1058)입니다.

## 1. 티켓 검토 (이슈 분석)

판정: 유효. origin/main accc03e 에서 직접 측정한 수기 합계는 782/1058 = 73.91%, 원본 합계는 807/1120 = 72.05% 이며 티켓 기준선과 일치합니다. 이 경로의 최근 커밋은 린트, 의존성 갱신, 문서 정리이고 커버리지 작업 PR 은 열려 있지 않습니다.

패키지별 기준선(전): cmd/sentinel-control 0.0% (0/124), cmd/sentinel-manager 22.4% (35/156), internal/config 96.5%, pkg/cachehandler 88.2%, pkg/dockerwatchhandler 98.3%, pkg/k8swatchhandler 92.4%, models 100%, pkg/monitoringbackend 100%.

미커버의 대부분은 두 main 패키지입니다(sentinel-control 124개 전부, sentinel-manager 156개 중 121개). 나머지 패키지의 미커버는 합쳐서 31 statement 이고, 805(필요 +23)에 이르려면 그중 23개(74%)를 새로 커버해야 하므로 단독으로는 비현실적입니다. sentinel-control 은 Docker API 를 호출하는 얇은 CLI 이므로 loopback 가짜 Docker 서버로 결정적으로 테스트할 수 있습니다. runService 는 RabbitMQ 연결이 필요해 범위에서 제외합니다.

## 2. 설계

제품 코드는 바꾸지 않고 테스트 파일만 추가합니다. 외부 네트워크와 비용 호출은 없고 가짜 서버는 `httptest` 로 loopback 에만 바인딩합니다.

- cmd/sentinel-control/main_test.go: `initCommand()` 를 실제 인자로 실행합니다. stdout 은 별도 고루틴이 읽는 `os.Pipe` 로 가로채고 반드시 복원합니다.
- cmd/sentinel-manager/main_test.go (기존 파일에 추가): `signalHandler`, `registerSignal`, `initProm` 을 검증합니다.

가짜 Docker 서버: 클라이언트는 `HEAD /_ping` 후 `GET /v1.43/containers/json?all=1` 을 보냅니다. `/_ping` 은 HEAD 와 GET 에 200 과 `Api-Version: 1.43` 을 응답하고 나머지는 `/v1.43/` 접두로 매칭합니다. `all` 은 `r.URL.Query()` 로 단언하고, 404 는 `Content-Type: application/json` 헤더와 JSON `{"message":...}` 를 줍니다(클라이언트는 이 헤더가 정확히 일치할 때만 오류 본문을 파싱함). 요청 로그는 mutex 로 보호합니다. 주소는 `http://` 를 `tcp://` 로 바꿉니다. list fixture 의 `State` 는 JSON 문자열입니다.

전역 상태 대응:
- 모든 sentinel-control 테스트는 `viper.Reset()` 후 `initCommand()` 를 새로 만들고 `t.Setenv` 로 `NAME`, `ALL`, `SENTINEL_BACKEND`, `DOCKER_SOCKET_PROXY_ADDRESS`, `REDIS_ADDRESS`, `REDIS_PASSWORD`, `REDIS_DATABASE` 를 명시합니다(빈 env 는 미설정 취급이므로 `NAME`/`ALL` 간섭이 차단됨). `t.Parallel()` 은 쓰지 않습니다.
- 매번 `cmd.SetArgs(...)` 를 지정합니다. 오류 경로는 `SilenceErrors`, `SilenceUsage`, `SetErr(io.Discard)` 를 지정합니다.
- `config.cfg` 는 검증 실패 시 갱신되지 않으므로 실패 케이스는 반환된 오류 메시지를 단언합니다. 두 cmd 패키지는 서로 다른 테스트 바이너리이고 기존 cmd/sentinel-manager 테스트는 `config.Get()` 에 의존하지 않으므로 `cfg` 간섭이 없습니다.
- `SENTINEL_BACKEND=kubernetes` 로 CLI 를 실행하지 않습니다(기본 프록시 주소에 대한 DNS 조회 발생). 허용은 docker 백엔드와 httptest 주소, 또는 검증에 실패하는 backend 값입니다.
- chSigs: 테스트 시작 시 버퍼를 비우고 실제 시그널 대신 값을 직접 보냅니다. 각 테스트는 자신이 띄운 고루틴이 신호를 소비하고 종료한 것을 확인한 뒤 끝내고 `t.Cleanup` 에서 `signal.Stop(chSigs)` 합니다.
- initProm: `http.DefaultServeMux` 등록은 프로세스당 1회이므로 initProm 호출은 테스트 전체에서 단 한 번이며 `sync.Once` 로 보호합니다. 경로는 고유값(`/metrics-voip1652`)입니다. 단일 흐름: 테스트가 `127.0.0.1:0` 에 `net.Listen` 으로 포트를 얻어 보유한 채 그 주소로 initProm 호출, logrus test hook 이 `Could not start prometheus listener` 를 관측하면(재시도 분기 확인) listener 를 닫고, 폴링으로 그 경로가 200 이고 본문에 `go_goroutines` 가 있을 때만 통과시킵니다. 해제는 hook 관측 뒤에만 일어나므로 첫 시도보다 앞설 수 없고 분기가 항상 실행됩니다. 재시도는 1초 sleep 뒤이므로 폴링 timeout 은 5초 이상(구현은 10초)입니다. 해제 후 다른 프로세스가 포트를 잡을 가능성은 고유 경로와 본문 단언이 막습니다. 고루틴 누수는 허용합니다.

verify 항목:
1. list: 이름 필터 일치, 불일치, 필터 없음(전체 출력), `--all` 이 쿼리 `all=1` 로 전달됨, 이름 없는 컨테이너, `/` 접두 제거, 네트워크 없음, nil endpoint, 필드별(id, name, image, state, status, addresses) 값 단언, Docker 오류의 래핑 오류.
2. get: `--name` 누락은 오류 메시지가 정확히 `container name is required` 이고 요청 로그가 비어 있음(이 검사가 없어도 클라이언트가 빈 이름을 거부하므로 메시지 단언이 필요하며 구현 때 실측), 정상 inspect 의 id, name(슬래시 제거), image, state, started_at, 주소, MAC 매핑, 404 래핑 오류, nil 분기(`{}` 는 ContainerJSONBase nil, Id 만 있는 JSON 은 State nil, Config nil, NetworkSettings nil, `"x":null` 은 nil endpoint).
3. sentinel-control 설정 검증: `SENTINEL_BACKEND` 미설정이면 PersistentPreRunE 가 오류를 반환하고 Docker 요청이 없음. `--docker_socket_proxy_address=` 플래그로 빈 값을 주면 Validate 가 거부함(env 로는 기본값이 적용되어 도달 불가).
4. newDockerClient 오류: CLI 를 통해 `DOCKER_SOCKET_PROXY_ADDRESS=no-scheme` 또는 `tcp://` 로 유발(둘 다 Validate 통과, `WithHost` 에서만 실패. 알 수 없는 스킴은 오류가 아니므로 쓰지 않음). printJSON marshal 실패는 `printJSON(make(chan int))` 을 직접 호출해 유발.
5. sentinel-manager `run`: `SENTINEL_BACKEND` 를 `t.Setenv` 로 비우고 `run(rootCmd, nil)` 을 호출하면 InitConfig 실패 오류(`failed to initialize config`)를 반환하고 initProm 과 runService 에 도달하지 않음. 잘못된 backend 값도 동일.
6. signalHandler: 신호 주입 전에는 짧은 대기 후에도 ctx 가 살아 있고, 주입 후 `<-ctx.Done()` 을 timeout 과 함께 기다림. registerSignal 은 고루틴을 띄워 주입된 신호로 cancel 됨. initProm 은 위 방식.

도달 불가(미커버로 남음): sentinel-control 의 `main`(3), Bootstrap 실패 `cobra.CheckErr`(1), BindPFlags 실패 분기(1), closeDockerClient 의 Close 오류 로그(1). sentinel-manager 의 `main`(3), `run` 의 InitConfig 이후 구간, `runService`, initProm 의 `break`(ListenAndServe 는 항상 오류를 반환함). 예상 합계는 약 900~910/1058(sentinel-control 약 118개 + signal 과 initProm 수 개)이며 구현 후 측정값으로 보고합니다.

변이 검사(scratch 복사본에서만 적용, 마지막에 worktree `git diff --stat` 이 테스트 파일만 가리키는지 확인): list 의 `!strings.Contains` 부정 제거, `--all` 전달 제거, get 의 name 빈값 검사 제거, TrimPrefix 제거(list 와 get 양쪽 fixture 에 `/` 접두 이름), MAC 맵 대입 제거, list 와 get 의 nil endpoint skip 제거(nil 역참조 panic 으로 실패), list 의 `NetworkSettings != nil` 검사 제거, signalHandler 의 cancel 제거. 변이가 테스트 실패로 이어짐을 확인하며, nil 역참조 panic 으로 테스트 바이너리가 중단되는 경우도 검출로 인정합니다. `nameFilter != ""` 조건 제거는 등가 변이이므로 쓰지 않습니다.
