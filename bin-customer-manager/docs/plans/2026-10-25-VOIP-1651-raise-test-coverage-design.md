# VOIP-1651: bin-customer-manager unit test coverage to 70%

## 1. 현황
- 기준선(origin/main 99b59c386, `GOFLAGS=-mod=mod go test -count=1 -coverprofile`): 수기 코드 2132/3095 = 68.89%, 생성 코드와 mock 을 포함한 원본 2132/3460 = 61.62%.
- 수기 코드는 `gens/`, `mock_*.go`, `Code generated ... DO NOT EDIT` 헤더 파일을 뺀 값이며 같은 블록의 중복은 최대 count 로 합친다.
- 미커버는 오류 경로에 집중되어 있다: cmd/customer-control, cmd/customer-manager, listenhandler, dbhandler, customerhandler, accesskeyhandler.
- cmd/customer-manager 의 기동 코드와 cmd/customer-control `run*` 의 연결 이후 부분은 DB, Redis, RabbitMQ 연결에 의존하므로 대상에서 제외한다. 연결 이전에 반환하는 검증 분기와 순수 헬퍼(`resolveUUID`, `resolveString`, `printJSON`), 명령 생성자는 대상이다.

## 2. 설계
테스트 파일만 추가한다. 제품 코드, 시그니처, 노출 범위, go.mod, go.sum 은 바꾸지 않는다. 합성값만 쓰고 외부 호출과 네트워크를 쓰지 않는다. 함수명은 `Test_` 접두사, gomock 컨트롤러 변수명은 `mc`, testify 와 `t.Parallel` 은 쓰지 않는다. 전역 상태(`os.Stdout`, viper, logrus)는 변경 후 복원한다.

1. pkg/customerhandler/error_paths_test.go: Get 의 NotFound 래핑과 그 외 오류 전달, Create, Delete, 갱신류, email 중복 검사, Signup, 이메일 검증, Freeze 와 Recover 의 오류 및 경합 분기. 오류 반환과 이후 호출 부재를 단언한다.
2. pkg/accesskeyhandler/error_paths_test.go: List, Get(NotFound 래핑), 고객별 목록, 생성, 갱신, 삭제의 오류 전파와 이후 호출 부재.
3. pkg/listenhandler/error_paths_test.go: 미커버 엔드포인트의 잘못된 본문, URI 항목 부족, filter 오류, 핸들러 오류 응답의 상태 코드를 현재 동작 그대로 고정하고, `Run` 의 큐 생성 오류 전파와 소비자 시작을 단언한다.
4. pkg/dbhandler/error_paths_test.go: 비공유 `:memory:` 연결을 즉시 닫아 쿼리 실행 오류 전파를 단언하고, 캐시 Set 오류는 gomock 으로 주입한다. CustomerDelete 와 AccesskeyDelete 의 대상 없음 분기를 단언한다.
5. cmd/customer-control/main_test.go: 헬퍼 함수, 명령 생성자의 이름과 플래그 표, `run*` 의 연결 이전 검증 분기.

## 3. verify 항목
- 수기 합계 70% 이상(목표 71% 이상), 패키지별 전후 표를 PR 본문에 기재.
- `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m` 통과, `-race` 통과, `scripts/check-test-conventions.sh` 통과.
- `git diff --stat origin/main...HEAD` 가 bin-customer-manager/ 안의 `_test.go` 와 이 문서뿐이며 go.mod, go.sum 변경이 없음.
- 변이 검사(scratch 복사본에서만 적용)로 NotFound 래핑 제거, 재조회 오류 무시, 갱신 오류 무시, 400 상태 변경, 오류 래핑 변경, 캐시 Set 오류 무시 등을 새 테스트가 잡는지 확인.
- VOIP-1416 과 겹치는 패키지는 PR 본문에 명시만 한다.
