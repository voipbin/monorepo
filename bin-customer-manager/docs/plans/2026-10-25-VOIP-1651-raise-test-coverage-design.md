# VOIP-1651: bin-customer-manager unit test coverage to 70%

## 1. 이슈 분석 (착수 전 티켓 검토)
- 기준선(origin/main 99b59c386, `GOFLAGS=-mod=mod go test -count=1 -coverprofile`): 수기 2132/3095 = 68.89%, 원본 2132/3460 = 61.62%, 테스트 실패 0건. 수기 제외 기준은 `gens/`, `mock_*.go`, `Code generated ... DO NOT EDIT` 헤더 파일이며 같은 블록은 최대 count 로 합친다. 70.0% 까지 35개, 71% 까지 66개 부족(올림 계산).
- 미커버 분포(수기, statement, 합 963): cmd/customer-control 594, cmd/customer-manager 99, pkg/listenhandler 101, pkg/dbhandler 83, pkg/customerhandler 55, pkg/accesskeyhandler 15, internal/config 8, pkg/cachehandler 5, models 2, pkg/metricshandler 1.
- 파일별 상위: dbhandler/customer.go 47, listenhandler/v1_customers.go 42, dbhandler/accesskey.go 32, listenhandler/v1_accesskeys.go 28, customerhandler/db.go 23, listenhandler/main.go 21, accesskeyhandler/db.go 15, customerhandler/signup.go 13, customerhandler/customer.go 11.
- cmd/customer-manager(99): main, runDaemon, startServices, initCache, initSignal, initProm 은 DB, Redis, RabbitMQ 연결과 시그널, HTTP 리스너에 의존하는 기동 코드라 단위 테스트 대상에서 제외한다.
- cmd/customer-control(594): `run*` 은 `initHandler` 가 실제 DB 와 Redis 를 열고 `initCustomerHandler` 도 RabbitMQ 연결(`sockHandler.Connect`)을 시도하므로 제외한다. 순수 헬퍼(`resolveUUID`, `resolveString`, `printJSON`)와 명령 생성자(`cmd*`, 플래그 정의)는 연결 없이 도달 가능하며 합쳐 약 100 statement 이상이다.
- 도달 불가로 제외하는 분기: 도메인 구조체 `json.Marshal` 실패 500(listenhandler), squirrel `ToSql` 실패, 구조체를 넘기는 `PrepareFields` 실패, `crypto/rand` 실패, `RowsAffected` 오류, `rows.Err()` 분기, 합성 행으로 만들려면 스키마 변경이 필요한 ScanRow 실패, config 의 viper 바인딩 오류 분기, `processRequest` 의 typed VoipbinError 와 ErrNotFound 분기(핸들러가 `err` 를 반환하는 경우는 `url.Parse` 실패뿐이라 기본 400 만 도달 가능).

## 2. 설계
테스트 파일만 추가한다. 제품 코드, 시그니처, 노출 범위, go.mod, go.sum 은 바꾸지 않는다. 합성값만 쓰고 외부 호출과 네트워크는 쓰지 않는다. 기존 `*_errmap_test.go`, `error_response_test.go` 와 겹치지 않게 미커버 블록 목록에서만 뽑는다. 함수명은 `Test_` 접두사, gomock 컨트롤러 변수명은 `mc`, testify 미사용, `t.Parallel` 미사용.

statement 예산(미커버 기준 도달 가능 추정, 구현 후 `hw.py` 로 실측해 PR 본문에 기재):

| 항목 | 예산 |
|---|---|
| customerhandler (Get, Create, Delete, 갱신류, email 중복 검사, Signup, 이메일 검증) | 약 50 |
| accesskeyhandler | 약 13 |
| listenhandler (잘못된 본문 400, 핸들러 오류 응답, url.Parse 400, Run) | 약 70 |
| dbhandler (실행 오류, 캐시 오류, CustomerDelete 행 없음) | 약 33 |
| cmd/customer-control 순수 헬퍼와 명령 생성자 | 약 60 |
| internal/config LoadGlobalConfig | 약 5 |
| 합계 | 약 230 (71% 필요 66, 70% 필요 35) |

항목:
1. pkg/customerhandler/error_paths_test.go: Get 의 ErrNotFound 는 `stderrors.As` 로 `*cerrors.VoipbinError` 를 꺼내 코드 `CUSTOMER_NOT_FOUND` 를 단언하고 `errors.Is(err, dbhandler.ErrNotFound)` 도 유지됨을 확인, 그 외 오류는 `err == sentinel` 로 그대로 전달됨을 단언. Create 의 secret 생성 오류와 CustomerCreate 오류와 생성 후 재조회 오류, Delete 의 Get, dbDelete, 재조회 오류, 기본 정보, 결제 계정, 메타데이터 갱신의 갱신 오류와 재조회 오류, email 중복 검사의 List 오류와 agent 조회 오류와 이미 사용 중인 email, Signup 의 secret 생성 오류와 재조회 오류, 이메일 검증의 락 획득 오류가 내부 오류로 변환됨, 토큰 삭제 실패와 비밀번호 재설정 요청 실패는 비치명(signup.go 의 해당 줄을 구현 전에 확인해 그 현재 동작을 단언). 각 실패 분기에서 이후 호출(예: 갱신 실패 뒤 재조회)이 없음을 mock 기대 부재로 단언. 재조회 실패는 현재 동작대로 오류 반환(`err != nil`)과 이후 호출 부재까지만 단언한다.
2. pkg/accesskeyhandler/error_paths_test.go: List 오류, Get 의 ErrNotFound 래핑(`ACCESSKEY_NOT_FOUND`)과 그 외 오류 전달, 고객별 목록 오류, 토큰 생성 오류, 생성 후 재조회 오류, Delete 와 Update 의 갱신과 재조회 오류(`err != nil` 과 이후 호출 부재까지).
3. pkg/listenhandler/error_paths_test.go: customers 와 accesskeys 의 미커버 엔드포인트별 잘못된 JSON 본문 400, 핸들러 오류 응답은 엔드포인트별 현재 상태 코드를 표로 고정해 단언(errorResponse 계열은 일반 오류 500, typed 오류는 해당 상태, ErrNotFound 는 404, freeze 계열은 코드로 확인한 현재 상태), 잘못된 퍼센트 인코딩 URI 의 `url.Parse` 실패가 processRequest 에서 기본 400 으로 변환, `Run` 은 QueueCreate 실패 시 래핑 오류 반환, 성공 시 ConsumeRPC mock 의 `DoAndReturn` 에서 채널을 닫고 테스트가 그 채널 수신 뒤에 끝나도록 해 `mc.Finish` 이후 호출과 레이스를 막음.
4. pkg/dbhandler/error_paths_test.go: `sql.Open("sqlite3", ":memory:")` 로 비공유 DSN 연결을 만들어 즉시 Close 한 핸들러로 Customer 와 Accesskey 의 Create, Get(캐시 미스 선행), List, Update, Delete 등 쿼리 실행 오류 전파를 단언. 전역 `dbTest` 와 공유 `file::memory:?cache=shared` DB 는 건드리지 않는다. 캐시 Set 오류는 gomock 으로 주입해 전파를 단언. CustomerDelete 의 영향 행 0 은 ErrNotFound.
5. cmd/customer-control/main_test.go: resolveUUID(빈 값, 잘못된 형식, 정상), resolveString(빈 값, 정상), printJSON(`os.Stdout` 교체 후 defer 복원, 쓰기 쪽 닫고 읽어 정상 출력 확인, Marshal 불가 값 오류), 명령 생성자의 Use 와 플래그 정의와 기본값(webhook-method 기본 POST 등). `initCommand` 는 호출만 하고 `Execute` 하지 않으며 하위 명령은 `Find` 로 확인한다(`Execute` 시 DB 에 닿으므로). viper 키는 각 테스트에서 설정하고 `t.Cleanup` 에서 비운다.
6. internal/config: LoadGlobalConfig 가 viper 값을 Config 필드로 옮기는지를 필드별 서로 다른 합성값으로 단언. `once` 와 `globalConfig` 를 테스트 안에서 초기화하고 `t.Cleanup` 으로 원복(기존 TestGet 영향 방지), `viper.Reset()` 은 기존 바인딩을 지우므로 쓰지 않고 키별로 설정과 정리.

## 3. verify 항목
- 수기 합계 C/T >= 0.70 (올림 없이 정확히), 목표 71% 이상. 항목별 증분을 `hw.py` 로 실측하고 패키지별 전후 표, 원본 합계, 제외 방법(계산 스크립트 규칙 동일)을 PR 본문에 기재.
- `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`. 신규 테스트는 `-race` 와 `-shuffle=on -count=3` 로도 실행해 순서 의존과 전역 상태 누수 확인.
- `git fetch origin main` 후 `scripts/check-test-conventions.sh`(Test_ 접두사, testify 금지, 컨트롤러명 mc) 실행. `git diff --stat origin/main...HEAD` 가 bin-customer-manager/ 안의 `_test.go` 와 이 문서뿐이며 go.mod, go.sum, vendor 변경이 없음을 확인.
- 변이 검사(scratch 복사본에서만 적용, 변이당 새 테스트 실패와 원복 후 통과를 기록, 생존 변이는 단언 보강): NotFound 래핑 제거, 재조회 실패 시 오류 무시, 갱신 오류 무시, 잘못된 본문 400 을 다른 상태로 변경, errorResponse 계열 상태 코드 변경, 락 오류 변환 제거, 비치명 오류를 치명으로 변경, dbhandler `%w` 를 `%v` 로 변경, 캐시 Set 오류 무시, CustomerDelete 의 ErrNotFound 반환 제거, LoadGlobalConfig 필드 매핑 교환, resolveUUID 의 nil 비교 제거, resolveString 빈 값 분기 제거, Run 의 오류 래핑 제거와 ConsumeRPC 호출 제거.
- 테스트 중 발견한 결함은 고치지 않고 카드 코멘트와 완료 요약에만 기록하며 문서와 테스트 주석에는 중립적으로만 쓴다.
- VOIP-1416 과 겹치는 패키지는 PR 본문에 명시만 한다.
