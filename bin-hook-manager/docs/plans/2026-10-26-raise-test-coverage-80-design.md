# VOIP-1657 bin-hook-manager 단위 테스트 커버리지 80% 계획

## 1. 티켓 검토 (부분 유효)

- 기준선(origin/main 161e9748b, `GOFLAGS=-mod=mod go test -count=1 -coverprofile`): 원본 80.5%, 수기 합계 79.55% (수기 statement 577개 중 459 커버). 80%까지 3개 이상 필요.
- 선행 PR(VOIP-1647, 1648, 1652)이 api, servicehandler, config 를 거의 100% 로 올렸고 남은 미커버는 `cmd/` 안에 집중. 중복 진행 PR 없음.
- 제품 코드 변경 없이 테스트만 추가. 외부 연결(RabbitMQ, DB)이 필요한 경로와 프로세스를 종료시키는 경로는 대상에서 제외.

## 2. 테스트 계획 (새 함수는 `Test_` 접두사, testify 미사용)

| 대상 | 케이스 |
|---|---|
| hook-control `main` | `os.Args` 를 `[hook-control, --help]` 로 바꾸고 stdout 을 파이프로 교체해 호출, 출력에 `send-email`, `send-message`, `send-conversation` 하위 명령 포함 단언. `os.Args`, `os.Stdout` 은 `t.Cleanup` 으로 복원, 기존 `resetViper(t)` 호출(main 이 viper 전역을 바꿈), 파이프는 읽기 goroutine 과 `Close` 로 처리, `t.Parallel` 미사용 |
| hook-manager `main` | `os.Args` 를 `[hook-manager, --help]` 로 바꾸고 stdout 을 파이프로 교체해 호출, 출력에 `hook-manager` 와 플래그 이름 포함 단언(`runService` 가 실행되지 않으므로 반환). `os.Args`, `os.Stdout` 복원, `rootCmd` 의 args 는 건드리지 않음 |
| hook-manager `writeBase64` | 잘못된 base64 입력. `logrus.StandardLogger().ExitFunc` 를 종료 코드를 기록하는 함수로 바꾸고(`t.Cleanup` 으로 복원) 종료 코드 1 호출 여부, 반환값 nil, 빈 파일을 단언. 파일은 `t.TempDir()` 경로 |

예상 증가: +4 statement (459 에서 463, 약 80.2%). 80.5% 권장치(+6)는 미달 예상이며 PR 본문에 명시.

## 3. verify 항목

- 수기 합계 80.0% 이상(gens, mock_, DO NOT EDIT 제외, 동일 블록 최대 count 병합). 항목별 증가를 `go tool cover` 로 확인.
- `git fetch origin main` 후 `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`, `go test -race -count=2 -shuffle=on ./...`, `bash scripts/check-test-conventions.sh`.
- 변이 검사(scratch 복사본): `writeBase64` 의 Fatalf 를 Errorf 로 변경, hook-control 하위 명령 `AddCommand` 1개 삭제, hook-manager `rootCmd.Use` 변경 시 새 테스트가 실패해야 함.
- 제품 코드 변경 없음: `git diff --stat origin/main...HEAD` 가 bin-hook-manager/ 안의 `_test.go` 와 이 문서만 포함.
