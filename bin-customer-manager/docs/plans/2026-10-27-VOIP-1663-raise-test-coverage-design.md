# VOIP-1663: bin-customer-manager unit test coverage to 90%

## 목표와 범위
수기 코드(gens/, mock_*.go, `Code generated ... DO NOT EDIT` 제외) 커버리지를 86.62%에서 90% 이상(90.5% 이상 권장)으로 올린다. 테스트 파일과 테스트 전용 의존성만 추가하며 제품 코드, 시그니처, 노출 범위는 바꾸지 않는다. 값은 합성값만 쓰고 외부 호출과 네트워크 의존은 없다. 서비스 문서 동기화 대상(listenhandler 라우팅, config, models, replace 지시문)은 바뀌지 않으므로 docs 갱신은 없다.

## 기준선 (수기, base 0bfbe2a1d)
2681/3095 = 86.62%. 90% 까지 105개, 90.5% 까지 120개 부족. 측정은 `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...` 이고 계산 방법은 PR 본문에 기재한다.

## 추가할 테스트와 예상 증분
1. cmd/customer-control (+54): `initCommand().Execute()` 로 PersistentPreRunE, LoadGlobalConfig, migrate 하위 명령 클로저를 지난다. 14개 run* 함수와 initHandler, initAccesskeyHandler 는 DB 연결 오류만 단언한다(표 테스트). DSN 은 항상 문법이 깨진 합성 문자열(`invalid-dsn`)이며 빈 문자열은 쓰지 않는다(빈 값은 기본 주소로 접속을 시도하므로). initCache 는 닫힌 miniredis 주소로 직접 호출한다. 첫 Execute 의 LoadGlobalConfig(once)가 globalConfig 를 viper 값으로 덮어쓰므로 DSN 은 Execute 이전에 `viper.Set("database_dsn", "invalid-dsn")` 로 주입하고 t.Cleanup 으로 원복한다(빈 값이면 기본 주소로 접속을 시도하므로 첫 호출부터 합성 DSN 이어야 한다). once 는 cmd 패키지에서 초기화할 수 없고 필요하지도 않다. Execute 테스트는 끝에서 viper 를 초기화한다.
2. cmd/customer-manager (+31): runDaemon 을 잘못된 DSN 으로 호출해 DB 연결 오류 반환을 단언한다. initProm 은 DefaultServeMux 에 등록하므로 호출마다 `PrometheusEndpoint` 를 서로 다른 경로(호출 카운터로 생성)로 설정하고 listen 주소는 `127.0.0.1:0` 으로 하여 `-count>1` 에서도 중복 등록 panic 이 없게 한다. initSignal 은 signal.Notify 등록 이후에 SIGHUP 을 자기 프로세스에 보내 chDone 수신(상한 있는 select)을 단언하고 cleanup 에서 signal.Stop 과 chDone 비우기를 한다. 두 채널은 버퍼 1이며 이전 호출이 남긴 고루틴이 신호를 받아도 chDone 에는 항상 한 값이 들어간다. initCache 는 닫힌 주소로 실패를 단언한다.
3. internal/config (+5): 같은 패키지 테스트에서 globalConfig 는 값으로 저장하고 once 는 복사하지 않고(copylocks) `once = sync.Once{}` 로 초기화하며 viper 값을 넣은 뒤 LoadGlobalConfig 가 각 필드를 옮기는지 단언한다. 두 번째 호출이 값을 바꾸지 않는지도 확인하고 cleanup 에서 globalConfig 를 원복하고 once 는 `once.Do(func(){})` 로 완료 상태로 되돌린다.
4. cachehandler (+5): ResendCountIncr 의 세 분기를 나눠 단언한다. Incr 실패는 (0, 오류), n==1 일 때 Expire 실패와 n>1 일 때 TTL 실패는 (n, 오류), n>1 이고 TTL 이 음수일 때 Expire 실패도 (n, 오류). 실패는 `Cache.AddHook` 으로 특정 명령만 실패시켜 주입하고 닫힌 miniredis 는 Incr 실패에 쓴다. setSerialize 는 같은 패키지에서 직렬화할 수 없는 값(채널)을 넘겨 직렬화 오류 반환을 단언한다.
5. dbhandler (+29~32): 기존 sqlite 하네스로 만들 수 없는 분기(RowsAffected 오류, rows iteration 오류, 행 스캔 오류)에만 go-sqlmock(테스트 전용, v1.5.2, 다른 서비스에 이미 있음)을 쓴다. customer 와 accesskey 에 대해 이 오류들의 래핑 문구와 errors.Is 를 단언한다. 캐시 hit 반환 테스트는 gomock 캐시 mock 이 hit 를 돌려주게 하고, 소비되지 않을 sqlmock `ExpectQuery` 하나를 sentinel 로 등록한 뒤 호출 후 `ExpectationsWereMet()` 가 오류를 돌려주는지(DB 조회가 없었는지)로 단언한다(gomock 만으로는 추가된 DB 조회를 볼 수 없다). customerUpdateToCache 와 accesskeyUpdateToCache 의 오류 반환, 캐시 갱신 오류 무시, 빈 필드 갱신 무동작도 이 목록에 포함한다.

예상 합계 약 122~126 stmt(86.62% 에서 약 90.6%). 90.0% 에는 15개 이상, 90.5% 에는 2~6개 여유로 얇다. listenhandler 는 대상에서 제외한다. 구현 중 90.5% 에 못 미치면 우선순위는 (1) 90.0% 확보, (2) cmd/customer-control, dbhandler, cachehandler 의 남은 도달 가능 분기, (3) 그래도 부족하면 근거와 함께 보고한다.

## 검증 계획
- 수기 합계 90.0% 이상(90.5% 권장), 패키지별 전후 표를 PR 본문에 첨부. 최종 HEAD 에서 dbhandler 를 제외한 패키지는 `-race -shuffle=on -count=1` 을 20회 반복하고 `-count=3` 도 한 번 실행해 전역 상태 누수를 확인한다. 기존 dbhandler sqlite 하네스는 고정 ID 를 공유 DB 에 넣어 원본에서도 shuffle 과 `-count>1` 에 실패하므로(이번 변경과 무관한 기존 동작) dbhandler 는 기본 순서 `-race -count=1` 로 전체를 확인하고, 새 sqlmock 테스트(테스트마다 인스턴스 생성)만 `-run` 으로 걸러 `-race -shuffle=on` 20회와 `-count=3` 을 별도로 확인한다. cachehandler 훅 실패 케이스는 사전 Incr 횟수와 TTL 유무로 n==1 Expire, n>1 TTL, TTL 음수 Expire 를 구분한다. `go mod tidy` 는 require 블록 순서를 바꿀 수 있으므로 'diff 없음' 은 tidy 이후 기준이다.
- 변이 검사 표본(복사본에서 실행, 변이마다 실패해야 할 테스트를 짝으로 기록하고 원복을 확인): RowsAffected 오류 처리 제거(dbhandler sqlmock 오류 테스트; RowsAffected 0 분기는 기존 freeze/recover 테스트가 잡는다), 캐시 hit 에서 DB 조회 추가(sqlmock sentinel 을 쓰는 캐시 hit 테스트), run* 오류 래핑 문구 변경(customer-control 표 테스트), LoadGlobalConfig 필드 매핑 변경과 once 제거(config 테스트), ResendCountIncr 의 오류 무시 또는 반환 값 변경(cachehandler 세 분기 테스트), initSignal 의 chDone 송신 제거(customer-manager 테스트), initCache 오류 무시(customer-manager 테스트).
- `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`, 저장소 루트의 `scripts/check-test-conventions.sh`, `git diff --stat origin/main...HEAD` 가 bin-customer-manager/ 안에만 있음을 확인한다. go.mod/go.sum 변경은 sqlmock 추가와 그 전이 의존성의 go.mod 해시 한 줄뿐이며 `go mod tidy` 후 diff 가 없어야 한다.
- 새 테스트는 저장소 규칙(`Test_` 접두 이름, `reflect.DeepEqual` 과 `t.Errorf` 단언, gomock 컨트롤러 변수 `mc`, 추가된 줄에만 적용)을 따르고 go.sum 은 sqlmock 과 전이 의존성의 3줄이 늘어난다.

## 구현 시 주의
- 전역 상태(viper, config, http.DefaultServeMux, signal)는 t.Cleanup 으로 복원하고 새 테스트는 시간과 외부 상태에 의존하지 않게 한다. 시간 상한이 있는 select 는 실패 판정용으로만 쓴다.
- `-shuffle` 에서 순서가 바뀌어도 첫 LoadGlobalConfig 가 합성 DSN 을 읽도록, cmd 의 모든 run* 테스트는 시작 시 viper 와 `*config.Get()` 양쪽에 합성 DSN 을 주입한다.
- 자기비교식 테스트와 빈 호출 테스트는 쓰지 않는다. 기대값은 소스와 독립적으로 문자열로 적는다. DSN 과 주소는 합성값이며 비밀번호 리터럴을 두지 않는다.
