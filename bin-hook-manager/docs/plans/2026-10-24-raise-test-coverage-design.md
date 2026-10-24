# VOIP-1647 bin-hook-manager 단위 테스트 커버리지 60% 계획

## 1. 티켓 검토 (유효)

- 기준선(origin/main d68e695b5, `GOFLAGS=-mod=mod go test -count=1 -coverprofile`): 원본(생성 코드와 mock 포함) 46.2%, 수기 합계 43.5% (수기 statement 577개 중 251개 커버). 목표는 수기 합계 61% 이상(인수 조건 60%, 여유 1%p)이며 약 352개 커버가 필요.
- 미커버: `cmd/hook-control/main.go`(0%), `cmd/hook-manager/main.go`(0%), `api/v1.0/v1.0.go`(0%), `config.LoadGlobalConfig`(50%), `ValidationError.Error/Unwrap`(0%).
- 선행 PR #486(머지됨)이 api, servicehandler, config 를 올렸고 중복 진행 PR 없음.
- 제외(제품 코드 변경 금지): `runSend*`의 성공 경로는 외부 연결(RabbitMQ)에 의존해 제외(검증 실패 경로는 단위 테스트 가능). `runService`(DB, RabbitMQ, 포트 80/443), 두 `main`(`os.Exit`, `log.Fatalf`), `writeBase64` 의 치명적 오류 경로(프로세스 종료), `config.Bootstrap` 의 BindPFlag 오류 경로(직전에 플래그를 정의하므로 도달 불가).

## 2. 테스트 계획 (새 함수는 `Test_` 접두사, testify 미사용)

| 대상 | 케이스 |
|---|---|
| hook-control `getPayloadData` | 둘 다 비어 있음, 둘 다 지정, `--data`, `--file`, 없는 파일 오류. 각 테스트에서 `viper.Reset()` 후 `viper.Set` 으로 data, file 를 명시(직접 호출 테스트에만 사용, Execute 테스트에서는 override 가 BindPFlags 변이를 가리므로 `viper.Set` 금지)하고 `t.Cleanup` 으로 `viper.Reset()`(Execute 테스트와 config 테스트도 동일). 반환 바이트를 단언하고 `--file` 은 `t.TempDir()` 파일 내용과 비교 |
| hook-control `printJSON` | 정상(stdout 교체, 출력 JSON 파싱 검증, `defer` 복원, `t.Parallel` 미사용), 마샬 불가 값(chan) 오류 |
| hook-control `cmdSend*` | Use 와 uri/data/file 플래그 존재와 기본값(주된 검증), RunE 비nil |
| hook-control `initCommand` | 하위 명령 3개, 영속 플래그 7개. `PersistentPreRunE` 는 직접 호출하지 않고(플래그 병합 전이라 BindPFlags 를 검증하지 못함) 로컬 플래그 `--data` 를 가진 no-op `Run` 보유 더미 하위 명령(Run 이 없으면 cobra 가 PersistentPreRunE 를 돌리지 않음)을 붙여 `SetArgs([]string{"dummy", "--data", "<가짜값>", "--rabbitmq_address", "<가짜값>"})` 후 `Execute()` 실행. `BindPFlags` 만 묶는 키는 로컬 플래그이므로 `viper.GetString("data")` 가 전달값과 같은지 단언(실행마다 검증됨). `config.Get()` 의 rabbitmq 값은 once 가 첫 값을 고정하므로 같은 가짜값만 사용하고 보조 확인으로만 둠 |
| hook-manager `init()`/`rootCmd` (이 패키지 테스트에서는 `viper.Reset()` 금지: init 이 전역 viper 에 바인딩함) | 테스트 파일 존재로 `init()` 이 자동 실행되어 커버됨(약 30 statement). 플래그 7개와 기본값 검증 |
| hook-manager `writeBase64` | `t.TempDir()` 경로 사용(상수 /tmp 경로 금지), 정상 디코드 기록 내용 일치(가짜 값), 빈 문자열(파일 생성과 크기 0 확인), 없는 디렉터리 경로 오류(권한이 아닌 경로 부재 방식) |
| hook-manager `initProm` | 프로세스당 1회만 호출(DefaultServeMux 중복 등록 panic 방지, `-count=2` 에서도 `sync.Once`). 루프백 임의 포트를 `net.Listen("127.0.0.1:0")` 로 얻어 닫은 뒤 전달. HTTP 200 은 `time.Sleep` 이 아닌 deadline 있는 polling 으로 확인하고 `/metrics` 응답 본문에 `go_goroutines` 포함까지 확인, 시간 내 실패 시 테스트 실패. 고루틴은 프로세스 종료까지 누수를 수용, 포트 해제 후 재사용의 짧은 경쟁 구간은 deadline polling 으로 실패가 드러나므로 수용 |
| `apiv1.ApplyRoutes` | `engine.Routes()` 의 메서드와 경로 쌍 단언: POST /v1.0/emails/:target, POST /v1.0/messages/:target, POST /v1.0/billing/paddle, Any(/v1.0/conversation/*any 는 등록 메서드 전부를 http.Method* 기대 집합과 비교) |
| config `LoadGlobalConfig` | 패키지 내부에서 `cfg` 저장 후 `cfg=nil`, `once = sync.Once{}` 초기화, 7개 필드에 서로 다른 값을 넣고 전부 단언, `t.Cleanup` 으로 `cfg` 복원과 `once = sync.Once{}` 재초기화(Once 는 복사 복원 불가). `Get` panic 은 기존 `TestGet` 과 중복이라 제외 |
| config 기존 테스트 순서 의존 정리 | 기존 `config_test.go` 가 `-count=2`, `-shuffle=on` 에서 이미 실패하므로(TestInitConfig 와 TestLoadGlobalConfig 순서 의존, TestGet 이 cfg 를 nil 로 남김) 각 테스트가 `cfg` 와 `once` 를 저장, 초기화하고 `t.Cleanup` 으로 `cfg` 복원, `once` 재초기화, `viper.Reset()` 하도록 `_test.go` 만 수정. 기존 `func TestXxx` 선언 줄은 수정하지 않고(`check-test-conventions` 가 추가된 줄만 검사) 본문만 고침 |
| servicehandler `ValidationError` | Error 메시지, `errors.As`/`errors.Is` 체인 (Err nil 케이스 제외) |

예상 증가(리뷰어 프로파일 합산): hook-control 도달 가능분 87, `init()` 30, `v1.0.go` 10, `LoadGlobalConfig` 약 2, `ValidationError` 2 로 약 +131(251+131=382/577, 약 66%). `initProm` 과 `writeBase64` 는 합계에 넣지 않았으며 안전마진. `PersistentPreRunE` 의 BindPFlags 오류 분기와 Bootstrap 오류 반환은 도달 불가로 제외.

## 3. verify 항목

- 수기 합계 61% 이상(gens, mock_, DO NOT EDIT 제외, 동일 블록 최대 count 병합). 패키지별 전후 표, 수기와 원본 합계를 PR 본문에 기록.
- `git fetch origin main` 후 `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`, `go test -race -count=2 -shuffle=on ./...`, `bash scripts/check-test-conventions.sh`. 기존 config 테스트 정리 후 `-count=2 -shuffle=on` 이 통과해야 함.
- 변이 검사(scratch 복사본): getPayloadData 의 상호배타 조건 반전, `ValidationError.Unwrap` 제거, `ApplyRoutes` 라우트 1개 삭제, `writeBase64` 기록 내용 변경, `PersistentPreRunE` 의 `BindPFlags` 제거, `LoadGlobalConfig` 본문 제거, `cmdSend*` 플래그 1개 삭제나 기본값 변경, `initCommand` 의 `AddCommand` 1개 삭제, `printJSON` 의 `MarshalIndent` 변경, hook-manager `init()` 플래그 기본값 변경, `initProm` 의 `http.Handle` 제거, `ValidationError.Error` 변경 시 새 테스트가 실패해야 함.
- 제품 코드 변경 없음: `git diff --stat origin/main...HEAD` 가 bin-hook-manager/ 안의 `_test.go` 와 이 문서만 포함.
