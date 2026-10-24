# VOIP-1651: bin-customer-manager unit test coverage to 70%

## 1. 이슈 분석 (착수 전 티켓 검토)
- 기준선(origin/main 99b59c386, `GOFLAGS=-mod=mod go test -count=1 -coverprofile`): 수기 2132/3095 = 68.89%, 원본 2132/3460 = 61.62%, 테스트 실패 0건. 수기 제외 기준은 `gens/`, `mock_*.go`, `Code generated ... DO NOT EDIT` 헤더 파일이며 같은 블록은 최대 count 로 합친다. 70.0% 까지 35개, 71% 까지 66개 부족(올림 계산).
- 미커버 분포(수기, statement, 합 963): cmd/customer-control 594, cmd/customer-manager 99, pkg/listenhandler 101, pkg/dbhandler 83, pkg/customerhandler 55, pkg/accesskeyhandler 15, internal/config 8, pkg/cachehandler 5, models 2, pkg/metricshandler 1.
- 파일별 상위: dbhandler/customer.go 47, listenhandler/v1_customers.go 42, dbhandler/accesskey.go 32, listenhandler/v1_accesskeys.go 28, customerhandler/db.go 23, listenhandler/main.go 21, accesskeyhandler/db.go 15, customerhandler/signup.go 13, customerhandler/customer.go 11.
- cmd/customer-manager(99): main, runDaemon, startServices, initCache 는 DB, Redis, RabbitMQ 연결에 의존하고 initSignal, initProm 은 프로세스 전역 시그널과 리스너를 다루는 기동 코드(합쳐 10 statement 안팎)라 효과 대비 부작용이 커서 제외한다.
- cmd/customer-control(594): `run*` 의 연결 이후 부분은 `initHandler` 가 실제 DB 와 Redis 를 열고 `initCustomerHandler` 도 RabbitMQ 연결(`sockHandler.Connect`)을 시도하므로 제외한다. 반면 `run*` 안의 `resolveUUID` 와 `resolveString` 검증 실패 분기는 `initHandler` 호출보다 앞에서 반환하므로 연결 없이 도달 가능하다. 순수 헬퍼(`resolveUUID`, `resolveString`, `printJSON`), 명령 생성자(`cmd*`, 플래그 정의, 실측 약 237), `initCommand`(약 85), 이 검증 분기가 연결 없이 도달 가능한 대상이다.
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
| cmd/customer-control (`initCommand` 호출이 모든 `cmd*` 생성자를 실행하므로 실측 initCommand 85 와 생성자 237 의 대부분, 헬퍼 16, run* 검증 분기) | 약 320 |
| internal/config LoadGlobalConfig | 약 5 |
| customerhandler Freeze 와 Recover 경합 분기 | 약 8 |
| 합계 | 약 560 (71% 필요 66, 70% 필요 35) |

항목:
1. pkg/customerhandler/error_paths_test.go: Get 의 ErrNotFound 는 `stderrors.As` 로 `*cerrors.VoipbinError` 를 꺼내 코드 `CUSTOMER_NOT_FOUND` 를 단언하고 `errors.Is(err, dbhandler.ErrNotFound)` 도 유지됨을 확인, 그 외 오류는 `err == sentinel` 로 그대로 전달됨을 단언. Create 의 secret 생성 오류와 CustomerCreate 오류와 생성 후 재조회 오류, Delete 의 Get, dbDelete, 재조회 오류, 기본 정보, 결제 계정, 메타데이터 갱신의 갱신 오류와 재조회 오류, email 중복 검사의 List 오류와 agent 조회 오류와 이미 사용 중인 email, Signup 의 secret 생성 오류와 재조회 오류, 이메일 검증의 락 획득 오류가 내부 오류로 변환됨, Freeze 와 Recover(freeze.go 의 경합 분기): `CustomerFreeze` 가 ErrNotFound 를 반환했을 때 재조회 결과가 frozen 이면 재조회 객체를 성공으로 반환, 재조회 결과가 다른 상태이거나 재조회가 실패하면 원래 오류 반환, ErrNotFound 가 아닌 오류는 그대로 전달(재조회 호출 없음). Recover 도 같은 구조(재조회 결과가 active 일 때 성공). Signup 의 accesskeyHandler.Create 실패는 치명(고정 오류 반환, 이후 이벤트 발행 없음), `sendSignupVerification` 실패(토큰 저장 또는 이메일 발송 오류)는 비치명(결과 객체와 nil 반환), validateCreate 실패는 "the email is not available" 반환, EmailVerify 의 `EmailVerifyTokenDelete` 실패와 `AgentV1PasswordForgot` 실패는 비치명이라 verified 결과 유지. 위 현재 동작을 함수별로 단언한다. 각 실패 분기에서 이후 호출(예: 갱신 실패 뒤 재조회)이 없음을 mock 기대 부재로 단언. 재조회 실패는 현재 동작대로 오류 반환(`err != nil`)과 이후 호출 부재까지만 단언한다.
2. pkg/accesskeyhandler/error_paths_test.go: List 오류, Get 의 ErrNotFound 래핑(`ACCESSKEY_NOT_FOUND`)과 그 외 오류 전달, 고객별 목록 오류, 토큰 생성 오류, 생성 후 재조회 오류, Delete 와 Update 의 갱신과 재조회 오류(`err != nil` 과 이후 호출 부재까지).
3. pkg/listenhandler/error_paths_test.go: customers 와 accesskeys 의 미커버 엔드포인트별 잘못된 JSON 본문 400, 핸들러 오류 응답은 엔드포인트별 현재 상태 코드를 표로 고정해 단언하는 특성화 테스트(의도인지 여부는 판단하지 않으며 주석은 중립적으로 쓴다). 확인한 현재 동작: customers 의 Get, Post, IDGet, IDPut 등 일부는 핸들러 오류를 `errorResponse` 로 변환(일반 오류 500, typed 오류는 해당 상태, ErrNotFound 404), 나머지 customers 와 accesskeys 의 오류는 구현 단계에서 엔드포인트마다 코드로 확인해 표에 옮기고, freeze, freeze_and_delete, recover 는 URI 항목 부족과 핸들러 오류 모두 400, 잘못된 퍼센트 인코딩 URI 의 `url.Parse` 실패가 processRequest 에서 기본 400 으로 변환, `Run` 은 QueueCreate 실패 시 래핑 오류 반환, 성공 시 ConsumeRPC mock 의 `DoAndReturn` 에서 채널을 닫고 테스트가 그 채널 수신 뒤에 끝나도록 해 `mc.Finish` 이후 호출과 레이스를 막음.
4. pkg/dbhandler/error_paths_test.go: `sql.Open("sqlite3", ":memory:")` 로 비공유 DSN 연결을 만들어 즉시 Close 한 핸들러로 Customer 와 Accesskey 의 Create, Get(캐시 미스 선행), List, Update, Delete 등 쿼리 실행 오류 전파를 단언. 전역 `dbTest` 와 공유 `file::memory:?cache=shared` DB 는 건드리지 않는다. 캐시 Set 오류는 gomock 으로 주입해 전파를 단언. CustomerDelete 의 영향 행 0 은 ErrNotFound.
5. cmd/customer-control/main_test.go: resolveUUID(빈 값, 잘못된 형식, 정상), resolveString(빈 값, 정상), printJSON(`os.Stdout` 교체 후 defer 복원, 쓰기 쪽 닫고 읽어 정상 출력 확인, Marshal 불가 값 오류), `run*` 의 연결 이전 검증 분기(runCreate 의 email 누락, runUpdate 의 id 오류와 email 누락, runUpdateBillingAccount 의 id 오류와 billing-account-id 오류, runGet 와 runDelete 와 runFreeze 와 runRecover 의 id 오류, runSetIdentityVerification 의 id 오류와 status 누락과 status 무효, runAccesskeyCreate 의 customer-id 오류와 name 누락과 expire 형식 오류(`time.ParseDuration`, initAccesskeyHandler 이전), runAccesskeyGet 과 runAccesskeyUpdate 와 runAccesskeyDelete 의 id 오류)이 `initHandler` 이전에 오류를 반환함을 오류 메시지 래핑까지 단언(viper 키 이름과 `run*` 의 일치를 함께 검증). 명령 생성자 단언은 (a) 모든 하위 명령이 `initCommand` 트리에 등록됨(Find), (b) 각 `run*` 이 읽는 viper 키가 해당 생성자에 플래그로 존재함(`run*` 소스에서 뽑은 키 목록(expire, billing-account-id, update 의 email 포함)을 테스트 안의 독립 리터럴 표로 대조, 소스 상수를 재사용하는 자기비교 금지), (c) webhook-method 기본값 POST 로 한정. `initCommand` 는 호출만 하고 `Execute` 하지 않으며(`Execute` 시 DB 에 닿음), `config.Bootstrap` 이 전역 viper 에 AutomaticEnv, BindPFlag, BindEnv 를 호출하고 logrus 설정을 바꾸므로 호출 전 logrus 레벨과 Formatter 를 저장해 `t.Cleanup` 으로 복원한다. `AutomaticEnv` 가 켜져 있으므로 `id`, `name`, `status` 등 키는 `t.Setenv` 로 같은 이름의 환경변수를 비워 환경 영향을 고정하고 빈 값 케이스는 `viper.Set(key, "")` 로 명시한다. viper 에는 키 삭제 API 가 없으므로 이전 값을 `viper.IsSet` 과 `viper.Get` 으로 저장했다가 `t.Cleanup` 에서 복원하거나 미설정이던 키는 `viper.Set(key, nil)` 로 해제한다. 빈 문자열 오버라이드를 남기지 않는다.
6. internal/config: LoadGlobalConfig 가 viper 값을 Config 필드로 옮기는지를 필드별 서로 다른 합성값으로 단언. `once` 와 `globalConfig` 를 테스트 안에서 초기화하고 `t.Cleanup` 으로 원복(기존 TestGet 영향 방지), `viper.Reset()` 은 기존 바인딩을 지우므로 쓰지 않으며 viper 키는 위 항목 5 와 같은 방식(이전 값 복원 또는 nil 해제)으로 정리.

## 3. verify 항목
- 수기 합계 C/T >= 0.70 (올림 없이 정확히), 목표 71% 이상. 항목별 증분을 `hw.py` 로 실측하고 패키지별 전후 표, 원본 합계, 제외 방법(계산 스크립트 규칙 동일)을 PR 본문에 기재.
- `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`. 신규 테스트는 `-race` 와 `-shuffle=on -count=3` 로도 실행해 순서 의존과 전역 상태 누수 확인.
- `git fetch origin main` 후 저장소 루트의 `scripts/check-test-conventions.sh`(Test_ 접두사, testify 금지, 컨트롤러명 mc) 실행. `git diff --stat origin/main...HEAD` 가 bin-customer-manager/ 안의 `_test.go` 와 이 문서뿐이며 go.mod, go.sum, vendor 변경이 없음을 확인.
- 변이 검사(scratch 복사본에서만 적용, 변이당 새 테스트 실패와 원복 후 통과를 기록, 생존 변이는 단언 보강): NotFound 래핑 제거, 재조회 실패 시 오류 무시, 갱신 오류 무시, 잘못된 본문 400 을 다른 상태로 변경, errorResponse 계열 상태 코드 변경, 락 오류 변환 제거, 비치명 오류를 치명으로 변경, dbhandler `%w` 를 `%v` 로 변경, 캐시 Set 오류 무시, CustomerDelete 의 ErrNotFound 반환 제거, LoadGlobalConfig 필드 매핑 교환, Freeze 와 Recover 경합 분기의 재조회 상태 비교 제거, VoipbinError 의 Wrap(err) 제거(errors.Is 유지 단언), accesskey Delete 재조회 오류 무시, Signup 의 accesskey 생성 실패를 비치명으로 변경과 sendSignupVerification 실패를 치명으로 변경, resolveUUID 의 nil 비교 제거, resolveString 빈 값 분기 제거, Run 의 오류 래핑 제거와 ConsumeRPC 호출 제거.
- 테스트 중 발견한 결함은 고치지 않고 카드 코멘트와 완료 요약에만 기록하며 문서와 테스트 주석에는 중립적으로만 쓴다.
- VOIP-1416 과 겹치는 패키지는 PR 본문에 명시만 한다.
